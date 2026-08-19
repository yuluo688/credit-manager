package management

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/keys"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

// Routes returns exact management routes under /v0/management.
func Routes() []pluginapi.ManagementRoute {
	paths := []struct {
		method, path string
	}{
		{http.MethodGet, "credit-manager/health"},
		{http.MethodGet, "credit-manager/overview"},
		{http.MethodPost, "credit-manager/callers"},
		{http.MethodGet, "credit-manager/callers"},
		{http.MethodPost, "credit-manager/callers/enabled"},
		{http.MethodPost, "credit-manager/keys"},
		{http.MethodGet, "credit-manager/keys"},
		{http.MethodPost, "credit-manager/keys/update"},
		{http.MethodPost, "credit-manager/keys/rotate"},
		{http.MethodPost, "credit-manager/keys/revoke"},
		{http.MethodPost, "credit-manager/keys/reveal"},
		{http.MethodPost, "credit-manager/keys/delete"},
		{http.MethodPost, "credit-manager/pricing"},
		{http.MethodGet, "credit-manager/pricing"},
		{http.MethodPost, "credit-manager/pricing/delete"},
		{http.MethodGet, "credit-manager/usage"},
		{http.MethodGet, "credit-manager/usage/summary"},
		{http.MethodGet, "credit-manager/audit"},
		{http.MethodGet, "credit-manager/balance"},
		{http.MethodGet, "credit-manager/auth-quotas"},
	}
	out := make([]pluginapi.ManagementRoute, 0, len(paths))
	for _, item := range paths {
		out = append(out, pluginapi.ManagementRoute{
			Method: item.method,
			Path:   item.path,
		})
	}
	return out
}

// Resources returns browser-navigable pages shown in the management UI sidebar.
// Host exposes them under /v0/resource/plugins/<pluginID>/...
func Resources() []pluginapi.ResourceRoute {
	return []pluginapi.ResourceRoute{
		{
			Path:        "/console",
			Menu:        "CPA 额度管理",
			Description: "Key / 模型 / 限额 / 使用统计可视化管理",
		},
		{
			Path: "/lookup",
		},
		{
			Path: "/lookup/data",
		},
	}
}

func Handle(ctx context.Context, req pluginapi.ManagementRequest) (pluginapi.ManagementResponse, error) {
	// Resource routes are unauthenticated browser pages (sidebar iframe).
	if resourcePath, ok := resourceRelativePath(req.Path); ok {
		if req.Method == http.MethodGet && (resourcePath == "console" || resourcePath == "") {
			return consolePage(), nil
		}
		if req.Method == http.MethodGet && resourcePath == "lookup" {
			return lookupPage(), nil
		}
		if req.Method == http.MethodGet && resourcePath == "lookup/data" {
			svc := service.Current()
			if svc == nil {
				return jsonErr(http.StatusServiceUnavailable, "service not configured"), nil
			}
			return lookupKey(ctx, svc, req.Headers, req.Query)
		}
		return htmlErr(http.StatusNotFound, "unknown resource"), nil
	}

	svc := service.Current()
	if svc == nil {
		return jsonErr(http.StatusServiceUnavailable, "service not configured"), nil
	}
	path := normalizePath(req.Path)
	switch {
	case req.Method == http.MethodGet && path == "credit-manager/health":
		return jsonOK(map[string]any{
			"status":  "ok",
			"plugin":  service.PluginID,
			"version": service.PluginVersion,
		}), nil
	case req.Method == http.MethodGet && path == "credit-manager/overview":
		return getOverview(ctx, svc, req.Query)
	case req.Method == http.MethodPost && path == "credit-manager/callers":
		return createCaller(ctx, svc, req.Body)
	case req.Method == http.MethodGet && path == "credit-manager/callers":
		return listCallers(ctx, svc, req.Query)
	case req.Method == http.MethodPost && path == "credit-manager/callers/enabled":
		return setEnabled(ctx, svc, req.Body)
	case req.Method == http.MethodPost && path == "credit-manager/keys":
		return createKey(ctx, svc, req.Body)
	case req.Method == http.MethodGet && path == "credit-manager/keys":
		return listKeys(ctx, svc, req.Query)
	case req.Method == http.MethodPost && path == "credit-manager/keys/update":
		return updateKey(ctx, svc, req.Body)
	case req.Method == http.MethodPost && path == "credit-manager/keys/rotate":
		return rotateKey(ctx, svc, req.Body)
	case req.Method == http.MethodPost && path == "credit-manager/keys/revoke":
		return revokeKey(ctx, svc, req.Body)
	case req.Method == http.MethodPost && path == "credit-manager/keys/reveal":
		return revealKey(ctx, svc, req.Body)
	case req.Method == http.MethodPost && path == "credit-manager/keys/delete":
		return deleteKeyPermanently(ctx, svc, req.Body)
	case req.Method == http.MethodPost && path == "credit-manager/pricing":
		return putPricing(ctx, svc, req.Body)
	case req.Method == http.MethodGet && path == "credit-manager/pricing":
		return listPricing(ctx, svc)
	case req.Method == http.MethodPost && path == "credit-manager/pricing/delete":
		return deletePricing(ctx, svc, req.Body)
	case req.Method == http.MethodGet && path == "credit-manager/usage":
		return listUsage(ctx, svc, req.Query)
	case req.Method == http.MethodGet && path == "credit-manager/usage/summary":
		return usageSummary(ctx, svc, req.Query)
	case req.Method == http.MethodGet && path == "credit-manager/audit":
		return listAudit(ctx, svc, req.Query)
	case req.Method == http.MethodGet && path == "credit-manager/balance":
		return getBalance(ctx, svc, req.Query)
	case req.Method == http.MethodGet && path == "credit-manager/auth-quotas":
		return getAuthQuotas(ctx, svc)
	default:
		return jsonErr(http.StatusNotFound, "unknown management route"), nil
	}
}

