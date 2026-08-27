package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/yuluo688/credit-manager/internal/money"
)

func TestModelAllowed(t *testing.T) {
	if ModelAllowed(PluginKey{}, "") {
		t.Fatal("empty model should be rejected")
	}
	if !ModelAllowed(PluginKey{}, "gpt-4o") {
		t.Fatal("empty allowlist should allow gpt-4o")
	}
	key := PluginKey{AllowedModels: []string{"gpt-4o", "claude-*"}}
	if !ModelAllowed(key, "gpt-4o") || !ModelAllowed(key, "claude-sonnet") {
		t.Fatal("allowlist should match exact and glob")
	}
	if ModelAllowed(key, "gemini-pro") {
		t.Fatal("unlisted model should be rejected")
	}
}

func TestMatchModelTokenLimit(t *testing.T) {
	limits := []ModelTokenLimit{
		{Model: "gpt-*", Daily: ModelPeriodTokenLimit{Tokens: 10}},
		{Model: "gpt-4o", Daily: ModelPeriodTokenLimit{Tokens: 5}},
		{Model: "claude-*", Daily: ModelPeriodTokenLimit{Mode: ModelTokenLimitModeAvailable}},
	}
	got, ok := MatchModelTokenLimit(limits, "gpt-4o")
	if !ok || got.Model != "gpt-4o" || got.Daily.Cap() != 5 {
		t.Fatalf("exact match = %+v ok=%t", got, ok)
	}
	got, ok = MatchModelTokenLimit(limits, "gpt-4.1")
	if !ok || got.Model != "gpt-*" || got.Daily.Cap() != 10 {
		t.Fatalf("glob match = %+v ok=%t", got, ok)
	}
	if _, ok := MatchModelTokenLimit(limits, "gemini-pro"); ok {
		t.Fatal("unlisted model should have no token policy")
	}
}

func TestReserveEnforcesModelTokenLimits(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{
		ModelTokenLimits: []ModelTokenLimit{{
			Model: "test-model",
			Daily: ModelPeriodTokenLimit{Tokens: 100},
		}},
	})
	first := reserveRequest(key, "first", 1)
	first.RequestTokenEstimate = 60
	if _, err := st.Reserve(ctx, first); err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	second := reserveRequest(key, "second", 1)
	second.RequestTokenEstimate = 50
	if _, err := st.Reserve(ctx, second); !errors.Is(err, ErrDailyTokenLimitExceeded) {
		t.Fatalf("second reserve error = %v, want %v", err, ErrDailyTokenLimitExceeded)
	}
	other := reserveRequest(key, "other", 1)
	other.Model = "other-model"
	other.RequestTokenEstimate = 50
	if _, err := st.Reserve(ctx, other); err != nil {
		t.Fatalf("unlisted model should not use token cap: %v", err)
	}
}

func TestReserveModelTokenLimitAvailableOrUnlimitedSkipsCap(t *testing.T) {
	for _, mode := range []string{ModelTokenLimitModeAvailable, ModelTokenLimitModeUnlimited} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			st := newTestStore(t)
			defer st.Close()
			key := newTestKey(t, ctx, st, PluginKeySpec{
				Kid:         "key-" + mode,
				Principal:   "credit-manager:key-" + mode,
				CallerScope: "credit-manager:key-" + mode,
				ModelTokenLimits: []ModelTokenLimit{{
					Model:   "test-model",
					Daily:   ModelPeriodTokenLimit{Mode: mode},
					Weekly:  ModelPeriodTokenLimit{Mode: mode},
					Monthly: ModelPeriodTokenLimit{Mode: mode},
				}},
			})
			req := reserveRequest(key, "first", 1)
			req.RequestTokenEstimate = 1_000_000
			if _, err := st.Reserve(ctx, req); err != nil {
				t.Fatalf("reserve with %s mode: %v", mode, err)
			}
		})
	}
}

func TestReserveModelTokenLimitGlobSharesPool(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{
		ModelTokenLimits: []ModelTokenLimit{{
			Model: "test-*",
			Daily: ModelPeriodTokenLimit{Tokens: 100},
		}},
	})
	first := reserveRequest(key, "first", 1)
	first.Model = "test-a"
	first.RequestTokenEstimate = 60
	if _, err := st.Reserve(ctx, first); err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	second := reserveRequest(key, "second", 1)
	second.Model = "test-b"
	second.RequestTokenEstimate = 50
	if _, err := st.Reserve(ctx, second); !errors.Is(err, ErrDailyTokenLimitExceeded) {
		t.Fatalf("second reserve error = %v, want %v", err, ErrDailyTokenLimitExceeded)
	}
}

