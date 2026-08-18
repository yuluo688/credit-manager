package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
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

func readCredentials(raw []byte) (quotaCredentials, bool, error) {
	var v map[string]any
	if json.Unmarshal(raw, &v) != nil {
		return quotaCredentials{}, true, fmt.Errorf("auth configuration is invalid")
	}
	token, evidence, stale := oauthToken(v)
	if !evidence {
		return quotaCredentials{}, false, nil
	}
	if token == "" {
		if stale {
			return quotaCredentials{}, true, fmt.Errorf("OAuth access token is unavailable")
		}
		return quotaCredentials{}, true, fmt.Errorf("OAuth access token is unavailable")
	}
	return quotaCredentials{token: token, accountID: findText(v, "account_id", "accountId", "chatgpt_account_id"), projectID: findText(v, "project_id", "projectId", "project"), userID: findText(v, "user_id", "userId", "userid", "sub", "subject")}, true, nil
}
func oauthToken(v any) (string, bool, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", false, false
	}
	evidence, stale := false, false
	for k, x := range m {
		key := strings.ToLower(k)
		switch key {
		case "tokens", "token", "oauth", "claudeaioauth":
			evidence = true
			t, e, s := oauthToken(x)
			if t != "" {
				return t, true, s
			}
			evidence = evidence || e
			stale = stale || s
		case "access_token", "accesstoken":
			evidence = true
			if t, ok := x.(string); ok && strings.TrimSpace(t) != "" {
				return strings.TrimSpace(t), true, stale
			}
		case "refresh_token", "refreshtoken", "id_token", "idtoken":
			evidence = true
			stale = true
		case "type", "auth_type":
			if t, ok := x.(string); ok && strings.Contains(strings.ToLower(t), "oauth") {
				evidence = true
			}
		}
	}
	return "", evidence, stale
}
func findText(v any, keys ...string) string {
	wanted := map[string]bool{}
	for _, k := range keys {
		wanted[strings.ToLower(k)] = true
	}
	var walk func(any) string
	walk = func(x any) string {
		switch y := x.(type) {
		case map[string]any:
			for k, v := range y {
				if wanted[strings.ToLower(k)] {
					if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
						return strings.TrimSpace(s)
					}
				}
			}
			for _, v := range y {
				if r := walk(v); r != "" {
					return r
				}
			}
		case []any:
			for _, v := range y {
				if r := walk(v); r != "" {
					return r
				}
			}
		}
		return ""
	}
	return walk(v)
}

