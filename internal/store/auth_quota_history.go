package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// AuthQuotaWindowObservation is one upstream quota window as seen by a sync
// (or seeded from older baselines/snapshots). StartUnixMS is the window start
// reported (or inferred) by upstream; ResetUnixMS is its scheduled end.
type AuthQuotaWindowObservation struct {
	Provider        string
	AuthID          string
	WindowID        string
	StartUnixMS     int64
	ResetUnixMS     int64
	DurationSeconds int64
	Label           string
	Used            *float64
	FirstUsed       *float64
	FirstSeenUnixMS int64 // 0 = unknown (seeded from snapshot)
	LastSeenUnixMS  int64
	Source          string
}

// UpsertAuthQuotaWindowObservation keeps first-seen data and refreshes the
// latest used value/seen time.
func (s *Store) UpsertAuthQuotaWindowObservation(ctx context.Context, o AuthQuotaWindowObservation) error {
	if o.Provider == "" || o.AuthID == "" || o.WindowID == "" || o.StartUnixMS <= 0 || o.ResetUnixMS <= o.StartUnixMS {
		return fmt.Errorf("%w: invalid quota window observation", ErrInvalidArgument)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO auth_quota_window_observations
		(provider, auth_id, window_id, start_unix_ms, reset_unix_ms, duration_seconds, label, used, first_used, first_seen_unix_ms, last_seen_unix_ms, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider, auth_id, window_id, start_unix_ms) DO UPDATE SET
			used = COALESCE(excluded.used, auth_quota_window_observations.used),
			label = CASE WHEN excluded.label <> '' THEN excluded.label ELSE auth_quota_window_observations.label END,
			last_seen_unix_ms = MAX(auth_quota_window_observations.last_seen_unix_ms, excluded.last_seen_unix_ms)`,
		o.Provider, o.AuthID, o.WindowID, o.StartUnixMS, o.ResetUnixMS, o.DurationSeconds, o.Label,
		nullFloat(o.Used), nullFloat(o.FirstUsed), o.FirstSeenUnixMS, o.LastSeenUnixMS, o.Source)
	if err != nil {
		return fmt.Errorf("upsert quota window observation: %w", err)
	}
	return nil
}

// InsertAuthQuotaWindowObservationIfMissing seeds without overwriting.
func (s *Store) InsertAuthQuotaWindowObservationIfMissing(ctx context.Context, o AuthQuotaWindowObservation) error {
	if o.Provider == "" || o.AuthID == "" || o.WindowID == "" || o.StartUnixMS <= 0 || o.ResetUnixMS <= o.StartUnixMS {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO auth_quota_window_observations
		(provider, auth_id, window_id, start_unix_ms, reset_unix_ms, duration_seconds, label, used, first_used, first_seen_unix_ms, last_seen_unix_ms, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.Provider, o.AuthID, o.WindowID, o.StartUnixMS, o.ResetUnixMS, o.DurationSeconds, o.Label,
		nullFloat(o.Used), nullFloat(o.FirstUsed), o.FirstSeenUnixMS, o.LastSeenUnixMS, o.Source)
	if err != nil {
		return fmt.Errorf("seed quota window observation: %w", err)
	}
	return nil
}

func (s *Store) ListAuthQuotaWindowObservations(ctx context.Context, provider, authID string) ([]AuthQuotaWindowObservation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider, auth_id, window_id, start_unix_ms, reset_unix_ms, duration_seconds, label,
		used, first_used, first_seen_unix_ms, last_seen_unix_ms, source
		FROM auth_quota_window_observations WHERE provider = ? AND auth_id = ? ORDER BY window_id, start_unix_ms`, provider, authID)
	if err != nil {
		return nil, fmt.Errorf("list quota window observations: %w", err)
	}
	defer rows.Close()
	var out []AuthQuotaWindowObservation
	for rows.Next() {
		var o AuthQuotaWindowObservation
		var used, firstUsed sql.NullFloat64
		if err := rows.Scan(&o.Provider, &o.AuthID, &o.WindowID, &o.StartUnixMS, &o.ResetUnixMS, &o.DurationSeconds, &o.Label,
			&used, &firstUsed, &o.FirstSeenUnixMS, &o.LastSeenUnixMS, &o.Source); err != nil {
			return nil, fmt.Errorf("scan quota window observation: %w", err)
		}
		if used.Valid {
			v := used.Float64
			o.Used = &v
		}
		if firstUsed.Valid {
			v := firstUsed.Float64
			o.FirstUsed = &v
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ListAuthQuotaWindowBaselines returns all stored baselines for one auth.
func (s *Store) ListAuthQuotaWindowBaselines(ctx context.Context, provider, authID string) ([]AuthQuotaWindowBaseline, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider, auth_id, window_id, cycle_key, baseline_used, created_at_unix_ms
		FROM auth_quota_window_baselines WHERE provider = ? AND auth_id = ?`, provider, authID)
	if err != nil {
		return nil, fmt.Errorf("list quota window baselines: %w", err)
	}
	defer rows.Close()
	var out []AuthQuotaWindowBaseline
	for rows.Next() {
		var b AuthQuotaWindowBaseline
		var created int64
		if err := rows.Scan(&b.Provider, &b.AuthID, &b.WindowID, &b.CycleKey, &b.BaselineUsed, &created); err != nil {
			return nil, fmt.Errorf("scan quota window baseline: %w", err)
		}
		b.CreatedAt = time.UnixMilli(created).UTC()
		out = append(out, b)
	}
	return out, rows.Err()
}

// AuthQuotaRequestBounds returns the first and last ledger request time in
// [from, to) for an auth identity (any model). ok=false when none.
func (s *Store) AuthQuotaRequestBounds(ctx context.Context, provider, authID, authIndex string, from, to time.Time) (first, last time.Time, ok bool, err error) {
	where, args, err := authQuotaUsageWhere(AuthQuotaUsageFilter{Provider: provider, AuthID: authID, AuthIndex: authIndex, From: from, To: to}, false)
	if err != nil {
		return time.Time{}, time.Time{}, false, err
	}
	var minMS, maxMS sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT MIN(created_at_unix_ms), MAX(created_at_unix_ms) FROM usage_ledger WHERE `+where, args...).Scan(&minMS, &maxMS); err != nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("auth quota request bounds: %w", err)
	}
	if !minMS.Valid {
		return time.Time{}, time.Time{}, false, nil
	}
	return time.UnixMilli(minMS.Int64).UTC(), time.UnixMilli(maxMS.Int64).UTC(), true, nil
}

// AuthQuotaResetCredit is one codex rate-limit reset credit.
type AuthQuotaResetCredit struct {
	CreditID        string
	ResetType       string
	Status          string
	GrantedAt       *time.Time
	ExpiresAt       *time.Time
	FirstSeenUnixMS int64
	LastSeenUnixMS  int64
	GoneAtUnixMS    *int64
}

func (s *Store) ListAuthQuotaResetCredits(ctx context.Context, provider, authID string) ([]AuthQuotaResetCredit, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT credit_id, reset_type, status, granted_at_unix_ms, expires_at_unix_ms,
		first_seen_unix_ms, last_seen_unix_ms, gone_at_unix_ms FROM auth_quota_reset_credits
		WHERE provider = ? AND auth_id = ? ORDER BY first_seen_unix_ms, credit_id`, provider, authID)
	if err != nil {
		return nil, fmt.Errorf("list reset credits: %w", err)
	}
	defer rows.Close()
	var out []AuthQuotaResetCredit
	for rows.Next() {
		var c AuthQuotaResetCredit
		var granted, expires, gone sql.NullInt64
		if err := rows.Scan(&c.CreditID, &c.ResetType, &c.Status, &granted, &expires, &c.FirstSeenUnixMS, &c.LastSeenUnixMS, &gone); err != nil {
			return nil, fmt.Errorf("scan reset credit: %w", err)
		}
		c.GrantedAt, c.ExpiresAt = msTime(granted), msTime(expires)
		if gone.Valid {
			v := gone.Int64
			c.GoneAtUnixMS = &v
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SyncAuthQuotaResetCredits upserts the credits seen now and marks previously
// present credits that are missing as gone at nowMS. It returns the credits
// that just disappeared.
func (s *Store) SyncAuthQuotaResetCredits(ctx context.Context, provider, authID string, credits []AuthQuotaResetCredit, nowMS int64) ([]AuthQuotaResetCredit, error) {
	before, err := s.ListAuthQuotaResetCredits(ctx, provider, authID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for _, c := range credits {
		if strings.TrimSpace(c.CreditID) == "" {
			continue
		}
		seen[c.CreditID] = true
		if _, err := tx.ExecContext(ctx, `INSERT INTO auth_quota_reset_credits
			(provider, auth_id, credit_id, reset_type, status, granted_at_unix_ms, expires_at_unix_ms, first_seen_unix_ms, last_seen_unix_ms, gone_at_unix_ms)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
			ON CONFLICT(provider, auth_id, credit_id) DO UPDATE SET reset_type = excluded.reset_type, status = excluded.status,
				granted_at_unix_ms = excluded.granted_at_unix_ms, expires_at_unix_ms = excluded.expires_at_unix_ms,
				last_seen_unix_ms = excluded.last_seen_unix_ms, gone_at_unix_ms = NULL`,
			provider, authID, c.CreditID, c.ResetType, c.Status, timeMS(c.GrantedAt), timeMS(c.ExpiresAt), nowMS, nowMS); err != nil {
			return nil, fmt.Errorf("upsert reset credit: %w", err)
		}
	}
	var gone []AuthQuotaResetCredit
	for _, c := range before {
		if seen[c.CreditID] || c.GoneAtUnixMS != nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE auth_quota_reset_credits SET gone_at_unix_ms = ? WHERE provider = ? AND auth_id = ? AND credit_id = ?`,
			nowMS, provider, authID, c.CreditID); err != nil {
			return nil, fmt.Errorf("mark reset credit gone: %w", err)
		}
		gone = append(gone, c)
	}
	return gone, tx.Commit()
}

