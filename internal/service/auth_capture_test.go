package service

import (
	"context"
	"testing"
	"time"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
)

func TestObserveHostUsageMatchesAliasBeforeSettle(t *testing.T) {
	svc := &Service{authPending: map[string]*pendingAuthCapture{}}
	svc.TrackAuthCapture("res-1", "claude-sonnet")

	auth := store.AuthIdentity{AuthID: "auth-1", AuthIndex: "idx-1", Provider: "claude", Label: "work"}
	usage := money.TokenUsage{Input: 10, Output: 4}
	ledgerID, ok := svc.ObserveHostUsage(time.Now(), auth, usage, "claude-sonnet-4", "claude-sonnet")
	if !ok {
		t.Fatal("expected correlation via alias")
	}
	if ledgerID != "" {
		t.Fatalf("ledger id = %q, want empty before settle", ledgerID)
	}

	got := svc.AuthForSettlement("res-1", "ledger-1")
	if got.Label != "work" || got.Provider != "claude" {
		t.Fatalf("auth = %#v", got)
	}
	if _, still := svc.authPending["res-1"]; still {
		t.Fatal("expected pending cleared after settle with auth+usage")
	}
}

func TestObserveHostUsageBackfillsAuthAfterSettle(t *testing.T) {
	svc := &Service{authPending: map[string]*pendingAuthCapture{}}
	svc.TrackAuthCapture("res-1b", "claude-sonnet")

	if got := svc.AuthForSettlement("res-1b", "ledger-1b"); !got.Empty() {
		t.Fatalf("auth before usage = %#v", got)
	}

	auth := store.AuthIdentity{AuthID: "auth-1b", Provider: "claude", Label: "work"}
	usage := money.TokenUsage{Input: 10, Output: 4}
	ledgerID, ok := svc.ObserveHostUsage(time.Now(), auth, usage, "claude-sonnet-4", "claude-sonnet")
	if !ok || ledgerID != "ledger-1b" {
		t.Fatalf("post-settle observe = (%q, %v)", ledgerID, ok)
	}
	if _, still := svc.authPending["res-1b"]; still {
		t.Fatal("expected pending cleared after auth+usage")
	}
}

func TestObserveHostUsageMatchesBuildSuffixAmongMultiplePending(t *testing.T) {
	svc := &Service{authPending: map[string]*pendingAuthCapture{}}
	svc.TrackAuthCapture("res-old", "gpt-5.6-sol")
	_ = svc.AuthForSettlement("res-old", "ledger-old")
	svc.TrackAuthCapture("res-grok", "grok-4.6")
	_ = svc.AuthForSettlement("res-grok", "ledger-grok")

	usage := money.TokenUsage{Input: 224, Output: 152, CacheRead: 128}
	ledgerID, ok := svc.ObserveHostUsage(time.Now(), store.AuthIdentity{Provider: "xai"}, usage, "grok-4.6-build")
	if !ok || ledgerID != "ledger-grok" {
		t.Fatalf("observe = (%q, %v), want ledger-grok", ledgerID, ok)
	}
}

func TestObserveHostUsageLooseMatchWhenModelRewritten(t *testing.T) {
	svc := &Service{authPending: map[string]*pendingAuthCapture{}}
	svc.TrackAuthCapture("res-2", "gpt-alias")

	auth := store.AuthIdentity{AuthID: "auth-2", Provider: "openai"}
	usage := money.TokenUsage{Input: 3, Output: 1}
	_, ok := svc.ObserveHostUsage(time.Now(), auth, usage, "gpt-4o-2024-11-20")
	if !ok {
		t.Fatal("expected loose time-window match when model was rewritten")
	}
	got := svc.AuthForSettlement("res-2", "ledger-2")
	if got.AuthID != "auth-2" {
		t.Fatalf("auth = %#v", got)
	}
}