func (s *Service) fetch(ctx context.Context, source AuthQuotaSource, callback, p string, c quotaCredentials) (quotaSnapshot, error) {
	switch p {
	case "codex":
		return codex(ctx, source, callback, c)
	case "claude":
		return claude(ctx, source, callback, c)
	case "antigravity":
		return antigravity(ctx, source, callback, c)
	case "kimi":
		return kimi(ctx, source, callback, c)
	case "xai":
		return xai(ctx, source, callback, c)
	}
	return quotaSnapshot{}, fmt.Errorf("unsupported auth quota provider")
}
func headers(token string) http.Header {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+token)
	h.Set("Accept", "application/json")
	return h
}
func request(ctx context.Context, s AuthQuotaSource, callback, method, url string, h http.Header, b []byte) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r, e := s.DoAuthQuotaHTTP(ctx, callback, AuthQuotaHTTPRequest{Method: method, URL: url, Header: h, Body: b})
	if e != nil {
		return nil, e
	}
	if r.StatusCode < 200 || r.StatusCode > 299 {
		return nil, fmt.Errorf("quota endpoint returned status %d", r.StatusCode)
	}
	var out map[string]any
	if json.Unmarshal(r.Body, &out) != nil {
		return nil, fmt.Errorf("decode quota response")
	}
	return out, nil
}
func codex(ctx context.Context, s AuthQuotaSource, cb string, c quotaCredentials) (quotaSnapshot, error) {
	h := headers(c.token)
	h.Set("User-Agent", "credit-manager")
	if c.accountID != "" {
		h.Set("Chatgpt-Account-Id", c.accountID)
	}
	d, e := request(ctx, s, cb, "GET", "https://chatgpt.com/backend-api/wham/usage", h, nil)
	if e != nil {
		return quotaSnapshot{}, e
	}
	w := codexWindows(d)
	if len(w) == 0 {
		return quotaSnapshot{}, fmt.Errorf("codex quota response has no recognized windows")
	}
	snap := quotaSnapshot{Plan: findText(d, "plan_type", "planType", "plan"), Windows: w}
	if credits, e := request(ctx, s, cb, "GET", "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits", h, nil); e == nil {
		snap.ResetCredits = number(credits, "available_count", "availableCount")
	}
	return snap, nil
}
func claude(ctx context.Context, s AuthQuotaSource, cb string, c quotaCredentials) (quotaSnapshot, error) {
	h := headers(c.token)
	h.Set("Anthropic-Beta", "oauth-2025-04-20")
	h.Set("Anthropic-Version", "2023-06-01")
	d, e := request(ctx, s, cb, "GET", "https://api.anthropic.com/api/oauth/usage", h, nil)
	if e != nil {
		return quotaSnapshot{}, e
	}
	var w []AuthQuotaWindow
	for _, spec := range []struct {
		id, scope, scopeID string
		duration           *int64
	}{
		{"five_hour", "account", "", int64ptr(18000)},
		{"seven_day", "account", "", int64ptr(604800)},
		{"seven_day_oauth_apps", "account", "", int64ptr(604800)},
		{"seven_day_opus", "model", "opus", int64ptr(604800)},
		{"seven_day_sonnet", "model", "sonnet", int64ptr(604800)},
		{"seven_day_cowork", "account", "", int64ptr(604800)},
		{"iguana_necktie", "account", "", nil},
	} {
		if m, ok := object(d, spec.id); ok {
			if x, ok := utilWindow(spec.id, spec.id, spec.scope, spec.scopeID, spec.duration, m); ok {
				w = append(w, x)
			}
		}
	}
	if m, ok := object(d, "extra_usage"); ok {
		if x, ok := valueWindow("extra_usage", "Extra usage", "credits", number(m, "monthly_limit"), number(m, "used_credits"), number(m, "utilization"), timeField(m, "resets_at"), nil); ok {
			w = append(w, x)
		}
	}
	if len(w) == 0 {
		return quotaSnapshot{}, fmt.Errorf("claude quota response has no recognized windows")
	}
	return quotaSnapshot{Plan: findText(d, "plan", "plan_type"), Windows: w}, nil
}
func antigravity(ctx context.Context, s AuthQuotaSource, cb string, c quotaCredentials) (quotaSnapshot, error) {
	if c.projectID == "" {
		return quotaSnapshot{}, fmt.Errorf("OAuth project id is unavailable")
	}
	h := headers(c.token)
	h.Set("Content-Type", "application/json")
	h.Set("User-Agent", "credit-manager")
	body := []byte(`{"project":` + strconv.Quote(c.projectID) + `}`)
	var d map[string]any
	var e error
	for _, url := range []string{"https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary", "https://daily-cloudcode-pa.sandbox.googleapis.com/v1internal:retrieveUserQuotaSummary", "https://cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary"} {
		d, e = request(ctx, s, cb, "POST", url, h, body)
		if e == nil {
			break
		}
	}
	if e != nil {
		return quotaSnapshot{}, e
	}
	w := antiWindows(d)
	if !has(w, "gemini-weekly") || !has(w, "3p-weekly") {
		return quotaSnapshot{}, fmt.Errorf("antigravity quota response is missing required weekly pools")
	}
	return quotaSnapshot{Plan: findText(d, "plan", "planName"), Windows: w}, nil
}
func kimi(ctx context.Context, s AuthQuotaSource, cb string, c quotaCredentials) (quotaSnapshot, error) {
	d, e := request(ctx, s, cb, "GET", "https://api.kimi.com/coding/v1/usages", headers(c.token), nil)
	if e != nil {
		return quotaSnapshot{}, e
	}
	w := kimiWindows(d)
	if len(w) == 0 {
		return quotaSnapshot{}, fmt.Errorf("kimi quota response has no limits")
	}
	return quotaSnapshot{Plan: findText(d, "plan", "plan_name"), Windows: w}, nil
}
func xai(ctx context.Context, s AuthQuotaSource, cb string, c quotaCredentials) (quotaSnapshot, error) {
	h := headers(c.token)
	h.Set("Accept", "*/*")
	h.Set("X-XAI-Token-Auth", "xai-grok-cli")
	h.Set("X-Grok-Client-Version", "0.2.93")
	h.Set("User-Agent", "xai-grok-workspace/0.2.93")
	if c.userID != "" {
		h.Set("X-Userid", c.userID)
	}
	credits, creditsErr := request(ctx, s, cb, "GET", "https://cli-chat-proxy.grok.com/v1/billing?format=credits", h, nil)
	billing, billingErr := request(ctx, s, cb, "GET", "https://cli-chat-proxy.grok.com/v1/billing", h, nil)
	if creditsErr != nil && billingErr != nil {
		return quotaSnapshot{}, billingErr
	}
	w := append(xaiCreditWindows(xaiBillingMap(credits)), xaiWindows(xaiBillingMap(billing))...)
	if len(w) == 0 {
		return quotaSnapshot{}, fmt.Errorf("xAI billing response has no balances")
	}
	return quotaSnapshot{Plan: first(findText(credits, "plan", "planName"), findText(billing, "plan", "planName")), Windows: w}, nil
}

