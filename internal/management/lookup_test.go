package management

import (
	"strings"
	"testing"

	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
)

func TestModelsDevResourceRoute(t *testing.T) {
	var found bool
	for _, resource := range Resources() {
		if resource.Path != "/models-dev" {
			continue
		}
		found = true
		if resource.Menu != "" {
			t.Fatalf("models-dev menu = %q, want no sidebar entry", resource.Menu)
		}
	}
	if !found {
		t.Fatal("models-dev resource is not registered")
	}
	path, ok := resourceRelativePath("/v0/resource/plugins/" + service.PluginID + "/models-dev")
	if !ok || path != "models-dev" {
		t.Fatalf("resourceRelativePath models-dev = %q, %t", path, ok)
	}
}

func TestConsoleModelsDevCatalogIsProxiedAndOptional(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"function modelsDevURL",
		"function fetchModelsDevCatalog",
		"/models-dev",
		"Promise.allSettled",
		"价格目录暂不可用",
		"已连接到 CPA，可稍后重试",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing models.dev fallback support: %q", text)
		}
	}
	if strings.Contains(page, "https://models.dev/api.json") {
		t.Fatal("console still fetches models.dev directly from the browser")
	}
}

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
	for _, text := range []string{"id=\"loginGate\"", "id=\"rememberKey\"", "credit-manager.lookup.token", "更换密钥", "data-lookup-authed", "function buildCustomSelect", "function buildCustomDateInput"} {
		if !strings.Contains(page, text) {
			t.Fatalf("lookup page is missing login gate: %q", text)
		}
	}
	if !strings.Contains(page, "Authorization:'Bearer '+token") {
		t.Fatal("lookup page does not send the Key as a Bearer header")
	}
	if !strings.Contains(page, "recent_only=1") {
		t.Fatal("lookup page does not use the lightweight recent-only pagination request")
	}
	for _, text := range []string{"function normalizeKey", "function validateKey", "密钥包含非英文字符"} {
		if !strings.Contains(page, text) {
			t.Fatalf("lookup page is missing mobile-safe Key validation: %q", text)
		}
	}
	for _, text := range []string{"recentPagination", "recentPageSize", "page_size=", "white-space:nowrap", "filter-panel", "page-summary", "page-meta", "page-nav", ".recent-pagination .custom-control-panel", ".recent-pagination .custom-option", "grid-template-columns:repeat(3,minmax(0,1fr))", "grid-template-columns:repeat(5,minmax(0,1fr))", "tokenUnitSwitch", "currencySwitch", "data-token-unit=\"qian\" title=\"千 (×1,000)\" data-no-i18n>千", "data-token-unit=\"k\" title=\"k (×1,000)\" data-no-i18n>k", "data-token-unit=\"wan\" title=\"万 (×10,000)\" data-no-i18n>万", "data-token-unit=\"w\" title=\"w (×10,000)\" data-no-i18n>w", "data-token-unit=\"baiwan\" title=\"百万 (×1,000,000)\" data-no-i18n>百万", "data-token-unit=\"m\" title=\"m (×1,000,000)\" data-no-i18n>m", "suffix:'千'", "suffix:'k'", "suffix:'万'", "suffix:'w'", "suffix:'百万'", "suffix:'m'"} {
		if !strings.Contains(page, text) {
			t.Fatalf("lookup page is missing recent usage pagination or one-line headers: %q", text)
		}
	}
}

