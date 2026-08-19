package management

import (
	"strings"
	"testing"

	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
)

func TestFXResourceRoute(t *testing.T) {
	var found bool
	for _, resource := range Resources() {
		if resource.Path != "/fx/usd-cny" {
			continue
		}
		found = true
		if resource.Menu != "" {
			t.Fatalf("fx menu = %q, want no sidebar entry", resource.Menu)
		}
	}
	if !found {
		t.Fatal("fx resource is not registered")
	}
	path, ok := resourceRelativePath("/v0/resource/plugins/" + service.PluginID + "/fx/usd-cny")
	if !ok || path != "fx/usd-cny" {
		t.Fatalf("resourceRelativePath fx = %q, %t", path, ok)
	}
}

func TestConsoleFetchesLiveUsdCnyRate(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"function fetchUsdCnyRate",
		"/fx/usd-cny",
		`id="usdCnyRate" data-no-i18n`,
		"实时美元兑人民币汇率",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing live FX support: %q", text)
		}
	}
	if strings.Contains(page, `id="usdCnyRate" type="number"`) {
		t.Fatal("console still uses a manual USD/CNY rate input")
	}
}

func TestLookupFetchesLiveUsdCnyRate(t *testing.T) {
	page := string(lookupPage().Body)
	for _, text := range []string{
		"function fetchUsdCnyRate",
		"/fx/usd-cny",
		`id="usdCnyRate" data-no-i18n`,
		"实时美元兑人民币汇率",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("lookup page is missing live FX support: %q", text)
		}
	}
	if strings.Contains(page, `id="usdCnyRate" type="number"`) {
		t.Fatal("lookup still uses a manual USD/CNY rate input")
	}
}

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
	for _, text := range []string{"recentPagination", "recentPageSize", "page_size=", "white-space:nowrap", "filter-panel", "page-summary", "page-meta", "page-nav", "grid-template-columns:repeat(3,minmax(0,1fr))", "grid-template-columns:repeat(5,minmax(0,1fr))", "tokenUnitSwitch", "currencySwitch", "data-token-unit=\"qian\" title=\"千 (×1,000)\" data-no-i18n>千", "data-token-unit=\"k\" title=\"k (×1,000)\" data-no-i18n>k", "data-token-unit=\"wan\" title=\"万 (×10,000)\" data-no-i18n>万", "data-token-unit=\"w\" title=\"w (×10,000)\" data-no-i18n>w", "data-token-unit=\"baiwan\" title=\"百万 (×1,000,000)\" data-no-i18n>百万", "data-token-unit=\"m\" title=\"m (×1,000,000)\" data-no-i18n>m", "suffix:'千'", "suffix:'k'", "suffix:'万'", "suffix:'w'", "suffix:'百万'", "suffix:'m'"} {
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

func TestLookupPageFollowsCPALanguageAndTheme(t *testing.T) {
	page := string(lookupPage().Body)
	for _, text := range []string{
		`data-theme="auto"`,
		`data-palette="white"`,
		"cli-proxy-language",
		"cli-proxy-theme",
		`data-locale="zh-CN"`,
		`data-locale="zh-TW"`,
		`data-locale="en"`,
		`data-locale="ru"`,
		`data-theme-value="auto"`,
		`data-theme-value="white"`,
		`data-theme-value="light"`,
		`data-theme-value="dark"`,
		"跟随系统",
		"纯白",
		"羊毛纸",
		"暗色",
		"function applyLocale",
		"function applyTheme",
		"function persistCPA",
		"简体中文",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("lookup page is missing CPA language/theme support: %q", text)
		}
	}
}

func TestLookupTrendGrainSwitch(t *testing.T) {
	page := string(lookupPage().Body)
	for _, text := range []string{
		`data-trend-grain="hour"`,
		`data-trend-grain="day"`,
		`data-trend-grain="month"`,
		"function setTrendGrain",
		"function getTrendGrain",
		"function bucketTrend",
		`data-trend-chart="token"`,
		`data-trend-chart="cost"`,
		"grain=hour",
		"containLabel:true",
		"share-body",
		"trend-metric-row",
		"legend:{show:false",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("lookup page is missing trend grain switch: %q", text)
		}
	}
	if strings.Contains(page, "class=\"chart-legend\"") {
		t.Fatal("lookup page still uses a duplicate HTML chart legend")
	}
}

