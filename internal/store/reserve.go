package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yuluo688/credit-manager/internal/money"
)

type ReservationStatus string

const (
	ReservationHeld     ReservationStatus = "held"
	ReservationSettled  ReservationStatus = "settled"
	ReservationReleased ReservationStatus = "released"
)

type Reservation struct {
	ID                   string
	CallerID             string
	PluginKeyID          string
	IdempotencyKey       string
	Model                string
	RequestTokenEstimate int64
	HeldMicroUSD         money.MicroUSD
	SettledMicroUSD      *money.MicroUSD
	Status               ReservationStatus
	RequestSummary       string
	SettlementSummary    string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	SettledAt            *time.Time
	ReleasedAt           *time.Time
}

type ReserveRequest struct {
	ReservationID        string
	CallerID             string
	PluginKeyID          string
	IdempotencyKey       string
	Model                string
	RequestTokenEstimate int64
	AmountMicroUSD       money.MicroUSD
	RequestSummary       string
}

func (s *Store) Reserve(ctx context.Context, request ReserveRequest) (Reservation, error) {
	if strings.TrimSpace(request.CallerID) == "" || strings.TrimSpace(request.PluginKeyID) == "" ||
		strings.TrimSpace(request.IdempotencyKey) == "" || request.RequestTokenEstimate <= 0 || request.AmountMicroUSD < 0 {
		return Reservation{}, fmt.Errorf("%w: caller, plugin key, idempotency key, positive token estimate, and non-negative amount are required", ErrInvalidArgument)
	}
	if request.ReservationID == "" {
		request.ReservationID = newID()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Reservation{}, err
	}
	defer tx.Rollback()
	if existing, err := getReservationByIdempotency(ctx, tx, request.CallerID, request.IdempotencyKey); err == nil {
		if existing.RequestTokenEstimate != request.RequestTokenEstimate ||
			existing.HeldMicroUSD != request.AmountMicroUSD ||
			existing.PluginKeyID != request.PluginKeyID {
			return Reservation{}, ErrIdempotencyConflict
		}
		return existing, nil
	} else if !errors.Is(err, ErrReservationNotFound) {
		return Reservation{}, err
	}

	// Fail-closed: caller and key must both be active.
	var callerEnabled int
	if err := tx.QueryRowContext(ctx, `SELECT enabled FROM callers WHERE id = ?`, request.CallerID).Scan(&callerEnabled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Reservation{}, ErrCallerNotFound
		}
		return Reservation{}, err
	}
	if callerEnabled != 1 {
		return Reservation{}, ErrCallerDisabled
	}
	var keyEnabled int
	var revoked, expires sql.NullInt64
	var allowedJSON, tokenLimitsJSON, unmatchedMode string
	var dailyQuota, weeklyQuota, monthlyQuota, maxConcurrent int64
	if err := tx.QueryRowContext(ctx, `SELECT enabled, revoked_at_unix_ms, expires_at_unix_ms, allowed_models_json, model_token_limits_json, unmatched_models_mode,
		daily_quota_micro_usd, weekly_quota_micro_usd, monthly_quota_micro_usd, max_concurrent_requests
		FROM plugin_keys WHERE id = ? AND caller_id = ?`,
		request.PluginKeyID, request.CallerID).Scan(&keyEnabled, &revoked, &expires, &allowedJSON, &tokenLimitsJSON, &unmatchedMode,
		&dailyQuota, &weeklyQuota, &monthlyQuota, &maxConcurrent); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Reservation{}, ErrPluginKeyNotFound
		}
		return Reservation{}, err
	}
	if revoked.Valid {
		return Reservation{}, ErrPluginKeyRevoked
	}
	if keyEnabled != 1 {
		return Reservation{}, ErrPluginKeyDisabled
	}
	if expires.Valid && expires.Int64 <= nowUnixMilli() {
		return Reservation{}, ErrPluginKeyExpired
	}
	allowed, err := unmarshalAllowedModels(allowedJSON)
	if err != nil {
		return Reservation{}, err
	}
	if !ModelAllowed(PluginKey{AllowedModels: allowed}, request.Model) {
		return Reservation{}, fmt.Errorf("%w: %s", ErrModelNotAllowed, request.Model)
	}
	tokenLimits, err := unmarshalModelTokenLimits(tokenLimitsJSON)
	if err != nil {
		return Reservation{}, err
	}

	now := nowUnixMilli()
	if err := enforceModelTokenLimits(ctx, tx, request.PluginKeyID, request.Model, request.RequestTokenEstimate, tokenLimits, unmatchedMode, now); err != nil {
		return Reservation{}, err
	}
	if maxConcurrent > 0 {
		var active int64
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM reservations
			WHERE plugin_key_id = ? AND status = 'held'`, request.PluginKeyID).Scan(&active); err != nil {
			return Reservation{}, fmt.Errorf("count active reservations: %w", err)
		}
		if active >= maxConcurrent {
			return Reservation{}, ErrConcurrentLimit
		}
	}
	for _, limit := range []struct {
		quota money.MicroUSD
		start int64
		err   error
	}{
		{money.MicroUSD(dailyQuota), utcDayStart(now), ErrDailyQuotaExceeded},
		{money.MicroUSD(weeklyQuota), utcWeekStart(now), ErrWeeklyQuotaExceeded},
		{money.MicroUSD(monthlyQuota), utcMonthStart(now), ErrMonthlyQuotaExceeded},
	} {
		if limit.quota == 0 {
			continue
		}
		used, err := reservedSpendSince(ctx, tx, request.PluginKeyID, limit.start)
		if err != nil {
			return Reservation{}, err
		}
		if used > limit.quota-request.AmountMicroUSD {
			return Reservation{}, limit.err
		}
	}
	// Key quota is the only spend limit. Caller records are retained for
	// ownership and historical attribution, but do not participate in accounting.
	// quota_micro_usd NULL or 0 means unlimited.
	result, err := tx.ExecContext(ctx, `UPDATE plugin_keys
		SET held_amount_micro_usd = held_amount_micro_usd + ?, updated_at_unix_ms = ?
		WHERE id = ? AND enabled = 1 AND revoked_at_unix_ms IS NULL
		AND (
			quota_micro_usd IS NULL
			OR quota_micro_usd = 0
			OR quota_micro_usd - settled_spend_micro_usd - held_amount_micro_usd >= ?
		)`,
		request.AmountMicroUSD, now, request.PluginKeyID, request.AmountMicroUSD)
	if err != nil {
		return Reservation{}, fmt.Errorf("hold key quota: %w", err)
	}
	if err := requireOneRow(result, ErrInsufficientQuota); err != nil {
		return Reservation{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO reservations(
		id, caller_id, plugin_key_id, idempotency_key, model, request_token_estimate, held_micro_usd, status,
		request_summary, created_at_unix_ms, updated_at_unix_ms
	) VALUES (?, ?, ?, ?, ?, ?, ?, 'held', ?, ?, ?)`,
		request.ReservationID, request.CallerID, request.PluginKeyID, request.IdempotencyKey,
		request.Model, request.RequestTokenEstimate, request.AmountMicroUSD, request.RequestSummary, now, now)
	if err != nil {
		return Reservation{}, fmt.Errorf("create reservation: %w", err)
	}
	if err := insertAudit(ctx, tx, request.CallerID, request.PluginKeyID, request.ReservationID, "quota_held", request.AmountMicroUSD, `{}`, now); err != nil {
		return Reservation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Reservation{}, err
	}
	return s.GetReservation(ctx, request.ReservationID)
}

