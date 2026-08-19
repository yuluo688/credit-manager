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

func TestCostPerImageIgnoresTokens(t *testing.T) {
	price := PricePerMTok{BillingMode: BillingPerImage, PerImage: 80_000, Input: 5_000_000, Output: 30_000_000}
	got, err := Cost(TokenUsage{Input: 12_000, Output: 4_000, Images: 2}, price)
	if err != nil {
		t.Fatal(err)
	}
	if got != 160_000 {
		t.Fatalf("per-image cost = %d, want 160000", got)
	}
}

func TestCostPerImageDefaultsToOne(t *testing.T) {
	got, err := Cost(TokenUsage{}, PricePerMTok{BillingMode: BillingPerImage, PerImage: 40_000})
	if err != nil {
		t.Fatal(err)
	}
	if got != 40_000 {
		t.Fatalf("default image count cost = %d, want 40000", got)
	}
}

func TestCostGrokHostUsageMatchesCapTracker(t *testing.T) {
	usage := TokenUsage{Input: 215, Output: 291, Reasoning: 290, Cached: 128, CacheRead: 128}
	price := PricePerMTok{Input: 2_000_000, Output: 6_000_000, CacheRead: 500_000}
	got, err := CostFor(usage, price, "grok-4.6", "xai")
	if err != nil {
		t.Fatal(err)
	}
	// billable input 87 * $2 + output 291 * $6 + cache 128 * $0.5
	if got != 1984 {
		t.Fatalf("grok host usage cost = %d, want 1984", got)
	}
}

func TestReportedTotalMatchesCapTracker(t *testing.T) {
	if got := ReportedTotal(TokenUsage{ReportedTotal: 506, Input: 215, Output: 291, Reasoning: 290, Cached: 128, CacheRead: 128}); got != 506 {
		t.Fatalf("host total = %d, want 506", got)
	}
	if got := ReportedTotal(TokenUsage{Input: 17, Output: 9, Reasoning: 4, Cached: 50, CacheRead: 50}); got != 30 {
		t.Fatalf("fallback total = %d, want 30", got)
	}
	if got := ReportedTotal(TokenUsage{Cached: 41, CacheRead: 99, CacheCreation: 100}); got != 41 {
		t.Fatalf("cached-only total = %d, want 41", got)
	}
}

func TestCostRejectsInvalidBillingMode(t *testing.T) {
	_, err := Cost(TokenUsage{Images: 1}, PricePerMTok{BillingMode: "per_token"})
	if err == nil {
		t.Fatal("expected invalid billing mode")
	}
}