func TestLookupCustomDatePickerMatchesConsole(t *testing.T) {
	lookup := string(lookupPage().Body)
	for _, text := range []string{
		`type="datetime-local"`,
		"function buildCustomDateInput",
		"custom-date-panel",
		"custom-date-trigger",
		"选择日期和时间",
		"input[type=\"datetime-local\"]:not(.native-control)",
		"input[type=\"time\"]:not(.native-control)",
		"buildCustomTimeInput",
		"custom-time-panel",
	} {
		if !strings.Contains(lookup, text) {
			t.Fatalf("lookup page is missing management date picker: %q", text)
		}
	}
	if strings.Contains(lookup, `type="date"`) {
		t.Fatal("lookup still uses native date inputs")
	}
	if strings.Contains(lookup, "date-field") {
		t.Fatal("lookup still uses the old date-field wrapper")
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
	page := strings.ReplaceAll(string(consolePage().Body), "\r\n", "\n")
	successPath := "await reloadWithModelCatalog();\n      closeConnectionModal();"
	if !strings.Contains(page, successPath) {
		t.Fatal("connection dialog is not closed after a successful load")
	}
	if !strings.Contains(page, "已连接到 CPA，可稍后重试") {
		t.Fatal("connection still fails when the models.dev catalog is unavailable")
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
		"data-tab=\"auth-quotas\"", "credit-manager/auth-quotas", "credit-manager/auth-quotas/refresh", "credit-manager/auth-quotas/warmup", "credit-manager/auth-quotas/warmup/settings", "auth-quota-warmup", "authQuotaWarmupStatus", "btnAuthWarmupSettings", "authWarmupSettingsModal", "btnAddAuthWarmupSchedule", "authWarmupScheduleList", "auth-warmup-task", "auth-warmup-settings-footer", "renderAuthWarmupSchedules", "openAuthWarmupSettings", "saveAuthWarmupSettings", "authQuotaIsFiveHour", "auth-quota-window-card", "auth-quota-bar", "function authQuotaPeriodBadge", "auth-quota-reload",
		"auth-quota-period-picker", "auth-quota-period-trigger", "auth-quota-period-menu", "auth-quota-period-option", "closeAuthQuotaPeriodMenus", "配额窗口", "authQuotaIsWeekly", "authQuotaIsCycleWindow", "function authQuotaPrimaryCycleWindows", "latestByBaseline", "authQuotaDisplayWindows", "const companions", "displayedCycles", "selectedPrimaryCycles", "authQuotaIsPartial(window)", "function authQuotaIsPartial", "return '不完整'", "authQuotaIsWeekly(window) || /quota|window/i.test(text)", "timeless", "function authQuotaWindowStartMs", "reset - duration * 1000", "authQuotaWindowCurrent", "6 * 60 * 60 * 1000", "includes('secondary')", "authQuotaCostForecast", "当前费用", "预估剩余", "预计可用", "authQuotaProviderFilter", "authQuotaNameFilter", "overflow-x:auto", "state.currentTab === 'auth-quotas'", "已刷新缓存", "同步上游额度", "auth-quota-status-banner", "btnAuthQuotaBannerRefresh", "需有本地用量观测", "authQuotaStatusBanner",
		"btnRefreshAuthQuotaPage", "刷新本页", "authQuotaPagination", "authQuotaPageSize", "credit-manager/auth-quotas?", "page_size",
		"authQuotaPlanName", "auth-quota-plan", "订阅类型",
		"auth-quota-concurrency", "最大并发", "credit-manager/auth-quotas/concurrency", "credit-manager/auth-quotas/concurrency/batch", "max_concurrent_requests", "active_requests", "auth-quota-active", "批量并发", "应用本页", "应用筛选", "筛选结果", "btnAuthQuotaBatchPage", "data-provider",
		"auth-codex-fill", "M6.469 8.776L16.512 23", "function authQuotaProviderIcon",
		"align-items:stretch", "flex-direction:column", "height:100%", "auth-quota-account-row", "function authQuotaRemainingLabel", "额度可用", "后刷新",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console auth quota view is missing %q", text)
		}
	}
	for _, removed := range []string{"auth-quota-windows", "本地归因", "平均 Token/请求", "预计剩余请求", "authQuotaWeekFilter", "authQuotaPeriodFilter", "额度区间", "btnLoadAuthQuotas", "刷新最多每 15 分钟查询一次", "请在卡片内单独刷新该账号"} {
		if strings.Contains(page, removed) {
			t.Fatalf("console auth quota view still exposes removed field %q", removed)
		}
	}
	if strings.Contains(page, "label.includes('周') || label.includes('週')") {
		t.Fatal("generic Chinese cycle labels are still misclassified as weekly")
	}
	for _, text := range []string{"label.includes('额度周')", "label.includes('額度週')", "label === '周'", "label === '週'"} {
		if !strings.Contains(page, text) {
			t.Fatalf("console does not recognize a weekly quota label without duration: %q", text)
		}
	}
	for _, removed := range []string{"auth-quota-cycle-selects", "data-auth-baseline", "selectedAuthQuotaWeeks", "authQuotaWeekGroups"} {
		if strings.Contains(page, removed) {
			t.Fatalf("console still renders a separate selector for each quota pool: %q", removed)
		}
	}
	if strings.Contains(page, "auth-quota-week-select") {
		t.Fatal("console still uses the native quota period select")
	}
	lookup := string(lookupPage().Body)
	for _, forbidden := range []string{"auth-quotas", "认证额度", "刷新本页", "authQuotaPagination", "auth-quota-plan"} {
		if strings.Contains(lookup, forbidden) {
			t.Fatalf("public lookup page exposes auth quota UI: %q", forbidden)
		}
	}
}

