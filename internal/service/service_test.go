package service

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/yuluo688/credit-manager/internal/config"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
	"github.com/yuluo688/credit-manager/internal/usageparse"
)

func TestApplyHostUsageRepricesReservedFallback(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CREDIT_MANAGER_TEST_PEPPERS", "active:0123456789abcdef0123456789abcdef")
	cfg := config.Default()
	cfg.DataDir = dir
	cfg.Keys.PepperEnv = "CREDIT_MANAGER_TEST_PEPPERS"
	cfg.Keys.ActivePepperID = "active"
	cfg.Settlement.HostUsageWait = 0
	svc, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open service: %v", err)
	}
	defer svc.Close()

	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "all", MatchKind: store.MatchGlob, Pattern: "*", Priority: 1, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 3_000_000},
	}); err != nil {
		t.Fatalf("put pricing: %v", err)
	}
	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "test", 10_000_000, nil)
	if err != nil {
		t.Fatalf("mint key: %v", err)
	}
	plan, err := svc.BuildReservePlan(ctx, "gpt-test", []byte(`{"model":"gpt-test","max_tokens":4096,"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.Amount <= 3 {
		t.Fatalf("reserve amount = %d, want conservative estimate", plan.Amount)
	}
	reservation, err := svc.Reserve(ctx, key, plan, "reprice")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if err := svc.SettleFromUsage(ctx, reservation, plan, usageparse.Result{}, "openai", store.UsageMetrics{}); err != nil {
		t.Fatalf("settle reserved fallback: %v", err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 1})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	if entries[0].Source != "reserved_fallback" || entries[0].CostMicroUSD != 0 || entries[0].Usage.Output != 0 {
		t.Fatalf("fallback entry = %#v", entries[0])
	}
	if entries[0].EstimatedCostMicroUSD != reservation.HeldMicroUSD {
		t.Fatalf("fallback estimated cost = %d, want held %d", entries[0].EstimatedCostMicroUSD, reservation.HeldMicroUSD)
	}

	if err := svc.ApplyHostUsage(ctx, entries[0].ID, money.TokenUsage{Input: 10, Output: 20}); err != nil {
		t.Fatalf("apply host usage: %v", err)
	}
	updated, err := svc.Store().GetUsage(ctx, entries[0].ID)
	if err != nil {
		t.Fatalf("get usage: %v", err)
	}
	want, err := money.CostFor(money.TokenUsage{Input: 10, Output: 20}, plan.Price, "gpt-test", "")
	if err != nil {
		t.Fatalf("cost: %v", err)
	}
	if updated.Usage.Input != 10 || updated.Usage.Output != 20 || updated.CostMicroUSD != want || updated.Source != "host_usage" {
		t.Fatalf("repriced usage = %#v, want cost %d", updated, want)
	}
	updatedKey, err := svc.Store().GetPluginKey(ctx, key.ID)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}
	if updatedKey.SettledSpendMicroUSD != want {
		t.Fatalf("settled spend = %d, want %d", updatedKey.SettledSpendMicroUSD, want)
	}
}

func TestApplyHostUsageDoesNotDoubleCountOpenAICache(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CREDIT_MANAGER_TEST_PEPPERS", "active:0123456789abcdef0123456789abcdef")
	cfg := config.Default()
	cfg.DataDir = dir
	cfg.Keys.PepperEnv = "CREDIT_MANAGER_TEST_PEPPERS"
	cfg.Keys.ActivePepperID = "active"
	cfg.Settlement.HostUsageWait = 0
	svc, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open service: %v", err)
	}
	defer svc.Close()

	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "gpt", MatchKind: store.MatchExact, Pattern: "gpt-4o", Priority: 10, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 2_000_000, CacheRead: 100_000},
	}); err != nil {
		t.Fatalf("put pricing: %v", err)
	}
	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "test", 10_000_000, nil)
	if err != nil {
		t.Fatalf("mint key: %v", err)
	}
	plan, err := svc.BuildReservePlan(ctx, "gpt-4o", []byte(`{"model":"gpt-4o","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	reservation, err := svc.Reserve(ctx, key, plan, "openai-cache")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if err := svc.SettleFromUsage(ctx, reservation, plan, usageparse.Result{}, "openai", store.UsageMetrics{}); err != nil {
		t.Fatalf("settle reserved fallback: %v", err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 1})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}

	usage := money.TokenUsage{Input: 100_000, Output: 5_000, Cached: 20_000, CacheRead: 20_000}
	if err := svc.ApplyHostUsage(ctx, entries[0].ID, usage); err != nil {
		t.Fatalf("apply host usage: %v", err)
	}
	updated, err := svc.Store().GetUsage(ctx, entries[0].ID)
	if err != nil {
		t.Fatalf("get usage: %v", err)
	}
	want, err := money.CostFor(usage, plan.Price, "gpt-4o", "")
	if err != nil {
		t.Fatalf("cost: %v", err)
	}
	if want != 92_000 {
		t.Fatalf("openai cache cost = %d, want 92000", want)
	}
	if updated.CostMicroUSD != want {
		t.Fatalf("settled cost = %d, want %d (double-counted cache?)", updated.CostMicroUSD, want)
	}
}

