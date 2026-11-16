package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/yuluo688/credit-manager/internal/service"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

func TestModelDirectoryFiltersVerifiedKey(t *testing.T) {
	resetDirectoryIntents()
	t.Cleanup(resetDirectoryIntents)
	ctx := context.Background()
	svc := newLifecycleTestService(t)
	_, claudeKey, err := svc.MintKeyWithPolicy(ctx, service.MintKeyRequest{
		CallerID:      service.BootstrapCallerID,
		Label:         "claude-only",
		QuotaMicroUSD: 1_000_000,
		AllowedModels: []string{"claude-*"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, geminiKey, err := svc.MintKeyWithPolicy(ctx, service.MintKeyRequest{
		CallerID:      service.BootstrapCallerID,
		Label:         "gemini-only",
		QuotaMicroUSD: 1_000_000,
		AllowedModels: []string{"gemini-2.5-pro"},
	})
	if err != nil {
		t.Fatal(err)
	}

	catalog := []byte(`{"object":"list","data":[{"id":"claude-sonnet","object":"model"},{"id":"gpt-4o","object":"model"},{"id":"gemini-2.5-pro","object":"model"}]}`)
	geminiCatalog := []byte(`{"models":[{"name":"models/gemini-2.5-pro"},{"name":"models/gemini-2.5-flash"}]}`)

	t.Run("bearer", func(t *testing.T) {
		body := filterDirectory(t, catalog, http.Header{"Authorization": {"Bearer " + claudeKey.Plaintext}})
		assertModelIDs(t, body, "claude-sonnet")
	})
	t.Run("goog header", func(t *testing.T) {
		body := filterDirectory(t, geminiCatalog, http.Header{"X-Goog-Api-Key": {geminiKey.Plaintext}})
		assertModelNames(t, body, "models/gemini-2.5-pro")
	})
	t.Run("anthropic header", func(t *testing.T) {
		body := filterDirectory(t, catalog, http.Header{"x-api-key": {claudeKey.Plaintext}})
		assertModelIDs(t, body, "claude-sonnet")
	})
	t.Run("query", func(t *testing.T) {
		headers := http.Header{"User-Agent": {"gemini-cli"}}
		authenticateDirectory(t, pluginapi.FrontendAuthRequest{
			Method:  http.MethodGet,
			Path:    "/v1beta/models",
			Headers: headers,
			Query:   url.Values{"key": {geminiKey.Plaintext}},
		})
		body := filterDirectory(t, geminiCatalog, headers)
		assertModelNames(t, body, "models/gemini-2.5-pro")
	})
	t.Run("path query", func(t *testing.T) {
		headers := http.Header{"User-Agent": {"claude-cli"}}
		authenticateDirectory(t, pluginapi.FrontendAuthRequest{
			Method:  http.MethodGet,
			Path:    "/v1/models?auth_token=" + url.QueryEscape(claudeKey.Plaintext),
			Headers: headers,
		})
		body := filterDirectory(t, catalog, headers)
		assertModelIDs(t, body, "claude-sonnet")
	})
	t.Run("anonymous", func(t *testing.T) {
		headers := http.Header{"User-Agent": {"management"}}
		authenticateDirectory(t, pluginapi.FrontendAuthRequest{
			Method:  http.MethodGet,
			Path:    "/v1/models",
			Headers: headers,
		})
		body := filterDirectory(t, catalog, headers)
		assertModelIDs(t, body, "claude-sonnet", "gpt-4o", "gemini-2.5-pro")
	})
	t.Run("invalid key stays public", func(t *testing.T) {
		body := filterDirectory(t, catalog, http.Header{"Authorization": {"Bearer tk-not-a-real-key"}})
		assertModelIDs(t, body, "claude-sonnet", "gpt-4o", "gemini-2.5-pro")
	})
	t.Run("header wins over queued query", func(t *testing.T) {
		headers := http.Header{"User-Agent": {"shared-client"}, "Authorization": {"Bearer " + claudeKey.Plaintext}}
		authenticateDirectory(t, pluginapi.FrontendAuthRequest{
			Method:  http.MethodGet,
			Path:    "/v1/models",
			Headers: http.Header{"User-Agent": {"shared-client"}},
			Query:   url.Values{"key": {geminiKey.Plaintext}},
		})
		body := filterDirectory(t, catalog, headers)
		assertModelIDs(t, body, "claude-sonnet")
	})
	t.Run("overlapping query keys fail open", func(t *testing.T) {
		headers := http.Header{"User-Agent": {"same-client"}}
		authenticateDirectory(t, pluginapi.FrontendAuthRequest{
			Method:  http.MethodGet,
			Path:    "/v1/models",
			Headers: headers,
			Query:   url.Values{"key": {claudeKey.Plaintext}},
		})
		authenticateDirectory(t, pluginapi.FrontendAuthRequest{
			Method:  http.MethodGet,
			Path:    "/v1beta/models",
			Headers: headers,
			Query:   url.Values{"key": {geminiKey.Plaintext}},
		})
		body := filterDirectory(t, catalog, headers)
		assertModelIDs(t, body, "claude-sonnet", "gpt-4o", "gemini-2.5-pro")
	})
}

func authenticateDirectory(t *testing.T, req pluginapi.FrontendAuthRequest) {
	t.Helper()
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := authenticate(raw)
	if err != nil {
		t.Fatal(err)
	}
	var env envelope
	if err := json.Unmarshal(resp, &env); err != nil || !env.OK {
		t.Fatalf("auth envelope = %s err=%v", resp, err)
	}
	var auth pluginapi.FrontendAuthResponse
	if err := json.Unmarshal(env.Result, &auth); err != nil {
		t.Fatal(err)
	}
	if !auth.Authenticated || auth.Principal != "credit-manager:model-directory" {
		t.Fatalf("directory auth = %+v", auth)
	}
}

func filterDirectory(t *testing.T, body []byte, headers http.Header) []byte {
	t.Helper()
	raw, err := json.Marshal(pluginapi.ResponseInterceptRequest{
		RequestHeaders: headers,
		Body:           body,
		StatusCode:     http.StatusOK,
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := interceptModelDirectoryResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	var env envelope
	if err := json.Unmarshal(resp, &env); err != nil || !env.OK {
		t.Fatalf("intercept envelope = %s err=%v", resp, err)
	}
	var out pluginapi.ResponseInterceptResponse
	if err := json.Unmarshal(env.Result, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Body) == 0 {
		return body
	}
	return out.Body
}

func assertModelIDs(t *testing.T, body []byte, want ...string) {
	t.Helper()
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Data) != len(want) {
		t.Fatalf("ids = %#v, want %v", payload.Data, want)
	}
	for i, id := range want {
		if payload.Data[i].ID != id {
			t.Fatalf("ids = %#v, want %v", payload.Data, want)
		}
	}
}

func assertModelNames(t *testing.T, body []byte, want ...string) {
	t.Helper()
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Models) != len(want) {
		t.Fatalf("names = %#v, want %v", payload.Models, want)
	}
	for i, name := range want {
		if payload.Models[i].Name != name {
			t.Fatalf("names = %#v, want %v", payload.Models, want)
		}
	}
}
