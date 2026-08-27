package management

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/yuluo688/credit-manager/internal/service"

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
		{http.MethodPost, "credit-manager/pricing/enabled"},
		{http.MethodPost, "credit-manager/pricing/delete"},
		{http.MethodGet, "credit-manager/usage"},
		{http.MethodGet, "credit-manager/usage/summary"},
		{http.MethodGet, "credit-manager/audit"},
		{http.MethodGet, "credit-manager/balance"},
		{http.MethodGet, "credit-manager/auth-quotas"},
		{http.MethodPost, "credit-manager/auth-quotas/refresh"},
		{http.MethodPost, "credit-manager/auth-quotas/concurrency"},
		{http.MethodPost, "credit-manager/auth-quotas/concurrency/batch"},
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
		{
			Path: "/fx/usd-cny",
		},
		{
			Path: "/models-dev",
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
		if req.Method == http.MethodGet && resourcePath == "fx/usd-cny" {
			return usdCNYRate(ctx, req.Query)
		}
		if req.Method == http.MethodGet && resourcePath == "models-dev" {
			return modelsDevCatalog(ctx, req.Query)
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
	case req.Method == http.MethodPost && path == "credit-manager/pricing/enabled":
		return setPricingEnabled(ctx, svc, req.Body)
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
		return getAuthQuotas(ctx, svc, req.Query)
	case req.Method == http.MethodPost && path == "credit-manager/auth-quotas/refresh":
		return refreshAuthQuota(ctx, svc, req.Body)
	case req.Method == http.MethodPost && path == "credit-manager/auth-quotas/concurrency/batch":
		return updateAuthQuotaConcurrencyBatch(ctx, svc, req.Body)
	case req.Method == http.MethodPost && path == "credit-manager/auth-quotas/concurrency":
		return updateAuthQuotaConcurrency(ctx, svc, req.Body)
	default:
		return jsonErr(http.StatusNotFound, "unknown management route"), nil
	}
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
