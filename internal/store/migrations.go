package store

import (
	"context"
	"database/sql"
	"fmt"
)

type migration struct {
	version int
	name    string
	up      []string
}

var migrations = []migration{
	{
		version: 1,
		name:    "plugin key credit ledger",
		up: []string{
			`CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				applied_at_unix_ms INTEGER NOT NULL
			)`,
			// Ownership records are retained for compatibility and usage attribution;
			// quota enforcement is exclusively per plugin key.
			`CREATE TABLE IF NOT EXISTS callers (
				id TEXT PRIMARY KEY,
				display_name TEXT NOT NULL DEFAULT '',
				quota_micro_usd INTEGER NOT NULL CHECK (quota_micro_usd >= 0),
				settled_spend_micro_usd INTEGER NOT NULL DEFAULT 0 CHECK (settled_spend_micro_usd >= 0),
				held_amount_micro_usd INTEGER NOT NULL DEFAULT 0 CHECK (held_amount_micro_usd >= 0),
				enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
				created_at_unix_ms INTEGER NOT NULL,
				updated_at_unix_ms INTEGER NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS plugin_keys (
				id TEXT PRIMARY KEY,
				caller_id TEXT NOT NULL REFERENCES callers(id) ON DELETE RESTRICT,
				kid TEXT NOT NULL UNIQUE,
				key_hash BLOB NOT NULL UNIQUE,
				pepper_id TEXT NOT NULL,
				fingerprint TEXT NOT NULL,
				label TEXT NOT NULL DEFAULT '',
				principal TEXT NOT NULL UNIQUE,
				caller_scope TEXT NOT NULL UNIQUE,
				enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
				revoked_at_unix_ms INTEGER,
				expires_at_unix_ms INTEGER,
				last_used_at_unix_ms INTEGER,
				created_at_unix_ms INTEGER NOT NULL,
				updated_at_unix_ms INTEGER NOT NULL,
				CHECK (length(key_hash) >= 16),
				CHECK (length(kid) >= 8),
				CHECK (length(caller_scope) >= 16)
			)`,
			`CREATE INDEX IF NOT EXISTS plugin_keys_caller_idx
				ON plugin_keys(caller_id, enabled)`,
			`CREATE TABLE IF NOT EXISTS pricing_rules (
				id TEXT PRIMARY KEY,
				match_kind TEXT NOT NULL CHECK (match_kind IN ('exact', 'glob', 'regexp')),
				pattern TEXT NOT NULL,
				priority INTEGER NOT NULL DEFAULT 0,
				input_per_mtok_micro_usd INTEGER NOT NULL CHECK (input_per_mtok_micro_usd >= 0),
				output_per_mtok_micro_usd INTEGER NOT NULL CHECK (output_per_mtok_micro_usd >= 0),
				reasoning_per_mtok_micro_usd INTEGER NOT NULL CHECK (reasoning_per_mtok_micro_usd >= 0),
				cached_per_mtok_micro_usd INTEGER NOT NULL CHECK (cached_per_mtok_micro_usd >= 0),
				cache_read_per_mtok_micro_usd INTEGER NOT NULL CHECK (cache_read_per_mtok_micro_usd >= 0),
				cache_creation_per_mtok_micro_usd INTEGER NOT NULL CHECK (cache_creation_per_mtok_micro_usd >= 0),
				enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
				created_at_unix_ms INTEGER NOT NULL,
				updated_at_unix_ms INTEGER NOT NULL,
				UNIQUE (match_kind, pattern, priority)
			)`,
			`CREATE INDEX IF NOT EXISTS pricing_rules_resolution_idx
				ON pricing_rules(enabled, priority DESC, id)`,
			`CREATE TABLE IF NOT EXISTS reservations (
				id TEXT PRIMARY KEY,
				caller_id TEXT NOT NULL REFERENCES callers(id) ON DELETE RESTRICT,
				plugin_key_id TEXT NOT NULL REFERENCES plugin_keys(id) ON DELETE RESTRICT,
				idempotency_key TEXT NOT NULL,
				model TEXT NOT NULL DEFAULT '',
				request_token_estimate INTEGER NOT NULL CHECK (request_token_estimate > 0),
				held_micro_usd INTEGER NOT NULL CHECK (held_micro_usd >= 0),
				settled_micro_usd INTEGER CHECK (settled_micro_usd IS NULL OR settled_micro_usd >= 0),
				status TEXT NOT NULL CHECK (status IN ('held', 'settled', 'released')),
				request_summary TEXT NOT NULL DEFAULT '',
				settlement_summary TEXT NOT NULL DEFAULT '',
				created_at_unix_ms INTEGER NOT NULL,
				updated_at_unix_ms INTEGER NOT NULL,
				settled_at_unix_ms INTEGER,
				released_at_unix_ms INTEGER,
				UNIQUE (caller_id, idempotency_key)
			)`,
			`CREATE INDEX IF NOT EXISTS reservations_caller_status_idx
				ON reservations(caller_id, status, created_at_unix_ms)`,
			`CREATE INDEX IF NOT EXISTS reservations_plugin_key_idx
				ON reservations(plugin_key_id, created_at_unix_ms)`,
			`CREATE TABLE IF NOT EXISTS usage_ledger (
				id TEXT PRIMARY KEY,
				reservation_id TEXT NOT NULL UNIQUE REFERENCES reservations(id) ON DELETE RESTRICT,
				caller_id TEXT NOT NULL REFERENCES callers(id) ON DELETE RESTRICT,
				plugin_key_id TEXT NOT NULL REFERENCES plugin_keys(id) ON DELETE RESTRICT,
				model TEXT NOT NULL,
				pricing_rule_id TEXT REFERENCES pricing_rules(id) ON DELETE SET NULL,
				input_tokens INTEGER NOT NULL CHECK (input_tokens >= 0),
				output_tokens INTEGER NOT NULL CHECK (output_tokens >= 0),
				reasoning_tokens INTEGER NOT NULL CHECK (reasoning_tokens >= 0),
				cached_tokens INTEGER NOT NULL CHECK (cached_tokens >= 0),
				cache_read_tokens INTEGER NOT NULL CHECK (cache_read_tokens >= 0),
				cache_creation_tokens INTEGER NOT NULL CHECK (cache_creation_tokens >= 0),
				cost_micro_usd INTEGER NOT NULL CHECK (cost_micro_usd >= 0),
				source TEXT NOT NULL DEFAULT 'usage',
				created_at_unix_ms INTEGER NOT NULL
			)`,
			`CREATE INDEX IF NOT EXISTS usage_ledger_caller_created_idx
				ON usage_ledger(caller_id, created_at_unix_ms)`,
			`CREATE TABLE IF NOT EXISTS audit_events (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				caller_id TEXT REFERENCES callers(id) ON DELETE SET NULL,
				plugin_key_id TEXT REFERENCES plugin_keys(id) ON DELETE SET NULL,
				reservation_id TEXT REFERENCES reservations(id) ON DELETE SET NULL,
				event_type TEXT NOT NULL,
				amount_micro_usd INTEGER,
				details_json TEXT NOT NULL DEFAULT '{}',
				created_at_unix_ms INTEGER NOT NULL
			)`,
			`CREATE INDEX IF NOT EXISTS audit_events_caller_created_idx
				ON audit_events(caller_id, created_at_unix_ms)`,
		},
	},
	{
		version: 2,
		name:    "key limits and allowed models",
		up: []string{
			// This transitional schema leaves the column nullable; migration 3
			// backfills old keys and new writes always provide a Key quota.
			`ALTER TABLE plugin_keys ADD COLUMN quota_micro_usd INTEGER`,
			`ALTER TABLE plugin_keys ADD COLUMN settled_spend_micro_usd INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE plugin_keys ADD COLUMN held_amount_micro_usd INTEGER NOT NULL DEFAULT 0`,
			// JSON array of exact/glob patterns; empty array means all models allowed.
			`ALTER TABLE plugin_keys ADD COLUMN allowed_models_json TEXT NOT NULL DEFAULT '[]'`,
			`CREATE INDEX IF NOT EXISTS usage_ledger_plugin_key_created_idx
				ON usage_ledger(plugin_key_id, created_at_unix_ms)`,
		},
	},
	{
		version: 4,
		name:    "usage execution metrics",
		up: []string{
			`ALTER TABLE usage_ledger ADD COLUMN tier TEXT`,
			`ALTER TABLE usage_ledger ADD COLUMN result TEXT`,
			`ALTER TABLE usage_ledger ADD COLUMN first_token_latency_ms INTEGER`,
			`ALTER TABLE usage_ledger ADD COLUMN generation_duration_ms INTEGER`,
			`ALTER TABLE usage_ledger ADD COLUMN tokens_per_second REAL`,
			`ALTER TABLE usage_ledger ADD COLUMN thinking_intensity TEXT`,
			`ALTER TABLE usage_ledger ADD COLUMN estimated_cost_micro_usd INTEGER`,
			`UPDATE usage_ledger
				SET estimated_cost_micro_usd = cost_micro_usd`,
		},
	},
	{
		version: 5,
		name:    "recoverable encrypted plugin keys",
		up: []string{
			`ALTER TABLE plugin_keys ADD COLUMN encrypted_key_material BLOB`,
		},
	},
	{
		version: 6,
		name:    "usage ledger selected auth identity",
		up: []string{
			`ALTER TABLE usage_ledger ADD COLUMN auth_id TEXT`,
			`ALTER TABLE usage_ledger ADD COLUMN auth_index TEXT`,
			`ALTER TABLE usage_ledger ADD COLUMN auth_name TEXT`,
			`ALTER TABLE usage_ledger ADD COLUMN auth_label TEXT`,
			`ALTER TABLE usage_ledger ADD COLUMN auth_provider TEXT`,
			`ALTER TABLE usage_ledger ADD COLUMN auth_type TEXT`,
			`ALTER TABLE usage_ledger ADD COLUMN auth_email TEXT`,
			`ALTER TABLE usage_ledger ADD COLUMN auth_path TEXT`,
			`CREATE INDEX IF NOT EXISTS usage_ledger_auth_created_idx
				ON usage_ledger(auth_id, created_at_unix_ms)`,
		},
	},
	{
		version: 7,
		name:    "plugin key period and concurrency limits",
		up: []string{
			// Zero means the corresponding limit is unlimited.
			`ALTER TABLE plugin_keys ADD COLUMN daily_quota_micro_usd INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE plugin_keys ADD COLUMN weekly_quota_micro_usd INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE plugin_keys ADD COLUMN monthly_quota_micro_usd INTEGER NOT NULL DEFAULT 0`,
			`ALTER TABLE plugin_keys ADD COLUMN max_concurrent_requests INTEGER NOT NULL DEFAULT 0`,
			`CREATE INDEX IF NOT EXISTS reservations_plugin_key_status_created_idx
				ON reservations(plugin_key_id, status, created_at_unix_ms)`,
			`CREATE INDEX IF NOT EXISTS usage_ledger_plugin_key_created_cost_idx
				ON usage_ledger(plugin_key_id, created_at_unix_ms, cost_micro_usd)`,
		},
	},
	{
		version: 8,
		name:    "auth quota snapshots",
		up: []string{
			`CREATE TABLE IF NOT EXISTS auth_quota_snapshots (
				provider TEXT NOT NULL,
				auth_id TEXT NOT NULL,
				snapshot_json TEXT NOT NULL DEFAULT '{}',
				auth_mod_time_unix_ms INTEGER,
				last_attempt_at_unix_ms INTEGER NOT NULL,
				last_success_at_unix_ms INTEGER,
				last_error TEXT NOT NULL DEFAULT '',
				PRIMARY KEY (provider, auth_id)
			)`,
			`CREATE INDEX IF NOT EXISTS auth_quota_snapshots_attempt_idx
				ON auth_quota_snapshots(last_attempt_at_unix_ms)`,
			`CREATE INDEX IF NOT EXISTS usage_ledger_auth_provider_id_created_idx
				ON usage_ledger(auth_provider, auth_id, created_at_unix_ms)`,
			`CREATE INDEX IF NOT EXISTS usage_ledger_auth_provider_index_created_idx
				ON usage_ledger(auth_provider, auth_index, created_at_unix_ms)`,
		},
	},
	{
		version: 9,
		name:    "auth quota snapshot error timestamp",
		up: []string{
			`ALTER TABLE auth_quota_snapshots ADD COLUMN last_error_at_unix_ms INTEGER`,
		},
	},
	{
		version: 10,
		name:    "pricing accounting mode",
		up: []string{
			`ALTER TABLE pricing_rules ADD COLUMN accounting_mode TEXT NOT NULL DEFAULT ''`,
		},
	},
	{
		version: 11,
		name:    "auth quota window baselines",
		up: []string{
			`CREATE TABLE IF NOT EXISTS auth_quota_window_baselines (
				provider TEXT NOT NULL,
				auth_id TEXT NOT NULL,
				window_id TEXT NOT NULL,
				cycle_key TEXT NOT NULL,
				baseline_used REAL NOT NULL,
				created_at_unix_ms INTEGER NOT NULL,
				PRIMARY KEY (provider, auth_id, window_id, cycle_key)
			)`,
		},
	},
	{
		version: 12,
		name:    "pricing billing mode and per-image price",
		up: []string{
			`ALTER TABLE pricing_rules ADD COLUMN billing_mode TEXT NOT NULL DEFAULT ''`,
			`ALTER TABLE pricing_rules ADD COLUMN per_image_micro_usd INTEGER NOT NULL DEFAULT 0`,
		},
	},
	{
		version: 13,
		name:    "persist official usage total tokens",
		up: []string{
			`ALTER TABLE usage_ledger ADD COLUMN total_tokens INTEGER NOT NULL DEFAULT 0`,
			`UPDATE usage_ledger SET total_tokens = CASE
				WHEN (input_tokens + output_tokens + reasoning_tokens) > 0 THEN (input_tokens + output_tokens + reasoning_tokens)
				ELSE cached_tokens
			END`,
		},
	},
	{
		version: 14,
		name:    "plugin key model token limits",
		up: []string{
			`ALTER TABLE plugin_keys ADD COLUMN model_token_limits_json TEXT NOT NULL DEFAULT '[]'`,
		},
	},
	{
		version: 15,
		name:    "plugin key unmatched models mode",
		up: []string{
			`ALTER TABLE plugin_keys ADD COLUMN unmatched_models_mode TEXT NOT NULL DEFAULT 'available'`,
		},
	},
	{
		version: 16,
		name:    "auth concurrency limits",
		up: []string{
			`CREATE TABLE IF NOT EXISTS auth_concurrency_limits (
				provider TEXT NOT NULL,
				auth_id TEXT NOT NULL,
				max_concurrent_requests INTEGER NOT NULL DEFAULT 0 CHECK (max_concurrent_requests >= 0),
				updated_at_unix_ms INTEGER NOT NULL,
				PRIMARY KEY (provider, auth_id)
			)`,
		},
	},
	{
		version: 17,
		name:    "usage ledger executor type",
		up: []string{
			`ALTER TABLE usage_ledger ADD COLUMN executor_type TEXT`,
		},
	},
	{
		version: 18,
		name:    "pricing context and service tiers",
		up: []string{
			`ALTER TABLE pricing_rules ADD COLUMN tiers_json TEXT NOT NULL DEFAULT '[]'`,
		},
	},
	{
		version: 19,
		name:    "plugin key spend reset epochs",
		up: []string{
			`ALTER TABLE plugin_keys ADD COLUMN total_spend_reset_at_unix_ms INTEGER`,
			`ALTER TABLE plugin_keys ADD COLUMN daily_spend_reset_at_unix_ms INTEGER`,
			`ALTER TABLE plugin_keys ADD COLUMN weekly_spend_reset_at_unix_ms INTEGER`,
			`ALTER TABLE plugin_keys ADD COLUMN monthly_spend_reset_at_unix_ms INTEGER`,
		},
	},
}

// Migrate applies every pending migration transactionally.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at_unix_ms INTEGER NOT NULL
	)`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	for _, m := range migrations {
		applied, err := migrationApplied(ctx, db, m.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := applyMigration(ctx, db, m); err != nil {
			return fmt.Errorf("apply migration %d (%s): %w", m.version, m.name, err)
		}
	}
	return nil
}

func migrationApplied(ctx context.Context, db *sql.DB, version int) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, version).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func applyMigration(ctx context.Context, db *sql.DB, m migration) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, statement := range m.up {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations(version, name, applied_at_unix_ms)
		 VALUES (?, ?, unixepoch('subsec') * 1000)`,
		m.version, m.name,
	); err != nil {
		return err
	}
	return tx.Commit()
}