func TestConsoleAuthQuotaShowsCurrentShorterCycles(t *testing.T) {
	page := strings.ReplaceAll(string(consolePage().Body), "\r\n", "\n")
	for _, text := range []string{
		"function authQuotaPrimaryCycleWindows(windows, now)",
		"function authQuotaLatestByIdentity",
		"function authQuotaGroupedWindows",
		"function authQuotaProviderKind",
		"function authQuotaResetRelative",
		"auth-quota-group",
		"allCycles.filter(window => !primary.has(window)",
		"const viewingCurrent = !selected || selectedCycles.some",
		"fiveHour.concat(companions, displayedCycles)",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console does not render current shorter quota cycles: %q", text)
		}
	}
}

func TestConsoleAuthQuotaUsesStyledPeriodMenu(t *testing.T) {
	page := strings.ReplaceAll(string(consolePage().Body), "\r\n", "\n")
	for _, text := range []string{
		"role=\"listbox\"", "role=\"option\"", "aria-expanded",
		"auth-quota-period-option.is-partial::before",
		"@keyframes auth-quota-period-menu-in",
		".auth-quota-card:has(.auth-quota-period-picker.is-open)",
		"max-height:260px", "overscroll-behavior:contain",
		"function setAuthQuotaPeriodMenu", "function focusAuthQuotaPeriodTrigger",
		"event.key === 'ArrowDown'", "event.key === 'Home'",
		"document.addEventListener('keydown'",
		"event.key === 'Escape'",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console period menu is missing %q", text)
		}
	}
}

func TestAuthQuotaRefreshRoute(t *testing.T) {
	var found bool
	for _, route := range Routes() {
		if route.Method == "POST" && route.Path == "credit-manager/auth-quotas/refresh" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("auth quota refresh route is not registered")
	}
}

func TestAuthQuotaWarmupRoute(t *testing.T) {
	var warmup, getSettings, saveSettings bool
	for _, route := range Routes() {
		switch {
		case route.Method == "POST" && route.Path == "credit-manager/auth-quotas/warmup":
			warmup = true
		case route.Method == "GET" && route.Path == "credit-manager/auth-quotas/warmup/settings":
			getSettings = true
		case route.Method == "POST" && route.Path == "credit-manager/auth-quotas/warmup/settings":
			saveSettings = true
		}
	}
	if !warmup || !getSettings || !saveSettings {
		t.Fatalf("warmup routes = warmup:%t get:%t save:%t", warmup, getSettings, saveSettings)
	}
}

func TestConsoleAuthQuotaWarmupClearsBusyStateBeforeReload(t *testing.T) {
	page := string(consolePage().Body)
	start := strings.Index(page, "async function warmupAuthQuota")
	if start < 0 {
		t.Fatal("warmup lifecycle function is missing")
	}
	end := strings.Index(page[start:], "async function fetchAuthWarmupModels")
	if end < 0 {
		t.Fatal("warmup lifecycle boundary is missing")
	}
	warmup := page[start : start+end]
	clearAt := strings.Index(warmup, "delete state.authQuotaWarming[itemKey];")
	reloadAt := strings.Index(warmup, "await loadAuthQuotas();")
	if clearAt < 0 || reloadAt < 0 || clearAt > reloadAt {
		t.Fatalf("warmup busy state is not cleared before reload: clear=%d reload=%d", clearAt, reloadAt)
	}
}

func TestConsoleAuthQuotaWarmupStatusOmitsCrowdedNoise(t *testing.T) {
	page := string(consolePage().Body)
	start := strings.Index(page, "function authQuotaWarmupStatus")
	if start < 0 {
		t.Fatal("warmup status renderer is missing")
	}
	end := strings.Index(page[start:], "async function loadAuthQuotas")
	if end < 0 {
		t.Fatal("warmup status renderer boundary is missing")
	}
	body := page[start : start+end]
	for _, noise := range []string{"预热统计", " in / ", "input / "} {
		if strings.Contains(body, noise) {
			t.Fatalf("warmup status still shows crowded noise %q", noise)
		}
	}
	for _, keep := range []string{"auth-quota-warmup-main", "窗口已更新", "<time>"} {
		if !strings.Contains(body, keep) {
			t.Fatalf("warmup status is missing %q", keep)
		}
	}
}

