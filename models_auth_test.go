package main

import (
	"net/http"
	"testing"
)

func TestPublicModelDirectoryRequest(t *testing.T) {
	for _, test := range []struct {
		method string
		path   string
		want   bool
	}{
		{"GET", "/v1/models", true},
		{"get", "/v1/models/", true},
		{"POST", "/v1/models", false},
		{"GET", "/v1/chat/completions", false},
		{"GET", "/v0/management/config", false},
	} {
		if got := publicModelDirectoryRequest(test.method, test.path); got != test.want {
			t.Errorf("publicModelDirectoryRequest(%q, %q) = %t, want %t", test.method, test.path, got, test.want)
		}
	}
}

func TestAllowedAuthQuotaRequest(t *testing.T) {
	for _, test := range []struct {
		method, url string
		want        bool
	}{
		{http.MethodGet, "https://chatgpt.com/backend-api/wham/usage", true},
		{http.MethodGet, "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits", true},
		{http.MethodGet, "https://cli-chat-proxy.grok.com/v1/billing?format=credits", true},
		{http.MethodPost, "https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary", true},
		{http.MethodGet, "https://chatgpt.com/backend-api/wham/usage", true},
		{http.MethodPost, "https://chatgpt.com/backend-api/wham/usage", false},
		{http.MethodGet, "http://chatgpt.com/backend-api/wham/usage", false},
		{http.MethodGet, "https://evil.example/backend-api/wham/usage", false},
		{http.MethodGet, "https://chatgpt.com.evil.example/backend-api/wham/usage", false},
		{http.MethodGet, "https://api.x.ai/v1/billing", true},
	} {
		if got := allowedAuthQuotaRequest(test.method, test.url); got != test.want {
			t.Errorf("allowedAuthQuotaRequest(%q, %q) = %t, want %t", test.method, test.url, got, test.want)
		}
	}
}
