package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yuluo688/credit-manager/internal/money"
)

var (
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrCallerNotFound       = errors.New("caller not found")
	ErrCallerDisabled       = errors.New("caller is disabled")
	ErrInsufficientQuota    = errors.New("insufficient quota")
	ErrReservationNotFound  = errors.New("reservation not found")
	ErrReservationFinalized = errors.New("reservation is already finalized")
	ErrIdempotencyConflict  = errors.New("idempotency key is already used for a different request")
	ErrPricingRuleNotFound  = errors.New("pricing rule not found")
	ErrPluginKeyNotFound    = errors.New("plugin key not found")
	ErrPluginKeyDisabled    = errors.New("plugin key is disabled")
	ErrPluginKeyRevoked     = errors.New("plugin key is revoked")
	ErrPluginKeyExpired     = errors.New("plugin key is expired")
	ErrModelNotAllowed      = errors.New("model is not allowed for this key")
	ErrDailyQuotaExceeded   = errors.New("daily quota exceeded")
	ErrWeeklyQuotaExceeded  = errors.New("weekly quota exceeded")
	ErrMonthlyQuotaExceeded = errors.New("monthly quota exceeded")
	ErrConcurrentLimit      = errors.New("maximum concurrent requests reached")
)

// Store owns the SQLite connection pool. MaxOpenConns is intentionally one.
type Store struct {
	db *sql.DB
}

type OpenOptions struct {
	BusyTimeout time.Duration
}

// FileLock describes the required host-owned cross-process writer lock.
type FileLock interface {
	Lock(ctx context.Context, path string) (unlock func() error, err error)
}

func LockPath(databasePath string) string {
	return filepath.Clean(databasePath) + ".lock"
}

