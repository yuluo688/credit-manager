package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

func xai(ctx context.Context, s AuthQuotaSource, cb string, c quotaCredentials) (quotaSnapshot, error) {
	h := headers(c.token)
	h.Set("Accept", "*/*")
	h.Set("X-XAI-Token-Auth", "xai-grok-cli")
	h.Set("X-Grok-Client-Version", "0.2.93")
	h.Set("User-Agent", "xai-grok-workspace/0.2.93")
	if c.userID != "" {
		h.Set("X-Userid", c.userID)
	}
	credits, creditsErr := request(ctx, s, cb, "GET", "https://cli-chat-proxy.grok.com/v1/billing?format=credits", h, nil)
	billing, billingErr := request(ctx, s, cb, "GET", "https://cli-chat-proxy.grok.com/v1/billing", h, nil)
	if creditsErr != nil && billingErr != nil {
		return quotaSnapshot{}, billingErr
	}
	w := append(xaiCreditWindows(xaiBillingMap(credits)), xaiWindows(xaiBillingMap(billing))...)
	if len(w) == 0 {
		return quotaSnapshot{}, fmt.Errorf("xAI billing response has no balances")
	}
	return quotaSnapshot{Plan: first(
		quotaPlanText(credits, "plan", "planName", "subscription", "subscriptionName", "product"),
		quotaPlanText(billing, "plan", "planName", "subscription", "subscriptionName"),
	), Windows: w}, nil
}

func xaiBillingMap(d map[string]any) map[string]any {
	if d == nil {
		return nil
	}
	if m, ok := object(d, "config"); ok {
		return m
	}
	return d
}

func xaiPeriodTimes(d, period map[string]any) (start, end *time.Time, seconds *int64) {
	start = timeField(period, "start")
	if start == nil {
		start = timeField(d, "billingPeriodStart", "billing_period_start")
	}
	end = timeField(period, "end")
	if end == nil {
		end = timeField(d, "billingPeriodEnd", "billing_period_end")
	}
	if start != nil && end != nil && end.After(*start) {
		value := int64(end.Sub(*start) / time.Second)
		seconds = &value
	}
	return start, end, seconds
}

func xaiPercentWindow(id, label, scope, scopeID string, usedPercent *float64, start, end *time.Time, seconds *int64) (AuthQuotaWindow, bool) {
	limit := 100.0
	var used *float64
	if usedPercent != nil {
		value := *usedPercent
		if !finiteNonNegative(value) || value > 100 {
			return AuthQuotaWindow{}, false
		}
		used = &value
	}
	w, ok := valueWindow(id, label, "percentage", &limit, used, nil, end, seconds)
	if !ok && usedPercent == nil {
		w = AuthQuotaWindow{ID: id, Label: label, Scope: first(scope, "account"), ScopeID: scopeID, Mode: "rolling", Unit: "percentage", ResetsAt: end, DurationSeconds: seconds}
		ok = true
	}
	if !ok {
		return AuthQuotaWindow{}, false
	}
	w.Scope = first(scope, "account")
	w.ScopeID = scopeID
	w.Mode = "rolling"
	if start != nil {
		w.CycleStartAt = start
		w.CycleStartSource = "upstream_period"
	} else if w.ResetsAt != nil && w.DurationSeconds != nil {
		inferred := w.ResetsAt.Add(-time.Duration(*w.DurationSeconds) * time.Second)
		w.CycleStartAt = &inferred
		w.CycleStartSource = "inferred_week_start"
	}
	return w, true
}

func xaiCreditWindows(d map[string]any) []AuthQuotaWindow {
	if d == nil {
		return nil
	}
	period, _ := object(d, "currentPeriod")
	if period == nil {
		period, _ = object(d, "current_period")
	}
	periodType := strings.ToLower(first(findText(period, "type"), findText(d, "periodType", "period_type")))
	percent := number(d, "creditUsagePercent", "credit_usage_percent")
	products, _ := d["productUsage"].([]any)
	if products == nil {
		products, _ = d["product_usage"].([]any)
	}
	if percent == nil && !strings.Contains(periodType, "weekly") && len(products) == 0 {
		return nil
	}
	start, end, seconds := xaiPeriodTimes(d, period)
	if seconds == nil {
		seconds = int64ptr(604800)
	}
	var out []AuthQuotaWindow
	if w, ok := xaiPercentWindow("weekly", "周限额", "account", "", percent, start, end, seconds); ok {
		out = append(out, w)
	}
	for i, item := range products {
		m, _ := item.(map[string]any)
		name := first(findText(m, "product", "name", "label"), fmt.Sprintf("产品 %d", i+1))
		id := "weekly-" + first(codexSlug(name), fmt.Sprintf("%d", i+1))
		if w, ok := xaiPercentWindow(id, name, "product", name, number(m, "usagePercent", "usage_percent"), start, end, seconds); ok {
			out = append(out, w)
		}
	}
	return out
}

func xaiWindows(m map[string]any) []AuthQuotaWindow {
	var out []AuthQuotaWindow
	if m == nil {
		return out
	}
	end := timeField(m, "billingPeriodEnd", "billing_period_end")
	monthlyLimit := cent(firstNumber(m, "monthlyLimit", "monthly_limit"))
	monthlyUsed := cent(number(m, "used"))
	if w, ok := valueWindow("monthly", "Monthly", "currency", monthlyLimit, monthlyUsed, nil, end, nil); ok {
		w.Mode = "fixed"
		w.Currency = "USD"
		if end != nil {
			start := end.AddDate(0, -1, 0)
			w.CycleStartAt = &start
			w.CycleStartSource = "inferred_month_start"
		}
		out = append(out, w)
	}
	onDemandLimit := cent(firstNumber(m, "onDemandCap", "on_demand_cap"))
	onDemandUsed := cent(firstNumber(m, "onDemandUsed", "on_demand_used"))
	if onDemandUsed == nil && monthlyUsed != nil && monthlyLimit != nil {
		value := math.Max(0, *monthlyUsed-*monthlyLimit)
		onDemandUsed = &value
	}
	if onDemandLimit != nil && *onDemandLimit > 0 {
		if w, ok := valueWindow("on-demand", "On demand", "currency", onDemandLimit, onDemandUsed, nil, end, nil); ok {
			w.Mode = "balance"
			w.Currency = "USD"
			out = append(out, w)
		}
	}
	return out
}
