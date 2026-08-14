package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	ID                   string
	CallerID             string
	Kid                  string
	KeyHash              []byte
	EncryptedKeyMaterial []byte
	PepperID             string
	Fingerprint          string
	Label                string
	Principal            string
	CallerScope          string
	Enabled              bool
	QuotaMicroUSD        *money.MicroUSD // 0 or nil = unlimited
	SettledSpendMicroUSD money.MicroUSD
	HeldAmountMicroUSD   money.MicroUSD
	AllowedModels        []string // empty = all models
	RevokedAt            *time.Time
	ExpiresAt            *time.Time
	LastUsedAt           *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
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
	ID                   string
	CallerID             string
	Kid                  string
	KeyHash              []byte
	EncryptedKeyMaterial []byte
	PepperID             string
	Fingerprint          string
	Label                string
	Principal            string
	CallerScope          string
	Enabled              bool
	ExpiresAt            *time.Time
	QuotaMicroUSD        money.MicroUSD
	AllowedModels        []string
}

// PluginKeyPolicyUpdate patches mutable admin fields on a key.
type PluginKeyPolicyUpdate struct {
	ID             string
	Label          *string
	Enabled        *bool
	QuotaMicroUSD  *money.MicroUSD
	AllowedModels  *[]string
	ExpiresAt      *time.Time
	ClearExpiresAt bool
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
	if spec.QuotaMicroUSD < 0 {
		return PluginKey{}, fmt.Errorf("%w: key quota must not be negative", ErrInvalidArgument)
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
		quota_micro_usd, settled_spend_micro_usd, held_amount_micro_usd, allowed_models_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?)`,
		spec.ID, spec.CallerID, spec.Kid, append([]byte(nil), spec.KeyHash...), append([]byte(nil), spec.EncryptedKeyMaterial...),
		spec.PepperID, spec.Fingerprint, spec.Label, spec.Principal, spec.CallerScope,
		boolInt(spec.Enabled), expires, now, now, quota, modelsJSON)
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
	if spec.QuotaMicroUSD < 0 {
		return PluginKey{}, fmt.Errorf("%w: key quota must not be negative", ErrInvalidArgument)
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
		quota_micro_usd, settled_spend_micro_usd, held_amount_micro_usd, allowed_models_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?)`,
		spec.ID, spec.CallerID, spec.Kid, append([]byte(nil), spec.KeyHash...), append([]byte(nil), spec.EncryptedKeyMaterial...),
		spec.PepperID, spec.Fingerprint, spec.Label, spec.Principal, spec.CallerScope,
		boolInt(spec.Enabled), expires, now, now, int64(spec.QuotaMicroUSD), modelsJSON)
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
		label = ?, enabled = ?, quota_micro_usd = ?, allowed_models_json = ?,
		expires_at_unix_ms = ?, updated_at_unix_ms = ?
		WHERE id = ?`,
		key.Label, boolInt(key.Enabled), quota, modelsJSON, expires, now, update.ID)
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
	quota_micro_usd, settled_spend_micro_usd, held_amount_micro_usd, allowed_models_json
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
		cache_creation_per_mtok_micro_usd, enabled, created_at_unix_ms, updated_at_unix_ms
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET match_kind=excluded.match_kind, pattern=excluded.pattern,
		priority=excluded.priority, input_per_mtok_micro_usd=excluded.input_per_mtok_micro_usd,
		output_per_mtok_micro_usd=excluded.output_per_mtok_micro_usd,
		reasoning_per_mtok_micro_usd=excluded.reasoning_per_mtok_micro_usd,
		cached_per_mtok_micro_usd=excluded.cached_per_mtok_micro_usd,
		cache_read_per_mtok_micro_usd=excluded.cache_read_per_mtok_micro_usd,
		cache_creation_per_mtok_micro_usd=excluded.cache_creation_per_mtok_micro_usd,
		enabled=excluded.enabled, updated_at_unix_ms=excluded.updated_at_unix_ms`,
		rule.ID, rule.MatchKind, rule.Pattern, rule.Priority, rule.Price.Input, rule.Price.Output,
		rule.Price.Reasoning, rule.Price.Cached, rule.Price.CacheRead, rule.Price.CacheCreation,
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
		enabled, created_at_unix_ms, updated_at_unix_ms
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

