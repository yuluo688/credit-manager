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
