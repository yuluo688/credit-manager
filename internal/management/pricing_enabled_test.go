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
