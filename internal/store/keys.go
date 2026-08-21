package store

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yuluo688/credit-manager/internal/money"
)

type PluginKey struct {
	ID                    string
	CallerID              string
	Kid                   string
	KeyHash               []byte
	EncryptedKeyMaterial  []byte
	PepperID              string
	Fingerprint           string
	Label                 string
	Principal             string
	CallerScope           string
	Enabled               bool
	QuotaMicroUSD         *money.MicroUSD // 0 or nil = unlimited
	DailyQuotaMicroUSD    money.MicroUSD  // 0 = unlimited, UTC calendar day
	WeeklyQuotaMicroUSD   money.MicroUSD  // 0 = unlimited, UTC calendar week starting Monday
	MonthlyQuotaMicroUSD  money.MicroUSD  // 0 = unlimited, UTC calendar month
	MaxConcurrentRequests int64           // 0 = unlimited
	SettledSpendMicroUSD  money.MicroUSD
	HeldAmountMicroUSD    money.MicroUSD
	AllowedModels         []string // empty = all models
	RevokedAt             *time.Time
	ExpiresAt             *time.Time
	LastUsedAt            *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// UnlimitedQuota reports whether the key has no spending cap (nil or 0).
func (k PluginKey) UnlimitedQuota() bool {
	return k.QuotaMicroUSD == nil || *k.QuotaMicroUSD == 0
}

func (k PluginKey) RemainingMicroUSD() money.MicroUSD {
	if k.UnlimitedQuota() {
		return 0
	}
	return money.MicroUSD(int64(*k.QuotaMicroUSD) - int64(k.SettledSpendMicroUSD) - int64(k.HeldAmountMicroUSD))
}

type PluginKeySpec struct {
	ID                    string
	CallerID              string
	Kid                   string
	KeyHash               []byte
	EncryptedKeyMaterial  []byte
	PepperID              string
	Fingerprint           string
	Label                 string
	Principal             string
	CallerScope           string
	Enabled               bool
	ExpiresAt             *time.Time
	QuotaMicroUSD         money.MicroUSD
	DailyQuotaMicroUSD    money.MicroUSD
	WeeklyQuotaMicroUSD   money.MicroUSD
	MonthlyQuotaMicroUSD  money.MicroUSD
	MaxConcurrentRequests int64
	AllowedModels         []string
}

// PluginKeyPolicyUpdate patches mutable admin fields on a key.
type PluginKeyPolicyUpdate struct {
	ID                    string
	Label                 *string
	Enabled               *bool
	QuotaMicroUSD         *money.MicroUSD
	DailyQuotaMicroUSD    *money.MicroUSD
	WeeklyQuotaMicroUSD   *money.MicroUSD
	MonthlyQuotaMicroUSD  *money.MicroUSD
	MaxConcurrentRequests *int64
	AllowedModels         *[]string
	ExpiresAt             *time.Time
	ClearExpiresAt        bool
}

func (s *Store) CreatePluginKey(ctx context.Context, spec PluginKeySpec) (PluginKey, error) {
	if strings.TrimSpace(spec.CallerID) == "" || strings.TrimSpace(spec.Kid) == "" ||
		len(spec.KeyHash) < 16 || strings.TrimSpace(spec.PepperID) == "" ||
		strings.TrimSpace(spec.Principal) == "" || strings.TrimSpace(spec.CallerScope) == "" {
		return PluginKey{}, fmt.Errorf("%w: plugin key fields are incomplete", ErrInvalidArgument)
	}
	if spec.ID == "" {
		spec.ID = newID()
	}
	if strings.TrimSpace(spec.Fingerprint) == "" {
		return PluginKey{}, fmt.Errorf("%w: fingerprint is required", ErrInvalidArgument)
	}
	if err := validatePluginKeyLimits(spec.QuotaMicroUSD, spec.DailyQuotaMicroUSD, spec.WeeklyQuotaMicroUSD, spec.MonthlyQuotaMicroUSD, spec.MaxConcurrentRequests); err != nil {
		return PluginKey{}, err
	}
	modelsJSON, err := marshalAllowedModels(spec.AllowedModels)
	if err != nil {
		return PluginKey{}, err
	}
	now := nowUnixMilli()
	var expires any
	if spec.ExpiresAt != nil {
		expires = spec.ExpiresAt.UTC().UnixMilli()
	}
	quota := int64(spec.QuotaMicroUSD)
	_, err = s.db.ExecContext(ctx, `INSERT INTO plugin_keys(
		id, caller_id, kid, key_hash, encrypted_key_material, pepper_id, fingerprint, label, principal, caller_scope,
		enabled, expires_at_unix_ms, created_at_unix_ms, updated_at_unix_ms,
		quota_micro_usd, daily_quota_micro_usd, weekly_quota_micro_usd, monthly_quota_micro_usd, max_concurrent_requests,
		settled_spend_micro_usd, held_amount_micro_usd, allowed_models_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?)`,
		spec.ID, spec.CallerID, spec.Kid, append([]byte(nil), spec.KeyHash...), append([]byte(nil), spec.EncryptedKeyMaterial...),
		spec.PepperID, spec.Fingerprint, spec.Label, spec.Principal, spec.CallerScope,
		boolInt(spec.Enabled), expires, now, now, quota, spec.DailyQuotaMicroUSD, spec.WeeklyQuotaMicroUSD, spec.MonthlyQuotaMicroUSD, spec.MaxConcurrentRequests, modelsJSON)
	if err != nil {
		return PluginKey{}, fmt.Errorf("create plugin key: %w", err)
	}
	return s.GetPluginKey(ctx, spec.ID)
}

func (s *Store) RotatePluginKey(ctx context.Context, oldKeyID string, spec PluginKeySpec) (PluginKey, error) {
	if strings.TrimSpace(oldKeyID) == "" || strings.TrimSpace(spec.CallerID) == "" || strings.TrimSpace(spec.Kid) == "" ||
		len(spec.KeyHash) < 16 || strings.TrimSpace(spec.PepperID) == "" ||
		strings.TrimSpace(spec.Principal) == "" || strings.TrimSpace(spec.CallerScope) == "" {
		return PluginKey{}, fmt.Errorf("%w: plugin key fields are incomplete", ErrInvalidArgument)
	}
	if spec.ID == "" {
		spec.ID = newID()
	}
	if strings.TrimSpace(spec.Fingerprint) == "" {
		return PluginKey{}, fmt.Errorf("%w: fingerprint is required", ErrInvalidArgument)
	}
	if err := validatePluginKeyLimits(spec.QuotaMicroUSD, spec.DailyQuotaMicroUSD, spec.WeeklyQuotaMicroUSD, spec.MonthlyQuotaMicroUSD, spec.MaxConcurrentRequests); err != nil {
		return PluginKey{}, err
	}
	modelsJSON, err := marshalAllowedModels(spec.AllowedModels)
	if err != nil {
		return PluginKey{}, err
	}
	var expires any
	if spec.ExpiresAt != nil {
		expires = spec.ExpiresAt.UTC().UnixMilli()
	}
	now := nowUnixMilli()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PluginKey{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO plugin_keys(
		id, caller_id, kid, key_hash, encrypted_key_material, pepper_id, fingerprint, label, principal, caller_scope,
		enabled, expires_at_unix_ms, created_at_unix_ms, updated_at_unix_ms,
		quota_micro_usd, daily_quota_micro_usd, weekly_quota_micro_usd, monthly_quota_micro_usd, max_concurrent_requests,
		settled_spend_micro_usd, held_amount_micro_usd, allowed_models_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?)`,
		spec.ID, spec.CallerID, spec.Kid, append([]byte(nil), spec.KeyHash...), append([]byte(nil), spec.EncryptedKeyMaterial...),
		spec.PepperID, spec.Fingerprint, spec.Label, spec.Principal, spec.CallerScope,
		boolInt(spec.Enabled), expires, now, now, int64(spec.QuotaMicroUSD), spec.DailyQuotaMicroUSD, spec.WeeklyQuotaMicroUSD, spec.MonthlyQuotaMicroUSD, spec.MaxConcurrentRequests, modelsJSON)
	if err != nil {
		return PluginKey{}, fmt.Errorf("create replacement plugin key: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE plugin_keys
		SET enabled = 0, revoked_at_unix_ms = ?, updated_at_unix_ms = ?
		WHERE id = ? AND revoked_at_unix_ms IS NULL`, now, now, oldKeyID)
	if err != nil {
		return PluginKey{}, fmt.Errorf("revoke replaced plugin key: %w", err)
	}
	if err := requireOneRow(result, ErrPluginKeyNotFound); err != nil {
		return PluginKey{}, err
	}
	if err := tx.Commit(); err != nil {
		return PluginKey{}, err
	}
	return s.GetPluginKey(ctx, spec.ID)
}

func (s *Store) GetPluginKey(ctx context.Context, id string) (PluginKey, error) {
	return scanPluginKey(s.db.QueryRowContext(ctx, pluginKeySelect+` WHERE id = ?`, id))
}

func (s *Store) GetPluginKeyByKid(ctx context.Context, kid string) (PluginKey, error) {
	return scanPluginKey(s.db.QueryRowContext(ctx, pluginKeySelect+` WHERE kid = ?`, kid))
}

func (s *Store) GetPluginKeyByPrincipal(ctx context.Context, principal string) (PluginKey, error) {
	return scanPluginKey(s.db.QueryRowContext(ctx, pluginKeySelect+` WHERE principal = ?`, principal))
}

func (s *Store) GetPluginKeyByCallerScope(ctx context.Context, callerScope string) (PluginKey, error) {
	return scanPluginKey(s.db.QueryRowContext(ctx, pluginKeySelect+` WHERE caller_scope = ?`, callerScope))
}

func (s *Store) ListPluginKeysByCaller(ctx context.Context, callerID string, limit int) ([]PluginKey, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, pluginKeySelect+` WHERE caller_id = ? ORDER BY created_at_unix_ms DESC LIMIT ?`,
		callerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PluginKey
	for rows.Next() {
		key, err := scanPluginKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, rows.Err()
}

func (s *Store) ListPluginKeys(ctx context.Context, limit int) ([]PluginKey, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, pluginKeySelect+` ORDER BY created_at_unix_ms DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PluginKey
	for rows.Next() {
		key, err := scanPluginKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, key)
	}
	return out, rows.Err()
}

func (s *Store) UpdatePluginKeyPolicy(ctx context.Context, update PluginKeyPolicyUpdate) (PluginKey, error) {
	if strings.TrimSpace(update.ID) == "" {
		return PluginKey{}, fmt.Errorf("%w: key id is required", ErrInvalidArgument)
	}
	key, err := s.GetPluginKey(ctx, update.ID)
	if err != nil {
		return PluginKey{}, err
	}
	if update.Label != nil {
		key.Label = *update.Label
	}
	if update.Enabled != nil {
		key.Enabled = *update.Enabled
	}
	if update.QuotaMicroUSD != nil {
		if *update.QuotaMicroUSD < 0 {
			return PluginKey{}, fmt.Errorf("%w: key quota must not be negative", ErrInvalidArgument)
		}
		q := *update.QuotaMicroUSD
		key.QuotaMicroUSD = &q
	}
	if update.DailyQuotaMicroUSD != nil {
		key.DailyQuotaMicroUSD = *update.DailyQuotaMicroUSD
	}
	if update.WeeklyQuotaMicroUSD != nil {
		key.WeeklyQuotaMicroUSD = *update.WeeklyQuotaMicroUSD
	}
	if update.MonthlyQuotaMicroUSD != nil {
		key.MonthlyQuotaMicroUSD = *update.MonthlyQuotaMicroUSD
	}
	if update.MaxConcurrentRequests != nil {
		key.MaxConcurrentRequests = *update.MaxConcurrentRequests
	}
	if err := validatePluginKeyLimits(derefQuota(key.QuotaMicroUSD), key.DailyQuotaMicroUSD, key.WeeklyQuotaMicroUSD, key.MonthlyQuotaMicroUSD, key.MaxConcurrentRequests); err != nil {
		return PluginKey{}, err
	}
	if update.AllowedModels != nil {
		key.AllowedModels = append([]string(nil), (*update.AllowedModels)...)
	}
	if update.ClearExpiresAt {
		key.ExpiresAt = nil
	} else if update.ExpiresAt != nil {
		exp := update.ExpiresAt.UTC()
		key.ExpiresAt = &exp
	}
	modelsJSON, err := marshalAllowedModels(key.AllowedModels)
	if err != nil {
		return PluginKey{}, err
	}
	if key.QuotaMicroUSD == nil {
		zero := money.MicroUSD(0)
		key.QuotaMicroUSD = &zero
	}
	quota := int64(*key.QuotaMicroUSD)
	var expires any
	if key.ExpiresAt != nil {
		expires = key.ExpiresAt.UTC().UnixMilli()
	}
	now := nowUnixMilli()
	result, err := s.db.ExecContext(ctx, `UPDATE plugin_keys SET
		label = ?, enabled = ?, quota_micro_usd = ?, daily_quota_micro_usd = ?, weekly_quota_micro_usd = ?,
		monthly_quota_micro_usd = ?, max_concurrent_requests = ?, allowed_models_json = ?,
		expires_at_unix_ms = ?, updated_at_unix_ms = ?
		WHERE id = ?`,
		key.Label, boolInt(key.Enabled), quota, key.DailyQuotaMicroUSD, key.WeeklyQuotaMicroUSD,
		key.MonthlyQuotaMicroUSD, key.MaxConcurrentRequests, modelsJSON, expires, now, update.ID)
	if err != nil {
		return PluginKey{}, fmt.Errorf("update plugin key policy: %w", err)
	}
	if err := requireOneRow(result, ErrPluginKeyNotFound); err != nil {
		return PluginKey{}, err
	}
	return s.GetPluginKey(ctx, update.ID)
}

func (s *Store) RevokePluginKey(ctx context.Context, id string) error {
	now := nowUnixMilli()
	result, err := s.db.ExecContext(ctx, `UPDATE plugin_keys
		SET enabled = 0, revoked_at_unix_ms = ?, updated_at_unix_ms = ?
		WHERE id = ? AND revoked_at_unix_ms IS NULL`, now, now, id)
	if err != nil {
		return fmt.Errorf("revoke plugin key: %w", err)
	}
	return requireOneRow(result, ErrPluginKeyNotFound)
}

// DeletePluginKey revokes a key while retaining its records as immutable
// accounting history. The compatibility caller is intentionally retained.
func (s *Store) DeletePluginKey(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: plugin key id is required", ErrInvalidArgument)
	}
	now := nowUnixMilli()
	result, err := s.db.ExecContext(ctx, `UPDATE plugin_keys
		SET enabled = 0, revoked_at_unix_ms = COALESCE(revoked_at_unix_ms, ?), updated_at_unix_ms = ?
		WHERE id = ?`, now, now, id)
	if err != nil {
		return fmt.Errorf("delete plugin key: %w", err)
	}
	return requireOneRow(result, ErrPluginKeyNotFound)
}

func (s *Store) TouchPluginKeyUsed(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE plugin_keys SET last_used_at_unix_ms = ?, updated_at_unix_ms = ?
		WHERE id = ?`, nowUnixMilli(), nowUnixMilli(), id)
	return err
}

// EnsurePluginKeyUsable validates enabled/revoked/expiry against a loaded key.
func EnsurePluginKeyUsable(key PluginKey, now time.Time) error {
	if key.RevokedAt != nil {
		return ErrPluginKeyRevoked
	}
	if !key.Enabled {
		return ErrPluginKeyDisabled
	}
	if key.ExpiresAt != nil && !key.ExpiresAt.After(now) {
		return ErrPluginKeyExpired
	}
	return nil
}

// ModelAllowed reports whether model may be used with this key.
// Empty allowlist means all models are allowed.
func ModelAllowed(key PluginKey, model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	if len(key.AllowedModels) == 0 {
		return true
	}
	for _, pattern := range key.AllowedModels {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if pattern == model {
			return true
		}
		if ok, err := filepath.Match(pattern, model); err == nil && ok {
			return true
		}
	}
	return false
}

const pluginKeySelect = `SELECT id, caller_id, kid, key_hash, encrypted_key_material, pepper_id, fingerprint, label, principal, caller_scope,
	enabled, revoked_at_unix_ms, expires_at_unix_ms, last_used_at_unix_ms, created_at_unix_ms, updated_at_unix_ms,
	quota_micro_usd, daily_quota_micro_usd, weekly_quota_micro_usd, monthly_quota_micro_usd, max_concurrent_requests,
	settled_spend_micro_usd, held_amount_micro_usd, allowed_models_json
	FROM plugin_keys`
