package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/yuluo688/credit-manager/internal/store"
)

// AuthPickCandidate is one host auth offered to the plugin scheduler.
type AuthPickCandidate struct {
	ID       string
	Provider string
}

func (s *Service) SetAuthConcurrencyLimit(ctx context.Context, provider, authID string, maxConcurrent int64) (AuthQuotaOverviewItem, error) {
	provider, authID = authLimitProvider(provider), strings.TrimSpace(authID)
	if provider == "" || authID == "" {
		return AuthQuotaOverviewItem{}, fmt.Errorf("%w: provider and auth id are required", store.ErrInvalidArgument)
	}
	if maxConcurrent < 0 {
		return AuthQuotaOverviewItem{}, fmt.Errorf("%w: max concurrent requests must not be negative", store.ErrInvalidArgument)
	}
	if err := s.store.UpsertAuthConcurrencyLimit(ctx, provider, authID, maxConcurrent); err != nil {
		return AuthQuotaOverviewItem{}, err
	}
	source, files, err := s.authQuotaFiles(ctx)
	if err != nil {
		return s.withAuthConcurrency(ctx, AuthQuotaOverviewItem{AuthID: authID, Provider: provider, MaxConcurrentRequests: maxConcurrent}), nil
	}
	for _, file := range files {
		if !matchAuthQuotaFile(file, provider, authID, "") {
			continue
		}
		item, ok := s.loadAuthQuotaItem(ctx, source, "", file, false)
		if !ok {
			continue
		}
		return s.withAuthConcurrency(ctx, item), nil
	}
	item := AuthQuotaOverviewItem{AuthID: authID, Provider: provider, MaxConcurrentRequests: maxConcurrent}
	return s.withAuthConcurrency(ctx, item), nil
}

// AuthConcurrencyTarget identifies one auth row for a batch concurrency update.
type AuthConcurrencyTarget struct {
	Provider string `json:"provider"`
	AuthID   string `json:"auth_id"`
}

// AuthConcurrencyBatchResult reports how many auth rows a batch update changed.
type AuthConcurrencyBatchResult struct {
	Updated               int   `json:"updated"`
	MaxConcurrentRequests int64 `json:"max_concurrent_requests"`
}

func (s *Service) SetAuthConcurrencyLimits(ctx context.Context, filter AuthQuotaFilter, targets []AuthConcurrencyTarget, maxConcurrent int64) (AuthConcurrencyBatchResult, error) {
	if maxConcurrent < 0 {
		return AuthConcurrencyBatchResult{}, fmt.Errorf("%w: max concurrent requests must not be negative", store.ErrInvalidArgument)
	}
	limits, err := s.authConcurrencyBatchLimits(ctx, filter, targets, maxConcurrent)
	if err != nil {
		return AuthConcurrencyBatchResult{}, err
	}
	if err := s.store.UpsertAuthConcurrencyLimits(ctx, limits); err != nil {
		return AuthConcurrencyBatchResult{}, err
	}
	return AuthConcurrencyBatchResult{Updated: len(limits), MaxConcurrentRequests: maxConcurrent}, nil
}

func (s *Service) authConcurrencyBatchLimits(ctx context.Context, filter AuthQuotaFilter, targets []AuthConcurrencyTarget, maxConcurrent int64) ([]store.AuthConcurrencyLimit, error) {
	_, files, err := s.authQuotaFiles(ctx)
	if err != nil {
		return nil, err
	}
	filter = normalizeAuthQuotaFilter(filter)
	want := map[string]struct{}{}
	for _, target := range targets {
		provider, authID := authLimitProvider(target.Provider), strings.TrimSpace(target.AuthID)
		if provider == "" || authID == "" {
			continue
		}
		want[provider+"\x00"+authID] = struct{}{}
	}
	restrict := len(want) > 0
	out := make([]store.AuthConcurrencyLimit, 0, len(files))
	seen := map[string]struct{}{}
	for _, file := range files {
		provider := quotaProvider(file.Provider)
		authID := first(file.ID, file.Name, file.AuthIndex)
		if provider == "" || authID == "" {
			continue
		}
		if !authQuotaFileMatchesProvider(file, filter.Provider) || !authQuotaFileMatchesQuery(file, provider, filter.Q) {
			continue
		}
		if saved, savedErr := s.store.GetAuthQuotaSnapshot(ctx, provider, authID); savedErr == nil && isAuthQuotaAPIKeySentinel(saved) {
			continue
		}
		key := provider + "\x00" + authID
		if restrict {
			if _, ok := want[key]; !ok {
				continue
			}
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, store.AuthConcurrencyLimit{Provider: provider, AuthID: authID, MaxConcurrentRequests: maxConcurrent})
	}
	return out, nil
}

func (s *Service) AdmitAuth(ctx context.Context, reservationID string, auth store.AuthIdentity) error {
	if s == nil || strings.TrimSpace(reservationID) == "" || auth.Empty() {
		return nil
	}
	provider, authID := authLimitIdentity(auth)
	if provider == "" || authID == "" {
		s.bindAuthCapture(reservationID, auth)
		return nil
	}
	limit, err := s.store.GetAuthConcurrencyLimit(ctx, provider, authID)
	if err != nil {
		return err
	}
	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.ensureAuthPendingLocked()
	if limit > 0 && s.activeAuthRequestsLocked(provider, authID, reservationID) >= limit {
		return store.ErrConcurrentLimit
	}
	s.bindAuthCaptureLocked(reservationID, auth)
	return nil
}