// KeyUsageOverview contains self-service usage figures for one plugin key.
// Period amounts include current holds because they also consume usable quota.
type KeyUsageOverview struct {
	RequestCount       int64
	CostMicroUSD       money.MicroUSD
	InputTokens        int64
	OutputTokens       int64
	ActiveReservations int64
	DailyMicroUSD      money.MicroUSD
	WeeklyMicroUSD     money.MicroUSD
	MonthlyMicroUSD    money.MicroUSD
}

func (s *Store) GetKeyUsageOverview(ctx context.Context, keyID string, now time.Time) (KeyUsageOverview, error) {
	if strings.TrimSpace(keyID) == "" {
		return KeyUsageOverview{}, fmt.Errorf("%w: plugin key id is required", ErrInvalidArgument)
	}
	var overview KeyUsageOverview
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1), COALESCE(SUM(cost_micro_usd), 0),
		COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0)
		FROM usage_ledger WHERE plugin_key_id = ?`, keyID).Scan(
		&overview.RequestCount, &overview.CostMicroUSD, &overview.InputTokens, &overview.OutputTokens,
	); err != nil {
		return KeyUsageOverview{}, fmt.Errorf("summarize key usage: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM reservations
		WHERE plugin_key_id = ? AND status = 'held'`, keyID).Scan(&overview.ActiveReservations); err != nil {
		return KeyUsageOverview{}, fmt.Errorf("count active key reservations: %w", err)
	}
	var err error
	if overview.DailyMicroUSD, err = keyPeriodSpend(ctx, s.db, keyID, utcDayStart(now.UTC().UnixMilli())); err != nil {
		return KeyUsageOverview{}, err
	}
	if overview.WeeklyMicroUSD, err = keyPeriodSpend(ctx, s.db, keyID, utcWeekStart(now.UTC().UnixMilli())); err != nil {
		return KeyUsageOverview{}, err
	}
	if overview.MonthlyMicroUSD, err = keyPeriodSpend(ctx, s.db, keyID, utcMonthStart(now.UTC().UnixMilli())); err != nil {
		return KeyUsageOverview{}, err
	}
	return overview, nil
}

