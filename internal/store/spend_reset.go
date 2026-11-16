package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/money"
)

// SpendResetScopes selects which used-quota counters to restart.
type SpendResetScopes struct {
	Total   bool
	Daily   bool
	Weekly  bool
	Monthly bool
}

func (s SpendResetScopes) Any() bool {
	return s.Total || s.Daily || s.Weekly || s.Monthly
}

type spendResetTimes struct {
	Total   int64
	Daily   int64
	Weekly  int64
	Monthly int64
}

func periodStart(calendarStart, resetAt int64) int64 {
	return periodStartMode(calendarStart, resetAt, false)
}

// periodStartMode returns the spend-window start. Sticky mode keeps a reset
// anchor past the next UTC boundary so a subscription cycle is not cut into
// calendar days. Calendar mode still prefers the later of the two.
func periodStartMode(calendarStart, resetAt int64, sticky bool) int64 {
	if sticky && resetAt > 0 {
		return resetAt
	}
	if resetAt > calendarStart {
		return resetAt
	}
	return calendarStart
}

const (
	settingFollowPlanCycle   = "follow_plan_cycle"
	settingPlanCycleResetAt  = "plan_cycle_reset_at_unix_ms"
	settingPlanCycleResetWhy = "plan_cycle_reset_detail"
)

func keyFollowsPlanCycle(ctx context.Context, db queryRower, keyID string) (bool, error) {
	var on int
	err := db.QueryRowContext(ctx, `SELECT follow_plan_cycle FROM plugin_keys WHERE id = ?`, keyID).Scan(&on)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("load key plan cycle: %w", err)
	}
	return on == 1, nil
}

func (s *Store) ListPlanAutoResetKeyIDs(ctx context.Context, provider, authID string) ([]string, error) {
	provider, authID = planAuthProvider(provider, authID), planAuthID(provider, authID)
	if provider == "" || authID == "" {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM plugin_keys
		WHERE revoked_at_unix_ms IS NULL AND plan_auto_reset = 1
		AND plan_auth_provider = ? AND plan_auth_id = ?
		ORDER BY created_at_unix_ms`, provider, authID)
	if err != nil {
		return nil, fmt.Errorf("list plan cycle keys: %w", err)
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan plan cycle key: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func followPlanCycle(ctx context.Context, db queryRower) (bool, error) {
	value, ok, err := pluginSetting(ctx, db, settingFollowPlanCycle)
	if err != nil || !ok {
		return false, err
	}
	return value == "1", nil
}

func pluginSetting(ctx context.Context, db queryRower, key string) (string, bool, error) {
	var value string
	err := db.QueryRowContext(ctx, `SELECT value FROM plugin_settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("load plugin setting %s: %w", key, err)
	}
	return value, true, nil
}