func (s *Service) PickAuth(ctx context.Context, candidates []AuthPickCandidate) (authID string, handled bool, err error) {
	if s == nil || len(candidates) == 0 {
		return "", false, nil
	}
	limits, err := s.store.ListAuthConcurrencyLimits(ctx)
	if err != nil {
		return "", false, err
	}
	anyLimit := false
	for _, candidate := range candidates {
		if authConcurrencyLimitOf(limits, candidate) > 0 {
			anyLimit = true
			break
		}
	}
	if !anyLimit {
		return "", false, nil
	}
	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.ensureAuthPendingLocked()
	available := make([]AuthPickCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		limit := authConcurrencyLimitOf(limits, candidate)
		provider, id := authLimitProvider(candidate.Provider), strings.TrimSpace(candidate.ID)
		if limit > 0 && provider != "" && id != "" && s.activeAuthRequestsLocked(provider, id, "") >= limit {
			continue
		}
		available = append(available, candidate)
	}
	if len(available) == 0 {
		return "", true, store.ErrConcurrentLimit
	}
	chosen := s.nextAuthPickLocked(available)
	s.bindOldestUnattributedLocked(store.AuthIdentity{AuthID: strings.TrimSpace(chosen.ID), Provider: chosen.Provider})
	return strings.TrimSpace(chosen.ID), true, nil
}

func (s *Service) withAuthConcurrency(ctx context.Context, item AuthQuotaOverviewItem) AuthQuotaOverviewItem {
	items := []AuthQuotaOverviewItem{item}
	s.attachAuthConcurrency(ctx, items)
	return items[0]
}

func (s *Service) attachAuthConcurrency(ctx context.Context, items []AuthQuotaOverviewItem) {
	if s == nil || len(items) == 0 {
		return
	}
	limits, err := s.store.ListAuthConcurrencyLimits(ctx)
	if err != nil {
		limits = map[string]int64{}
	}
	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.ensureAuthPendingLocked()
	for i := range items {
		provider, authID := authLimitProvider(items[i].Provider), strings.TrimSpace(items[i].AuthID)
		if provider != "" && authID != "" {
			items[i].MaxConcurrentRequests = limits[provider+"\x00"+authID]
		}
		items[i].ActiveRequests = s.activeAuthRequestsLocked(provider, authID, "")
	}
}

func (s *Service) bindAuthCapture(reservationID string, auth store.AuthIdentity) {
	if s == nil {
		return
	}
	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.bindAuthCaptureLocked(reservationID, auth)
}

func (s *Service) bindAuthCaptureLocked(reservationID string, auth store.AuthIdentity) {
	s.ensureAuthPendingLocked()
	reservationID = strings.TrimSpace(reservationID)
	if reservationID == "" || auth.Empty() {
		return
	}
	pending := s.authPending[reservationID]
	if pending == nil {
		return
	}
	pending.auth = auth
	pending.hasAuth = true
}

func (s *Service) bindOldestUnattributedLocked(auth store.AuthIdentity) {
	if auth.Empty() {
		return
	}
	var oldest *pendingAuthCapture
	for _, pending := range s.authPending {
		if pending == nil || pending.hasAuth {
			continue
		}
		if oldest == nil || pending.startedAt.Before(oldest.startedAt) {
			oldest = pending
		}
	}
	if oldest == nil {
		return
	}
	oldest.auth = auth
	oldest.hasAuth = true
}

func (s *Service) nextAuthPickLocked(available []AuthPickCandidate) AuthPickCandidate {
	if s.authPickCursor == nil {
		s.authPickCursor = map[string]int{}
	}
	key := authLimitProvider(available[0].Provider)
	if key == "" {
		key = "auth"
	}
	i := s.authPickCursor[key] % len(available)
	s.authPickCursor[key] = i + 1
	return available[i]
}

func (s *Service) activeAuthRequestsLocked(provider, authID, exceptReservation string) int64 {
	if authID == "" {
		return 0
	}
	provider = authLimitProvider(provider)
	exceptReservation = strings.TrimSpace(exceptReservation)
	var n int64
	for id, pending := range s.authPending {
		if pending == nil || !pending.hasAuth || id == exceptReservation {
			continue
		}
		pendingProvider, pendingID := authLimitIdentity(pending.auth)
		if pendingID == authID && (provider == "" || pendingProvider == provider) {
			n++
		}
	}
	return n
}

func authConcurrencyLimitOf(limits map[string]int64, candidate AuthPickCandidate) int64 {
	provider, authID := authLimitProvider(candidate.Provider), strings.TrimSpace(candidate.ID)
	if provider == "" || authID == "" || len(limits) == 0 {
		return 0
	}
	return limits[provider+"\x00"+authID]
}

func authLimitIdentity(auth store.AuthIdentity) (provider, authID string) {
	return authLimitProvider(auth.Provider), first(auth.AuthID, auth.AuthIndex)
}

func authLimitProvider(provider string) string {
	if normalized := quotaProvider(provider); normalized != "" {
		return normalized
	}
	return strings.ToLower(strings.TrimSpace(provider))
}
