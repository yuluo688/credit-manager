package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/store"
)

const (
	authQuotaRequestTimeout     = 15 * time.Second
	authQuotaWeeklyHistoryLimit = 8
	AuthQuotaDefaultPageSize    = 12
	AuthQuotaMaxPageSize        = 24
	authQuotaAPIKeySentinel     = "hidden:api_key"
)

var errAuthQuotaAPIKey = errors.New(authQuotaAPIKeySentinel)

type AuthQuotaFile struct {
	ID        string    `json:"id"`
	AuthIndex string    `json:"auth_index"`
	Name      string    `json:"name"`
	Label     string    `json:"label"`
	Provider  string    `json:"provider"`
	Type      string    `json:"type"`
	Email     string    `json:"email"`
	Account   string    `json:"account"`
	Path      string    `json:"path"`
	ModTime   time.Time `json:"mod_time"`
}
type AuthQuotaHTTPRequest struct {
	Method string      `json:"method"`
	URL    string      `json:"url"`
	Header http.Header `json:"header,omitempty"`
	Body   []byte      `json:"body,omitempty"`
}
type AuthQuotaHTTPResponse struct {
	StatusCode int         `json:"status_code"`
	Header     http.Header `json:"header,omitempty"`
	Body       []byte      `json:"body"`
}
type AuthQuotaSource interface {
	ListAuthQuotaFiles(context.Context) ([]AuthQuotaFile, error)
	GetAuthQuotaJSON(context.Context, string) ([]byte, error)
	DoAuthQuotaHTTP(context.Context, string, AuthQuotaHTTPRequest) (AuthQuotaHTTPResponse, error)
}

type AuthQuotaFilter struct {
	Page     int
	PageSize int
	Provider string
	Q        string
}

