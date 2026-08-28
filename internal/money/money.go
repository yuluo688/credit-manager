package money

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

const TokensPerMTok int64 = 1_000_000

const (
	// AccountingInputExcludesCache bills input and cache counters independently.
	// This matches Anthropic: input_tokens does not include cache tokens.
	AccountingInputExcludesCache = "input_excludes_cache"
	// AccountingInputIncludesCache subtracts cache tokens from input before
	// billing. This matches OpenAI-compatible usage: prompt_tokens already
	// includes cached tokens.
	AccountingInputIncludesCache = "input_includes_cache"

	// BillingToken charges USD per million tokens.
	BillingToken = "token"
	// BillingPerImage charges a flat USD price per generated image.
	BillingPerImage = "per_image"
)

var (
	ErrNegativeValue = errors.New("money and token values must not be negative")
	ErrOverflow      = errors.New("micro-USD calculation overflow")
	ErrAccounting    = errors.New("invalid accounting mode")
	ErrBilling       = errors.New("invalid billing mode")
)

// MicroUSD is the only persisted monetary unit. Floating-point currency is
// intentionally excluded from the ledger boundary.
type MicroUSD int64

// TokenUsage separates token classes because each can have a distinct price.
type TokenUsage struct {
	Input         int64 `json:"input"`
	Output        int64 `json:"output"`
	Reasoning     int64 `json:"reasoning"`
	Cached        int64 `json:"cached"`
	CacheRead     int64 `json:"cache_read"`
	CacheCreation int64 `json:"cache_creation"`
	Images        int64 `json:"images,omitempty"`
	ReportedTotal int64 `json:"reported_total,omitempty"`
}

// PricePerMTok is an integer micro-USD price for one million tokens.
type PricePerMTok struct {
	Input          MicroUSD `json:"input"`
	Output         MicroUSD `json:"output"`
	Reasoning      MicroUSD `json:"reasoning"`
	Cached         MicroUSD `json:"cached"`
	CacheRead      MicroUSD `json:"cache_read"`
	CacheCreation  MicroUSD `json:"cache_creation"`
	AccountingMode string   `json:"accounting_mode,omitempty"`
	BillingMode    string   `json:"billing_mode,omitempty"`
	PerImage       MicroUSD `json:"per_image,omitempty"`
}

// BillableTokens is the de-duplicated usage actually sent to the price table.
// It follows cap-token-usage-tracker: four components, no extra reasoning line.
type BillableTokens struct {
	Input         int64
	Output        int64
	CacheRead     int64
	CacheCreation int64
	Mode          string
}

func (u TokenUsage) HasTokens() bool {
	return u.Input > 0 || u.Output > 0 || u.Reasoning > 0 || u.Cached > 0 ||
		u.CacheRead > 0 || u.CacheCreation > 0 || u.Images > 0 || u.ReportedTotal > 0
}

func (u TokenUsage) Validate() error {
	for name, value := range map[string]int64{
		"input": u.Input, "output": u.Output, "reasoning": u.Reasoning,
		"cached": u.Cached, "cache_read": u.CacheRead, "cache_creation": u.CacheCreation,
		"images": u.Images, "reported_total": u.ReportedTotal,
	} {
		if value < 0 {
			return fmt.Errorf("%w: %s tokens", ErrNegativeValue, name)
		}
	}
	return nil
}

func (p PricePerMTok) Validate() error {
	for name, value := range map[string]MicroUSD{
		"input": p.Input, "output": p.Output, "reasoning": p.Reasoning,
		"cached": p.Cached, "cache_read": p.CacheRead, "cache_creation": p.CacheCreation,
		"per_image": p.PerImage,
	} {
		if value < 0 {
			return fmt.Errorf("%w: %s price", ErrNegativeValue, name)
		}
	}
	switch strings.TrimSpace(p.BillingMode) {
	case "", BillingToken, BillingPerImage:
	default:
		return fmt.Errorf("%w: %q", ErrBilling, p.BillingMode)
	}
	switch strings.TrimSpace(p.AccountingMode) {
	case "", AccountingInputExcludesCache, AccountingInputIncludesCache:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrAccounting, p.AccountingMode)
	}
}

// ResolveBillingMode treats an empty mode as token billing.
func ResolveBillingMode(mode string) string {
	if strings.TrimSpace(mode) == BillingPerImage {
		return BillingPerImage
	}
	return BillingToken
}

func (p PricePerMTok) IsPerImage() bool {
	return ResolveBillingMode(p.BillingMode) == BillingPerImage
}

// ReportedTotal matches cap-token-usage-tracker: prefer the official
// total_tokens, otherwise input+output+reasoning, and only then cached.
// Cache counters are never added on top of input.
func ReportedTotal(usage TokenUsage) int64 {
	if usage.ReportedTotal > 0 {
		return usage.ReportedTotal
	}
	total := usage.Input + usage.Output + usage.Reasoning
	if total > 0 {
		return total
	}
	return usage.Cached
}

