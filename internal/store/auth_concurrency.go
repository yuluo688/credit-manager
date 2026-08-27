package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// AuthConcurrencyLimit is the max in-flight requests allowed for one auth identity.
// Zero means unlimited.
type AuthConcurrencyLimit struct {
	Provider              string
	AuthID                string
	MaxConcurrentRequests int64
}

func authConcurrencyKey(provider, authID string) (string, string, error) {
	provider, authID = strings.TrimSpace(provider), strings.TrimSpace(authID)
	if provider == "" || authID == "" {
		return "", "", fmt.Errorf("%w: provider and auth id are required", ErrInvalidArgument)
	}
	return provider, authID, nil
}

// GetAuthConcurrencyLimit returns the stored cap for one auth identity.
// Missing rows are unlimited (0).
func (s *Store) GetAuthConcurrencyLimit(ctx context.Context, provider, authID string) (int64, error) {
	provider, authID, err := authConcurrencyKey(provider, authID)
	if err != nil {
		return 0, err
	}
	var limit int64
	err = s.db.QueryRowContext(ctx, `SELECT max_concurrent_requests FROM auth_concurrency_limits WHERE provider = ? AND auth_id = ?`,
		provider, authID).Scan(&limit)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get auth concurrency limit: %w", err)
	}
	return limit, nil
}

// UpsertAuthConcurrencyLimit stores the cap for one auth identity. Zero is unlimited.
func (s *Store) UpsertAuthConcurrencyLimit(ctx context.Context, provider, authID string, maxConcurrent int64) error {
	provider, authID, err := authConcurrencyKey(provider, authID)
	if err != nil {
		return err
	}
	if maxConcurrent < 0 {
		return fmt.Errorf("%w: max concurrent requests must not be negative", ErrInvalidArgument)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO auth_concurrency_limits(
		provider, auth_id, max_concurrent_requests, updated_at_unix_ms
	) VALUES (?, ?, ?, ?)
	ON CONFLICT(provider, auth_id) DO UPDATE SET
		max_concurrent_requests = excluded.max_concurrent_requests,
		updated_at_unix_ms = excluded.updated_at_unix_ms`,
		provider, authID, maxConcurrent, nowUnixMilli())
	if err != nil {
		return fmt.Errorf("upsert auth concurrency limit: %w", err)
	}
	return nil
}

// UpsertAuthConcurrencyLimits stores caps for many auth identities in one transaction.
func (s *Store) UpsertAuthConcurrencyLimits(ctx context.Context, limits []AuthConcurrencyLimit) error {
	if len(limits) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin auth concurrency batch: %w", err)
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO auth_concurrency_limits(
		provider, auth_id, max_concurrent_requests, updated_at_unix_ms
	) VALUES (?, ?, ?, ?)
	ON CONFLICT(provider, auth_id) DO UPDATE SET
		max_concurrent_requests = excluded.max_concurrent_requests,
		updated_at_unix_ms = excluded.updated_at_unix_ms`)
	if err != nil {
		return fmt.Errorf("prepare auth concurrency batch: %w", err)
	}
	defer stmt.Close()
	now := nowUnixMilli()
	for _, item := range limits {
		provider, authID, err := authConcurrencyKey(item.Provider, item.AuthID)
		if err != nil {
			return err
		}
		if item.MaxConcurrentRequests < 0 {
			return fmt.Errorf("%w: max concurrent requests must not be negative", ErrInvalidArgument)
		}
		if _, err := stmt.ExecContext(ctx, provider, authID, item.MaxConcurrentRequests, now); err != nil {
			return fmt.Errorf("upsert auth concurrency limit: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit auth concurrency batch: %w", err)
	}
	return nil
}

// ListAuthConcurrencyLimits returns every stored cap keyed as provider + "\x00" + auth_id.
func (s *Store) ListAuthConcurrencyLimits(ctx context.Context) (map[string]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider, auth_id, max_concurrent_requests FROM auth_concurrency_limits`)
	if err != nil {
		return nil, fmt.Errorf("list auth concurrency limits: %w", err)
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var item AuthConcurrencyLimit
		if err := rows.Scan(&item.Provider, &item.AuthID, &item.MaxConcurrentRequests); err != nil {
			return nil, fmt.Errorf("scan auth concurrency limit: %w", err)
		}
		out[item.Provider+"\x00"+item.AuthID] = item.MaxConcurrentRequests
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate auth concurrency limits: %w", err)
	}
	return out, nil
}
