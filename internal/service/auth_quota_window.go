package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
)

func cent(v *float64) *float64 {
	if v == nil {
		return nil
	}
	x := *v / 100
	return &x
}
func percentWindow(id, label, scope, scopeID string, m map[string]any) (AuthQuotaWindow, bool) {
	p := number(m, "used_percent", "usedPercent")
	if p != nil && *p > 100 {
		v := 100.0
		p = &v
	}
	if p != nil && *p < 0 {
		return AuthQuotaWindow{}, false
	}
	l := 100.0
	w := AuthQuotaWindow{ID: id, Label: label, Scope: scope, ScopeID: scopeID, Mode: "rolling", Unit: "percentage", Limit: &l, ResetsAt: timeField(m, "reset_at", "resetAt"), DurationSeconds: duration(m)}
	if p != nil {
		u := *p
		r := math.Max(0, l-u)
		ur := u / l
		rr := r / l
		w.Used, w.Remaining, w.UsedRatio, w.RemainingRatio = &u, &r, &ur, &rr
	}
	if w.ResetsAt != nil && w.DurationSeconds != nil {
		v := w.ResetsAt.Add(-time.Duration(*w.DurationSeconds) * time.Second)
		w.CycleStartAt = &v
	}
	return w, true
}
func utilWindow(id, label, scope, scopeID string, seconds *int64, m map[string]any) (AuthQuotaWindow, bool) {
	w, ok := valueWindow(id, label, "requests", ptr(1), nil, number(m, "utilization"), timeField(m, "resets_at", "resetsAt"), seconds)
	if ok {
		w.Scope, w.ScopeID = scope, scopeID
		if w.ResetsAt != nil && seconds != nil {
			start := w.ResetsAt.Add(-time.Duration(*seconds) * time.Second)
			w.CycleStartAt = &start
		}
	}
	return w, ok
}
func ptr(v float64) *float64  { return &v }
func int64ptr(v int64) *int64 { return &v }
func valueWindow(id, label, unit string, l, u, util *float64, reset *time.Time, dur *int64) (AuthQuotaWindow, bool) {
	if l != nil && (!finiteNonNegative(*l)) {
		l = nil
	}
	if u != nil && (!finiteNonNegative(*u)) {
		u = nil
	}
	if util != nil && (!finiteNonNegative(*util)) {
		util = nil
	}
	if l == nil && u == nil && util == nil {
		return AuthQuotaWindow{}, false
	}
	if util != nil && u == nil {
		v := *util
		if v > 1 {
			v /= 100
		}
		if l != nil {
			x := *l * v
			u = &x
		}
	}
	var r, ur, rr *float64
	if l != nil && u != nil {
		x := math.Max(0, *l-*u)
		r = &x
		if *l > 0 {
			y := *u / *l
			ur = &y
			z := math.Max(0, 1-y)
			rr = &z
		}
	}
	return AuthQuotaWindow{ID: id, Label: label, Scope: "account", Mode: "rolling", Unit: unit, Limit: l, Used: u, Remaining: r, UsedRatio: ur, RemainingRatio: rr, ResetsAt: reset, DurationSeconds: dur}, true
}
func object(m map[string]any, key string) (map[string]any, bool) {
	v, ok := m[key]
	x, ok2 := v.(map[string]any)
	return x, ok && ok2
}
func objectAny(m map[string]any, keys ...string) (map[string]any, bool) {
	for _, key := range keys {
		if x, ok := object(m, key); ok {
			return x, true
		}
	}
	return nil, false
}
func anySlice(m map[string]any, keys ...string) []any {
	if m == nil {
		return nil
	}
	for _, key := range keys {
		if xs, ok := m[key].([]any); ok {
			return xs
		}
	}
	return nil
}
func number(m map[string]any, keys ...string) *float64 {
	for _, k := range keys {
		if x, ok := m[k]; ok {
			if n, ok := numeric(x); ok && finiteNonNegative(n) {
				return &n
			}
		}
	}
	return nil
}
func firstNumber(m map[string]any, keys ...string) *float64 { return number(m, keys...) }
func finiteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}
func numeric(x any) (float64, bool) {
	if m, ok := x.(map[string]any); ok {
		return numeric(m["val"])
	}
	switch v := x.(type) {
	case float64:
		return v, !math.IsNaN(v) && !math.IsInf(v, 0)
	case string:
		n, e := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return n, e == nil && !math.IsNaN(n) && !math.IsInf(n, 0)
	case json.Number:
		n, e := v.Float64()
		return n, e == nil && !math.IsNaN(n) && !math.IsInf(n, 0)
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}
func timeField(m map[string]any, keys ...string) *time.Time {
	for _, k := range keys {
		if x, ok := m[k]; ok {
			if n, ok := numeric(x); ok {
				var t time.Time
				if n > 1e12 {
					t = time.UnixMilli(int64(n))
				} else {
					t = time.Unix(int64(n), 0)
				}
				t = t.UTC()
				return &t
			}
			if s, ok := x.(string); ok {
				for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05", time.DateOnly} {
					if t, e := time.Parse(layout, s); e == nil {
						t = t.UTC()
						return &t
					}
				}
			}
		}
	}
	return nil
}
func duration(m map[string]any) *int64 {
	for _, k := range []string{"limit_window_seconds", "limitWindowSeconds", "duration_seconds", "durationSeconds", "duration"} {
		if n := number(m, k); n != nil {
			v := *n
			if k == "duration" {
				u := strings.ToLower(findText(m, "duration_unit"))
				if strings.HasPrefix(u, "hour") {
					v *= 3600
				} else if strings.HasPrefix(u, "minute") {
					v *= 60
				}
			}
			x := int64(v)
			return &x
		}
	}
	return nil
}