func TestObserveHostUsageAllowsLateAuthAfterUsageWithoutCredentials(t *testing.T) {
	svc := &Service{authPending: map[string]*pendingAuthCapture{}}
	svc.TrackAuthCapture("res-3", "model-a")
	_ = svc.AuthForSettlement("res-3", "ledger-3")

	usage := money.TokenUsage{Input: 8, Output: 2}
	ledgerID, ok := svc.ObserveHostUsage(time.Now(), store.AuthIdentity{}, usage, "model-a")
	if !ok || ledgerID != "ledger-3" {
		t.Fatalf("usage-only observe = (%q, %v)", ledgerID, ok)
	}
	if _, still := svc.authPending["res-3"]; !still {
		t.Fatal("pending should remain until auth arrives")
	}

	auth := store.AuthIdentity{AuthID: "auth-3", Label: "ops", Provider: "codex"}
	ledgerID, ok = svc.ObserveHostUsage(time.Now(), auth, money.TokenUsage{}, "model-a")
	if !ok || ledgerID != "ledger-3" {
		t.Fatalf("late auth observe = (%q, %v)", ledgerID, ok)
	}
	if _, still := svc.authPending["res-3"]; still {
		t.Fatal("expected pending cleared after late auth")
	}
}

func TestObserveHostUsageMatchesRequestedAtNotNewest(t *testing.T) {
	svc := &Service{authPending: map[string]*pendingAuthCapture{}}
	older := time.Now().Add(-2 * time.Second)
	newer := time.Now().Add(-200 * time.Millisecond)
	svc.authPending["res-old"] = &pendingAuthCapture{reservationID: "res-old", models: []string{"grok-4.6"}, startedAt: older, ledgerID: "ledger-old"}
	svc.authPending["res-new"] = &pendingAuthCapture{reservationID: "res-new", models: []string{"grok-4.6"}, startedAt: newer}

	auth := store.AuthIdentity{AuthID: "auth-old", Provider: "xai", Label: "ops"}
	usage := money.TokenUsage{Input: 11, Output: 2}
	ledgerID, ok := svc.ObserveHostUsage(older, auth, usage, "grok-4.6")
	if !ok || ledgerID != "ledger-old" {
		t.Fatalf("matched (%q, %v), want ledger-old", ledgerID, ok)
	}
}

func TestObserveHostUsageMatchesSolePendingWithoutModel(t *testing.T) {
	svc := &Service{authPending: map[string]*pendingAuthCapture{}}
	svc.TrackAuthCapture("res-only", "client-alias")
	auth := store.AuthIdentity{AuthID: "auth-only", Provider: "xai", Label: "ops"}
	_, ok := svc.ObserveHostUsage(time.Time{}, auth, money.TokenUsage{Input: 4, Output: 1}, "upstream-rewritten-id")
	if !ok {
		t.Fatal("expected sole pending to match rewritten model")
	}
	got := svc.AuthForSettlement("res-only", "ledger-only")
	if got.Label != "ops" {
		t.Fatalf("auth = %#v", got)
	}
}

func TestWaitForHostUsageReceivesLateCallback(t *testing.T) {
	svc := &Service{}
	svc.TrackAuthCapture("res-wait", "grok-4.6")
	go func() {
		time.Sleep(30 * time.Millisecond)
		svc.ObserveHostUsage(time.Now(), store.AuthIdentity{AuthID: "a", Provider: "xai"}, money.TokenUsage{Input: 2, Output: 1}, "grok-4.6")
	}()
	usage, ok := svc.WaitForHostUsage(context.Background(), "res-wait", 400*time.Millisecond)
	if !ok || usage.Input != 2 || usage.Output != 1 {
		t.Fatalf("wait = (%#v, %v)", usage, ok)
	}
}

func TestObserveHostUsageMatchesReportedTotalOnly(t *testing.T) {
	svc := &Service{authPending: map[string]*pendingAuthCapture{}}
	svc.TrackAuthCapture("res-glm", "glm-5.3-flash")
	_, ok := svc.ObserveHostUsage(time.Now(), store.AuthIdentity{}, money.TokenUsage{ReportedTotal: 64}, "glm-5.3-flash")
	if !ok {
		t.Fatal("expected official total_tokens to correlate usage")
	}
	got, captured := svc.CapturedHostUsage("res-glm")
	if !captured || got.ReportedTotal != 64 {
		t.Fatalf("captured = (%#v, %v)", got, captured)
	}
}
