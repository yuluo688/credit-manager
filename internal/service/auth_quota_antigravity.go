package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func antigravity(ctx context.Context, s AuthQuotaSource, cb string, c quotaCredentials) (quotaSnapshot, error) {
	if c.projectID == "" {
		return quotaSnapshot{}, fmt.Errorf("OAuth project id is unavailable")
	}
	h := headers(c.token)
	h.Set("Content-Type", "application/json")
	h.Set("User-Agent", "credit-manager")
	body := []byte(`{"project":` + strconv.Quote(c.projectID) + `}`)
	var d map[string]any
	var e error
	for _, url := range []string{"https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary", "https://daily-cloudcode-pa.sandbox.googleapis.com/v1internal:retrieveUserQuotaSummary", "https://cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary"} {
		d, e = request(ctx, s, cb, "POST", url, h, body)
		if e == nil {
			break
		}
	}
	if e != nil {
		return quotaSnapshot{}, e
	}
	w := antiWindows(d)
	if len(w) == 0 {
		return quotaSnapshot{}, fmt.Errorf("antigravity quota response has no recognized windows")
	}
	return quotaSnapshot{Plan: findText(d, "plan", "planName"), Windows: w}, nil
}
func antiWindows(d map[string]any) []AuthQuotaWindow {
	var out []AuthQuotaWindow
	specs := []struct {
		id, scopeID string
		duration    int64
	}{
		{"gemini-5h", "gemini", 18000},
		{"3p-5h", "third-party", 18000},
		{"gemini-weekly", "gemini", 604800},
		{"3p-weekly", "third-party", 604800},
	}
	groups, _ := d["groups"].([]any)
	for _, spec := range specs {
		var window *AuthQuotaWindow
		for _, g := range groups {
			gm, _ := g.(map[string]any)
			bs, _ := gm["buckets"].([]any)
			for _, b := range bs {
				m, _ := b.(map[string]any)
				id := findText(m, "bucketId", "bucket_id")
				if !antiBucketMatch(id, spec.id) || m["enabled"] == false || m["isEnabled"] == false {
					continue
				}
				r := antiRemainingFraction(m)
				if r == nil {
					continue
				}
				limit := 100.0
				remaining := limit * *r
				used := limit - remaining
				usedRatio := used / limit
				duration := spec.duration
				reset := timeField(m, "resetTime", "resetAt", "reset_time", "reset_at")
				var start *time.Time
				if reset != nil {
					value := reset.Add(-time.Duration(duration) * time.Second)
					start = &value
				}
				window = &AuthQuotaWindow{ID: spec.id, Label: first(findText(gm, "displayName", "display_name"), spec.id), Scope: "model_pool", ScopeID: spec.scopeID, Mode: "rolling", Unit: "percentage", Limit: &limit, Used: &used, Remaining: &remaining, UsedRatio: &usedRatio, RemainingRatio: r, ResetsAt: reset, DurationSeconds: &duration, CycleStartAt: start, CycleStartSource: "inferred_window_start"}
				break
			}
			if window != nil {
				break
			}
		}
		if window != nil {
			out = append(out, *window)
		}
	}
	return out
}

func antiBucketMatch(id, spec string) bool {
	normalize := func(value string) string {
		return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", "-"))
	}
	return normalize(id) == normalize(spec)
}

func antiRemainingFraction(m map[string]any) *float64 {
	r := number(m, "remainingFraction", "remaining_fraction", "remainingPercent", "remaining_percent")
	if r == nil {
		return nil
	}
	value := *r
	if value > 1 {
		value /= 100
	}
	if value < 0 || value > 1 {
		return nil
	}
	return &value
}

func antigravityModelPool(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(model, "gemini"):
		return "gemini"
	case strings.Contains(model, "claude"), strings.Contains(model, "gpt"), strings.Contains(model, "openai"):
		return "third-party"
	default:
		return ""
	}
}