func (s *Store) ResolvePricingRule(ctx context.Context, model string) (PricingRule, error) {
	if strings.TrimSpace(model) == "" {
		return PricingRule{}, fmt.Errorf("%w: model is required", ErrInvalidArgument)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, match_kind, pattern, priority,
		input_per_mtok_micro_usd, output_per_mtok_micro_usd, reasoning_per_mtok_micro_usd,
		cached_per_mtok_micro_usd, cache_read_per_mtok_micro_usd, cache_creation_per_mtok_micro_usd,
		enabled, created_at_unix_ms, updated_at_unix_ms
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
	if err := tx.QueryRowContext(ctx, `SELECT enabled, revoked_at_unix_ms, expires_at_unix_ms, allowed_models_json
		FROM plugin_keys WHERE id = ? AND caller_id = ?`,
		request.PluginKeyID, request.CallerID).Scan(&keyEnabled, &revoked, &expires, &allowedJSON); err != nil {
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
		reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, cost_micro_usd, estimated_cost_micro_usd, source,
		tier, result, first_token_latency_ms, generation_duration_ms, tokens_per_second, thinking_intensity,
		auth_id, auth_index, auth_name, auth_label, auth_provider, auth_type, auth_email, auth_path, created_at_unix_ms
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		settlement.LedgerID, settlement.ReservationID, reservation.CallerID, reservation.PluginKeyID, settlement.Model,
		settlement.PricingRuleID, settlement.Usage.Input, settlement.Usage.Output,
		settlement.Usage.Reasoning, settlement.Usage.Cached, settlement.Usage.CacheRead,
		settlement.Usage.CacheCreation, settlement.CostMicroUSD, settlement.EstimatedCostMicroUSD, settlement.Source,
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

func (s *Store) ListUsage(ctx context.Context, filter UsageFilter) ([]UsageEntry, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT u.id, u.reservation_id, u.caller_id, u.plugin_key_id,
		u.model, u.pricing_rule_id,
		u.input_tokens, u.output_tokens, u.reasoning_tokens, u.cached_tokens, u.cache_read_tokens,
		u.cache_creation_tokens, u.cost_micro_usd, u.estimated_cost_micro_usd, u.source, u.tier, u.result, u.first_token_latency_ms,
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
			&entry.Usage.CacheRead, &entry.Usage.CacheCreation, &entry.CostMicroUSD, &entry.EstimatedCostMicroUSD, &entry.Source, &tier,
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
	CallerID        string
	PluginKeyID     string
	Model           string
	Source          string
	From            *time.Time
	To              *time.Time
	MinCostMicroUSD *money.MicroUSD
	MaxCostMicroUSD *money.MicroUSD
	MinTokens       *int64
	MaxTokens       *int64
	Limit           int
	Offset          int
}

func usageWhere(filter UsageFilter, alias string) (string, []any) {
	prefix := ""
	if strings.TrimSpace(alias) != "" {
		prefix = strings.TrimSpace(alias) + "."
	}
	conds := make([]string, 0, 10)
	args := make([]any, 0, 10)
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
	totalTokens := "(" + prefix + "input_tokens + " + prefix + "output_tokens + " + prefix + "reasoning_tokens + " + prefix + "cached_tokens + " + prefix + "cache_read_tokens + " + prefix + "cache_creation_tokens)"
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
}

type UsageOverviewSummary struct {
	RequestCount          int64          `json:"request_count"`
	TotalTokens           int64          `json:"total_tokens"`
	EstimatedCostMicroUSD money.MicroUSD `json:"estimated_cost_micro_usd"`
}

type UsageTokenTrendPoint struct {
	Date        string `json:"date"`
	TotalTokens int64  `json:"total_tokens"`
}

func (s *Store) UsageOverviewSummary(ctx context.Context) (UsageOverviewSummary, error) {
	var summary UsageOverviewSummary
	err := s.db.QueryRowContext(ctx, `SELECT
		COUNT(1),
		COALESCE(SUM(input_tokens + output_tokens + reasoning_tokens + cached_tokens + cache_read_tokens + cache_creation_tokens), 0),
		COALESCE(SUM(COALESCE(estimated_cost_micro_usd, cost_micro_usd)), 0)
	FROM usage_ledger`).Scan(&summary.RequestCount, &summary.TotalTokens, &summary.EstimatedCostMicroUSD)
	if err != nil {
		return UsageOverviewSummary{}, fmt.Errorf("summarize overview usage: %w", err)
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
		COALESCE(SUM(input_tokens + output_tokens + reasoning_tokens + cached_tokens + cache_read_tokens + cache_creation_tokens), 0)
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
	COALESCE(SUM(u.input_tokens), 0), COALESCE(SUM(u.output_tokens), 0)
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
		if err := rows.Scan(&item.Model, &item.RequestCount, &item.CostMicroUSD, &item.InputTokens, &item.OutputTokens); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

type AuditEvent struct {
	ID             int64
	CallerID       *string
	PluginKeyID    *string
	ReservationID  *string
	EventType      string
	AmountMicroUSD *money.MicroUSD
	DetailsJSON    string
	CreatedAt      time.Time
}

func (s *Store) ListAuditEvents(ctx context.Context, callerID string, limit int) ([]AuditEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, caller_id, plugin_key_id, reservation_id, event_type, amount_micro_usd, details_json, created_at_unix_ms
		FROM audit_events`
	var rows *sql.Rows
	var err error
	if strings.TrimSpace(callerID) == "" {
		rows, err = s.db.QueryContext(ctx, query+` ORDER BY id DESC LIMIT ?`, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, query+` WHERE caller_id = ? ORDER BY id DESC LIMIT ?`, callerID, limit)
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
	var settled, held int64
	var allowedJSON string
	var created, updated int64
	if err := row.Scan(&key.ID, &key.CallerID, &key.Kid, &key.KeyHash, &key.EncryptedKeyMaterial, &key.PepperID, &key.Fingerprint,
		&key.Label, &key.Principal, &key.CallerScope, &enabled, &revoked, &expires, &lastUsed, &created, &updated,
		&quota, &settled, &held, &allowedJSON); err != nil {
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
		&rule.Price.CacheRead, &rule.Price.CacheCreation, &enabled, &created, &updated); err != nil {
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