func reservedSpendSince(ctx context.Context, tx *sql.Tx, keyID string, startUnixMilli int64) (money.MicroUSD, error) {
	return keyPeriodSpend(ctx, tx, keyID, startUnixMilli)
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func keyPeriodSpend(ctx context.Context, db queryRower, keyID string, startUnixMilli int64) (money.MicroUSD, error) {
	var settled, held int64
	if err := db.QueryRowContext(ctx, `SELECT
		COALESCE((SELECT SUM(cost_micro_usd) FROM usage_ledger WHERE plugin_key_id = ? AND created_at_unix_ms >= ?), 0),
		COALESCE((SELECT SUM(held_micro_usd) FROM reservations WHERE plugin_key_id = ? AND status = 'held' AND created_at_unix_ms >= ?), 0)`,
		keyID, startUnixMilli, keyID, startUnixMilli).Scan(&settled, &held); err != nil {
		return 0, fmt.Errorf("sum period spend: %w", err)
	}
	return money.MicroUSD(settled + held), nil
}

func utcDayStart(nowUnixMilli int64) int64 {
	now := time.UnixMilli(nowUnixMilli).UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).UnixMilli()
}

func utcWeekStart(nowUnixMilli int64) int64 {
	now := time.UnixMilli(nowUnixMilli).UTC()
	daysSinceMonday := (int(now.Weekday()) + 6) % 7
	return time.Date(now.Year(), now.Month(), now.Day()-daysSinceMonday, 0, 0, 0, 0, time.UTC).UnixMilli()
}

func utcMonthStart(nowUnixMilli int64) int64 {
	now := time.UnixMilli(nowUnixMilli).UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).UnixMilli()
}

func validatePluginKeyLimits(total, daily, weekly, monthly money.MicroUSD, maxConcurrent int64) error {
	if total < 0 || daily < 0 || weekly < 0 || monthly < 0 || maxConcurrent < 0 {
		return fmt.Errorf("%w: key limits must not be negative", ErrInvalidArgument)
	}
	return nil
}

func derefQuota(quota *money.MicroUSD) money.MicroUSD {
	if quota == nil {
		return 0
	}
	return *quota
}

func (s *Store) GetReservation(ctx context.Context, reservationID string) (Reservation, error) {
	return scanReservation(s.db.QueryRowContext(ctx, reservationSelect+` WHERE id = ?`, reservationID))
}