func TestConsoleAuthQuotaWarmupSettingsControls(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"auth-warmup-settings-footer",
		"auth-warmup-task-actions",
		"auth-warmup-task-toggle",
		"auth-warmup-task-delete",
		"auth-warmup-task-frequency",
		"auth-warmup-task-weekdays",
		`class="auth-warmup-task-weekdays" multiple`,
		"authWarmupTimezoneOptions",
		"auth-warmup-task-timezone",
		".auth-warmup-task-grid .auth-warmup-task-weekday",
		"auth-warmup-task-datetime",
		"datetime-local",
		"buildCustomTimeInput",
		"custom-time-panel",
		"每天",
		"每周",
		"单次",
		`type="checkbox" role="switch"`,
		".auth-warmup-task-actions { flex:0 0 auto; align-items:center; gap:2px; height:30px;",
		".auth-warmup-task .icon-btn { display:grid; place-items:center; width:30px; height:30px;",
		".auth-warmup-task .icon-btn svg { width:16px; height:16px; }",
		".auth-warmup-settings-footer .btn { min-height:38px;",
		"stroke-width=\"1.85\"",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("warmup settings controls are missing %q", text)
		}
	}
}

func TestConsoleUsageRecentIncludesExecutor(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{"<th>执行器</th>", "u.executor_type || '—'", "'执行器': {"} {
		if !strings.Contains(page, text) {
			t.Fatalf("recent usage is missing executor column content: %q", text)
		}
	}
}

func TestConsoleAuthDisplayShortensCompatibleProviders(t *testing.T) {
	page := strings.ReplaceAll(string(consolePage().Body), "\r\n", "\n")
	for _, text := range []string{
		"function formatProviderDisplay",
		"openai-compatible-",
		"function formatAuthDisplay",
		"formatAuthDisplay(provider, account)",
		"authQuotaProviderName(provider)",
		"authQuotaProviderIsBrand",
		" is-custom",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console auth display is missing %q", text)
		}
	}
	if strings.Contains(page, "const display = [provider, account].filter(Boolean).join(' · ') || '未知账号';") {
		t.Fatal("usage auth cell still concatenates raw provider and account")
	}
	if strings.Contains(page, "<p class=\"auth-quota-provider\">'+esc(provider || t('未知提供商'))+'</p>") {
		t.Fatal("auth quota cards still show the raw provider id")
	}
}

func TestConsoleAuthQuotaRefreshReconcilesCurrentPage(t *testing.T) {
	page := string(consolePage().Body)
	start := strings.Index(page, "async function refreshAuthQuota")
	if start < 0 {
		t.Fatal("auth quota card refresh function is missing")
	}
	refresh := page[start:]
	end := strings.Index(refresh, "function renderAuthQuotaPagination")
	if end < 0 || !strings.Contains(refresh[:end], "await loadAuthQuotas();") {
		t.Fatal("auth quota card refresh does not reconcile the current page")
	}
	if !strings.Contains(page, "{ silent: true, reconcile: false }") {
		t.Fatal("bulk auth quota refresh does not defer page reconciliation")
	}
}

func TestConsoleAuthQuotaResetRebindsCurrentPeriod(t *testing.T) {
	page := strings.ReplaceAll(string(consolePage().Body), "\r\n", "\n")
	for _, text := range []string{
		"authQuotaCurrentWeeks: {}",
		"function selectedAuthQuotaWeek",
		"const current = weeks.filter(week => week.current)",
		"const currentKeys = current.map(week => week.key)",
		"const storedCurrent = state.authQuotaCurrentWeeks[itemKey]",
		"previousCurrent.includes(state.authQuotaWeeks[itemKey])",
		"!currentKeys.includes(state.authQuotaWeeks[itemKey])",
		"state.authQuotaCurrentWeeks[itemKey] = currentKeys",
		"state.authQuotaCurrentWeeks = {}",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console does not rebind the current quota week after reset: %q", text)
		}
	}
}