func lookupKey(ctx context.Context, svc *service.Service, headers http.Header, query map[string][]string) (pluginapi.ManagementResponse, error) {
	key, err := svc.LookupPluginKeyFromHeaders(ctx, headers)
	if err != nil {
		return jsonErr(http.StatusUnauthorized, "invalid or unavailable plugin key"), nil
	}
	filter, err := lookupUsageFilter(query, key.ID)
	if err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	recentOnly := firstQuery(query, "recent_only") == "1"
	if recentOnly {
		return lookupRecentUsage(ctx, svc, filter, query)
	}
	overview, err := svc.Store().GetKeyUsageOverview(ctx, key.ID, time.Now().UTC())
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	byModel, err := svc.Store().SummarizeUsageByModelFiltered(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	dailyTrend, err := svc.Store().SummarizeUsageDailyFiltered(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	usageSummary, err := svc.Store().SummarizeUsageFiltered(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	pageSize := queryInt(query, "page_size", 50)
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	page := queryInt(query, "page", 1)
	if page < 1 {
		page = 1
	}
	filter.Limit = pageSize
	total, err := svc.Store().CountUsage(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	if totalPages == 0 {
		page = 1
	} else if int64(page) > totalPages {
		page = int(totalPages)
	}
	filter.Offset = (page - 1) * pageSize
	recent, err := svc.Store().ListUsage(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	recentView := make([]map[string]any, 0, len(recent))
	for _, item := range recent {
		recentView = append(recentView, publicUsageView(item))
	}
	response := pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    http.Header{"Cache-Control": []string{"no-store"}},
		Body: mustJSON(map[string]any{
			"key": map[string]any{
				"label":                   key.Label,
				"fingerprint":             key.Fingerprint,
				"quota_micro_usd":         keyQuota(key),
				"daily_quota_micro_usd":   key.DailyQuotaMicroUSD,
				"weekly_quota_micro_usd":  key.WeeklyQuotaMicroUSD,
				"monthly_quota_micro_usd": key.MonthlyQuotaMicroUSD,
				"max_concurrent_requests": key.MaxConcurrentRequests,
				"settled_spend_micro_usd": key.SettledSpendMicroUSD,
				"held_amount_micro_usd":   key.HeldAmountMicroUSD,
				"remaining_micro_usd":     key.RemainingMicroUSD(),
				"allowed_models":          key.AllowedModels,
				"expires_at":              key.ExpiresAt,
			},
			"overview":      lookupOverviewView(overview),
			"by_model":      byModel,
			"daily_trend":   dailyTrend,
			"usage_summary": lookupUsageSummaryView(usageSummary),
			"recent_usage":  recentView,
			"recent_pagination": map[string]any{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": totalPages,
			},
		}),
	}
	return response, nil
}

func lookupRecentUsage(ctx context.Context, svc *service.Service, filter store.UsageFilter, query map[string][]string) (pluginapi.ManagementResponse, error) {
	pageSize := queryInt(query, "page_size", 50)
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	page := queryInt(query, "page", 1)
	if page < 1 {
		page = 1
	}
	filter.Limit = pageSize
	total, err := svc.Store().CountUsage(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	if totalPages == 0 {
		page = 1
	} else if int64(page) > totalPages {
		page = int(totalPages)
	}
	filter.Offset = (page - 1) * pageSize
	recent, err := svc.Store().ListUsage(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	recentView := make([]map[string]any, 0, len(recent))
	for _, item := range recent {
		recentView = append(recentView, publicUsageView(item))
	}
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    http.Header{"Cache-Control": []string{"no-store"}},
		Body: mustJSON(map[string]any{
			"recent_usage": recentView,
			"recent_pagination": map[string]any{
				"page":        page,
				"page_size":   pageSize,
				"total":       total,
				"total_pages": totalPages,
			},
		}),
	}, nil
}

func lookupUsageSummaryView(summary store.UsageFilteredSummary) map[string]any {
	return map[string]any{
		"request_count":         summary.RequestCount,
		"input_tokens":          summary.InputTokens,
		"output_tokens":         summary.OutputTokens,
		"reasoning_tokens":      summary.ReasoningTokens,
		"cached_tokens":         summary.CachedTokens,
		"cache_read_tokens":     summary.CacheReadTokens,
		"cache_creation_tokens": summary.CacheCreationTokens,
		"total_tokens":          summary.TotalTokens,
		"cost_micro_usd":        summary.CostMicroUSD,
	}
}

func lookupUsageFilter(query map[string][]string, keyID string) (store.UsageFilter, error) {
	filter, err := usageFilterFromQuery(query, 50)
	if err != nil {
		return store.UsageFilter{}, err
	}
	// The public endpoint always scopes filters to the authenticated Key.
	filter.PluginKeyID = keyID
	filter.CallerID = ""
	filter.Source = ""
	filter.MinCostMicroUSD = nil
	filter.MaxCostMicroUSD = nil
	filter.MinTokens = nil
	filter.MaxTokens = nil
	return filter, nil
}

func keyQuota(key store.PluginKey) money.MicroUSD {
	if key.QuotaMicroUSD == nil {
		return 0
	}
	return *key.QuotaMicroUSD
}

func lookupOverviewView(overview store.KeyUsageOverview) map[string]any {
	return map[string]any{
		"request_count":       overview.RequestCount,
		"cost_micro_usd":      overview.CostMicroUSD,
		"input_tokens":        overview.InputTokens,
		"output_tokens":       overview.OutputTokens,
		"active_reservations": overview.ActiveReservations,
		"daily_micro_usd":     overview.DailyMicroUSD,
		"weekly_micro_usd":    overview.WeeklyMicroUSD,
		"monthly_micro_usd":   overview.MonthlyMicroUSD,
	}
}

func publicUsageView(entry store.UsageEntry) map[string]any {
	return map[string]any{
		"model":                    entry.Model,
		"input_tokens":             entry.Usage.Input,
		"output_tokens":            entry.Usage.Output,
		"reasoning_tokens":         entry.Usage.Reasoning,
		"cached_tokens":            entry.Usage.Cached,
		"cache_read_tokens":        entry.Usage.CacheRead,
		"cache_creation_tokens":    entry.Usage.CacheCreation,
		"total_tokens":             money.ReportedTotal(entry.Usage),
		"cost_micro_usd":           entry.CostMicroUSD,
		"estimated_cost_micro_usd": entry.EstimatedCostMicroUSD,
		"source":                   entry.Source,
		"result":                   entry.Metrics.Result,
		"first_token_latency_ms":   durationMilliseconds(entry.Metrics.FirstTokenLatency),
		"generation_duration_ms":   durationMilliseconds(entry.Metrics.GenerationDuration),
		"tokens_per_second":        entry.Metrics.TokensPerSecond,
		"thinking_intensity":       entry.Metrics.ThinkingIntensity,
		"created_at":               entry.CreatedAt,
	}
}

func createCaller(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		Enabled     *bool  `json:"enabled"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	caller, err := svc.Store().CreateCaller(ctx, store.CallerSpec{
		ID: req.ID, DisplayName: req.DisplayName, QuotaMicroUSD: 0, Enabled: enabled,
	})
	if err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	return jsonOK(callerView(caller)), nil
}

func listCallers(ctx context.Context, svc *service.Service, query map[string][]string) (pluginapi.ManagementResponse, error) {
	limit := queryInt(query, "limit", 100)
	items, err := svc.Store().ListCallers(ctx, limit)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, callerView(item))
	}
	return jsonOK(map[string]any{"items": out}), nil
}

func setEnabled(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		CallerID string `json:"caller_id"`
		Enabled  bool   `json:"enabled"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	if err := svc.Store().SetCallerEnabled(ctx, req.CallerID, req.Enabled); err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	return jsonOK(map[string]any{"caller_id": req.CallerID, "enabled": req.Enabled}), nil
}

func createKey(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		CallerID              string   `json:"caller_id"`
		Label                 string   `json:"label"`
		KeyMaterial           string   `json:"key_material"`
		ExpiresAt             string   `json:"expires_at"`
		QuotaMicroUSD         *int64   `json:"quota_micro_usd"`
		TotalQuotaMicroUSD    *int64   `json:"total_quota_micro_usd"`
		DailyQuotaMicroUSD    *int64   `json:"daily_quota_micro_usd"`
		WeeklyQuotaMicroUSD   *int64   `json:"weekly_quota_micro_usd"`
		MonthlyQuotaMicroUSD  *int64   `json:"monthly_quota_micro_usd"`
		MaxConcurrentRequests *int64   `json:"max_concurrent_requests"`
		AllowedModels         []string `json:"allowed_models"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	var expires *time.Time
	if strings.TrimSpace(req.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return jsonErr(http.StatusBadRequest, "expires_at must be RFC3339"), nil
		}
		expires = &parsed
	}
	// nil or 0 = unlimited
	quota := money.MicroUSD(0)
	if req.QuotaMicroUSD != nil {
		if *req.QuotaMicroUSD < 0 {
			return jsonErr(http.StatusBadRequest, "quota_micro_usd must not be negative"), nil
		}
		quota = money.MicroUSD(*req.QuotaMicroUSD)
	}
	if req.TotalQuotaMicroUSD != nil {
		if *req.TotalQuotaMicroUSD < 0 {
			return jsonErr(http.StatusBadRequest, "total_quota_micro_usd must not be negative"), nil
		}
		quota = money.MicroUSD(*req.TotalQuotaMicroUSD)
	}
	if err := validateKeyLimitFields(req.DailyQuotaMicroUSD, req.WeeklyQuotaMicroUSD, req.MonthlyQuotaMicroUSD, req.MaxConcurrentRequests); err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	key, material, err := svc.MintKeyWithPolicy(ctx, service.MintKeyRequest{
		CallerID:              req.CallerID,
		Label:                 req.Label,
		KeyMaterial:           req.KeyMaterial,
		ExpiresAt:             expires,
		QuotaMicroUSD:         quota,
		DailyQuotaMicroUSD:    optionalMicroUSD(req.DailyQuotaMicroUSD),
		WeeklyQuotaMicroUSD:   optionalMicroUSD(req.WeeklyQuotaMicroUSD),
		MonthlyQuotaMicroUSD:  optionalMicroUSD(req.MonthlyQuotaMicroUSD),
		MaxConcurrentRequests: optionalInt64(req.MaxConcurrentRequests),
		AllowedModels:         req.AllowedModels,
	})
	if err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	headers := http.Header{}
	headers.Set("Cache-Control", "no-store")
	view := keyView(key)
	view["plaintext"] = material.Plaintext
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       mustJSON(view),
	}, nil
}

func rotateKey(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID          string `json:"id"`
		KeyMaterial string `json:"key_material"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	key, material, err := svc.RotateKey(ctx, req.ID, req.KeyMaterial)
	if err != nil {
		if errors.Is(err, store.ErrPluginKeyNotFound) {
			return jsonErr(http.StatusNotFound, err.Error()), nil
		}
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	headers := http.Header{}
	headers.Set("Cache-Control", "no-store")
	view := keyView(key)
	view["plaintext"] = material.Plaintext
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       mustJSON(view),
	}, nil
}

func listKeys(ctx context.Context, svc *service.Service, query map[string][]string) (pluginapi.ManagementResponse, error) {
	callerID := firstQuery(query, "caller_id")
	limit := queryInt(query, "limit", 200)
	var (
		items []store.PluginKey
		err   error
	)
	if callerID == "" {
		items, err = svc.Store().ListPluginKeys(ctx, limit)
	} else {
		items, err = svc.Store().ListPluginKeysByCaller(ctx, callerID, limit)
	}
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, keyView(item))
	}
	return jsonOK(map[string]any{"items": out}), nil
}

func updateKey(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID                    string   `json:"id"`
		Label                 *string  `json:"label"`
		Enabled               *bool    `json:"enabled"`
		QuotaMicroUSD         *int64   `json:"quota_micro_usd"`
		TotalQuotaMicroUSD    *int64   `json:"total_quota_micro_usd"`
		DailyQuotaMicroUSD    *int64   `json:"daily_quota_micro_usd"`
		WeeklyQuotaMicroUSD   *int64   `json:"weekly_quota_micro_usd"`
		MonthlyQuotaMicroUSD  *int64   `json:"monthly_quota_micro_usd"`
		MaxConcurrentRequests *int64   `json:"max_concurrent_requests"`
		AllowedModels         []string `json:"allowed_models"`
		SetAllowedModels      bool     `json:"set_allowed_models"`
		ExpiresAt             string   `json:"expires_at"`
		ClearExpiresAt        bool     `json:"clear_expires_at"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	update := store.PluginKeyPolicyUpdate{
		ID:             req.ID,
		Label:          req.Label,
		Enabled:        req.Enabled,
		ClearExpiresAt: req.ClearExpiresAt,
	}
	if req.QuotaMicroUSD != nil {
		q := money.MicroUSD(*req.QuotaMicroUSD)
		update.QuotaMicroUSD = &q
	}
	if req.TotalQuotaMicroUSD != nil {
		q := money.MicroUSD(*req.TotalQuotaMicroUSD)
		update.QuotaMicroUSD = &q
	}
	if err := validateKeyLimitFields(req.DailyQuotaMicroUSD, req.WeeklyQuotaMicroUSD, req.MonthlyQuotaMicroUSD, req.MaxConcurrentRequests); err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	if req.DailyQuotaMicroUSD != nil {
		q := money.MicroUSD(*req.DailyQuotaMicroUSD)
		update.DailyQuotaMicroUSD = &q
	}
	if req.WeeklyQuotaMicroUSD != nil {
		q := money.MicroUSD(*req.WeeklyQuotaMicroUSD)
		update.WeeklyQuotaMicroUSD = &q
	}
	if req.MonthlyQuotaMicroUSD != nil {
		q := money.MicroUSD(*req.MonthlyQuotaMicroUSD)
		update.MonthlyQuotaMicroUSD = &q
	}
	if req.MaxConcurrentRequests != nil {
		q := *req.MaxConcurrentRequests
		update.MaxConcurrentRequests = &q
	}
	if req.SetAllowedModels {
		models := append([]string(nil), req.AllowedModels...)
		update.AllowedModels = &models
	}
	if strings.TrimSpace(req.ExpiresAt) != "" && !req.ClearExpiresAt {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return jsonErr(http.StatusBadRequest, "expires_at must be RFC3339"), nil
		}
		update.ExpiresAt = &parsed
	}
	key, err := svc.Store().UpdatePluginKeyPolicy(ctx, update)
	if err != nil {
		if errors.Is(err, store.ErrPluginKeyNotFound) {
			return jsonErr(http.StatusNotFound, err.Error()), nil
		}
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	return jsonOK(keyView(key)), nil
}

func revokeKey(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	if err := svc.Store().RevokePluginKey(ctx, req.ID); err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	return jsonOK(map[string]any{"id": req.ID, "revoked": true}), nil
}

func revealKey(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	key, err := svc.Store().GetPluginKey(ctx, req.ID)
	if err != nil {
		if errors.Is(err, store.ErrPluginKeyNotFound) {
			return jsonErr(http.StatusNotFound, err.Error()), nil
		}
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	if len(key.EncryptedKeyMaterial) == 0 {
		return jsonErr(http.StatusConflict, "this key was created before recoverable storage; rotate it to create a viewable replacement"), nil
	}
	plaintext, err := keys.DecryptPlaintext(key.EncryptedKeyMaterial, svc.Peppers())
	if err != nil {
		return jsonErr(http.StatusInternalServerError, "decrypt plugin key: "+err.Error()), nil
	}
	if _, ok := keys.Verify(plaintext, key.KeyHash, key.PepperID, svc.Peppers()); !ok {
		return jsonErr(http.StatusInternalServerError, "stored plugin key material does not match its verification hash"), nil
	}
	headers := http.Header{}
	headers.Set("Cache-Control", "no-store")
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       mustJSON(map[string]any{"id": key.ID, "plaintext": plaintext}),
	}, nil
}

func deleteKeyPermanently(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	if strings.TrimSpace(req.ID) == "" {
		return jsonErr(http.StatusBadRequest, "id is required"), nil
	}
	if err := svc.Store().DeletePluginKey(ctx, req.ID); err != nil {
		if errors.Is(err, store.ErrPluginKeyNotFound) {
			return jsonErr(http.StatusNotFound, err.Error()), nil
		}
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	return jsonOK(map[string]any{"id": req.ID, "deleted": true, "history_retained": true}), nil
}

func putPricing(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID        string `json:"id"`
		MatchKind string `json:"match_kind"`
		Pattern   string `json:"pattern"`
		Priority  int    `json:"priority"`
		Enabled   *bool  `json:"enabled"`
		Price     struct {
			Input          int64  `json:"input"`
			Output         int64  `json:"output"`
			Reasoning      int64  `json:"reasoning"`
			Cached         int64  `json:"cached"`
			CacheRead      int64  `json:"cache_read"`
			CacheCreation  int64  `json:"cache_creation"`
			AccountingMode string `json:"accounting_mode"`
			BillingMode    string `json:"billing_mode"`
			PerImage       int64  `json:"per_image"`
		} `json:"price"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule := store.PricingRule{
		ID: req.ID, MatchKind: store.MatchKind(req.MatchKind), Pattern: req.Pattern, Priority: req.Priority, Enabled: enabled,
		Price: money.PricePerMTok{
			Input: money.MicroUSD(req.Price.Input), Output: money.MicroUSD(req.Price.Output),
			Reasoning: money.MicroUSD(req.Price.Reasoning), Cached: money.MicroUSD(req.Price.Cached),
			CacheRead: money.MicroUSD(req.Price.CacheRead), CacheCreation: money.MicroUSD(req.Price.CacheCreation),
			AccountingMode: strings.TrimSpace(req.Price.AccountingMode),
			BillingMode:    strings.TrimSpace(req.Price.BillingMode),
			PerImage:       money.MicroUSD(req.Price.PerImage),
		},
	}
	if err := svc.Store().PutPricingRule(ctx, rule); err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	return jsonOK(map[string]any{"id": rule.ID, "saved": true}), nil
}

func listPricing(ctx context.Context, svc *service.Service) (pluginapi.ManagementResponse, error) {
	items, err := svc.Store().ListPricingRules(ctx)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	return jsonOK(map[string]any{"items": items}), nil
}

func deletePricing(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	if err := svc.Store().DeletePricingRule(ctx, req.ID); err != nil {
		if errors.Is(err, store.ErrPricingRuleNotFound) {
			return jsonErr(http.StatusNotFound, err.Error()), nil
		}
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	return jsonOK(map[string]any{"id": req.ID, "deleted": true}), nil
}

func listUsage(ctx context.Context, svc *service.Service, query map[string][]string) (pluginapi.ManagementResponse, error) {
	pageSize := queryInt(query, "page_size", queryInt(query, "limit", 50))
	if pageSize > 200 {
		pageSize = 200
	}
	filter, err := usageFilterFromQuery(query, pageSize)
	if err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	filter.Limit = pageSize
	page := queryInt(query, "page", 1)
	if page < 1 {
		page = 1
	}
	total, err := svc.Store().CountUsage(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	totalPages := (total + int64(filter.Limit) - 1) / int64(filter.Limit)
	if totalPages == 0 {
		page = 1
	} else if int64(page) > totalPages {
		page = int(totalPages)
	}
	filter.Offset = (page - 1) * filter.Limit
	items, err := svc.Store().ListUsage(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, usageView(item))
	}
	return jsonOK(map[string]any{
		"items":       out,
		"page":        page,
		"page_size":   filter.Limit,
		"total":       total,
		"total_pages": totalPages,
	}), nil
}

func usageSummary(ctx context.Context, svc *service.Service, query map[string][]string) (pluginapi.ManagementResponse, error) {
	filter, err := usageFilterFromQuery(query, 100)
	if err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	byKey, err := svc.Store().SummarizeUsageByKeyFiltered(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	byModel, err := svc.Store().SummarizeUsageByModelFiltered(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	return jsonOK(map[string]any{
		"by_key":   byKey,
		"by_model": byModel,
		"filters":  usageFilterView(filter),
	}), nil
}

func usageFilterFromQuery(query map[string][]string, fallbackLimit int) (store.UsageFilter, error) {
	filter := store.UsageFilter{
		CallerID:     firstQuery(query, "caller_id"),
		PluginKeyID:  firstQuery(query, "plugin_key_id"),
		Model:        firstQuery(query, "model"),
		Source:       firstQuery(query, "source"),
		AuthID:       firstQuery(query, "auth_id"),
		AuthProvider: firstQuery(query, "auth_provider"),
		AuthIndex:    firstQuery(query, "auth_index"),
		Limit:        queryInt(query, "limit", fallbackLimit),
	}
	var err error
	if filter.From, err = queryTime(query, "from"); err != nil {
		return store.UsageFilter{}, err
	}
	if filter.To, err = queryTime(query, "to"); err != nil {
		return store.UsageFilter{}, err
	}
	if filter.MinCostMicroUSD, err = queryMicroUSD(query, "min_cost_micro_usd"); err != nil {
		return store.UsageFilter{}, err
	}
	if filter.MaxCostMicroUSD, err = queryMicroUSD(query, "max_cost_micro_usd"); err != nil {
		return store.UsageFilter{}, err
	}
	if filter.MinTokens, err = queryOptionalInt64(query, "min_tokens"); err != nil {
		return store.UsageFilter{}, err
	}
	if filter.MaxTokens, err = queryOptionalInt64(query, "max_tokens"); err != nil {
		return store.UsageFilter{}, err
	}
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return store.UsageFilter{}, errors.New("from must not be later than to")
	}
	if filter.MinCostMicroUSD != nil && filter.MaxCostMicroUSD != nil && *filter.MinCostMicroUSD > *filter.MaxCostMicroUSD {
		return store.UsageFilter{}, errors.New("min_cost_micro_usd must not be greater than max_cost_micro_usd")
	}
	if filter.MinTokens != nil && filter.MaxTokens != nil && *filter.MinTokens > *filter.MaxTokens {
		return store.UsageFilter{}, errors.New("min_tokens must not be greater than max_tokens")
	}
	return filter, nil
}

func usageFilterView(filter store.UsageFilter) map[string]any {
	view := map[string]any{
		"caller_id":          filter.CallerID,
		"plugin_key_id":      filter.PluginKeyID,
		"model":              filter.Model,
		"source":             filter.Source,
		"auth_id":            filter.AuthID,
		"auth_provider":      filter.AuthProvider,
		"auth_index":         filter.AuthIndex,
		"limit":              filter.Limit,
		"min_tokens":         filter.MinTokens,
		"max_tokens":         filter.MaxTokens,
		"min_cost_micro_usd": filter.MinCostMicroUSD,
		"max_cost_micro_usd": filter.MaxCostMicroUSD,
	}
	if filter.From != nil {
		view["from"] = filter.From.UTC()
	}
	if filter.To != nil {
		view["to"] = filter.To.UTC()
	}
	return view
}

func queryTime(query map[string][]string, key string) (*time.Time, error) {
	raw := firstQuery(query, key)
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		value, err = time.Parse("2006-01-02T15:04", raw)
	}
	if err != nil {
		return nil, errors.New(key + " must be RFC3339 or YYYY-MM-DDTHH:MM")
	}
	value = value.UTC()
	return &value, nil
}

func queryMicroUSD(query map[string][]string, key string) (*money.MicroUSD, error) {
	raw := firstQuery(query, key)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return nil, errors.New(key + " must be a non-negative integer")
	}
	micro := money.MicroUSD(value)
	return &micro, nil
}

func queryOptionalInt64(query map[string][]string, key string) (*int64, error) {
	raw := firstQuery(query, key)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return nil, errors.New(key + " must be a non-negative integer")
	}
	return &value, nil
}

func getOverview(ctx context.Context, svc *service.Service, query map[string][]string) (pluginapi.ManagementResponse, error) {
	filter, err := usageFilterFromQuery(query, 500)
	if err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	keys, err := svc.Store().ListPluginKeys(ctx, 500)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	pricing, err := svc.Store().ListPricingRules(ctx)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	byKey, err := svc.Store().SummarizeUsageByKeyFiltered(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	byModel, err := svc.Store().SummarizeUsageByModelFiltered(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	usedModels, err := svc.Store().SummarizeUsageByModelFiltered(ctx, store.UsageFilter{})
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	usedAuths, err := svc.Store().ListUsedAuths(ctx)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	recent, err := svc.Store().ListUsage(ctx, filter)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	keyViews := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		keyViews = append(keyViews, keyView(k))
	}
	usageViews := make([]map[string]any, 0, len(recent))
	for _, u := range recent {
		usageViews = append(usageViews, usageView(u))
	}
	return jsonOK(map[string]any{
		"status":         "ok",
		"plugin":         service.PluginID,
		"version":        service.PluginVersion,
		"data_dir":       svc.Config().DataDir,
		"keys":           keyViews,
		"pricing":        pricing,
		"usage_by_key":   byKey,
		"usage_by_model": byModel,
		"used_models":    usedModels,
		"used_auths":     usedAuths,
		"filters":        usageFilterView(filter),
		"recent_usage":   usageViews,
	}), nil
}

func listAudit(ctx context.Context, svc *service.Service, query map[string][]string) (pluginapi.ManagementResponse, error) {
	items, err := svc.Store().ListAuditEventsFiltered(ctx, store.AuditFilter{
		CallerID:    firstQuery(query, "caller_id"),
		PluginKeyID: firstQuery(query, "plugin_key_id"),
		Limit:       queryInt(query, "limit", 100),
	})
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, auditView(item))
	}
	return jsonOK(map[string]any{"items": out}), nil
}

func getBalance(ctx context.Context, svc *service.Service, query map[string][]string) (pluginapi.ManagementResponse, error) {
	keyID := firstQuery(query, "key_id")
	if keyID == "" {
		return jsonErr(http.StatusBadRequest, "key_id query is required"), nil
	}
	key, err := svc.Store().GetPluginKey(ctx, keyID)
	if err != nil {
		return jsonErr(http.StatusNotFound, err.Error()), nil
	}
	return jsonOK(keyView(key)), nil
}

// HostCallbackIDContextKey identifies the host callback associated with a management request.
// The host will populate it when management requests expose callback metadata.
type HostCallbackIDContextKey struct{}

func getAuthQuotas(ctx context.Context, svc *service.Service) (pluginapi.ManagementResponse, error) {
	hostCallbackID, _ := ctx.Value(HostCallbackIDContextKey{}).(string)
	overview, err := svc.AuthQuotaOverview(ctx, hostCallbackID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unavailable") {
			return jsonErrNoStore(http.StatusServiceUnavailable, "auth quota information unavailable"), nil
		}
		return jsonErrNoStore(http.StatusInternalServerError, "auth quota information failed"), nil
	}
	return jsonOKNoStore(map[string]any{"items": overview}), nil
}

