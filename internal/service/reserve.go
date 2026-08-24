package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/config"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
	"github.com/yuluo688/credit-manager/internal/usageparse"
)

type ReservePlan struct {
	Model          string
	InputEstimate  int64
	OutputEstimate int64
	TokenEstimate  int64
	ImageCount     int64
	Price          money.PricePerMTok
	PricingRuleID  *string
	Amount         money.MicroUSD
	AllowUnpriced  bool
}

func (s *Service) BuildReservePlan(ctx context.Context, model string, body []byte) (ReservePlan, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		model = extractModel(body)
	}
	if model == "" {
		return ReservePlan{}, fmt.Errorf("%w: model is required", store.ErrInvalidArgument)
	}
	plan := ReservePlan{Model: model}
	rule, err := s.store.ResolvePricingRule(ctx, model)
	switch {
	case err == nil:
		if !rule.Enabled {
			return ReservePlan{}, fmt.Errorf("%w: %s", store.ErrModelDisabled, model)
		}
		plan.Price = rule.Price
		id := rule.ID
		plan.PricingRuleID = &id
	case errors.Is(err, store.ErrPricingRuleNotFound):
		switch s.cfg.Pricing.UnknownPolicy {
		case config.UnknownPricingDeny:
			return ReservePlan{}, fmt.Errorf("no pricing rule for model %q", model)
		case config.UnknownPricingAllow:
			plan.AllowUnpriced = true
			plan.Amount = 0
			return plan, nil
		case config.UnknownPricingDefault:
			if s.cfg.Pricing.Default == nil {
				return ReservePlan{}, errors.New("default pricing missing")
			}
			plan.Price = money.PricePerMTok{
				Input: money.MicroUSD(s.cfg.Pricing.Default.Input), Output: money.MicroUSD(s.cfg.Pricing.Default.Output),
				Reasoning: money.MicroUSD(s.cfg.Pricing.Default.Reasoning), Cached: money.MicroUSD(s.cfg.Pricing.Default.Cached),
				CacheRead: money.MicroUSD(s.cfg.Pricing.Default.CacheRead), CacheCreation: money.MicroUSD(s.cfg.Pricing.Default.CacheCreation),
			}
		}
	default:
		return ReservePlan{}, err
	}
	if plan.Price.IsPerImage() {
		images := extractImageCount(body)
		plan.ImageCount = images
		plan.TokenEstimate = images
		cost, err := money.Cost(money.TokenUsage{Images: images}, plan.Price)
		if err != nil {
			return ReservePlan{}, err
		}
		plan.Amount = cost
		return plan, nil
	}
	inputEst, outputEst := estimateTokens(body, s.cfg.Limits.DefaultOutputReserve, s.cfg.Limits.MaxTokenEstimate)
	if err := s.cfg.ValidateTokenEstimate(inputEst + outputEst); err != nil {
		return ReservePlan{}, err
	}
	plan.InputEstimate = inputEst
	plan.OutputEstimate = outputEst
	plan.TokenEstimate = inputEst + outputEst
	cost, err := money.CostFor(money.TokenUsage{Input: inputEst, Output: outputEst}, plan.Price, plan.Model, "")
	if err != nil {
		return ReservePlan{}, err
	}
	plan.Amount = cost
	return plan, nil
}

func (s *Service) Reserve(ctx context.Context, key store.PluginKey, plan ReservePlan, idempotency string) (store.Reservation, error) {
	if _, err := s.cleanupStaleReservations(ctx, false); err != nil {
		return store.Reservation{}, fmt.Errorf("release stale reservations: %w", err)
	}
	if idempotency == "" {
		idempotency = newIdempotency(key.ID, plan.Model, plan.TokenEstimate, plan.Amount)
	}
	return s.store.Reserve(ctx, store.ReserveRequest{
		CallerID:             key.CallerID,
		PluginKeyID:          key.ID,
		IdempotencyKey:       idempotency,
		Model:                plan.Model,
		RequestTokenEstimate: plan.TokenEstimate,
		AmountMicroUSD:       plan.Amount,
		RequestSummary:       reserveSummary(plan),
	})
}