func TestListModelTokenUsageBucketsPeriods(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{})
	now := time.Now().UTC()
	monthStart := time.UnixMilli(utcMonthStart(now.UnixMilli())).UTC()
	if _, err := st.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatal(err)
	}
	insert := func(id string, at time.Time, tokens int64) {
		t.Helper()
		_, err := st.db.ExecContext(ctx, `INSERT INTO usage_ledger(id, reservation_id, caller_id, plugin_key_id, model, input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, cost_micro_usd, source, created_at_unix_ms) VALUES (?, ?, ?, ?, 'test-model', ?, 0, 0, 0, 0, 0, 1, 'usage', ?)`, id, "r-"+id, key.CallerID, key.ID, tokens, at.UnixMilli())
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("u-day", now, 10)
	insert("u-old", monthStart.Add(-time.Hour), 20)
	got, err := st.ListModelTokenUsage(ctx, key.ID, []ModelTokenLimit{{Model: "test-model"}}, now.UnixMilli())
	if err != nil || len(got) != 1 {
		t.Fatalf("usage=%#v err=%v", got, err)
	}
	if got[0].DailyUsed != 10 || got[0].MonthlyUsed != 10 {
		t.Fatalf("daily=%d monthly=%d", got[0].DailyUsed, got[0].MonthlyUsed)
	}
}

func TestReserveUnmatchedModelsDisabled(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{
		UnmatchedModelsMode: UnmatchedModelsDisabled,
		ModelTokenLimits: []ModelTokenLimit{{
			Model: "allowed-model",
			Daily: ModelPeriodTokenLimit{Mode: ModelTokenLimitModeUnlimited},
		}},
	})
	okReq := reserveRequest(key, "ok", 1)
	okReq.Model = "allowed-model"
	if _, err := st.Reserve(ctx, okReq); err != nil {
		t.Fatalf("listed model: %v", err)
	}
	blocked := reserveRequest(key, "blocked", 1)
	blocked.Model = "other-model"
	if _, err := st.Reserve(ctx, blocked); !errors.Is(err, ErrModelNotAllowed) {
		t.Fatalf("unmatched error = %v, want %v", err, ErrModelNotAllowed)
	}
}

func TestReserveUnmatchedDisabledEmptyListBlocksAll(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{UnmatchedModelsMode: UnmatchedModelsDisabled})
	if _, err := st.Reserve(ctx, reserveRequest(key, "any", 1)); !errors.Is(err, ErrModelNotAllowed) {
		t.Fatalf("empty list + disabled error = %v, want %v", err, ErrModelNotAllowed)
	}
}

func TestUpdatePluginKeyPolicyModelTokenLimits(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{})
	limits := []ModelTokenLimit{{
		Model:   "gpt-4o",
		Daily:   ModelPeriodTokenLimit{Tokens: 1000},
		Weekly:  ModelPeriodTokenLimit{Mode: ModelTokenLimitModeAvailable},
		Monthly: ModelPeriodTokenLimit{Mode: ModelTokenLimitModeUnlimited},
	}}
	updated, err := st.UpdatePluginKeyPolicy(ctx, PluginKeyPolicyUpdate{ID: key.ID, ModelTokenLimits: &limits})
	if err != nil {
		t.Fatalf("update policy: %v", err)
	}
	if len(updated.ModelTokenLimits) != 1 || updated.ModelTokenLimits[0].Model != "gpt-4o" ||
		updated.ModelTokenLimits[0].Daily.Cap() != 1000 ||
		updated.ModelTokenLimits[0].Weekly.NormalizedMode() != ModelTokenLimitModeAvailable ||
		updated.ModelTokenLimits[0].Monthly.NormalizedMode() != ModelTokenLimitModeUnlimited {
		t.Fatalf("updated limits = %+v", updated.ModelTokenLimits)
	}
	empty := []ModelTokenLimit{}
	cleared, err := st.UpdatePluginKeyPolicy(ctx, PluginKeyPolicyUpdate{ID: key.ID, ModelTokenLimits: &empty})
	if err != nil {
		t.Fatalf("clear policy: %v", err)
	}
	if len(cleared.ModelTokenLimits) != 0 {
		t.Fatalf("cleared limits = %+v", cleared.ModelTokenLimits)
	}
}

