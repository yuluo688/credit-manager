package management

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/yuluo688/credit-manager/internal/service"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

func resourceGet(t *testing.T, path string, headers http.Header) pluginapi.ManagementResponse {
	t.Helper()
	response, err := Handle(context.Background(), pluginapi.ManagementRequest{
		Method:  http.MethodGet,
		Path:    "/v0/resource/plugins/" + service.PluginID + "/" + path,
		Headers: headers,
	})
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return response
}

func gunzip(t *testing.T, body []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("gunzip: %v", err)
	}
	return raw
}

func TestConsolePageShipsInlineBootSkeleton(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		`<div class="boot-skeleton" id="bootSkeleton" aria-hidden="true">`,
		"html.cm-booting .boot-skeleton { display:block; }",
		"html.cm-booting .workspace { display:none !important; }",
		"root.classList.add('cm-booting')",
		"window.creditManagerBoot = {",
		"window.creditManagerBoot.ready(",
		"document.addEventListener('DOMContentLoaded', function () { if (!ready) fail('script'); });",
		"function showDataSkeletons",
		"function settleDataSkeletons",
		"settleDataSkeletons(e.message);",
		"'数据加载失败': {",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing boot skeleton support: %q", text)
		}
	}
	skeleton := strings.Index(page, `id="bootSkeleton"`)
	workspace := strings.Index(page, `<main class="workspace">`)
	if skeleton < 0 || workspace < 0 || skeleton > workspace {
		t.Fatalf("boot skeleton must precede the workspace markup (skeleton=%d workspace=%d)", skeleton, workspace)
	}
	// The theme shim must run before the stylesheet so dark mode never flashes white.
	if shim, style := strings.Index(page, "cli-proxy-theme"), strings.Index(page, "<style>"); shim < 0 || shim > style {
		t.Fatalf("theme boot shim must precede the inline stylesheet (shim=%d style=%d)", shim, style)
	}
}

func TestPagesHaveNoRenderBlockingCDNScripts(t *testing.T) {
	pages := map[string]string{"console": string(consolePage().Body), "lookup": string(lookupPage().Body)}
	for name, page := range pages {
		for _, text := range []string{"cdn.jsdelivr.net/npm/echarts", `<script src="http`, `<link rel="stylesheet" href="http`} {
			if strings.Contains(page, text) {
				t.Fatalf("%s page still loads an external render-blocking resource: %q", name, text)
			}
		}
		tag := `<script src="./` + echartsAssetPath + `?v=` + assetVersion(mustPageFile("web/"+echartsAssetPath)) + `"></script>`
		at := strings.Index(page, tag)
		if at < 0 {
			t.Fatalf("%s page does not load the vendored ECharts build", name)
		}
		if head := strings.Index(page, "</head>"); at < head {
			t.Fatalf("%s page loads ECharts in <head>; it must not block first paint", name)
		}
	}
}

