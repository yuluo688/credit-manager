package money

import "testing"

func TestCostAnthropicUsesAllFourComponents(t *testing.T) {
	usage := TokenUsage{
		Input:         100_000,
		Output:        10_000,
		Cached:        99_000,
		CacheRead:     20_000,
		CacheCreation: 5_000,
	}
	price := PricePerMTok{
		Input: 2_000_000, Output: 10_000_000, CacheRead: 200_000, CacheCreation: 2_500_000,
		AccountingMode: AccountingInputExcludesCache,
	}
	got, err := Cost(usage, price)
	if err != nil {
		t.Fatal(err)
	}
	// 0.2 + 0.1 + 0.004 + 0.0125 USD = 316_500 micro-USD
	if got != 316_500 {
		t.Fatalf("anthropic cost = %d, want 316500", got)
	}
	billable := Billable(usage, AccountingInputExcludesCache)
	if billable.Input != 100_000 || billable.CacheRead != 20_000 || billable.CacheCreation != 5_000 {
		t.Fatalf("anthropic billable = %+v", billable)
	}
}

func TestCostInputIncludesCacheAndCachedFallback(t *testing.T) {
	usage := TokenUsage{
		Input:         100_000,
		Output:        5_000,
		Cached:        20_000,
		CacheCreation: 5_000,
	}
	price := PricePerMTok{
		Input: 1_000_000, Output: 2_000_000, CacheRead: 100_000, CacheCreation: 1_000_000,
		AccountingMode: AccountingInputIncludesCache,
	}
	got, err := Cost(usage, price)
	if err != nil {
		t.Fatal(err)
	}
	// billable input 75k + output 5k + cache read 20k + cache create 5k
	// 0.075 + 0.01 + 0.002 + 0.005 USD = 92_000 micro-USD
	if got != 92_000 {
		t.Fatalf("openai cost = %d, want 92000", got)
	}
	billable := Billable(usage, AccountingInputIncludesCache)
	if billable.Input != 75_000 || billable.CacheRead != 20_000 {
		t.Fatalf("openai billable = %+v", billable)
	}
}

func TestCostDoesNotDoubleCountOpenAICachedAndCacheRead(t *testing.T) {
	usage := TokenUsage{
		Input:     100_000,
		Output:    5_000,
		Cached:    20_000,
		CacheRead: 20_000,
	}
	price := PricePerMTok{
		Input: 1_000_000, Output: 2_000_000, Cached: 100_000, CacheRead: 100_000,
		AccountingMode: AccountingInputIncludesCache,
	}
	got, err := Cost(usage, price)
	if err != nil {
		t.Fatal(err)
	}
	// 80k input + 5k output + 20k cache read = 0.08 + 0.01 + 0.002 = 92_000
	if got != 92_000 {
		t.Fatalf("deduped openai cost = %d, want 92000", got)
	}
}

func TestCostUsesCachedPriceWhenCacheReadPriceMissing(t *testing.T) {
	usage := TokenUsage{Input: 10_000, CacheRead: 10_000}
	price := PricePerMTok{Cached: 1_000_000, AccountingMode: AccountingInputExcludesCache}
	got, err := Cost(usage, price)
	if err != nil {
		t.Fatal(err)
	}
	if got != 10_000 {
		t.Fatalf("cached price fallback = %d, want 10000", got)
	}
}

func TestCostDoesNotBillReasoningSeparately(t *testing.T) {
	usage := TokenUsage{Input: 1_000, Output: 1_000, Reasoning: 500}
	price := PricePerMTok{Input: 1_000_000, Output: 2_000_000, Reasoning: 5_000_000}
	got, err := Cost(usage, price)
	if err != nil {
		t.Fatal(err)
	}
	if got != 3_000 {
		t.Fatalf("reasoning must not be an extra line: cost = %d, want 3000", got)
	}
}

func TestDefaultAccountingMode(t *testing.T) {
	if DefaultAccountingMode("claude-sonnet-4", "") != AccountingInputExcludesCache {
		t.Fatal("claude model should exclude cache from input")
	}
	if DefaultAccountingMode("gpt-4o", "openai") != AccountingInputIncludesCache {
		t.Fatal("openai should include cache in input")
	}
	if ResolveAccountingMode(AccountingInputExcludesCache, "gpt-4o", "") != AccountingInputExcludesCache {
		t.Fatal("explicit mode must win")
	}
}

func TestCostRejectsInvalidAccountingMode(t *testing.T) {
	_, err := Cost(TokenUsage{Input: 1}, PricePerMTok{AccountingMode: "split"})
	if err == nil {
		t.Fatal("expected invalid accounting mode")
	}
}
