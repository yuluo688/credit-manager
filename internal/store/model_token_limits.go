package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	ModelTokenLimitModeUnlimited = "unlimited"
	ModelTokenLimitModeAvailable = "available"
	UnmatchedModelsAvailable     = "available"
	UnmatchedModelsDisabled      = "disabled"
	maxModelTokenLimits          = 200
)

// ModelPeriodTokenLimit is one day/week/month token cap for a model.
// Tokens > 0 is a hard cap. Tokens == 0 uses Mode:
//   - unlimited: no token cap
//   - available: model remains usable with no token cap for this period
type ModelPeriodTokenLimit struct {
	Tokens int64  `json:"tokens"`
	Mode   string `json:"mode"`
}

func (p ModelPeriodTokenLimit) Cap() int64 {
	if p.Tokens > 0 {
		return p.Tokens
	}
	return 0
}

func (p ModelPeriodTokenLimit) NormalizedMode() string {
	mode := strings.ToLower(strings.TrimSpace(p.Mode))
	if mode == ModelTokenLimitModeAvailable {
		return ModelTokenLimitModeAvailable
	}
	return ModelTokenLimitModeUnlimited
}

type ModelTokenLimit struct {
	Model   string                `json:"model"`
	Daily   ModelPeriodTokenLimit `json:"daily"`
	Weekly  ModelPeriodTokenLimit `json:"weekly"`
	Monthly ModelPeriodTokenLimit `json:"monthly"`
}

type ModelTokenUsage struct {
	Model       string                `json:"model"`
	Daily       ModelPeriodTokenLimit `json:"daily"`
	Weekly      ModelPeriodTokenLimit `json:"weekly"`
	Monthly     ModelPeriodTokenLimit `json:"monthly"`
	DailyUsed   int64                 `json:"daily_used"`
	WeeklyUsed  int64                 `json:"weekly_used"`
	MonthlyUsed int64                 `json:"monthly_used"`
}

type tokenQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (p ModelPeriodTokenLimit) normalize() (ModelPeriodTokenLimit, error) {
	if p.Tokens < 0 {
		return ModelPeriodTokenLimit{}, fmt.Errorf("%w: model token limits must not be negative", ErrInvalidArgument)
	}
	mode := strings.ToLower(strings.TrimSpace(p.Mode))
	if mode == "" {
		mode = ModelTokenLimitModeUnlimited
	}
	if mode != ModelTokenLimitModeUnlimited && mode != ModelTokenLimitModeAvailable {
		return ModelPeriodTokenLimit{}, fmt.Errorf("%w: model token limit mode must be available or unlimited", ErrInvalidArgument)
	}
	return ModelPeriodTokenLimit{Tokens: p.Tokens, Mode: mode}, nil
}

func normalizeModelTokenLimits(limits []ModelTokenLimit) ([]ModelTokenLimit, error) {
	if len(limits) > maxModelTokenLimits {
		return nil, fmt.Errorf("%w: at most %d model token limits are allowed", ErrInvalidArgument, maxModelTokenLimits)
	}
	out := make([]ModelTokenLimit, 0, len(limits))
	seen := make(map[string]struct{}, len(limits))
	for _, item := range limits {
		model := strings.TrimSpace(item.Model)
		if model == "" {
			return nil, fmt.Errorf("%w: model token limit requires a model", ErrInvalidArgument)
		}
		if _, exists := seen[model]; exists {
			return nil, fmt.Errorf("%w: duplicate model token limit for %s", ErrInvalidArgument, model)
		}
		seen[model] = struct{}{}
		daily, err := item.Daily.normalize()
		if err != nil {
			return nil, err
		}
		weekly, err := item.Weekly.normalize()
		if err != nil {
			return nil, err
		}
		monthly, err := item.Monthly.normalize()
		if err != nil {
			return nil, err
		}
		out = append(out, ModelTokenLimit{Model: model, Daily: daily, Weekly: weekly, Monthly: monthly})
	}
	return out, nil
}

func marshalModelTokenLimits(limits []ModelTokenLimit) (string, error) {
	clean, err := normalizeModelTokenLimits(limits)
	if err != nil {
		return "", err
	}
	if clean == nil {
		clean = []ModelTokenLimit{}
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return "", fmt.Errorf("%w: encode model token limits: %v", ErrInvalidArgument, err)
	}
	return string(raw), nil
}