func TestReserveEnforcesKeyLimits(t *testing.T) {
	tests := []struct {
		name   string
		spec   PluginKeySpec
		first  money.MicroUSD
		second money.MicroUSD
		want   error
	}{
		{
			name:   "total quota",
			spec:   PluginKeySpec{QuotaMicroUSD: 10},
			first:  6,
			second: 5,
			want:   ErrInsufficientQuota,
		},
		{
			name:   "daily quota",
			spec:   PluginKeySpec{DailyQuotaMicroUSD: 10},
			first:  6,
			second: 5,
			want:   ErrDailyQuotaExceeded,
		},
		{
			name:   "weekly quota",
			spec:   PluginKeySpec{WeeklyQuotaMicroUSD: 10},
			first:  6,
			second: 5,
			want:   ErrWeeklyQuotaExceeded,
		},
		{
			name:   "monthly quota",
			spec:   PluginKeySpec{MonthlyQuotaMicroUSD: 10},
			first:  6,
			second: 5,
			want:   ErrMonthlyQuotaExceeded,
		},
		{
			name:   "maximum concurrency",
			spec:   PluginKeySpec{MaxConcurrentRequests: 1},
			first:  1,
			second: 1,
			want:   ErrConcurrentLimit,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			st := newTestStore(t)
			defer st.Close()
			key := newTestKey(t, ctx, st, test.spec)

			if _, err := st.Reserve(ctx, reserveRequest(key, "first", test.first)); err != nil {
				t.Fatalf("first reserve: %v", err)
			}
			if _, err := st.Reserve(ctx, reserveRequest(key, "second", test.second)); !errors.Is(err, test.want) {
				t.Fatalf("second reserve error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestReserveReleasesConcurrencySlot(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{MaxConcurrentRequests: 1})

	first, err := st.Reserve(ctx, reserveRequest(key, "first", 1))
	if err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	if _, err := st.Reserve(ctx, reserveRequest(key, "blocked", 1)); !errors.Is(err, ErrConcurrentLimit) {
		t.Fatalf("blocked reserve error = %v, want %v", err, ErrConcurrentLimit)
	}
	if _, err := st.Release(ctx, first.ID, "test"); err != nil {
		t.Fatalf("release first reservation: %v", err)
	}
	if _, err := st.Reserve(ctx, reserveRequest(key, "after-release", 1)); err != nil {
		t.Fatalf("reserve after release: %v", err)
	}
}

func TestReserveConcurrentRequestsCannotExceedLimit(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{MaxConcurrentRequests: 1})

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, idempotency := range []string{"first", "second"} {
		wg.Add(1)
		go func(idempotency string) {
			defer wg.Done()
			<-start
			_, err := st.Reserve(ctx, reserveRequest(key, idempotency, 1))
			errs <- err
		}(idempotency)
	}
	close(start)
	wg.Wait()
	close(errs)

	var succeeded, limited int
	for err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		if errors.Is(err, ErrConcurrentLimit) {
			limited++
			continue
		}
		t.Fatalf("reserve error = %v, want success or %v", err, ErrConcurrentLimit)
	}
	if succeeded != 1 || limited != 1 {
		t.Fatalf("results: succeeded=%d limited=%d, want 1 each", succeeded, limited)
	}
}

func TestPeriodStartsUseUTCCalendarBoundaries(t *testing.T) {
	for _, test := range []struct {
		name string
		now  time.Time
		got  func(int64) int64
		want time.Time
	}{
		{
			name: "day",
			now:  time.Date(2026, time.August, 17, 23, 59, 59, 0, time.FixedZone("UTC+8", 8*60*60)),
			got:  utcDayStart,
			want: time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "week starts Monday",
			now:  time.Date(2026, time.August, 16, 23, 59, 59, 0, time.UTC),
			got:  utcWeekStart,
			want: time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "month",
			now:  time.Date(2026, time.August, 31, 23, 59, 59, 0, time.UTC),
			got:  utcMonthStart,
			want: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := time.UnixMilli(test.got(test.now.UnixMilli())).UTC(); !got.Equal(test.want) {
				t.Fatalf("start = %s, want %s", got, test.want)
			}
		})
	}
}

func TestListAuditEventsFiltersByPluginKey(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	first := newTestKey(t, ctx, st, PluginKeySpec{})
	secondSpec := PluginKeySpec{Kid: "test-key-two", Principal: "credit-manager:test-key-two", CallerScope: "credit-manager:test-key-two"}
	second := newTestKey(t, ctx, st, secondSpec)
	if _, err := st.Reserve(ctx, reserveRequest(first, "first", 1)); err != nil {
		t.Fatalf("reserve first key: %v", err)
	}
	if _, err := st.Reserve(ctx, reserveRequest(second, "second", 1)); err != nil {
		t.Fatalf("reserve second key: %v", err)
	}
	events, err := st.ListAuditEventsFiltered(ctx, AuditFilter{PluginKeyID: second.ID, Limit: 10})
	if err != nil {
		t.Fatalf("list filtered audit events: %v", err)
	}
	if len(events) != 1 || events[0].PluginKeyID == nil || *events[0].PluginKeyID != second.ID {
		t.Fatalf("filtered events = %#v, want only key %q", events, second.ID)
	}
}

func TestAuditEventJSONUsesAPINames(t *testing.T) {
	event := AuditEvent{ID: 1, EventType: "quota_held"}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal audit event: %v", err)
	}
	if !bytes.Contains(raw, []byte(`"event_type":"quota_held"`)) || bytes.Contains(raw, []byte(`"EventType"`)) {
		t.Fatalf("audit json = %s, want snake_case API fields", raw)
	}
}

