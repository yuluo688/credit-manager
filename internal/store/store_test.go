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
