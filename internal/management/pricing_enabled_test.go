package management

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/yuluo688/credit-manager/internal/config"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
)

func TestSetPricingEnabledRequiresFlag(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CREDIT_MANAGER_TEST_PEPPERS", "active:0123456789abcdef0123456789abcdef")
	cfg := config.Default()
	cfg.DataDir = dir
	cfg.Keys.PepperEnv = "CREDIT_MANAGER_TEST_PEPPERS"
	cfg.Keys.ActivePepperID = "active"
	svc, err := service.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "gpt-4o", MatchKind: store.MatchExact, Pattern: "gpt-4o", Priority: 100, Enabled: true,
		Price: money.PricePerMTok{Input: 1},
	}); err != nil {
		t.Fatal(err)
	}

	resp, err := setPricingEnabled(ctx, svc, []byte(`{"id":"gpt-4o"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	rule, err := svc.Store().GetPricingRule(ctx, "gpt-4o")
	if err != nil || !rule.Enabled {
		t.Fatalf("omitted enabled must not disable the rule: %#v err=%v", rule, err)
	}

	resp, err = setPricingEnabled(ctx, svc, []byte(`{"id":"gpt-4o","enabled":false}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("disable status = %d body=%s", resp.StatusCode, resp.Body)
	}
	rule, err = svc.Store().GetPricingRule(ctx, "gpt-4o")
	if err != nil || rule.Enabled {
		t.Fatalf("after disable = %#v err=%v", rule, err)
	}
	var body map[string]any
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body["enabled"] != false {
		t.Fatalf("response = %#v", body)
	}
}

func TestPutPricingPersistsSpecialTiers(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CREDIT_MANAGER_TEST_PEPPERS", "active:0123456789abcdef0123456789abcdef")
	cfg := config.Default()
	cfg.DataDir = dir
	cfg.Keys.PepperEnv = "CREDIT_MANAGER_TEST_PEPPERS"
	cfg.Keys.ActivePepperID = "active"
	svc, err := service.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	payload := []byte(`{
		"id":"gpt-5.6-luna","match_kind":"exact","pattern":"gpt-5.6-luna","priority":100,"enabled":true,
		"price":{"input":200000,"output":1200000,"cache_read":20000,"billing_mode":"token"},
		"tiers":[
			{"kind":"context","label":"272K","threshold":272000,"price":{"input":400000,"output":1800000,"cache_read":40000}},
			{"kind":"service","service":"priority","price":{"input":400000,"output":2400000}}
		]
	}`)
	resp, err := putPricing(ctx, svc, payload)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, resp.Body)
	}
	rule, err := svc.Store().GetPricingRule(ctx, "gpt-5.6-luna")
	if err != nil {
		t.Fatal(err)
	}
	if len(rule.Tiers) != 2 || rule.Tiers[0].Threshold != 272000 || rule.Tiers[1].Service != "priority" {
		t.Fatalf("saved tiers = %#v", rule.Tiers)
	}

	omit := []byte(`{"id":"gpt-5.6-luna","match_kind":"exact","pattern":"gpt-5.6-luna","priority":100,"price":{"input":200000,"output":1500000,"billing_mode":"token"}}`)
	resp, err = putPricing(ctx, svc, omit)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("omit status = %d body=%s", resp.StatusCode, resp.Body)
	}
	rule, err = svc.Store().GetPricingRule(ctx, "gpt-5.6-luna")
	if err != nil {
		t.Fatal(err)
	}
	if len(rule.Tiers) != 2 || rule.Price.Output != 1_500_000 {
		t.Fatalf("omitted tiers must keep special cards: %#v", rule)
	}

	clear := []byte(`{"id":"gpt-5.6-luna","match_kind":"exact","pattern":"gpt-5.6-luna","priority":100,"price":{"input":200000,"output":1500000,"billing_mode":"token"},"tiers":[]}`)
	resp, err = putPricing(ctx, svc, clear)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("clear status = %d body=%s", resp.StatusCode, resp.Body)
	}
	rule, err = svc.Store().GetPricingRule(ctx, "gpt-5.6-luna")
	if err != nil || len(rule.Tiers) != 0 {
		t.Fatalf("explicit empty tiers should clear: %#v err=%v", rule, err)
	}
}

func TestListKeysSearchesLabelsAndClampsPagination(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	t.Setenv("CREDIT_MANAGER_TEST_PEPPERS", "active:0123456789abcdef0123456789abcdef")
	cfg := config.Default()
	cfg.DataDir = dir
	cfg.Keys.PepperEnv = "CREDIT_MANAGER_TEST_PEPPERS"
	cfg.Keys.ActivePepperID = "active"
	svc, err := service.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	alphaOne, _, err := svc.MintKey(ctx, service.BootstrapCallerID, "Alpha one", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	alphaTwo, _, err := svc.MintKey(ctx, service.BootstrapCallerID, "Alpha two", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.MintKey(ctx, service.BootstrapCallerID, "Other", 0, nil); err != nil {
		t.Fatal(err)
	}

	resp, err := listKeys(ctx, svc, map[string][]string{
		"active_only": {"1"},
		"page":        {"5"},
		"page_size":   {"1"},
		"q":           {"ALPHA"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, resp.Body)
	}
	var result struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
		Page  int   `json:"page"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || result.Page != 2 || len(result.Items) != 1 {
		t.Fatalf("alpha result = %#v", result)
	}
	if result.Items[0].ID != alphaOne.ID && result.Items[0].ID != alphaTwo.ID {
		t.Fatalf("alpha item = %#v", result.Items[0])
	}

	resp, err = listKeys(ctx, svc, map[string][]string{"active_only": {"1"}, "q": {alphaOne.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("id-only result = %#v", result)
	}
}