func codexWindows(d map[string]any) []AuthQuotaWindow {
	var out []AuthQuotaWindow
	add := func(family, label, scope, scopeID string, v any) {
		m, ok := v.(map[string]any)
		if !ok {
			return
		}
		for _, k := range []string{"primary_window", "secondary_window"} {
			if x, ok := object(m, k); ok {
				if w, ok := percentWindow(family+"-"+strings.TrimSuffix(k, "_window"), label+" "+strings.TrimSuffix(k, "_window"), scope, scopeID, x); ok {
					out = append(out, w)
				}
			}
		}
	}
	if m, ok := object(d, "rate_limit"); ok {
		add("rate-limit", "Rate limit", "account", "", m)
	}
	if m, ok := object(d, "code_review_rate_limit"); ok {
		add("code-review", "Code review", "model", "codex_code_review", m)
	}
	if xs, ok := d["additional_rate_limits"].([]any); ok {
		for i, x := range xs {
			m, _ := x.(map[string]any)
			feature := first(findText(m, "metered_feature"), fmt.Sprintf("additional-%d", i+1))
			name := first(findText(m, "limit_name"), feature)
			if limit, ok := object(m, "rate_limit"); ok {
				slug := codexSlug(feature)
				add("additional-"+slug, name, "model", "codex_"+slug, limit)
			}
		}
	}
	return out
}
func codexSlug(value string) string {
	var b strings.Builder
	underscore := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			underscore = false
		} else if b.Len() > 0 && !underscore {
			b.WriteByte('_')
			underscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}
func antiWindows(d map[string]any) []AuthQuotaWindow {
	var out []AuthQuotaWindow
	specs := []struct {
		id, scopeID string
		duration    int64
	}{
		{"gemini-5h", "gemini", 18000},
		{"3p-5h", "third-party", 18000},
		{"gemini-weekly", "gemini", 604800},
		{"3p-weekly", "third-party", 604800},
	}
	groups, _ := d["groups"].([]any)
	for _, spec := range specs {
		var window *AuthQuotaWindow
		for _, g := range groups {
			gm, _ := g.(map[string]any)
			bs, _ := gm["buckets"].([]any)
			for _, b := range bs {
				m, _ := b.(map[string]any)
				id := findText(m, "bucketId", "bucket_id")
				r := number(m, "remainingFraction", "remaining_fraction")
				if id != spec.id || r == nil || *r < 0 || *r > 1 || m["enabled"] == false || m["isEnabled"] == false {
					continue
				}
				limit := 100.0
				remaining := limit * *r
				used := limit - remaining
				usedRatio := used / limit
				duration := spec.duration
				window = &AuthQuotaWindow{ID: spec.id, Label: first(findText(gm, "displayName", "display_name"), spec.id), Scope: "model_pool", ScopeID: spec.scopeID, Mode: "rolling", Unit: "percentage", Limit: &limit, Used: &used, Remaining: &remaining, UsedRatio: &usedRatio, RemainingRatio: r, ResetsAt: timeField(m, "resetTime", "resetAt", "reset_time", "reset_at"), DurationSeconds: &duration}
				break
			}
			if window != nil {
				break
			}
		}
		if window != nil {
			out = append(out, *window)
		}
	}
	return out
}
func kimiWindows(d map[string]any) []AuthQuotaWindow {
	var out []AuthQuotaWindow
	limits, _ := d["limits"].([]any)
	usage, ok := object(d, "usage")
	if !ok {
		usage = d
	}
	if usage != nil {
		if w, ok := kimiWindow("summary", "Summary", usage, nil); ok {
			out = append(out, w)
		}
	}
	for i, x := range limits {
		m, _ := x.(map[string]any)
		detail := m
		if nested, ok := object(m, "detail"); ok {
			detail = nested
		}
		id := first(findText(m, "id", "name", "label"), fmt.Sprintf("limit-%d", i+1))
		window, _ := object(m, "window")
		if w, ok := kimiWindow(id, id, detail, window); ok {
			out = append(out, w)
		}
	}
	return out
}
func xaiBillingMap(d map[string]any) map[string]any {
	if d == nil {
		return nil
	}
	if m, ok := object(d, "config"); ok {
		return m
	}
	return d
}

