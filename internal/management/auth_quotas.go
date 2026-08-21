package management

import (
	"context"
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
