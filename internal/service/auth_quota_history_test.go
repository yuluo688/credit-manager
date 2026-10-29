package service

import (
	"testing"
	"time"

	"github.com/yuluo688/credit-manager/internal/store"
)

func TestBuildQuotaWindowHistoryFallingcliffScenario(t *testing.T) {
	loc := time.UTC
	at := func(d, h, m, s int) time.Time { return time.Date(2026, 10, d, h, m, s, 0, loc) }
	week := int64(7 * 24 * 3600)
	f := func(v float64) *float64 { return &v }
	obs := func(start time.Time, seen time.Time, firstUsed *float64) store.AuthQuotaWindowObservation {
		o := store.AuthQuotaWindowObservation{WindowID: "secondary", StartUnixMS: start.UnixMilli(), ResetUnixMS: start.Add(7 * 24 * time.Hour).UnixMilli(), DurationSeconds: week, Used: f(10), FirstUsed: firstUsed, Source: "sync"}
		if !seen.IsZero() {
			o.FirstSeenUnixMS = seen.UnixMilli()
		}
		return o
	}
	raw := []store.AuthQuotaWindowObservation{
		obs(at(11, 10, 52, 35), time.Time{}, nil),
		obs(at(20, 6, 35, 6), at(20, 9, 0, 0), f(3)),
		obs(at(20, 6, 35, 9), at(21, 9, 0, 0), f(5)),    // reset_at jitter -> dedupe
		obs(at(23, 2, 27, 44), at(23, 2, 27, 44), f(0)), // fake start while idle
		obs(at(28, 6, 34, 59), at(28, 7, 0, 0), f(1)),
	}
	requests := []time.Time{at(19, 2, 58, 22), at(19, 5, 0, 0), at(23, 2, 41, 19), at(28, 7, 30, 0)}
	probe := func(from, to time.Time) (time.Time, time.Time, bool) {
		var first, last time.Time
		for _, r := range requests {
			if !r.Before(from) && r.Before(to) {
				if first.IsZero() {
					first = r
				}
				last = r
			}
		}
		return first, last, !first.IsZero()
	}
	got := buildQuotaWindowHistory(raw, at(29, 12, 0, 0), probe)
	want := []struct {
		start, end time.Time
		reason     string
		approx     bool
	}{
		{at(11, 10, 52, 35), at(18, 10, 52, 35), "natural", false},
		{at(19, 2, 58, 22), at(20, 6, 35, 6), "early_reset", true},
		{at(20, 6, 35, 6), at(23, 2, 27, 44), "early_reset", true},
		{at(23, 2, 41, 19), at(28, 6, 34, 59), "early_reset", true},
		{at(28, 6, 34, 59), at(4+31, 6, 34, 59), "natural", false},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d windows: %#v", len(got), got)
	}
	for i, w := range want {
		g := got[i]
		if !g.Start.Equal(w.start) || !g.End.Equal(w.end) || g.EndReason != w.reason || g.Approximate != w.approx {
			t.Fatalf("window %d = %s..%s %s approx=%v source=%s; want %s..%s %s approx=%v", i, g.Start, g.End, g.EndReason, g.Approximate, g.Source, w.start, w.end, w.reason, w.approx)
		}
		if i > 0 && got[i-1].End.After(g.Start) {
			t.Fatalf("overlap between %d and %d", i-1, i)
		}
	}
}

func TestBuildQuotaWindowHistoryDropsFakeStartWithoutRequests(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	zero := 0.0
	raw := []store.AuthQuotaWindowObservation{{WindowID: "w", StartUnixMS: start.UnixMilli(), ResetUnixMS: start.Add(time.Hour).UnixMilli(), DurationSeconds: 3600, FirstUsed: &zero, FirstSeenUnixMS: start.UnixMilli()}}
	none := func(from, to time.Time) (time.Time, time.Time, bool) { return time.Time{}, time.Time{}, false }
	if got := buildQuotaWindowHistory(raw, start.Add(3*time.Hour), none); len(got) != 0 {
		t.Fatalf("fake start without requests should vanish: %#v", got)
	}
}