// DefaultAccountingMode matches cap-token-usage-tracker: Anthropic/Claude
// exclude cache from input; every other provider includes cache in input.
func DefaultAccountingMode(model, provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	model = strings.ToLower(strings.TrimSpace(model))
	if provider == "anthropic" || provider == "claude" || strings.Contains(model, "claude") || strings.Contains(model, "anthropic") {
		return AccountingInputExcludesCache
	}
	return AccountingInputIncludesCache
}

// ResolveAccountingMode keeps an explicit price-book mode and otherwise infers
// from the model or provider identity.
func ResolveAccountingMode(mode, model, provider string) string {
	switch strings.TrimSpace(mode) {
	case AccountingInputExcludesCache, AccountingInputIncludesCache:
		return strings.TrimSpace(mode)
	default:
		return DefaultAccountingMode(model, provider)
	}
}

// Billable applies tracker cache accounting. CacheRead falls back to Cached.
// Cached and Reasoning are not billed as their own classes.
func Billable(usage TokenUsage, mode string) BillableTokens {
	cacheRead := usage.CacheRead
	if cacheRead == 0 {
		cacheRead = usage.Cached
	}
	cacheCreation := usage.CacheCreation
	billableInput := usage.Input
	if mode == AccountingInputIncludesCache {
		cacheTokens := saturatingAdd(cacheRead, cacheCreation)
		if billableInput > cacheTokens {
			billableInput -= cacheTokens
		} else {
			billableInput = 0
		}
	}
	return BillableTokens{
		Input:         billableInput,
		Output:        usage.Output,
		CacheRead:     cacheRead,
		CacheCreation: cacheCreation,
		Mode:          mode,
	}
}

// CostFor resolves the accounting mode from an explicit price setting or the
// model/provider identity, then bills the request.
func CostFor(usage TokenUsage, price PricePerMTok, model, provider string) (MicroUSD, error) {
	price.AccountingMode = ResolveAccountingMode(price.AccountingMode, model, provider)
	return Cost(usage, price)
}

// Cost calculates rounded-up micro-USD using the same four billable counters as
// cap-token-usage-tracker. Each class is rounded upward so a positive priced
// usage is never free.
func Cost(usage TokenUsage, price PricePerMTok) (MicroUSD, error) {
	if err := usage.Validate(); err != nil {
		return 0, err
	}
	if err := price.Validate(); err != nil {
		return 0, err
	}
	if price.IsPerImage() {
		return costPerImage(usage.Images, price.PerImage)
	}

	mode := ResolveAccountingMode(price.AccountingMode, "", "")
	billable := Billable(usage, mode)
	cacheReadPrice := price.CacheRead
	if cacheReadPrice == 0 {
		cacheReadPrice = price.Cached
	}

	pairs := []struct {
		tokens int64
		price  MicroUSD
	}{
		{billable.Input, price.Input},
		{billable.Output, price.Output},
		{billable.CacheRead, cacheReadPrice},
		{billable.CacheCreation, price.CacheCreation},
	}

	var total int64
	for _, pair := range pairs {
		part, err := mulDivCeil(pair.tokens, int64(pair.price), TokensPerMTok)
		if err != nil {
			return 0, err
		}
		if part > math.MaxInt64-total {
			return 0, ErrOverflow
		}
		total += part
	}
	return MicroUSD(total), nil
}

func costPerImage(images int64, price MicroUSD) (MicroUSD, error) {
	if images < 0 || price < 0 {
		return 0, ErrNegativeValue
	}
	if images <= 0 {
		images = 1
	}
	if price == 0 {
		return 0, nil
	}
	if int64(price) > math.MaxInt64/images {
		return 0, ErrOverflow
	}
	return MicroUSD(images * int64(price)), nil
}

func saturatingAdd(left, right int64) int64 {
	if left < 0 {
		left = 0
	}
	if right < 0 {
		right = 0
	}
	if left > math.MaxInt64-right {
		return math.MaxInt64
	}
	return left + right
}

func mulDivCeil(a, b, divisor int64) (int64, error) {
	if a < 0 || b < 0 || divisor <= 0 {
		return 0, ErrNegativeValue
	}
	if a == 0 || b == 0 {
		return 0, nil
	}
	quotient, remainder := a/divisor, a%divisor
	if quotient > math.MaxInt64/b {
		return 0, ErrOverflow
	}
	result := quotient * b
	if remainder > math.MaxInt64/b {
		return 0, ErrOverflow
	}
	fraction := remainder * b
	add := fraction / divisor
	if fraction%divisor != 0 {
		add++
	}
	if add > math.MaxInt64-result {
		return 0, ErrOverflow
	}
	return result + add, nil
}