func xaiPeriodTimes(d, period map[string]any) (start, end *time.Time, seconds *int64) {
	start = timeField(period, "start")
	if start == nil {
		start = timeField(d, "billingPeriodStart", "billing_period_start")
	}
	end = timeField(period, "end")
	if end == nil {
		end = timeField(d, "billingPeriodEnd", "billing_period_end")
	}
	if start != nil && end != nil && end.After(*start) {
		value := int64(end.Sub(*start) / time.Second)
		seconds = &value
	}
	return start, end, seconds
}

func xaiPercentWindow(id, label, scope, scopeID string, usedPercent *float64, start, end *time.Time, seconds *int64) (AuthQuotaWindow, bool) {
	limit := 100.0
	var used *float64
	if usedPercent != nil {
		value := *usedPercent
		if !finiteNonNegative(value) || value > 100 {
			return AuthQuotaWindow{}, false
		}
		used = &value
	}
	w, ok := valueWindow(id, label, "percentage", &limit, used, nil, end, seconds)
	if !ok && usedPercent == nil {
		w = AuthQuotaWindow{ID: id, Label: label, Scope: first(scope, "account"), ScopeID: scopeID, Mode: "rolling", Unit: "percentage", ResetsAt: end, DurationSeconds: seconds}
		ok = true
	}
	if !ok {
		return AuthQuotaWindow{}, false
	}
	w.Scope = first(scope, "account")
	w.ScopeID = scopeID
	w.Mode = "rolling"
	if start != nil {
		w.CycleStartAt = start
		w.CycleStartSource = "upstream_period"
	} else if w.ResetsAt != nil && w.DurationSeconds != nil {
		inferred := w.ResetsAt.Add(-time.Duration(*w.DurationSeconds) * time.Second)
		w.CycleStartAt = &inferred
		w.CycleStartSource = "inferred_week_start"
	}
	return w, true
}

func xaiCreditWindows(d map[string]any) []AuthQuotaWindow {
	if d == nil {
		return nil
	}
	period, _ := object(d, "currentPeriod")
	if period == nil {
		period, _ = object(d, "current_period")
	}
	periodType := strings.ToLower(first(findText(period, "type"), findText(d, "periodType", "period_type")))
	percent := number(d, "creditUsagePercent", "credit_usage_percent")
	products, _ := d["productUsage"].([]any)
	if products == nil {
		products, _ = d["product_usage"].([]any)
	}
	if percent == nil && !strings.Contains(periodType, "weekly") && len(products) == 0 {
		return nil
	}
	start, end, seconds := xaiPeriodTimes(d, period)
	if seconds == nil {
		seconds = int64ptr(604800)
	}
	var out []AuthQuotaWindow
	if w, ok := xaiPercentWindow("weekly", "周限额", "account", "", percent, start, end, seconds); ok {
		out = append(out, w)
	}
	for i, item := range products {
		m, _ := item.(map[string]any)
		name := first(findText(m, "product", "name", "label"), fmt.Sprintf("产品 %d", i+1))
		id := "weekly-" + first(codexSlug(name), fmt.Sprintf("%d", i+1))
		if w, ok := xaiPercentWindow(id, name, "product", name, number(m, "usagePercent", "usage_percent"), start, end, seconds); ok {
			out = append(out, w)
		}
	}
	return out
}