func unmarshalModelTokenLimits(raw string) ([]ModelTokenLimit, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []ModelTokenLimit{}, nil
	}
	var limits []ModelTokenLimit
	if err := json.Unmarshal([]byte(raw), &limits); err != nil {
		return nil, fmt.Errorf("%w: decode model token limits: %v", ErrInvalidArgument, err)
	}
	return normalizeModelTokenLimits(limits)
}

func modelMatchesPattern(pattern, model string) bool {
	pattern = strings.TrimSpace(pattern)
	model = strings.TrimSpace(model)
	if pattern == "" || model == "" {
		return false
	}
	if pattern == model {
		return true
	}
	ok, err := filepath.Match(pattern, model)
	return err == nil && ok
}

// MatchModelTokenLimit returns the most specific policy for model.
// Exact names win; otherwise the longest matching glob is used.
func MatchModelTokenLimit(limits []ModelTokenLimit, model string) (ModelTokenLimit, bool) {
	model = strings.TrimSpace(model)
	if model == "" || len(limits) == 0 {
		return ModelTokenLimit{}, false
	}
	var glob ModelTokenLimit
	globLen := -1
	for _, limit := range limits {
		pattern := strings.TrimSpace(limit.Model)
		if pattern == "" {
			continue
		}
		if pattern == model {
			return limit, true
		}
		if ok, err := filepath.Match(pattern, model); err == nil && ok && len(pattern) > globLen {
			glob = limit
			globLen = len(pattern)
		}
	}
	if globLen >= 0 {
		return glob, true
	}
	return ModelTokenLimit{}, false
}

func parseUnmatchedModelsMode(mode string) (string, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return UnmatchedModelsAvailable, nil
	}
	if mode != UnmatchedModelsAvailable && mode != UnmatchedModelsDisabled {
		return "", fmt.Errorf("%w: unmatched models mode must be available or disabled", ErrInvalidArgument)
	}
	return mode, nil
}

func normalizeUnmatchedModelsMode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), UnmatchedModelsDisabled) {
		return UnmatchedModelsDisabled
	}
	return UnmatchedModelsAvailable
}

func enforceModelTokenLimits(ctx context.Context, db tokenQueryer, keyID, model string, estimate int64, limits []ModelTokenLimit, unmatchedMode string, nowUnixMilli int64) error {
	matched, ok := MatchModelTokenLimit(limits, model)
	if !ok {
		if normalizeUnmatchedModelsMode(unmatchedMode) == UnmatchedModelsDisabled {
			return fmt.Errorf("%w: unmatched model %s is disabled", ErrModelNotAllowed, model)
		}
		return nil
	}
	if estimate < 0 {
		estimate = 0
	}
	if matched.Daily.Cap() <= 0 && matched.Weekly.Cap() <= 0 && matched.Monthly.Cap() <= 0 {
		return nil
	}
	totals, err := loadModelPeriodTokenTotals(ctx, db, keyID, nowUnixMilli)
	if err != nil {
		return err
	}
	for _, period := range []struct {
		cap  int64
		used int64
		err  error
	}{
		{matched.Daily.Cap(), sumMatchingModelTokens(totals.daily, matched.Model), ErrDailyTokenLimitExceeded},
		{matched.Weekly.Cap(), sumMatchingModelTokens(totals.weekly, matched.Model), ErrWeeklyTokenLimitExceeded},
		{matched.Monthly.Cap(), sumMatchingModelTokens(totals.monthly, matched.Model), ErrMonthlyTokenLimitExceeded},
	} {
		if period.cap <= 0 {
			continue
		}
		if period.used > period.cap-estimate {
			return period.err
		}
	}
	return nil
}

type modelPeriodTokenTotals struct {
	daily   map[string]int64
	weekly  map[string]int64
	monthly map[string]int64
}

