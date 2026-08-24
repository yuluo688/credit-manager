package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yuluo688/credit-manager/internal/money"
)

// AuthIdentity is the host auth-file / credential selected for a request.
type AuthIdentity struct {
	AuthID    string
	AuthIndex string
	Name      string
	Label     string
	Provider  string
	Type      string
	Email     string
	Path      string
}

// AuthQuotaSnapshot is the sanitized quota state most recently fetched for an
// auth identity. SnapshotJSON must never contain auth material or other secrets.
type AuthQuotaSnapshot struct {
	Provider      string
	AuthID        string
	SnapshotJSON  string
	AuthModTime   *time.Time
	LastAttemptAt time.Time
	LastSuccessAt *time.Time
	LastErrorAt   *time.Time
	LastError     string
}

// AuthQuotaWindowBaseline is the upstream used amount first observed for a
// quota window cycle, before later local plugin usage.
type AuthQuotaWindowBaseline struct {
	Provider     string
	AuthID       string
	WindowID     string
	CycleKey     string
	BaselineUsed float64
	CreatedAt    time.Time
}

// AuthQuotaUsageFilter selects usage attributed to an auth identity during a
// local quota window. Models, when non-empty, limits the aggregate to exact names.
type AuthQuotaUsageFilter struct {
	Provider  string
	AuthID    string
	AuthIndex string
	From      time.Time
	To        time.Time
	Models    []string
}

// AuthQuotaUsage summarizes actual settled ledger usage for a quota window.
type AuthQuotaUsage struct {
	RequestCount        int64
	InputTokens         int64
	OutputTokens        int64
	ReasoningTokens     int64
	CachedTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	ActualCostMicroUSD  money.MicroUSD
}

// AuthQuotaUsageByModel is a settled usage aggregate for one normalized model.
type AuthQuotaUsageByModel struct {
	Model string
	AuthQuotaUsage
}

// GetAuthQuotaSnapshot returns the persisted quota state for an auth identity.
func (s *Store) GetAuthQuotaSnapshot(ctx context.Context, provider, authID string) (AuthQuotaSnapshot, error) {
	provider, authID = strings.TrimSpace(provider), strings.TrimSpace(authID)
	if provider == "" || authID == "" {
		return AuthQuotaSnapshot{}, fmt.Errorf("%w: provider and auth id are required", ErrInvalidArgument)
	}
	var snapshot AuthQuotaSnapshot
	var authModTime, lastSuccess, lastErrorAt sql.NullInt64
	var lastAttempt int64
	err := s.db.QueryRowContext(ctx, `SELECT provider, auth_id, snapshot_json, auth_mod_time_unix_ms,
		last_attempt_at_unix_ms, last_success_at_unix_ms, last_error_at_unix_ms, last_error
		FROM auth_quota_snapshots WHERE provider = ? AND auth_id = ?`, provider, authID).Scan(
		&snapshot.Provider, &snapshot.AuthID, &snapshot.SnapshotJSON, &authModTime,
		&lastAttempt, &lastSuccess, &lastErrorAt, &snapshot.LastError,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return AuthQuotaSnapshot{}, fmt.Errorf("%w: auth quota snapshot not found", ErrInvalidArgument)
	}
	if err != nil {
		return AuthQuotaSnapshot{}, fmt.Errorf("get auth quota snapshot: %w", err)
	}
	if authModTime.Valid {
		value := time.UnixMilli(authModTime.Int64).UTC()
		snapshot.AuthModTime = &value
	}
	if lastSuccess.Valid {
		value := time.UnixMilli(lastSuccess.Int64).UTC()
		snapshot.LastSuccessAt = &value
	}
	if lastErrorAt.Valid {
		value := time.UnixMilli(lastErrorAt.Int64).UTC()
		snapshot.LastErrorAt = &value
	}
	snapshot.LastAttemptAt = time.UnixMilli(lastAttempt).UTC()
	return snapshot, nil
}

