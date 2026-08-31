package management

import (
	"strings"
	"testing"
)

func TestConsolePageBlocksAPIBaseOverride(t *testing.T) {
	response := consolePage()
	if got := response.Headers.Get("Content-Security-Policy"); got != "connect-src 'self'" {
		t.Fatalf("console CSP = %q, want connect-src 'self'", got)
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
		"function stripDangerousQuery",
		"function isSameOriginBase",
		"function assertSameOriginRequest",
		"function savedSessionToken",
		"history.replaceState",
		"credit_manager_mgmt_token_origin",
		"已拒绝向非同源地址发送管理密钥",
		"input.readOnly = true",
		"本页只连接当前站点",
		`id="apiBase"`,
		"readonly",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing same-origin management lock: %q", text)
		}
	}
	if !strings.Contains(page, "stripDangerousQuery();\n  const savedBase = detectDefaultBase();") {
		t.Fatal("console does not strip api_base query parameters before boot")
	}
	if !strings.Contains(page, "if (saved && isSameOriginBase(apiBase()))") {
		t.Fatal("console still auto-sends a saved management token without a same-origin check")
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