func TestAuthQuotaConcurrencyRoute(t *testing.T) {
	var found, batch bool
	for _, route := range Routes() {
		if route.Method != "POST" {
			continue
		}
		if route.Path == "credit-manager/auth-quotas/concurrency" {
			found = true
		}
		if route.Path == "credit-manager/auth-quotas/concurrency/batch" {
			batch = true
		}
	}
	if !found {
		t.Fatal("auth quota concurrency route is not registered")
	}
	if !batch {
		t.Fatal("auth quota concurrency batch route is not registered")
	}
}

func TestConsoleAuthQuotaBatchUsesCustomConfirmation(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		`id="authQuotaBatchModal"`,
		`id="authQuotaBatchTarget"`,
		`id="authQuotaBatchValue"`,
		`id="btnConfirmAuthQuotaBatch"`,
		"function openAuthQuotaBatchModal",
		"async function confirmAuthQuotaConcurrencyBatch",
		"openAuthQuotaBatchModal(scope, payload, count);",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console auth quota batch confirmation is missing %q", text)
		}
	}
	start := strings.Index(page, "async function saveAuthQuotaConcurrencyBatch")
	end := strings.Index(page, "async function refreshAuthQuota")
	if start < 0 || end < 0 || start >= end {
		t.Fatal("auth quota batch save function is missing")
	}
	if strings.Contains(page[start:end], "confirm(") {
		t.Fatal("auth quota batch save still uses a browser confirmation")
	}
}

func TestKeySpendResetRoute(t *testing.T) {
	var found bool
	for _, route := range Routes() {
		if route.Method == "POST" && route.Path == "credit-manager/keys/reset-spend" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("key spend reset route is not registered")
	}
}

func TestConsoleKeySpendResetUI(t *testing.T) {
	page := strings.ReplaceAll(string(consolePage().Body), "\r\n", "\n")
	for _, text := range []string{
		"credit-manager/keys/reset-spend",
		"function openResetSpendModal",
		"function syncKeySelectionControls",
		"payload.ids = state.resetSpendIDs",
		"data-select-all-keys",
		"data-select-key",
		"btnResetSelectedKeySpend",
		"重置已选额度",
		"function syncResetSpendScopeOrder",
		"resetSpendTotal",
		"data-reset-period=\"daily\"",
		"data-reset-period=\"weekly\"",
		"data-reset-period=\"monthly\"",
		"data-reset-period=\"total\"",
		"重置全部额度",
		"跟随账号重置",
		"同步重置时间",
		"自动清零已用",
		"keyModalPlanEnabled",
		"keyModalPlanCycle",
		"keyModalPlanAutoReset",
		"keyModalPlanAuth",
		"id=\"keyModalPlanAuthWrap\">跟随账号",
		"function planPayload",
		"credit-manager/plan-accounts",
		"重置额度",
		"仅重置所选周期的已用统计，历史账本与上限配置保留。",
		"总额度",
		"data-reset-spend",
		"btnResetManagedKeySpend",
		"keyPagination",
		"pricingPagination",
		"function loadKeys",
		"active_only: '1'",
		"function renderKeyPagination",
		"function renderPricingPagination",
		".usage-pagination .custom-control-panel",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console spend reset UI is missing %q", text)
		}
	}
	if !strings.Contains(page, "await reload();\n      flash(t('已重置')") {
		t.Fatal("reset success toast is cleared before the refreshed data finishes loading")
	}
}

func TestConsoleKeyFilterShowsLabelOnly(t *testing.T) {
	page := strings.ReplaceAll(string(consolePage().Body), "\r\n", "\n")
	for _, text := range []string{
		"function keyFilterDisplayLabel(key)",
		"input.value = keyFilterDisplayLabel(key)",
		"input.dataset.selectedKeyId = key.id",
		"resolveKeyFilter($('overviewKeyFilter'))",
		"resolveKeyFilter($('usageKeyFilter'))",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console key filter display is missing %q", text)
		}
	}
}

