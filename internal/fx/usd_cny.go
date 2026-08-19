package fx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultUSDToCNY = 7.2
	cacheTTL        = 30 * time.Minute
	minUSDToCNY     = 4.0
	maxUSDToCNY     = 12.0
	requestTimeout  = 8 * time.Second
)

// Rate is a USD→CNY quote used only for display conversion.
type Rate struct {
	USDToCNY  float64   `json:"usd_to_cny"`
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetched_at"`
	Cached    bool      `json:"cached"`
}

type source struct {
	Name  string
	URL   string
	Parse func([]byte) (float64, error)
}

// Client fetches and caches a USD/CNY rate from public quotes.
type Client struct {
	HTTP     *http.Client
	Sources  []source
	CacheTTL time.Duration
	Now      func() time.Time

	mu      sync.Mutex
	fetchMu sync.Mutex
	cached  Rate
	until   time.Time
}

// Default is the process-wide rate client used by the management UI.
var Default = NewClient(nil)

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: requestTimeout}
	}
	return &Client{
		HTTP:     httpClient,
		Sources:  defaultSources(),
		CacheTTL: cacheTTL,
		Now:      time.Now,
	}
}

func defaultSources() []source {
	return []source{
		{Name: "jsdelivr", URL: "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/usd.min.json", Parse: parseCurrencyAPI},
		{Name: "cloudflare", URL: "https://latest.currency-api.pages.dev/v1/currencies/usd.min.json", Parse: parseCurrencyAPI},
		{Name: "er-api", URL: "https://open.er-api.com/v6/latest/USD", Parse: parseERAPI},
		{Name: "floatrates", URL: "https://www.floatrates.com/daily/usd.json", Parse: parseFloatRates},
	}
}

func GetUSDCNY(ctx context.Context, refresh bool) (Rate, error) {
	return Default.Get(ctx, refresh)
}

func (c *Client) Get(ctx context.Context, refresh bool) (Rate, error) {
	if c == nil {
		return Rate{}, errors.New("fx client is nil")
	}
	if rate, ok := c.cachedRate(refresh); ok {
		return rate, nil
	}
	c.fetchMu.Lock()
	defer c.fetchMu.Unlock()
	if rate, ok := c.cachedRate(refresh); ok {
		return rate, nil
	}

	now := c.now()
	c.mu.Lock()
	stale := c.cached
	c.mu.Unlock()

	rate, err := c.fetch(ctx)
	if err != nil {
		if Valid(stale.USDToCNY) {
			stale.Cached = true
			return stale, nil
		}
		return Rate{USDToCNY: DefaultUSDToCNY, Source: "default", FetchedAt: now.UTC(), Cached: true}, nil
	}
	ttl := c.CacheTTL
	if ttl <= 0 {
		ttl = cacheTTL
	}
	c.mu.Lock()
	c.cached = rate
	c.until = now.Add(ttl)
	c.mu.Unlock()
	return rate, nil
}

func (c *Client) cachedRate(refresh bool) (Rate, bool) {
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if refresh || !c.until.After(now) || !Valid(c.cached.USDToCNY) {
		return Rate{}, false
	}
	out := c.cached
	out.Cached = true
	return out, true
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Client) fetch(ctx context.Context) (Rate, error) {
	sources := c.Sources
	if len(sources) == 0 {
		sources = defaultSources()
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: requestTimeout}
	}
	var errs []string
	for _, src := range sources {
		rate, err := fetchSource(ctx, client, src)
		if err != nil {
			errs = append(errs, src.Name+": "+err.Error())
			continue
		}
		return rate, nil
	}
	if len(errs) == 0 {
		return Rate{}, errors.New("no usd/cny sources configured")
	}
	return Rate{}, errors.New("usd/cny rate unavailable: " + strings.Join(errs, "; "))
}

func fetchSource(ctx context.Context, client *http.Client, src source) (Rate, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return Rate{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "credit-manager/fx")
	res, err := client.Do(req)
	if err != nil {
		return Rate{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return Rate{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Rate{}, errors.New("http " + strconv.Itoa(res.StatusCode))
	}
	if src.Parse == nil {
		return Rate{}, errors.New("missing parser")
	}
	value, err := src.Parse(body)
	if err != nil {
		return Rate{}, err
	}
	if !Valid(value) {
		return Rate{}, errors.New("rate out of range")
	}
	return Rate{USDToCNY: value, Source: src.Name, FetchedAt: time.Now().UTC()}, nil
}

func Valid(rate float64) bool {
	return rate >= minUSDToCNY && rate <= maxUSDToCNY
}

func parseCurrencyAPI(body []byte) (float64, error) {
	var payload struct {
		USD map[string]float64 `json:"usd"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, err
	}
	return mapRate(payload.USD, "cny")
}

func parseERAPI(body []byte) (float64, error) {
	var payload struct {
		Result string             `json:"result"`
		Rates  map[string]float64 `json:"rates"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, err
	}
	if payload.Result != "" && !strings.EqualFold(payload.Result, "success") {
		return 0, errors.New(payload.Result)
	}
	return mapRate(payload.Rates, "CNY", "cny")
}

func parseFloatRates(body []byte) (float64, error) {
	var payload map[string]struct {
		Rate json.Number `json:"rate"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, err
	}
	item, ok := payload["cny"]
	if !ok {
		item, ok = payload["CNY"]
	}
	if !ok {
		return 0, errors.New("cny missing")
	}
	value, err := item.Rate.Float64()
	if err != nil {
		return 0, err
	}
	return value, nil
}

func mapRate(rates map[string]float64, keys ...string) (float64, error) {
	if rates == nil {
		return 0, errors.New("cny missing")
	}
	for _, key := range keys {
		if value, ok := rates[key]; ok {
			return value, nil
		}
		if value, ok := rates[strings.ToLower(key)]; ok {
			return value, nil
		}
		if value, ok := rates[strings.ToUpper(key)]; ok {
			return value, nil
		}
	}
	return 0, errors.New("cny missing")
}