func xaiWindows(m map[string]any) []AuthQuotaWindow {
	var out []AuthQuotaWindow
	if m == nil {
		return out
	}
	end := timeField(m, "billingPeriodEnd", "billing_period_end")
	monthlyLimit := cent(firstNumber(m, "monthlyLimit", "monthly_limit"))
	monthlyUsed := cent(number(m, "used"))
	if w, ok := valueWindow("monthly", "Monthly", "currency", monthlyLimit, monthlyUsed, nil, end, nil); ok {
		w.Mode = "fixed"
		w.Currency = "USD"
		if end != nil {
			start := end.AddDate(0, -1, 0)
			w.CycleStartAt = &start
			w.CycleStartSource = "inferred_month_start"
		}
		out = append(out, w)
	}
	onDemandLimit := cent(firstNumber(m, "onDemandCap", "on_demand_cap"))
	onDemandUsed := cent(firstNumber(m, "onDemandUsed", "on_demand_used"))
	if onDemandUsed == nil && monthlyUsed != nil && monthlyLimit != nil {
		value := math.Max(0, *monthlyUsed-*monthlyLimit)
		onDemandUsed = &value
	}
	if onDemandLimit != nil && *onDemandLimit > 0 {
		if w, ok := valueWindow("on-demand", "On demand", "currency", onDemandLimit, onDemandUsed, nil, end, nil); ok {
			w.Mode = "balance"
			w.Currency = "USD"
			out = append(out, w)
		}
	}
	return out
}
func cent(v *float64) *float64 {
	if v == nil {
		return nil
	}
	x := *v / 100
	return &x
}
func percentWindow(id, label, scope, scopeID string, m map[string]any) (AuthQuotaWindow, bool) {
	p := number(m, "used_percent", "usedPercent")
	if p == nil || *p < 0 || *p > 100 {
		return AuthQuotaWindow{}, false
	}
	l := 100.0
	u := *p
	r := math.Max(0, l-u)
	ur := u / l
	rr := r / l
	w := AuthQuotaWindow{ID: id, Label: label, Scope: scope, ScopeID: scopeID, Mode: "rolling", Unit: "percentage", Limit: &l, Used: &u, Remaining: &r, UsedRatio: &ur, RemainingRatio: &rr, ResetsAt: timeField(m, "reset_at", "resetAt"), DurationSeconds: duration(m)}
	if w.ResetsAt != nil && w.DurationSeconds != nil {
		v := w.ResetsAt.Add(-time.Duration(*w.DurationSeconds) * time.Second)
		w.CycleStartAt = &v
	}
	return w, true
}
func utilWindow(id, label, scope, scopeID string, seconds *int64, m map[string]any) (AuthQuotaWindow, bool) {
	w, ok := valueWindow(id, label, "requests", ptr(1), nil, number(m, "utilization"), timeField(m, "resets_at", "resetsAt"), seconds)
	if ok {
		w.Scope, w.ScopeID = scope, scopeID
		if w.ResetsAt != nil && seconds != nil {
			start := w.ResetsAt.Add(-time.Duration(*seconds) * time.Second)
			w.CycleStartAt = &start
		}
	}
	return w, ok
}
func ptr(v float64) *float64  { return &v }
func int64ptr(v int64) *int64 { return &v }
func valueWindow(id, label, unit string, l, u, util *float64, reset *time.Time, dur *int64) (AuthQuotaWindow, bool) {
	if l != nil && (!finiteNonNegative(*l)) {
		l = nil
	}
	if u != nil && (!finiteNonNegative(*u)) {
		u = nil
	}
	if util != nil && (!finiteNonNegative(*util)) {
		util = nil
	}
	if l == nil && u == nil && util == nil {
		return AuthQuotaWindow{}, false
	}
	if util != nil && u == nil {
		v := *util
		if v > 1 {
			v /= 100
		}
		if l != nil {
			x := *l * v
			u = &x
		}
	}
	var r, ur, rr *float64
	if l != nil && u != nil {
		x := math.Max(0, *l-*u)
		r = &x
		if *l > 0 {
			y := *u / *l
			ur = &y
			z := math.Max(0, 1-y)
			rr = &z
		}
	}
	return AuthQuotaWindow{ID: id, Label: label, Scope: "account", Mode: "rolling", Unit: unit, Limit: l, Used: u, Remaining: r, UsedRatio: ur, RemainingRatio: rr, ResetsAt: reset, DurationSeconds: dur}, true
}
func kimiWindow(id, label string, detail, window map[string]any) (AuthQuotaWindow, bool) {
	reset := timeField(detail, "resetTime", "resetAt", "reset_time", "reset_at")
	if reset == nil {
		if seconds := number(detail, "resetIn", "reset_in", "ttl"); seconds != nil {
			v := time.Now().UTC().Add(time.Duration(*seconds * float64(time.Second)))
			reset = &v
		}
	}
	w, ok := valueWindow(id, label, "requests", number(detail, "limit", "total", "max", "quota"), number(detail, "used", "usage", "consumed"), number(detail, "utilization", "used_percent"), reset, kimiDuration(window))
	if !ok {
		return w, false
	}
	if remaining := number(detail, "remaining"); remaining != nil {
		w.Remaining = remaining
		if w.Limit != nil && *w.Limit > 0 {
			ratio := math.Max(0, *remaining / *w.Limit)
			w.RemainingRatio = &ratio
			if w.Used == nil {
				used := math.Max(0, *w.Limit-*remaining)
				w.Used = &used
			}
		}
	}
	if w.ResetsAt != nil && w.DurationSeconds != nil {
		start := w.ResetsAt.Add(-time.Duration(*w.DurationSeconds) * time.Second)
		w.CycleStartAt = &start
	}
	return w, true
}