// AuthQuotaResetEvent is an inferred manual reset between two syncs.
type AuthQuotaResetEvent struct {
	Kind         string
	AfterUnixMS  int64
	BeforeUnixMS int64
	CreditID     string
	Detail       string
}

func (s *Store) InsertAuthQuotaResetEvent(ctx context.Context, provider, authID string, e AuthQuotaResetEvent) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO auth_quota_reset_events
		(provider, auth_id, kind, after_unix_ms, before_unix_ms, credit_id, detail) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		provider, authID, e.Kind, e.AfterUnixMS, e.BeforeUnixMS, e.CreditID, e.Detail)
	if err != nil {
		return fmt.Errorf("insert reset event: %w", err)
	}
	return nil
}

func (s *Store) ListAuthQuotaResetEvents(ctx context.Context, provider, authID string) ([]AuthQuotaResetEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT kind, after_unix_ms, before_unix_ms, credit_id, detail FROM auth_quota_reset_events
		WHERE provider = ? AND auth_id = ? ORDER BY after_unix_ms`, provider, authID)
	if err != nil {
		return nil, fmt.Errorf("list reset events: %w", err)
	}
	defer rows.Close()
	var out []AuthQuotaResetEvent
	for rows.Next() {
		var e AuthQuotaResetEvent
		if err := rows.Scan(&e.Kind, &e.AfterUnixMS, &e.BeforeUnixMS, &e.CreditID, &e.Detail); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func timeMS(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().UnixMilli()
}

func msTime(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := time.UnixMilli(v.Int64).UTC()
	return &t
}