type AuthQuotaOverview struct {
	Items      []AuthQuotaOverviewItem `json:"items"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	Total      int                     `json:"total"`
	TotalPages int                     `json:"total_pages"`
	Providers  []string                `json:"providers"`
}

type AuthQuotaOverviewItem struct {
	AuthID                string            `json:"auth_id"`
	AuthIndex             string            `json:"auth_index"`
	Provider              string            `json:"provider"`
	DisplayName           string            `json:"display_name"`
	Status                string            `json:"status"`
	LastAttemptAt         *time.Time        `json:"last_attempt_at,omitempty"`
	LastSuccessAt         *time.Time        `json:"last_success_at,omitempty"`
	LastErrorAt           *time.Time        `json:"last_error_at,omitempty"`
	Error                 string            `json:"error,omitempty"`
	Plan                  string            `json:"plan,omitempty"`
	ResetCredits          *float64          `json:"reset_credits,omitempty"`
	MaxConcurrentRequests int64             `json:"max_concurrent_requests"`
	ActiveRequests        int64             `json:"active_requests"`
	Windows               []AuthQuotaWindow `json:"windows"`
}
type AuthQuotaLocalUsage struct {
	RequestCount          int64 `json:"request_count"`
	InputTokens           int64 `json:"input_tokens"`
	OutputTokens          int64 `json:"output_tokens"`
	ReasoningTokens       int64 `json:"reasoning_tokens"`
	CachedTokens          int64 `json:"cached_tokens"`
	CacheReadTokens       int64 `json:"cache_read_tokens"`
	CacheCreationTokens   int64 `json:"cache_creation_tokens"`
	TotalTokens           int64 `json:"total_tokens"`
	EstimatedCostMicroUSD int64 `json:"estimated_cost_micro_usd"`
}
type AuthQuotaWindow struct {
	ID                         string               `json:"id"`
	Label                      string               `json:"label"`
	Scope                      string               `json:"scope"`
	ScopeID                    string               `json:"scope_id,omitempty"`
	Mode                       string               `json:"mode"`
	Unit                       string               `json:"unit"`
	Currency                   string               `json:"currency,omitempty"`
	Used                       *float64             `json:"used,omitempty"`
	Remaining                  *float64             `json:"remaining,omitempty"`
	Limit                      *float64             `json:"limit,omitempty"`
	UsedRatio                  *float64             `json:"used_ratio,omitempty"`
	RemainingRatio             *float64             `json:"remaining_ratio,omitempty"`
	ResetsAt                   *time.Time           `json:"resets_at,omitempty"`
	DurationSeconds            *int64               `json:"duration_seconds,omitempty"`
	CycleStartAt               *time.Time           `json:"cycle_start_at,omitempty"`
	CycleStartSource           string               `json:"cycle_start_source,omitempty"`
	LocalAttributionStatus     string               `json:"local_attribution_status"`
	LocalUsage                 *AuthQuotaLocalUsage `json:"local_usage,omitempty"`
	AverageTokensPerRequest    *float64             `json:"average_tokens_per_request,omitempty"`
	EstimatedRemainingRequests *int64               `json:"estimated_remaining_requests,omitempty"`
	PredictionAvailable        bool                 `json:"prediction_available"`
	BaselineUsed               *float64             `json:"baseline_used,omitempty"`
	ObservedUsed               *float64             `json:"observed_used,omitempty"`
}
type quotaSnapshot struct {
	Plan         string            `json:"plan,omitempty"`
	ResetCredits *float64          `json:"reset_credits,omitempty"`
	Windows      []AuthQuotaWindow `json:"windows"`
}
type quotaCredentials struct{ token, accountID, projectID, userID string }

func (s *Service) authQuotaSourceValue() AuthQuotaSource {
	s.authQuotaMu.RLock()
	defer s.authQuotaMu.RUnlock()
	return s.authQuotaSource
}
func (s *Service) AuthQuotaOverview(ctx context.Context, callback string, filter AuthQuotaFilter) (AuthQuotaOverview, error) {
	// Listing is cache-only so opening the console does not query every
	// upstream account. Do not wait on authQuotaRefreshMu: a 15s card fetch
	// must not block the overview.
	source, files, err := s.authQuotaFiles(ctx)
	if err != nil {
		return AuthQuotaOverview{}, err
	}
	filter = normalizeAuthQuotaFilter(filter)
	providers := authQuotaProviderList(files)
	matched := make([]AuthQuotaFile, 0, len(files))
	for _, file := range files {
		provider := quotaProvider(file.Provider)
		if provider == "" {
			continue
		}
		if !authQuotaFileMatchesProvider(file, filter.Provider) {
			continue
		}
		if !authQuotaFileMatchesQuery(file, provider, filter.Q) {
			continue
		}
		id := first(file.ID, file.Name, file.AuthIndex)
		if id == "" {
			continue
		}
		if saved, savedErr := s.store.GetAuthQuotaSnapshot(ctx, provider, id); savedErr == nil && isAuthQuotaAPIKeySentinel(saved) {
			continue
		}
		matched = append(matched, file)
	}
	total := len(matched)
	page, totalPages, start, end := authQuotaPageBounds(filter.Page, filter.PageSize, total)
	pageFiles := matched[start:end]
	items := make([]AuthQuotaOverviewItem, 0, len(pageFiles))
	hidden := 0
	for _, file := range pageFiles {
		item, ok := s.loadAuthQuotaItem(ctx, source, callback, file, false)
		if !ok {
			hidden++
			continue
		}
		items = append(items, item)
	}
	total -= hidden
	if total < 0 {
		total = 0
	}
	_, totalPages, _, _ = authQuotaPageBounds(page, filter.PageSize, total)
	s.attachAuthConcurrency(ctx, items)
	return AuthQuotaOverview{
		Items:      items,
		Page:       page,
		PageSize:   filter.PageSize,
		Total:      total,
		TotalPages: totalPages,
		Providers:  providers,
	}, nil
}

func (s *Service) RefreshAuthQuota(ctx context.Context, callback, provider, authID, authIndex string) (AuthQuotaOverviewItem, error) {
	s.authQuotaRefreshMu.Lock()
	defer s.authQuotaRefreshMu.Unlock()
	source, files, err := s.authQuotaFiles(ctx)
	if err != nil {
		return AuthQuotaOverviewItem{}, err
	}
	for _, file := range files {
		if !matchAuthQuotaFile(file, provider, authID, authIndex) {
			continue
		}
		item, ok := s.loadAuthQuotaItem(ctx, source, callback, file, true)
		if !ok {
			continue
		}
		return s.withAuthConcurrency(ctx, item), nil
	}
	return AuthQuotaOverviewItem{}, fmt.Errorf("auth quota not found")
}

func (s *Service) authQuotaFiles(ctx context.Context) (AuthQuotaSource, []AuthQuotaFile, error) {
	source := s.authQuotaSourceValue()
	if source == nil {
		return nil, nil, fmt.Errorf("auth quota source unavailable")
	}
	files, err := source.ListAuthQuotaFiles(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("auth quota files unavailable: %w", err)
	}
	return source, files, nil
}

func normalizeAuthQuotaFilter(filter AuthQuotaFilter) AuthQuotaFilter {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = AuthQuotaDefaultPageSize
	}
	if filter.PageSize > AuthQuotaMaxPageSize {
		filter.PageSize = AuthQuotaMaxPageSize
	}
	filter.Provider = strings.TrimSpace(filter.Provider)
	filter.Q = strings.TrimSpace(filter.Q)
	return filter
}

func authQuotaPageBounds(page, pageSize, total int) (clampedPage, totalPages, start, end int) {
	if pageSize < 1 {
		pageSize = AuthQuotaDefaultPageSize
	}
	totalPages = 1
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start = (page - 1) * pageSize
	if start > total {
		start = total
	}
	end = start + pageSize
	if end > total {
		end = total
	}
	return page, totalPages, start, end
}

func authQuotaProviderList(files []AuthQuotaFile) []string {
	seen := make(map[string]struct{}, len(files))
	out := make([]string, 0)
	for _, file := range files {
		provider := quotaProvider(file.Provider)
		if provider == "" {
			continue
		}
		if _, ok := seen[provider]; ok {
			continue
		}
		seen[provider] = struct{}{}
		out = append(out, provider)
	}
	sort.Strings(out)
	return out
}

func authQuotaFileMatchesProvider(file AuthQuotaFile, provider string) bool {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return true
	}
	want := quotaProvider(provider)
	if want == "" {
		want = strings.ToLower(provider)
	}
	return quotaProvider(file.Provider) == want
}

func authQuotaFileMatchesQuery(file AuthQuotaFile, provider, q string) bool {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return true
	}
	id := first(file.ID, file.Name, file.AuthIndex)
	display := first(file.Label, file.Email, file.Account, file.Name, id)
	haystack := strings.ToLower(strings.Join([]string{
		file.Label, file.Email, file.Account, file.Name, file.ID, file.AuthIndex, id, display, provider,
	}, " "))
	return strings.Contains(haystack, q)
}

func isAuthQuotaAPIKeySentinel(saved store.AuthQuotaSnapshot) bool {
	return strings.TrimSpace(saved.LastError) == authQuotaAPIKeySentinel
}

func matchAuthQuotaFile(file AuthQuotaFile, provider, authID, authIndex string) bool {
	provider, authID, authIndex = strings.TrimSpace(provider), strings.TrimSpace(authID), strings.TrimSpace(authIndex)
	if authID == "" && authIndex == "" {
		return false
	}
	fileProvider := quotaProvider(file.Provider)
	fileID := first(file.ID, file.Name, file.AuthIndex)
	if provider != "" {
		want := quotaProvider(provider)
		if want == "" {
			want = strings.ToLower(provider)
		}
		if fileProvider != want {
			return false
		}
	}
	if authIndex != "" && file.AuthIndex != authIndex {
		return false
	}
	if authID != "" && fileID != authID && file.ID != authID && file.AuthIndex != authID {
		return false
	}
	return true
}

func (s *Service) loadAuthQuotaItem(ctx context.Context, source AuthQuotaSource, callback string, file AuthQuotaFile, fetch bool) (AuthQuotaOverviewItem, bool) {
	provider := quotaProvider(file.Provider)
	id := first(file.ID, file.Name, file.AuthIndex)
	if provider == "" || id == "" {
		return AuthQuotaOverviewItem{}, false
	}
	item := quotaItem(file, provider, id)
	saved, savedErr := s.store.GetAuthQuotaSnapshot(ctx, provider, id)
	if savedErr == nil && isAuthQuotaAPIKeySentinel(saved) && !fetch {
		return AuthQuotaOverviewItem{}, false
	}
	if !fetch && savedErr == nil {
		if quotaFresh(saved, file.ModTime) {
			return s.fromSnapshot(ctx, item, saved, "fresh"), true
		}
		return s.fromSnapshot(ctx, item, saved, "stale"), true
	}
	raw, err := source.GetAuthQuotaJSON(ctx, file.AuthIndex)
	if err != nil {
		if !fetch {
			item.Status = "unavailable"
			item.Error = "read auth configuration failed"
			return item, true
		}
		return s.failedQuotaItem(ctx, item, saved, savedErr, provider, id, file.ModTime, fmt.Errorf("read auth configuration failed")), true
	}
	creds, oauth, err := readCredentials(raw)
	if !oauth {
		_ = s.store.RecordAuthQuotaFailure(ctx, provider, id, modTime(file.ModTime), errAuthQuotaAPIKey)
		return AuthQuotaOverviewItem{}, false
	}
	if err != nil {
		if !fetch {
			item.Status = "unavailable"
			item.Error = err.Error()
			return item, true
		}
		return s.failedQuotaItem(ctx, item, saved, savedErr, provider, id, file.ModTime, err), true
	}
	if !fetch {
		item.Status = "idle"
		if savedErr == nil {
			item.LastSuccessAt = saved.LastSuccessAt
			if !saved.LastAttemptAt.IsZero() {
				attempt := saved.LastAttemptAt
				item.LastAttemptAt = &attempt
			}
			if saved.LastError != "" {
				item.Status = "unavailable"
				item.Error = saved.LastError
				item.LastErrorAt = saved.LastErrorAt
			}
		}
		return item, true
	}
	requestCtx, cancel := context.WithTimeout(ctx, authQuotaRequestTimeout)
	snap, fetchErr := s.fetch(requestCtx, source, callback, provider, creds)
	cancel()
	if fetchErr == nil {
		if savedErr == nil && hasQuotaSnapshot(saved) {
			var prev quotaSnapshot
			if json.Unmarshal([]byte(saved.SnapshotJSON), &prev) == nil {
				snap = mergeHistoricalQuotaWindows(prev, snap)
			}
		}
		if snap.Plan == "" {
			snap.Plan = quotaPlanFromAuthJSON(raw)
		}
		snap.Plan = normalizeQuotaPlan(snap.Plan)
		data, e := json.Marshal(snap)
		if e != nil {
			fetchErr = e
		} else {
			fetchErr = s.store.UpsertAuthQuotaSuccess(ctx, provider, id, string(data), modTime(file.ModTime))
		}
	}
	if fetchErr == nil {
		now := time.Now().UTC()
		item.Status = "fresh"
		item.Plan = snap.Plan
		item.ResetCredits = snap.ResetCredits
		item.Windows = snap.Windows
		item.LastAttemptAt = &now
		item.LastSuccessAt = &now
		return s.forecast(ctx, item), true
	}
	return s.failedQuotaItem(ctx, item, saved, savedErr, provider, id, file.ModTime, fetchErr), true
}

func (s *Service) failedQuotaItem(ctx context.Context, item AuthQuotaOverviewItem, saved store.AuthQuotaSnapshot, savedErr error, provider, id string, fileModTime time.Time, fail error) AuthQuotaOverviewItem {
	_ = s.store.RecordAuthQuotaFailure(ctx, provider, id, modTime(fileModTime), fail)
	if refreshed, err := s.store.GetAuthQuotaSnapshot(ctx, provider, id); err == nil {
		saved, savedErr = refreshed, nil
	}
	if savedErr == nil && hasQuotaSnapshot(saved) {
		return s.fromSnapshot(ctx, item, saved, "stale")
	}
	item.Status = "unavailable"
	if fail != nil {
		item.Error = fail.Error()
	}
	if savedErr == nil {
		item.LastSuccessAt = saved.LastSuccessAt
		if !saved.LastAttemptAt.IsZero() {
			attempt := saved.LastAttemptAt
			item.LastAttemptAt = &attempt
		}
		item.LastErrorAt = saved.LastErrorAt
	}
	return item
}
func quotaItem(f AuthQuotaFile, p, id string) AuthQuotaOverviewItem {
	return AuthQuotaOverviewItem{AuthID: id, AuthIndex: f.AuthIndex, Provider: p, DisplayName: first(f.Label, f.Email, f.Account, f.Name, id), Windows: []AuthQuotaWindow{}}
}
func modTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	v := t.UTC()
	return &v
}
func quotaFresh(x store.AuthQuotaSnapshot, t time.Time) bool {
	return x.LastError == "" && x.LastSuccessAt != nil && hasQuotaSnapshot(x) && sameQuotaAuthVersion(x, t) && snapshotWindowsCurrent(x.SnapshotJSON)
}

func sameQuotaAuthVersion(x store.AuthQuotaSnapshot, t time.Time) bool {
	// Auth modification times are persisted as Unix milliseconds; compare at the
	// same precision so host nanoseconds do not bypass the refresh gate.
	return (t.IsZero() && x.AuthModTime == nil) || (!t.IsZero() && x.AuthModTime != nil && x.AuthModTime.UnixMilli() == t.UTC().UnixMilli())
}

func snapshotWindowsCurrent(raw string) bool {
	var snapshot quotaSnapshot
	if json.Unmarshal([]byte(raw), &snapshot) != nil || len(snapshot.Windows) == 0 {
		return false
	}
	now := time.Now().UTC()
	for _, window := range snapshot.Windows {
		if window.ResetsAt == nil || window.ResetsAt.After(now) {
			return true
		}
	}
	return false
}
func (s *Service) fromSnapshot(ctx context.Context, item AuthQuotaOverviewItem, x store.AuthQuotaSnapshot, status string) AuthQuotaOverviewItem {
	var snap quotaSnapshot
	if !hasQuotaSnapshot(x) || json.Unmarshal([]byte(x.SnapshotJSON), &snap) != nil {
		item.Status = "unavailable"
		if x.LastError != "" {
			item.Error = x.LastError
		} else {
			item.Error = "stored quota snapshot is invalid"
		}
		item.LastSuccessAt = x.LastSuccessAt
		if !x.LastAttemptAt.IsZero() {
			v := x.LastAttemptAt
			item.LastAttemptAt = &v
		}
		item.LastErrorAt = x.LastErrorAt
		return item
	}
	item.Status = status
	item.Plan = snap.Plan
	item.ResetCredits = snap.ResetCredits
	item.Windows = snap.Windows
	item.LastSuccessAt = x.LastSuccessAt
	if !x.LastAttemptAt.IsZero() {
		v := x.LastAttemptAt
		item.LastAttemptAt = &v
	}
	if x.LastError != "" {
		item.Error = x.LastError
		item.LastErrorAt = x.LastErrorAt
	}
	return s.forecast(ctx, item)
}

func hasQuotaSnapshot(x store.AuthQuotaSnapshot) bool {
	raw := strings.TrimSpace(x.SnapshotJSON)
	if raw == "" || raw == "{}" {
		return false
	}
	var snap quotaSnapshot
	return json.Unmarshal([]byte(raw), &snap) == nil && len(snap.Windows) > 0
}

func (s *Service) forecast(ctx context.Context, item AuthQuotaOverviewItem) AuthQuotaOverviewItem {
	now := time.Now().UTC()
	for i := range item.Windows {
		w := &item.Windows[i]
		if w.ResetsAt == nil {
			w.LocalAttributionStatus = "unavailable"
			continue
		}
		to := now
		closed := !w.ResetsAt.After(now)
		if closed {
			to = w.ResetsAt.UTC()
		}
		if w.Scope != "account" && !(item.Provider == "antigravity" && w.Scope == "model_pool") {
			w.LocalAttributionStatus = "unsupported"
			continue
		}
		from := cycleStart(item.Provider, w)
		if from.IsZero() || !to.After(from) {
			w.LocalAttributionStatus = "unavailable"
			continue
		}
		w.CycleStartAt = timePtr(from)
		if item.Provider == "xai" && w.CycleStartSource == "" {
			w.CycleStartSource = "inferred_month_start"
		}
		filter := store.AuthQuotaUsageFilter{Provider: item.Provider, AuthID: item.AuthID, AuthIndex: item.AuthIndex, From: from, To: to}
		usage, complete, e := s.windowUsage(ctx, filter, item.Provider, w.ScopeID)
		if e != nil {
			w.LocalAttributionStatus = "unavailable"
			continue
		}
		if !complete {
			w.LocalAttributionStatus = "unmatched_local_usage"
			continue
		}
		tokens := totalAuthQuotaTokens(usage)
		w.LocalAttributionStatus = "complete"
		w.LocalUsage = authQuotaLocalUsage(usage)
		if closed {
			w.PredictionAvailable = false
			w.EstimatedRemainingRequests = nil
			continue
		}
		observed, baseline, ok := s.observedWindowUsed(ctx, item, w, usage.RequestCount)
		if ok {
			w.ObservedUsed = &observed
			w.BaselineUsed = &baseline
		}
		if usage.RequestCount > 0 {
			a := float64(tokens) / float64(usage.RequestCount)
			w.AverageTokensPerRequest = &a
			if ok {
				if remainingRequests, ready := estimateRemainingRequests(usage.RequestCount, w, observed); ready {
					w.EstimatedRemainingRequests = &remainingRequests
					w.PredictionAvailable = true
				}
			}
		}
	}
	return item
}

func mergeHistoricalQuotaWindows(prev, next quotaSnapshot) quotaSnapshot {
	current := make(map[string]struct{}, len(next.Windows))
	for i := range next.Windows {
		current[windowHistoryKey(&next.Windows[i])] = struct{}{}
	}
	type hist struct {
		window AuthQuotaWindow
		reset  int64
		cycle  string
	}
	extra := make([]hist, 0, len(prev.Windows))
	for _, window := range prev.Windows {
		if !quotaWindowIsWeekly(window) {
			continue
		}
		key := windowHistoryKey(&window)
		if _, ok := current[key]; ok {
			continue
		}
		cycle := windowCycleKey(&window)
		if cycle == "" {
			continue
		}
		reset := int64(0)
		if window.ResetsAt != nil {
			reset = window.ResetsAt.UTC().UnixMilli()
		}
		cleared := window
		cleared.PredictionAvailable = false
		cleared.EstimatedRemainingRequests = nil
		cleared.AverageTokensPerRequest = nil
		cleared.LocalUsage = nil
		cleared.LocalAttributionStatus = ""
		cleared.ObservedUsed = nil
		cleared.BaselineUsed = nil
		extra = append(extra, hist{cleared, reset, cycle})
	}
	sort.Slice(extra, func(i, j int) bool { return extra[i].reset > extra[j].reset })
	keptCycles := make(map[string]struct{}, authQuotaWeeklyHistoryLimit)
	for _, item := range extra {
		if _, ok := keptCycles[item.cycle]; !ok {
			if len(keptCycles) >= authQuotaWeeklyHistoryLimit {
				continue
			}
			keptCycles[item.cycle] = struct{}{}
		}
		next.Windows = append(next.Windows, item.window)
	}
	return next
}