func TestSettleFromUsageWaitsForHostCallback(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CREDIT_MANAGER_TEST_PEPPERS", "active:0123456789abcdef0123456789abcdef")
	cfg := config.Default()
	cfg.DataDir = dir
	cfg.Keys.PepperEnv = "CREDIT_MANAGER_TEST_PEPPERS"
	cfg.Keys.ActivePepperID = "active"
	cfg.Settlement.HostUsageWait = 400 * time.Millisecond
	svc, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open service: %v", err)
	}
	defer svc.Close()

	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "all", MatchKind: store.MatchGlob, Pattern: "*", Priority: 1, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 3_000_000},
	}); err != nil {
		t.Fatalf("put pricing: %v", err)
	}
	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "test", 10_000_000, nil)
	if err != nil {
		t.Fatalf("mint key: %v", err)
	}
	plan, err := svc.BuildReservePlan(ctx, "grok-4.6", []byte(`{"model":"grok-4.6","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	reservation, err := svc.Reserve(ctx, key, plan, "wait-host")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	svc.TrackAuthCapture(reservation.ID, plan.Model)
	go func() {
		time.Sleep(40 * time.Millisecond)
		svc.ObserveHostUsage(time.Now(), store.AuthIdentity{AuthID: "auth-x", Provider: "xai", Label: "ops"}, money.TokenUsage{Input: 9, Output: 3}, "grok-4.6")
	}()
	if err := svc.SettleFromUsage(ctx, reservation, plan, usageparse.Result{}, "openai", store.UsageMetrics{}); err != nil {
		t.Fatalf("settle: %v", err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 1})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	if entries[0].Source != "host_usage" || entries[0].Usage.Input != 9 || entries[0].Auth.Label != "ops" {
		t.Fatalf("settled entry = %#v", entries[0])
	}
}

func TestBuildReservePlanPerImageUsesRequestCount(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CREDIT_MANAGER_TEST_PEPPERS", "active:0123456789abcdef0123456789abcdef")
	cfg := config.Default()
	cfg.DataDir = dir
	cfg.Keys.PepperEnv = "CREDIT_MANAGER_TEST_PEPPERS"
	cfg.Keys.ActivePepperID = "active"
	svc, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open service: %v", err)
	}
	defer svc.Close()

	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "gpt-image-1", MatchKind: store.MatchExact, Pattern: "gpt-image-1", Priority: 100, Enabled: true,
		Price: money.PricePerMTok{BillingMode: money.BillingPerImage, PerImage: 40_000, Input: 5_000_000},
	}); err != nil {
		t.Fatalf("put pricing: %v", err)
	}
	plan, err := svc.BuildReservePlan(ctx, "gpt-image-1", []byte(`{"model":"gpt-image-1","n":3,"prompt":"cat"}`))
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.ImageCount != 3 || plan.Amount != 120_000 || plan.TokenEstimate != 3 {
		t.Fatalf("image plan = %#v", plan)
	}

	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "image", 10_000_000, nil)
	if err != nil {
		t.Fatalf("mint key: %v", err)
	}
	reservation, err := svc.Reserve(ctx, key, plan, "image-n")
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if err := svc.SettleFromUsage(ctx, reservation, plan, usageparse.Result{}, "openai", store.UsageMetrics{}); err != nil {
		t.Fatalf("settle: %v", err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 1})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	if entries[0].CostMicroUSD != 120_000 {
		t.Fatalf("image cost = %d, want 120000", entries[0].CostMicroUSD)
	}
}

func TestOpenHandsDatabaseToReplacementInstance(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CREDIT_MANAGER_TEST_PEPPERS", "active:0123456789abcdef0123456789abcdef")
	cfg := config.Default()
	cfg.DataDir = dir
	cfg.Keys.PepperEnv = "CREDIT_MANAGER_TEST_PEPPERS"
	cfg.Keys.ActivePepperID = "active"
	cfg.Settlement.HostUsageWait = 0

	first, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open first service: %v", err)
	}
	t.Cleanup(func() { _ = first.Close() })

	second, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open replacement service: %v", err)
	}
	defer second.Close()

	if _, err := first.Store().GetCaller(ctx, BootstrapCallerID); err == nil {
		t.Fatal("retired service still serves queries")
	}
	if _, err := second.Store().GetCaller(ctx, BootstrapCallerID); err != nil {
		t.Fatalf("replacement service lost bootstrap caller: %v", err)
	}
}

func TestConfigureReusesSameDatabaseWithoutHandover(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CREDIT_MANAGER_TEST_PEPPERS", "active:0123456789abcdef0123456789abcdef")
	raw := []byte("data_dir: " + strconv.Quote(filepath.ToSlash(dir)) + "\nkeys:\n  pepper_env: CREDIT_MANAGER_TEST_PEPPERS\n  active_pepper_id: active\n")
	t.Cleanup(Shutdown)

	if err := Configure(ctx, raw); err != nil {
		t.Fatalf("first configure: %v", err)
	}
	first := Current()
	if first == nil {
		t.Fatal("current service is nil")
	}
	if err := Configure(ctx, raw); err != nil {
		t.Fatalf("reconfigure: %v", err)
	}
	second := Current()
	if second == nil || second.Store() != first.Store() {
		t.Fatal("reconfigure opened a second store")
	}
	if _, err := second.Store().GetCaller(ctx, BootstrapCallerID); err != nil {
		t.Fatalf("reused store lost bootstrap caller: %v", err)
	}
}
