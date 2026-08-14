package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
)

const (
	BootstrapCallerID      = "default"
	BootstrapCallerName    = "Default (auto)"
	BootstrapPricingRuleID = "bootstrap-all-models"
)

// ensureBootstrap seeds only the default ownership record and free pricing.
// Keys are always created explicitly through the management API.
func (s *Service) ensureBootstrap(ctx context.Context) error {
	if s == nil || s.store == nil {
		return errors.New("service not open")
	}
	if err := s.ensureBootstrapCaller(ctx); err != nil {
		return err
	}
	return s.ensureBootstrapPricing(ctx)
}

func (s *Service) ensureBootstrapCaller(ctx context.Context) error {
	if _, err := s.store.GetCaller(ctx, BootstrapCallerID); err == nil {
		return nil
	} else if !errors.Is(err, store.ErrCallerNotFound) {
		// GetCaller may wrap sql.ErrNoRows as ErrCallerNotFound — check both.
		if !isNotFound(err) {
			return fmt.Errorf("bootstrap caller lookup: %w", err)
		}
	}
	_, err := s.store.CreateCaller(ctx, store.CallerSpec{
		ID:            BootstrapCallerID,
		DisplayName:   BootstrapCallerName,
		QuotaMicroUSD: 0,
		Enabled:       true,
	})
	if err != nil {
		// Concurrent first-start: treat existing as success.
		if _, getErr := s.store.GetCaller(ctx, BootstrapCallerID); getErr == nil {
			return nil
		}
		return fmt.Errorf("bootstrap create caller: %w", err)
	}
	return nil
}

func (s *Service) ensureBootstrapPricing(ctx context.Context) error {
	rules, err := s.store.ListPricingRules(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap list pricing: %w", err)
	}
	if len(rules) > 0 {
		return nil
	}
	// Free match-all so any model works out of the box; operators can replace later.
	err = s.store.PutPricingRule(ctx, store.PricingRule{
		ID:        BootstrapPricingRuleID,
		MatchKind: store.MatchRegexp,
		Pattern:   ".*",
		Priority:  0,
		Price:     money.PricePerMTok{},
		Enabled:   true,
	})
	if err != nil {
		return fmt.Errorf("bootstrap pricing: %w", err)
	}
	return nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, store.ErrCallerNotFound) || errors.Is(err, store.ErrPluginKeyNotFound) {
		return true
	}
	// store scanners often wrap "sql: no rows in result set"
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no rows") || strings.Contains(msg, "not found")
}
