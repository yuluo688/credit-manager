package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
)

const (
	authPendingTTL       = 15 * time.Minute
	authLooseMatchWindow = 45 * time.Second
)

type pendingAuthCapture struct {
	reservationID string
	models        []string
	startedAt     time.Time
	ledgerID      string
	auth          store.AuthIdentity
	hasAuth       bool
	usage         money.TokenUsage
	hasUsage      bool
}

// TrackAuthCapture registers a reservation that will later receive selected auth identity.
// Extra model names (aliases / rewritten upstream ids) improve usage correlation.
func (s *Service) TrackAuthCapture(reservationID string, models ...string) {
	if s == nil {
		return
	}
	reservationID = strings.TrimSpace(reservationID)
	cleaned := uniqueModels(models...)
	if reservationID == "" || len(cleaned) == 0 {
		return
	}
	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.ensureAuthPendingLocked()
	s.pruneAuthPendingLocked(time.Now())
	s.authPending[reservationID] = &pendingAuthCapture{
		reservationID: reservationID,
		models:        cleaned,
		startedAt:     time.Now(),
	}
}

// ObserveHostUsage correlates a host usage record to a pending reservation and
// returns the ledger id when the ledger row already exists and needs an update.
// Usage callbacks can lack credential data, so auth is optional here.
func (s *Service) ObserveHostUsage(requestedAt time.Time, auth store.AuthIdentity, usage money.TokenUsage, models ...string) (ledgerID string, ok bool) {
	if s == nil {
		return "", false
	}
	modelSet := modelSetOf(models...)
	if len(modelSet) == 0 {
		return "", false
	}

	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.ensureAuthPendingLocked()
	s.pruneAuthPendingLocked(time.Now())

	best := s.pickPendingAuthLocked(requestedAt, auth, usage, modelSet, true)
	if best == nil {
		best = s.pickPendingAuthLocked(requestedAt, auth, usage, modelSet, false)
	}
	if best == nil {
		return "", false
	}
	if !auth.Empty() && !best.hasAuth {
		best.auth = auth
		best.hasAuth = true
	}
	if !best.hasUsage && usageFound(usage) {
		best.usage = usage
		best.hasUsage = true
	}
	s.signalAuthPendingLocked()
	if strings.TrimSpace(best.ledgerID) == "" {
		return "", true
	}
	ledgerID = best.ledgerID
	if best.hasAuth && best.hasUsage {
		delete(s.authPending, best.reservationID)
	}
	return ledgerID, true
}

func (s *Service) pickPendingAuthLocked(requestedAt time.Time, auth store.AuthIdentity, usage money.TokenUsage, modelSet map[string]struct{}, requireModelMatch bool) *pendingAuthCapture {
	requestedAtKnown := !requestedAt.IsZero()
	var eligible []*pendingAuthCapture
	for _, pending := range s.authPending {
		if pending == nil {
			continue
		}
		canUseUsage := !pending.hasUsage && usageFound(usage)
		canUseAuth := !pending.hasAuth && !auth.Empty()
		if !canUseUsage && !canUseAuth {
			continue
		}
		if requireModelMatch && !pendingMatchesModels(pending, modelSet) {
			continue
		}
		eligible = append(eligible, pending)
	}
	if !requireModelMatch && len(eligible) == 1 {
		return eligible[0]
	}
	var best *pendingAuthCapture
	bestDelta := time.Duration(1<<63 - 1)
	for _, pending := range eligible {
		delta := pendingTimeDelta(pending, requestedAt, requestedAtKnown)
		if !requireModelMatch && len(eligible) > 1 && delta > authLooseMatchWindow {
			continue
		}
		if best == nil || betterPendingMatch(pending, best, delta, bestDelta, requestedAtKnown) {
			best = pending
			bestDelta = delta
		}
	}
	return best
}

func pendingTimeDelta(pending *pendingAuthCapture, requestedAt time.Time, requestedAtKnown bool) time.Duration {
	if pending == nil {
		return time.Duration(1<<63 - 1)
	}
	if requestedAtKnown {
		delta := requestedAt.Sub(pending.startedAt)
		if delta < 0 {
			return -delta
		}
		return delta
	}
	delta := time.Since(pending.startedAt)
	if delta < 0 {
		return -delta
	}
	return delta
}

func betterPendingMatch(candidate, best *pendingAuthCapture, candidateDelta, bestDelta time.Duration, requestedAtKnown bool) bool {
	if best == nil {
		return true
	}
	if requestedAtKnown {
		if candidateDelta < bestDelta {
			return true
		}
		if candidateDelta > bestDelta {
			return false
		}
		return candidate.startedAt.Before(best.startedAt)
	}
	return candidate.startedAt.Before(best.startedAt)
}

