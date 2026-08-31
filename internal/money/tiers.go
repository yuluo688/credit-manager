package money

import (
	"fmt"
	"strings"
)

const (
	PriceTierContext = "context"
	PriceTierService = "service"
	maxPriceTiers    = 16
)

// PriceTier is an optional rate card selected at settlement from actual usage
// and the upstream service_tier. Zero overlay rates inherit the base card.
type PriceTier struct {
	Kind      string       `json:"kind"`
	Label     string       `json:"label,omitempty"`
	Threshold int64        `json:"threshold,omitempty"`
	Service   string       `json:"service,omitempty"`
	Price     PricePerMTok `json:"price"`
}

func (t PriceTier) Validate() error {
	switch strings.ToLower(strings.TrimSpace(t.Kind)) {
	case PriceTierContext:
		if t.Threshold <= 0 {
			return fmt.Errorf("%w: context price tier threshold must be positive", ErrPriceTier)
		}
	case PriceTierService:
		if strings.TrimSpace(t.Service) == "" {
			return fmt.Errorf("%w: service price tier name is required", ErrPriceTier)
		}
	default:
		return fmt.Errorf("%w: kind %q", ErrPriceTier, t.Kind)
	}
	return t.Price.Validate()
}

func NormalizeTiers(tiers []PriceTier) ([]PriceTier, error) {
	if len(tiers) == 0 {
		return nil, nil
	}
	if len(tiers) > maxPriceTiers {
		return nil, fmt.Errorf("%w: at most %d price tiers", ErrPriceTier, maxPriceTiers)
	}
	out := make([]PriceTier, 0, len(tiers))
	for _, tier := range tiers {
		tier.Kind = strings.ToLower(strings.TrimSpace(tier.Kind))
		tier.Label = strings.TrimSpace(tier.Label)
		tier.Service = strings.ToLower(strings.TrimSpace(tier.Service))
		if tier.Kind == "" && tier.Threshold <= 0 && tier.Service == "" && tier.Price == (PricePerMTok{}) {
			continue
		}
		if err := tier.Validate(); err != nil {
			return nil, err
		}
		out = append(out, tier)
	}
	return out, nil
}

// NormalizeServiceTier maps upstream service_tier values onto billing keys.
// default / auto / standard mean "no special service card".
func NormalizeServiceTier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	switch value {
	case "", "default", "auto", "standard", "on-demand", "scale":
		return ""
	case "prio":
		return "priority"
	default:
		return value
	}
}

// ContextTokens is the prompt size used to match a context price tier.
// OpenAI-style input already includes cache; Anthropic-style input does not.
func ContextTokens(usage TokenUsage, accountingMode string) int64 {
	cacheRead := usage.CacheRead
	if cacheRead == 0 {
		cacheRead = usage.Cached
	}
	if ResolveAccountingMode(accountingMode, "", "") == AccountingInputExcludesCache {
		return saturatingAdd(usage.Input, saturatingAdd(cacheRead, usage.CacheCreation))
	}
	if usage.Input > 0 {
		return usage.Input
	}
	return saturatingAdd(cacheRead, usage.CacheCreation)
}

// SelectPrice picks the rate card for a request.
// A matching service_tier card wins; otherwise the highest matching context
// threshold is used; otherwise the base price.
func SelectPrice(base PricePerMTok, tiers []PriceTier, usage TokenUsage, serviceTier string) PricePerMTok {
	if base.IsPerImage() || len(tiers) == 0 {
		return base
	}
	if service := NormalizeServiceTier(serviceTier); service != "" {
		for _, tier := range tiers {
			if strings.ToLower(strings.TrimSpace(tier.Kind)) != PriceTierService {
				continue
			}
			if serviceMatches(tier.Service, service) {
				return overlayRates(base, tier.Price)
			}
		}
	}
	tokens := ContextTokens(usage, base.AccountingMode)
	var best *PriceTier
	for i := range tiers {
		tier := &tiers[i]
		if strings.ToLower(strings.TrimSpace(tier.Kind)) != PriceTierContext {
			continue
		}
		if tier.Threshold > 0 && tokens >= tier.Threshold {
			if best == nil || tier.Threshold > best.Threshold {
				best = tier
			}
		}
	}
	if best != nil {
		return overlayRates(base, best.Price)
	}
	return base
}

func CostForTiered(usage TokenUsage, base PricePerMTok, tiers []PriceTier, model, provider, serviceTier string) (MicroUSD, error) {
	return CostFor(usage, SelectPrice(base, tiers, usage, serviceTier), model, provider)
}

// ServiceTiers returns only service_tier cards so reserve holds do not use
// crude token estimates against context thresholds.
func ServiceTiers(tiers []PriceTier) []PriceTier {
	if len(tiers) == 0 {
		return nil
	}
	out := make([]PriceTier, 0, len(tiers))
	for _, tier := range tiers {
		if strings.ToLower(strings.TrimSpace(tier.Kind)) == PriceTierService {
			out = append(out, tier)
		}
	}
	return out
}

func overlayRates(base, overlay PricePerMTok) PricePerMTok {
	out := base
	if overlay.Input != 0 {
		out.Input = overlay.Input
	}
	if overlay.Output != 0 {
		out.Output = overlay.Output
	}
	if overlay.Reasoning != 0 {
		out.Reasoning = overlay.Reasoning
	}
	if overlay.Cached != 0 {
		out.Cached = overlay.Cached
	}
	if overlay.CacheRead != 0 {
		out.CacheRead = overlay.CacheRead
	}
	if overlay.CacheCreation != 0 {
		out.CacheCreation = overlay.CacheCreation
	}
	return out
}

func serviceMatches(configured, actual string) bool {
	actual = NormalizeServiceTier(actual)
	if actual == "" {
		return false
	}
	for _, part := range strings.Split(configured, ",") {
		if NormalizeServiceTier(part) == actual {
			return true
		}
	}
	return false
}
