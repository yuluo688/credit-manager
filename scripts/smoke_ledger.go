//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yuluo688/credit-manager/internal/config"
	"github.com/yuluo688/credit-manager/internal/keys"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
	"github.com/yuluo688/credit-manager/internal/usageparse"
)

func main() {
	dir, err := os.MkdirTemp("", "credit-manager-smoke-*")
	must(err)
	defer os.RemoveAll(dir)

	pepperEnv := "CREDIT_MANAGER_SMOKE_PEPPERS"
	must(os.Setenv(pepperEnv, "active:0123456789abcdef0123456789abcdef"))

	cfg := config.Default()
	cfg.DataDir = dir
	cfg.Keys.PepperEnv = pepperEnv
	cfg.Keys.ActivePepperID = "active"
	must(cfg.Validate())

	ctx := context.Background()
	svc, err := service.Open(ctx, cfg)
	must(err)
	defer svc.Close()

	caller, err := svc.Store().CreateCaller(ctx, store.CallerSpec{
		ID: "caller-1", DisplayName: "demo", QuotaMicroUSD: 0, Enabled: true,
	})
	must(err)

	must(svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "all", MatchKind: store.MatchGlob, Pattern: "*", Priority: 1, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 2_000_000},
	}))

	key, material, err := svc.MintKey(ctx, caller.ID, "primary", money.MicroUSD(10_000_000), nil)
	must(err)
	if material.Plaintext == "" || key.CallerScope == "" {
		panic("missing plaintext or caller scope")
	}
	if keys.CallerScope(material.Principal) != key.CallerScope {
		panic("caller scope mismatch")
	}

	// Authenticate success
	principal, _, ok := svc.Authenticate(ctx, map[string][]string{
		"Authorization": {"Bearer " + material.Plaintext},
	})
	if !ok || principal != material.Principal {
		panic("auth failed")
	}

	// Reserve + settle from usage
	plan, err := svc.BuildReservePlan(ctx, "gpt-test", []byte(`{"model":"gpt-test","max_tokens":100,"messages":[{"role":"user","content":"hi"}]}`))
	must(err)
	res, err := svc.Reserve(ctx, key, plan, "idem-1")
	must(err)
	parsed := usageparse.FromResponseBody([]byte(`{"usage":{"prompt_tokens":10,"completion_tokens":20}}`), "openai")
	if !parsed.Found {
		panic("usage not parsed")
	}
	must(svc.SettleFromUsage(ctx, res, plan, parsed, "openai", store.UsageMetrics{}))

	// Over-settlement path is allowed; next reserve may fail if balance insufficient.
	rotatedKey, rotatedMaterial, err := svc.RotateKey(ctx, key.ID, "")
	must(err)
	if _, _, ok := svc.Authenticate(ctx, map[string][]string{"Authorization": {"Bearer " + material.Plaintext}}); ok {
		panic("rotated key still authenticates")
	}
	if _, _, ok := svc.Authenticate(ctx, map[string][]string{"Authorization": {"Bearer " + rotatedMaterial.Plaintext}}); !ok {
		panic("replacement key does not authenticate")
	}
	if rotatedKey.RevokedAt != nil {
		panic("replacement key is revoked")
	}

	// Revoke key
	must(svc.Store().RevokePluginKey(ctx, rotatedKey.ID))
	if _, _, ok := svc.Authenticate(ctx, map[string][]string{"Authorization": {"Bearer " + rotatedMaterial.Plaintext}}); ok {
		panic("revoked key still authenticates")
	}

	fmt.Printf("smoke ok db=%s\n", filepath.Join(dir, "credit-manager.db"))
	fmt.Printf("remaining=%d held=%d settled=%d\n",
		mustKey(svc, rotatedKey.ID).RemainingMicroUSD(),
		mustKey(svc, rotatedKey.ID).HeldAmountMicroUSD,
		mustKey(svc, rotatedKey.ID).SettledSpendMicroUSD,
	)
	_ = time.Now()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func mustKey(svc *service.Service, id string) store.PluginKey {
	key, err := svc.Store().GetPluginKey(context.Background(), id)
	must(err)
	return key
}
