package modelsdev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func sampleCatalog(provider, model string, input float64) []byte {
	payload := map[string]any{
		provider: map[string]any{
			"id": provider,
			"models": map[string]any{
				model: map[string]any{
					"id": model,
					"cost": map[string]any{
						"input":  input,
						"output": input * 3,
					},
				},
			},
		},
	}
	raw, _ := json.Marshal(payload)
	return raw
}

func TestValidCatalog(t *testing.T) {
	if validCatalog([]byte(`{}`)) || validCatalog([]byte(`[]`)) || validCatalog([]byte(`{"openai":{}}`)) {
		t.Fatal("invalid catalogs should be rejected")
	}
	if !validCatalog(sampleCatalog("openai", "gpt-4", 2.5)) {
		t.Fatal("provider catalog should be accepted")
	}
}

func TestClientCachesAndFallsBack(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n > 1 {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		_, _ = w.Write(sampleCatalog("openai", "gpt-4", 2.5))
	}))
	t.Cleanup(server.Close)

	now := time.Date(2026, 8, 20, 6, 0, 0, 0, time.UTC)
	client := &Client{
		HTTP: server.Client(),
		Sources: []source{{
			Name: "test",
			URL:  server.URL,
		}},
		CacheTTL: time.Hour,
		Now:      func() time.Time { return now },
	}

	first := client.Get(context.Background(), false)
	if first.Error != "" || first.Cached || first.Source != "test" || !validCatalog(first.Providers) {
		t.Fatalf("first = %+v", first)
	}
	if hits.Load() != 1 {
		t.Fatalf("hits = %d", hits.Load())
	}

	second := client.Get(context.Background(), false)
	if !second.Cached || second.Source != "test" || hits.Load() != 1 {
		t.Fatalf("cached = %+v hits=%d", second, hits.Load())
	}

	stale := client.Get(context.Background(), true)
	if !stale.Cached || stale.Source != "test" || hits.Load() != 2 || stale.Error == "" {
		t.Fatalf("stale fallback = %+v hits=%d", stale, hits.Load())
	}
}

func TestClientUsesDiskCacheWhenRemoteFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "models-dev-api.json")
	if err := os.WriteFile(path, mustJSON(diskCache{
		Source:    "disk",
		FetchedAt: time.Date(2026, 8, 19, 0, 0, 0, 0, time.UTC),
		Catalog:   sampleCatalog("anthropic", "claude-3", 3),
	}), 0o600); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)

	client := &Client{
		HTTP: server.Client(),
		Sources: []source{{
			Name: "down",
			URL:  server.URL,
		}},
		CacheFile: path,
	}
	got := client.Get(context.Background(), false)
	if !got.Cached || got.Source != "disk" || !validCatalog(got.Providers) {
		t.Fatalf("disk cache = %+v", got)
	}
}

func TestClientReturnsEmptyWhenUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)
	client := &Client{
		HTTP: server.Client(),
		Sources: []source{{
			Name: "down",
			URL:  server.URL,
		}},
	}
	got := client.Get(context.Background(), false)
	if got.Source != "unavailable" || got.Error == "" || string(got.Providers) != "{}" {
		t.Fatalf("unavailable = %+v", got)
	}
}

func TestClientPrefersFirstSuccessfulSource(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(sampleCatalog("google", "gemini", 0.1))
	}))
	t.Cleanup(good.Close)
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	t.Cleanup(bad.Close)

	client := &Client{
		HTTP: good.Client(),
		Sources: []source{
			{Name: "bad", URL: bad.URL},
			{Name: "good", URL: good.URL},
		},
	}
	got := client.Get(context.Background(), false)
	if got.Source != "good" || got.Error != "" || !validCatalog(got.Providers) {
		t.Fatalf("got = %+v", got)
	}
}

func mustJSON(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}
