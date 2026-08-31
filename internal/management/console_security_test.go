package management

import (
	"strings"
	"testing"
)

func TestConsolePageBlocksAPIBaseOverride(t *testing.T) {
	response := consolePage()
	if got := response.Headers.Get("Content-Security-Policy"); got == "connect-src 'self'" {
		t.Fatal("console CSP still blocks opted-in custom API addresses")
	}
	if got := response.Headers.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("console Cache-Control = %q, want no-store", got)
	}
	page := strings.ReplaceAll(string(response.Body), "\r\n", "\n")
	for _, text := range []string{
		"q.get('api_base')",
		`q.get("api_base")`,
		"q.get('api')",
		`q.get("api")`,
		"fromQuery",
	} {
		if strings.Contains(page, text) {
			t.Fatalf("console still reads API base from the page URL: %q", text)
		}
	}
	for _, text := range []string{
		"function isSameOriginBase",
		"function assertSameOriginRequest",
		"function savedSessionToken",
		"function customAPIBaseEnabled",
		"function persistCustomAPIBaseEnabled",
		"function persistCustomAPIBase",
		"function savedCustomAPIBase",
		"function restoreCustomAPIBasePreference",
		"function syncAPIBaseField",
		"credit_manager_mgmt_token_origin",
		"credit_manager_custom_api_enabled",
		"已拒绝向非同源地址发送管理密钥",
		`id="customApiBase"`,
		`id="apiBase"`,
		"自定义 API 地址",
		"input.readOnly = !enabled",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing custom API address gate: %q", text)
		}
	}
	if strings.Contains(page, `id="customApiBase" type="checkbox" checked`) {
		t.Fatal("custom API address checkbox must default to unchecked")
	}
	if strings.Contains(page, "function stripDangerousQuery") || strings.Contains(page, "history.replaceState") {
		t.Fatal("console still mutates the page URL for api_base query parameters")
	}
	if !strings.Contains(page, "if (saved && (customAPIBaseEnabled() || isSameOriginBase(apiBase())))") {
		t.Fatal("console still auto-sends a saved management token without a same-origin or custom-API check")
	}
	if !strings.Contains(page, "if (customAPIBaseEnabled()) return;") {
		t.Fatal("same-origin request lock is not gated by the custom API checkbox")
	}
}

func TestLookupPageSetsConnectSrcCSP(t *testing.T) {
	response := lookupPage()
	if got := response.Headers.Get("Content-Security-Policy"); got != "connect-src 'self'" {
		t.Fatalf("lookup CSP = %q, want connect-src 'self'", got)
	}
	if got := response.Headers.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("lookup Cache-Control = %q, want no-store", got)
	}
}