func (s *Store) SetFollowPlanCycle(ctx context.Context, on bool) error {
	value := "0"
	if on {
		value = "1"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO plugin_settings(key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, settingFollowPlanCycle, value)
	if err != nil {
		return fmt.Errorf("set follow plan cycle: %w", err)
	}
	return nil
}

// EnsureFollowPlanCycle seeds the switch once. A later console change is kept
// across restarts and host config reloads.
func (s *Store) EnsureFollowPlanCycle(ctx context.Context, fallback bool) (bool, error) {
	value := "0"
	if fallback {
		value = "1"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO plugin_settings(key, value) VALUES (?, ?)
		ON CONFLICT(key) DO NOTHING`, settingFollowPlanCycle, value)
	if err != nil {
		return false, fmt.Errorf("ensure follow plan cycle: %w", err)
	}
	return followPlanCycle(ctx, s.db)
}

// PlanCycleStatus is the operator-facing state of subscription-cycle spend.
type PlanCycleStatus struct {
	Enabled bool
	ResetAt *time.Time
	Detail  string
}

func (s *Store) PlanCycleStatus(ctx context.Context) (PlanCycleStatus, error) {
	enabled, err := followPlanCycle(ctx, s.db)
	if err != nil {
		return PlanCycleStatus{}, err
	}
	status := PlanCycleStatus{Enabled: enabled}
	raw, ok, err := pluginSetting(ctx, s.db, settingPlanCycleResetAt)
	if err != nil {
		return PlanCycleStatus{}, err
	}
	if ok {
		ms, convErr := parseInt64Setting(raw)
		if convErr == nil && ms > 0 {
			at := time.UnixMilli(ms).UTC()
			status.ResetAt = &at
		}
	}
	detail, _, err := pluginSetting(ctx, s.db, settingPlanCycleResetWhy)
	if err != nil {
		return PlanCycleStatus{}, err
	}
	status.Detail = detail
	return status, nil
}

func (s *Store) SetPlanCycleResetNotice(ctx context.Context, at time.Time, detail string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := putPluginSetting(ctx, tx, settingPlanCycleResetAt, fmt.Sprintf("%d", at.UTC().UnixMilli())); err != nil {
		return err
	}
	if err := putPluginSetting(ctx, tx, settingPlanCycleResetWhy, detail); err != nil {
		return err
	}
	return tx.Commit()
}

func putPluginSetting(ctx context.Context, tx *sql.Tx, key, value string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO plugin_settings(key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("save plugin setting %s: %w", key, err)
	}
	return nil
}

func parseInt64Setting(raw string) (int64, error) {
	var n int64
	_, err := fmt.Sscan(raw, &n)
	return n, err
}

func nullUnixMilli(value sql.NullInt64) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func loadSpendResetTimes(ctx context.Context, db queryRower, keyID string) (spendResetTimes, error) {
	var total, daily, weekly, monthly sql.NullInt64
	err := db.QueryRowContext(ctx, `SELECT total_spend_reset_at_unix_ms, daily_spend_reset_at_unix_ms,
		weekly_spend_reset_at_unix_ms, monthly_spend_reset_at_unix_ms
		FROM plugin_keys WHERE id = ?`, keyID).Scan(&total, &daily, &weekly, &monthly)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return spendResetTimes{}, nil
		}
		return spendResetTimes{}, fmt.Errorf("load spend reset times: %w", err)
	}
	return spendResetTimes{
		Total:   nullUnixMilli(total),
		Daily:   nullUnixMilli(daily),
		Weekly:  nullUnixMilli(weekly),
		Monthly: nullUnixMilli(monthly),
	}, nil
}

func keyPeriodStarts(ctx context.Context, db queryRower, keyID string, nowUnixMilli int64) (day, week, month int64, err error) {
	_, day, week, month, err = keyTokenPeriodStarts(ctx, db, keyID, nowUnixMilli)
	return day, week, month, err
}

// keyTokenPeriodStarts returns the token usage windows. Total begins at the
// most recent total reset, while day/week/month also honor their UTC boundary.
func keyTokenPeriodStarts(ctx context.Context, db queryRower, keyID string, nowUnixMilli int64) (total, day, week, month int64, err error) {
	resets, err := loadSpendResetTimes(ctx, db, keyID)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	sticky, err := keyFollowsPlanCycle(ctx, db, keyID)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return resets.Total,
		periodStartMode(utcDayStart(nowUnixMilli), resets.Daily, sticky),
		periodStartMode(utcWeekStart(nowUnixMilli), resets.Weekly, sticky),
		periodStartMode(utcMonthStart(nowUnixMilli), resets.Monthly, false), nil
}

func uniquePluginKeyIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// ResetPluginKeySpend starts a new used-quota cycle for selected keys.
// Usage ledger rows are kept. Total reset zeroes settled_spend; day/week/month
// reset only moves the period start to now until the next UTC calendar boundary.
func (s *Store) ResetPluginKeySpend(ctx context.Context, ids []string, all bool, scopes SpendResetScopes) (int, error) {
	return s.resetPluginKeySpend(ctx, ids, all, scopes, 0, 0)
}

// ResetPluginKeySpendAt is ResetPluginKeySpend with upstream window starts.
// A zero time falls back to now. Manual resets keep using now.
func (s *Store) ResetPluginKeySpendAt(ctx context.Context, ids []string, scopes SpendResetScopes, dailyAt, weeklyAt time.Time) (int, error) {
	return s.resetPluginKeySpend(ctx, ids, false, scopes, unixMilliOrZero(dailyAt), unixMilliOrZero(weeklyAt))
}

// AlignUnsetPlanCycleAnchors sets the day and week start on plan-cycle keys
// that do not have one yet. Existing anchors and opted-out keys are left alone,
// so this does not clear usage already inside the current window.
func (s *Store) AlignUnsetPlanCycleAnchors(ctx context.Context, provider, authID string, dailyAt, weeklyAt time.Time) error {
	provider, authID = planAuthProvider(provider, authID), planAuthID(provider, authID)
	dailyMS := unixMilliOrZero(dailyAt)
	weeklyMS := unixMilliOrZero(weeklyAt)
	if provider == "" || authID == "" || (dailyMS == 0 && weeklyMS == 0) {
		return nil
	}
	now := nowUnixMilli()
	_, err := s.db.ExecContext(ctx, `UPDATE plugin_keys SET
		daily_spend_reset_at_unix_ms = CASE WHEN ? > 0 AND COALESCE(daily_spend_reset_at_unix_ms, 0) = 0 THEN ? ELSE daily_spend_reset_at_unix_ms END,
		weekly_spend_reset_at_unix_ms = CASE WHEN ? > 0 AND COALESCE(weekly_spend_reset_at_unix_ms, 0) = 0 THEN ? ELSE weekly_spend_reset_at_unix_ms END,
		updated_at_unix_ms = ?
		WHERE revoked_at_unix_ms IS NULL AND follow_plan_cycle = 1
		AND plan_auth_provider = ? AND plan_auth_id = ?
		AND (
			(? > 0 AND COALESCE(daily_spend_reset_at_unix_ms, 0) = 0)
			OR (? > 0 AND COALESCE(weekly_spend_reset_at_unix_ms, 0) = 0)
		)`,
		dailyMS, dailyMS, weeklyMS, weeklyMS, now, provider, authID, dailyMS, weeklyMS)
	if err != nil {
		return fmt.Errorf("align plan cycle anchors: %w", err)
	}
	return nil
}

func unixMilliOrZero(at time.Time) int64 {
	if at.IsZero() {
		return 0
	}
	return at.UTC().UnixMilli()
}

func anchorOrNow(anchor, now int64) int64 {
	if anchor > 0 {
		return anchor
	}
	return now
}

func (s *Store) resetPluginKeySpend(ctx context.Context, ids []string, all bool, scopes SpendResetScopes, dailyAt, weeklyAt int64) (int, error) {
	if !scopes.Any() {
		return 0, fmt.Errorf("%w: select at least one of total, daily, weekly, monthly", ErrInvalidArgument)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if all {
		rows, err := tx.QueryContext(ctx, `SELECT id FROM plugin_keys WHERE revoked_at_unix_ms IS NULL ORDER BY created_at_unix_ms`)
		if err != nil {
			return 0, fmt.Errorf("list keys for spend reset: %w", err)
		}
		listed := make([]string, 0)
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return 0, fmt.Errorf("scan key for spend reset: %w", err)
			}
			listed = append(listed, id)
		}
		if err := rows.Close(); err != nil {
			return 0, err
		}
		if err := rows.Err(); err != nil {
			return 0, err
		}
		ids = listed
	} else {
		ids = uniquePluginKeyIDs(ids)
		if len(ids) == 0 {
			return 0, fmt.Errorf("%w: key id is required", ErrInvalidArgument)
		}
	}
	now := nowUnixMilli()
	for _, id := range ids {
		if err := resetOnePluginKeySpend(ctx, tx, id, scopes, now, dailyAt, weeklyAt); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(ids), nil
}

func resetOnePluginKeySpend(ctx context.Context, tx *sql.Tx, id string, scopes SpendResetScopes, now, dailyAt, weeklyAt int64) error {
	var callerID string
	var settled int64
	if err := tx.QueryRowContext(ctx, `SELECT caller_id, settled_spend_micro_usd FROM plugin_keys WHERE id = ?`, id).Scan(&callerID, &settled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPluginKeyNotFound
		}
		return fmt.Errorf("load key for spend reset: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE plugin_keys SET
		settled_spend_micro_usd = CASE WHEN ? THEN 0 ELSE settled_spend_micro_usd END,
		total_spend_reset_at_unix_ms = CASE WHEN ? THEN ? ELSE total_spend_reset_at_unix_ms END,
		daily_spend_reset_at_unix_ms = CASE WHEN ? THEN ? ELSE daily_spend_reset_at_unix_ms END,
		weekly_spend_reset_at_unix_ms = CASE WHEN ? THEN ? ELSE weekly_spend_reset_at_unix_ms END,
		monthly_spend_reset_at_unix_ms = CASE WHEN ? THEN ? ELSE monthly_spend_reset_at_unix_ms END,
		updated_at_unix_ms = ?
		WHERE id = ?`,
		boolInt(scopes.Total),
		boolInt(scopes.Total), now,
		boolInt(scopes.Daily), anchorOrNow(dailyAt, now),
		boolInt(scopes.Weekly), anchorOrNow(weeklyAt, now),
		boolInt(scopes.Monthly), now,
		now, id)
	if err != nil {
		return fmt.Errorf("reset key spend: %w", err)
	}
	if err := requireOneRow(result, ErrPluginKeyNotFound); err != nil {
		return err
	}
	cleared := int64(0)
	if scopes.Total {
		cleared = settled
	}
	details, err := json.Marshal(map[string]any{
		"total":                  scopes.Total,
		"daily":                  scopes.Daily,
		"weekly":                 scopes.Weekly,
		"monthly":                scopes.Monthly,
		"cleared_settled_spend":  cleared,
		"previous_settled_spend": settled,
	})
	if err != nil {
		return fmt.Errorf("encode spend reset audit: %w", err)
	}
	return insertAudit(ctx, tx, callerID, id, "", "quota_spend_reset", money.MicroUSD(cleared), string(details), now)
}
