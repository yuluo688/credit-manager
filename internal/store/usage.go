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
	ExecutorType          string
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

// UpdateUsageTier records the actual upstream service_tier on a ledger row.
func (s *Store) UpdateUsageTier(ctx context.Context, ledgerID, tier string) error {
	ledgerID = strings.TrimSpace(ledgerID)
	tier = strings.TrimSpace(tier)
	if ledgerID == "" || tier == "" {
		return fmt.Errorf("%w: ledger id and tier are required", ErrInvalidArgument)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE usage_ledger SET tier = ? WHERE id = ?`, tier, ledgerID)
	if err != nil {
		return fmt.Errorf("update usage tier: %w", err)
	}
	return requireOneRow(result, ErrInvalidArgument)
}

// UpdateUsageExecutor attaches the concrete host executor to an existing ledger row.
func (s *Store) UpdateUsageExecutor(ctx context.Context, ledgerID, executorType string) error {
	ledgerID = strings.TrimSpace(ledgerID)
	executorType = strings.TrimSpace(executorType)
	if ledgerID == "" || executorType == "" {
		return fmt.Errorf("%w: ledger id and executor type are required", ErrInvalidArgument)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE usage_ledger SET
		executor_type = COALESCE(NULLIF(TRIM(executor_type), ''), ?)
		WHERE id = ?`, executorType, ledgerID)
	if err != nil {
		return fmt.Errorf("update usage executor: %w", err)
	}
	if err := requireOneRow(result, ErrInvalidArgument); err != nil {
		return err
	}
	return nil
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
	cleaned := uniqueLowerModels(models)
	rows, err := s.db.QueryContext(ctx, `SELECT id, model FROM usage_ledger
		WHERE source = 'reserved_fallback'
		  AND created_at_unix_ms >= ?
		ORDER BY created_at_unix_ms DESC`, time.Now().Add(-window).UnixMilli())
	if err != nil {
		return UsageEntry{}, false, fmt.Errorf("find reserved fallback: %w", err)
	}
	chosenID := ""
	relatedID := ""
	for rows.Next() {
		var id, model string
		if err := rows.Scan(&id, &model); err != nil {
			_ = rows.Close()
			return UsageEntry{}, false, err
		}
		if len(cleaned) == 0 || exactModelMatch(model, cleaned) {
			chosenID = id
			break
		}
		if relatedID == "" && relatedModelMatch(model, cleaned) {
			relatedID = id
		}
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return UsageEntry{}, false, err
	}
	if chosenID == "" {
		chosenID = relatedID
	}
	if chosenID == "" {
		return UsageEntry{}, false, nil
	}
	entry, err := s.GetUsage(ctx, chosenID)
	if err != nil {
		return UsageEntry{}, false, err
	}
	return entry, true, nil
}

func uniqueLowerModels(models []string) []string {
	seen := make(map[string]struct{}, len(models))
	out := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.ToLower(strings.TrimSpace(model))
		if model == "" {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		out = append(out, model)
	}
	return out
}

func exactModelMatch(model string, wanted []string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	for _, want := range wanted {
		if model == want {
			return true
		}
	}
	return false
}

func relatedModelMatch(model string, wanted []string) bool {
	for _, want := range wanted {
		if ModelsRelated(model, want) {
			return true
		}
	}
	return false
}

// ModelsRelated reports whether two model ids are the same or a dated/build suffix of each other.
func ModelsRelated(left, right string) bool {
	left = strings.ToLower(strings.TrimSpace(left))
	right = strings.ToLower(strings.TrimSpace(right))
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	return modelHasPrefix(left, right) || modelHasPrefix(right, left)
}

func modelHasPrefix(model, prefix string) bool {
	if !strings.HasPrefix(model, prefix) || len(model) <= len(prefix) {
		return false
	}
	next := model[len(prefix)]
	return next == '-' || next == '.' || next == '_'
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
	var previousCost, createdAt int64
	var totalReset sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT u.caller_id, u.plugin_key_id, u.reservation_id, u.cost_micro_usd, u.created_at_unix_ms,
		k.total_spend_reset_at_unix_ms
		FROM usage_ledger u JOIN plugin_keys k ON k.id = u.plugin_key_id
		WHERE u.id = ?`, ledgerID).Scan(&callerID, &pluginKeyID, &reservationID, &previousCost, &createdAt, &totalReset); err != nil {
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
		if !(totalReset.Valid && createdAt < totalReset.Int64) {
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
		COALESCE(u.executor_type, ''), u.model, u.pricing_rule_id,
		u.input_tokens, u.output_tokens, u.reasoning_tokens, u.cached_tokens, u.cache_read_tokens,
		u.cache_creation_tokens, u.total_tokens, u.cost_micro_usd, COALESCE(u.estimated_cost_micro_usd, u.cost_micro_usd, 0), u.source, u.tier, u.result, u.first_token_latency_ms,
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
			&entry.ExecutorType, &entry.Model, &pricing, &entry.Usage.Input, &entry.Usage.Output, &entry.Usage.Reasoning, &entry.Usage.Cached,
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
	Model              string         `json:"model"`
	RequestCount       int64          `json:"request_count"`
	CostMicroUSD       money.MicroUSD `json:"cost_micro_usd"`
	InputTokens        int64          `json:"input_tokens"`
	OutputTokens       int64          `json:"output_tokens"`
	TotalTokens        int64          `json:"total_tokens"`
	CacheReadTokens    int64          `json:"cache_read_tokens"`
	AvgTokensPerSecond *float64       `json:"avg_tokens_per_second,omitempty"`
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
		COALESCE(SUM(`+usageReportedTotalSQL("")+`), 0),
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
		COALESCE(SUM(`+usageReportedTotalSQL("")+`), 0)
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
	COALESCE(SUM(` + usageReportedTotalSQL("u.") + `), 0),
	COALESCE(SUM(u.cache_read_tokens), 0),
	AVG(CASE WHEN u.tokens_per_second IS NOT NULL AND u.tokens_per_second > 0 THEN u.tokens_per_second END)
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
		var avgTPS sql.NullFloat64
		if err := rows.Scan(&item.Model, &item.RequestCount, &item.CostMicroUSD, &item.InputTokens, &item.OutputTokens, &item.TotalTokens, &item.CacheReadTokens, &avgTPS); err != nil {
			return nil, err
		}
		if avgTPS.Valid {
			value := avgTPS.Float64
			item.AvgTokensPerSecond = &value
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

func usageTrendLayout(grain string) string {
	switch strings.ToLower(strings.TrimSpace(grain)) {
	case "hour":
		return "%Y-%m-%dT%H:00"
	case "month":
		return "%Y-%m"
	default:
		return "%Y-%m-%d"
	}
}

func (s *Store) SummarizeUsageTrendFiltered(ctx context.Context, filter UsageFilter, grain string) ([]UsageDailySummary, error) {
	query := `SELECT strftime('` + usageTrendLayout(grain) + `', u.created_at_unix_ms / 1000, 'unixepoch') AS bucket,
		COALESCE(SUM(u.input_tokens), 0), COALESCE(SUM(u.output_tokens), 0), COALESCE(SUM(u.cached_tokens), 0),
		COALESCE(SUM(u.cache_read_tokens), 0), COALESCE(SUM(u.cache_creation_tokens), 0),
		COALESCE(SUM(u.cost_micro_usd), 0)
	FROM usage_ledger u`
	where, args := usageWhere(filter, "u")
	if where != "" {
		query += ` WHERE ` + where
	}
	query += ` GROUP BY bucket ORDER BY bucket`
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