func TestEchartsVendorAssetRoute(t *testing.T) {
	var found bool
	for _, resource := range Resources() {
		if resource.Path != "/"+echartsAssetPath {
			continue
		}
		found = true
		if resource.Menu != "" {
			t.Fatalf("echarts asset menu = %q, want no sidebar entry", resource.Menu)
		}
	}
	if !found {
		t.Fatal("echarts asset resource is not registered")
	}

	response := resourceGet(t, echartsAssetPath, nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if got := response.Headers.Get("Content-Type"); !strings.HasPrefix(got, "application/javascript") {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := response.Headers.Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if response.Headers.Get("Content-Encoding") != "" {
		t.Fatal("asset must not be compressed without Accept-Encoding")
	}
	if !bytes.Contains(response.Body, []byte("Licensed to the Apache Software Foundation")) || len(response.Body) < 500_000 {
		t.Fatalf("unexpected echarts body (%d bytes)", len(response.Body))
	}
	etag := response.Headers.Get("ETag")
	if etag == "" {
		t.Fatal("asset response has no ETag")
	}

	notModified := resourceGet(t, echartsAssetPath, http.Header{"If-None-Match": []string{etag}})
	if notModified.StatusCode != http.StatusNotModified || len(notModified.Body) != 0 {
		t.Fatalf("If-None-Match status = %d body = %d, want 304 with empty body", notModified.StatusCode, len(notModified.Body))
	}

	compressed := resourceGet(t, echartsAssetPath, http.Header{"Accept-Encoding": []string{"gzip, deflate, br"}})
	if compressed.Headers.Get("Content-Encoding") != "gzip" || compressed.Headers.Get("Vary") != "Accept-Encoding" {
		t.Fatalf("gzip headers = %v", compressed.Headers)
	}
	if !bytes.Equal(gunzip(t, compressed.Body), response.Body) {
		t.Fatal("gzip body does not round-trip to the original asset")
	}
}

func TestPagesNegotiateGzip(t *testing.T) {
	for path, raw := range map[string][]byte{"console": consolePageBody, "lookup": lookupPageBody} {
		plain := resourceGet(t, path, nil)
		if !bytes.Equal(plain.Body, raw) || plain.Headers.Get("Content-Encoding") != "" {
			t.Fatalf("%s: plain response changed", path)
		}
		if plain.Headers.Get("Cache-Control") != "no-store" || plain.Headers.Get("Vary") != "Accept-Encoding" {
			t.Fatalf("%s: headers = %v", path, plain.Headers)
		}
		compressed := resourceGet(t, path, http.Header{"accept-encoding": []string{"gzip"}})
		if compressed.Headers.Get("Content-Encoding") != "gzip" || compressed.Headers.Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: gzip headers = %v", path, compressed.Headers)
		}
		if len(compressed.Body) >= len(raw) || !bytes.Equal(gunzip(t, compressed.Body), raw) {
			t.Fatalf("%s: gzip body invalid (%d >= %d?)", path, len(compressed.Body), len(raw))
		}
	}
	if lookupPage().Headers.Get("Content-Encoding") != "" || consolePage().Headers.Get("Vary") != "" {
		t.Fatal("withGzip must not mutate the shared page headers")
	}
}

func TestAcceptsGzip(t *testing.T) {
	for header, want := range map[string]bool{
		"":                    false,
		"br":                  false,
		"gzip":                true,
		"GZIP":                true,
		"deflate, gzip;q=0.8": true,
		"gzip;q=0":            false,
		"gzip; q=0.000, br":   false,
		"*":                   true,
		"identity":            false,
	} {
		if got := acceptsGzip(header); got != want {
			t.Fatalf("acceptsGzip(%q) = %t, want %t", header, got, want)
		}
	}
}

func TestConsoleAuthQuotasPaintWithTheirTab(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"['authQuotaList', 'authQuota']",
		"function authQuotaSkeletonCards",
		"auth-quota-card auth-quota-card-skeleton data-skeleton",
		".auth-quota-card.auth-quota-card-skeleton {",
		"function showAuthQuotaSkeleton",
		"function settleAuthQuotaSkeleton",
		"function prefetchAuthQuotas",
		"function paintAuthQuotasFromCache",
		"await loadAuthQuotas({ reuseInflight: true });",
		"if (reuseInflight && pending && pending.query === query",
		"function emptyState(message)",
		"当前没有可用的认证额度数据",
		"if (state.currentTab === 'auth-quotas') renderAuthQuotas(result);",
		"if (epoch !== state.authQuotaEpoch || query !== authQuotaQuery()) return;",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing auth quota early paint support: %q", text)
		}
	}
	// The list must be filled (cached cards or skeleton) before the pane is
	// un-hidden, so both land in the same frame.
	setTab := page[strings.Index(page, "function setTab(name)"):]
	paint := strings.Index(setTab, "paintAuthQuotasFromCache()")
	toggle := strings.Index(setTab, "pane.classList.toggle('hidden'")
	if paint < 0 || toggle < 0 || paint > toggle {
		t.Fatalf("setTab must paint auth quotas before showing the pane (paint=%d toggle=%d)", paint, toggle)
	}
	// The prefetch runs in parallel with, not after, the main reload chain.
	reload := page[strings.Index(page, "async function reloadWithModelCatalog()"):]
	prefetch := strings.Index(reload, "prefetchAuthQuotas();")
	await := strings.Index(reload, "await reload();")
	if prefetch < 0 || await < 0 || prefetch > await {
		t.Fatalf("auth quotas must be prefetched before awaiting the main reload (prefetch=%d await=%d)", prefetch, await)
	}
	load := page[strings.Index(page, "async function loadAuthQuotas(options)"):]
	end := strings.Index(load, "\n  function replaceAuthQuotaItem")
	if end < 0 {
		t.Fatal("loadAuthQuotas body not found")
	}
	load = load[:end]
	if strings.Contains(load, "if (seq !== state.tabLoadSeq) return") {
		t.Fatal("loadAuthQuotas must not drop auth-quota paint on tabLoadSeq mismatch during initial reload")
	}
	if strings.Contains(load, "const emptyState") {
		t.Fatal("emptyState must be a shared helper, not local to loadAuthQuotas")
	}
}