func TestLookupModelEfficiencyRank(t *testing.T) {
	page := string(lookupPage().Body)
	for _, text := range []string{
		"模型效率排行",
		`id="modelRank"`,
		`id="modelRankMetric"`,
		`data-rank-metric="value"`,
		`data-rank-metric="unit"`,
		`data-rank-metric="cache"`,
		`data-rank-metric="tps"`,
		"function renderModelRank",
		"function modelRankSpec",
		"function drillLookupModel",
		"rank-card",
		"avg_tokens_per_second",
		"cache_read_tokens",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("lookup page is missing model efficiency rank: %q", text)
		}
	}
	if strings.Contains(page, "min-height:570px") {
		t.Fatal("lookup share card still spans the full analytics height")
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
		`data-token-unit="raw" title="原始数量" data-no-i18n>个`,
		`data-token-unit="qian" title="千 (×1,000)" data-no-i18n>千`,
		`data-token-unit="k" title="k (×1,000)" data-no-i18n>k`,
		`data-token-unit="wan" title="万 (×10,000)" data-no-i18n>万`,
		`data-token-unit="w" title="w (×10,000)" data-no-i18n>w`,
		`data-token-unit="baiwan" title="百万 (×1,000,000)" data-no-i18n>百万`,
		`data-token-unit="m" title="m (×1,000,000)" data-no-i18n>m`,
		`suffix: '千'`,
		`suffix: 'k'`,
		`suffix: '万'`,
		`suffix: 'w'`,
		`suffix: '百万'`,
		`suffix: 'm'`,
		`cfg.suffix`,
		`data-no-i18n`,
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing token units: %q", text)
		}
	}
}

func TestConsoleModelEfficiencyRank(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"模型效率排行",
		`id="overviewModelRank"`,
		`id="modelRankMetricSwitch"`,
		`data-rank-metric="value"`,
		`data-rank-metric="unit"`,
		`data-rank-metric="cache"`,
		`data-rank-metric="tps"`,
		"function renderOverviewRank",
		"function modelRankSpec",
		"function setModelRankMetric",
		"overview-rank-card",
		"grid-column:2; grid-row:2",
		"tpsSum",
		"cacheRelated",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing model efficiency rank: %q", text)
		}
	}
	if strings.Contains(page, "grid-row:1 / span 2") {
		t.Fatal("model share card still spans both dashboard rows")
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
		`data-trend-chart="token"`,
		`data-trend-chart="cost"`,
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

func TestConsoleTokenTotalsMatchCapTracker(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"const reported = Number(item.total_tokens || 0);",
		"Number(item.input_tokens || 0) + Number(item.output_tokens || 0) + Number(item.reasoning_tokens || 0)",
		"to.getFullYear(), to.getMonth(), to.getDate()",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing CAP-aligned token totals: %q", text)
		}
	}
	if strings.Contains(page, "cached_tokens || 0) + Number(item.cache_read_tokens || 0) + Number(item.cache_creation_tokens || 0));") {
		t.Fatal("console still adds cache counters into total tokens")
	}
	lookup := string(lookupPage().Body)
	if !strings.Contains(lookup, "const reported=Number(item.total_tokens||0); if(reported>0) return reported;") {
		t.Fatal("lookup page does not use official total_tokens")
	}
}

func TestConsoleImagePricingUsesPerImageBilling(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"function isImageGenerationModel",
		"function modelsDevTokenPriced",
		"function syncPriceBillingFields",
		`value="per_image"`,
		"按张（出图）",
		"每张 USD",
		"不能套用 Token 价",
		"billing_mode",
		"per_image",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing image pricing support: %q", text)
		}
	}
}

func TestConsoleAuthQuotaViewIsManagementOnly(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"data-tab=\"auth-quotas\"", "credit-manager/auth-quotas", "刷新最多每 15 分钟查询一次", "auth-quota-window-card",
		"auth-quota-week-select", "额度周", "authQuotaIsWeekly", "authQuotaIsFiveHour", "authQuotaDisplayWindows", "includes('secondary')", "authQuotaCostForecast", "当前费用", "预估剩余", "预计可用", "authQuotaProviderFilter", "authQuotaNameFilter", "overflow-x:auto", "state.currentTab === 'auth-quotas'",
		"align-items:stretch", "flex-direction:column", "height:100%",
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
