package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/store"
)

const authQuotaCacheTTL = 15 * time.Minute
const authQuotaRequestTimeout = 15 * time.Second

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

type AuthQuotaOverviewItem struct {
	AuthID        string            `json:"auth_id"`
	AuthIndex     string            `json:"auth_index"`
	Provider      string            `json:"provider"`
	DisplayName   string            `json:"display_name"`
	Status        string            `json:"status"`
	LastAttemptAt *time.Time        `json:"last_attempt_at,omitempty"`
	LastSuccessAt *time.Time        `json:"last_success_at,omitempty"`
	LastErrorAt   *time.Time        `json:"last_error_at,omitempty"`
	Error         string            `json:"error,omitempty"`
	Plan          string            `json:"plan,omitempty"`
	ResetCredits  *float64          `json:"reset_credits,omitempty"`
	Windows       []AuthQuotaWindow `json:"windows"`
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
func (s *Service) AuthQuotaOverview(ctx context.Context, callback string) ([]AuthQuotaOverviewItem, error) {
	// The host callback bridge is process-global. Serializing refreshes also
	// coalesces concurrent console loads into one cache population.
	s.authQuotaRefreshMu.Lock()
	defer s.authQuotaRefreshMu.Unlock()
	source := s.authQuotaSourceValue()
	if source == nil {
		return nil, fmt.Errorf("auth quota source unavailable")
	}
	files, err := source.ListAuthQuotaFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth quota files unavailable: %w", err)
	}
	out := make([]AuthQuotaOverviewItem, 0, len(files))
	for _, file := range files {
		provider := quotaProvider(file.Provider)
		id := first(file.ID, file.Name, file.AuthIndex)
		if provider == "" || id == "" {
			continue
		}
		item := quotaItem(file, provider, id)
		saved, savedErr := s.store.GetAuthQuotaSnapshot(ctx, provider, id)
		if savedErr == nil && quotaFresh(saved, file.ModTime) {
			out = append(out, s.fromSnapshot(ctx, item, saved, "fresh"))
			continue
		}
		if savedErr == nil && quotaAttemptThrottled(saved, file.ModTime) {
			out = append(out, s.fromSnapshot(ctx, item, saved, "stale"))
			continue
		}
		raw, err := source.GetAuthQuotaJSON(ctx, file.AuthIndex)
		if err != nil {
			out = append(out, s.failedQuotaItem(ctx, item, saved, savedErr, provider, id, file.ModTime, fmt.Errorf("read auth configuration failed")))
			continue
		}
		creds, oauth, err := readCredentials(raw)
		if !oauth {
			continue
		}
		if err != nil {
			out = append(out, s.failedQuotaItem(ctx, item, saved, savedErr, provider, id, file.ModTime, err))
			continue
		}
		requestCtx, cancel := context.WithTimeout(ctx, authQuotaRequestTimeout)
		snap, fetchErr := s.fetch(requestCtx, source, callback, provider, creds)
		cancel()
		if fetchErr == nil {
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
			out = append(out, s.forecast(ctx, item))
			continue
		}
		out = append(out, s.failedQuotaItem(ctx, item, saved, savedErr, provider, id, file.ModTime, fetchErr))
	}
	return out, nil
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
	return x.LastError == "" && x.LastSuccessAt != nil && time.Since(*x.LastSuccessAt) < authQuotaCacheTTL && x.SnapshotJSON != "" && sameQuotaAuthVersion(x, t) && snapshotWindowsCurrent(x.SnapshotJSON)
}

func quotaAttemptThrottled(x store.AuthQuotaSnapshot, t time.Time) bool {
	if x.LastAttemptAt.IsZero() || time.Since(x.LastAttemptAt) >= authQuotaCacheTTL || !sameQuotaAuthVersion(x, t) {
		return false
	}
	// A successful snapshot whose windows have all expired must refresh even
	// inside the attempt TTL; otherwise the console stays stuck on a dead cycle.
	if x.LastError == "" && hasQuotaSnapshot(x) && !snapshotWindowsCurrent(x.SnapshotJSON) {
		return false
	}
	return true
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
		if w.ResetsAt == nil || !w.ResetsAt.After(now) {
			w.LocalAttributionStatus = "unavailable"
			continue
		}
		if w.Scope != "account" && !(item.Provider == "antigravity" && w.Scope == "model_pool") {
			w.LocalAttributionStatus = "unsupported"
			continue
		}
		from := cycleStart(item.Provider, w)
		if from.IsZero() || !now.After(from) {
			w.LocalAttributionStatus = "unavailable"
			continue
		}
		w.CycleStartAt = timePtr(from)
		if item.Provider == "xai" && w.CycleStartSource == "" {
			w.CycleStartSource = "inferred_month_start"
		}
		filter := store.AuthQuotaUsageFilter{Provider: item.Provider, AuthID: item.AuthID, AuthIndex: item.AuthIndex, From: from, To: now}
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