func object(m map[string]any, key string) (map[string]any, bool) {
	v, ok := m[key]
	x, ok2 := v.(map[string]any)
	return x, ok && ok2
}
func number(m map[string]any, keys ...string) *float64 {
	for _, k := range keys {
		if x, ok := m[k]; ok {
			if n, ok := numeric(x); ok && finiteNonNegative(n) {
				return &n
			}
		}
	}
	return nil
}
func firstNumber(m map[string]any, keys ...string) *float64 { return number(m, keys...) }
func finiteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}
func numeric(x any) (float64, bool) {
	if m, ok := x.(map[string]any); ok {
		return numeric(m["val"])
	}
	switch v := x.(type) {
	case float64:
		return v, !math.IsNaN(v) && !math.IsInf(v, 0)
	case string:
		n, e := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return n, e == nil && !math.IsNaN(n) && !math.IsInf(n, 0)
	case json.Number:
		n, e := v.Float64()
		return n, e == nil && !math.IsNaN(n) && !math.IsInf(n, 0)
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}
func timeField(m map[string]any, keys ...string) *time.Time {
	for _, k := range keys {
		if x, ok := m[k]; ok {
			if n, ok := numeric(x); ok {
				var t time.Time
				if n > 1e12 {
					t = time.UnixMilli(int64(n))
				} else {
					t = time.Unix(int64(n), 0)
				}
				t = t.UTC()
				return &t
			}
			if s, ok := x.(string); ok {
				for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05", time.DateOnly} {
					if t, e := time.Parse(layout, s); e == nil {
						t = t.UTC()
						return &t
					}
				}
			}
		}
	}
	return nil
}
func duration(m map[string]any) *int64 {
	for _, k := range []string{"limit_window_seconds", "duration_seconds", "reset_after_seconds", "duration"} {
		if n := number(m, k); n != nil {
			v := *n
			if k == "duration" {
				u := strings.ToLower(findText(m, "duration_unit"))
				if strings.HasPrefix(u, "hour") {
					v *= 3600
				} else if strings.HasPrefix(u, "minute") {
					v *= 60
				}
			}
			x := int64(v)
			return &x
		}
	}
	return nil
}
func kimiDuration(window map[string]any) *int64 {
	if window == nil {
		return nil
	}
	v := number(window, "duration")
	if v == nil {
		return nil
	}
	unit := strings.ToLower(findText(window, "timeUnit", "time_unit", "duration_unit"))
	multiplier := float64(1)
	switch {
	case strings.Contains(unit, "week"):
		multiplier = 604800
	case strings.Contains(unit, "day"):
		multiplier = 86400
	case strings.Contains(unit, "hour"):
		multiplier = 3600
	case strings.Contains(unit, "minute"):
		multiplier = 60
	}
	seconds := int64(*v * multiplier)
	return &seconds
}