func TestConsoleOtherTabsAvoidFirstPaintRaces(t *testing.T) {
	page := strings.ReplaceAll(string(consolePage().Body), "\r\n", "\n")

	// emptyState must stay shared — the auth-quotas blank list was caused by a
	// local helper that other renderers could not see.
	if strings.Count(page, "function emptyState(message)") != 1 {
		t.Fatal("expected exactly one shared emptyState helper")
	}
	for _, name := range []string{"renderUsage", "renderAuthQuotas", "renderKeys", "renderPricing", "renderOverview"} {
		if !strings.Contains(page, "function "+name+"(") {
			t.Fatalf("missing renderer %s", name)
		}
	}
	usage := page[strings.Index(page, "function renderUsage("):]
	usage = usage[:strings.Index(usage, "\n  function renderUsagePagination")]
	if strings.Contains(usage, "const emptyState") || strings.Contains(usage, "function emptyState") {
		t.Fatal("renderUsage must use the shared emptyState helper")
	}
	if !strings.Contains(usage, "emptyState('当前筛选条件下暂无按密钥汇总')") {
		t.Fatal("renderUsage empty states are missing")
	}

	// Auth quota filter/paging must invalidate via authQuotaEpoch, not tabLoadSeq,
	// or a concurrent overview/usage/pricing load loses its first paint.
	for _, marker := range []string{
		"$('authQuotaProviderFilter').addEventListener('change'",
		"$('authQuotaNameFilter').addEventListener('input'",
		"$('authQuotaPageSize').addEventListener('change'",
		"$('btnAuthQuotaPrev').addEventListener('click'",
		"$('btnAuthQuotaNext').addEventListener('click'",
	} {
		start := strings.Index(page, marker)
		if start < 0 {
			t.Fatalf("missing handler %q", marker)
		}
		chunk := page[start : start+320]
		if strings.Contains(chunk, "state.tabLoadSeq += 1") {
			t.Fatalf("auth quota UI must not bump tabLoadSeq (cancels other tabs): %q", marker)
		}
		if !strings.Contains(chunk, "state.authQuotaEpoch += 1") {
			t.Fatalf("auth quota UI must bump authQuotaEpoch: %q", marker)
		}
	}

	// Keys search/paging uses a dedicated seq so typing in the keys box cannot
	// abort the overview/usage first paint that shares tabLoadSeq.
	if !strings.Contains(page, "keyLoadSeq: 0,") {
		t.Fatal("keyLoadSeq state is missing")
	}
	loadKeys := page[strings.Index(page, "async function loadKeys()"):]
	loadKeys = loadKeys[:strings.Index(loadKeys, "\n  async function toggleKeyEnabled")]
	if strings.Contains(loadKeys, "state.tabLoadSeq") {
		t.Fatal("loadKeys must not gate paint on tabLoadSeq")
	}
	if !strings.Contains(loadKeys, "const seq = state.keyLoadSeq") || !strings.Contains(loadKeys, "if (seq !== state.keyLoadSeq) return") {
		t.Fatal("loadKeys must gate paint on keyLoadSeq")
	}
	queue := page[strings.Index(page, "function queueKeyNameSearch()"):]
	queue = queue[:strings.Index(queue, "\n  function queuePricingModelSearch")]
	if strings.Contains(queue, "state.tabLoadSeq += 1") {
		t.Fatal("key search must not bump tabLoadSeq")
	}
	if !strings.Contains(queue, "state.keyLoadSeq += 1") {
		t.Fatal("key search must bump keyLoadSeq")
	}

	// Switching to the keys tab must call the paginated keys API, not only the
	// overview dump (search/paging otherwise drift or look empty after filter).
	refresh := page[strings.Index(page, "async function refreshActiveTab()"):]
	refresh = refresh[:strings.Index(refresh, "\n  function overviewRangeDates")]
	keysBranch := strings.Index(refresh, "if (tab === 'keys')")
	loadKeysCall := strings.Index(refresh, "await loadKeys()")
	if keysBranch < 0 || loadKeysCall < 0 || loadKeysCall < keysBranch {
		t.Fatal("refreshActiveTab must await loadKeys when the keys tab is active")
	}
}

