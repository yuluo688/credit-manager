package store

import (
	"context"
	"errors"
	"testing"

	"github.com/yuluo688/credit-manager/internal/money"
)

func TestFinishExecutionKeepsFinancialHoldAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{MaxConcurrentRequests: 1, QuotaMicroUSD: 10})
	first, err := st.Reserve(ctx, reserveRequest(key, "first", 7))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := st.FinishExecution(ctx, first.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.Reserve(ctx, reserveRequest(key, "too-expensive", 4)); !errors.Is(err, ErrInsufficientQuota) {
		t.Fatalf("quota was released together with concurrency: %v", err)
	}
	second, err := st.Reserve(ctx, reserveRequest(key, "second", 3))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.FinishExecution(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Reserve(ctx, reserveRequest(key, "third", 0)); !errors.Is(err, ErrConcurrentLimit) {
		t.Fatalf("duplicate finish released another request's slot: %v", err)
	}
	settlement := Settlement{ReservationID: first.ID, Model: "test-model", CostMicroUSD: 2, Usage: money.TokenUsage{Input: 2}}
	for i := 0; i < 2; i++ {
		if _, err := st.Settle(ctx, settlement); err != nil {
			t.Fatal(err)
		}
	}
	got, err := st.GetPluginKey(ctx, key.ID)
	if err != nil || got.HeldAmountMicroUSD != second.HeldMicroUSD || got.SettledSpendMicroUSD != 2 {
		t.Fatalf("financial accounting changed: %v", err)
	}
}
