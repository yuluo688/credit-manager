package management

import (
	"strings"
	"testing"

	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
)

func TestLookupResourceRoute(t *testing.T) {
	var found bool
	for _, resource := range Resources() {
		if resource.Path != "/lookup" {
			continue
		}
		found = true
		if resource.Menu != "" {
			t.Fatalf("lookup menu = %q, want no sidebar entry", resource.Menu)
		}
	}
	if !found {
		t.Fatal("lookup resource is not registered")
	}
}

func TestLookupPageDoesNotPersistKey(t *testing.T) {
	response := lookupPage()
	page := string(response.Body)
	if response.Headers.Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", response.Headers.Get("Cache-Control"))
	}
	for _, text := range []string{"sessionStorage", "JSON.stringify({key})", "localStorage.setItem('credit-manager.key"} {
		if strings.Contains(page, text) {
			t.Fatalf("lookup page must not contain %q", text)
		}
	}
	if !strings.Contains(page, "Authorization:'Bearer '+token") {
		t.Fatal("lookup page does not send the Key as a Bearer header")
	}
	if !strings.Contains(page, "recent_only=1") {
		t.Fatal("lookup page does not use the lightweight recent-only pagination request")
	}
	for _, text := range []string{"function normalizeKey", "function validateKey", "Key 包含非英文字符"} {
		if !strings.Contains(page, text) {
			t.Fatalf("lookup page is missing mobile-safe Key validation: %q", text)
		}
	}
	for _, text := range []string{"recentPagination", "recentPageSize", "page_size=", "white-space:nowrap", "filter-panel", "page-summary", "page-meta", "page-nav", "grid-template-columns:repeat(3,minmax(0,1fr))", "grid-template-columns:repeat(5,minmax(0,1fr))", "tokenUnitSwitch", "currencySwitch", "data-token-unit=\"k\" title=\"千 (×1,000)\">千", "suffix:'千'", "suffix:'万'", "suffix:'百万'"} {
		if !strings.Contains(page, text) {
			t.Fatalf("lookup page is missing recent usage pagination or one-line headers: %q", text)
		}
	}
}

func TestPublicUsageViewExcludesAuthIdentity(t *testing.T) {
	view := publicUsageView(store.UsageEntry{Auth: store.AuthIdentity{
		AuthID: "host-credential", Email: "host@example.test", Path: "C:/host/credential.json",
	}})
	for _, forbidden := range []string{"auth_id", "auth_email", "auth_name", "auth_path", "plugin_key_id", "caller_id"} {
		if _, found := view[forbidden]; found {
			t.Fatalf("public usage view exposes %q", forbidden)
		}
	}
}

func TestLookupPathRecognition(t *testing.T) {
	path, ok := resourceRelativePath("/v0/resource/plugins/" + service.PluginID + "/lookup")
	if !ok || path != "lookup" {
		t.Fatalf("resourceRelativePath lookup = %q, %t", path, ok)
	}
}

func TestConsoleTokenUnitsUseChineseLabels(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		`data-token-unit="k" title="千 (×1,000)">千`,
		`data-token-unit="w" title="万 (×10,000)">万`,
		`data-token-unit="m" title="百万 (×1,000,000)">百万`,
		`suffix: '千'`,
		`suffix: '万'`,
		`suffix: '百万'`,
		`t(cfg.suffix)`,
		`'百万': { 'zh-TW':'百萬', en:'m', ru:'m' }`,
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing Chinese token units: %q", text)
		}
	}
}

func TestConsoleTrendGrainSwitch(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		`data-trend-grain="hour"`,
		`data-trend-grain="day"`,
		`data-trend-grain="month"`,
		"function overviewTrendSeries",
		"function setTrendGrain",
		"function trendMetric",
		"trend-metric-row",
		"趋势时间维度",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing trend grain switch: %q", text)
		}
	}
	if strings.Contains(page, "function overviewDailySeries") {
		t.Fatal("console page still uses daily-only trend aggregation")
	}
}

func TestConsoleClipboardFallbackSupportsMobileBrowsers(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"navigator.clipboard && typeof navigator.clipboard.writeText === 'function'",
		"document.execCommand && document.execCommand('copy')",
		"复制失败，请手动复制",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing clipboard fallback: %q", text)
		}
	}
}

func TestConsoleMobileFlashIsReadable(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"top:calc(88px + env(safe-area-inset-top))",
		"max-height:min(42vh, 280px)",
		"overflow-y:auto",
		"overflow-wrap:anywhere",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing mobile flash protection: %q", text)
		}
	}
}

func TestConsoleClosesConnectionAfterSuccessfulLoad(t *testing.T) {
	page := string(consolePage().Body)
	successPath := "await reloadWithModelCatalog();\n      closeConnectionModal();"
	if !strings.Contains(page, successPath) {
		t.Fatal("connection dialog is not closed after a successful load")
	}
}

func TestConsoleAuthQuotaViewIsManagementOnly(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"data-tab=\"auth-quotas\"", "credit-manager/auth-quotas", "刷新最多每 15 分钟查询一次", "auth-quota-window-card",
		"auth-quota-week-select", "额度周", "authQuotaIsWeekly", "authQuotaIsFiveHour", "authQuotaDisplayWindows", "includes('secondary')", "authQuotaCostForecast", "当前费用", "预估剩余", "预计可用", "authQuotaProviderFilter", "authQuotaNameFilter", "overflow-x:auto", "state.currentTab === 'auth-quotas'",
		"align-items:start", "flex-direction:column",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console auth quota view is missing %q", text)
		}
	}
	for _, removed := range []string{"auth-quota-windows", "本地归因", "平均 Token/请求", "预计剩余请求", "authQuotaWeekFilter", "authQuotaPeriodFilter", "额度区间"} {
		if strings.Contains(page, removed) {
			t.Fatalf("console auth quota view still exposes removed field %q", removed)
		}
	}
	lookup := string(lookupPage().Body)
	for _, forbidden := range []string{"auth-quotas", "认证额度"} {
		if strings.Contains(lookup, forbidden) {
			t.Fatalf("public lookup page exposes auth quota UI: %q", forbidden)
		}
	}
}
