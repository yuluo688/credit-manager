package management

import (
	"context"
	"net/http"

	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

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
