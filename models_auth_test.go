package main

import "testing"

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