func loadModelPeriodTokenTotals(ctx context.Context, db tokenQueryer, keyID string, nowUnixMilli int64) (modelPeriodTokenTotals, error) {
	dayStart, weekStart, monthStart, err := keyPeriodStarts(ctx, db, keyID, nowUnixMilli)
	if err != nil {
		return modelPeriodTokenTotals{}, err
	}
	from := dayStart
	if weekStart < from {
		from = weekStart
	}
	if monthStart < from {
		from = monthStart
	}
	tokenExpr := usageReportedTotalSQL("")
	settled, err := queryModelPeriodTokenTotals(ctx, db, `SELECT model,
		COALESCE(SUM(CASE WHEN created_at_unix_ms >= ? THEN (`+tokenExpr+`) ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN created_at_unix_ms >= ? THEN (`+tokenExpr+`) ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN created_at_unix_ms >= ? THEN (`+tokenExpr+`) ELSE 0 END), 0)
		FROM usage_ledger WHERE plugin_key_id = ? AND created_at_unix_ms >= ? GROUP BY model`,
		dayStart, weekStart, monthStart, keyID, from)
	if err != nil {
		return modelPeriodTokenTotals{}, fmt.Errorf("sum model period tokens: %w", err)
	}
	held, err := queryModelPeriodTokenTotals(ctx, db, `SELECT model,
		COALESCE(SUM(CASE WHEN created_at_unix_ms >= ? THEN request_token_estimate ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN created_at_unix_ms >= ? THEN request_token_estimate ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN created_at_unix_ms >= ? THEN request_token_estimate ELSE 0 END), 0)
		FROM reservations WHERE plugin_key_id = ? AND status = 'held' AND created_at_unix_ms >= ? GROUP BY model`,
		dayStart, weekStart, monthStart, keyID, from)
	if err != nil {
		return modelPeriodTokenTotals{}, fmt.Errorf("sum model period held tokens: %w", err)
	}
	for model, tokens := range held.daily {
		settled.daily[model] += tokens
	}
	for model, tokens := range held.weekly {
		settled.weekly[model] += tokens
	}
	for model, tokens := range held.monthly {
		settled.monthly[model] += tokens
	}
	return settled, nil
}

func queryModelPeriodTokenTotals(ctx context.Context, db tokenQueryer, query string, args ...any) (modelPeriodTokenTotals, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return modelPeriodTokenTotals{}, err
	}
	defer rows.Close()
	out := modelPeriodTokenTotals{
		daily:   make(map[string]int64),
		weekly:  make(map[string]int64),
		monthly: make(map[string]int64),
	}
	for rows.Next() {
		var model string
		var daily, weekly, monthly int64
		if err := rows.Scan(&model, &daily, &weekly, &monthly); err != nil {
			return modelPeriodTokenTotals{}, err
		}
		out.daily[model] += daily
		out.weekly[model] += weekly
		out.monthly[model] += monthly
	}
	return out, rows.Err()
}

func sumMatchingModelTokens(totals map[string]int64, pattern string) int64 {
	var total int64
	for model, tokens := range totals {
		if modelMatchesPattern(pattern, model) {
			total += tokens
		}
	}
	return total
}

func (s *Store) ListModelTokenUsage(ctx context.Context, keyID string, limits []ModelTokenLimit, nowUnixMilli int64) ([]ModelTokenUsage, error) {
	if strings.TrimSpace(keyID) == "" {
		return nil, fmt.Errorf("%w: plugin key id is required", ErrInvalidArgument)
	}
	clean, err := normalizeModelTokenLimits(limits)
	if err != nil {
		return nil, err
	}
	out := make([]ModelTokenUsage, 0, len(clean))
	if len(clean) == 0 {
		return out, nil
	}
	totals, err := loadModelPeriodTokenTotals(ctx, s.db, keyID, nowUnixMilli)
	if err != nil {
		return nil, err
	}
	for _, limit := range clean {
		out = append(out, ModelTokenUsage{
			Model:       limit.Model,
			Daily:       limit.Daily,
			Weekly:      limit.Weekly,
			Monthly:     limit.Monthly,
			DailyUsed:   sumMatchingModelTokens(totals.daily, limit.Model),
			WeeklyUsed:  sumMatchingModelTokens(totals.weekly, limit.Model),
			MonthlyUsed: sumMatchingModelTokens(totals.monthly, limit.Model),
		})
	}
	return out, nil
}