func callerView(caller store.Caller) map[string]any {
	return map[string]any{
		"id":           caller.ID,
		"display_name": caller.DisplayName,
		"enabled":      caller.Enabled,
		"created_at":   caller.CreatedAt,
		"updated_at":   caller.UpdatedAt,
	}
}

func auditView(event store.AuditEvent) map[string]any {
	return map[string]any{
		"id":               event.ID,
		"caller_id":        event.CallerID,
		"plugin_key_id":    event.PluginKeyID,
		"reservation_id":   event.ReservationID,
		"event_type":       event.EventType,
		"amount_micro_usd": event.AmountMicroUSD,
		"details_json":     event.DetailsJSON,
		"created_at":       event.CreatedAt,
	}
}

func keyView(key store.PluginKey) map[string]any {
	quota := money.MicroUSD(0)
	if key.QuotaMicroUSD != nil {
		quota = *key.QuotaMicroUSD
	}
	return map[string]any{
		"id":                      key.ID,
		"kid":                     key.Kid,
		"fingerprint":             key.Fingerprint,
		"label":                   key.Label,
		"enabled":                 key.Enabled,
		"quota_micro_usd":         quota,
		"total_quota_micro_usd":   quota,
		"daily_quota_micro_usd":   key.DailyQuotaMicroUSD,
		"weekly_quota_micro_usd":  key.WeeklyQuotaMicroUSD,
		"monthly_quota_micro_usd": key.MonthlyQuotaMicroUSD,
		"max_concurrent_requests": key.MaxConcurrentRequests,
		"settled_spend_micro_usd": key.SettledSpendMicroUSD,
		"held_amount_micro_usd":   key.HeldAmountMicroUSD,
		"remaining_micro_usd":     key.RemainingMicroUSD(),
		"allowed_models":          key.AllowedModels,
		"revoked_at":              key.RevokedAt,
		"expires_at":              key.ExpiresAt,
		"last_used_at":            key.LastUsedAt,
		"created_at":              key.CreatedAt,
	}
}