func TestConsoleAuthQuotaCardRefreshUsesEpoch(t *testing.T) {
	page := strings.ReplaceAll(string(consolePage().Body), "\r\n", "\n")

	refreshStart := strings.Index(page, "async function refreshAuthQuota(")
	refreshEnd := strings.Index(page, "async function warmupAuthQuota(")
	if refreshStart < 0 || refreshEnd < 0 || refreshStart >= refreshEnd {
		t.Fatal("refreshAuthQuota body not found")
	}
	refresh := page[refreshStart:refreshEnd]
	if strings.Contains(refresh, "state.tabLoadSeq") {
		t.Fatal("refreshAuthQuota must not gate on tabLoadSeq (cancels other tabs / drops card updates)")
	}
	if !strings.Contains(refresh, "const epoch = state.authQuotaEpoch") || !strings.Contains(refresh, "epoch !== state.authQuotaEpoch") {
		t.Fatal("refreshAuthQuota must gate on authQuotaEpoch")
	}

	warmupStart := refreshEnd
	warmupEnd := strings.Index(page[warmupStart:], "async function fetchAuthWarmupModels(")
	if warmupEnd < 0 {
		t.Fatal("warmupAuthQuota body not found")
	}
	warmup := page[warmupStart : warmupStart+warmupEnd]
	if strings.Contains(warmup, "state.tabLoadSeq") {
		t.Fatal("warmupAuthQuota must not gate on tabLoadSeq")
	}
	if !strings.Contains(warmup, "const epoch = state.authQuotaEpoch") {
		t.Fatal("warmupAuthQuota must gate on authQuotaEpoch")
	}

	visibleStart := strings.Index(page, "async function refreshVisibleAuthQuotas(")
	visibleEnd := strings.Index(page[visibleStart:], "async function reload(")
	if visibleStart < 0 || visibleEnd < 0 {
		t.Fatal("refreshVisibleAuthQuotas body not found")
	}
	visible := page[visibleStart : visibleStart+visibleEnd]
	if strings.Contains(visible, "++state.tabLoadSeq") || strings.Contains(visible, "state.tabLoadSeq += 1") {
		t.Fatal("refreshVisibleAuthQuotas must not bump tabLoadSeq")
	}
	if !strings.Contains(visible, "const epoch = state.authQuotaEpoch") {
		t.Fatal("refreshVisibleAuthQuotas must capture authQuotaEpoch")
	}

	if !strings.Contains(page, "modelCatalogLoaded: false") {
		t.Fatal("modelCatalogLoaded state is missing")
	}
	refreshTab := page[strings.Index(page, "async function refreshActiveTab()"):]
	refreshTab = refreshTab[:strings.Index(refreshTab, "\n  function overviewRangeDates")]
	if !strings.Contains(refreshTab, "if (state.modelCatalogLoaded)") {
		t.Fatal("pricing tab must reuse a loaded model catalog")
	}
	if !strings.Contains(page, "min-height:299px") {
		t.Fatal("auth quota cards must keep a min-height matching the skeleton")
	}
}

func TestConsoleStatSkeletonKeepsCardPadding(t *testing.T) {
	page := string(consolePage().Body)
	for _, want := range []string{
		".stat.stat-skeleton.data-skeleton",
		"min-height:78px",
		"align-self:start",
		`[0, 1, 2, 3, 4].map(() => '<div class="stat stat-skeleton data-skeleton"`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("console page missing stat skeleton layout fix: %q", want)
		}
	}
}

func TestConsolePricingListUsesUniformMetricGrid(t *testing.T) {
	page := string(consolePage().Body)
	for _, want := range []string{
		"function renderModelsDevPriceGrid",
		"function renderCurrentRuleGrid",
		".price-metrics",
		"grid-template-columns:repeat(4, minmax(0, 1fr))",
		"参考价",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("console page missing pricing list clarity fix: %q", want)
		}
	}
}
