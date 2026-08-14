package service

import (
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
)

const authPendingTTL = 15 * time.Minute

type pendingAuthCapture struct {
	reservationID string
	model         string
	startedAt     time.Time
	ledgerID      string
	auth          store.AuthIdentity
	hasAuth       bool
	usage         money.TokenUsage
	hasUsage      bool
}

// TrackAuthCapture registers a reservation that will later receive selected auth identity.
func (s *Service) TrackAuthCapture(reservationID, model string) {
	if s == nil {
		return
	}
	reservationID = strings.TrimSpace(reservationID)
	model = strings.TrimSpace(model)
	if reservationID == "" || model == "" {
		return
	}
	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.ensureAuthPendingLocked()
	s.pruneAuthPendingLocked(time.Now())
	s.authPending[reservationID] = &pendingAuthCapture{
		reservationID: reservationID,
		model:         model,
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
	modelSet := make(map[string]struct{}, len(models))
	for _, model := range models {
		if model = strings.TrimSpace(model); model != "" {
			modelSet[strings.ToLower(model)] = struct{}{}
		}
	}
	if len(modelSet) == 0 {
		return "", false
	}
	now := time.Now()
	if requestedAt.IsZero() {
		requestedAt = now
	}

	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.ensureAuthPendingLocked()
	s.pruneAuthPendingLocked(now)

	var best *pendingAuthCapture
	bestDelta := time.Duration(1<<63 - 1)
	for _, pending := range s.authPending {
		if pending == nil || pending.hasUsage {
			continue
		}
		if _, matches := modelSet[strings.ToLower(pending.model)]; !matches {
			continue
		}
		delta := requestedAt.Sub(pending.startedAt)
		if delta < 0 {
			delta = -delta
		}
		if best == nil || delta < bestDelta || (delta == bestDelta && pending.startedAt.Before(best.startedAt)) {
			best = pending
			bestDelta = delta
		}
	}
	if best == nil {
		return "", false
	}
	if !auth.Empty() {
		best.auth = auth
		best.hasAuth = true
	}
	best.usage = usage
	best.hasUsage = usageFound(usage)
	if strings.TrimSpace(best.ledgerID) == "" {
		return "", true
	}
	ledgerID = best.ledgerID
	delete(s.authPending, best.reservationID)
	return ledgerID, true
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

func usageFound(usage money.TokenUsage) bool {
	return usage.Input > 0 || usage.Output > 0 || usage.Reasoning > 0 || usage.Cached > 0 || usage.CacheRead > 0 || usage.CacheCreation > 0
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
		// Retain the correlation until host usage arrives; it can be emitted
		// after settlement and must still replace fallback token estimates.
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
}

func (s *Service) ensureAuthPendingLocked() {
	if s.authPending == nil {
		s.authPending = make(map[string]*pendingAuthCapture)
	}
}

func (s *Service) pruneAuthPendingLocked(now time.Time) {
	for id, pending := range s.authPending {
		if pending == nil || now.Sub(pending.startedAt) > authPendingTTL {
			delete(s.authPending, id)
		}
	}
}