func TestGetKeyUsageOverviewIncludesHeldSpend(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{})
	if _, err := st.Reserve(ctx, reserveRequest(key, "held", 7)); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	overview, err := st.GetKeyUsageOverview(ctx, key.ID, time.Now().UTC())
	if err != nil {
		t.Fatalf("get key usage overview: %v", err)
	}
	if overview.ActiveReservations != 1 {
		t.Fatalf("active reservations = %d, want 1", overview.ActiveReservations)
	}
	if overview.DailyMicroUSD != 7 || overview.WeeklyMicroUSD != 7 || overview.MonthlyMicroUSD != 7 {
		t.Fatalf("period usage = daily:%d weekly:%d monthly:%d, want 7", overview.DailyMicroUSD, overview.WeeklyMicroUSD, overview.MonthlyMicroUSD)
	}
}

func TestReleaseStaleReservationsReleasesHold(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{MaxConcurrentRequests: 1})
	reservation, err := st.Reserve(ctx, reserveRequest(key, "stale", 7))
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	staleAt := time.Now().UTC().Add(-time.Hour).UnixMilli()
	if _, err := st.db.ExecContext(ctx, `UPDATE reservations SET updated_at_unix_ms = ? WHERE id = ?`, staleAt, reservation.ID); err != nil {
		t.Fatalf("age reservation: %v", err)
	}
	released, err := st.ReleaseStaleReservations(ctx, time.Now().UTC().Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("release stale reservations: %v", err)
	}
	if released != 1 {
		t.Fatalf("released = %d, want 1", released)
	}
	updated, err := st.GetReservation(ctx, reservation.ID)
	if err != nil {
		t.Fatalf("get released reservation: %v", err)
	}
	if updated.Status != ReservationReleased || updated.SettlementSummary != "stale_timeout" {
		t.Fatalf("released reservation = %#v, want stale released", updated)
	}
	updatedKey, err := st.GetPluginKey(ctx, key.ID)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}
	if updatedKey.HeldAmountMicroUSD != 0 {
		t.Fatalf("held amount = %d, want 0", updatedKey.HeldAmountMicroUSD)
	}
}

func TestTouchReservationKeepsHeldReservationFresh(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{})
	reservation, err := st.Reserve(ctx, reserveRequest(key, "heartbeat", 1))
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	staleAt := time.Now().UTC().Add(-time.Hour).UnixMilli()
	if _, err := st.db.ExecContext(ctx, `UPDATE reservations SET updated_at_unix_ms = ? WHERE id = ?`, staleAt, reservation.ID); err != nil {
		t.Fatalf("age reservation: %v", err)
	}
	if err := st.TouchReservation(ctx, reservation.ID); err != nil {
		t.Fatalf("touch reservation: %v", err)
	}
	released, err := st.ReleaseStaleReservations(ctx, time.Now().UTC().Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("release stale reservations: %v", err)
	}
	if released != 0 {
		t.Fatalf("released = %d, want 0 after heartbeat", released)
	}
}

func TestUpdateUsageDetailRepricesSettledSpend(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{QuotaMicroUSD: 10_000})
	reservation, err := st.Reserve(ctx, reserveRequest(key, "reprice", 1_000))
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	ledgerID := NewID()
	if _, err := st.Settle(ctx, Settlement{
		LedgerID:              ledgerID,
		ReservationID:         reservation.ID,
		Model:                 "test-model",
		Usage:                 money.TokenUsage{Input: 100_000, Output: 4_096},
		CostMicroUSD:          1_000,
		EstimatedCostMicroUSD: 1_000,
		Source:                "reserved_fallback",
	}); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if err := st.UpdateUsageDetail(ctx, ledgerID, money.TokenUsage{Input: 20, Output: 8}, 3); err != nil {
		t.Fatalf("update usage detail: %v", err)
	}
	entry, err := st.GetUsage(ctx, ledgerID)
	if err != nil {
		t.Fatalf("get usage: %v", err)
	}
	if entry.Usage.Input != 20 || entry.Usage.Output != 8 || entry.CostMicroUSD != 3 || entry.Source != "host_usage" {
		t.Fatalf("usage = %#v", entry)
	}
	updatedKey, err := st.GetPluginKey(ctx, key.ID)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}
	if updatedKey.SettledSpendMicroUSD != 3 {
		t.Fatalf("settled spend = %d, want 3", updatedKey.SettledSpendMicroUSD)
	}
	updatedReservation, err := st.GetReservation(ctx, reservation.ID)
	if err != nil {
		t.Fatalf("get reservation: %v", err)
	}
	if updatedReservation.SettledMicroUSD == nil || *updatedReservation.SettledMicroUSD != 3 {
		t.Fatalf("settled reservation = %#v", updatedReservation.SettledMicroUSD)
	}
}