func (s *Store) Release(ctx context.Context, reservationID string, reason string) (Reservation, error) {
	if strings.TrimSpace(reservationID) == "" {
		return Reservation{}, fmt.Errorf("%w: reservation id is required", ErrInvalidArgument)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Reservation{}, err
	}
	defer tx.Rollback()
	reservation, err := getReservation(ctx, tx, reservationID)
	if err != nil {
		return Reservation{}, err
	}
	if reservation.Status == ReservationReleased {
		return reservation, nil
	}
	if reservation.Status != ReservationHeld {
		return Reservation{}, ErrReservationFinalized
	}
	now := nowUnixMilli()
	result, err := tx.ExecContext(ctx, `UPDATE plugin_keys SET
		held_amount_micro_usd = CASE WHEN held_amount_micro_usd >= ? THEN held_amount_micro_usd - ? ELSE 0 END,
		updated_at_unix_ms = ?
		WHERE id = ?`,
		reservation.HeldMicroUSD, reservation.HeldMicroUSD, now, reservation.PluginKeyID)
	if err != nil {
		return Reservation{}, fmt.Errorf("release key hold: %w", err)
	}
	if err := requireOneRow(result, ErrPluginKeyNotFound); err != nil {
		return Reservation{}, err
	}
	summary := reason
	result, err = tx.ExecContext(ctx, `UPDATE reservations SET status='released', released_at_unix_ms=?,
		settlement_summary=?, updated_at_unix_ms=? WHERE id=? AND status='held'`, now, summary, now, reservationID)
	if err != nil {
		return Reservation{}, err
	}
	if err := requireOneRow(result, ErrReservationFinalized); err != nil {
		return Reservation{}, err
	}
	details := fmt.Sprintf(`{"reason":%q}`, reason)
	if err := insertAudit(ctx, tx, reservation.CallerID, reservation.PluginKeyID, reservation.ID, "quota_released", reservation.HeldMicroUSD, details, now); err != nil {
		return Reservation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Reservation{}, err
	}
	return s.GetReservation(ctx, reservationID)
}

// ReleaseStaleReservations releases abandoned holds left by a terminated host or
// plugin process so they cannot consume concurrency indefinitely.
func (s *Store) ReleaseStaleReservations(ctx context.Context, olderThan time.Time) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id FROM reservations
		WHERE status = 'held' AND updated_at_unix_ms < ?`, olderThan.UTC().UnixMilli())
	if err != nil {
		return 0, fmt.Errorf("list stale reservations: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return 0, fmt.Errorf("scan stale reservation: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	var released int64
	for _, id := range ids {
		reservation, err := getReservation(ctx, tx, id)
		if err != nil {
			return 0, err
		}
		now := nowUnixMilli()
		if _, err := tx.ExecContext(ctx, `UPDATE plugin_keys SET
			held_amount_micro_usd = CASE WHEN held_amount_micro_usd >= ? THEN held_amount_micro_usd - ? ELSE 0 END,
			updated_at_unix_ms = ? WHERE id = ?`, reservation.HeldMicroUSD, reservation.HeldMicroUSD, now, reservation.PluginKeyID); err != nil {
			return 0, fmt.Errorf("release stale key hold: %w", err)
		}
		result, err := tx.ExecContext(ctx, `UPDATE reservations SET status = 'released', released_at_unix_ms = ?,
			settlement_summary = 'stale_timeout', updated_at_unix_ms = ? WHERE id = ? AND status = 'held'`, now, now, id)
		if err != nil {
			return 0, fmt.Errorf("release stale reservation: %w", err)
		}
		changed, err := result.RowsAffected()
		if err != nil || changed != 1 {
			return 0, ErrReservationFinalized
		}
		if err := insertAudit(ctx, tx, reservation.CallerID, reservation.PluginKeyID, reservation.ID, "quota_released", reservation.HeldMicroUSD, `{"reason":"stale_timeout"}`, now); err != nil {
			return 0, err
		}
		released++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return released, nil
}

func (s *Store) TouchReservation(ctx context.Context, reservationID string) error {
	if strings.TrimSpace(reservationID) == "" {
		return fmt.Errorf("%w: reservation id is required", ErrInvalidArgument)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE reservations SET updated_at_unix_ms = ?
		WHERE id = ? AND status = 'held'`, nowUnixMilli(), reservationID)
	if err != nil {
		return fmt.Errorf("touch reservation: %w", err)
	}
	if changed, err := result.RowsAffected(); err != nil || changed != 1 {
		return ErrReservationFinalized
	}
	return nil
}

const reservationSelect = `SELECT id, caller_id, plugin_key_id, idempotency_key, model, request_token_estimate,
	held_micro_usd, settled_micro_usd, status, request_summary, settlement_summary,
	created_at_unix_ms, updated_at_unix_ms, settled_at_unix_ms, released_at_unix_ms FROM reservations`

func getReservation(ctx context.Context, tx *sql.Tx, reservationID string) (Reservation, error) {
	return scanReservation(tx.QueryRowContext(ctx, reservationSelect+` WHERE id = ?`, reservationID))
}

func getReservationByIdempotency(ctx context.Context, tx *sql.Tx, callerID, key string) (Reservation, error) {
	return scanReservation(tx.QueryRowContext(ctx, reservationSelect+` WHERE caller_id = ? AND idempotency_key = ?`, callerID, key))
}