func validateKeyLimitFields(limits ...*int64) error {
	for _, limit := range limits {
		if limit != nil && *limit < 0 {
			return errors.New("key limits must not be negative")
		}
	}
	return nil
}

func optionalMicroUSD(value *int64) money.MicroUSD {
	if value == nil {
		return 0
	}
	return money.MicroUSD(*value)
}

func optionalInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func usageView(entry store.UsageEntry) map[string]any {
	return map[string]any{
		"id":                       entry.ID,
		"plugin_key_id":            entry.PluginKeyID,
		"auth_id":                  entry.Auth.AuthID,
		"auth_index":               entry.Auth.AuthIndex,
		"auth_name":                entry.Auth.Name,
		"auth_label":               entry.Auth.Label,
		"auth_provider":            entry.Auth.Provider,
		"auth_type":                entry.Auth.Type,
		"auth_email":               entry.Auth.Email,
		"auth_path":                entry.Auth.Path,
		"model":                    entry.Model,
		"pricing_rule_id":          entry.PricingRuleID,
		"input_tokens":             entry.Usage.Input,
		"output_tokens":            entry.Usage.Output,
		"reasoning_tokens":         entry.Usage.Reasoning,
		"cached_tokens":            entry.Usage.Cached,
		"cache_read_tokens":        entry.Usage.CacheRead,
		"cache_creation_tokens":    entry.Usage.CacheCreation,
		"total_tokens":             money.ReportedTotal(entry.Usage),
		"cost_micro_usd":           entry.CostMicroUSD,
		"estimated_cost_micro_usd": entry.EstimatedCostMicroUSD,
		"source":                   entry.Source,
		"tier":                     entry.Metrics.Tier,
		"result":                   entry.Metrics.Result,
		"first_token_latency_ms":   durationMilliseconds(entry.Metrics.FirstTokenLatency),
		"generation_duration_ms":   durationMilliseconds(entry.Metrics.GenerationDuration),
		"tokens_per_second":        entry.Metrics.TokensPerSecond,
		"thinking_intensity":       entry.Metrics.ThinkingIntensity,
		"created_at":               entry.CreatedAt,
	}
}