func (s *Service) cleanupStaleReservations(ctx context.Context, force bool) (int64, error) {
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()
	now := time.Now().UTC()
	if !force && !s.lastCleanup.IsZero() && now.Sub(s.lastCleanup) < staleCleanupInterval {
		return 0, nil
	}
	released, err := s.store.ReleaseStaleReservations(ctx, now.Add(-s.cfg.Stream.StaleReservationTimeout))
	if err != nil {
		return 0, err
	}
	s.lastCleanup = now
	return released, nil
}

func (s *Service) TouchReservation(ctx context.Context, reservationID string) error {
	return s.store.TouchReservation(ctx, reservationID)
}

func (s *Service) SettleFromUsage(ctx context.Context, reservation store.Reservation, plan ReservePlan, parsed usageparse.Result, format string, metrics store.UsageMetrics) error {
	if hostUsage, ok := s.CapturedHostUsage(reservation.ID); ok {
		return s.settleResolvedUsage(ctx, reservation, plan, hostUsage, "host_usage", "host_usage_callback", metrics)
	}
	if parsed.Found {
		return s.settleResolvedUsage(ctx, reservation, plan, parsed.Usage, parsed.Source, fmt.Sprintf("format=%s source=%s", format, parsed.Source), metrics)
	}
	if s.cfg.Settlement.MissingUsage != config.MissingUsageRelease && s.cfg.Settlement.HostUsageWait > 0 {
		if hostUsage, ok := s.WaitForHostUsage(ctx, reservation.ID, s.cfg.Settlement.HostUsageWait); ok {
			return s.settleResolvedUsage(ctx, reservation, plan, hostUsage, "host_usage", "host_usage_callback", metrics)
		}
	}
	switch s.cfg.Settlement.MissingUsage {
	case config.MissingUsageRelease:
		s.CancelAuthCapture(reservation.ID)
		_, err := s.store.Release(ctx, reservation.ID, "missing_usage")
		return err
	default:
		if plan.Price.IsPerImage() {
			return s.settleResolvedUsage(ctx, reservation, plan, money.TokenUsage{Images: plan.ImageCount}, "per_image", "missing_usage_settle_per_image", metrics)
		}
		// Do not bill max_tokens / body-length estimates as actual spend.
		// Keep a ledger row so a later usage.handle can reprice from official tokens.
		return s.settleWithAuth(ctx, store.Settlement{
			ReservationID:         reservation.ID,
			Model:                 plan.Model,
			PricingRuleID:         plan.PricingRuleID,
			Usage:                 money.TokenUsage{},
			CostMicroUSD:          0,
			EstimatedCostMicroUSD: reservation.HeldMicroUSD,
			Source:                "reserved_fallback",
			Metrics:               metrics,
			SettlementSummary:     "missing_usage_pending_host",
		})
	}
}

func (s *Service) settleResolvedUsage(ctx context.Context, reservation store.Reservation, plan ReservePlan, usage money.TokenUsage, source, summary string, metrics store.UsageMetrics) error {
	if plan.Price.IsPerImage() && usage.Images <= 0 {
		usage.Images = plan.ImageCount
	}
	cost, err := money.CostFor(usage, plan.Price, plan.Model, "")
	if err != nil {
		return err
	}
	if plan.AllowUnpriced {
		cost = 0
	}
	metrics.TokensPerSecond = tokensPerSecond(usage.Output, metrics.GenerationDuration)
	return s.settleWithAuth(ctx, store.Settlement{
		ReservationID:         reservation.ID,
		Model:                 plan.Model,
		PricingRuleID:         plan.PricingRuleID,
		Usage:                 usage,
		CostMicroUSD:          cost,
		EstimatedCostMicroUSD: reservation.HeldMicroUSD,
		Source:                source,
		Metrics:               metrics,
		SettlementSummary:     summary,
	})
}

