package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

func kimi(ctx context.Context, s AuthQuotaSource, cb string, c quotaCredentials) (quotaSnapshot, error) {
	d, e := request(ctx, s, cb, "GET", "https://api.kimi.com/coding/v1/usages", headers(c.token), nil)
	if e != nil {
		return quotaSnapshot{}, e
	}
	w := kimiWindows(d)
	if len(w) == 0 {
		return quotaSnapshot{}, fmt.Errorf("kimi quota response has no limits")
	}
	return quotaSnapshot{Plan: quotaPlanText(d, "plan", "plan_name", "membership", "membershipType", "membership_type", "package", "packageName"), Windows: w}, nil
}
func kimiWindows(d map[string]any) []AuthQuotaWindow {
	var out []AuthQuotaWindow
	limits := anySlice(d, "limits")
	usage, ok := objectAny(d, "usage")
	if !ok {
		usage = d
	}
	if usage != nil {
		if w, ok := kimiWindow("weekly", "周限额", usage, nil); ok {
			applyKimiWindowMeta(&w, "weekly", nil)
			out = append(out, w)
		}
	}
	for i, x := range limits {
		m, _ := x.(map[string]any)
		detail := m
		if nested, ok := objectAny(m, "detail"); ok {
			detail = nested
		}
		id := first(findText(m, "id", "name", "label"), fmt.Sprintf("limit-%d", i+1))
		window, _ := objectAny(m, "window")
		if w, ok := kimiWindow(id, id, detail, window); ok {
			applyKimiWindowMeta(&w, id, window)
			out = append(out, w)
		}
	}
	return out
}

func applyKimiWindowMeta(w *AuthQuotaWindow, id string, window map[string]any) {
	if d := kimiDuration(window); d != nil {
		w.DurationSeconds = d
	} else if kimiLooksWeekly(id, w) && (w.DurationSeconds == nil || *w.DurationSeconds <= 0) {
		w.DurationSeconds = int64ptr(604800)
	}
	if w.ResetsAt != nil && w.DurationSeconds != nil {
		start := w.ResetsAt.Add(-time.Duration(*w.DurationSeconds) * time.Second)
		w.CycleStartAt = &start
		w.CycleStartSource = "inferred_window_start"
	}
}

func kimiLooksWeekly(id string, w *AuthQuotaWindow) bool {
	name := strings.ToLower(first(id, w.ID, w.Label))
	if strings.Contains(name, "week") || strings.Contains(name, "7d") || strings.Contains(name, "seven") || strings.Contains(name, "周") {
		return true
	}
	return w.DurationSeconds != nil && *w.DurationSeconds >= 500000 && *w.DurationSeconds <= 700000
}
func kimiWindow(id, label string, detail, window map[string]any) (AuthQuotaWindow, bool) {
	reset := timeField(detail, "resetTime", "resetAt", "reset_time", "reset_at")
	if reset == nil {
		if seconds := number(detail, "resetIn", "reset_in", "ttl"); seconds != nil {
			v := time.Now().UTC().Add(time.Duration(*seconds * float64(time.Second)))
			reset = &v
		}
	}
	w, ok := valueWindow(id, label, "requests", number(detail, "limit", "total", "max", "quota"), number(detail, "used", "usage", "consumed"), number(detail, "utilization", "used_percent"), reset, kimiDuration(window))
	if !ok {
		return w, false
	}
	if remaining := number(detail, "remaining"); remaining != nil {
		w.Remaining = remaining
		if w.Limit != nil && *w.Limit > 0 {
			ratio := math.Max(0, *remaining / *w.Limit)
			w.RemainingRatio = &ratio
			if w.Used == nil {
				used := math.Max(0, *w.Limit-*remaining)
				w.Used = &used
			}
		}
	}
	if w.ResetsAt != nil && w.DurationSeconds != nil {
		start := w.ResetsAt.Add(-time.Duration(*w.DurationSeconds) * time.Second)
		w.CycleStartAt = &start
	}
	return w, true
}

func kimiDuration(window map[string]any) *int64 {
	if window == nil {
		return nil
	}
	v := number(window, "duration")
	if v == nil {
		return nil
	}
	unit := strings.ToLower(findText(window, "timeUnit", "time_unit", "duration_unit"))
	multiplier := float64(1)
	switch {
	case strings.Contains(unit, "week"):
		multiplier = 604800
	case strings.Contains(unit, "day"):
		multiplier = 86400
	case strings.Contains(unit, "hour"):
		multiplier = 3600
	case strings.Contains(unit, "minute"):
		multiplier = 60
	}
	seconds := int64(*v * multiplier)
	return &seconds
}