func TestConsoleManagementListSearch(t *testing.T) {
	page := strings.ReplaceAll(string(consolePage().Body), "\r\n", "\n")
	for _, text := range []string{
		`id="keyNameFilter" type="text"`,
		`placeholder="搜索标签"`,
		`data-clear-search="keyNameFilter"`,
		`class="list-count" id="keysCount"`,
		`id="pricingModelFilter" type="text"`,
		`placeholder="搜索模型 ID"`,
		`data-clear-search="pricingModelFilter"`,
		`class="list-count" id="modelCatalogCount"`,
		"keySearch: ''",
		"pricingSearch: ''",
		"params.set('q', query);",
		"$('modelCatalogCount').textContent = models.length.toLocaleString();",
		"function syncListSearchClearButton(inputID)",
		"function queueKeyNameSearch()",
		"function queuePricingModelSearch()",
		"state.keyPage = 1;",
		"state.pricingPage = 1;",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console management list search is missing %q", text)
		}
	}
	start := strings.Index(page, "function renderPricing(items)")
	if start < 0 {
		t.Fatal("pricing renderer is missing")
	}
	end := strings.Index(page[start:], "function renderPricingPagination")
	if end < 0 {
		t.Fatal("pricing renderer could not be bounded")
	}
	pricing := page[start : start+end]
	filterAt := strings.Index(pricing, "const filteredModels =")
	sliceAt := strings.Index(pricing, "const pageModels = filteredModels.slice")
	if filterAt < 0 || sliceAt < 0 || filterAt > sliceAt {
		t.Fatal("pricing search is not applied before pagination")
	}
}

func TestPricingEnabledRoute(t *testing.T) {
	var found bool
	for _, route := range Routes() {
		if route.Method == "POST" && route.Path == "credit-manager/pricing/enabled" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("pricing enabled route is not registered")
	}
}

func TestConsoleKeyEnabledSwitch(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		`id="keyModalEnabledWrap"`,
		`id="keyModalEnabled" type="checkbox" role="switch"`,
		"data-enable-key",
		"function toggleKeyEnabled",
		"key-switch-ui",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing key enable switch: %q", text)
		}
	}
	if strings.Contains(page, `<select id="keyModalEnabled">`) {
		t.Fatal("key modal still uses a select for enable/disable")
	}
}

func TestConsoleKeyExpiry(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		`id="keyModalExpiryField"`,
		`id="keyModalExpiresAt" type="datetime-local"`,
		`data-expiry-preset="never"`,
		`data-expiry-preset="7"`,
		`data-expiry-preset="30"`,
		`data-expiry-preset="90"`,
		`data-expiry-preset="custom"`,
		"function setKeyExpiryPreset",
		"function collectKeyExpiryPayload",
		"function keyExpiryMeta",
		"clear_expires_at",
		"expires_at",
		"已过期",
		`id="keyModalExpiresWrap"`,
		`data-expiry-preset="never" class="active">长期`,
		">指定</button>",
		"到期时间",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing key expiry controls: %q", text)
		}
	}
	lookup := string(lookupPage().Body)
	for _, text := range []string{
		`id="keyExpiry"`,
		"function formatLookupExpiry",
		"key.expires_at",
	} {
		if !strings.Contains(lookup, text) {
			t.Fatalf("lookup page is missing key expiry display: %q", text)
		}
	}
}

func TestConsoleUsesRepoLogo(t *testing.T) {
	for name, page := range map[string]string{"console": string(consolePage().Body), "lookup": string(lookupPage().Body)} {
		for _, text := range []string{`data-cm-logo="v2"`, `class="brand-logo" src="data:image/png;base64,`, `<link rel="icon" type="image/png"`} {
			if !strings.Contains(page, text) {
				t.Fatalf("%s page is missing logo marker: %q", name, text)
			}
		}
		if strings.Contains(page, `fill="#F7F5F0"`) {
			t.Fatalf("%s page still uses the old logo", name)
		}
	}
}

func TestConsoleModelTokenLimits(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		`id="keyModalTokenLimits"`,
		`id="keyModalTokenLimitsEnabled"`,
		`id="btnAddKeyTokenLimit"`,
		`id="keyModalTokenLimitSearch"`,
		`id="keyModalTokenLimitOptions"`,
		"function openTokenLimitModelSearch",
		"function collectModelTokenLimits",
		"function renderKeyTokenLimits",
		"set_model_token_limits",
		"model_token_limits",
		"未列入限制的模型",
		"data-unmatched-set=\"disabled\"",
		"unmatched_models_mode",
		"未添加模型",
		"全部模型（默认）",
		"btnKeyModalAllowAllModels",
		"periodField('total', '总', item.total)",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing model token limits: %q", text)
		}
	}
	lookup := string(lookupPage().Body)
	for _, text := range []string{
		`id="modelTokenLimitsSection"`,
		"function renderModelTokenLimits",
		"model_token_usage",
		"key.model_token_limits",
		"token-limit-card",
	} {
		if !strings.Contains(lookup, text) {
			t.Fatalf("lookup page is missing model token limits: %q", text)
		}
	}
}