func durationMilliseconds(value *time.Duration) *int64 {
	if value == nil {
		return nil
	}
	milliseconds := value.Milliseconds()
	return &milliseconds
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "/v0/management/")
	path = strings.TrimPrefix(path, "/")
	return path
}

// resourceRelativePath returns the path under /v0/resource/plugins/<pluginID>/.
func resourceRelativePath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", false
	}
	base := "/v0/resource/plugins/" + service.PluginID
	switch {
	case path == base, path == base+"/":
		return "", true
	case strings.HasPrefix(path, base+"/"):
		return strings.TrimPrefix(path, base+"/"), true
	case path == "v0/resource/plugins/"+service.PluginID, path == "v0/resource/plugins/"+service.PluginID+"/":
		return "", true
	case strings.HasPrefix(path, "v0/resource/plugins/"+service.PluginID+"/"):
		return strings.TrimPrefix(path, "v0/resource/plugins/"+service.PluginID+"/"), true
	case path == "/console", path == "console":
		return "console", true
	default:
		return "", false
	}
}

func firstQuery(query map[string][]string, key string) string {
	if query == nil {
		return ""
	}
	if values := query[key]; len(values) > 0 {
		return strings.TrimSpace(values[0])
	}
	for k, values := range query {
		if strings.EqualFold(k, key) && len(values) > 0 {
			return strings.TrimSpace(values[0])
		}
	}
	return ""
}

func queryInt(query map[string][]string, key string, fallback int) int {
	raw := firstQuery(query, key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func jsonOK(v any) pluginapi.ManagementResponse {
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       mustJSON(v),
	}
}

func jsonErr(status int, message string) pluginapi.ManagementResponse {
	return pluginapi.ManagementResponse{
		StatusCode: status,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       mustJSON(map[string]any{"error": message}),
	}
}

func jsonOKNoStore(v any) pluginapi.ManagementResponse {
	response := jsonOK(v)
	response.Headers.Set("Cache-Control", "no-store")
	return response
}

func jsonErrNoStore(status int, message string) pluginapi.ManagementResponse {
	response := jsonErr(status, message)
	response.Headers.Set("Cache-Control", "no-store")
	return response
}

func htmlErr(status int, message string) pluginapi.ManagementResponse {
	return pluginapi.ManagementResponse{
		StatusCode: status,
		Headers:    http.Header{"Content-Type": []string{"text/plain; charset=utf-8"}},
		Body:       []byte(message),
	}
}

func mustJSON(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"error":"encode_failed"}`)
	}
	return raw
}
