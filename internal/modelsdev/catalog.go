package modelsdev

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	cacheTTL        = 6 * time.Hour
	requestTimeout  = 8 * time.Second
	maxCatalogBytes = 16 << 20
)

// Catalog is a models.dev provider map used to backfill management prices.
type Catalog struct {
	Providers json.RawMessage `json:"catalog"`
	Source    string          `json:"source"`
	FetchedAt time.Time       `json:"fetched_at"`
	Cached    bool            `json:"cached"`
	Error     string          `json:"error,omitempty"`
}

type source struct {
	Name string
	URL  string
}

type diskCache struct {
	Source    string          `json:"source"`
	FetchedAt time.Time       `json:"fetched_at"`
	Catalog   json.RawMessage `json:"catalog"`
}

// Client fetches and caches the public models.dev provider catalog.
type Client struct {
	HTTP      *http.Client
	Sources   []source
	CacheTTL  time.Duration
	CacheFile string
	Now       func() time.Time

	mu      sync.Mutex
	fetchMu sync.Mutex
	cached  Catalog
	until   time.Time
}

// Default is the process-wide catalog client used by the management UI.
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
		{Name: "models.dev", URL: "https://models.dev/api.json"},
		{Name: "models.opencode.ai", URL: "https://models.opencode.ai/api.json"},
	}
}

func Get(ctx context.Context, refresh bool) Catalog {
	return Default.Get(ctx, refresh)
}

func (c *Client) SetCacheFile(path string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.CacheFile = strings.TrimSpace(path)
	c.mu.Unlock()
}

func (c *Client) Get(ctx context.Context, refresh bool) Catalog {
	if c == nil {
		return unavailable("models.dev client is nil")
	}
	if catalog, ok := c.cachedCatalog(refresh); ok {
		return catalog
	}
	c.fetchMu.Lock()
	defer c.fetchMu.Unlock()
	if catalog, ok := c.cachedCatalog(refresh); ok {
		return catalog
	}

	stale := c.memoryStale()
	catalog, err := c.fetch(ctx)
	if err == nil {
		ttl := c.CacheTTL
		if ttl <= 0 {
			ttl = cacheTTL
		}
		now := c.now()
		c.mu.Lock()
		c.cached = catalog
		c.until = now.Add(ttl)
		c.mu.Unlock()
		c.writeCacheFile(catalog)
		return catalog
	}
	if len(stale.Providers) > 0 {
		stale.Cached = true
		stale.Error = err.Error()
		return stale
	}
	if disk, ok := c.readCacheFile(); ok {
		return disk
	}
	out := unavailable(err.Error())
	out.FetchedAt = c.now().UTC()
	return out
}

func (c *Client) cachedCatalog(refresh bool) (Catalog, bool) {
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if refresh || !c.until.After(now) || !validCatalog(c.cached.Providers) {
		return Catalog{}, false
	}
	out := c.cached
	out.Cached = true
	return out, true
}

func (c *Client) memoryStale() Catalog {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !validCatalog(c.cached.Providers) {
		return Catalog{}
	}
	return c.cached
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Client) cacheFile() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.TrimSpace(c.CacheFile)
}

func (c *Client) fetch(ctx context.Context) (Catalog, error) {
	sources := c.Sources
	if len(sources) == 0 {
		sources = defaultSources()
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: requestTimeout}
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	type result struct {
		catalog Catalog
		err     error
		idx     int
	}
	ch := make(chan result, len(sources))
	for i, src := range sources {
		go func(i int, src source) {
			catalog, err := fetchSource(ctx, client, src)
			ch <- result{catalog: catalog, err: err, idx: i}
		}(i, src)
	}

	var errs []string
	pending := len(sources)
	for pending > 0 {
		select {
		case <-ctx.Done():
			if len(errs) == 0 {
				return Catalog{}, ctx.Err()
			}
			return Catalog{}, errors.New("models.dev catalog unavailable: " + strings.Join(errs, "; "))
		case item := <-ch:
			pending--
			if item.err != nil {
				errs = append(errs, sources[item.idx].Name+": "+item.err.Error())
				continue
			}
			cancel()
			return item.catalog, nil
		}
	}
	if len(errs) == 0 {
		return Catalog{}, errors.New("no models.dev sources configured")
	}
	return Catalog{}, errors.New("models.dev catalog unavailable: " + strings.Join(errs, "; "))
}

func fetchSource(ctx context.Context, client *http.Client, src source) (Catalog, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return Catalog{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "credit-manager/modelsdev")
	res, err := client.Do(req)
	if err != nil {
		return Catalog{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxCatalogBytes+1))
	if err != nil {
		return Catalog{}, err
	}
	if len(body) > maxCatalogBytes {
		return Catalog{}, errors.New("catalog too large")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Catalog{}, errors.New("http " + strconv.Itoa(res.StatusCode))
	}
	if !validCatalog(body) {
		return Catalog{}, errors.New("malformed catalog")
	}
	return Catalog{Providers: json.RawMessage(body), Source: src.Name, FetchedAt: time.Now().UTC()}, nil
}

func (c *Client) writeCacheFile(catalog Catalog) {
	path := c.cacheFile()
	if path == "" || !validCatalog(catalog.Providers) {
		return
	}
	payload, err := json.Marshal(diskCache{
		Source:    catalog.Source,
		FetchedAt: catalog.FetchedAt,
		Catalog:   catalog.Providers,
	})
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
	}
}

func (c *Client) readCacheFile() (Catalog, bool) {
	path := c.cacheFile()
	if path == "" {
		return Catalog{}, false
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, false
	}
	var cached diskCache
	if err := json.Unmarshal(body, &cached); err != nil || !validCatalog(cached.Catalog) {
		return Catalog{}, false
	}
	return Catalog{
		Providers: cached.Catalog,
		Source:    cached.Source,
		FetchedAt: cached.FetchedAt.UTC(),
		Cached:    true,
	}, true
}

func validCatalog(raw []byte) bool {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil || len(root) == 0 {
		return false
	}
	for _, value := range root {
		var provider struct {
			Models json.RawMessage `json:"models"`
		}
		if json.Unmarshal(value, &provider) != nil || len(provider.Models) == 0 {
			continue
		}
		var models map[string]json.RawMessage
		if json.Unmarshal(provider.Models, &models) == nil && len(models) > 0 {
			return true
		}
	}
	return false
}

func unavailable(message string) Catalog {
	return Catalog{
		Providers: json.RawMessage(`{}`),
		Source:    "unavailable",
		Error:     message,
	}
}
