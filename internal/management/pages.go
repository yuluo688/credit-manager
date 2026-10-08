package management

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

//go:embed web/*
var pageFiles embed.FS

// echartsAssetPath is the vendored ECharts build (Apache-2.0, see
// web/vendor/echarts-LICENSE). It used to load from cdn.jsdelivr.net as a
// parser-blocking <head> script, which kept the console iframe blank until the
// CDN answered; it is now served by the plugin with long-lived caching. The
// version is part of the path, so the immutable cache is safe across upgrades.
const echartsAssetPath = "vendor/echarts-5.6.0.min.js"

var (
	consolePageBody []byte
	lookupPageBody  []byte

	consolePageGzip = &lazyGzip{}
	lookupPageGzip  = &lazyGzip{}
	echartsAsset    = &staticAsset{}
)

func init() {
	consolePageBody = assemblePage("console")
	lookupPageBody = assemblePage("lookup")
	consolePageGzip.raw = consolePageBody
	lookupPageGzip.raw = lookupPageBody
	echartsAsset.body.raw = mustPageFile("web/" + echartsAssetPath)
	echartsAsset.contentType = "application/javascript; charset=utf-8"
	echartsAsset.etag = contentETag(echartsAsset.body.raw)
}

func assemblePage(name string) []byte {
	html := mustPageFile("web/" + name + ".html")
	css := mustPageFile("web/" + name + ".css")
	js := mustPageFile("web/" + name + ".js")
	cssTag := []byte(`<link rel="stylesheet" href="./` + name + `.css">`)
	jsTag := []byte(`<script src="./` + name + `.js" defer></script>`)
	if n := bytes.Count(html, cssTag); n != 1 {
		panic(fmt.Sprintf("web/%s.html: expected 1 stylesheet link, got %d", name, n))
	}
	if n := bytes.Count(html, jsTag); n != 1 {
		panic(fmt.Sprintf("web/%s.html: expected 1 script tag, got %d", name, n))
	}
	html = bytes.Replace(html, cssTag, concatPage([]byte("<style>"), css, []byte("</style>")), 1)
	html = bytes.Replace(html, jsTag, concatPage([]byte("<script>"), js, []byte("</script>")), 1)
	return html
}

func concatPage(parts ...[]byte) []byte {
	n := 0
	for _, part := range parts {
		n += len(part)
	}
	out := make([]byte, 0, n)
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

func mustPageFile(name string) []byte {
	raw, err := pageFiles.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return raw
}

func htmlPageHeaders(connectSrcSelf bool) http.Header {
	headers := http.Header{
		"Content-Type":  []string{"text/html; charset=utf-8"},
		"Cache-Control": []string{"no-store"},
	}
	if connectSrcSelf {
		headers.Set("Content-Security-Policy", "connect-src 'self'")
	}
	return headers
}

func consolePage() pluginapi.ManagementResponse {
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    htmlPageHeaders(false),
		Body:       consolePageBody,
	}
}

func lookupPage() pluginapi.ManagementResponse {
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    htmlPageHeaders(true),
		Body:       lookupPageBody,
	}
}

// lazyGzip compresses an immutable embedded body once, on first use.
type lazyGzip struct {
	raw  []byte
	once sync.Once
	gz   []byte
}

func (l *lazyGzip) gzipped() []byte {
	l.once.Do(func() {
		var buf bytes.Buffer
		w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
		if err != nil {
			return
		}
		if _, err := w.Write(l.raw); err != nil {
			return
		}
		if err := w.Close(); err != nil {
			return
		}
		if buf.Len() < len(l.raw) {
			l.gz = buf.Bytes()
		}
	})
	return l.gz
}

// staticAsset is a versioned, immutable file served from a resource route.
type staticAsset struct {
	body        lazyGzip
	contentType string
	etag        string
}

func contentETag(body []byte) string {
	sum := sha256.Sum256(body)
	return `"` + hex.EncodeToString(sum[:12]) + `"`
}

func staticAssetResponse(reqHeaders http.Header, asset *staticAsset) pluginapi.ManagementResponse {
	headers := http.Header{
		"Content-Type":           []string{asset.contentType},
		"Cache-Control":          []string{"public, max-age=31536000, immutable"},
		"Etag":                   []string{asset.etag},
		"X-Content-Type-Options": []string{"nosniff"},
	}
	if etagMatches(requestHeader(reqHeaders, "If-None-Match"), asset.etag) {
		return pluginapi.ManagementResponse{StatusCode: http.StatusNotModified, Headers: headers}
	}
	return withGzip(reqHeaders, pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       asset.body.raw,
	}, &asset.body)
}

// withGzip serves the precompressed body when the client accepts gzip. The
// host relays plugin responses as-is and does not compress them itself.
func withGzip(reqHeaders http.Header, response pluginapi.ManagementResponse, body *lazyGzip) pluginapi.ManagementResponse {
	headers := response.Headers.Clone()
	if headers == nil {
		headers = http.Header{}
	}
	headers.Set("Vary", "Accept-Encoding")
	response.Headers = headers
	if !acceptsGzip(requestHeader(reqHeaders, "Accept-Encoding")) {
		return response
	}
	gz := body.gzipped()
	if len(gz) == 0 {
		return response
	}
	headers.Set("Content-Encoding", "gzip")
	response.Body = gz
	return response
}

func requestHeader(headers http.Header, name string) string {
	if headers == nil {
		return ""
	}
	if value := headers.Get(name); value != "" {
		return value
	}
	for key, values := range headers {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func acceptsGzip(header string) bool {
	for _, part := range strings.Split(header, ",") {
		name, params, _ := strings.Cut(part, ";")
		name = strings.TrimSpace(name)
		if !strings.EqualFold(name, "gzip") && name != "*" {
			continue
		}
		q := strings.ReplaceAll(strings.TrimSpace(params), " ", "")
		if q == "q=0" || q == "q=0.0" || q == "q=0.00" || q == "q=0.000" {
			return false
		}
		return true
	}
	return false
}

func etagMatches(header, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		candidate = strings.TrimPrefix(candidate, "W/")
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}