func TestFindRecentFallbackIncludesAuth(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{QuotaMicroUSD: 10_000})
	reservation, err := st.Reserve(ctx, reserveRequest(key, "fallback-auth", 1_000))
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if _, err := st.Settle(ctx, Settlement{
		ReservationID: reservation.ID,
		Model:         "grok-4.6",
		Usage:         money.TokenUsage{Input: 10, Output: 2},
		CostMicroUSD:  1,
		Source:        "reserved_fallback",
	}); err != nil {
		t.Fatalf("settle: %v", err)
	}
	entry, found, err := st.FindRecentFallback(ctx, []string{"grok-4.6"}, time.Minute)
	if err != nil || !found || entry.Model != "grok-4.6" || !entry.Auth.Empty() {
		t.Fatalf("find = (%#v, %v, %v)", entry, found, err)
	}
	if err := st.UpdateUsageAuth(ctx, entry.ID, AuthIdentity{AuthID: "auth-1", Provider: "xai", Label: "ops"}); err != nil {
		t.Fatalf("update auth: %v", err)
	}
	entry, found, err = st.FindRecentFallback(ctx, []string{"grok-4.6"}, time.Minute)
	if err != nil || !found || entry.Auth.AuthID != "auth-1" {
		t.Fatalf("find after auth = (%#v, %v, %v)", entry, found, err)
	}
}

