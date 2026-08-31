package money

import "testing"

func TestSelectPriceUsesHighestContextTier(t *testing.T) {
	base := PricePerMTok{Input: 200_000, Output: 1_200_000, CacheRead: 20_000}
	tiers := []PriceTier{
		{Kind: PriceTierContext, Label: "128K", Threshold: 128_000, Price: PricePerMTok{Input: 300_000, Output: 1_500_000}},
		{Kind: PriceTierContext, Label: "272K", Threshold: 272_000, Price: PricePerMTok{Input: 400_000, Output: 1_800_000, CacheRead: 40_000}},
	}
	got := SelectPrice(base, tiers, TokenUsage{Input: 272_000, Output: 100}, "")
	if got.Input != 400_000 || got.Output != 1_800_000 || got.CacheRead != 40_000 {
		t.Fatalf("272k price = %+v", got)
	}
	below := SelectPrice(base, tiers, TokenUsage{Input: 271_999, Output: 100}, "")
	if below.Input != 300_000 || below.Output != 1_500_000 || below.CacheRead != 20_000 {
		t.Fatalf("128k overlay should inherit cache read: %+v", below)
	}
	small := SelectPrice(base, tiers, TokenUsage{Input: 1_000, Output: 10}, "")
	if small != base {
		t.Fatalf("small prompt = %+v, want base", small)
	}
}

func TestSelectPriceServiceTierBeatsContext(t *testing.T) {
	base := PricePerMTok{Input: 200_000, Output: 1_200_000}
	tiers := []PriceTier{
		{Kind: PriceTierContext, Label: "272K", Threshold: 272_000, Price: PricePerMTok{Input: 400_000, Output: 1_800_000}},
		{Kind: PriceTierService, Service: "fast,priority", Price: PricePerMTok{Input: 400_000, Output: 2_400_000}},
	}
	got := SelectPrice(base, tiers, TokenUsage{Input: 300_000, Output: 20}, "priority")
	if got.Input != 400_000 || got.Output != 2_400_000 {
		t.Fatalf("priority should win over 272k: %+v", got)
	}
	fast := SelectPrice(base, tiers, TokenUsage{Input: 1_000, Output: 20}, "fast")
	if fast.Output != 2_400_000 {
		t.Fatalf("fast alias = %+v", fast)
	}
	def := SelectPrice(base, tiers, TokenUsage{Input: 300_000, Output: 20}, "default")
	if def.Output != 1_800_000 {
		t.Fatalf("default response must not keep priority: %+v", def)
	}
}

func TestSelectPriceClaudeContextIncludesCache(t *testing.T) {
	base := PricePerMTok{Input: 3_000_000, Output: 15_000_000, AccountingMode: AccountingInputExcludesCache}
	tiers := []PriceTier{
		{Kind: PriceTierContext, Threshold: 200_000, Price: PricePerMTok{Input: 6_000_000, Output: 22_500_000}},
	}
	usage := TokenUsage{Input: 150_000, CacheRead: 40_000, CacheCreation: 10_000, Output: 20}
	got := SelectPrice(base, tiers, usage, "")
	if got.Input != 6_000_000 {
		t.Fatalf("claude prompt size 200k should hit context tier: %+v tokens=%d", got, ContextTokens(usage, base.AccountingMode))
	}
}

func TestCostForTieredUses272KRates(t *testing.T) {
	base := PricePerMTok{Input: 200_000, Output: 1_200_000}
	tiers := []PriceTier{
		{Kind: PriceTierContext, Threshold: 272_000, Price: PricePerMTok{Input: 400_000, Output: 1_800_000}},
	}
	usage := TokenUsage{Input: 272_000, Output: 1_000}
	got, err := CostForTiered(usage, base, tiers, "gpt-5.6-luna", "openai", "")
	if err != nil {
		t.Fatal(err)
	}
	want, err := CostFor(usage, PricePerMTok{Input: 400_000, Output: 1_800_000}, "gpt-5.6-luna", "openai")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("tiered cost = %d, want %d", got, want)
	}
}

func TestServiceTiersDropsContextCards(t *testing.T) {
	tiers := []PriceTier{
		{Kind: PriceTierContext, Threshold: 272_000, Price: PricePerMTok{Input: 400_000}},
		{Kind: PriceTierService, Service: "priority", Price: PricePerMTok{Input: 400_000, Output: 2_400_000}},
	}
	got := ServiceTiers(tiers)
	if len(got) != 1 || got[0].Service != "priority" {
		t.Fatalf("service tiers = %#v", got)
	}
}

func TestNormalizeServiceTier(t *testing.T) {
	if NormalizeServiceTier("PRIORITY") != "priority" {
		t.Fatal("priority")
	}
	if NormalizeServiceTier("auto") != "" {
		t.Fatal("auto is default")
	}
	if NormalizeServiceTier("flex") != "flex" {
		t.Fatal("flex")
	}
}
