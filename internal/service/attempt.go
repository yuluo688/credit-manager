package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
)

const (
	SourceAttemptFailed    = "attempt_failed"
	SourceAttemptRejected  = "attempt_rejected"
	SourceAttemptCancelled = "attempt_cancelled"
)

// SettleZeroCostAttempt finalizes a held reservation into a zero-cost usage_ledger
// row. Used for post-reserve failures that previously only Release'd (upstream
// error, concurrency reject after admit, client disconnect, image failure).
// Ignores settlement.missing_usage so failures always appear in usage stats.
func (s *Service) SettleZeroCostAttempt(ctx context.Context, reservation store.Reservation, model string, metrics store.UsageMetrics, source, summary string) (err error) {
	ctx, cancel := accountingContext(ctx)
	defer cancel()
	defer func() {
		if err != nil {
			log.Printf("credit-manager: settle zero-cost attempt %s: %v", reservation.ID, err)
		}
	}()
	_ = s.FinishExecution(ctx, reservation.ID)
	model = strings.TrimSpace(model)
	if model == "" {
		model = strings.TrimSpace(reservation.Model)
	}
	if model == "" {
		return fmt.Errorf("%w: model is required", store.ErrInvalidArgument)
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = SourceAttemptFailed
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = source
	}
	if metrics.Result == nil || strings.TrimSpace(*metrics.Result) == "" {
		failed := "failed"
		metrics.Result = &failed
	}
	settlement := store.Settlement{
		ReservationID:         reservation.ID,
		Model:                 model,
		Usage:                 money.TokenUsage{},
		CostMicroUSD:          0,
		EstimatedCostMicroUSD: reservation.HeldMicroUSD,
		Source:                source,
		Metrics:               metrics,
		SettlementSummary:     summary,
	}
	return s.settleWithAuth(ctx, settlement)
}

// RecordRejectedAttempt records a pre-reserve rejection into usage_ledger with
// zero cost (synthetic settled reservation, no quota hold).
func (s *Service) RecordRejectedAttempt(ctx context.Context, key store.PluginKey, model, result, source, summary string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		model = "unknown"
	}
	result = strings.TrimSpace(result)
	if result == "" {
		result = "failed"
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = SourceAttemptRejected
	}
	_, err := s.store.InsertRejectedAttempt(ctx, store.RejectedAttempt{
		CallerID:    key.CallerID,
		PluginKeyID: key.ID,
		Model:       model,
		Result:      result,
		Source:      source,
		Summary:     summary,
	})
	return err
}

// AttemptMetrics builds UsageMetrics with a result label.
func AttemptMetrics(result string) store.UsageMetrics {
	result = strings.TrimSpace(result)
	if result == "" {
		result = "failed"
	}
	return store.UsageMetrics{Result: &result}
}
