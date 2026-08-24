package management

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/yuluo688/credit-manager/internal/fx"
	"github.com/yuluo688/credit-manager/internal/modelsdev"
	"github.com/yuluo688/credit-manager/internal/service"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func usdCNYRate(ctx context.Context, _ map[string][]string) (pluginapi.ManagementResponse, error) {
	rate, err := fx.GetUSDCNY(ctx, false)
	if err != nil {
		return jsonErrNoStore(http.StatusBadGateway, "usd/cny rate unavailable"), nil
	}
	return jsonOKNoStore(map[string]any{
		"usd_to_cny": rate.USDToCNY,
		"source":     rate.Source,
		"fetched_at": rate.FetchedAt.UTC(),
		"cached":     rate.Cached,
	}), nil
}

func modelsDevCatalog(ctx context.Context, query map[string][]string) (pluginapi.ManagementResponse, error) {
	if svc := service.Current(); svc != nil {
		if dir := strings.TrimSpace(svc.Config().DataDir); dir != "" {
			modelsdev.Default.SetCacheFile(filepath.Join(dir, "models-dev-api.json"))
		}
	}
	catalog := modelsdev.Get(ctx, firstQuery(query, "refresh") == "1")
	providers := catalog.Providers
	if len(providers) == 0 {
		providers = json.RawMessage(`{}`)
	}
	body := map[string]any{
		"catalog":    providers,
		"source":     catalog.Source,
		"fetched_at": catalog.FetchedAt.UTC(),
		"cached":     catalog.Cached,
	}
	if catalog.Error != "" {
		body["error"] = catalog.Error
	}
	return jsonOKNoStore(body), nil
}
