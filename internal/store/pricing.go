package store

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yuluo688/credit-manager/internal/money"
)

type MatchKind string

const (
	MatchExact  MatchKind = "exact"
	MatchGlob   MatchKind = "glob"
	MatchRegexp MatchKind = "regexp"
)

const pricingRuleColumns = `id, match_kind, pattern, priority,
		input_per_mtok_micro_usd, output_per_mtok_micro_usd, reasoning_per_mtok_micro_usd,
		cached_per_mtok_micro_usd, cache_read_per_mtok_micro_usd, cache_creation_per_mtok_micro_usd,
		accounting_mode, billing_mode, per_image_micro_usd, tiers_json, enabled, created_at_unix_ms, updated_at_unix_ms`

type PricingRule struct {
	ID        string             `json:"id"`
	MatchKind MatchKind          `json:"match_kind"`
	Pattern   string             `json:"pattern"`
	Priority  int                `json:"priority"`
	Price     money.PricePerMTok `json:"price"`
	Tiers     []money.PriceTier  `json:"tiers,omitempty"`
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
	if _, err := money.NormalizeTiers(r.Tiers); err != nil {
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
	tiers, err := money.NormalizeTiers(rule.Tiers)
	if err != nil {
		return err
	}
	tiersJSON, err := marshalTiers(tiers)
	if err != nil {
		return err
	}
	now := nowUnixMilli()
	_, err = s.db.ExecContext(ctx, `INSERT INTO pricing_rules(
		id, match_kind, pattern, priority, input_per_mtok_micro_usd, output_per_mtok_micro_usd,
		reasoning_per_mtok_micro_usd, cached_per_mtok_micro_usd, cache_read_per_mtok_micro_usd,
		cache_creation_per_mtok_micro_usd, accounting_mode, billing_mode, per_image_micro_usd,
		tiers_json, enabled, created_at_unix_ms, updated_at_unix_ms
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET match_kind=excluded.match_kind, pattern=excluded.pattern,
		priority=excluded.priority, input_per_mtok_micro_usd=excluded.input_per_mtok_micro_usd,
		output_per_mtok_micro_usd=excluded.output_per_mtok_micro_usd,
		reasoning_per_mtok_micro_usd=excluded.reasoning_per_mtok_micro_usd,
		cached_per_mtok_micro_usd=excluded.cached_per_mtok_micro_usd,
		cache_read_per_mtok_micro_usd=excluded.cache_read_per_mtok_micro_usd,
		cache_creation_per_mtok_micro_usd=excluded.cache_creation_per_mtok_micro_usd,
		accounting_mode=excluded.accounting_mode, billing_mode=excluded.billing_mode,
		per_image_micro_usd=excluded.per_image_micro_usd,
		tiers_json=excluded.tiers_json,
		enabled=excluded.enabled, updated_at_unix_ms=excluded.updated_at_unix_ms`,
		rule.ID, rule.MatchKind, rule.Pattern, rule.Priority, rule.Price.Input, rule.Price.Output,
		rule.Price.Reasoning, rule.Price.Cached, rule.Price.CacheRead, rule.Price.CacheCreation,
		rule.Price.AccountingMode, rule.Price.BillingMode, rule.Price.PerImage, tiersJSON,
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

func (s *Store) SetPricingRuleEnabled(ctx context.Context, id string, enabled bool) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: pricing rule id is required", ErrInvalidArgument)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE pricing_rules SET enabled = ?, updated_at_unix_ms = ? WHERE id = ?`,
		boolInt(enabled), nowUnixMilli(), id)
	if err != nil {
		return fmt.Errorf("set pricing rule enabled: %w", err)
	}
	return requireOneRow(result, ErrPricingRuleNotFound)
}

func (s *Store) ListPricingRules(ctx context.Context) ([]PricingRule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+pricingRuleColumns+`
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
	return scanPricingRule(s.db.QueryRowContext(ctx, `SELECT `+pricingRuleColumns+`
		FROM pricing_rules WHERE id = ?`, id))
}

func (s *Store) ResolvePricingRule(ctx context.Context, model string) (PricingRule, error) {
	if strings.TrimSpace(model) == "" {
		return PricingRule{}, fmt.Errorf("%w: model is required", ErrInvalidArgument)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+pricingRuleColumns+`
		FROM pricing_rules ORDER BY priority DESC, id ASC`)
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

func WinningPricingRule(rules []PricingRule, model string) (PricingRule, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return PricingRule{}, fmt.Errorf("%w: model is required", ErrInvalidArgument)
	}
	sorted := append([]PricingRule(nil), rules...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		return sorted[i].ID < sorted[j].ID
	})
	for _, rule := range sorted {
		matched, err := ruleMatches(rule, model)
		if err != nil {
			return PricingRule{}, err
		}
		if matched {
			return rule, nil
		}
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

func marshalTiers(tiers []money.PriceTier) (string, error) {
	if len(tiers) == 0 {
		return "[]", nil
	}
	raw, err := json.Marshal(tiers)
	if err != nil {
		return "", fmt.Errorf("%w: encode pricing tiers: %v", ErrInvalidArgument, err)
	}
	return string(raw), nil
}

func unmarshalTiers(raw string) ([]money.PriceTier, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil, nil
	}
	var tiers []money.PriceTier
	if err := json.Unmarshal([]byte(raw), &tiers); err != nil {
		return nil, fmt.Errorf("%w: decode pricing tiers: %v", ErrInvalidArgument, err)
	}
	return money.NormalizeTiers(tiers)
}
