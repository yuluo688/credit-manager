package service

import (
	"context"
	"testing"

	"github.com/yuluo688/credit-manager/internal/config"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
	"github.com/yuluo688/credit-manager/internal/usageparse"
)

func TestFinishExecutionKeepsLateAuthUsage(t *testing.T) {
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
	reservation, err := svc.Reserve(ctx, key, plan, "cancel")
	if err != nil {
		t.Fatal(err)
	}
	svc.TrackAuthCapture(reservation.ID, plan.Model)
	auth := store.AuthIdentity{Provider: "codex", AuthID: "test-auth"}
	if err := svc.AdmitAuth(ctx, reservation.ID, auth); err != nil {
		t.Fatal(err)
	}
	if err := svc.FinishExecution(ctx, reservation.ID); err != nil {
		t.Fatal(err)
	}
	svc.authMu.Lock()
	active := svc.activeAuthRequestsLocked(auth.Provider, auth.AuthID, "")
	svc.authMu.Unlock()
	if active != 0 {
		t.Fatalf("finished auth count = %d", active)
	}
	svc.ObserveHostUsage(reservation.CreatedAt, auth, money.TokenUsage{Input: 7, Output: 2}, plan.Model)
	usage, ok := svc.CapturedHostUsage(reservation.ID)
	if !ok || usage.Input != 7 {
		t.Fatal("finishing execution discarded late usage correlation")
	}
}

func TestFinishClientExecutionKeepsAuthConcurrency(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	svc, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "client-finish", 10_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := svc.BuildReservePlan(ctx, "client-finish-model", []byte(`{"max_tokens":16}`))
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := svc.Reserve(ctx, key, plan, "client-finish")
	if err != nil {
		t.Fatal(err)
	}
	auth := store.AuthIdentity{Provider: "codex", AuthID: "client-finish-auth"}
	svc.TrackAuthCapture(reservation.ID, plan.Model)
	if err := svc.AdmitAuth(ctx, reservation.ID, auth); err != nil {
		t.Fatal(err)
	}
	if err := svc.FinishClientExecution(ctx, reservation.ID); err != nil {
		t.Fatal(err)
	}
	overview, err := svc.store.GetKeyUsageOverview(ctx, key.ID, reservation.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if overview.ActiveReservations != 0 {
		t.Fatalf("active key slots = %d, want 0", overview.ActiveReservations)
	}
	svc.authMu.Lock()
	active := svc.activeAuthRequestsLocked(auth.Provider, auth.AuthID, "")
	svc.authMu.Unlock()
	if active != 1 {
		t.Fatalf("auth concurrency = %d, want 1 while upstream may still run", active)
	}
}

func TestSettleAndReleaseIgnoreCanceledContext(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	svc, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "canceled-settle", 10_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := svc.BuildReservePlan(ctx, "canceled-model", []byte(`{"max_tokens":16,"input":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	settled, err := svc.Reserve(ctx, key, plan, "canceled-settle")
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := svc.SettleFromUsage(canceled, settled, plan, usageparse.Result{
		Found: true, Source: "test", Usage: money.TokenUsage{Input: 4, Output: 2},
	}, "openai", store.UsageMetrics{}); err != nil {
		t.Fatalf("settle with canceled context: %v", err)
	}
	got, err := svc.Store().GetReservation(ctx, settled.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.ReservationSettled || got.SettledMicroUSD == nil {
		t.Fatalf("settled reservation = %+v", got)
	}
	released, err := svc.Reserve(ctx, key, plan, "canceled-release")
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel = context.WithCancel(ctx)
	cancel()
	if err := svc.Release(canceled, released.ID, "settle_failed"); err != nil {
		t.Fatalf("release with canceled context: %v", err)
	}
	got, err = svc.Store().GetReservation(ctx, released.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.ReservationReleased || got.SettlementSummary != "settle_failed" {
		t.Fatalf("released reservation = %+v", got)
	}
	updatedKey, err := svc.Store().GetPluginKey(ctx, key.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updatedKey.HeldAmountMicroUSD != 0 {
		t.Fatalf("held amount = %d, want 0", updatedKey.HeldAmountMicroUSD)
	}
	if _, err := svc.cleanupStaleReservations(canceled, true); err != nil {
		t.Fatalf("cleanup with canceled context: %v", err)
	}
}

func TestReleaseIfHeldLeavesSettledReservation(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	svc, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "release-if-held", 10_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := svc.BuildReservePlan(ctx, "release-if-held", []byte(`{"max_tokens":16,"input":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	settled, err := svc.Reserve(ctx, key, plan, "release-if-held-settled")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SettleFromUsage(ctx, settled, plan, usageparse.Result{
		Found: true, Source: "test", Usage: money.TokenUsage{Input: 4, Output: 2},
	}, "openai", store.UsageMetrics{}); err != nil {
		t.Fatal(err)
	}
	svc.ReleaseIfHeld(ctx, settled.ID, "unsettled_exit")
	got, err := svc.Store().GetReservation(ctx, settled.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.ReservationSettled {
		t.Fatalf("settled reservation status = %s", got.Status)
	}
	held, err := svc.Reserve(ctx, key, plan, "release-if-held-open")
	if err != nil {
		t.Fatal(err)
	}
	svc.ReleaseIfHeld(ctx, held.ID, "unsettled_exit")
	got, err = svc.Store().GetReservation(ctx, held.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.ReservationReleased || got.SettlementSummary != "unsettled_exit" {
		t.Fatalf("held reservation = %+v", got)
	}
}

func TestOpenReleasesHeldReservationsFromPreviousProcess(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	svc, err := Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "restart-orphan", 10_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := svc.BuildReservePlan(ctx, "restart-orphan", []byte(`{"max_tokens":16,"input":"test"}`))
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := svc.Reserve(ctx, key, plan, "restart-orphan")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Close(); err != nil {
		t.Fatal(err)
	}
	svc, err = Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	got, err := svc.Store().GetReservation(ctx, reservation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.ReservationReleased || got.SettlementSummary != "process_restart" {
		t.Fatalf("orphan reservation = %+v", got)
	}
	updatedKey, err := svc.Store().GetPluginKey(ctx, key.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updatedKey.HeldAmountMicroUSD != 0 {
		t.Fatalf("held amount = %d, want 0", updatedKey.HeldAmountMicroUSD)
	}
}
