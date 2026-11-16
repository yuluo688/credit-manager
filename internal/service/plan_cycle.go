package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/store"
)

const (
	// planCycleJitter ignores the few seconds of reset_at noise between syncs.
	planCycleJitter = 2 * time.Minute
	// Windows up to two days are the subscription "day" bucket (Codex primary
	// is often five hours). Longer windows up to two weeks are the week bucket.
	planCycleDailyMax  = 48 * time.Hour
	planCycleWeeklyMax = 14 * 24 * time.Hour
)

// planCycleReset is the child-key effect of one upstream quota sync.
// Anchors are the upstream window starts, not the moment the sync noticed them.
type planCycleReset struct {
	store.SpendResetScopes
	DailyAt  time.Time
	WeeklyAt time.Time
}

// planCycleResetScopes decides which child-key spend counters a quota sync
// should restart. The first snapshot is ignored so enabling the feature does
// not zero usage that is already inside the current plan window. A used reset
// credit does not add scopes by itself; the matching window must have moved.
func planCycleResetScopes(prev *quotaSnapshot, next quotaSnapshot) store.SpendResetScopes {
	return planCycleResetPlan(prev, next).SpendResetScopes
}

func planCycleResetPlan(prev *quotaSnapshot, next quotaSnapshot) planCycleReset {
	var shift planCycleReset
	if prev == nil {
		return shift
	}
	previous := map[string]AuthQuotaWindow{}
	for _, window := range prev.Windows {
		if !planCycleCandidate(window) {
			continue
		}
		previous[windowBaselineID(&window)] = window
	}
	for _, window := range next.Windows {
		if !planCycleCandidate(window) {
			continue
		}
		old, ok := previous[windowBaselineID(&window)]
		if !ok {
			continue
		}
		oldStart, oldEnd, oldOK := quotaWindowBounds(old)
		newStart, newEnd, newOK := quotaWindowBounds(window)
		if !oldOK || !newOK || !planCycleRolled(oldStart, oldEnd, newStart, newEnd) {
			continue
		}
		daily, weekly := planCycleScope(newEnd.Sub(newStart))
		if daily {
			shift.Daily = true
			shift.DailyAt = earlierTime(shift.DailyAt, newStart)
		}
		if weekly {
			shift.Weekly = true
			shift.WeeklyAt = earlierTime(shift.WeeklyAt, newStart)
		}
	}
	// The day bucket ends with the week. A short window still resets the day
	// on its own; a week rollover must not leave that usage in place.
	if shift.Weekly {
		shift.Daily = true
		if shift.DailyAt.IsZero() {
			shift.DailyAt = shift.WeeklyAt
		}
	}
	return shift
}

// planCycleCurrentAnchors is where an unset plan-cycle key should start
// counting. Only the account's main day and week windows count.
func planCycleCurrentAnchors(snap quotaSnapshot) (daily, weekly time.Time) {
	for _, window := range snap.Windows {
		if !planCycleCandidate(window) {
			continue
		}
		start, end, ok := quotaWindowBounds(window)
		if !ok {
			continue
		}
		isDaily, isWeekly := planCycleScope(end.Sub(start))
		if isDaily {
			daily = earlierTime(daily, start)
		}
		if isWeekly {
			weekly = earlierTime(weekly, start)
		}
	}
	return daily, weekly
}

func earlierTime(current, candidate time.Time) time.Time {
	if candidate.IsZero() {
		return current
	}
	if current.IsZero() || candidate.Before(current) {
		return candidate
	}
	return current
}

func planCycleCandidate(window AuthQuotaWindow) bool {
	if window.Partial || window.ResetsAt == nil || quotaWindowExcludedFromCycle(window) || planCycleAuxiliary(window) {
		return false
	}
	_, _, ok := quotaWindowBounds(window)
	return ok
}

// planCycleAuxiliary rejects extra limits that are not the account plan,
// such as Codex code review and additional rate limits.
func planCycleAuxiliary(window AuthQuotaWindow) bool {
	id := strings.ToLower(strings.TrimSpace(window.ID))
	label := strings.ToLower(strings.TrimSpace(window.Label))
	scopeID := strings.ToLower(strings.TrimSpace(window.ScopeID))
	if strings.Contains(id, "code-review") || strings.Contains(id, "code_review") || strings.HasPrefix(id, "additional-") || strings.Contains(id, "additional_") {
		return true
	}
	if strings.Contains(label, "code review") || strings.Contains(label, "code-review") {
		return true
	}
	return strings.Contains(scopeID, "code_review")
}

func planCycleScope(duration time.Duration) (daily, weekly bool) {
	if duration <= 0 {
		return false, false
	}
	if duration <= planCycleDailyMax {
		return true, false
	}
	if duration <= planCycleWeeklyMax {
		return false, true
	}
	return false, false
}

func planCycleRolled(oldStart, oldEnd, newStart, newEnd time.Time) bool {
	if newStart.After(oldStart.Add(planCycleJitter)) {
		return true
	}
	return !oldEnd.IsZero() && newEnd.Before(oldEnd.Add(-planCycleJitter))
}

func (s *Service) applyPlanCycleReset(ctx context.Context, provider, authID string, prev *quotaSnapshot, next quotaSnapshot, creditUsed bool) {
	if s == nil || s.store == nil {
		return
	}
	dailyAt, weeklyAt := planCycleCurrentAnchors(next)
	if !dailyAt.IsZero() || !weeklyAt.IsZero() {
		_ = s.store.AlignUnsetPlanCycleAnchors(ctx, provider, authID, dailyAt, weeklyAt)
	}
	shift := planCycleResetPlan(prev, next)
	if !shift.Any() {
		return
	}
	ids, err := s.store.ListPlanAutoResetKeyIDs(ctx, provider, authID)
	if err != nil || len(ids) == 0 {
		return
	}
	n, err := s.store.ResetPluginKeySpendAt(ctx, ids, shift.SpendResetScopes, shift.DailyAt, shift.WeeklyAt)
	if err != nil || n == 0 {
		return
	}
	detail := fmt.Sprintf("%s/%s", provider, authID)
	switch {
	case creditUsed:
		detail += " reset credit"
	case shift.Daily && shift.Weekly:
		detail += " plan cycle"
	case shift.Weekly:
		detail += " weekly window"
	default:
		detail += " daily window"
	}
	_ = s.store.SetPlanCycleResetNotice(ctx, time.Now().UTC(), detail)
}
