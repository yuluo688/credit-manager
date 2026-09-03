package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
	if resetAt > calendarStart {
		return resetAt
	}
	return calendarStart
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
	return resets.Total,
		periodStart(utcDayStart(nowUnixMilli), resets.Daily),
		periodStart(utcWeekStart(nowUnixMilli), resets.Weekly),
		periodStart(utcMonthStart(nowUnixMilli), resets.Monthly), nil
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
		if err := resetOnePluginKeySpend(ctx, tx, id, scopes, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(ids), nil
}

func resetOnePluginKeySpend(ctx context.Context, tx *sql.Tx, id string, scopes SpendResetScopes, now int64) error {
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
		boolInt(scopes.Daily), now,
		boolInt(scopes.Weekly), now,
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