func first(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
func quotaProvider(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	switch {
	case strings.Contains(p, "codex") || strings.Contains(p, "openai"):
		return "codex"
	case strings.Contains(p, "claude") || strings.Contains(p, "anthropic"):
		return "claude"
	case strings.Contains(p, "antigravity") || strings.Contains(p, "google"):
		return "antigravity"
	case strings.Contains(p, "kimi") || strings.Contains(p, "moonshot"):
		return "kimi"
	case p == "xai" || strings.Contains(p, "grok"):
		return "xai"
	}
	return ""
}
func windowBaselineID(window *AuthQuotaWindow) string {
	if window == nil {
		return "window"
	}
	if id := strings.TrimSpace(window.ID); id != "" {
		return id
	}
	return first(strings.TrimSpace(window.Scope+"|"+window.ScopeID), "window")
}

func windowCycleKey(window *AuthQuotaWindow) string {
	if window == nil {
		return ""
	}
	if window.CycleStartAt != nil && window.ResetsAt != nil {
		return fmt.Sprintf("%d:%d", window.CycleStartAt.UTC().UnixMilli(), window.ResetsAt.UTC().UnixMilli())
	}
	if window.ResetsAt != nil {
		return fmt.Sprintf("reset:%d", window.ResetsAt.UTC().UnixMilli())
	}
	return ""
}

const authQuotaMinObservedRatio = 0.005

func estimateRemainingRequests(requestCount int64, window *AuthQuotaWindow, observed float64) (int64, bool) {
	if window == nil || requestCount <= 0 || observed <= 0 || !finiteNonNegative(observed) {
		return 0, false
	}
	remainingRatio := math.NaN()
	observedRatio := math.NaN()
	if window.RemainingRatio != nil && finiteNonNegative(*window.RemainingRatio) {
		remainingRatio = *window.RemainingRatio
	}
	if window.Limit != nil && *window.Limit > 0 {
		observedRatio = observed / *window.Limit
		if math.IsNaN(remainingRatio) && window.Remaining != nil && finiteNonNegative(*window.Remaining) {
			remainingRatio = *window.Remaining / *window.Limit
		}
	} else if window.Unit == "percentage" {
		observedRatio = observed / 100
		if math.IsNaN(remainingRatio) && window.Remaining != nil && finiteNonNegative(*window.Remaining) {
			remainingRatio = *window.Remaining / 100
		}
	} else if window.Used != nil && *window.Used > 0 && window.UsedRatio != nil && *window.UsedRatio > 0 {
		observedRatio = observed * (*window.UsedRatio) / (*window.Used)
		if math.IsNaN(remainingRatio) && window.Remaining != nil && finiteNonNegative(*window.Remaining) {
			remainingRatio = (*window.Remaining) * (*window.UsedRatio) / (*window.Used)
		}
	} else if window.Remaining != nil && finiteNonNegative(*window.Remaining) {
		return int64(math.Floor(float64(requestCount) * (*window.Remaining) / observed)), true
	}
	if math.IsNaN(remainingRatio) || math.IsNaN(observedRatio) || observedRatio < authQuotaMinObservedRatio || remainingRatio < 0 {
		return 0, false
	}
	return int64(math.Floor(float64(requestCount) * remainingRatio / observedRatio)), true
}

func (s *Service) observedWindowUsed(ctx context.Context, item AuthQuotaOverviewItem, window *AuthQuotaWindow, localRequests int64) (observed, baseline float64, ok bool) {
	if window == nil || window.Used == nil || *window.Used < 0 || !finiteNonNegative(*window.Used) {
		return 0, 0, false
	}
	used := *window.Used
	cycleKey := windowCycleKey(window)
	windowID := windowBaselineID(window)
	if strings.TrimSpace(item.AuthID) == "" || cycleKey == "" {
		return used, 0, used > 0
	}
	saved, err := s.store.GetAuthQuotaWindowBaseline(ctx, item.Provider, item.AuthID, windowID, cycleKey)
	if err != nil {
		baseline = 0
		if localRequests == 0 {
			baseline = used
		}
		if upsertErr := s.store.UpsertAuthQuotaWindowBaseline(ctx, item.Provider, item.AuthID, windowID, cycleKey, baseline); upsertErr != nil {
			return used, 0, used > 0
		}
		if localRequests == 0 {
			return 0, baseline, false
		}
		return used, 0, used > 0
	}
	observed = used - saved.BaselineUsed
	if observed <= 0 {
		return 0, saved.BaselineUsed, false
	}
	return observed, saved.BaselineUsed, true
}

func cycleStart(provider string, window *AuthQuotaWindow) time.Time {
	if window.CycleStartAt != nil {
		return window.CycleStartAt.UTC()
	}
	if window.ResetsAt == nil {
		return time.Time{}
	}
	if window.DurationSeconds != nil && *window.DurationSeconds > 0 {
		return window.ResetsAt.Add(-time.Duration(*window.DurationSeconds) * time.Second).UTC()
	}
	if provider == "xai" {
		return window.ResetsAt.AddDate(0, -1, 0).UTC()
	}
	return time.Time{}
}

func (s *Service) windowUsage(ctx context.Context, filter store.AuthQuotaUsageFilter, provider, scopeID string) (store.AuthQuotaUsage, bool, error) {
	if provider != "antigravity" {
		usage, err := s.store.GetAuthQuotaUsage(ctx, filter)
		return usage, err == nil, err
	}
	byModel, err := s.store.GetAuthQuotaUsageByModel(ctx, filter)
	if err != nil {
		return store.AuthQuotaUsage{}, false, err
	}
	var result store.AuthQuotaUsage
	for _, item := range byModel {
		pool := antigravityModelPool(item.Model)
		if pool == "" {
			return store.AuthQuotaUsage{}, false, nil
		}
		if pool == scopeID {
			addAuthQuotaUsage(&result, item.AuthQuotaUsage)
		}
	}
	return result, true, nil
}

func addAuthQuotaUsage(to *store.AuthQuotaUsage, from store.AuthQuotaUsage) {
	to.RequestCount += from.RequestCount
	to.InputTokens += from.InputTokens
	to.OutputTokens += from.OutputTokens
	to.ReasoningTokens += from.ReasoningTokens
	to.CachedTokens += from.CachedTokens
	to.CacheReadTokens += from.CacheReadTokens
	to.CacheCreationTokens += from.CacheCreationTokens
	to.ActualCostMicroUSD += from.ActualCostMicroUSD
}

func totalAuthQuotaTokens(usage store.AuthQuotaUsage) int64 {
	return money.ReportedTotal(money.TokenUsage{
		Input: usage.InputTokens, Output: usage.OutputTokens, Reasoning: usage.ReasoningTokens, Cached: usage.CachedTokens,
	})
}

func authQuotaLocalUsage(usage store.AuthQuotaUsage) *AuthQuotaLocalUsage {
	return &AuthQuotaLocalUsage{
		RequestCount: usage.RequestCount, InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens,
		ReasoningTokens: usage.ReasoningTokens, CachedTokens: usage.CachedTokens, CacheReadTokens: usage.CacheReadTokens,
		CacheCreationTokens: usage.CacheCreationTokens, TotalTokens: totalAuthQuotaTokens(usage),
		EstimatedCostMicroUSD: int64(usage.ActualCostMicroUSD),
	}
}

func timePtr(value time.Time) *time.Time {
	value = value.UTC()
	return &value
}
