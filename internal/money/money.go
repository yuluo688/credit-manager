package money

import (
	"errors"
	"fmt"
	"math"
)

const TokensPerMTok int64 = 1_000_000

var (
	ErrNegativeValue = errors.New("money and token values must not be negative")
	ErrOverflow      = errors.New("micro-USD calculation overflow")
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
}

// PricePerMTok is an integer micro-USD price for one million tokens.
type PricePerMTok struct {
	Input         MicroUSD `json:"input"`
	Output        MicroUSD `json:"output"`
	Reasoning     MicroUSD `json:"reasoning"`
	Cached        MicroUSD `json:"cached"`
	CacheRead     MicroUSD `json:"cache_read"`
	CacheCreation MicroUSD `json:"cache_creation"`
}

func (u TokenUsage) Validate() error {
	for name, value := range map[string]int64{
		"input": u.Input, "output": u.Output, "reasoning": u.Reasoning,
		"cached": u.Cached, "cache_read": u.CacheRead, "cache_creation": u.CacheCreation,
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
	} {
		if value < 0 {
			return fmt.Errorf("%w: %s price", ErrNegativeValue, name)
		}
	}
	return nil
}

// Cost calculates rounded-up micro-USD for each token class independently.
// Rounding each class upward ensures a positive priced usage is never free.
func Cost(usage TokenUsage, price PricePerMTok) (MicroUSD, error) {
	if err := usage.Validate(); err != nil {
		return 0, err
	}
	if err := price.Validate(); err != nil {
		return 0, err
	}

	pairs := []struct {
		tokens int64
		price  MicroUSD
	}{
		{usage.Input, price.Input},
		{usage.Output, price.Output},
		{usage.Reasoning, price.Reasoning},
		{usage.Cached, price.Cached},
		{usage.CacheRead, price.CacheRead},
		{usage.CacheCreation, price.CacheCreation},
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
