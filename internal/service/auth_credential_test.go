package service

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/yuluo688/credit-manager/internal/config"
)

func TestPluginKeyFromRequestPriority(t *testing.T) {
	const token = "tk-kid-secret"
	for _, test := range []struct {
		name    string
		headers http.Header
		query   url.Values
		path    string
		want    string
	}{
		{name: "bearer", headers: http.Header{"Authorization": {"Bearer " + token}}, want: token},
		{name: "bearer case", headers: http.Header{"authorization": {"bearer " + token}}, want: token},
		{name: "goog", headers: http.Header{"X-Goog-Api-Key": {token}}, want: token},
		{name: "goog lower", headers: http.Header{"x-goog-api-key": {token}}, want: token},
		{name: "anthropic", headers: http.Header{"X-Api-Key": {token}}, want: token},
		{name: "query key", query: url.Values{"key": {token}}, want: token},
		{name: "query auth token", query: url.Values{"auth_token": {token}}, want: token},
		{name: "path query", path: "/v1beta/models/gemini:generateContent?key=" + token, want: token},
		{name: "path query among others", path: "/v1/messages?beta=true&key=" + token + "&x=1", want: token},
		{
			name:    "bearer wins",
			headers: http.Header{"Authorization": {"Bearer from-bearer"}, "X-Goog-Api-Key": {token}},
			query:   url.Values{"key": {token}},
			want:    "from-bearer",
		},
		{
			name:    "goog wins over api key and query",
			headers: http.Header{"X-Goog-Api-Key": {"from-goog"}, "X-Api-Key": {token}},
			query:   url.Values{"key": {token}},
			want:    "from-goog",
		},
		{
			name:    "api key wins over query",
			headers: http.Header{"x-api-key": {"from-api"}},
			query:   url.Values{"key": {token}, "auth_token": {"other"}},
			want:    "from-api",
		},
		{
			name:  "key wins over auth_token",
			query: url.Values{"auth_token": {"other"}, "key": {token}},
			want:  token,
		},
		{
			name:  "explicit query wins over path",
			query: url.Values{"key": {"from-query"}},
			path:  "/v1beta/models/gemini:generateContent?key=" + token,
			want:  "from-query",
		},
		{
			name:  "path fills missing query key",
			query: url.Values{"page": {"1"}},
			path:  "/v1/messages?key=" + token,
			want:  token,
		},
		{name: "empty bearer falls through", headers: http.Header{"Authorization": {"Bearer"}, "X-Api-Key": {token}}, want: token},
		{name: "blank", path: "/v1/chat/completions"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := PluginKeyFromRequest(test.headers, test.query, test.path); got != test.want {
				t.Fatalf("PluginKeyFromRequest() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestAuthenticateAcceptsNativeAPIKeyLocations(t *testing.T) {
	ctx := context.Background()
	svc := openCredentialTestService(t)
	_, material, err := svc.MintKey(ctx, BootstrapCallerID, "native", 1_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	token := material.Plaintext
	for _, test := range []struct {
		name    string
		headers http.Header
		query   url.Values
		path    string
	}{
		{name: "bearer", headers: http.Header{"Authorization": {"Bearer " + token}}},
		{name: "goog", headers: http.Header{"x-goog-api-key": {token}}},
		{name: "anthropic", headers: http.Header{"X-Api-Key": {"Bearer " + token}}},
		{name: "query", query: url.Values{"key": {token}}},
		{name: "auth_token", query: url.Values{"auth_token": {token}}},
		{name: "path", path: "/v1beta/models/gemini-3.5-flash:generateContent?key=" + url.QueryEscape(token)},
	} {
		t.Run(test.name, func(t *testing.T) {
			principal, meta, ok := svc.Authenticate(ctx, test.headers, test.query, test.path)
			if !ok || principal != material.Principal || meta["kid"] == "" {
				t.Fatalf("Authenticate() = %q %#v %t", principal, meta, ok)
			}
			key, _, err := svc.ResolveIdentity(ctx, test.headers, test.query, nil)
			if test.path != "" && test.query == nil && len(test.headers) == 0 {
				if err == nil {
					t.Fatal("execution identity must not invent a key from a path it never sees")
				}
				return
			}
			if err != nil || key.ID == "" || key.Principal != material.Principal {
				t.Fatalf("ResolveIdentity() = %#v %v", key, err)
			}
			lookedUp, err := svc.LookupPluginKeyFromHeaders(ctx, test.headers, test.query)
			if err != nil || lookedUp.ID != key.ID {
				t.Fatalf("LookupPluginKeyFromHeaders() = %#v %v", lookedUp, err)
			}
		})
	}

	if _, _, ok := svc.Authenticate(ctx, http.Header{"X-Goog-Api-Key": {"tk-not-a-real-key"}}, nil, ""); ok {
		t.Fatal("invalid x-goog-api-key authenticated")
	}
	if _, _, ok := svc.Authenticate(ctx, http.Header{
		"Authorization":  {"Bearer not-a-key"},
		"X-Goog-Api-Key": {token},
	}, nil, ""); ok {
		t.Fatal("a present Bearer token must not fall through to x-goog-api-key")
	}
}

func openCredentialTestService(t *testing.T) *Service {
	t.Helper()
	t.Setenv("CREDIT_MANAGER_TEST_PEPPERS", "active:0123456789abcdef0123456789abcdef")
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	cfg.Keys.PepperEnv = "CREDIT_MANAGER_TEST_PEPPERS"
	cfg.Keys.ActivePepperID = "active"
	svc, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	return svc
}
