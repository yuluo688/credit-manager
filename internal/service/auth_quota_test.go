package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/yuluo688/credit-manager/internal/config"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
)

type fakeQuotaSource struct {
	files     []AuthQuotaFile
	auth      string
	responses map[string]string
	requests  []AuthQuotaHTTPRequest
	gets      int
	fail      bool
	failHTTP  bool
}

func (f *fakeQuotaSource) ListAuthQuotaFiles(context.Context) ([]AuthQuotaFile, error) {
	return f.files, nil
}
func (f *fakeQuotaSource) GetAuthQuotaJSON(context.Context, string) ([]byte, error) {
	f.gets++
	if f.fail {
		return nil, errors.New("host failed")
	}
	return []byte(f.auth), nil
}
func (f *fakeQuotaSource) DoAuthQuotaHTTP(_ context.Context, _ string, r AuthQuotaHTTPRequest) (AuthQuotaHTTPResponse, error) {
	f.requests = append(f.requests, r)
	if f.failHTTP {
		return AuthQuotaHTTPResponse{}, errors.New("upstream failed")
	}
	for k, v := range f.responses {
		if strings.Contains(r.URL, k) {
			return AuthQuotaHTTPResponse{StatusCode: http.StatusOK, Body: []byte(v)}, nil
		}
	}
	return AuthQuotaHTTPResponse{StatusCode: http.StatusNotFound}, nil
}
func quotaService(t *testing.T) *Service {
	t.Helper()
	t.Setenv("CREDIT_MANAGER_TEST_PEPPERS", "active:0123456789abcdef0123456789abcdef")
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	cfg.Keys.PepperEnv = "CREDIT_MANAGER_TEST_PEPPERS"
	cfg.Keys.ActivePepperID = "active"
	s, e := Open(context.Background(), cfg)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func quotaTestKey(t *testing.T, s *Service, label string) store.PluginKey {
	t.Helper()
	key, err := s.Store().CreatePluginKey(context.Background(), store.PluginKeySpec{
		CallerID: "default", Kid: "quota-" + label, KeyHash: []byte("quota-test-key-hash-" + label),
		PepperID: "active", Fingerprint: "quota-test", Label: label, Principal: "credit-manager:quota-" + label,
		CallerScope: "credit-manager:quota-" + label, Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return key
}
func quotaJSON(provider string) string {
	if provider == "antigravity" {
		return `{"tokens":{"access_token":"oauth-token","project_id":"project-1"}}`
	}
	if provider == "xai" {
		return `{"oauth":{"access_token":"oauth-token","user_id":"user-1"}}`
	}
	return `{"tokens":{"access_token":"oauth-token","account_id":"account-1"}}`
}

func TestAuthQuotaProviderPayloadsAndAuthorization(t *testing.T) {
	cases := []struct {
		provider, key, body string
		windows             int
	}{
		{"codex", "chatgpt.com", `{"rate_limit":{"primary_window":{"used_percent":25,"limit_window_seconds":3600,"reset_at":4102444800},"secondary_window":{"used_percent":10}},"code_review_rate_limit":{"primary_window":{"used_percent":5}},"additional_rate_limits":[{"limit_name":"Extra quota","metered_feature":"extra_feature","rate_limit":{"primary_window":{"used_percent":15}}}]}`, 4},
		{"claude", "anthropic.com", `{"five_hour":{"utilization":0.1,"resets_at":"2030-01-01T00:00:00Z"},"seven_day":{"utilization":0.2},"seven_day_oauth_apps":{"utilization":0.3},"seven_day_opus":{"utilization":0.4},"seven_day_sonnet":{"utilization":0.5},"seven_day_cowork":{"utilization":0.6},"iguana_necktie":{"utilization":0.7},"extra_usage":{"monthly_limit":100,"used_credits":10}}`, 8},
		{"antigravity", "cloudcode", `{"groups":[{"buckets":[{"bucketId":"gemini-weekly","remainingFraction":0.5,"resetTime":"2030-01-01T00:00:00Z"},{"bucketId":"3p-weekly","remainingFraction":0.25}]}]}`, 2},
		{"kimi", "api.kimi.com", `{"usage":{"used":"20","limit":"100","remaining":"80","resetTime":"2030-01-01T00:00:00Z"},"limits":[{"name":"hourly","window":{"duration":"1","timeUnit":"TIME_UNIT_HOUR"},"detail":{"used":"5","limit":"10","remaining":"5","resetIn":"3600"}},{"name":"weekly","window":{"duration":1,"timeUnit":"TIME_UNIT_WEEK"},"detail":{"used":1,"limit":4,"ttl":7200}}]}`, 3},
		{"xai", "cli-chat-proxy.grok.com", `{"config":{"monthlyLimit":{"val":10000},"used":{"val":1200},"onDemandCap":500,"onDemandUsed":10,"billingPeriodEnd":"2030-01-01T00:00:00Z"}}`, 2},
		{"xai-credits", "format=credits", `{"creditUsagePercent":3,"currentPeriod":{"type":"weekly","start":"2026-08-14T13:16:00Z","end":"2026-08-21T13:16:00Z"},"productUsage":[{"product":"GrokChat","usagePercent":3},{"product":"GrokBuild"}]}`, 3},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			s := quotaService(t)
			provider := tc.provider
			if provider == "xai-credits" {
				provider = "xai"
			}
			src := &fakeQuotaSource{files: []AuthQuotaFile{{ID: "auth", AuthIndex: "idx", Provider: provider}}, auth: quotaJSON(provider), responses: map[string]string{tc.key: tc.body, "usage/credits": `{"available_count":3}`}}
			s.SetAuthQuotaSource(src)
			items, e := s.AuthQuotaOverview(context.Background(), "callback")
			if e != nil || len(items) != 1 || items[0].Status != "fresh" || len(items[0].Windows) != tc.windows {
				t.Fatalf("items=%#v err=%v", items, e)
			}
			for _, r := range src.requests {
				if r.Header.Get("Authorization") != "Bearer oauth-token" {
					t.Fatalf("missing bearer header: %#v", r.Header)
				}
			}
			if tc.provider == "antigravity" {
				if string(src.requests[0].Body) != `{"project":"project-1"}` {
					t.Fatalf("body=%s", src.requests[0].Body)
				}
			}
			if provider == "xai" && src.requests[0].Header.Get("X-Userid") != "user-1" {
				t.Fatal("missing xAI user id")
			}
			encoded, _ := json.Marshal(items[0])
			if strings.Contains(string(encoded), "oauth-token") {
				t.Fatal("credential leaked in DTO")
			}
		})
	}
}
func TestAuthQuotaOAuthFailureIsThrottled(t *testing.T) {
	s := quotaService(t)
	src := &fakeQuotaSource{files: []AuthQuotaFile{{ID: "auth", AuthIndex: "idx", Provider: "codex"}}, auth: `{"type":"oauth","refresh_token":"refresh-only"}`}
	s.SetAuthQuotaSource(src)
	first, err := s.AuthQuotaOverview(context.Background(), "")
	if err != nil || len(first) != 1 || first[0].Status != "unavailable" || first[0].Error == "" {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	if src.gets != 1 {
		t.Fatalf("gets=%d, want 1", src.gets)
	}
	second, err := s.AuthQuotaOverview(context.Background(), "")
	if err != nil || len(second) != 1 || second[0].Status != "unavailable" || src.gets != 1 {
		t.Fatalf("second=%#v gets=%d err=%v", second, src.gets, err)
	}
}

func TestAuthQuotaForecastMatchesHostProviderAliases(t *testing.T) {
	s := quotaService(t)
	reset := time.Now().UTC().Add(time.Hour)
	duration := int64(7200)
	key := quotaTestKey(t, s, "alias")
	reservation, err := s.Store().Reserve(context.Background(), store.ReserveRequest{CallerID: "default", PluginKeyID: key.ID, Model: "gpt-test", RequestTokenEstimate: 10, AmountMicroUSD: 1, IdempotencyKey: "quota-forecast-alias"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Store().Settle(context.Background(), store.Settlement{ReservationID: reservation.ID, Model: "gpt-test", Usage: money.TokenUsage{Input: 11, Output: 12}, CostMicroUSD: 17, Auth: store.AuthIdentity{AuthID: "auth", AuthIndex: "idx", Provider: "openai"}})
	if err != nil {
		t.Fatal(err)
	}
	used, remaining := 25.0, 75.0
	window := s.forecast(context.Background(), AuthQuotaOverviewItem{AuthID: "auth", AuthIndex: "idx", Provider: "codex", Windows: []AuthQuotaWindow{{Scope: "account", Used: &used, Remaining: &remaining, ResetsAt: &reset, DurationSeconds: &duration}}}).Windows[0]
	if window.LocalUsage == nil || window.LocalUsage.RequestCount != 1 || window.LocalUsage.TotalTokens != 23 || window.LocalUsage.EstimatedCostMicroUSD != 17 || !window.PredictionAvailable {
		t.Fatalf("window=%#v", window)
	}
}

func TestAuthQuotaRequestHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	src := &fakeQuotaSource{responses: map[string]string{"chatgpt.com": `{"ok":true}`}}
	if _, err := request(ctx, src, "", "GET", "https://chatgpt.com/backend-api/wham/usage", nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v, want context.Canceled", err)
	}
	if len(src.requests) != 0 {
		t.Fatalf("requests=%#v, want none after cancel", src.requests)
	}
}

func TestAuthQuotaExpiredWindowsBypassAttemptThrottle(t *testing.T) {
	s := quotaService(t)
	expired := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	src := &fakeQuotaSource{
		files: []AuthQuotaFile{{ID: "auth", AuthIndex: "idx", Provider: "codex"}},
		auth:  quotaJSON("codex"),
		responses: map[string]string{
			"chatgpt.com": `{"rate_limit":{"primary_window":{"used_percent":10,"reset_at":"` + expired + `"}}}`,
		},
	}
	s.SetAuthQuotaSource(src)
	first, err := s.AuthQuotaOverview(context.Background(), "")
	if err != nil || len(first) != 1 {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	attempts := len(src.requests)
	if attempts == 0 {
		t.Fatal("expected an initial quota request")
	}
	src.responses["chatgpt.com"] = `{"rate_limit":{"primary_window":{"used_percent":12,"reset_at":4102444800}}}`
	second, err := s.AuthQuotaOverview(context.Background(), "")
	if err != nil || len(second) != 1 || second[0].Status != "fresh" || len(src.requests) == attempts {
		t.Fatalf("second=%#v attempts=%d->%d err=%v", second, attempts, len(src.requests), err)
	}
}

func TestAuthQuotaAPIKeyOnlyIsHidden(t *testing.T) {
	s := quotaService(t)
	src := &fakeQuotaSource{files: []AuthQuotaFile{{ID: "auth", AuthIndex: "idx", Provider: "codex"}}, auth: `{"api_key":"secret"}`}
	s.SetAuthQuotaSource(src)
	items, e := s.AuthQuotaOverview(context.Background(), "")
	if e != nil || len(items) != 0 {
		t.Fatalf("items=%#v err=%v", items, e)
	}
}
func TestAuthQuotaCacheFailureRetainsSnapshot(t *testing.T) {
	s := quotaService(t)
	src := &fakeQuotaSource{files: []AuthQuotaFile{{ID: "auth", AuthIndex: "idx", Provider: "codex"}}, auth: quotaJSON("codex"), responses: map[string]string{"chatgpt.com": `{"rate_limit":{"primary_window":{"used_percent":10}}}`}}
	s.SetAuthQuotaSource(src)
	if _, e := s.AuthQuotaOverview(context.Background(), ""); e != nil {
		t.Fatal(e)
	}
	src.failHTTP = true
	src.files[0].ModTime = time.Now().UTC()
	items, e := s.AuthQuotaOverview(context.Background(), "")
	if e != nil || len(items) != 1 || items[0].Status != "stale" || len(items[0].Windows) == 0 || items[0].Error == "" {
		t.Fatalf("items=%#v err=%v", items, e)
	}
}

func TestAuthQuotaFailureIsThrottledAndRemainsStale(t *testing.T) {
	s := quotaService(t)
	src := &fakeQuotaSource{files: []AuthQuotaFile{{ID: "auth", AuthIndex: "idx", Provider: "codex"}}, auth: quotaJSON("codex"), responses: map[string]string{"chatgpt.com": `{"rate_limit":{"primary_window":{"used_percent":10,"reset_at":4102444800}}}`}}
	s.SetAuthQuotaSource(src)
	if _, err := s.AuthQuotaOverview(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	src.failHTTP = true
	src.files[0].ModTime = time.Now().UTC()
	first, err := s.AuthQuotaOverview(context.Background(), "")
	if err != nil || len(first) != 1 || first[0].Status != "stale" {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	attempts := len(src.requests)
	second, err := s.AuthQuotaOverview(context.Background(), "")
	if err != nil || len(second) != 1 || second[0].Status != "stale" || len(src.requests) != attempts {
		t.Fatalf("second=%#v attempts=%d want=%d err=%v", second, len(src.requests), attempts, err)
	}
}

func TestXAIWindowsUseUSDAndInferredMonthlyCycle(t *testing.T) {
	windows := xaiWindows(map[string]any{"monthlyLimit": 15000, "used": 4200, "onDemandCap": 5000, "billingPeriodEnd": "2030-02-01T00:00:00Z"})
	if len(windows) != 2 {
		t.Fatalf("windows=%#v", windows)
	}
	monthly := windows[0]
	if monthly.Unit != "currency" || monthly.Currency != "USD" || monthly.Mode != "fixed" || monthly.Limit == nil || *monthly.Limit != 150 || monthly.Used == nil || *monthly.Used != 42 || monthly.CycleStartAt == nil || monthly.CycleStartSource != "inferred_month_start" || !monthly.CycleStartAt.Equal(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("monthly=%#v", monthly)
	}
	onDemand := windows[1]
	if onDemand.Unit != "currency" || onDemand.Currency != "USD" || onDemand.Used == nil || *onDemand.Used != 0 || onDemand.Remaining == nil || *onDemand.Remaining != 50 {
		t.Fatalf("on-demand=%#v", onDemand)
	}
}

func TestXAICreditWindowsUseCurrentWeeklyPeriod(t *testing.T) {
	windows := xaiCreditWindows(map[string]any{
		"creditUsagePercent": 3,
		"currentPeriod":      map[string]any{"type": "weekly", "start": "2026-08-14T13:16:00Z", "end": "2026-08-21T13:16:00Z"},
		"productUsage":       []any{map[string]any{"product": "GrokChat", "usagePercent": 3}, map[string]any{"product": "GrokBuild", "usagePercent": 1}},
	})
	if len(windows) != 3 {
		t.Fatalf("windows=%#v", windows)
	}
	weekly := windows[0]
	if weekly.ID != "weekly" || weekly.Label != "周限额" || weekly.Used == nil || *weekly.Used != 3 || weekly.Remaining == nil || *weekly.Remaining != 97 || weekly.CycleStartAt == nil || !weekly.CycleStartAt.Equal(time.Date(2026, 8, 14, 13, 16, 0, 0, time.UTC)) || weekly.ResetsAt == nil || !weekly.ResetsAt.Equal(time.Date(2026, 8, 21, 13, 16, 0, 0, time.UTC)) {
		t.Fatalf("weekly=%#v", weekly)
	}
	if windows[1].Label != "GrokChat" || windows[1].Used == nil || *windows[1].Used != 3 || windows[2].Label != "GrokBuild" || windows[2].Used == nil || *windows[2].Used != 1 || windows[2].Remaining == nil || *windows[2].Remaining != 99 {
		t.Fatalf("products=%#v", windows[1:])
	}
}
func TestAuthQuotaForecastUsesUpstreamUsedRatio(t *testing.T) {
	s := quotaService(t)
	used, remaining := 25.0, 75.0
	reset := time.Now().UTC().Add(time.Hour)
	duration := int64(7200)
	item := AuthQuotaOverviewItem{AuthID: "auth", AuthIndex: "idx", Provider: "codex", Windows: []AuthQuotaWindow{{ID: "primary", Scope: "account", Used: &used, Remaining: &remaining, ResetsAt: &reset, DurationSeconds: &duration}}}
	from := reset.Add(-time.Duration(duration) * time.Second)
	if _, e := s.store.GetAuthQuotaUsage(context.Background(), store.AuthQuotaUsageFilter{Provider: "codex", AuthID: "auth", From: from, To: time.Now().UTC()}); e != nil {
		t.Fatal(e)
	}
	got := s.forecast(context.Background(), item)
	if got.Windows[0].PredictionAvailable {
		t.Fatal("no records must not produce forecast")
	}
}

func TestAuthQuotaForecastIncludesFullLocalUsage(t *testing.T) {
	s := quotaService(t)
	reset := time.Now().UTC().Add(time.Hour)
	duration := int64(7200)
	key := quotaTestKey(t, s, "full")
	reservation, err := s.Store().Reserve(context.Background(), store.ReserveRequest{CallerID: "default", PluginKeyID: key.ID, Model: "gpt-test", RequestTokenEstimate: 10, AmountMicroUSD: 1, IdempotencyKey: "quota-forecast-full"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Store().Settle(context.Background(), store.Settlement{ReservationID: reservation.ID, Model: "gpt-test", Usage: money.TokenUsage{Input: 11, Output: 12, Reasoning: 13, Cached: 14, CacheRead: 15, CacheCreation: 16}, CostMicroUSD: 17, Auth: store.AuthIdentity{AuthID: "auth", AuthIndex: "idx", Provider: "codex"}})
	if err != nil {
		t.Fatal(err)
	}
	used, remaining := 25.0, 75.0
	window := s.forecast(context.Background(), AuthQuotaOverviewItem{AuthID: "auth", AuthIndex: "idx", Provider: "codex", Windows: []AuthQuotaWindow{{Scope: "account", Used: &used, Remaining: &remaining, ResetsAt: &reset, DurationSeconds: &duration}}}).Windows[0]
	if window.LocalUsage == nil || window.LocalUsage.RequestCount != 1 || window.LocalUsage.TotalTokens != 36 || window.LocalUsage.EstimatedCostMicroUSD != 17 || !window.PredictionAvailable || window.EstimatedRemainingRequests == nil || *window.EstimatedRemainingRequests != 3 || window.ObservedUsed == nil || *window.ObservedUsed != 25 {
		t.Fatalf("window=%#v", window)
	}
}

func TestAuthQuotaForecastIgnoresPrePluginUsage(t *testing.T) {
	s := quotaService(t)
	reset := time.Now().UTC().Add(time.Hour)
	duration := int64(7200)
	used, remaining := 70.0, 30.0
	first := s.forecast(context.Background(), AuthQuotaOverviewItem{AuthID: "auth", AuthIndex: "idx", Provider: "codex", Windows: []AuthQuotaWindow{{ID: "primary", Scope: "account", Used: &used, Remaining: &remaining, ResetsAt: &reset, DurationSeconds: &duration}}}).Windows[0]
	if first.PredictionAvailable || first.ObservedUsed != nil {
		t.Fatalf("baseline capture should not forecast: %#v", first)
	}
	key := quotaTestKey(t, s, "preexisting")
	reservation, err := s.Store().Reserve(context.Background(), store.ReserveRequest{CallerID: "default", PluginKeyID: key.ID, Model: "gpt-test", RequestTokenEstimate: 10, AmountMicroUSD: 1, IdempotencyKey: "quota-forecast-preexisting"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Store().Settle(context.Background(), store.Settlement{ReservationID: reservation.ID, Model: "gpt-test", Usage: money.TokenUsage{Input: 11, Output: 12}, CostMicroUSD: 17, Auth: store.AuthIdentity{AuthID: "auth", AuthIndex: "idx", Provider: "codex"}})
	if err != nil {
		t.Fatal(err)
	}
	used, remaining = 80.0, 20.0
	window := s.forecast(context.Background(), AuthQuotaOverviewItem{AuthID: "auth", AuthIndex: "idx", Provider: "codex", Windows: []AuthQuotaWindow{{ID: "primary", Scope: "account", Used: &used, Remaining: &remaining, ResetsAt: &reset, DurationSeconds: &duration}}}).Windows[0]
	if window.ObservedUsed == nil || *window.ObservedUsed != 10 || window.BaselineUsed == nil || *window.BaselineUsed != 70 || window.EstimatedRemainingRequests == nil || *window.EstimatedRemainingRequests != 2 {
		t.Fatalf("window=%#v", window)
	}
}

func TestEstimateRemainingRequestsUsesRatiosAndGuardsTinyObserved(t *testing.T) {
	limit := 100.0
	remainingRatio := 0.73456
	window := &AuthQuotaWindow{Unit: "percentage", Limit: &limit, RemainingRatio: &remainingRatio}
	got, ok := estimateRemainingRequests(10, window, 12.345)
	if !ok || got != 59 {
		t.Fatalf("ratio estimate = %d ok=%t, want 59 true", got, ok)
	}
	if _, ok := estimateRemainingRequests(10, window, 0.4); ok {
		t.Fatal("observed ratio below 0.5% must not estimate")
	}
	remaining := 20.0
	absolute := &AuthQuotaWindow{Remaining: &remaining}
	got, ok = estimateRemainingRequests(1, absolute, 10)
	if !ok || got != 2 {
		t.Fatalf("absolute fallback = %d ok=%t, want 2 true", got, ok)
	}
}

func TestAuthQuotaAntigravityUnmatchedModelDisablesPrediction(t *testing.T) {
	s := quotaService(t)
	reset := time.Now().UTC().Add(time.Hour)
	duration := int64(7200)
	key := quotaTestKey(t, s, "pool")
	reservation, err := s.Store().Reserve(context.Background(), store.ReserveRequest{CallerID: "default", PluginKeyID: key.ID, Model: "unknown-model", RequestTokenEstimate: 10, AmountMicroUSD: 1, IdempotencyKey: "quota-pool-unmatched"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Store().Settle(context.Background(), store.Settlement{ReservationID: reservation.ID, Model: "unknown-model", Usage: money.TokenUsage{Input: 1}, CostMicroUSD: 1, Auth: store.AuthIdentity{AuthID: "auth", AuthIndex: "idx", Provider: "antigravity"}})
	if err != nil {
		t.Fatal(err)
	}
	used, remaining := 25.0, 75.0
	window := s.forecast(context.Background(), AuthQuotaOverviewItem{AuthID: "auth", AuthIndex: "idx", Provider: "antigravity", Windows: []AuthQuotaWindow{{Scope: "model_pool", ScopeID: "gemini", Used: &used, Remaining: &remaining, ResetsAt: &reset, DurationSeconds: &duration}}}).Windows[0]
	if window.LocalAttributionStatus != "unmatched_local_usage" || window.PredictionAvailable {
		t.Fatalf("window=%#v", window)
	}
}

func TestAuthQuotaContracts(t *testing.T) {
	t.Run("codex weekly classification", func(t *testing.T) {
		plus := codexWindows(map[string]any{"rate_limit": map[string]any{
			"primary_window":   map[string]any{"used_percent": 1, "limit_window_seconds": 18000, "reset_after_seconds": 1000, "reset_at": 4102444800},
			"secondary_window": map[string]any{"used_percent": 27, "reset_after_seconds": 245230, "reset_at": 4102444800},
		}})
		if len(plus) != 2 || plus[0].DurationSeconds == nil || *plus[0].DurationSeconds != 18000 || plus[1].ID != "rate-limit-secondary" || plus[1].DurationSeconds == nil || *plus[1].DurationSeconds != 604800 {
			t.Fatalf("plus=%#v", plus)
		}
		team := codexWindows(map[string]any{"usage": map[string]any{"rate_limit": map[string]any{
			"primary_window":   map[string]any{"used_percent": 9, "limit_window_seconds": 604800, "reset_at": 4102444800},
			"secondary_window": nil,
		}}})
		if len(team) != 1 || team[0].ID != "rate-limit-primary" || team[0].DurationSeconds == nil || *team[0].DurationSeconds != 604800 {
			t.Fatalf("team=%#v", team)
		}
		clamped := codexWindows(map[string]any{"rate_limit": map[string]any{"primary_window": map[string]any{"used_percent": 150, "limit_window_seconds": 604800}}})
		if len(clamped) != 1 || clamped[0].Used == nil || *clamped[0].Used != 100 {
			t.Fatalf("clamped=%#v", clamped)
		}
		extra := codexWindows(map[string]any{
			"rate_limit": map[string]any{"primary_window": map[string]any{"used_percent": 1, "limit_window_seconds": 604800}},
			"additional_rate_limits": []any{map[string]any{
				"id": "codex-spark", "title": "Codex Spark",
				"primary_window":   map[string]any{"used_percent": 10, "limit_window_seconds": 18000},
				"secondary_window": map[string]any{"used_percent": 20, "limit_window_seconds": 604800},
			}},
		})
		got := map[string]AuthQuotaWindow{}
		for _, window := range extra {
			got[window.ID] = window
		}
		if len(extra) != 3 || got["additional-codex_spark-secondary"].DurationSeconds == nil || *got["additional-codex_spark-secondary"].DurationSeconds != 604800 {
			t.Fatalf("extra=%#v", extra)
		}
	})
	t.Run("codex nested additional limits and scopes", func(t *testing.T) {
		windows := codexWindows(map[string]any{
			"rate_limit":             map[string]any{"primary_window": map[string]any{"used_percent": 25}},
			"code_review_rate_limit": map[string]any{"secondary_window": map[string]any{"used_percent": 5}},
			"additional_rate_limits": []any{map[string]any{
				"limit_name":      "Vision requests",
				"metered_feature": "vision-requests",
				"rate_limit":      map[string]any{"primary_window": map[string]any{"used_percent": 15}, "secondary_window": map[string]any{"used_percent": 10}},
			}},
		})
		if len(windows) != 4 {
			t.Fatalf("windows=%#v", windows)
		}
		got := map[string]AuthQuotaWindow{}
		for _, window := range windows {
			got[window.ID] = window
		}
		if window := got["rate-limit-primary"]; window.Scope != "account" || window.ScopeID != "" {
			t.Fatalf("rate limit window=%#v", window)
		}
		if window := got["code-review-secondary"]; window.Scope != "model" || window.ScopeID != "codex_code_review" {
			t.Fatalf("code review window=%#v", window)
		}
		for _, id := range []string{"additional-vision_requests-primary", "additional-vision_requests-secondary"} {
			window := got[id]
			if window.Scope != "model" || window.ScopeID != "codex_vision_requests" || window.Label != "Vision requests "+strings.TrimPrefix(id, "additional-vision_requests-") {
				t.Fatalf("additional window=%#v", window)
			}
		}
	})
	t.Run("codex reset endpoint", func(t *testing.T) {
		s := quotaService(t)
		src := &fakeQuotaSource{files: []AuthQuotaFile{{ID: "auth", AuthIndex: "idx", Provider: "codex"}}, auth: quotaJSON("codex"), responses: map[string]string{"/wham/usage": `{"rate_limit":{"primary_window":{"used_percent":25}}}`, "/rate-limit-reset-credits": `{"available_count":3}`}}
		s.SetAuthQuotaSource(src)
		items, err := s.AuthQuotaOverview(context.Background(), "")
		if err != nil || len(items) != 1 || items[0].ResetCredits == nil || *items[0].ResetCredits != 3 {
			t.Fatalf("items=%#v err=%v", items, err)
		}
		if len(src.requests) != 2 || src.requests[1].URL != "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits" {
			t.Fatalf("requests=%#v", src.requests)
		}
	})
	t.Run("claude scopes and durations", func(t *testing.T) {
		data := map[string]any{}
		for _, id := range []string{"five_hour", "seven_day", "seven_day_oauth_apps", "seven_day_opus", "seven_day_sonnet", "seven_day_cowork", "iguana_necktie"} {
			data[id] = map[string]any{"utilization": 0.5, "resets_at": "2030-01-01T00:00:00Z"}
		}
		windows, err := claude(context.Background(), &fakeQuotaSource{responses: map[string]string{"anthropic.com": mustJSON(t, data)}}, "", quotaCredentials{token: "token"})
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range windows.Windows {
			if w.ID == "iguana_necktie" {
				if w.DurationSeconds != nil || w.CycleStartAt != nil {
					t.Fatalf("%s must not have a known duration: %#v", w.ID, w)
				}
				continue
			}
			want := int64(604800)
			if w.ID == "five_hour" {
				want = 18000
			}
			if w.DurationSeconds == nil || *w.DurationSeconds != want {
				t.Fatalf("%s duration=%v", w.ID, w.DurationSeconds)
			}
			if (w.ID == "seven_day_opus" && (w.Scope != "model" || w.ScopeID != "opus")) || (w.ID == "seven_day_sonnet" && (w.Scope != "model" || w.ScopeID != "sonnet")) || (w.ID != "seven_day_opus" && w.ID != "seven_day_sonnet" && w.Scope != "account") {
				t.Fatalf("window=%#v", w)
			}
		}
	})
	t.Run("claude omelette weekly alias", func(t *testing.T) {
		windows, err := claude(context.Background(), &fakeQuotaSource{responses: map[string]string{"anthropic.com": `{"seven_day_omelette":{"utilization":0.4,"resets_at":"2030-01-01T00:00:00Z"}}`}}, "", quotaCredentials{token: "token"})
		if err != nil || len(windows.Windows) != 1 || windows.Windows[0].ID != "seven_day_omelette" || windows.Windows[0].DurationSeconds == nil || *windows.Windows[0].DurationSeconds != 604800 {
			t.Fatalf("windows=%#v err=%v", windows, err)
		}
	})
	t.Run("kimi official usage shape", func(t *testing.T) {
		windows := kimiWindows(map[string]any{"used": "20", "limit": "100", "remaining": "80", "resetAt": "2030-01-01T00:00:00Z", "limits": []any{map[string]any{"name": "daily", "window": map[string]any{"duration": "1", "timeUnit": "TIME_UNIT_DAY"}, "detail": map[string]any{"used": 10, "limit": 20, "remaining": 10, "resetIn": "3600"}}}})
		if len(windows) != 2 || windows[0].ID != "weekly" || windows[0].Label != "周限额" || windows[0].DurationSeconds == nil || *windows[0].DurationSeconds != 604800 || windows[1].DurationSeconds == nil || *windows[1].DurationSeconds != 86400 || windows[1].ResetsAt == nil {
			t.Fatalf("windows=%#v", windows)
		}
	})
	t.Run("antigravity model pool scopes", func(t *testing.T) {
		windows := antiWindows(map[string]any{"groups": []any{map[string]any{"displayName": "Gemini", "buckets": []any{map[string]any{"bucketId": "gemini-weekly", "remainingFraction": 0.5, "resetTime": "2030-01-01T00:00:00Z"}, map[string]any{"bucketId": "3p-weekly", "remainingFraction": 0.25}, map[string]any{"bucketId": "ignored", "remainingFraction": 0.5}, map[string]any{"bucketId": "gemini-5h", "remainingFraction": 1, "enabled": false}}}, map[string]any{"buckets": []any{map[string]any{"bucketId": "gemini-5h", "remainingFraction": 0.8}, map[string]any{"bucketId": "3p-5h", "remainingFraction": 0.2}, map[string]any{"bucketId": "3p-weekly", "remainingFraction": 2}}}}})
		if len(windows) != 4 {
			t.Fatalf("windows=%#v", windows)
		}
		for i, want := range []struct {
			id, label, scopeID string
			duration           int64
			remaining          float64
		}{{"gemini-5h", "gemini-5h", "gemini", 18000, 80}, {"3p-5h", "3p-5h", "third-party", 18000, 20}, {"gemini-weekly", "Gemini", "gemini", 604800, 50}, {"3p-weekly", "Gemini", "third-party", 604800, 25}} {
			window := windows[i]
			if window.ID != want.id || window.Label != want.label || window.Scope != "model_pool" || window.ScopeID != want.scopeID || window.Mode != "rolling" || window.Unit != "percentage" || window.Limit == nil || *window.Limit != 100 || window.Used == nil || *window.Used != 100-want.remaining || window.Remaining == nil || *window.Remaining != want.remaining || window.UsedRatio == nil || *window.UsedRatio != (100-want.remaining)/100 || window.RemainingRatio == nil || *window.RemainingRatio != want.remaining/100 || window.DurationSeconds == nil || *window.DurationSeconds != want.duration {
				t.Fatalf("window=%#v", window)
			}
		}
		if windows[2].ResetsAt == nil || windows[2].CycleStartAt == nil {
			t.Fatalf("windows=%#v", windows)
		}
	})
	t.Run("antigravity accepts percent remaining and partial weekly pools", func(t *testing.T) {
		windows := antiWindows(map[string]any{"groups": []any{map[string]any{"buckets": []any{
			map[string]any{"bucketId": "gemini_weekly", "remaining_fraction": 80},
		}}}})
		if len(windows) != 1 || windows[0].ID != "gemini-weekly" || windows[0].Remaining == nil || *windows[0].Remaining != 80 {
			t.Fatalf("windows=%#v", windows)
		}
	})
	t.Run("antigravity single weekly pool is enough", func(t *testing.T) {
		s := quotaService(t)
		src := &fakeQuotaSource{files: []AuthQuotaFile{{ID: "auth", AuthIndex: "idx", Provider: "antigravity"}}, auth: quotaJSON("antigravity"), responses: map[string]string{"cloudcode": `{"groups":[{"buckets":[{"bucketId":"gemini-weekly","remainingFraction":0.4,"resetTime":"2030-01-01T00:00:00Z"}]}]}`}}
		s.SetAuthQuotaSource(src)
		items, err := s.AuthQuotaOverview(context.Background(), "")
		if err != nil || len(items) != 1 || items[0].Status != "fresh" || len(items[0].Windows) != 1 || items[0].Windows[0].ID != "gemini-weekly" {
			t.Fatalf("items=%#v err=%v", items, err)
		}
	})
	t.Run("antigravity model pools require attributable local usage", func(t *testing.T) {
		used, remaining := 0.5, 0.5
		reset := time.Now().UTC().Add(time.Hour)
		duration := int64(3600)
		item := AuthQuotaOverviewItem{AuthID: "auth", AuthIndex: "idx", Provider: "antigravity", Windows: []AuthQuotaWindow{{Scope: "model_pool", ScopeID: "gemini", Used: &used, Remaining: &remaining, ResetsAt: &reset, DurationSeconds: &duration}}}
		got := quotaService(t).forecast(context.Background(), item).Windows[0]
		if got.PredictionAvailable || got.LocalAttributionStatus != "complete" || got.LocalUsage == nil {
			t.Fatalf("window=%#v", got)
		}
	})
	t.Run("forecast expired reset is unavailable", func(t *testing.T) {
		used, remaining := 25.0, 75.0
		reset := time.Now().UTC().Add(-time.Hour)
		got := quotaService(t).forecast(context.Background(), AuthQuotaOverviewItem{Windows: []AuthQuotaWindow{{Scope: "account", Used: &used, Remaining: &remaining, ResetsAt: &reset}}}).Windows[0]
		if got.LocalAttributionStatus != "unavailable" || got.PredictionAvailable {
			t.Fatalf("window=%#v", got)
		}
	})
	t.Run("forecast completes without records", func(t *testing.T) {
		used, remaining := 25.0, 75.0
		reset := time.Now().UTC().Add(time.Hour)
		duration := int64(7200)
		got := quotaService(t).forecast(context.Background(), AuthQuotaOverviewItem{AuthID: "auth", AuthIndex: "idx", Provider: "codex", Windows: []AuthQuotaWindow{{Scope: "account", Used: &used, Remaining: &remaining, ResetsAt: &reset, DurationSeconds: &duration}}}).Windows[0]
		if got.LocalAttributionStatus != "complete" || got.PredictionAvailable || got.LocalUsage == nil {
			t.Fatalf("window=%#v", got)
		}
	})
}
func mustJSON(t *testing.T, value any) string {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