func TestConsoleDiscoversAIProviderModels(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"function collectProviderKeyModels",
		"function addPrefixedModelID",
		"openai-compatibility",
		"gemini-api-key",
		"models?scope=available",
		"管理接口与 /v1/models 均未获取到模型",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing AI provider model discovery: %q", text)
		}
	}
	if strings.Contains(page, "Fallback only when management discovery returned nothing.") {
		t.Fatal("console still skips /v1/models when auth-files return models")
	}
}

func TestConsolePricingSpecialTiers(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"function modelsDevContextTiers",
		"function collectPriceTiers",
		"function fillPriceTiers",
		`id="priceTierSection"`,
		`id="priceContextTiers"`,
		`id="priceServiceTiers"`,
		"添加上下文档位",
		"特殊档位",
		"service_tier",
		"272000",
		"kind: 'context'",
		"kind: 'service'",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing special pricing tiers: %q", text)
		}
	}
}

func TestConsolePricingEnableDisable(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		`id="priceEnabledWrap"`,
		`id="priceEnabled" type="checkbox" role="switch"`,
		"data-enable-price",
		"function toggleModelPricingEnabled",
		"credit-manager/pricing/enabled",
		"function ruleEnabled",
		"function setModelPricingEnabled",
		"function modelIsDisabled",
		"function winningPricingRule",
		"function nextExactPriority",
		"'[^/]*'",
		"function comparePricingModels",
		"function pricingModelPriceAmount",
		"禁用后无法调用",
		"也不会出现在客户端模型列表中",
		"密钥管理",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing model enable/disable support: %q", text)
		}
	}
	if strings.Contains(page, `<select id="priceEnabled">`) {
		t.Fatal("price modal still uses a select for enable/disable")
	}
	if strings.Contains(page, "data-toggle-price") {
		t.Fatal("pricing table still uses enable/disable buttons")
	}
	if strings.Contains(page, "Key 管理") {
		t.Fatal("console still uses Key 管理 instead of 密钥管理")
	}
}

func TestLookupStatsIconsAndEmptyLayouts(t *testing.T) {
	page := string(lookupPage().Body)
	for _, text := range []string{
		"['activity',t('请求数')",
		"['layers','Token'",
		"['coin',t('费用')",
		"['bolt',t('在途请求')",
		"tr class=\"empty-row\"",
		"share-body:has(.chart-empty)",
		"chart(id,empty,null,moneyAxis?'coin':'chart')",
		"return chart('modelShare',t('暂无数据'),null,'pie')",
		`id="keyLabel"`,
		`class="heading-icon" aria-hidden="true"`,
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("lookup page is missing stats/empty layout fix: %q", text)
		}
	}
	if strings.Contains(page, "[t('请求数'),count(summary.request_count)+' '+t('请求'),'activity']") {
		t.Fatal("lookup stats still put icon names in the value slot")
	}
	if strings.Contains(page, "return rows.length?'<table><thead>") {
		t.Fatal("lookup recent table still drops headers when empty")
	}
	if strings.Contains(page, ".key-head h2::before") {
		t.Fatal("lookup key title still uses the caret-like accent bar")
	}
}

