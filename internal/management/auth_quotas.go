package management

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/yuluo688/credit-manager/internal/service"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

// HostCallbackIDContextKey identifies the host callback associated with a management request.
// The host will populate it when management requests expose callback metadata.
type HostCallbackIDContextKey struct{}

func getAuthQuotas(ctx context.Context, svc *service.Service, query map[string][]string) (pluginapi.ManagementResponse, error) {
	hostCallbackID, _ := ctx.Value(HostCallbackIDContextKey{}).(string)
	overview, err := svc.AuthQuotaOverview(ctx, hostCallbackID, service.AuthQuotaFilter{
		Page:     queryInt(query, "page", 1),
		PageSize: queryInt(query, "page_size", service.AuthQuotaDefaultPageSize),
		Provider: firstQuery(query, "provider"),
		Q:        firstQuery(query, "q"),
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unavailable") {
			return jsonErrNoStore(http.StatusServiceUnavailable, "auth quota information unavailable"), nil
		}
		return jsonErrNoStore(http.StatusInternalServerError, "auth quota information failed"), nil
	}
	return jsonOKNoStore(overview), nil
}

func refreshAuthQuota(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		Provider  string `json:"provider"`
		AuthID    string `json:"auth_id"`
		AuthIndex string `json:"auth_index"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErrNoStore(http.StatusBadRequest, "invalid json"), nil
	}
	if strings.TrimSpace(req.AuthID) == "" && strings.TrimSpace(req.AuthIndex) == "" {
		return jsonErrNoStore(http.StatusBadRequest, "auth_id or auth_index is required"), nil
	}
	hostCallbackID, _ := ctx.Value(HostCallbackIDContextKey{}).(string)
	item, err := svc.RefreshAuthQuota(ctx, hostCallbackID, req.Provider, req.AuthID, req.AuthIndex)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "not found") {
			return jsonErrNoStore(http.StatusNotFound, "auth quota not found"), nil
		}
		if strings.Contains(msg, "unavailable") {
			return jsonErrNoStore(http.StatusServiceUnavailable, "auth quota information unavailable"), nil
		}
		return jsonErrNoStore(http.StatusInternalServerError, "auth quota refresh failed"), nil
	}
	return jsonOKNoStore(map[string]any{"item": item}), nil
}

func updateAuthQuotaConcurrency(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		Provider              string `json:"provider"`
		AuthID                string `json:"auth_id"`
		AuthIndex             string `json:"auth_index"`
		MaxConcurrentRequests *int64 `json:"max_concurrent_requests"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErrNoStore(http.StatusBadRequest, "invalid json"), nil
	}
	authID := strings.TrimSpace(req.AuthID)
	if authID == "" {
		authID = strings.TrimSpace(req.AuthIndex)
	}
	if authID == "" {
		return jsonErrNoStore(http.StatusBadRequest, "auth_id or auth_index is required"), nil
	}
	if req.MaxConcurrentRequests == nil {
		return jsonErrNoStore(http.StatusBadRequest, "max_concurrent_requests is required"), nil
	}
	if *req.MaxConcurrentRequests < 0 {
		return jsonErrNoStore(http.StatusBadRequest, "max_concurrent_requests must not be negative"), nil
	}
	item, err := svc.SetAuthConcurrencyLimit(ctx, req.Provider, authID, *req.MaxConcurrentRequests)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "required") || strings.Contains(msg, "negative") {
			return jsonErrNoStore(http.StatusBadRequest, err.Error()), nil
		}
		if strings.Contains(msg, "unavailable") {
			return jsonErrNoStore(http.StatusServiceUnavailable, "auth quota information unavailable"), nil
		}
		return jsonErrNoStore(http.StatusInternalServerError, "auth concurrency update failed"), nil
	}
	return jsonOKNoStore(map[string]any{"item": item}), nil
}

func updateAuthQuotaConcurrencyBatch(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		Provider              string                          `json:"provider"`
		Q                     string                          `json:"q"`
		MaxConcurrentRequests *int64                          `json:"max_concurrent_requests"`
		Items                 []service.AuthConcurrencyTarget `json:"items"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErrNoStore(http.StatusBadRequest, "invalid json"), nil
	}
	if req.MaxConcurrentRequests == nil {
		return jsonErrNoStore(http.StatusBadRequest, "max_concurrent_requests is required"), nil
	}
	if *req.MaxConcurrentRequests < 0 {
		return jsonErrNoStore(http.StatusBadRequest, "max_concurrent_requests must not be negative"), nil
	}
	result, err := svc.SetAuthConcurrencyLimits(ctx, service.AuthQuotaFilter{Provider: req.Provider, Q: req.Q}, req.Items, *req.MaxConcurrentRequests)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "required") || strings.Contains(msg, "negative") {
			return jsonErrNoStore(http.StatusBadRequest, err.Error()), nil
		}
		if strings.Contains(msg, "unavailable") {
			return jsonErrNoStore(http.StatusServiceUnavailable, "auth quota information unavailable"), nil
		}
		return jsonErrNoStore(http.StatusInternalServerError, "auth concurrency batch update failed"), nil
	}
	return jsonOKNoStore(result), nil
}
