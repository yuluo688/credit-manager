package service

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yuluo688/credit-manager/internal/store"
)

const (
	// A window whose start equals the sync time with 0% used is the "no active
	// window" placeholder (upstream reports reset = now + duration).
	quotaFakeStartTolerance = 2 * time.Minute
	// Upstream reset_at jitters by a few seconds between syncs.
	quotaSameStartTolerance = 2 * time.Minute
	quotaHistoryMaxInferred = 400
)

// quotaLedgerProbe returns the first/last ledger request in [from, to).
type quotaLedgerProbe func(from, to time.Time) (first, last time.Time, ok bool)

type quotaHistoryWindow struct {
	Start, End, Scheduled time.Time
	Duration              time.Duration
	Used                  *float64
	Label                 string
	Source                string
	EndReason             string
	Approximate           bool
}

type quotaHistoryObs struct {
	start, sched time.Time
	dur          time.Duration
	used         *float64
	firstUsed    *float64
	firstSeen    time.Time
	lastSeen     time.Time
	label        string
	source       string
	approx       bool
	fake         bool
}

// buildQuotaWindowHistory turns the raw observations of ONE window family into
// a non-overlapping, start-ordered list of windows (past and possibly current).
// Pure apart from probe so it can be unit-tested.
func buildQuotaWindowHistory(raw []store.AuthQuotaWindowObservation, now time.Time, probe quotaLedgerProbe) []quotaHistoryWindow {
	if len(raw) == 0 {
		return nil
	}
	obs := make([]quotaHistoryObs, 0, len(raw))
	for _, r := range raw {
		o := quotaHistoryObs{
			start:     time.UnixMilli(r.StartUnixMS).UTC(),
			sched:     time.UnixMilli(r.ResetUnixMS).UTC(),
			dur:       time.Duration(r.DurationSeconds) * time.Second,
			used:      r.Used,
			firstUsed: r.FirstUsed,
			label:     r.Label,
			source:    r.Source,
		}
		if r.FirstSeenUnixMS > 0 {
			o.firstSeen = time.UnixMilli(r.FirstSeenUnixMS).UTC()
		}
		if r.LastSeenUnixMS > 0 {
			o.lastSeen = time.UnixMilli(r.LastSeenUnixMS).UTC()
		}
		if o.dur <= 0 {
			o.dur = o.sched.Sub(o.start)
		}
		if o.sched.After(o.start) {
			obs = append(obs, o)
		}
	}
	obs = mergeQuotaHistoryObs(obs)

	// Fake starts: replace the start by the first request after the sync, or
	// drop the placeholder (the window never started) but keep the sync time
	// as proof that the previous window had already ended.
	var boundaries []time.Time
	kept := obs[:0]
	for _, o := range obs {
		if o.source != "plan_history" && !o.firstSeen.IsZero() && absDuration(o.start.Sub(o.firstSeen)) <= quotaFakeStartTolerance &&
			(o.firstUsed == nil || *o.firstUsed == 0) {
			boundaries = append(boundaries, o.firstSeen)
			if probe == nil {
				continue
			}
			first, _, ok := probe(o.firstSeen, minTime(o.sched, now))
			if !ok {
				continue
			}
			o.start = first
			o.sched = first.Add(o.dur)
			o.approx = true
			o.fake = true
			o.source = "ledger_first_request"
		}
		kept = append(kept, o)
	}
	obs = mergeQuotaHistoryObs(kept)

	out := make([]quotaHistoryWindow, 0, len(obs))
	for i, o := range obs {
		w := quotaHistoryWindow{Start: o.start, End: o.sched, Scheduled: o.sched, Duration: o.dur, Used: o.used, Label: o.label, Source: o.source, EndReason: "natural", Approximate: o.approx}
		if i+1 < len(obs) && obs[i+1].start.Before(w.End) {
			w.End = obs[i+1].start
			w.EndReason = "early_reset"
		}
		for _, b := range boundaries {
			if b.After(w.Start) && b.Before(w.End) {
				w.End = b
				w.EndReason = "early_reset"
				w.Approximate = true
			}
		}
		out = append(out, w)
		// Gap until the next observed window (or now): requests there prove
		// a window was running; infer it from the first request.
		gapEnd := now
		if i+1 < len(obs) {
			gapEnd = obs[i+1].start
		}
		out = append(out, inferQuotaGapWindows(w.End, gapEnd, o.dur, o.label, now, i+1 < len(obs), probe)...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

func inferQuotaGapWindows(from, to time.Time, dur time.Duration, label string, now time.Time, bounded bool, probe quotaLedgerProbe) []quotaHistoryWindow {
	if probe == nil || dur <= 0 || !to.After(from) {
		return nil
	}
	var out []quotaHistoryWindow
	for n := 0; n < quotaHistoryMaxInferred && to.After(from); n++ {
		first, _, ok := probe(from, to)
		if !ok {
			break
		}
		w := quotaHistoryWindow{Start: first, End: first.Add(dur), Scheduled: first.Add(dur), Duration: dur, Label: label, Source: "ledger_inferred", EndReason: "natural", Approximate: true}
		if bounded && w.End.After(to) {
			w.End = to
			w.EndReason = "early_reset"
		}
		out = append(out, w)
		if !w.End.After(from) || w.End.After(now) {
			break
		}
		from = w.End
	}
	return out
}

// mergeQuotaHistoryObs sorts by start and folds near-duplicate starts.
func mergeQuotaHistoryObs(obs []quotaHistoryObs) []quotaHistoryObs {
	sort.SliceStable(obs, func(i, j int) bool { return obs[i].start.Before(obs[j].start) })
	out := make([]quotaHistoryObs, 0, len(obs))
	for _, o := range obs {
		if n := len(out); n > 0 && absDuration(o.start.Sub(out[n-1].start)) <= quotaSameStartTolerance {
			out[n-1] = foldQuotaHistoryObs(out[n-1], o)
			continue
		}
		out = append(out, o)
	}
	return out
}

func quotaObsRank(o quotaHistoryObs) int {
	switch {
	case o.source == "plan_history":
		return 3
	case !o.fake && !o.approx:
		return 2
	default:
		return 1
	}
}

func foldQuotaHistoryObs(a, b quotaHistoryObs) quotaHistoryObs {
	keep, other := a, b
	if quotaObsRank(b) > quotaObsRank(a) {
		keep, other = b, a
	}
	if keep.firstSeen.IsZero() || (!other.firstSeen.IsZero() && other.firstSeen.Before(keep.firstSeen)) {
		if !other.firstSeen.IsZero() && keep.source != "plan_history" && !keep.fake {
			keep.firstSeen = other.firstSeen
			keep.firstUsed = other.firstUsed
		}
	}
	if keep.used == nil || (other.used != nil && *other.used > *keep.used && other.lastSeen.After(keep.lastSeen)) {
		if keep.source != "plan_history" {
			keep.used = other.used
		}
	}
	if keep.label == "" {
		keep.label = other.label
	}
	return keep
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

var quotaHistorySeeded sync.Map

// seedQuotaWindowObservations backfills observations from the legacy
// baselines table and the stored snapshot (idempotent INSERT OR IGNORE).
func (s *Service) seedQuotaWindowObservations(ctx context.Context, provider, authID string, snap *quotaSnapshot, force bool) {
	key := s.cfg.DatabasePath() + "|" + provider + "|" + authID
	if _, done := quotaHistorySeeded.Load(key); done && !force {
		return
	}
	quotaHistorySeeded.Store(key, true)
	baselines, err := s.store.ListAuthQuotaWindowBaselines(ctx, provider, authID)
	if err == nil {
		for _, b := range baselines {
			start, end, ok := parseQuotaCycleKey(b.CycleKey)
			if !ok {
				continue
			}
			used := b.BaselineUsed
			seen := b.CreatedAt.UTC().UnixMilli()
			_ = s.store.InsertAuthQuotaWindowObservationIfMissing(ctx, store.AuthQuotaWindowObservation{
				Provider: provider, AuthID: authID, WindowID: b.WindowID, StartUnixMS: start, ResetUnixMS: end,
				DurationSeconds: (end - start) / 1000, Used: &used, FirstUsed: &used,
				FirstSeenUnixMS: seen, LastSeenUnixMS: seen, Source: "baseline",
			})
		}
	}
	if snap != nil {
		for _, w := range snap.Windows {
			if w.Partial || !quotaWindowTracksCycle(w) || w.CycleStartAt == nil || w.ResetsAt == nil || w.HistorySource != "" {
				continue
			}
			o := quotaWindowObservation(provider, authID, w, 0, "snapshot")
			_ = s.store.InsertAuthQuotaWindowObservationIfMissing(ctx, o)
		}
	}
}

func parseQuotaCycleKey(key string) (int64, int64, bool) {
	a, b, ok := strings.Cut(key, ":")
	if !ok {
		return 0, 0, false
	}
	start, e1 := strconv.ParseInt(a, 10, 64)
	end, e2 := strconv.ParseInt(b, 10, 64)
	if e1 != nil || e2 != nil || start <= 0 || end <= start {
		return 0, 0, false
	}
	return start, end, true
}

func quotaWindowObservation(provider, authID string, w AuthQuotaWindow, seenMS int64, source string) store.AuthQuotaWindowObservation {
	start, end, _ := quotaWindowBounds(w)
	dur := int64(0)
	if w.DurationSeconds != nil {
		dur = *w.DurationSeconds
	} else {
		dur = int64(end.Sub(start).Seconds())
	}
	return store.AuthQuotaWindowObservation{
		Provider: provider, AuthID: authID, WindowID: windowBaselineID(&w),
		StartUnixMS: start.UTC().UnixMilli(), ResetUnixMS: end.UTC().UnixMilli(), DurationSeconds: dur,
		Label: w.Label, Used: w.Used, FirstUsed: w.Used, FirstSeenUnixMS: seenMS, LastSeenUnixMS: seenMS, Source: source,
	}
}

// recordQuotaObservations stores every tracked window seen by a successful
// sync, plus plan_limit_history periods, and infers used reset credits.
func (s *Service) recordQuotaObservations(ctx context.Context, provider, authID string, prev *quotaSnapshot, prevSuccess *time.Time, snap quotaSnapshot, now time.Time) {
	nowMS := now.UTC().UnixMilli()
	ambiguous := ambiguousQuotaWindowIDs(snap.Windows, now)
	for _, w := range snap.Windows {
		if w.Partial || w.ResetsAt == nil || !quotaWindowTracksCycle(w) || ambiguous[windowBaselineID(&w)] {
			continue
		}
		if _, _, ok := quotaWindowBounds(w); !ok {
			continue
		}
		_ = s.store.UpsertAuthQuotaWindowObservation(ctx, quotaWindowObservation(provider, authID, w, nowMS, "sync"))
	}
	for _, p := range snap.planHistory {
		id := ""
		for _, w := range snap.Windows {
			if strings.HasPrefix(w.ID, "rate-limit-") && w.DurationSeconds != nil && *w.DurationSeconds == p.WindowMinutes*60 {
				id = w.ID
			}
		}
		if id == "" {
			continue
		}
		_ = s.store.UpsertAuthQuotaWindowObservation(ctx, store.AuthQuotaWindowObservation{
			Provider: provider, AuthID: authID, WindowID: id, StartUnixMS: p.StartsAt.UnixMilli(), ResetUnixMS: p.EndsAt.UnixMilli(),
			DurationSeconds: p.WindowMinutes * 60, Used: p.Used, FirstUsed: p.Used, LastSeenUnixMS: nowMS, Source: "plan_history",
		})
	}
	if !snap.resetCreditsFetched {
		return
	}
	credits := make([]store.AuthQuotaResetCredit, 0, len(snap.ResetCreditList))
	for _, c := range snap.ResetCreditList {
		credits = append(credits, store.AuthQuotaResetCredit{CreditID: c.ID, ResetType: c.ResetType, Status: c.Status, GrantedAt: c.GrantedAt, ExpiresAt: c.ExpiresAt})
	}
	gone, err := s.store.SyncAuthQuotaResetCredits(ctx, provider, authID, credits, nowMS)
	if err != nil {
		return
	}
	after := int64(0)
	if prevSuccess != nil {
		after = prevSuccess.UTC().UnixMilli()
	}
	recorded := false
	for _, c := range gone {
		if c.Status != "" && !strings.EqualFold(c.Status, "available") {
			continue
		}
		if c.ExpiresAt != nil && !c.ExpiresAt.After(now) {
			continue // expired, not used
		}
		from := after
		if from == 0 || c.LastSeenUnixMS > from {
			from = c.LastSeenUnixMS
		}
		_ = s.store.InsertAuthQuotaResetEvent(ctx, provider, authID, store.AuthQuotaResetEvent{Kind: "credit_used", AfterUnixMS: from, BeforeUnixMS: nowMS, CreditID: c.CreditID, Detail: "credit disappeared"})
		recorded = true
	}
	if !recorded && after > 0 && prev != nil && prev.ResetCredits != nil && snap.ResetCredits != nil && *snap.ResetCredits < *prev.ResetCredits {
		_ = s.store.InsertAuthQuotaResetEvent(ctx, provider, authID, store.AuthQuotaResetEvent{Kind: "credit_used", AfterUnixMS: after, BeforeUnixMS: nowMS,
			Detail: "available_count " + strconv.FormatFloat(*prev.ResetCredits, 'f', -1, 64) + "->" + strconv.FormatFloat(*snap.ResetCredits, 'f', -1, 64)})
	}
}

// applyQuotaWindowHistory replaces the ended/partial tracked windows in the
// item with history rebuilt from the observation table.
func (s *Service) applyQuotaWindowHistory(ctx context.Context, item *AuthQuotaOverviewItem, now time.Time) {
	obs, err := s.store.ListAuthQuotaWindowObservations(ctx, item.Provider, item.AuthID)
	if err != nil || len(obs) == 0 {
		return
	}
	families := map[string][]store.AuthQuotaWindowObservation{}
	for _, o := range obs {
		families[o.WindowID] = append(families[o.WindowID], o)
	}
	ambiguous := ambiguousQuotaWindowIDs(item.Windows, now)
	templates := map[string]AuthQuotaWindow{}
	current := map[string][]AuthQuotaWindow{}
	originals := map[string][]AuthQuotaWindow{}
	kept := make([]AuthQuotaWindow, 0, len(item.Windows))
	for _, w := range item.Windows {
		id := windowBaselineID(&w)
		if _, ok := families[id]; !ok || w.ResetsAt == nil || !quotaWindowTracksCycle(w) || ambiguous[id] {
			kept = append(kept, w)
			continue
		}
		originals[id] = append(originals[id], w)
		if t, ok := templates[id]; !ok || quotaTemplateBetter(w, t) {
			templates[id] = w
		}
		if !w.Partial && w.HistorySource == "" && w.ResetsAt != nil && w.ResetsAt.After(now) {
			kept = append(kept, w)
			current[id] = append(current[id], w)
		}
	}
	probe := func(from, to time.Time) (time.Time, time.Time, bool) {
		if !to.After(from) {
			return time.Time{}, time.Time{}, false
		}
		f, l, ok, err := s.store.AuthQuotaRequestBounds(ctx, item.Provider, item.AuthID, item.AuthIndex, from, to)
		return f, l, ok && err == nil
	}
	ids := make([]string, 0, len(templates))
	for id := range templates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		template := templates[id]
		if !quotaTemplateRecent(template, now) {
			// Legacy window (e.g. an ID that used to be weekly): no live
			// family to anchor inference to, so leave its windows as-is.
			if len(current[id]) == 0 {
				kept = append(kept, originals[id]...)
			}
			continue
		}
		built := buildQuotaWindowHistory(sameDurationObservations(families[id], template), now, probe)
		var ended []AuthQuotaWindow
		for _, h := range built {
			end := h.End
			for _, cur := range current[id] {
				if cs, _, ok := quotaWindowBounds(cur); ok && cs.Before(end) {
					end = cs
					if h.EndReason == "natural" && cs.Before(h.Scheduled) {
						h.EndReason = "early_reset"
					}
				}
			}
			if end.After(now) || !end.After(h.Start) {
				continue
			}
			h.End = end
			ended = append(ended, quotaHistoryToWindow(template, h))
		}
		if len(ended) > authQuotaWindowHistoryLimit {
			ended = ended[len(ended)-authQuotaWindowHistoryLimit:]
		}
		kept = append(kept, ended...)
	}
	item.Windows = kept
}

func quotaHistoryToWindow(t AuthQuotaWindow, h quotaHistoryWindow) AuthQuotaWindow {
	w := t
	start, end := h.Start, h.End
	w.CycleStartAt, w.ResetsAt = &start, &end
	w.CycleStartSource = h.Source
	w.HistorySource = h.Source
	w.EndReason = h.EndReason
	w.Approximate = h.Approximate
	w.Partial = false
	w.ScheduledResetAt = nil
	if h.EndReason == "early_reset" {
		sched := h.Scheduled
		w.ScheduledResetAt = &sched
	}
	if h.Label != "" && w.Label == "" {
		w.Label = h.Label
	}
	w.Used, w.Remaining, w.UsedRatio, w.RemainingRatio = nil, nil, nil, nil
	if h.Used != nil {
		u := *h.Used
		w.Used = &u
		if w.Limit != nil && *w.Limit > 0 {
			r := math.Max(0, *w.Limit-u)
			ur, rr := u / *w.Limit, r / *w.Limit
			w.Remaining, w.UsedRatio, w.RemainingRatio = &r, &ur, &rr
		}
	}
	w.LocalUsage = nil
	w.LocalAttributionStatus = ""
	w.PredictionAvailable = false
	w.AverageTokensPerRequest = nil
	w.EstimatedRemainingRequests = nil
	w.BaselineUsed = nil
	w.ObservedUsed = nil
	return w
}

func decodeQuotaSnapshot(raw string) (quotaSnapshot, bool) {
	var snap quotaSnapshot
	if strings.TrimSpace(raw) == "" || json.Unmarshal([]byte(raw), &snap) != nil {
		return quotaSnapshot{}, false
	}
	return snap, true
}

// ambiguousQuotaWindowIDs flags IDs shared by more than one current upstream
// window (e.g. two kimi "weekly" entries); their history cannot be separated.
func ambiguousQuotaWindowIDs(windows []AuthQuotaWindow, now time.Time) map[string]bool {
	count := map[string]int{}
	for _, w := range windows {
		if w.Partial || w.HistorySource != "" || w.ResetsAt == nil || !w.ResetsAt.After(now) {
			continue
		}
		count[windowBaselineID(&w)]++
	}
	out := map[string]bool{}
	for id, n := range count {
		if n > 1 {
			out[id] = true
		}
	}
	return out
}

// quotaTemplateBetter prefers non-partial windows, then the latest reset, so a
// family is shaped by its newest upstream window.
func quotaTemplateBetter(a, b AuthQuotaWindow) bool {
	if a.Partial != b.Partial {
		return !a.Partial
	}
	if a.ResetsAt == nil || b.ResetsAt == nil {
		return b.ResetsAt == nil && a.ResetsAt != nil
	}
	return a.ResetsAt.After(*b.ResetsAt)
}

// sameDurationObservations keeps observations whose period matches the
// template, so a window ID that changed length upstream is not mixed.
func sameDurationObservations(obs []store.AuthQuotaWindowObservation, template AuthQuotaWindow) []store.AuthQuotaWindowObservation {
	if template.DurationSeconds == nil || *template.DurationSeconds <= 0 {
		return obs
	}
	want := *template.DurationSeconds
	out := obs[:0:0]
	for _, o := range obs {
		d := o.DurationSeconds
		if d <= 0 {
			d = (o.ResetUnixMS - o.StartUnixMS) / 1000
		}
		if d-want <= 60 && want-d <= 60 {
			out = append(out, o)
		}
	}
	return out
}

func quotaTemplateRecent(w AuthQuotaWindow, now time.Time) bool {
	if w.ResetsAt == nil {
		return false
	}
	grace := time.Duration(0)
	if w.DurationSeconds != nil {
		grace = time.Duration(*w.DurationSeconds) * time.Second
	}
	return w.ResetsAt.Add(grace).After(now)
}
