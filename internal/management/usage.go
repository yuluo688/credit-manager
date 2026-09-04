package management

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func listUsage(ctx context.Context, svc *service.Service, query map[string][]string) (pluginapi.ManagementResponse, error) {
	pageSize := queryInt(query, "page_size", queryInt(query, "limit", 10))
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
