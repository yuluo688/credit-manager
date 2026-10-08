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

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
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
		tag := `<script src="./` + echartsAssetPath + `"></script>`
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