// CapturedHostUsage returns final host usage that arrived before settlement.
func (s *Service) CapturedHostUsage(reservationID string) (money.TokenUsage, bool) {
	if s == nil {
		return money.TokenUsage{}, false
	}
	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.pruneAuthPendingLocked(time.Now())
	pending := s.authPending[strings.TrimSpace(reservationID)]
	if pending == nil || !pending.hasUsage {
		return money.TokenUsage{}, false
	}
	return pending.usage, true
}

// WaitForHostUsage blocks until usage.handle fills the reservation or the timeout elapses.
func (s *Service) WaitForHostUsage(ctx context.Context, reservationID string, timeout time.Duration) (money.TokenUsage, bool) {
	if s == nil || timeout <= 0 {
		return s.CapturedHostUsage(reservationID)
	}
	reservationID = strings.TrimSpace(reservationID)
	if reservationID == "" {
		return money.TokenUsage{}, false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.ensureAuthPendingLocked()

	timedOut := false
	timer := time.AfterFunc(timeout, func() {
		s.authMu.Lock()
		timedOut = true
		s.signalAuthPendingLocked()
		s.authMu.Unlock()
	})
	defer timer.Stop()

	stop := context.AfterFunc(ctx, func() {
		s.authMu.Lock()
		s.signalAuthPendingLocked()
		s.authMu.Unlock()
	})
	defer stop()

	for {
		s.pruneAuthPendingLocked(time.Now())
		if pending := s.authPending[reservationID]; pending != nil && pending.hasUsage {
			return pending.usage, true
		}
		if timedOut || ctx.Err() != nil {
			return money.TokenUsage{}, false
		}
		s.authCond.Wait()
	}
}

func usageFound(usage money.TokenUsage) bool {
	return usage.Input > 0 || usage.Output > 0 || usage.Reasoning > 0 || usage.Cached > 0 || usage.CacheRead > 0 || usage.CacheCreation > 0
}

func uniqueModels(models ...string) []string {
	seen := make(map[string]struct{}, len(models))
	out := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		key := strings.ToLower(model)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, model)
	}
	return out
}

func modelSetOf(models ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(models))
	for _, model := range models {
		if model = strings.TrimSpace(model); model != "" {
			out[strings.ToLower(model)] = struct{}{}
		}
	}
	return out
}

func pendingMatchesModels(pending *pendingAuthCapture, modelSet map[string]struct{}) bool {
	if pending == nil || len(modelSet) == 0 {
		return false
	}
	for _, model := range pending.models {
		if _, ok := modelSet[strings.ToLower(model)]; ok {
			return true
		}
	}
	return false
}

// AuthForSettlement returns any already-captured auth for the reservation and
// keeps the pending entry open when auth is still missing so a later usage.handle can fill it.
func (s *Service) AuthForSettlement(reservationID, ledgerID string) store.AuthIdentity {
	if s == nil {
		return store.AuthIdentity{}
	}
	reservationID = strings.TrimSpace(reservationID)
	ledgerID = strings.TrimSpace(ledgerID)
	if reservationID == "" {
		return store.AuthIdentity{}
	}

	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.ensureAuthPendingLocked()
	s.pruneAuthPendingLocked(time.Now())
	pending := s.authPending[reservationID]
	if pending == nil {
		return store.AuthIdentity{}
	}
	if pending.hasAuth {
		auth := pending.auth
		if pending.hasUsage {
			delete(s.authPending, reservationID)
		} else if ledgerID != "" {
			pending.ledgerID = ledgerID
		}
		return auth
	}
	if ledgerID != "" {
		pending.ledgerID = ledgerID
	}
	return store.AuthIdentity{}
}

// CancelAuthCapture drops a pending capture when the reservation is released without settle.
func (s *Service) CancelAuthCapture(reservationID string) {
	if s == nil {
		return
	}
	reservationID = strings.TrimSpace(reservationID)
	if reservationID == "" {
		return
	}
	s.authMu.Lock()
	defer s.authMu.Unlock()
	if s.authPending != nil {
		delete(s.authPending, reservationID)
	}
	s.signalAuthPendingLocked()
}

func (s *Service) ensureAuthPendingLocked() {
	if s.authPending == nil {
		s.authPending = make(map[string]*pendingAuthCapture)
	}
	if s.authCond == nil {
		s.authCond = sync.NewCond(&s.authMu)
	}
}

func (s *Service) signalAuthPendingLocked() {
	if s.authCond != nil {
		s.authCond.Broadcast()
	}
}

func (s *Service) pruneAuthPendingLocked(now time.Time) {
	for id, pending := range s.authPending {
		if pending == nil || now.Sub(pending.startedAt) > authPendingTTL {
			delete(s.authPending, id)
		}
	}
}
