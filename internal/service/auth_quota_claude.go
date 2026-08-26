package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func claude(ctx context.Context, s AuthQuotaSource, cb string, c quotaCredentials) (quotaSnapshot, error) {
	h := headers(c.token)
	h.Set("Anthropic-Beta", "oauth-2025-04-20")
	h.Set("Anthropic-Version", "2023-06-01")
	d, e := request(ctx, s, cb, "GET", "https://api.anthropic.com/api/oauth/usage", h, nil)
	if e != nil {
		return quotaSnapshot{}, e
	}
	var w []AuthQuotaWindow
	for _, spec := range []struct {
		id, scope, scopeID string
		duration           *int64
	}{
		{"five_hour", "account", "", int64ptr(18000)},
		{"seven_day", "account", "", int64ptr(604800)},
		{"seven_day_oauth_apps", "account", "", int64ptr(604800)},
		{"seven_day_opus", "model", "opus", int64ptr(604800)},
		{"seven_day_sonnet", "model", "sonnet", int64ptr(604800)},
		{"seven_day_cowork", "account", "", int64ptr(604800)},
		{"seven_day_omelette", "account", "", int64ptr(604800)},
		{"iguana_necktie", "account", "", nil},
	} {
		if m, ok := objectAny(d, spec.id, strings.ReplaceAll(spec.id, "_", "")); ok {
			if x, ok := utilWindow(spec.id, spec.id, spec.scope, spec.scopeID, spec.duration, m); ok {
				w = append(w, x)
			}
		}
	}
	if m, ok := object(d, "extra_usage"); ok {
		if x, ok := valueWindow("extra_usage", "Extra usage", "credits", number(m, "monthly_limit"), number(m, "used_credits"), number(m, "utilization"), timeField(m, "resets_at"), nil); ok {
			w = append(w, x)
		}
	}
	if len(w) == 0 {
		return quotaSnapshot{}, fmt.Errorf("claude quota response has no recognized windows")
	}
	return quotaSnapshot{Plan: claudePlan(ctx, s, cb, h, d), Windows: w}, nil
}

func claudePlan(ctx context.Context, s AuthQuotaSource, cb string, h http.Header, usage map[string]any) string {
	plan := quotaPlanText(usage, "plan_type", "plan", "subscriptionType", "subscription_type", "rate_limit_tier")
	profile, err := request(ctx, s, cb, "GET", "https://api.anthropic.com/api/oauth/profile", h, nil)
	if err == nil {
		plan = first(
			quotaPlanText(profile, "rate_limit_tier", "rateLimitTier"),
			quotaPlanText(profile, "organization_type", "organizationType"),
			quotaPlanText(profile, "subscriptionType", "subscription_type"),
			plan,
		)
		if plan == "" {
			if max, ok := findBool(profile, "has_claude_max"); ok && max {
				plan = "max"
			} else if pro, ok := findBool(profile, "has_claude_pro"); ok && pro {
				plan = "pro"
			}
		}
	}
	return plan
}
