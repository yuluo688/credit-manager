package service

import (
	"testing"
	"time"
)

func TestPlanCycleResetScopes(t *testing.T) {
	start := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	fiveHours := int64((5 * time.Hour).Seconds())
	week := int64((7 * 24 * time.Hour).Seconds())
	daily := windowAt("primary", start, fiveHours)
	weekly := windowAt("secondary", start, week)
	prev := &quotaSnapshot{Windows: []AuthQuotaWindow{daily, weekly}}

	if got := planCycleResetScopes(nil, quotaSnapshot{Windows: []AuthQuotaWindow{windowAt("primary", start.Add(5*time.Hour), fiveHours)}}); got.Any() {
		t.Fatalf("first snapshot must not reset, got %+v", got)
	}
	if got := planCycleResetScopes(prev, quotaSnapshot{Windows: []AuthQuotaWindow{daily, weekly}}); got.Any() {
		t.Fatalf("unchanged windows reset %+v", got)
	}
	jitter := windowAt("primary", start.Add(30*time.Second), fiveHours)
	if got := planCycleResetScopes(prev, quotaSnapshot{Windows: []AuthQuotaWindow{jitter, weekly}}); got.Any() {
		t.Fatalf("jitter reset %+v", got)
	}

	rolledDaily := windowAt("primary", start.Add(5*time.Hour), fiveHours)
	got := planCycleResetPlan(prev, quotaSnapshot{Windows: []AuthQuotaWindow{rolledDaily, weekly}})
	if !got.Daily || got.Weekly || got.Total || got.Monthly || !got.DailyAt.Equal(start.Add(5*time.Hour)) {
		t.Fatalf("5h rollover = %+v, want daily only at the new window start", got)
	}

	earlyWeek := weekly
	earlier := start.Add(time.Duration(week) * time.Second).Add(-6 * time.Hour)
	earlyWeek.ResetsAt = &earlier
	earlyStart := earlier.Add(-7 * 24 * time.Hour)
	earlyWeek.CycleStartAt = &earlyStart
	got = planCycleResetPlan(prev, quotaSnapshot{Windows: []AuthQuotaWindow{daily, earlyWeek}})
	if !got.Daily || !got.Weekly || got.Total || got.Monthly || !got.DailyAt.Equal(earlyStart) || !got.WeeklyAt.Equal(earlyStart) {
		t.Fatalf("early weekly reset = %+v, want both anchors at %s", got, earlyStart)
	}

	oldReview := windowAt("code-review-primary", start, fiveHours)
	oldExtra := windowAt("additional-images-primary", start, fiveHours)
	codeReview := windowAt("code-review-primary", start.Add(5*time.Hour), fiveHours)
	codeReview.Label = "Code review primary"
	extra := windowAt("additional-images-primary", start.Add(5*time.Hour), fiveHours)
	prevWithExtra := &quotaSnapshot{Windows: []AuthQuotaWindow{daily, weekly, oldReview, oldExtra}}
	if extraReset := planCycleResetScopes(prevWithExtra, quotaSnapshot{Windows: []AuthQuotaWindow{daily, weekly, codeReview, extra}}); extraReset.Any() {
		t.Fatalf("auxiliary windows reset %+v", extraReset)
	}
	rolledWeek := windowAt("secondary", start.Add(7*24*time.Hour), week)
	both := planCycleResetPlan(prev, quotaSnapshot{Windows: []AuthQuotaWindow{rolledDaily, rolledWeek, codeReview}})
	if !both.DailyAt.Equal(start.Add(5*time.Hour)) || !both.WeeklyAt.Equal(start.Add(7*24*time.Hour)) {
		t.Fatalf("mixed rollover anchors = daily %s weekly %s", both.DailyAt, both.WeeklyAt)
	}
	currentDaily, currentWeekly := planCycleCurrentAnchors(quotaSnapshot{Windows: []AuthQuotaWindow{daily, weekly, codeReview}})
	if !currentDaily.Equal(start) || !currentWeekly.Equal(start) {
		t.Fatalf("current anchors = %s %s, want the main windows", currentDaily, currentWeekly)
	}

	unchanged := planCycleResetScopes(prev, quotaSnapshot{Windows: []AuthQuotaWindow{daily, weekly}})
	if unchanged.Any() {
		t.Fatalf("reset credit without a window change = %+v", unchanged)
	}
	onlyDaily := planCycleResetScopes(prev, quotaSnapshot{Windows: []AuthQuotaWindow{rolledDaily, weekly}})
	if !onlyDaily.Daily || onlyDaily.Weekly {
		t.Fatalf("reset credit with only the 5h window = %+v, want daily only", onlyDaily)
	}
}

func windowAt(id string, start time.Time, seconds int64) AuthQuotaWindow {
	end := start.Add(time.Duration(seconds) * time.Second)
	return AuthQuotaWindow{ID: id, CycleStartAt: &start, ResetsAt: &end, DurationSeconds: &seconds}
}