func TestLookupAvoidsRememberedKeyBlankFirstPaint(t *testing.T) {
	page := strings.ReplaceAll(string(lookupPage().Body), "\r\n", "\n")
	for _, text := range []string{
		`id="bootLoading"`,
		"html[data-lookup-authed]:not([data-lookup-ready]) .boot-loading { display:block; }",
		"html[data-lookup-authed] .login-gate { display:none; }",
		"credit-manager.lookup.token",
		"dataset.lookupReady='1'",
		"delete document.documentElement.dataset.lookupReady",
		"let token = '', data = null, modelMetric = 'requests', modelRankMetric = 'value', tokenTrendGrain = 'hour', costTrendGrain = 'hour', recentPage = 1, recentPageSize = 10, loadSeq = 0",
		"const seq = ++loadSeq",
		"if(seq !== loadSeq) return",
		`id="loadError"`,
		"loadError.textContent=message",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("lookup page is missing remembered-key first-paint guard: %q", text)
		}
	}
	boot := strings.Index(page, `id="bootLoading"`)
	dash := strings.Index(page, `id="dashboard"`)
	if boot < 0 || dash < 0 || boot > dash {
		t.Fatalf("boot loading must precede the dashboard (boot=%d dashboard=%d)", boot, dash)
	}
	// Stale concurrent filter/refresh responses must not clobber a newer paint
	// or re-enable the button while a later load is still in flight.
	load := page[strings.Index(page, "async function load(isNew=false, resetPage=false, recentOnly=false)"):]
	end := strings.Index(load, "\n  function syncPrefMenus()")
	if end < 0 {
		t.Fatal("load body not found")
	}
	load = load[:end]
	if strings.Count(load, "if(seq !== loadSeq) return") < 2 {
		t.Fatal("load must ignore stale success and error paths via loadSeq")
	}
	if !strings.Contains(load, "showLoginGate(!data)") {
		t.Fatal("invalid remembered key must clear the boot overlay via showLoginGate")
	}
	if strings.Contains(load, "dataset.lookupAuthed='1'") {
		t.Fatal("successful login must not flip data-lookup-authed mid-session (boot skeleton flicker)")
	}
	if !strings.Contains(load, "if(seq === loadSeq)") {
		t.Fatal("load finally must only settle the button for the latest seq")
	}
	// Refresh failures while the dashboard is visible must not rely solely on
	// #error inside the (hidden) login gate.
	if !strings.Contains(load, "isNew || !data") || !strings.Contains(load, "loadError") {
		t.Fatal("load must surface dashboard refresh errors on #loadError")
	}
}

func TestConsoleClearLocalDataResetsLoadedView(t *testing.T) {
	page := strings.ReplaceAll(string(consolePage().Body), "\r\n", "\n")
	start := strings.Index(page, "function clearLoadedSession()")
	if start < 0 {
		t.Fatal("clearLoadedSession is missing")
	}
	body := page[start:]
	end := strings.Index(body, "function renderOverviewEChart")
	if end < 0 {
		t.Fatal("clearLoadedSession body could not be bounded")
	}
	reset := body[:end]
	for _, text := range []string{
		"state.tabLoadSeq += 1;",
		"state.overview = null;",
		"state.keys = [];",
		"state.authQuotas = null;",
		"state.usageSummary = null;",
		"resetDataBoundFilters();",
		"renderDisconnectedOverview();",
		"renderDisconnectedTabStates();",
	} {
		if !strings.Contains(reset, text) {
			t.Fatalf("clearLoadedSession is missing %q", text)
		}
	}
	handlerStart := strings.Index(page, "$('btnClearToken').addEventListener('click'")
	if handlerStart < 0 {
		t.Fatal("clear local data button handler is missing")
	}
	handler := page[handlerStart:]
	handlerEnd := strings.Index(handler, "let authQuotaSearchTimer")
	if handlerEnd < 0 || !strings.Contains(handler[:handlerEnd], "clearLoadedSession();") {
		t.Fatal("clear local data does not reset the loaded workspace")
	}
	if !strings.Contains(handler[:handlerEnd], "$('mgmtToken').value = '';") {
		t.Fatal("clear local data does not clear the management token input")
	}
	disconnectedStart := strings.Index(page, "function renderDisconnectedTabStates()")
	if disconnectedStart < 0 {
		t.Fatal("renderDisconnectedTabStates is missing")
	}
	disconnected := page[disconnectedStart:]
	disconnectedEnd := strings.Index(disconnected, "function resetDataBoundFilters()")
	if disconnectedEnd < 0 || !strings.Contains(disconnected[:disconnectedEnd], "'authQuotaList'") {
		t.Fatal("disconnected tab states do not clear auth quota cards")
	}
	if !strings.Contains(page, "if (stats) stats.innerHTML = '';") {
		t.Fatal("disconnected overview does not clear overview stats")
	}
	bundleStart := strings.Index(page, "async function loadOverviewBundle()")
	if bundleStart < 0 {
		t.Fatal("loadOverviewBundle is missing")
	}
	bundle := page[bundleStart:]
	bundleEnd := strings.Index(bundle, "async function refreshActiveTab()")
	if bundleEnd < 0 || !strings.Contains(bundle[:bundleEnd], "if (seq !== state.tabLoadSeq) return null;") {
		t.Fatal("overview reload can still paint after local data is cleared")
	}
}
