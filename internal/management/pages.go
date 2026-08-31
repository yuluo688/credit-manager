package management

import (
	"bytes"
	"embed"
	"fmt"
	"net/http"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

//go:embed web/*
var pageFiles embed.FS

var (
	consolePageBody []byte
	lookupPageBody  []byte
)

func init() {
	consolePageBody = assemblePage("console")
	lookupPageBody = assemblePage("lookup")
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

func htmlPageHeaders() http.Header {
	return http.Header{
		"Content-Type":            []string{"text/html; charset=utf-8"},
		"Cache-Control":           []string{"no-store"},
		"Content-Security-Policy": []string{"connect-src 'self'"},
	}
}

func consolePage() pluginapi.ManagementResponse {
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    htmlPageHeaders(),
		Body:       consolePageBody,
	}
}

func lookupPage() pluginapi.ManagementResponse {
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    htmlPageHeaders(),
		Body:       lookupPageBody,
	}
}
