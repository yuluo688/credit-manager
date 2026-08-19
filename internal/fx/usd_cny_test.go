package fx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseCurrencyAPI(t *testing.T) {
	rate, err := parseCurrencyAPI([]byte(`{"date":"2026-08-19","usd":{"cny":7.1842,"eur":0.86}}`))
	if err != nil {
		t.Fatal(err)
	}
	if rate != 7.1842 {
		t.Fatalf("rate = %v", rate)
	}
}

func TestParseERAPI(t *testing.T) {
	rate, err := parseERAPI([]byte(`{"result":"success","rates":{"CNY":7.2011}}`))
	if err != nil {
		t.Fatal(err)
	}
	if rate != 7.2011 {
		t.Fatalf("rate = %v", rate)
	}
}

func TestParseFloatRatesString(t *testing.T) {
	rate, err := parseFloatRates([]byte(`{"cny":{"rate":"7.1933"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if rate != 7.1933 {
		t.Fatalf("rate = %v", rate)
	}
}

func TestValidRejectsOutliers(t *testing.T) {
	if Valid(0) || Valid(1.2) || Valid(20) {
		t.Fatal("outlier rates should be rejected")
	}
	if !Valid(7.2) {
		t.Fatal("typical rate should be accepted")
	}
}

func TestClientCachesAndFallsBack(t *testing.T) {
	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits > 1 {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"usd": map[string]float64{"cny": 7.1666}})
	}))
	t.Cleanup(server.Close)

	now := time.Date(2026, 8, 19, 6, 0, 0, 0, time.UTC)
	client := &Client{
		HTTP: server.Client(),
		Sources: []source{{
			Name:  "test",
			URL:   server.URL,
			Parse: parseCurrencyAPI,
		}},
		CacheTTL: time.Hour,
		Now:      func() time.Time { return now },
	}

	first, err := client.Get(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if first.USDToCNY != 7.1666 || first.Cached || first.Source != "test" {
		t.Fatalf("first = %+v", first)
	}

	second, err := client.Get(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Cached || second.USDToCNY != 7.1666 || hits != 1 {
		t.Fatalf("cached = %+v hits=%d", second, hits)
	}

	stale, err := client.Get(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !stale.Cached || stale.USDToCNY != 7.1666 || hits != 2 {
		t.Fatalf("stale fallback = %+v hits=%d", stale, hits)
	}
}

func TestClientRejectsInvalidRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"usd": map[string]float64{"cny": 1.01}})
	}))
	t.Cleanup(server.Close)
	client := &Client{
		HTTP: server.Client(),
		Sources: []source{{
			Name:  "bad",
			URL:   server.URL,
			Parse: parseCurrencyAPI,
		}},
	}
	rate, err := client.Get(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if rate.USDToCNY != DefaultUSDToCNY || rate.Source != "default" {
		t.Fatalf("rate = %+v, want default %.1f", rate, DefaultUSDToCNY)
	}
}

func TestClientFallsBackToDefaultWhenFetchFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)
	client := &Client{
		HTTP: server.Client(),
		Sources: []source{{
			Name:  "down",
			URL:   server.URL,
			Parse: parseCurrencyAPI,
		}},
	}
	rate, err := client.Get(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if rate.USDToCNY != DefaultUSDToCNY || rate.Source != "default" || !rate.Cached {
		t.Fatalf("rate = %+v", rate)
	}
}