func TestAuthQuotaSnapshotMigrationAndPersistence(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	var applied int
	if err := st.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations WHERE version = 8`).Scan(&applied); err != nil {
		t.Fatalf("query migration: %v", err)
	}
	if applied != 1 {
		t.Fatalf("migration 8 count = %d, want 1", applied)
	}
	modTime := time.Date(2026, time.August, 17, 9, 30, 0, 0, time.UTC)
	if err := st.UpsertAuthQuotaSuccess(ctx, "antigravity", "auth-1", `{"remaining":42,"access_token":"secret","nested":{"api_key":"nested-secret","limit":100}}`, &modTime); err != nil {
		t.Fatalf("upsert quota snapshot: %v", err)
	}
	snapshot, err := st.GetAuthQuotaSnapshot(ctx, "antigravity", "auth-1")
	if err != nil {
		t.Fatalf("get quota snapshot: %v", err)
	}
	if snapshot.AuthModTime == nil || !snapshot.AuthModTime.Equal(modTime) || snapshot.LastSuccessAt == nil || snapshot.LastError != "" {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if bytes.Contains([]byte(snapshot.SnapshotJSON), []byte("secret")) || !bytes.Contains([]byte(snapshot.SnapshotJSON), []byte(`"remaining":42`)) || !bytes.Contains([]byte(snapshot.SnapshotJSON), []byte(`"limit":100`)) {
		t.Fatalf("sanitized snapshot = %s", snapshot.SnapshotJSON)
	}
	if err := st.RecordAuthQuotaFailure(ctx, "antigravity", "auth-1", nil, errors.New("upstream unavailable")); err != nil {
		t.Fatalf("record quota failure: %v", err)
	}
	failed, err := st.GetAuthQuotaSnapshot(ctx, "antigravity", "auth-1")
	if err != nil || failed.SnapshotJSON != snapshot.SnapshotJSON || failed.LastSuccessAt == nil || failed.LastErrorAt == nil || failed.LastError != "upstream unavailable" {
		t.Fatalf("snapshot after failure = %#v, err = %v", failed, err)
	}
}
func TestAuthQuotaWindowBaselinePersistsFirstObservation(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	if err := st.UpsertAuthQuotaWindowBaseline(ctx, "codex", "auth-1", "primary", "reset:1", 70); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertAuthQuotaWindowBaseline(ctx, "codex", "auth-1", "primary", "reset:1", 80); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetAuthQuotaWindowBaseline(ctx, "codex", "auth-1", "primary", "reset:1")
	if err != nil || got.BaselineUsed != 70 {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

func TestGetAuthQuotaUsageMatchesAuthAndLegacyIndex(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{QuotaMicroUSD: 10_000})
	from := time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	if _, err := st.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("disable foreign keys: %v", err)
	}
	insert := func(id, provider, authID, index, model string, at time.Time, input, output, cost int64) {
		t.Helper()
		_, err := st.db.ExecContext(ctx, `INSERT INTO usage_ledger(id, reservation_id, caller_id, plugin_key_id, model, input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, cost_micro_usd, source, auth_provider, auth_id, auth_index, created_at_unix_ms) VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0, 0, 0, ?, 'usage', ?, ?, ?, ?)`, id, "r-"+id, key.CallerID, key.ID, model, input, output, cost, provider, authID, index, at.UnixMilli())
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("exact", "antigravity", "auth-1", "index-1", "pool-a", from.Add(time.Hour), 10, 5, 11)
	insert("legacy", "antigravity", "", "index-1", "pool-a", from.Add(2*time.Hour), 7, 8, 13)
	insert("other-auth", "antigravity", "auth-2", "index-1", "pool-a", from.Add(3*time.Hour), 100, 0, 101)
	insert("other-provider", "other", "auth-1", "index-1", "pool-a", from.Add(4*time.Hour), 100, 0, 103)
	insert("other-model", "antigravity", "auth-1", "index-1", "pool-b", from.Add(5*time.Hour), 100, 0, 107)
	insert("outside", "antigravity", "auth-1", "index-1", "pool-a", to, 100, 0, 109)
	usage, err := st.GetAuthQuotaUsage(ctx, AuthQuotaUsageFilter{Provider: "antigravity", AuthID: "auth-1", AuthIndex: "index-1", From: from, To: to, Models: []string{"pool-a"}})
	if err != nil || usage.RequestCount != 2 || usage.InputTokens != 17 || usage.OutputTokens != 13 || usage.ActualCostMicroUSD != 24 {
		t.Fatalf("usage = %#v, err = %v", usage, err)
	}
}

func TestUsageFilterMatchesAuthIdentity(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{QuotaMicroUSD: 10_000})
	at := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.UTC)
	if _, err := st.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("disable foreign keys: %v", err)
	}
	insert := func(id, provider, authID, index string) {
		t.Helper()
		_, err := st.db.ExecContext(ctx, `INSERT INTO usage_ledger(id, reservation_id, caller_id, plugin_key_id, model, input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, cost_micro_usd, source, auth_provider, auth_id, auth_index, auth_label, created_at_unix_ms) VALUES (?, ?, ?, ?, 'm', 1, 1, 0, 0, 0, 0, 1, 'usage', ?, ?, ?, 'acct', ?)`, id, "r-"+id, key.CallerID, key.ID, provider, authID, index, at.UnixMilli())
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("exact", "openai", "auth-1", "index-1")
	insert("legacy", "openai", "", "index-1")
	insert("other", "openai", "auth-2", "index-2")
	insert("alias", "chatgpt", "auth-1", "index-1")
	count, err := st.CountUsage(ctx, UsageFilter{AuthProvider: "codex", AuthID: "auth-1", AuthIndex: "index-1"})
	if err != nil || count != 3 {
		t.Fatalf("auth filter count = %d, err = %v", count, err)
	}
	auths, err := st.ListUsedAuths(ctx)
	if err != nil || len(auths) < 2 {
		t.Fatalf("used auths = %#v, err = %v", auths, err)
	}
}

func TestGetAuthQuotaUsageMatchesProviderAliases(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{QuotaMicroUSD: 10_000})
	from := time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	if _, err := st.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("disable foreign keys: %v", err)
	}
	insert := func(id, provider, model string, input, output, cost int64) {
		t.Helper()
		_, err := st.db.ExecContext(ctx, `INSERT INTO usage_ledger(id, reservation_id, caller_id, plugin_key_id, model, input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, cost_micro_usd, source, auth_provider, auth_id, auth_index, created_at_unix_ms) VALUES (?, ?, ?, ?, ?, ?, ?, 0, 0, 0, 0, ?, 'usage', ?, 'auth-1', 'index-1', ?)`, id, "r-"+id, key.CallerID, key.ID, model, input, output, cost, provider, from.Add(time.Hour).UnixMilli())
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("openai", "openai", "gpt-test", 10, 5, 11)
	insert("chatgpt", "ChatGPT", "gpt-test", 7, 8, 13)
	insert("claude", "anthropic", "claude-test", 3, 1, 5)
	usage, err := st.GetAuthQuotaUsage(ctx, AuthQuotaUsageFilter{Provider: "codex", AuthID: "auth-1", AuthIndex: "index-1", From: from, To: to})
	if err != nil || usage.RequestCount != 2 || usage.InputTokens != 17 || usage.OutputTokens != 13 || usage.ActualCostMicroUSD != 24 {
		t.Fatalf("codex usage = %#v, err = %v", usage, err)
	}
	byModel, err := st.GetAuthQuotaUsageByModel(ctx, AuthQuotaUsageFilter{Provider: "claude", AuthID: "auth-1", AuthIndex: "index-1", From: from, To: to})
	if err != nil || len(byModel) != 1 || byModel[0].Model != "claude-test" || byModel[0].RequestCount != 1 || byModel[0].ActualCostMicroUSD != 5 {
		t.Fatalf("claude usage = %#v, err = %v", byModel, err)
	}
}

func TestSummarizeUsageTrendFilteredHourAndMonth(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{QuotaMicroUSD: 10_000})
	if _, err := st.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("disable foreign keys: %v", err)
	}
	insert := func(id string, at time.Time) {
		t.Helper()
		_, err := st.db.ExecContext(ctx, `INSERT INTO usage_ledger(id, reservation_id, caller_id, plugin_key_id, model, input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, cost_micro_usd, source, created_at_unix_ms) VALUES (?, ?, ?, ?, 'm', 10, 2, 0, 0, 1, 0, 100, 'usage', ?)`, id, "r-"+id, key.CallerID, key.ID, at.UnixMilli())
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("h1", time.Date(2026, time.August, 19, 12, 10, 0, 0, time.UTC))
	insert("h2", time.Date(2026, time.August, 19, 13, 10, 0, 0, time.UTC))
	insert("h3", time.Date(2026, time.July, 1, 8, 0, 0, 0, time.UTC))
	hourly, err := st.SummarizeUsageTrendFiltered(ctx, UsageFilter{PluginKeyID: key.ID}, "hour")
	if err != nil || len(hourly) != 3 || hourly[1].Date != "2026-08-19T12:00" || hourly[2].Date != "2026-08-19T13:00" {
		t.Fatalf("hourly = %#v, err = %v", hourly, err)
	}
	monthly, err := st.SummarizeUsageTrendFiltered(ctx, UsageFilter{PluginKeyID: key.ID}, "month")
	if err != nil || len(monthly) != 2 || monthly[0].Date != "2026-07" || monthly[1].Date != "2026-08" {
		t.Fatalf("monthly = %#v, err = %v", monthly, err)
	}
}

func TestListUsageAllowsNullEstimatedCost(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{QuotaMicroUSD: 10_000})
	at := time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)
	if _, err := st.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("disable foreign keys: %v", err)
	}
	_, err := st.db.ExecContext(ctx, `INSERT INTO usage_ledger(id, reservation_id, caller_id, plugin_key_id, model, input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, cost_micro_usd, source, created_at_unix_ms) VALUES ('u-null-est', 'r-null-est', ?, ?, 'grok-4.6', 10, 2, 0, 0, 0, 0, 1500, 'usage', ?)`, key.CallerID, key.ID, at.UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.ListUsage(ctx, UsageFilter{PluginKeyID: key.ID, Limit: 10})
	if err != nil {
		t.Fatalf("list usage: %v", err)
	}
	if len(got) != 1 || got[0].CostMicroUSD != 1500 || got[0].EstimatedCostMicroUSD != 1500 {
		t.Fatalf("usage = %#v", got)
	}
}

func TestSummarizeUsageByModelIncludesCacheAndTPS(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()
	key := newTestKey(t, ctx, st, PluginKeySpec{QuotaMicroUSD: 10_000})
	at := time.Date(2026, time.August, 19, 12, 0, 0, 0, time.UTC)
	if _, err := st.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("disable foreign keys: %v", err)
	}
	_, err := st.db.ExecContext(ctx, `INSERT INTO usage_ledger(id, reservation_id, caller_id, plugin_key_id, model, input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, cost_micro_usd, source, tokens_per_second, created_at_unix_ms) VALUES ('u1', 'r-u1', ?, ?, 'grok-4.6', 100, 10, 0, 0, 80, 0, 2000, 'usage', 40, ?)`, key.CallerID, key.ID, at.UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.db.ExecContext(ctx, `INSERT INTO usage_ledger(id, reservation_id, caller_id, plugin_key_id, model, input_tokens, output_tokens, reasoning_tokens, cached_tokens, cache_read_tokens, cache_creation_tokens, cost_micro_usd, source, tokens_per_second, created_at_unix_ms) VALUES ('u2', 'r-u2', ?, ?, 'grok-4.6', 50, 5, 0, 0, 20, 0, 1000, 'usage', 20, ?)`, key.CallerID, key.ID, at.UnixMilli())
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.SummarizeUsageByModelFiltered(ctx, UsageFilter{PluginKeyID: key.ID})
	if err != nil || len(got) != 1 {
		t.Fatalf("summaries = %#v, err = %v", got, err)
	}
	item := got[0]
	if item.Model != "grok-4.6" || item.RequestCount != 2 || item.CacheReadTokens != 100 || item.AvgTokensPerSecond == nil || *item.AvgTokensPerSecond != 30 {
		t.Fatalf("model summary = %#v", item)
	}
}

func TestResolvePricingRuleDisabledExactBlocksGlob(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()

	if err := st.PutPricingRule(ctx, PricingRule{
		ID: "all", MatchKind: MatchGlob, Pattern: "*", Priority: 1, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 2_000_000},
	}); err != nil {
		t.Fatalf("put glob: %v", err)
	}
	if err := st.PutPricingRule(ctx, PricingRule{
		ID: "gpt-4o", MatchKind: MatchExact, Pattern: "gpt-4o", Priority: 100, Enabled: false,
		Price: money.PricePerMTok{Input: 2_500_000, Output: 10_000_000},
	}); err != nil {
		t.Fatalf("put exact: %v", err)
	}

	rule, err := st.ResolvePricingRule(ctx, "gpt-4o")
	if err != nil {
		t.Fatalf("resolve gpt-4o: %v", err)
	}
	if rule.ID != "gpt-4o" || rule.Enabled {
		t.Fatalf("gpt-4o rule = %#v, want disabled exact", rule)
	}
}

func TestSetPricingRuleEnabled(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()

	if err := st.PutPricingRule(ctx, PricingRule{
		ID: "gpt-4o", MatchKind: MatchExact, Pattern: "gpt-4o", Priority: 100, Enabled: true,
		Price: money.PricePerMTok{Input: 1, Output: 2},
	}); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := st.SetPricingRuleEnabled(ctx, "gpt-4o", false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	rule, err := st.GetPricingRule(ctx, "gpt-4o")
	if err != nil || rule.Enabled {
		t.Fatalf("after disable = %#v, err=%v", rule, err)
	}
	if err := st.SetPricingRuleEnabled(ctx, "missing", false); !errors.Is(err, ErrPricingRuleNotFound) {
		t.Fatalf("missing error = %v, want %v", err, ErrPricingRuleNotFound)
	}
}

func TestWinningPricingRuleUsesPriorityAndDisabled(t *testing.T) {
	rules := []PricingRule{
		{ID: "all", MatchKind: MatchGlob, Pattern: "*", Priority: 1, Enabled: true},
		{ID: "gpt-4o", MatchKind: MatchExact, Pattern: "gpt-4o", Priority: 100, Enabled: false},
	}
	rule, err := WinningPricingRule(rules, "gpt-4o")
	if err != nil || rule.ID != "gpt-4o" || rule.Enabled {
		t.Fatalf("gpt-4o = %#v err=%v", rule, err)
	}
	rule, err = WinningPricingRule(rules, "claude-sonnet")
	if err != nil || rule.ID != "all" || !rule.Enabled {
		t.Fatalf("claude-sonnet = %#v err=%v", rule, err)
	}
}

func TestAuthConcurrencyLimitRoundTrip(t *testing.T) {
	ctx := context.Background()
	st := newTestStore(t)
	defer st.Close()

	got, err := st.GetAuthConcurrencyLimit(ctx, "codex", "auth-1")
	if err != nil || got != 0 {
		t.Fatalf("missing limit = %d, %v", got, err)
	}
	if err := st.UpsertAuthConcurrencyLimit(ctx, "codex", "auth-1", 3); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err = st.GetAuthConcurrencyLimit(ctx, "codex", "auth-1")
	if err != nil || got != 3 {
		t.Fatalf("get = %d, %v", got, err)
	}
	if err := st.UpsertAuthConcurrencyLimit(ctx, "codex", "auth-1", 0); err != nil {
		t.Fatalf("clear: %v", err)
	}
	got, err = st.GetAuthConcurrencyLimit(ctx, "codex", "auth-1")
	if err != nil || got != 0 {
		t.Fatalf("cleared = %d, %v", got, err)
	}
	if err := st.UpsertAuthConcurrencyLimit(ctx, "codex", "auth-2", 1); err != nil {
		t.Fatalf("second: %v", err)
	}
	listed, err := st.ListAuthConcurrencyLimits(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if listed["codex\x00auth-2"] != 1 {
		t.Fatalf("list = %#v", listed)
	}
	if err := st.UpsertAuthConcurrencyLimits(ctx, []AuthConcurrencyLimit{
		{Provider: "claude", AuthID: "a", MaxConcurrentRequests: 4},
		{Provider: "claude", AuthID: "b", MaxConcurrentRequests: 5},
	}); err != nil {
		t.Fatalf("batch: %v", err)
	}
	listed, err = st.ListAuthConcurrencyLimits(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if listed["claude\x00a"] != 4 || listed["claude\x00b"] != 5 {
		t.Fatalf("batch list = %#v", listed)
	}
	if err := st.UpsertAuthConcurrencyLimit(ctx, "codex", "auth-1", -1); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("negative error = %v", err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(context.Background(), filepath.Join(t.TempDir(), "credit-manager.db"), OpenOptions{BusyTimeout: time.Second})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if _, err := st.CreateCaller(context.Background(), CallerSpec{ID: "caller", QuotaMicroUSD: 0, Enabled: true}); err != nil {
		t.Fatalf("create caller: %v", err)
	}
	return st
}

func newTestKey(t *testing.T, ctx context.Context, st *Store, limits PluginKeySpec) PluginKey {
	t.Helper()
	limits.CallerID = "caller"
	if limits.Kid == "" {
		limits.Kid = "test-key"
	}
	limits.KeyHash = []byte("test-key-hash-" + limits.Kid)
	limits.PepperID = "active"
	limits.Fingerprint = "fingerprint"
	if limits.Principal == "" {
		limits.Principal = "credit-manager:" + limits.Kid
	}
	if limits.CallerScope == "" {
		limits.CallerScope = "credit-manager:" + limits.Kid
	}
	limits.Enabled = true
	key, err := st.CreatePluginKey(ctx, limits)
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	return key
}

func reserveRequest(key PluginKey, idempotency string, amount money.MicroUSD) ReserveRequest {
	return ReserveRequest{
		CallerID:             key.CallerID,
		PluginKeyID:          key.ID,
		IdempotencyKey:       idempotency,
		Model:                "test-model",
		RequestTokenEstimate: 1,
		AmountMicroUSD:       amount,
	}
}