// UpsertAuthQuotaSuccess replaces the sanitized quota snapshot and clears any
// prior fetch error for the provider/auth identity.
func (s *Store) UpsertAuthQuotaSuccess(ctx context.Context, provider, authID, snapshotJSON string, authModTime *time.Time) error {
	provider, authID = strings.TrimSpace(provider), strings.TrimSpace(authID)
	if provider == "" || authID == "" {
		return fmt.Errorf("%w: provider and auth id are required", ErrInvalidArgument)
	}
	snapshotJSON, err := sanitizeAuthQuotaSnapshotJSON(snapshotJSON)
	if err != nil {
		return err
	}
	now := nowUnixMilli()
	var modTime any
	if authModTime != nil {
		modTime = authModTime.UTC().UnixMilli()
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO auth_quota_snapshots(
		provider, auth_id, snapshot_json, auth_mod_time_unix_ms, last_attempt_at_unix_ms, last_success_at_unix_ms, last_error_at_unix_ms, last_error
	) VALUES (?, ?, ?, ?, ?, ?, NULL, '')
	ON CONFLICT(provider, auth_id) DO UPDATE SET
		snapshot_json = excluded.snapshot_json,
		auth_mod_time_unix_ms = excluded.auth_mod_time_unix_ms,
		last_attempt_at_unix_ms = excluded.last_attempt_at_unix_ms,
		last_success_at_unix_ms = excluded.last_success_at_unix_ms,
		last_error_at_unix_ms = NULL,
		last_error = ''`, provider, authID, snapshotJSON, modTime, now, now)
	if err != nil {
		return fmt.Errorf("upsert auth quota snapshot: %w", err)
	}
	return nil
}

// RecordAuthQuotaFailure records a failed fetch without replacing the last
// known-good sanitized snapshot.
func (s *Store) RecordAuthQuotaFailure(ctx context.Context, provider, authID string, authModTime *time.Time, fetchErr error) error {
	provider, authID = strings.TrimSpace(provider), strings.TrimSpace(authID)
	if provider == "" || authID == "" || fetchErr == nil {
		return fmt.Errorf("%w: provider, auth id, and fetch error are required", ErrInvalidArgument)
	}
	now := nowUnixMilli()
	var modTime any
	if authModTime != nil {
		modTime = authModTime.UTC().UnixMilli()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO auth_quota_snapshots(
		provider, auth_id, auth_mod_time_unix_ms, last_attempt_at_unix_ms, last_error_at_unix_ms, last_error
	) VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(provider, auth_id) DO UPDATE SET
		auth_mod_time_unix_ms = excluded.auth_mod_time_unix_ms,
		last_attempt_at_unix_ms = excluded.last_attempt_at_unix_ms,
		last_error_at_unix_ms = excluded.last_error_at_unix_ms,
		last_error = excluded.last_error`, provider, authID, modTime, now, now, fetchErr.Error())
	if err != nil {
		return fmt.Errorf("record auth quota failure: %w", err)
	}
	return nil
}

// GetAuthQuotaWindowBaseline returns the first-seen used amount for one window cycle.
func (s *Store) GetAuthQuotaWindowBaseline(ctx context.Context, provider, authID, windowID, cycleKey string) (AuthQuotaWindowBaseline, error) {
	provider, authID, windowID, cycleKey = strings.TrimSpace(provider), strings.TrimSpace(authID), strings.TrimSpace(windowID), strings.TrimSpace(cycleKey)
	if provider == "" || authID == "" || windowID == "" || cycleKey == "" {
		return AuthQuotaWindowBaseline{}, fmt.Errorf("%w: provider, auth id, window id, and cycle key are required", ErrInvalidArgument)
	}
	var row AuthQuotaWindowBaseline
	var created int64
	err := s.db.QueryRowContext(ctx, `SELECT provider, auth_id, window_id, cycle_key, baseline_used, created_at_unix_ms
		FROM auth_quota_window_baselines WHERE provider = ? AND auth_id = ? AND window_id = ? AND cycle_key = ?`,
		provider, authID, windowID, cycleKey).Scan(&row.Provider, &row.AuthID, &row.WindowID, &row.CycleKey, &row.BaselineUsed, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return AuthQuotaWindowBaseline{}, fmt.Errorf("%w: auth quota window baseline not found", ErrInvalidArgument)
	}
	if err != nil {
		return AuthQuotaWindowBaseline{}, fmt.Errorf("get auth quota window baseline: %w", err)
	}
	row.CreatedAt = time.UnixMilli(created).UTC()
	return row, nil
}

