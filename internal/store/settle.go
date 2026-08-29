package store

import (
	"context"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/yuluo688/credit-manager/internal/money"
)

type Settlement struct {
	LedgerID              string
	ReservationID         string
	ExecutorType          string
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
		id, reservation_id, caller_id, plugin_key_id, executor_type, model, pricing_rule_id, input_tokens, output_tokens,
		reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, total_tokens, cost_micro_usd, estimated_cost_micro_usd, source,
		tier, result, first_token_latency_ms, generation_duration_ms, tokens_per_second, thinking_intensity,
		auth_id, auth_index, auth_name, auth_label, auth_provider, auth_type, auth_email, auth_path, created_at_unix_ms
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		settlement.LedgerID, settlement.ReservationID, reservation.CallerID, reservation.PluginKeyID, nullableString(settlement.ExecutorType), settlement.Model,
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
