package service

import (
	"context"
	"errors"
	"testing"

	"github.com/yuluo688/credit-manager/internal/store"
)

func TestSetAuthConcurrencyLimitRoundTrip(t *testing.T) {
	s := quotaService(t)
	s.SetAuthQuotaSource(&fakeQuotaSource{files: []AuthQuotaFile{{ID: "auth-1", AuthIndex: "idx-1", Provider: "codex", Label: "ops"}}, auth: quotaJSON("codex")})
	item, err := s.SetAuthConcurrencyLimit(context.Background(), "codex", "auth-1", 2)
	if err != nil {
		t.Fatal(err)
	}
	if item.MaxConcurrentRequests != 2 || item.AuthID != "auth-1" {
		t.Fatalf("item = %#v", item)
	}
	listed, err := s.AuthQuotaOverview(context.Background(), "", AuthQuotaFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Items) != 1 || listed.Items[0].MaxConcurrentRequests != 2 {
		t.Fatalf("overview = %#v", listed.Items)
	}
}

func TestSetAuthConcurrencyLimitsBatch(t *testing.T) {
	s := quotaService(t)
	s.SetAuthQuotaSource(&fakeQuotaSource{files: []AuthQuotaFile{
		{ID: "auth-1", Provider: "codex", Label: "ops"},
		{ID: "auth-2", Provider: "codex", Label: "bot"},
		{ID: "auth-3", Provider: "xai", Label: "grok"},
	}, auth: quotaJSON("codex")})
	got, err := s.SetAuthConcurrencyLimits(context.Background(), AuthQuotaFilter{Provider: "codex"}, nil, 3)
	if err != nil {
		t.Fatal(err)
	}
	if got.Updated != 2 || got.MaxConcurrentRequests != 3 {
		t.Fatalf("filter batch = %#v", got)
	}
	page, err := s.SetAuthConcurrencyLimits(context.Background(), AuthQuotaFilter{}, []AuthConcurrencyTarget{{Provider: "xai", AuthID: "auth-3"}}, 1)
	if err != nil || page.Updated != 1 {
		t.Fatalf("item batch = %#v, %v", page, err)
	}
	listed, err := s.AuthQuotaOverview(context.Background(), "", AuthQuotaFilter{PageSize: 24})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]int64{}
	for _, item := range listed.Items {
		byID[item.AuthID] = item.MaxConcurrentRequests
	}
	if byID["auth-1"] != 3 || byID["auth-2"] != 3 || byID["auth-3"] != 1 {
		t.Fatalf("limits = %#v", byID)
	}
}

func TestAdmitAuthEnforcesConcurrencyLimit(t *testing.T) {
	s := quotaService(t)
	if err := s.Store().UpsertAuthConcurrencyLimit(context.Background(), "claude", "auth-1", 1); err != nil {
		t.Fatal(err)
	}
	s.TrackAuthCapture("res-1", "claude-sonnet")
	s.TrackAuthCapture("res-2", "claude-sonnet")
	auth := store.AuthIdentity{AuthID: "auth-1", Provider: "claude"}
	if err := s.AdmitAuth(context.Background(), "res-1", auth); err != nil {
		t.Fatalf("first admit: %v", err)
	}
	if err := s.AdmitAuth(context.Background(), "res-2", auth); !errors.Is(err, store.ErrConcurrentLimit) {
		t.Fatalf("second admit error = %v", err)
	}
	s.CancelAuthCapture("res-1")
	if err := s.AdmitAuth(context.Background(), "res-2", auth); err != nil {
		t.Fatalf("admit after release: %v", err)
	}
}

func TestPickAuthSkipsBusyAndFallsBackWhenUnlimited(t *testing.T) {
	s := quotaService(t)
	candidates := []AuthPickCandidate{{ID: "a", Provider: "codex"}, {ID: "b", Provider: "codex"}}
	id, handled, err := s.PickAuth(context.Background(), candidates)
	if err != nil || handled || id != "" {
		t.Fatalf("unlimited pick = (%q, %t, %v)", id, handled, err)
	}
	if err := s.Store().UpsertAuthConcurrencyLimit(context.Background(), "codex", "a", 1); err != nil {
		t.Fatal(err)
	}
	if err := s.Store().UpsertAuthConcurrencyLimit(context.Background(), "codex", "b", 1); err != nil {
		t.Fatal(err)
	}
	s.TrackAuthCapture("res-a", "gpt")
	if err := s.AdmitAuth(context.Background(), "res-a", store.AuthIdentity{AuthID: "a", Provider: "codex"}); err != nil {
		t.Fatal(err)
	}
	id, handled, err = s.PickAuth(context.Background(), candidates)
	if err != nil || !handled || id != "b" {
		t.Fatalf("busy skip = (%q, %t, %v)", id, handled, err)
	}
	s.TrackAuthCapture("res-b", "gpt")
	if err := s.AdmitAuth(context.Background(), "res-b", store.AuthIdentity{AuthID: "b", Provider: "codex"}); err != nil {
		t.Fatal(err)
	}
	if _, handled, err = s.PickAuth(context.Background(), candidates); !handled || !errors.Is(err, store.ErrConcurrentLimit) {
		t.Fatalf("all busy = handled=%t err=%v", handled, err)
	}
}

func TestPickAuthBindsOldestUnattributedReservation(t *testing.T) {
	s := quotaService(t)
	if err := s.Store().UpsertAuthConcurrencyLimit(context.Background(), "claude", "auth-1", 3); err != nil {
		t.Fatal(err)
	}
	s.TrackAuthCapture("res-1", "claude-sonnet")
	id, handled, err := s.PickAuth(context.Background(), []AuthPickCandidate{{ID: "auth-1", Provider: "claude"}})
	if err != nil || !handled || id != "auth-1" {
		t.Fatalf("pick = (%q, %t, %v)", id, handled, err)
	}
	s.authMu.Lock()
	got := s.activeAuthRequestsLocked("claude", "auth-1", "")
	s.authMu.Unlock()
	if got != 1 {
		t.Fatalf("active = %d", got)
	}
}
