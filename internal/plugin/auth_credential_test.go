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

func TestAuthenticateAcceptsNativeAPIKeyLocations(t *testing.T) {
	ctx := context.Background()
	svc := newLifecycleTestService(t)
	_, material, err := svc.MintKey(ctx, service.BootstrapCallerID, "native-auth", 1_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	token := material.Plaintext
	for _, test := range []struct {
		name string
		req  pluginapi.FrontendAuthRequest
		ok   bool
	}{
		{
			name: "goog header",
			req: pluginapi.FrontendAuthRequest{
				Method:  http.MethodPost,
				Path:    "/v1beta/models/gemini-3.5-flash:generateContent",
				Headers: http.Header{"X-Goog-Api-Key": {token}},
			},
			ok: true,
		},
		{
			name: "anthropic header",
			req: pluginapi.FrontendAuthRequest{
				Method:  http.MethodPost,
				Path:    "/v1/messages",
				Headers: http.Header{"x-api-key": {token}},
			},
			ok: true,
		},
		{
			name: "query",
			req: pluginapi.FrontendAuthRequest{
				Method: http.MethodPost,
				Path:   "/v1beta/models/gemini-3.5-flash:generateContent",
				Query:  url.Values{"key": {token}},
			},
			ok: true,
		},
		{
			name: "path query",
			req: pluginapi.FrontendAuthRequest{
				Method: http.MethodPost,
				Path:   "/v1beta/models/gemini-3.5-flash:generateContent?key=" + url.QueryEscape(token),
			},
			ok: true,
		},
		{
			name: "missing",
			req: pluginapi.FrontendAuthRequest{
				Method: http.MethodPost,
				Path:   "/v1beta/models/gemini-3.5-flash:generateContent",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw, err := json.Marshal(test.req)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := authenticate(raw)
			if err != nil {
				t.Fatal(err)
			}
			var env envelope
			if err := json.Unmarshal(resp, &env); err != nil || !env.OK {
				t.Fatalf("envelope = %s err=%v", resp, err)
			}
			var auth pluginapi.FrontendAuthResponse
			if err := json.Unmarshal(env.Result, &auth); err != nil {
				t.Fatal(err)
			}
			if auth.Authenticated != test.ok {
				t.Fatalf("authenticated = %t, want %t (%s)", auth.Authenticated, test.ok, resp)
			}
			if test.ok && auth.Principal != material.Principal {
				t.Fatalf("principal = %q, want %q", auth.Principal, material.Principal)
			}
		})
	}
}

func TestRouteModelClaimsNativeAPIKey(t *testing.T) {
	for _, test := range []struct {
		name    string
		headers http.Header
		query   url.Values
		want    bool
	}{
		{name: "goog", headers: http.Header{"X-Goog-Api-Key": {"tk-test"}}, want: true},
		{name: "anthropic", headers: http.Header{"x-api-key": {"tk-test"}}, want: true},
		{name: "query", query: url.Values{"key": {"tk-test"}}, want: true},
		{name: "absent"},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw, err := json.Marshal(pluginapi.ModelRouteRequest{
				SourceFormat:   "gemini",
				RequestedModel: "gemini-3.5-flash",
				Headers:        test.headers,
				Query:          test.query,
			})
			if err != nil {
				t.Fatal(err)
			}
			resp, err := routeModel(raw)
			if err != nil {
				t.Fatal(err)
			}
			var env envelope
			if err := json.Unmarshal(resp, &env); err != nil || !env.OK {
				t.Fatalf("envelope = %s err=%v", resp, err)
			}
			var routed pluginapi.ModelRouteResponse
			if err := json.Unmarshal(env.Result, &routed); err != nil {
				t.Fatal(err)
			}
			if routed.Handled != test.want {
				t.Fatalf("handled = %t, want %t (%s)", routed.Handled, test.want, resp)
			}
		})
	}
}
