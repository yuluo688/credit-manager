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
	// Same headers the Codex desktop client sends; read-only GETs only. Never
	// call rate-limit-reset-credits/consume from here.
	ch := h.Clone()
	ch.Set("OpenAI-Beta", "codex-1")
	ch.Set("Originator", "Codex Desktop")
	if credits, e := request(ctx, s, cb, "GET", "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits", ch, nil); e == nil {
		snap.ResetCredits = number(credits, "available_count", "availableCount")
		snap.ResetCreditList = parseCodexResetCredits(credits)
		snap.resetCreditsFetched = true
	}
	if history, e := request(ctx, s, cb, "GET", "https://chatgpt.com/backend-api/wham/usage/plan_limit_history?days=7", ch, nil); e == nil {
		snap.planHistory = parseCodexPlanHistory(history)
		snap.PlanHistoryStatus = fmt.Sprintf("ok: %d periods", len(snap.planHistory))
	} else {
		// Optional endpoint (404 on most plans); keep the outcome for diagnostics only.
		snap.PlanHistoryStatus = truncateQuotaDiag(e.Error())
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

// codexResetCredit mirrors one entry of rate-limit-reset-credits credits[].
type codexResetCredit struct {
	ID        string     `json:"id"`
	ResetType string     `json:"reset_type,omitempty"`
	Status    string     `json:"status,omitempty"`
	GrantedAt *time.Time `json:"granted_at,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func parseCodexResetCredits(d map[string]any) []codexResetCredit {
	out := []codexResetCredit{}
	for _, x := range anySlice(d, "credits") {
		m, ok := x.(map[string]any)
		if !ok {
			continue
		}
		c := codexResetCredit{
			ID:        findText(m, "id"),
			ResetType: findText(m, "reset_type", "resetType"),
			Status:    findText(m, "status"),
			GrantedAt: codexTime(m, "granted_at", "grantedAt"),
			ExpiresAt: codexTime(m, "expires_at", "expiresAt"),
		}
		if c.ID != "" {
			out = append(out, c)
		}
	}
	return out
}

// codexPlanPeriod is one plan_limit_history period (authoritative past window).
type codexPlanPeriod struct {
	WindowMinutes int64
	StartsAt      time.Time
	EndsAt        time.Time
	Used          *float64
}

func parseCodexPlanHistory(d map[string]any) []codexPlanPeriod {
	var out []codexPlanPeriod
	for _, x := range anySlice(d, "periods") {
		m, ok := x.(map[string]any)
		if !ok {
			continue
		}
		minutes := number(m, "window_minutes", "windowMinutes")
		start, end := codexTime(m, "starts_at", "startsAt"), codexTime(m, "ends_at", "endsAt")
		if minutes == nil || *minutes <= 0 || start == nil || end == nil || !end.After(*start) {
			continue
		}
		p := codexPlanPeriod{WindowMinutes: int64(*minutes), StartsAt: *start, EndsAt: *end}
		if bp := number(m, "used_basis_points", "usedBasisPoints"); bp != nil {
			v := *bp / 100
			p.Used = &v
		}
		out = append(out, p)
	}
	return out
}

func codexTime(m map[string]any, keys ...string) *time.Time {
	for _, k := range keys {
		switch v := m[k].(type) {
		case string:
			if t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(v)); err == nil {
				u := t.UTC()
				return &u
			}
		case float64:
			if v > 0 {
				sec := v
				if sec > 1e12 {
					sec /= 1000
				}
				u := time.Unix(int64(sec), 0).UTC()
				return &u
			}
		}
	}
	return nil
}

func truncateQuotaDiag(v string) string {
	if len(v) > 160 {
		return v[:160]
	}
	return v
}
