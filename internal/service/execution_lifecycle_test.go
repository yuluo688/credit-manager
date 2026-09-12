package service

import (
	"context"
	"github.com/yuluo688/credit-manager/internal/config"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
	"testing"
)

func TestFinishExecutionAfterCancellationKeepsLateAuthUsage(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	svc, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "test-cancel", 10_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := svc.BuildReservePlan(ctx, "cancel-model", []byte(`{"max_tokens":16,"input":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	r, err := svc.Reserve(ctx, key, plan, "cancel")
	if err != nil {
		t.Fatal(err)
	}
	svc.TrackAuthCapture(r.ID, plan.Model)
	auth := store.AuthIdentity{Provider: "codex", AuthID: "test-auth"}
	if err := svc.AdmitAuth(ctx, r.ID, auth); err != nil {
		t.Fatal(err)
	}
	svc.authMu.Lock()
	before := svc.activeAuthRequestsLocked("codex", "test-auth", "")
	svc.authMu.Unlock()
	if before != 1 {
		t.Fatalf("auth count = %d", before)
	}
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	for i := 0; i < 2; i++ {
		if err := svc.FinishExecution(cancelCtx, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	svc.authMu.Lock()
	after := svc.activeAuthRequestsLocked("codex", "test-auth", "")
	svc.authMu.Unlock()
	if after != 0 {
		t.Fatalf("finished auth count = %d", after)
	}
	svc.ObserveHostUsage(r.CreatedAt, auth, money.TokenUsage{Input: 7, Output: 2}, plan.Model)
	usage, ok := svc.CapturedHostUsage(r.ID)
	if !ok || usage.Input != 7 {
		t.Fatal("finishing execution discarded late usage correlation")
	}
}