// UpsertAuthQuotaWindowBaseline records the first-seen used amount for a window
// cycle. Later observations keep the original baseline.
func (s *Store) UpsertAuthQuotaWindowBaseline(ctx context.Context, provider, authID, windowID, cycleKey string, baselineUsed float64) error {
	provider, authID, windowID, cycleKey = strings.TrimSpace(provider), strings.TrimSpace(authID), strings.TrimSpace(windowID), strings.TrimSpace(cycleKey)
	if provider == "" || authID == "" || windowID == "" || cycleKey == "" || math.IsNaN(baselineUsed) || math.IsInf(baselineUsed, 0) || baselineUsed < 0 {
		return fmt.Errorf("%w: provider, auth id, window id, cycle key, and a finite baseline are required", ErrInvalidArgument)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO auth_quota_window_baselines(
		provider, auth_id, window_id, cycle_key, baseline_used, created_at_unix_ms
	) VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(provider, auth_id, window_id, cycle_key) DO NOTHING`,
		provider, authID, windowID, cycleKey, baselineUsed, nowUnixMilli())
	if err != nil {
		return fmt.Errorf("upsert auth quota window baseline: %w", err)
	}
	return nil
}

// GetAuthQuotaUsageByModel returns settled local usage grouped by model for one
// auth identity. Exact auth IDs take precedence over legacy index-only rows.
func (s *Store) GetAuthQuotaUsageByModel(ctx context.Context, filter AuthQuotaUsageFilter) ([]AuthQuotaUsageByModel, error) {
	where, args, err := authQuotaUsageWhere(filter, false)
	if err != nil {
		return nil, err
	}
	query := `SELECT model, COUNT(1), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
		COALESCE(SUM(reasoning_tokens), 0), COALESCE(SUM(cached_tokens), 0), COALESCE(SUM(cache_read_tokens), 0),
		COALESCE(SUM(cache_creation_tokens), 0), COALESCE(SUM(cost_micro_usd), 0)
		FROM usage_ledger WHERE ` + where + ` GROUP BY model`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get auth quota usage by model: %w", err)
	}
	defer rows.Close()
	var result []AuthQuotaUsageByModel
	for rows.Next() {
		var item AuthQuotaUsageByModel
		if err := rows.Scan(&item.Model, &item.RequestCount, &item.InputTokens, &item.OutputTokens,
			&item.ReasoningTokens, &item.CachedTokens, &item.CacheReadTokens, &item.CacheCreationTokens,
			&item.ActualCostMicroUSD); err != nil {
			return nil, fmt.Errorf("scan auth quota usage by model: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate auth quota usage by model: %w", err)
	}
	return result, nil
}

// GetAuthQuotaUsage aggregates an auth identity's actual settled usage. Rows
// with an exact auth ID always match; legacy rows without one match AuthIndex.
func (s *Store) GetAuthQuotaUsage(ctx context.Context, filter AuthQuotaUsageFilter) (AuthQuotaUsage, error) {
	where, args, err := authQuotaUsageWhere(filter, true)
	if err != nil {
		return AuthQuotaUsage{}, err
	}
	query := `SELECT COUNT(1), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
		COALESCE(SUM(reasoning_tokens), 0), COALESCE(SUM(cached_tokens), 0), COALESCE(SUM(cache_read_tokens), 0),
		COALESCE(SUM(cache_creation_tokens), 0), COALESCE(SUM(cost_micro_usd), 0)
		FROM usage_ledger WHERE ` + where
	var usage AuthQuotaUsage
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&usage.RequestCount, &usage.InputTokens, &usage.OutputTokens, &usage.ReasoningTokens,
		&usage.CachedTokens, &usage.CacheReadTokens, &usage.CacheCreationTokens, &usage.ActualCostMicroUSD,
	); err != nil {
		return AuthQuotaUsage{}, fmt.Errorf("get auth quota usage: %w", err)
	}
	return usage, nil
}

func authQuotaUsageWhere(filter AuthQuotaUsageFilter, includeModels bool) (string, []any, error) {
	filter.Provider, filter.AuthID, filter.AuthIndex = strings.TrimSpace(filter.Provider), strings.TrimSpace(filter.AuthID), strings.TrimSpace(filter.AuthIndex)
	if filter.Provider == "" || filter.AuthID == "" || filter.From.IsZero() || filter.To.IsZero() || !filter.To.After(filter.From) {
		return "", nil, fmt.Errorf("%w: provider, auth id, and a non-empty time window are required", ErrInvalidArgument)
	}
	providers := authQuotaProviderAliases(filter.Provider)
	placeholders := strings.TrimRight(strings.Repeat("?,", len(providers)), ",")
	conditions := []string{"LOWER(TRIM(COALESCE(auth_provider, ''))) IN (" + placeholders + ")", "created_at_unix_ms >= ?", "created_at_unix_ms < ?"}
	args := make([]any, 0, len(providers)+6)
	for _, provider := range providers {
		args = append(args, provider)
	}
	args = append(args, filter.From.UTC().UnixMilli(), filter.To.UTC().UnixMilli())
	if filter.AuthIndex == "" {
		conditions = append(conditions, "auth_id = ?")
		args = append(args, filter.AuthID)
	} else {
		conditions = append(conditions, "(auth_id = ? OR (COALESCE(auth_id, '') = '' AND auth_index = ?))")
		args = append(args, filter.AuthID, filter.AuthIndex)
	}
	if includeModels {
		models := uniqueTrimmedStrings(filter.Models)
		if len(models) > 0 {
			modelPlaceholders := strings.TrimRight(strings.Repeat("?,", len(models)), ",")
			conditions = append(conditions, "model IN ("+modelPlaceholders+")")
			for _, model := range models {
				args = append(args, model)
			}
		}
	}
	return strings.Join(conditions, " AND "), args, nil
}

func authQuotaProviderAliases(provider string) []string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	aliases := map[string][]string{
		"codex":       {"codex", "openai", "chatgpt"},
		"openai":      {"codex", "openai", "chatgpt"},
		"chatgpt":     {"codex", "openai", "chatgpt"},
		"claude":      {"claude", "anthropic"},
		"anthropic":   {"claude", "anthropic"},
		"antigravity": {"antigravity", "google", "gemini"},
		"google":      {"antigravity", "google", "gemini"},
		"gemini":      {"antigravity", "google", "gemini"},
		"kimi":        {"kimi", "moonshot"},
		"moonshot":    {"kimi", "moonshot"},
		"xai":         {"xai", "grok"},
		"grok":        {"xai", "grok"},
	}
	if mapped := aliases[provider]; len(mapped) > 0 {
		return mapped
	}
	return []string{provider}
}

func sanitizeAuthQuotaSnapshotJSON(raw string) (string, error) {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return "", fmt.Errorf("%w: snapshot json must be valid json", ErrInvalidArgument)
	}
	sanitizeAuthQuotaJSONValue(value)
	sanitized, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("sanitize auth quota snapshot: %w", err)
	}
	return string(sanitized), nil
}

func sanitizeAuthQuotaJSONValue(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if isSensitiveAuthQuotaJSONKey(key) {
				delete(typed, key)
				continue
			}
			sanitizeAuthQuotaJSONValue(nested)
		}
	case []any:
		for _, nested := range typed {
			sanitizeAuthQuotaJSONValue(nested)
		}
	}
}

func isSensitiveAuthQuotaJSONKey(key string) bool {
	key = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(key), "_", ""), "-", ""))
	return strings.Contains(key, "token") || strings.Contains(key, "secret") || strings.Contains(key, "password") ||
		strings.Contains(key, "credential") || strings.Contains(key, "apikey") || key == "key" || strings.Contains(key, "authorization")
}

func uniqueTrimmedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (a AuthIdentity) Empty() bool {
	return strings.TrimSpace(a.AuthID) == "" &&
		strings.TrimSpace(a.AuthIndex) == "" &&
		strings.TrimSpace(a.Name) == "" &&
		strings.TrimSpace(a.Label) == "" &&
		strings.TrimSpace(a.Email) == "" &&
		strings.TrimSpace(a.Path) == ""
}
