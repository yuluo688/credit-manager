package service

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func codex(ctx context.Context, s AuthQuotaSource, cb string, c quotaCredentials) (quotaSnapshot, error) {
	h := headers(c.token)
	h.Set("User-Agent", "credit-manager")
	if c.accountID != "" {
		h.Set("Chatgpt-Account-Id", c.accountID)
	}
	d, e := request(ctx, s, cb, "GET", "https://chatgpt.com/backend-api/wham/usage", h, nil)
	if e != nil {
		return quotaSnapshot{}, e
	}
	w := codexWindows(d)
	if len(w) == 0 {
		return quotaSnapshot{}, fmt.Errorf("codex quota response has no recognized windows")
	}
	snap := quotaSnapshot{Plan: findText(d, "plan_type", "planType", "plan"), Windows: w}
	if credits, e := request(ctx, s, cb, "GET", "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits", h, nil); e == nil {
		snap.ResetCredits = number(credits, "available_count", "availableCount")
	}
	return snap, nil
}
func codexPayload(d map[string]any) map[string]any {
	if d == nil {
		return nil
	}
	if _, ok := objectAny(d, "rate_limit", "rateLimit"); ok {
		return d
	}
	if usage, ok := objectAny(d, "usage"); ok {
		return usage
	}
	return d
}

func codexWindows(d map[string]any) []AuthQuotaWindow {
	d = codexPayload(d)
	var out []AuthQuotaWindow
	add := func(family, label, scope, scopeID string, v any) {
		m, ok := v.(map[string]any)
		if !ok {
			return
		}
		primary, hasPrimary := objectAny(m, "primary_window", "primaryWindow")
		secondary, hasSecondary := objectAny(m, "secondary_window", "secondaryWindow")
		if hasPrimary {
			if w, ok := percentWindow(family+"-primary", label+" primary", scope, scopeID, primary); ok {
				applyCodexWindowMeta(&w, "primary", primary, hasSecondary)
				out = append(out, w)
			}
		}
		if hasSecondary {
			if w, ok := percentWindow(family+"-secondary", label+" secondary", scope, scopeID, secondary); ok {
				applyCodexWindowMeta(&w, "secondary", secondary, hasSecondary)
				out = append(out, w)
			}
		}
	}
	if m, ok := objectAny(d, "rate_limit", "rateLimit"); ok {
		add("rate-limit", "Rate limit", "account", "", m)
	}
	if m, ok := objectAny(d, "code_review_rate_limit", "codeReviewRateLimit"); ok {
		add("code-review", "Code review", "model", "codex_code_review", m)
	}
	for i, x := range anySlice(d, "additional_rate_limits", "additionalRateLimits") {
		m, _ := x.(map[string]any)
		feature := first(findText(m, "metered_feature", "meteredFeature", "id"), fmt.Sprintf("additional-%d", i+1))
		name := first(findText(m, "limit_name", "limitName", "title"), feature)
		if limit, ok := objectAny(m, "rate_limit", "rateLimit"); ok {
			add("additional-"+codexSlug(feature), name, "model", "codex_"+codexSlug(feature), limit)
			continue
		}
		add("additional-"+codexSlug(feature), name, "model", "codex_"+codexSlug(feature), m)
	}
	return out
}

func applyCodexWindowMeta(w *AuthQuotaWindow, slot string, raw map[string]any, hasSecondary bool) {
	if d := codexWindowSeconds(slot, raw, hasSecondary); d != nil {
		w.DurationSeconds = d
	}
	if w.ResetsAt != nil && w.DurationSeconds != nil {
		start := w.ResetsAt.Add(-time.Duration(*w.DurationSeconds) * time.Second)
		w.CycleStartAt = &start
		w.CycleStartSource = "inferred_window_start"
	}
}

func codexWindowSeconds(slot string, m map[string]any, hasSecondary bool) *int64 {
	if n := number(m, "limit_window_seconds", "limitWindowSeconds", "duration_seconds", "durationSeconds"); n != nil && *n > 0 {
		v := int64(*n)
		return &v
	}
	if slot == "secondary" || (slot == "primary" && !hasSecondary) {
		return int64ptr(604800)
	}
	if slot == "primary" {
		return int64ptr(18000)
	}
	return nil
}
func codexSlug(value string) string {
	var b strings.Builder
	underscore := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			underscore = false
		} else if b.Len() > 0 && !underscore {
			b.WriteByte('_')
			underscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}