func has(ws []AuthQuotaWindow, id string) bool {
	for _, w := range ws {
		if w.ID == id {
			return true
		}
	}
	return false
}
func first(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
func quotaProvider(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	switch {
	case strings.Contains(p, "codex") || strings.Contains(p, "openai"):
		return "codex"
	case strings.Contains(p, "claude") || strings.Contains(p, "anthropic"):
		return "claude"
	case strings.Contains(p, "antigravity") || strings.Contains(p, "google"):
		return "antigravity"
	case strings.Contains(p, "kimi") || strings.Contains(p, "moonshot"):
		return "kimi"
	case p == "xai" || strings.Contains(p, "grok"):
		return "xai"
	}
	return ""
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
		if usage.RequestCount > 0 {
			a := float64(tokens) / float64(usage.RequestCount)
			w.AverageTokensPerRequest = &a
			if w.Used != nil && w.Remaining != nil && *w.Used > 0 {
				n := int64(math.Floor(float64(usage.RequestCount) * *w.Remaining / *w.Used))
				w.EstimatedRemainingRequests = &n
				w.PredictionAvailable = true
			}
		}
	}
	return item
}

func cycleStart(provider string, window *AuthQuotaWindow) time.Time {
	if window.CycleStartAt != nil {
		return window.CycleStartAt.UTC()
	}
	if window.ResetsAt == nil {
		return time.Time{}
	}
	if window.DurationSeconds != nil && *window.DurationSeconds > 0 {
		return window.ResetsAt.Add(-time.Duration(*window.DurationSeconds) * time.Second).UTC()
	}
	if provider == "xai" {
		return window.ResetsAt.AddDate(0, -1, 0).UTC()
	}
	return time.Time{}
}

func (s *Service) windowUsage(ctx context.Context, filter store.AuthQuotaUsageFilter, provider, scopeID string) (store.AuthQuotaUsage, bool, error) {
	if provider != "antigravity" {
		usage, err := s.store.GetAuthQuotaUsage(ctx, filter)
		return usage, err == nil, err
	}
	byModel, err := s.store.GetAuthQuotaUsageByModel(ctx, filter)
	if err != nil {
		return store.AuthQuotaUsage{}, false, err
	}
	var result store.AuthQuotaUsage
	for _, item := range byModel {
		pool := antigravityModelPool(item.Model)
		if pool == "" {
			return store.AuthQuotaUsage{}, false, nil
		}
		if pool == scopeID {
			addAuthQuotaUsage(&result, item.AuthQuotaUsage)
		}
	}
	return result, true, nil
}

func antigravityModelPool(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(model, "gemini"):
		return "gemini"
	case strings.Contains(model, "claude"), strings.Contains(model, "gpt"), strings.Contains(model, "openai"):
		return "third-party"
	default:
		return ""
	}
}

func addAuthQuotaUsage(to *store.AuthQuotaUsage, from store.AuthQuotaUsage) {
	to.RequestCount += from.RequestCount
	to.InputTokens += from.InputTokens
	to.OutputTokens += from.OutputTokens
	to.ReasoningTokens += from.ReasoningTokens
	to.CachedTokens += from.CachedTokens
	to.CacheReadTokens += from.CacheReadTokens
	to.CacheCreationTokens += from.CacheCreationTokens
	to.ActualCostMicroUSD += from.ActualCostMicroUSD
}

func totalAuthQuotaTokens(usage store.AuthQuotaUsage) int64 {
	return usage.InputTokens + usage.OutputTokens + usage.ReasoningTokens + usage.CachedTokens + usage.CacheReadTokens + usage.CacheCreationTokens
}

func authQuotaLocalUsage(usage store.AuthQuotaUsage) *AuthQuotaLocalUsage {
	return &AuthQuotaLocalUsage{
		RequestCount: usage.RequestCount, InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens,
		ReasoningTokens: usage.ReasoningTokens, CachedTokens: usage.CachedTokens, CacheReadTokens: usage.CacheReadTokens,
		CacheCreationTokens: usage.CacheCreationTokens, TotalTokens: totalAuthQuotaTokens(usage),
		EstimatedCostMicroUSD: int64(usage.ActualCostMicroUSD),
	}
}

func timePtr(value time.Time) *time.Time {
	value = value.UTC()
	return &value
}