func Open(ctx context.Context, databasePath string, options OpenOptions) (*Store, error) {
	if strings.TrimSpace(databasePath) == "" {
		return nil, fmt.Errorf("%w: database path is required", ErrInvalidArgument)
	}
	if options.BusyTimeout <= 0 || options.BusyTimeout > 5*time.Minute {
		return nil, fmt.Errorf("%w: busy timeout must be greater than zero and at most 5 minutes", ErrInvalidArgument)
	}

	dsn := sqliteDSN(databasePath, options.BusyTimeout)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &Store{db: db}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := Migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func OpenLocked(ctx context.Context, databasePath string, options OpenOptions, locker FileLock) (*Store, error) {
	if locker == nil {
		return nil, fmt.Errorf("%w: file locker is required", ErrInvalidArgument)
	}
	unlock, err := locker.Lock(ctx, LockPath(databasePath))
	if err != nil {
		return nil, fmt.Errorf("acquire database lock: %w", err)
	}
	store, err := Open(ctx, databasePath, options)
	if err != nil {
		_ = unlock()
		return nil, err
	}
	storeUnlockers.Store(store, unlock)
	return store, nil
}

var storeUnlockers = newUnlockRegistry()

type unlockRegistry struct {
	items map[*Store]func() error
	gate  chan struct{}
}

func newUnlockRegistry() *unlockRegistry {
	r := &unlockRegistry{items: make(map[*Store]func() error), gate: make(chan struct{}, 1)}
	r.gate <- struct{}{}
	return r
}

func (r *unlockRegistry) Store(store *Store, unlock func() error) {
	<-r.gate
	r.items[store] = unlock
	r.gate <- struct{}{}
}

func (r *unlockRegistry) LoadAndDelete(store *Store) func() error {
	<-r.gate
	unlock := r.items[store]
	delete(r.items, store)
	r.gate <- struct{}{}
	return unlock
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	dbErr := s.db.Close()
	var lockErr error
	if unlock := storeUnlockers.LoadAndDelete(s); unlock != nil {
		lockErr = unlock()
	}
	return errors.Join(dbErr, lockErr)
}

func (s *Store) DB() *sql.DB { return s.db }

func sqliteDSN(path string, busyTimeout time.Duration) string {
	milliseconds := busyTimeout.Milliseconds()
	query := url.Values{}
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", milliseconds))
	query.Add("_pragma", "foreign_keys(ON)")
	return "file:" + filepath.ToSlash(filepath.Clean(path)) + "?" + query.Encode()
}

type Caller struct {
	ID                   string
	DisplayName          string
	QuotaMicroUSD        money.MicroUSD
	SettledSpendMicroUSD money.MicroUSD
	HeldAmountMicroUSD   money.MicroUSD
	Enabled              bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (c Caller) RemainingMicroUSD() money.MicroUSD {
	remaining := int64(c.QuotaMicroUSD) - int64(c.SettledSpendMicroUSD) - int64(c.HeldAmountMicroUSD)
	return money.MicroUSD(remaining)
}

type CallerSpec struct {
	ID            string
	DisplayName   string
	QuotaMicroUSD money.MicroUSD
	Enabled       bool
}

func (s *Store) CreateCaller(ctx context.Context, spec CallerSpec) (Caller, error) {
	if strings.TrimSpace(spec.ID) == "" || spec.QuotaMicroUSD < 0 {
		return Caller{}, fmt.Errorf("%w: caller id and non-negative quota are required", ErrInvalidArgument)
	}
	now := nowUnixMilli()
	_, err := s.db.ExecContext(ctx, `INSERT INTO callers(
		id, display_name, quota_micro_usd, enabled, created_at_unix_ms, updated_at_unix_ms
	) VALUES (?, ?, ?, ?, ?, ?)`,
		spec.ID, spec.DisplayName, spec.QuotaMicroUSD, boolInt(spec.Enabled), now, now)
	if err != nil {
		return Caller{}, fmt.Errorf("create caller: %w", err)
	}
	return s.GetCaller(ctx, spec.ID)
}

func (s *Store) GetCaller(ctx context.Context, callerID string) (Caller, error) {
	return scanCaller(s.db.QueryRowContext(ctx, `SELECT id, display_name, quota_micro_usd,
		settled_spend_micro_usd, held_amount_micro_usd, enabled, created_at_unix_ms, updated_at_unix_ms
		FROM callers WHERE id = ?`, callerID))
}

func (s *Store) ListCallers(ctx context.Context, limit int) ([]Caller, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, display_name, quota_micro_usd,
		settled_spend_micro_usd, held_amount_micro_usd, enabled, created_at_unix_ms, updated_at_unix_ms
		FROM callers ORDER BY created_at_unix_ms DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Caller
	for rows.Next() {
		caller, err := scanCaller(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, caller)
	}
	return out, rows.Err()
}

func (s *Store) SetCallerQuota(ctx context.Context, callerID string, quota money.MicroUSD) error {
	if quota < 0 {
		return fmt.Errorf("%w: quota must not be negative", ErrInvalidArgument)
	}
	// Allow raising/lowering quota freely; over-settled balances stay fail-closed on reserve.
	result, err := s.db.ExecContext(ctx, `UPDATE callers SET quota_micro_usd = ?, updated_at_unix_ms = ?
		WHERE id = ?`, quota, nowUnixMilli(), callerID)
	if err != nil {
		return fmt.Errorf("set caller quota: %w", err)
	}
	return requireOneRow(result, ErrCallerNotFound)
}

func (s *Store) SetCallerEnabled(ctx context.Context, callerID string, enabled bool) error {
	result, err := s.db.ExecContext(ctx, `UPDATE callers SET enabled = ?, updated_at_unix_ms = ? WHERE id = ?`,
		boolInt(enabled), nowUnixMilli(), callerID)
	if err != nil {
		return fmt.Errorf("set caller enabled: %w", err)
	}
	return requireOneRow(result, ErrCallerNotFound)
}

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

type MatchKind string

const (
	MatchExact  MatchKind = "exact"
	MatchGlob   MatchKind = "glob"
	MatchRegexp MatchKind = "regexp"
)

type PricingRule struct {
	ID        string             `json:"id"`
	MatchKind MatchKind          `json:"match_kind"`
	Pattern   string             `json:"pattern"`
	Priority  int                `json:"priority"`
	Price     money.PricePerMTok `json:"price"`
	Enabled   bool               `json:"enabled"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

func (r PricingRule) Validate() error {
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.Pattern) == "" {
		return fmt.Errorf("%w: pricing rule id and pattern are required", ErrInvalidArgument)
	}
	if err := r.Price.Validate(); err != nil {
		return err
	}
	switch r.MatchKind {
	case MatchExact:
	case MatchGlob:
		if _, err := filepath.Match(r.Pattern, "validation"); err != nil {
			return fmt.Errorf("%w: invalid glob: %v", ErrInvalidArgument, err)
		}
	case MatchRegexp:
		if _, err := regexp.Compile(r.Pattern); err != nil {
			return fmt.Errorf("%w: invalid regexp: %v", ErrInvalidArgument, err)
		}
	default:
		return fmt.Errorf("%w: invalid match kind %q", ErrInvalidArgument, r.MatchKind)
	}
	return nil
}

func (s *Store) PutPricingRule(ctx context.Context, rule PricingRule) error {
	if err := rule.Validate(); err != nil {
		return err
	}
	now := nowUnixMilli()
	_, err := s.db.ExecContext(ctx, `INSERT INTO pricing_rules(
		id, match_kind, pattern, priority, input_per_mtok_micro_usd, output_per_mtok_micro_usd,
		reasoning_per_mtok_micro_usd, cached_per_mtok_micro_usd, cache_read_per_mtok_micro_usd,
		cache_creation_per_mtok_micro_usd, accounting_mode, billing_mode, per_image_micro_usd,
		enabled, created_at_unix_ms, updated_at_unix_ms
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET match_kind=excluded.match_kind, pattern=excluded.pattern,
		priority=excluded.priority, input_per_mtok_micro_usd=excluded.input_per_mtok_micro_usd,
		output_per_mtok_micro_usd=excluded.output_per_mtok_micro_usd,
		reasoning_per_mtok_micro_usd=excluded.reasoning_per_mtok_micro_usd,
		cached_per_mtok_micro_usd=excluded.cached_per_mtok_micro_usd,
		cache_read_per_mtok_micro_usd=excluded.cache_read_per_mtok_micro_usd,
		cache_creation_per_mtok_micro_usd=excluded.cache_creation_per_mtok_micro_usd,
		accounting_mode=excluded.accounting_mode, billing_mode=excluded.billing_mode,
		per_image_micro_usd=excluded.per_image_micro_usd,
		enabled=excluded.enabled, updated_at_unix_ms=excluded.updated_at_unix_ms`,
		rule.ID, rule.MatchKind, rule.Pattern, rule.Priority, rule.Price.Input, rule.Price.Output,
		rule.Price.Reasoning, rule.Price.Cached, rule.Price.CacheRead, rule.Price.CacheCreation,
		rule.Price.AccountingMode, rule.Price.BillingMode, rule.Price.PerImage,
		boolInt(rule.Enabled), now, now)
	if err != nil {
		return fmt.Errorf("put pricing rule: %w", err)
	}
	return nil
}

func (s *Store) DeletePricingRule(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM pricing_rules WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireOneRow(result, ErrPricingRuleNotFound)
}

func (s *Store) ListPricingRules(ctx context.Context) ([]PricingRule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, match_kind, pattern, priority,
		input_per_mtok_micro_usd, output_per_mtok_micro_usd, reasoning_per_mtok_micro_usd,
		cached_per_mtok_micro_usd, cache_read_per_mtok_micro_usd, cache_creation_per_mtok_micro_usd,
		accounting_mode, billing_mode, per_image_micro_usd, enabled, created_at_unix_ms, updated_at_unix_ms
		FROM pricing_rules ORDER BY priority DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PricingRule
	for rows.Next() {
		rule, err := scanPricingRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}

func (s *Store) GetPricingRule(ctx context.Context, id string) (PricingRule, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return PricingRule{}, fmt.Errorf("%w: pricing rule id is required", ErrInvalidArgument)
	}
	return scanPricingRule(s.db.QueryRowContext(ctx, `SELECT id, match_kind, pattern, priority,
		input_per_mtok_micro_usd, output_per_mtok_micro_usd, reasoning_per_mtok_micro_usd,
		cached_per_mtok_micro_usd, cache_read_per_mtok_micro_usd, cache_creation_per_mtok_micro_usd,
		accounting_mode, billing_mode, per_image_micro_usd, enabled, created_at_unix_ms, updated_at_unix_ms
		FROM pricing_rules WHERE id = ?`, id))
}

func (s *Store) ResolvePricingRule(ctx context.Context, model string) (PricingRule, error) {
	if strings.TrimSpace(model) == "" {
		return PricingRule{}, fmt.Errorf("%w: model is required", ErrInvalidArgument)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, match_kind, pattern, priority,
		input_per_mtok_micro_usd, output_per_mtok_micro_usd, reasoning_per_mtok_micro_usd,
		cached_per_mtok_micro_usd, cache_read_per_mtok_micro_usd, cache_creation_per_mtok_micro_usd,
		accounting_mode, billing_mode, per_image_micro_usd, enabled, created_at_unix_ms, updated_at_unix_ms
		FROM pricing_rules WHERE enabled = 1 ORDER BY priority DESC, id ASC`)
	if err != nil {
		return PricingRule{}, fmt.Errorf("list pricing rules: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		rule, err := scanPricingRule(rows)
		if err != nil {
			return PricingRule{}, err
		}
		matched, err := ruleMatches(rule, model)
		if err != nil {
			return PricingRule{}, err
		}
		if matched {
			return rule, nil
		}
	}
	if err := rows.Err(); err != nil {
		return PricingRule{}, err
	}
	return PricingRule{}, ErrPricingRuleNotFound
}

func ruleMatches(rule PricingRule, model string) (bool, error) {
	switch rule.MatchKind {
	case MatchExact:
		return model == rule.Pattern, nil
	case MatchGlob:
		return filepath.Match(rule.Pattern, model)
	case MatchRegexp:
		return regexp.MatchString(rule.Pattern, model)
	default:
		return false, fmt.Errorf("stored pricing rule %q has invalid match kind %q", rule.ID, rule.MatchKind)
	}
}

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
	var allowedJSON string
	var dailyQuota, weeklyQuota, monthlyQuota, maxConcurrent int64
	if err := tx.QueryRowContext(ctx, `SELECT enabled, revoked_at_unix_ms, expires_at_unix_ms, allowed_models_json,
		daily_quota_micro_usd, weekly_quota_micro_usd, monthly_quota_micro_usd, max_concurrent_requests
		FROM plugin_keys WHERE id = ? AND caller_id = ?`,
		request.PluginKeyID, request.CallerID).Scan(&keyEnabled, &revoked, &expires, &allowedJSON,
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

	now := nowUnixMilli()
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

type Settlement struct {
	LedgerID              string
	ReservationID         string
	Model                 string
	PricingRuleID         *string
	Usage                 money.TokenUsage
	CostMicroUSD          money.MicroUSD
	EstimatedCostMicroUSD money.MicroUSD
	Source                string
	Auth                  AuthIdentity
	Metrics               UsageMetrics
	SettlementSummary     string
}

// Settle finalizes a held reservation. Cost may exceed held amount; remaining
// balance is allowed to go negative so real usage is never discarded.
func (s *Store) Settle(ctx context.Context, settlement Settlement) (Reservation, error) {
	if strings.TrimSpace(settlement.ReservationID) == "" || strings.TrimSpace(settlement.Model) == "" || settlement.CostMicroUSD < 0 {
		return Reservation{}, fmt.Errorf("%w: reservation, model, and non-negative cost are required", ErrInvalidArgument)
	}
	if err := settlement.Usage.Validate(); err != nil {
		return Reservation{}, err
	}
	if settlement.LedgerID == "" {
		settlement.LedgerID = newID()
	}
	if settlement.Source == "" {
		settlement.Source = "usage"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Reservation{}, err
	}
	defer tx.Rollback()
	reservation, err := getReservation(ctx, tx, settlement.ReservationID)
	if err != nil {
		return Reservation{}, err
	}
	if reservation.Status == ReservationSettled {
		return reservation, nil
	}
	if reservation.Status != ReservationHeld {
		return Reservation{}, ErrReservationFinalized
	}
	now := nowUnixMilli()
	// Release the key hold first, then add actual cost. No remaining-quota
	// check: overage is recorded and all subsequent reservations fail closed.
	result, err := tx.ExecContext(ctx, `UPDATE plugin_keys SET
		held_amount_micro_usd = CASE WHEN held_amount_micro_usd >= ? THEN held_amount_micro_usd - ? ELSE 0 END,
		settled_spend_micro_usd = settled_spend_micro_usd + ?,
		updated_at_unix_ms = ?
		WHERE id = ?`,
		reservation.HeldMicroUSD, reservation.HeldMicroUSD, settlement.CostMicroUSD, now, reservation.PluginKeyID)
	if err != nil {
		return Reservation{}, fmt.Errorf("settle key quota: %w", err)
	}
	if err := requireOneRow(result, ErrPluginKeyNotFound); err != nil {
		return Reservation{}, err
	}
	result, err = tx.ExecContext(ctx, `UPDATE reservations SET status='settled', settled_micro_usd=?,
		model=?, settlement_summary=?, settled_at_unix_ms=?, updated_at_unix_ms=? WHERE id=? AND status='held'`,
		settlement.CostMicroUSD, settlement.Model, settlement.SettlementSummary, now, now, settlement.ReservationID)
	if err != nil {
		return Reservation{}, err
	}
	if err := requireOneRow(result, ErrReservationFinalized); err != nil {
		return Reservation{}, err
	}
	firstTokenLatencyMillis := nullableDurationMillis(settlement.Metrics.FirstTokenLatency)
	generationDurationMillis := nullableDurationMillis(settlement.Metrics.GenerationDuration)
	resultLabel := nullableText(settlement.Metrics.Result)
	auth := settlement.Auth
	_, err = tx.ExecContext(ctx, `INSERT INTO usage_ledger(
		id, reservation_id, caller_id, plugin_key_id, model, pricing_rule_id, input_tokens, output_tokens,
		reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, total_tokens, cost_micro_usd, estimated_cost_micro_usd, source,
		tier, result, first_token_latency_ms, generation_duration_ms, tokens_per_second, thinking_intensity,
		auth_id, auth_index, auth_name, auth_label, auth_provider, auth_type, auth_email, auth_path, created_at_unix_ms
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		settlement.LedgerID, settlement.ReservationID, reservation.CallerID, reservation.PluginKeyID, settlement.Model,
		settlement.PricingRuleID, settlement.Usage.Input, settlement.Usage.Output,
		settlement.Usage.Reasoning, settlement.Usage.Cached, settlement.Usage.CacheRead,
		settlement.Usage.CacheCreation, money.ReportedTotal(settlement.Usage), settlement.CostMicroUSD, settlement.EstimatedCostMicroUSD, settlement.Source,
		nullableText(settlement.Metrics.Tier), resultLabel, firstTokenLatencyMillis, generationDurationMillis,
		settlement.Metrics.TokensPerSecond, nullableText(settlement.Metrics.ThinkingIntensity),
		nullableString(auth.AuthID), nullableString(auth.AuthIndex), nullableString(auth.Name), nullableString(auth.Label),
		nullableString(auth.Provider), nullableString(auth.Type), nullableString(auth.Email), nullableString(auth.Path), now)
	if err != nil {
		return Reservation{}, fmt.Errorf("write usage ledger: %w", err)
	}
	details := fmt.Sprintf(`{"source":%q,"over_held":%t}`, settlement.Source, settlement.CostMicroUSD > reservation.HeldMicroUSD)
	if err := insertAudit(ctx, tx, reservation.CallerID, reservation.PluginKeyID, reservation.ID, "quota_settled", settlement.CostMicroUSD, details, now); err != nil {
		return Reservation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Reservation{}, err
	}
	return s.GetReservation(ctx, settlement.ReservationID)
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

type UsageMetrics struct {
	Tier               *string
	Result             *string
	FirstTokenLatency  *time.Duration
	GenerationDuration *time.Duration
	TokensPerSecond    *float64
	ThinkingIntensity  *string
}

type UsageEntry struct {
	ID                    string
	ReservationID         string
	CallerID              string
	PluginKeyID           string
	Model                 string
	PricingRuleID         *string
	Usage                 money.TokenUsage
	CostMicroUSD          money.MicroUSD
	EstimatedCostMicroUSD money.MicroUSD
	Source                string
	Auth                  AuthIdentity
	Metrics               UsageMetrics
	CreatedAt             time.Time
}

// UpdateUsageAuth attaches selected auth-file identity to an existing ledger row.
func (s *Store) UpdateUsageAuth(ctx context.Context, ledgerID string, auth AuthIdentity) error {
	ledgerID = strings.TrimSpace(ledgerID)
	if ledgerID == "" || auth.Empty() {
		return fmt.Errorf("%w: ledger id and auth identity are required", ErrInvalidArgument)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE usage_ledger SET
		auth_id = COALESCE(NULLIF(TRIM(auth_id), ''), ?),
		auth_index = COALESCE(NULLIF(TRIM(auth_index), ''), ?),
		auth_name = COALESCE(NULLIF(TRIM(auth_name), ''), ?),
		auth_label = COALESCE(NULLIF(TRIM(auth_label), ''), ?),
		auth_provider = COALESCE(NULLIF(TRIM(auth_provider), ''), ?),
		auth_type = COALESCE(NULLIF(TRIM(auth_type), ''), ?),
		auth_email = COALESCE(NULLIF(TRIM(auth_email), ''), ?),
		auth_path = COALESCE(NULLIF(TRIM(auth_path), ''), ?)
		WHERE id = ?`,
		nullableString(auth.AuthID), nullableString(auth.AuthIndex), nullableString(auth.Name), nullableString(auth.Label),
		nullableString(auth.Provider), nullableString(auth.Type), nullableString(auth.Email), nullableString(auth.Path), ledgerID)
	if err != nil {
		return fmt.Errorf("update usage auth: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("%w: usage ledger not found", ErrInvalidArgument)
	}
	return nil
}

// FindRecentFallback returns the newest reserved_fallback row so a late host
// usage callback can replace estimate placeholders, including rows that already
// have auth identity attached.
func (s *Store) FindRecentFallback(ctx context.Context, models []string, window time.Duration) (UsageEntry, bool, error) {
	if window <= 0 {
		window = 15 * time.Minute
	}
	query := `SELECT id FROM usage_ledger
		WHERE source = 'reserved_fallback'
		  AND created_at_unix_ms >= ?`
	args := []any{time.Now().Add(-window).UnixMilli()}
	cleaned := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		model = strings.ToLower(strings.TrimSpace(model))
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		cleaned = append(cleaned, model)
	}
	if len(cleaned) > 0 {
		placeholders := make([]string, len(cleaned))
		for i, model := range cleaned {
			placeholders[i] = "?"
			args = append(args, model)
		}
		query += ` AND LOWER(model) IN (` + strings.Join(placeholders, ",") + `)`
	}
	query += ` ORDER BY created_at_unix_ms DESC LIMIT 1`
	var id string
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UsageEntry{}, false, nil
		}
		return UsageEntry{}, false, fmt.Errorf("find reserved fallback: %w", err)
	}
	entry, err := s.GetUsage(ctx, id)
	if err != nil {
		return UsageEntry{}, false, err
	}
	return entry, true, nil
}

// UpdateUsageDetail replaces fallback token estimates with the host's final
// usage record and reprices the already-settled ledger row. Preserve
// protocol-derived sources when they were already available; only relabel
// the synthetic fallback source.
func (s *Store) UpdateUsageDetail(ctx context.Context, ledgerID string, usage money.TokenUsage, cost money.MicroUSD) error {
	ledgerID = strings.TrimSpace(ledgerID)
	if ledgerID == "" {
		return fmt.Errorf("%w: usage ledger id is required", ErrInvalidArgument)
	}
	if cost < 0 {
		return fmt.Errorf("%w: cost must not be negative", ErrInvalidArgument)
	}
	if err := usage.Validate(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var callerID, pluginKeyID, reservationID string
	var previousCost int64
	if err := tx.QueryRowContext(ctx, `SELECT caller_id, plugin_key_id, reservation_id, cost_micro_usd
		FROM usage_ledger WHERE id = ?`, ledgerID).Scan(&callerID, &pluginKeyID, &reservationID, &previousCost); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: usage ledger not found", ErrInvalidArgument)
		}
		return fmt.Errorf("load usage ledger: %w", err)
	}

	now := nowUnixMilli()
	result, err := tx.ExecContext(ctx, `UPDATE usage_ledger SET
		input_tokens=?, output_tokens=?, reasoning_tokens=?, cached_tokens=?, cache_read_tokens=?, cache_creation_tokens=?,
		total_tokens=?, cost_micro_usd=?,
		source=CASE WHEN source='reserved_fallback' THEN 'host_usage' ELSE source END
		WHERE id=?`,
		usage.Input, usage.Output, usage.Reasoning, usage.Cached, usage.CacheRead, usage.CacheCreation,
		money.ReportedTotal(usage), cost, ledgerID)
	if err != nil {
		return fmt.Errorf("update usage detail: %w", err)
	}
	if err := requireOneRow(result, ErrInvalidArgument); err != nil {
		return err
	}

	delta := int64(cost) - previousCost
	if delta != 0 {
		result, err = tx.ExecContext(ctx, `UPDATE plugin_keys SET
			settled_spend_micro_usd = CASE
				WHEN settled_spend_micro_usd + ? < 0 THEN 0
				ELSE settled_spend_micro_usd + ?
			END,
			updated_at_unix_ms = ?
			WHERE id = ?`, delta, delta, now, pluginKeyID)
		if err != nil {
			return fmt.Errorf("adjust settled spend: %w", err)
		}
		if err := requireOneRow(result, ErrPluginKeyNotFound); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE reservations SET settled_micro_usd=?, updated_at_unix_ms=?
			WHERE id=? AND status='settled'`, cost, now, reservationID); err != nil {
			return fmt.Errorf("adjust settled reservation: %w", err)
		}
		details := fmt.Sprintf(`{"previous":%d,"cost":%d}`, previousCost, cost)
		if err := insertAudit(ctx, tx, callerID, pluginKeyID, reservationID, "quota_repriced", cost, details, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetUsage(ctx context.Context, ledgerID string) (UsageEntry, error) {
	ledgerID = strings.TrimSpace(ledgerID)
	if ledgerID == "" {
		return UsageEntry{}, fmt.Errorf("%w: usage ledger id is required", ErrInvalidArgument)
	}
	entries, err := s.ListUsage(ctx, UsageFilter{LedgerID: ledgerID, Limit: 1})
	if err != nil {
		return UsageEntry{}, err
	}
	if len(entries) == 0 {
		return UsageEntry{}, fmt.Errorf("%w: usage ledger not found", ErrInvalidArgument)
	}
	return entries[0], nil
}

func (s *Store) ListUsage(ctx context.Context, filter UsageFilter) ([]UsageEntry, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT u.id, u.reservation_id, u.caller_id, u.plugin_key_id,
		u.model, u.pricing_rule_id,
		u.input_tokens, u.output_tokens, u.reasoning_tokens, u.cached_tokens, u.cache_read_tokens,
		u.cache_creation_tokens, u.total_tokens, u.cost_micro_usd, u.estimated_cost_micro_usd, u.source, u.tier, u.result, u.first_token_latency_ms,
		u.generation_duration_ms, u.tokens_per_second, u.thinking_intensity,
		COALESCE(u.auth_id, ''), COALESCE(u.auth_index, ''), COALESCE(u.auth_name, ''), COALESCE(u.auth_label, ''),
		COALESCE(u.auth_provider, ''), COALESCE(u.auth_type, ''), COALESCE(u.auth_email, ''), COALESCE(u.auth_path, ''),
		u.created_at_unix_ms
		FROM usage_ledger u`
	where, args := usageWhere(filter, "u")
	if where != "" {
		query += ` WHERE ` + where
	}
	query += ` ORDER BY u.created_at_unix_ms DESC LIMIT ? OFFSET ?`
	args = append(args, limit, max(filter.Offset, 0))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UsageEntry
	for rows.Next() {
		var entry UsageEntry
		var pricing sql.NullString
		var created int64
		var firstTokenLatency, generationDuration sql.NullInt64
		var tokensPerSecond sql.NullFloat64
		var tier, resultLabel, thinkingIntensity sql.NullString
		if err := rows.Scan(&entry.ID, &entry.ReservationID, &entry.CallerID, &entry.PluginKeyID,
			&entry.Model, &pricing, &entry.Usage.Input, &entry.Usage.Output, &entry.Usage.Reasoning, &entry.Usage.Cached,
			&entry.Usage.CacheRead, &entry.Usage.CacheCreation, &entry.Usage.ReportedTotal, &entry.CostMicroUSD, &entry.EstimatedCostMicroUSD, &entry.Source, &tier,
			&resultLabel, &firstTokenLatency, &generationDuration, &tokensPerSecond, &thinkingIntensity,
			&entry.Auth.AuthID, &entry.Auth.AuthIndex, &entry.Auth.Name, &entry.Auth.Label,
			&entry.Auth.Provider, &entry.Auth.Type, &entry.Auth.Email, &entry.Auth.Path,
			&created); err != nil {
			return nil, err
		}
		if pricing.Valid {
			value := pricing.String
			entry.PricingRuleID = &value
		}
		if tier.Valid {
			value := tier.String
			entry.Metrics.Tier = &value
		}
		if resultLabel.Valid {
			value := resultLabel.String
			entry.Metrics.Result = &value
		}
		if thinkingIntensity.Valid {
			value := thinkingIntensity.String
			entry.Metrics.ThinkingIntensity = &value
		}
		if firstTokenLatency.Valid {
			value := time.Duration(firstTokenLatency.Int64) * time.Millisecond
			entry.Metrics.FirstTokenLatency = &value
		}
		if generationDuration.Valid {
			value := time.Duration(generationDuration.Int64) * time.Millisecond
			entry.Metrics.GenerationDuration = &value
		}
		if tokensPerSecond.Valid {
			value := tokensPerSecond.Float64
			entry.Metrics.TokensPerSecond = &value
		}
		entry.CreatedAt = fromUnixMilli(created)
		out = append(out, entry)
	}
	return out, rows.Err()
}

func (s *Store) CountUsage(ctx context.Context, filter UsageFilter) (int64, error) {
	where, args := usageWhere(filter, "u")
	query := `SELECT COUNT(1) FROM usage_ledger u`
	if where != "" {
		query += ` WHERE ` + where
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count usage: %w", err)
	}
	return total, nil
}

type UsageFilter struct {
	LedgerID        string
	CallerID        string
	PluginKeyID     string
	Model           string
	Source          string
	AuthID          string
	AuthProvider    string
	AuthIndex       string
	From            *time.Time
	To              *time.Time
	MinCostMicroUSD *money.MicroUSD
	MaxCostMicroUSD *money.MicroUSD
	MinTokens       *int64
	MaxTokens       *int64
	Limit           int
	Offset          int
}

// UsageAuthSummary is a distinct auth identity observed in the usage ledger.
type UsageAuthSummary struct {
	AuthID    string `json:"auth_id"`
	AuthIndex string `json:"auth_index"`
	Provider  string `json:"auth_provider"`
	Name      string `json:"auth_name"`
	Label     string `json:"auth_label"`
	Email     string `json:"auth_email"`
}

func usageReportedTotalSQL(prefix string) string {
	return "CASE WHEN " + prefix + "total_tokens > 0 THEN " + prefix + "total_tokens WHEN (" +
		prefix + "input_tokens + " + prefix + "output_tokens + " + prefix + "reasoning_tokens) > 0 THEN (" +
		prefix + "input_tokens + " + prefix + "output_tokens + " + prefix + "reasoning_tokens) ELSE " +
		prefix + "cached_tokens END"
}

func usageWhere(filter UsageFilter, alias string) (string, []any) {
	prefix := ""
	if strings.TrimSpace(alias) != "" {
		prefix = strings.TrimSpace(alias) + "."
	}
	conds := make([]string, 0, 14)
	args := make([]any, 0, 14)
	if strings.TrimSpace(filter.LedgerID) != "" {
		conds = append(conds, prefix+"id = ?")
		args = append(args, filter.LedgerID)
	}
	if strings.TrimSpace(filter.CallerID) != "" {
		conds = append(conds, prefix+"caller_id = ?")
		args = append(args, filter.CallerID)
	}
	if strings.TrimSpace(filter.PluginKeyID) != "" {
		conds = append(conds, prefix+"plugin_key_id = ?")
		args = append(args, filter.PluginKeyID)
	}
	if strings.TrimSpace(filter.Model) != "" {
		conds = append(conds, prefix+"model = ?")
		args = append(args, filter.Model)
	}
	if strings.TrimSpace(filter.Source) != "" {
		conds = append(conds, prefix+"source = ?")
		args = append(args, filter.Source)
	}
	authID := strings.TrimSpace(filter.AuthID)
	authIndex := strings.TrimSpace(filter.AuthIndex)
	authProvider := strings.TrimSpace(filter.AuthProvider)
	if authProvider != "" {
		providers := authQuotaProviderAliases(authProvider)
		placeholders := strings.TrimRight(strings.Repeat("?,", len(providers)), ",")
		conds = append(conds, "LOWER(TRIM(COALESCE("+prefix+"auth_provider, ''))) IN ("+placeholders+")")
		for _, provider := range providers {
			args = append(args, provider)
		}
	}
	if authID != "" {
		if authIndex == "" {
			conds = append(conds, prefix+"auth_id = ?")
			args = append(args, authID)
		} else {
			conds = append(conds, "("+prefix+"auth_id = ? OR (COALESCE("+prefix+"auth_id, '') = '' AND "+prefix+"auth_index = ?))")
			args = append(args, authID, authIndex)
		}
	} else if authIndex != "" {
		conds = append(conds, prefix+"auth_index = ?")
		args = append(args, authIndex)
	}
	if filter.From != nil {
		conds = append(conds, prefix+"created_at_unix_ms >= ?")
		args = append(args, filter.From.UTC().UnixMilli())
	}
	if filter.To != nil {
		conds = append(conds, prefix+"created_at_unix_ms <= ?")
		args = append(args, filter.To.UTC().UnixMilli())
	}
	if filter.MinCostMicroUSD != nil {
		conds = append(conds, prefix+"cost_micro_usd >= ?")
		args = append(args, int64(*filter.MinCostMicroUSD))
	}
	if filter.MaxCostMicroUSD != nil {
		conds = append(conds, prefix+"cost_micro_usd <= ?")
		args = append(args, int64(*filter.MaxCostMicroUSD))
	}
	totalTokens := usageReportedTotalSQL(prefix)
	if filter.MinTokens != nil {
		conds = append(conds, totalTokens+" >= ?")
		args = append(args, *filter.MinTokens)
	}
	if filter.MaxTokens != nil {
		conds = append(conds, totalTokens+" <= ?")
		args = append(args, *filter.MaxTokens)
	}
	return strings.Join(conds, ` AND `), args
}

type UsageKeySummary struct {
	Label        string         `json:"label"`
	RequestCount int64          `json:"request_count"`
	CostMicroUSD money.MicroUSD `json:"cost_micro_usd"`
	InputTokens  int64          `json:"input_tokens"`
	OutputTokens int64          `json:"output_tokens"`
}

type UsageModelSummary struct {
	Model        string         `json:"model"`
	RequestCount int64          `json:"request_count"`
	CostMicroUSD money.MicroUSD `json:"cost_micro_usd"`
	InputTokens  int64          `json:"input_tokens"`
	OutputTokens int64          `json:"output_tokens"`
	TotalTokens  int64          `json:"total_tokens"`
}

// UsageDailySummary is a UTC calendar-day usage rollup for dashboard trends.
type UsageDailySummary struct {
	Date                string         `json:"date"`
	InputTokens         int64          `json:"input_tokens"`
	OutputTokens        int64          `json:"output_tokens"`
	CachedTokens        int64          `json:"cached_tokens"`
	CacheReadTokens     int64          `json:"cache_read_tokens"`
	CacheCreationTokens int64          `json:"cache_creation_tokens"`
	CostMicroUSD        money.MicroUSD `json:"cost_micro_usd"`
}

type UsageOverviewSummary struct {
	RequestCount          int64          `json:"request_count"`
	TotalTokens           int64          `json:"total_tokens"`
	EstimatedCostMicroUSD money.MicroUSD `json:"estimated_cost_micro_usd"`
}

// UsageFilteredSummary is the aggregate for an arbitrary usage filter.
type UsageFilteredSummary struct {
	RequestCount        int64          `json:"request_count"`
	InputTokens         int64          `json:"input_tokens"`
	OutputTokens        int64          `json:"output_tokens"`
	ReasoningTokens     int64          `json:"reasoning_tokens"`
	CachedTokens        int64          `json:"cached_tokens"`
	CacheReadTokens     int64          `json:"cache_read_tokens"`
	CacheCreationTokens int64          `json:"cache_creation_tokens"`
	TotalTokens         int64          `json:"total_tokens"`
	CostMicroUSD        money.MicroUSD `json:"cost_micro_usd"`
}

type UsageTokenTrendPoint struct {
	Date        string `json:"date"`
	TotalTokens int64  `json:"total_tokens"`
}

func (s *Store) UsageOverviewSummary(ctx context.Context) (UsageOverviewSummary, error) {
	var summary UsageOverviewSummary
	err := s.db.QueryRowContext(ctx, `SELECT
		COUNT(1),
		COALESCE(SUM(` + usageReportedTotalSQL("") + `), 0),
		COALESCE(SUM(cost_micro_usd), 0)
	FROM usage_ledger`).Scan(&summary.RequestCount, &summary.TotalTokens, &summary.EstimatedCostMicroUSD)
	if err != nil {
		return UsageOverviewSummary{}, fmt.Errorf("summarize overview usage: %w", err)
	}
	return summary, nil
}

func (s *Store) SummarizeUsageFiltered(ctx context.Context, filter UsageFilter) (UsageFilteredSummary, error) {
	query := `SELECT COUNT(1), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
		COALESCE(SUM(reasoning_tokens), 0), COALESCE(SUM(cached_tokens), 0),
		COALESCE(SUM(cache_read_tokens), 0), COALESCE(SUM(cache_creation_tokens), 0),
		COALESCE(SUM(` + usageReportedTotalSQL("") + `), 0),
		COALESCE(SUM(cost_micro_usd), 0)
	FROM usage_ledger u`
	where, args := usageWhere(filter, "u")
	if where != "" {
		query += ` WHERE ` + where
	}
	var summary UsageFilteredSummary
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&summary.RequestCount, &summary.InputTokens, &summary.OutputTokens, &summary.ReasoningTokens,
		&summary.CachedTokens, &summary.CacheReadTokens, &summary.CacheCreationTokens, &summary.TotalTokens, &summary.CostMicroUSD,
	); err != nil {
		return UsageFilteredSummary{}, fmt.Errorf("summarize filtered usage: %w", err)
	}
	return summary, nil
}

func (s *Store) UsageTokenTrend(ctx context.Context, days int) ([]UsageTokenTrendPoint, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	from := time.Now().UTC().AddDate(0, 0, 1-days).Truncate(24 * time.Hour).UnixMilli()
	rows, err := s.db.QueryContext(ctx, `SELECT
		strftime('%Y-%m-%d', created_at_unix_ms / 1000, 'unixepoch') AS day,
		COALESCE(SUM(` + usageReportedTotalSQL("") + `), 0)
	FROM usage_ledger
	WHERE created_at_unix_ms >= ?
	GROUP BY day
	ORDER BY day`, from)
	if err != nil {
		return nil, fmt.Errorf("summarize token trend: %w", err)
	}
	defer rows.Close()
	points := make([]UsageTokenTrendPoint, 0, days)
	for rows.Next() {
		var point UsageTokenTrendPoint
		if err := rows.Scan(&point.Date, &point.TotalTokens); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, rows.Err()
}

func (s *Store) SummarizeUsageByKey(ctx context.Context, callerID string) ([]UsageKeySummary, error) {
	return s.SummarizeUsageByKeyFiltered(ctx, UsageFilter{CallerID: callerID})
}

func (s *Store) SummarizeUsageByKeyFiltered(ctx context.Context, filter UsageFilter) ([]UsageKeySummary, error) {
	query := `
SELECT COALESCE(k.label, ''),
	COUNT(1), COALESCE(SUM(u.cost_micro_usd), 0),
	COALESCE(SUM(u.input_tokens), 0), COALESCE(SUM(u.output_tokens), 0)
FROM usage_ledger u
LEFT JOIN plugin_keys k ON k.id = u.plugin_key_id`
	where, args := usageWhere(filter, "u")
	if where != "" {
		query += ` WHERE ` + where
	}
	query += ` GROUP BY u.plugin_key_id, COALESCE(k.label, '')
ORDER BY COALESCE(SUM(u.cost_micro_usd), 0) DESC, COUNT(1) DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UsageKeySummary
	for rows.Next() {
		var item UsageKeySummary
		if err := rows.Scan(&item.Label, &item.RequestCount, &item.CostMicroUSD,
			&item.InputTokens, &item.OutputTokens); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) SummarizeUsageByModel(ctx context.Context, callerID, pluginKeyID string) ([]UsageModelSummary, error) {
	return s.SummarizeUsageByModelFiltered(ctx, UsageFilter{CallerID: callerID, PluginKeyID: pluginKeyID})
}

func (s *Store) SummarizeUsageByModelFiltered(ctx context.Context, filter UsageFilter) ([]UsageModelSummary, error) {
	query := `
	SELECT u.model, COUNT(1), COALESCE(SUM(u.cost_micro_usd), 0),
	COALESCE(SUM(u.input_tokens), 0), COALESCE(SUM(u.output_tokens), 0),
	COALESCE(SUM(` + usageReportedTotalSQL("u.") + `), 0)
FROM usage_ledger u`
	where, args := usageWhere(filter, "u")
	if where != "" {
		query += ` WHERE ` + where
	}
	query += ` GROUP BY u.model ORDER BY COALESCE(SUM(u.cost_micro_usd), 0) DESC, COUNT(1) DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UsageModelSummary
	for rows.Next() {
		var item UsageModelSummary
		if err := rows.Scan(&item.Model, &item.RequestCount, &item.CostMicroUSD, &item.InputTokens, &item.OutputTokens, &item.TotalTokens); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// ListUsedAuths returns distinct auth identities that appear in the usage ledger.
func (s *Store) ListUsedAuths(ctx context.Context) ([]UsageAuthSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT COALESCE(auth_id, ''), COALESCE(auth_index, ''), COALESCE(auth_provider, ''),
	COALESCE(MAX(auth_name), ''), COALESCE(MAX(auth_label), ''), COALESCE(MAX(auth_email), '')
FROM usage_ledger
WHERE COALESCE(TRIM(auth_id), '') != '' OR COALESCE(TRIM(auth_index), '') != ''
GROUP BY 1, 2, 3
ORDER BY 5 COLLATE NOCASE, 6 COLLATE NOCASE, 4 COLLATE NOCASE, 1 COLLATE NOCASE, 2 COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UsageAuthSummary
	for rows.Next() {
		var item UsageAuthSummary
		if err := rows.Scan(&item.AuthID, &item.AuthIndex, &item.Provider, &item.Name, &item.Label, &item.Email); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) SummarizeUsageDailyFiltered(ctx context.Context, filter UsageFilter) ([]UsageDailySummary, error) {
	query := `SELECT strftime('%Y-%m-%d', u.created_at_unix_ms / 1000, 'unixepoch') AS day,
		COALESCE(SUM(u.input_tokens), 0), COALESCE(SUM(u.output_tokens), 0), COALESCE(SUM(u.cached_tokens), 0),
		COALESCE(SUM(u.cache_read_tokens), 0), COALESCE(SUM(u.cache_creation_tokens), 0),
		COALESCE(SUM(u.cost_micro_usd), 0)
	FROM usage_ledger u`
	where, args := usageWhere(filter, "u")
	if where != "" {
		query += ` WHERE ` + where
	}
	query += ` GROUP BY day ORDER BY day`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UsageDailySummary
	for rows.Next() {
		var item UsageDailySummary
		if err := rows.Scan(&item.Date, &item.InputTokens, &item.OutputTokens, &item.CachedTokens, &item.CacheReadTokens, &item.CacheCreationTokens, &item.CostMicroUSD); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

type AuditEvent struct {
	ID             int64           `json:"id"`
	CallerID       *string         `json:"caller_id"`
	PluginKeyID    *string         `json:"plugin_key_id"`
	ReservationID  *string         `json:"reservation_id"`
	EventType      string          `json:"event_type"`
	AmountMicroUSD *money.MicroUSD `json:"amount_micro_usd"`
	DetailsJSON    string          `json:"details_json"`
	CreatedAt      time.Time       `json:"created_at"`
}

type AuditFilter struct {
	CallerID    string
	PluginKeyID string
	Limit       int
}

func (s *Store) ListAuditEvents(ctx context.Context, callerID string, limit int) ([]AuditEvent, error) {
	return s.ListAuditEventsFiltered(ctx, AuditFilter{CallerID: callerID, Limit: limit})
}

func (s *Store) ListAuditEventsFiltered(ctx context.Context, filter AuditFilter) ([]AuditEvent, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, caller_id, plugin_key_id, reservation_id, event_type, amount_micro_usd, details_json, created_at_unix_ms
		FROM audit_events`
	var rows *sql.Rows
	var err error
	conds := make([]string, 0, 2)
	args := make([]any, 0, 3)
	if strings.TrimSpace(filter.CallerID) != "" {
		conds = append(conds, "caller_id = ?")
		args = append(args, filter.CallerID)
	}
	if strings.TrimSpace(filter.PluginKeyID) != "" {
		conds = append(conds, "plugin_key_id = ?")
		args = append(args, filter.PluginKeyID)
	}
	if len(conds) == 0 {
		rows, err = s.db.QueryContext(ctx, query+` ORDER BY id DESC LIMIT ?`, limit)
	} else {
		args = append(args, limit)
		rows, err = s.db.QueryContext(ctx, query+` WHERE `+strings.Join(conds, " AND ")+` ORDER BY id DESC LIMIT ?`, args...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditEvent
	for rows.Next() {
		var event AuditEvent
		var caller, key, reservation sql.NullString
		var amount sql.NullInt64
		var created int64
		if err := rows.Scan(&event.ID, &caller, &key, &reservation, &event.EventType, &amount, &event.DetailsJSON, &created); err != nil {
			return nil, err
		}
		if caller.Valid {
			value := caller.String
			event.CallerID = &value
		}
		if key.Valid {
			value := key.String
			event.PluginKeyID = &value
		}
		if reservation.Valid {
			value := reservation.String
			event.ReservationID = &value
		}
		if amount.Valid {
			value := money.MicroUSD(amount.Int64)
			event.AmountMicroUSD = &value
		}
		event.CreatedAt = fromUnixMilli(created)
		out = append(out, event)
	}
	return out, rows.Err()
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

type rowScanner interface{ Scan(...any) error }

func scanCaller(row rowScanner) (Caller, error) {
	var caller Caller
	var quota, settled, held int64
	var enabled int
	var created, updated int64
	if err := row.Scan(&caller.ID, &caller.DisplayName, &quota, &settled, &held, &enabled, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Caller{}, ErrCallerNotFound
		}
		return Caller{}, err
	}
	caller.QuotaMicroUSD = money.MicroUSD(quota)
	caller.SettledSpendMicroUSD = money.MicroUSD(settled)
	caller.HeldAmountMicroUSD = money.MicroUSD(held)
	caller.Enabled = enabled == 1
	caller.CreatedAt, caller.UpdatedAt = fromUnixMilli(created), fromUnixMilli(updated)
	return caller, nil
}

func scanPluginKey(row rowScanner) (PluginKey, error) {
	var key PluginKey
	var enabled int
	var revoked, expires, lastUsed, quota sql.NullInt64
	var dailyQuota, weeklyQuota, monthlyQuota, maxConcurrent, settled, held int64
	var allowedJSON string
	var created, updated int64
	if err := row.Scan(&key.ID, &key.CallerID, &key.Kid, &key.KeyHash, &key.EncryptedKeyMaterial, &key.PepperID, &key.Fingerprint,
		&key.Label, &key.Principal, &key.CallerScope, &enabled, &revoked, &expires, &lastUsed, &created, &updated,
		&quota, &dailyQuota, &weeklyQuota, &monthlyQuota, &maxConcurrent, &settled, &held, &allowedJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PluginKey{}, ErrPluginKeyNotFound
		}
		return PluginKey{}, err
	}
	key.Enabled = enabled == 1
	if revoked.Valid {
		value := fromUnixMilli(revoked.Int64)
		key.RevokedAt = &value
	}
	if expires.Valid {
		value := fromUnixMilli(expires.Int64)
		key.ExpiresAt = &value
	}
	if lastUsed.Valid {
		value := fromUnixMilli(lastUsed.Int64)
		key.LastUsedAt = &value
	}
	if quota.Valid {
		value := money.MicroUSD(quota.Int64)
		key.QuotaMicroUSD = &value
	}
	key.DailyQuotaMicroUSD = money.MicroUSD(dailyQuota)
	key.WeeklyQuotaMicroUSD = money.MicroUSD(weeklyQuota)
	key.MonthlyQuotaMicroUSD = money.MicroUSD(monthlyQuota)
	key.MaxConcurrentRequests = maxConcurrent
	key.SettledSpendMicroUSD = money.MicroUSD(settled)
	key.HeldAmountMicroUSD = money.MicroUSD(held)
	models, err := unmarshalAllowedModels(allowedJSON)
	if err != nil {
		return PluginKey{}, err
	}
	key.AllowedModels = models
	key.CreatedAt, key.UpdatedAt = fromUnixMilli(created), fromUnixMilli(updated)
	return key, nil
}

func marshalAllowedModels(models []string) (string, error) {
	clean := make([]string, 0, len(models))
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		clean = append(clean, m)
	}
	if clean == nil {
		clean = []string{}
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return "", fmt.Errorf("%w: encode allowed models: %v", ErrInvalidArgument, err)
	}
	return string(raw), nil
}

func unmarshalAllowedModels(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}, nil
	}
	var models []string
	if err := json.Unmarshal([]byte(raw), &models); err != nil {
		return nil, fmt.Errorf("%w: decode allowed models: %v", ErrInvalidArgument, err)
	}
	out := make([]string, 0, len(models))
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m != "" {
			out = append(out, m)
		}
	}
	return out, nil
}

func scanPricingRule(row rowScanner) (PricingRule, error) {
	var rule PricingRule
	var enabled int
	var created, updated int64
	if err := row.Scan(&rule.ID, &rule.MatchKind, &rule.Pattern, &rule.Priority,
		&rule.Price.Input, &rule.Price.Output, &rule.Price.Reasoning, &rule.Price.Cached,
		&rule.Price.CacheRead, &rule.Price.CacheCreation, &rule.Price.AccountingMode,
		&rule.Price.BillingMode, &rule.Price.PerImage, &enabled, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PricingRule{}, ErrPricingRuleNotFound
		}
		return PricingRule{}, err
	}
	rule.Enabled = enabled == 1
	rule.CreatedAt, rule.UpdatedAt = fromUnixMilli(created), fromUnixMilli(updated)
	return rule, nil
}

func scanReservation(row rowScanner) (Reservation, error) {
	var reservation Reservation
	var held int64
	var settled, settledAt, releasedAt sql.NullInt64
	var created, updated int64
	if err := row.Scan(&reservation.ID, &reservation.CallerID, &reservation.PluginKeyID, &reservation.IdempotencyKey,
		&reservation.Model, &reservation.RequestTokenEstimate, &held, &settled, &reservation.Status,
		&reservation.RequestSummary, &reservation.SettlementSummary, &created, &updated, &settledAt, &releasedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Reservation{}, ErrReservationNotFound
		}
		return Reservation{}, err
	}
	reservation.HeldMicroUSD = money.MicroUSD(held)
	if settled.Valid {
		value := money.MicroUSD(settled.Int64)
		reservation.SettledMicroUSD = &value
	}
	reservation.CreatedAt, reservation.UpdatedAt = fromUnixMilli(created), fromUnixMilli(updated)
	if settledAt.Valid {
		value := fromUnixMilli(settledAt.Int64)
		reservation.SettledAt = &value
	}
	if releasedAt.Valid {
		value := fromUnixMilli(releasedAt.Int64)
		reservation.ReleasedAt = &value
	}
	return reservation, nil
}

func nullableText(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return strings.TrimSpace(*value)
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableDurationMillis(value *time.Duration) any {
	if value == nil {
		return nil
	}
	return value.Milliseconds()
}

func insertAudit(ctx context.Context, tx *sql.Tx, callerID, pluginKeyID, reservationID, eventType string, amount money.MicroUSD, details string, now int64) error {
	if details == "" {
		details = "{}"
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_events(caller_id, plugin_key_id, reservation_id, event_type,
		amount_micro_usd, details_json, created_at_unix_ms) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		callerID, pluginKeyID, reservationID, eventType, amount, details, now)
	if err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}
	return nil
}

func requireOneRow(result sql.Result, zeroErr error) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return zeroErr
	}
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nowUnixMilli() int64                 { return time.Now().UTC().UnixMilli() }
func fromUnixMilli(value int64) time.Time { return time.UnixMilli(value).UTC() }

func newID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(value[:])
}

// NewID returns a new opaque entity identifier.
func NewID() string { return newID() }
