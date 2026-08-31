package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
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
		svc.ObserveHostUsageWithExecutor(time.Now(), store.AuthIdentity{AuthID: "auth-x", Provider: "xai", Label: "ops"}, money.TokenUsage{Input: 9, Output: 3}, "XAIExecutor", "grok-4.6")
	}()
	if err := svc.SettleFromUsage(ctx, reservation, plan, usageparse.Result{}, "openai", store.UsageMetrics{}); err != nil {
		t.Fatalf("settle: %v", err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 1})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	if entries[0].Source != "host_usage" || entries[0].Usage.Input != 9 || entries[0].Auth.Label != "ops" || entries[0].ExecutorType != "XAIExecutor" {
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

func TestBuildReservePlanDisabledModel(t *testing.T) {
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
		ID: "all", MatchKind: store.MatchGlob, Pattern: "*", Priority: 1, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 2_000_000},
	}); err != nil {
		t.Fatalf("put glob: %v", err)
	}
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "gpt-4o", MatchKind: store.MatchExact, Pattern: "gpt-4o", Priority: 100, Enabled: false,
		Price: money.PricePerMTok{Input: 2_500_000, Output: 10_000_000},
	}); err != nil {
		t.Fatalf("put exact: %v", err)
	}

	if _, err := svc.BuildReservePlan(ctx, "gpt-4o", []byte(`{"model":"gpt-4o","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)); !errors.Is(err, store.ErrModelDisabled) {
		t.Fatalf("disabled model error = %v, want %v", err, store.ErrModelDisabled)
	}
}

func TestFilterModelDirectoryRemovesDisabled(t *testing.T) {
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
		ID: "all", MatchKind: store.MatchGlob, Pattern: "*", Priority: 1, Enabled: true,
		Price: money.PricePerMTok{Input: 1},
	}); err != nil {
		t.Fatalf("put glob: %v", err)
	}
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "gpt-4o", MatchKind: store.MatchExact, Pattern: "gpt-4o", Priority: 100, Enabled: false,
		Price: money.PricePerMTok{Input: 1},
	}); err != nil {
		t.Fatalf("put exact: %v", err)
	}

	body := []byte(`{"object":"list","data":[{"id":"gpt-4o","object":"model"},{"id":"claude-sonnet","object":"model"}]}`)
	filtered, changed, err := svc.FilterModelDirectory(ctx, body)
	if err != nil || !changed {
		t.Fatalf("filter = %s changed=%t err=%v", filtered, changed, err)
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(filtered, &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].ID != "claude-sonnet" {
		t.Fatalf("filtered = %#v", payload.Data)
	}

	keyFiltered, keyChanged, err := svc.FilterModelDirectoryForKey(ctx, body, store.PluginKey{AllowedModels: []string{"gpt-4o"}})
	if err != nil || !keyChanged {
		t.Fatalf("key filter = %s changed=%t err=%v", keyFiltered, keyChanged, err)
	}
	if err := json.Unmarshal(keyFiltered, &payload); err != nil {
		t.Fatalf("decode key filter: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].ID != "claude-sonnet" {
		t.Fatalf("directory list should only drop globally disabled models, got %#v", payload.Data)
	}

	allowedBody := []byte(`{"object":"list","data":[{"id":"claude-sonnet","object":"model"},{"id":"other","object":"model"}]}`)
	allowedFiltered, allowedChanged, err := svc.FilterModelDirectoryForKey(ctx, allowedBody, store.PluginKey{AllowedModels: []string{"claude-sonnet"}})
	if err != nil {
		t.Fatalf("allowlist filter err=%v", err)
	}
	if allowedChanged {
		t.Fatalf("allowlist must not clip GET /v1/models, got %s", allowedFiltered)
	}

	hidden := svc.HiddenDirectoryModels(ctx, []string{"gpt-4o", "claude-sonnet", "other"})
	if len(hidden) != 1 || hidden[0] != "gpt-4o" {
		t.Fatalf("hidden = %#v", hidden)
	}
}

func TestFilterModelDirectoryHidesGlobDisabled(t *testing.T) {
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
		ID: "all", MatchKind: store.MatchGlob, Pattern: "*", Priority: 1, Enabled: true,
		Price: money.PricePerMTok{Input: 1},
	}); err != nil {
		t.Fatalf("put glob all: %v", err)
	}
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "claude", MatchKind: store.MatchGlob, Pattern: "claude-*", Priority: 50, Enabled: false,
		Price: money.PricePerMTok{Input: 1},
	}); err != nil {
		t.Fatalf("put glob claude: %v", err)
	}

	body := []byte(`{"object":"list","data":[{"id":"gpt-4o"},{"id":"claude-sonnet"},{"id":"claude-opus"}]}`)
	filtered, changed, err := svc.FilterModelDirectory(ctx, body)
	if err != nil || !changed {
		t.Fatalf("filter = %s changed=%t err=%v", filtered, changed, err)
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(filtered, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Data) != 1 || payload.Data[0].ID != "gpt-4o" {
		t.Fatalf("filtered = %#v", payload.Data)
	}
	hidden := svc.HiddenDirectoryModels(ctx, []string{"gpt-4o", "claude-sonnet", "claude-opus"})
	if len(hidden) != 2 || hidden[0] != "claude-opus" || hidden[1] != "claude-sonnet" {
		t.Fatalf("hidden = %#v", hidden)
	}
	disabled, err := svc.disabledDirectoryModels(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(disabled) != 2 || disabled[0] != "claude-opus" || disabled[1] != "claude-sonnet" {
		t.Fatalf("sync hidden = %#v", disabled)
	}
}

func TestMergeAuthExcludedModelsPreservesUserExclusions(t *testing.T) {
	raw := []byte(`{"type":"gemini","excluded_models":["keep-me","old-hidden"],"credit_manager_excluded_models":["old-hidden"],"token":"abc&def","expires":1712345678901,"nested":{"n":1,"s":"x<y"}}`)
	out, changed, err := MergeAuthExcludedModels(raw, []string{"gpt-4o", "old-hidden"})
	if err != nil || !changed {
		t.Fatalf("merge changed=%t err=%v", changed, err)
	}
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(out, &meta); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(meta["token"]) != `"abc&def"` {
		t.Fatalf("token rewritten: %s", meta["token"])
	}
	if string(meta["expires"]) != "1712345678901" {
		t.Fatalf("expires rewritten: %s", meta["expires"])
	}
	if string(meta["nested"]) != `{"n":1,"s":"x<y"}` {
		t.Fatalf("nested rewritten: %s", meta["nested"])
	}
	got := stringSliceFromRaw(meta["excluded_models"])
	if !stringSlicesEqual(got, []string{"gpt-4o", "keep-me", "old-hidden"}) {
		t.Fatalf("excluded_models = %#v", got)
	}
	if idx := bytes.Index(out, []byte(`"type"`)); idx < 0 || bytes.Index(out, []byte(`"token"`)) < idx {
		t.Fatalf("top-level key order changed: %s", out)
	}
	if _, changedAgain, err := MergeAuthExcludedModels(out, []string{"gpt-4o", "old-hidden"}); err != nil || changedAgain {
		t.Fatalf("second merge changed=%t err=%v", changedAgain, err)
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

func lunaPricingRule() store.PricingRule {
	return store.PricingRule{
		ID: "gpt-5.6-luna", MatchKind: store.MatchExact, Pattern: "gpt-5.6-luna", Priority: 100, Enabled: true,
		Price: money.PricePerMTok{Input: 200_000, Output: 1_200_000, CacheRead: 20_000},
		Tiers: []money.PriceTier{
			{Kind: money.PriceTierContext, Label: "272K", Threshold: 272_000, Price: money.PricePerMTok{Input: 400_000, Output: 1_800_000, CacheRead: 40_000}},
			{Kind: money.PriceTierService, Service: "fast,priority", Price: money.PricePerMTok{Input: 400_000, Output: 2_400_000}},
		},
	}
}

func TestSettleUsesContextTierWhenPromptHits272K(t *testing.T) {
	ctx := context.Background()
	svc := openTestService(t)
	defer svc.Close()
	if err := svc.Store().PutPricingRule(ctx, lunaPricingRule()); err != nil {
		t.Fatal(err)
	}
	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "tier", 10_000_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := svc.BuildReservePlan(ctx, "gpt-5.6-luna", []byte(`{"model":"gpt-5.6-luna","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := svc.Reserve(ctx, key, plan, "ctx-272k")
	if err != nil {
		t.Fatal(err)
	}
	usage := money.TokenUsage{Input: 272_000, Output: 1_000}
	parsed := usageparse.Result{Found: true, Usage: usage, Source: "openai"}
	if err := svc.SettleFromUsage(ctx, reservation, plan, parsed, "openai", store.UsageMetrics{}); err != nil {
		t.Fatal(err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 1})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	want, err := money.CostForTiered(usage, plan.Price, plan.Tiers, plan.Model, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].CostMicroUSD != want {
		t.Fatalf("cost = %d, want 272k card %d", entries[0].CostMicroUSD, want)
	}
	base, err := money.CostFor(usage, plan.Price, plan.Model, "")
	if err != nil {
		t.Fatal(err)
	}
	if want == base {
		t.Fatal("272k settlement used the default card")
	}
}

func TestSettleUsesResponseServiceTierNotRequest(t *testing.T) {
	ctx := context.Background()
	svc := openTestService(t)
	defer svc.Close()
	if err := svc.Store().PutPricingRule(ctx, lunaPricingRule()); err != nil {
		t.Fatal(err)
	}
	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "tier", 10_000_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"model":"gpt-5.6-luna","service_tier":"priority","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	plan, err := svc.BuildReservePlan(ctx, "gpt-5.6-luna", body)
	if err != nil {
		t.Fatal(err)
	}
	usage := money.TokenUsage{Input: 1_000, Output: 500}
	priorityHold, err := money.CostForTiered(money.TokenUsage{Input: plan.InputEstimate, Output: plan.OutputEstimate}, plan.Price, plan.Tiers, plan.Model, "", "priority")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Amount != priorityHold {
		t.Fatalf("reserve amount = %d, want priority hold %d", plan.Amount, priorityHold)
	}
	reservation, err := svc.Reserve(ctx, key, plan, "resp-tier")
	if err != nil {
		t.Fatal(err)
	}
	parsed := usageparse.Result{Found: true, Usage: usage, Source: "openai", ServiceTier: "default"}
	requestTier := "priority"
	metrics := store.UsageMetrics{Tier: &requestTier}
	if err := svc.SettleFromUsage(ctx, reservation, plan, parsed, "openai", metrics); err != nil {
		t.Fatal(err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 1})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	want, err := money.CostFor(usage, plan.Price, plan.Model, "")
	if err != nil {
		t.Fatal(err)
	}
	priorityCost, err := money.CostForTiered(usage, plan.Price, plan.Tiers, plan.Model, "", "priority")
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].CostMicroUSD != want {
		t.Fatalf("cost = %d, want default card %d (not request priority %d)", entries[0].CostMicroUSD, want, priorityCost)
	}
	if entries[0].Metrics.Tier == nil || *entries[0].Metrics.Tier != "default" {
		t.Fatalf("stored tier = %#v, want response default", entries[0].Metrics.Tier)
	}
}

func TestApplyHostUsageRepricesWithContextTier(t *testing.T) {
	ctx := context.Background()
	svc := openTestService(t)
	defer svc.Close()
	if err := svc.Store().PutPricingRule(ctx, lunaPricingRule()); err != nil {
		t.Fatal(err)
	}
	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "tier", 10_000_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := svc.BuildReservePlan(ctx, "gpt-5.6-luna", []byte(`{"model":"gpt-5.6-luna","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := svc.Reserve(ctx, key, plan, "host-272k")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SettleFromUsage(ctx, reservation, plan, usageparse.Result{}, "openai", store.UsageMetrics{}); err != nil {
		t.Fatal(err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 1})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	usage := money.TokenUsage{Input: 280_000, Output: 800}
	if err := svc.ApplyHostUsage(ctx, entries[0].ID, usage); err != nil {
		t.Fatal(err)
	}
	updated, err := svc.Store().GetUsage(ctx, entries[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	want, err := money.CostForTiered(usage, plan.Price, plan.Tiers, plan.Model, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if updated.CostMicroUSD != want {
		t.Fatalf("host reprice = %d, want 272k card %d", updated.CostMicroUSD, want)
	}
}

func TestApplyHostUsageRecordIgnoresUnconfirmedRequestTier(t *testing.T) {
	ctx := context.Background()
	svc := openTestService(t)
	defer svc.Close()
	if err := svc.Store().PutPricingRule(ctx, lunaPricingRule()); err != nil {
		t.Fatal(err)
	}
	key, _, err := svc.MintKey(ctx, BootstrapCallerID, "tier", 10_000_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"model":"gpt-5.6-luna","service_tier":"priority","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`)
	plan, err := svc.BuildReservePlan(ctx, "gpt-5.6-luna", body)
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := svc.Reserve(ctx, key, plan, "host-default")
	if err != nil {
		t.Fatal(err)
	}
	requestTier := "priority"
	if err := svc.SettleFromUsage(ctx, reservation, plan, usageparse.Result{}, "openai", store.UsageMetrics{Tier: &requestTier}); err != nil {
		t.Fatal(err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 1})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	usage := money.TokenUsage{Input: 1_000, Output: 500}
	if err := svc.ApplyHostUsageRecord(ctx, entries[0].ID, usage, ""); err != nil {
		t.Fatal(err)
	}
	updated, err := svc.Store().GetUsage(ctx, entries[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	want, err := money.CostFor(usage, plan.Price, plan.Model, "")
	if err != nil {
		t.Fatal(err)
	}
	priorityCost, err := money.CostForTiered(usage, plan.Price, plan.Tiers, plan.Model, "", "priority")
	if err != nil {
		t.Fatal(err)
	}
	if updated.CostMicroUSD != want {
		t.Fatalf("host reprice without confirmed tier = %d, want default %d (not request priority %d)", updated.CostMicroUSD, want, priorityCost)
	}
}

func TestBuildReservePlanDoesNotApplyContextTierToBodyEstimate(t *testing.T) {
	ctx := context.Background()
	svc := openTestService(t)
	defer svc.Close()
	if err := svc.Store().PutPricingRule(ctx, lunaPricingRule()); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"model": "gpt-5.6-luna", "max_tokens": 16,
		"messages": []map[string]string{{"role": "user", "content": strings.Repeat("a", 600_000)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := svc.BuildReservePlan(ctx, "gpt-5.6-luna", payload)
	if err != nil {
		t.Fatal(err)
	}
	if plan.InputEstimate < 272_000 {
		t.Fatalf("expected oversized body estimate, got in=%d", plan.InputEstimate)
	}
	usage := money.TokenUsage{Input: plan.InputEstimate, Output: plan.OutputEstimate}
	base, err := money.CostFor(usage, plan.Price, plan.Model, "")
	if err != nil {
		t.Fatal(err)
	}
	contextCost, err := money.CostForTiered(usage, plan.Price, plan.Tiers, plan.Model, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Amount != base {
		t.Fatalf("reserve amount = %d, want base card %d (not 272k estimate %d)", plan.Amount, base, contextCost)
	}
}

func openTestService(t *testing.T) *Service {
	t.Helper()
	ctx := context.Background()
	t.Setenv("CREDIT_MANAGER_TEST_PEPPERS", "active:0123456789abcdef0123456789abcdef")
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	cfg.Keys.PepperEnv = "CREDIT_MANAGER_TEST_PEPPERS"
	cfg.Keys.ActivePepperID = "active"
	cfg.Settlement.HostUsageWait = 0
	svc, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open service: %v", err)
	}
	return svc
}