func (s *Service) ApplyHostUsage(ctx context.Context, ledgerID string, usage money.TokenUsage) error {
	if !usageFound(usage) && usage.Images <= 0 {
		return nil
	}
	entry, err := s.store.GetUsage(ctx, ledgerID)
	if err != nil {
		return err
	}
	price, err := s.priceForUsage(ctx, entry)
	if err != nil {
		return err
	}
	if price.IsPerImage() && usage.Images <= 0 {
		return s.store.UpdateUsageDetail(ctx, ledgerID, usage, entry.CostMicroUSD)
	}
	cost, err := money.CostFor(usage, price, entry.Model, entry.Auth.Provider)
	if err != nil {
		return err
	}
	if price == (money.PricePerMTok{}) && entry.PricingRuleID == nil {
		cost = 0
	}
	return s.store.UpdateUsageDetail(ctx, ledgerID, usage, cost)
}

func (s *Service) priceForUsage(ctx context.Context, entry store.UsageEntry) (money.PricePerMTok, error) {
	if entry.PricingRuleID != nil {
		if rule, err := s.store.GetPricingRule(ctx, *entry.PricingRuleID); err == nil {
			return rule.Price, nil
		} else if !errors.Is(err, store.ErrPricingRuleNotFound) {
			return money.PricePerMTok{}, err
		}
	}
	rule, err := s.store.ResolvePricingRule(ctx, entry.Model)
	if err == nil {
		return rule.Price, nil
	}
	if !errors.Is(err, store.ErrPricingRuleNotFound) {
		return money.PricePerMTok{}, err
	}
	if s.cfg.Pricing.UnknownPolicy == config.UnknownPricingDefault && s.cfg.Pricing.Default != nil {
		return money.PricePerMTok{
			Input: money.MicroUSD(s.cfg.Pricing.Default.Input), Output: money.MicroUSD(s.cfg.Pricing.Default.Output),
			Reasoning: money.MicroUSD(s.cfg.Pricing.Default.Reasoning), Cached: money.MicroUSD(s.cfg.Pricing.Default.Cached),
			CacheRead: money.MicroUSD(s.cfg.Pricing.Default.CacheRead), CacheCreation: money.MicroUSD(s.cfg.Pricing.Default.CacheCreation),
		}, nil
	}
	return money.PricePerMTok{}, nil
}

func (s *Service) settleWithAuth(ctx context.Context, settlement store.Settlement) error {
	if strings.TrimSpace(settlement.LedgerID) == "" {
		settlement.LedgerID = store.NewID()
	}
	settlement.Auth = s.AuthForSettlement(settlement.ReservationID, settlement.LedgerID)
	_, err := s.store.Settle(ctx, settlement)
	return err
}

func tokensPerSecond(output int64, generationDuration *time.Duration) *float64 {
	if output <= 0 || generationDuration == nil || *generationDuration <= 0 {
		return nil
	}
	value := float64(output) / generationDuration.Seconds()
	return &value
}

func (s *Service) Release(ctx context.Context, reservationID, reason string) error {
	s.CancelAuthCapture(reservationID)
	_, err := s.store.Release(ctx, reservationID, reason)
	return err
}

func reserveSummary(plan ReservePlan) string {
	if plan.Price.IsPerImage() {
		return fmt.Sprintf("model=%s images=%d", plan.Model, plan.ImageCount)
	}
	return fmt.Sprintf("model=%s in=%d out=%d", plan.Model, plan.InputEstimate, plan.OutputEstimate)
}

func newIdempotency(keyID, model string, tokens int64, amount money.MicroUSD) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d|%d|%d", keyID, model, tokens, amount, time.Now().UnixNano())))
	return hex.EncodeToString(sum[:16])
}
