package management

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

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
