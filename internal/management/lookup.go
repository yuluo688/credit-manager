package management

import (
	"context"
	"net/http"
	"time"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

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
	dailyTrend, err := svc.Store().SummarizeUsageTrendFiltered(ctx, filter, firstQuery(query, "grain"))
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
	modelTokenUsage, err := svc.Store().ListModelTokenUsage(ctx, key.ID, key.ModelTokenLimits, time.Now().UTC().UnixMilli())
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
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
				"model_token_limits":      modelTokenLimitViews(key.ModelTokenLimits),
				"unmatched_models_mode":   unmatchedModelsModeView(key.UnmatchedModelsMode),
				"expires_at":              key.ExpiresAt,
			},
			"overview":          lookupOverviewView(overview),
			"model_token_usage": modelTokenUsageViews(modelTokenUsage),
			"by_model":          byModel,
			"daily_trend":       dailyTrend,
			"usage_summary":     lookupUsageSummaryView(usageSummary),
			"recent_usage":      recentView,
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
