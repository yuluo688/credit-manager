package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/keys"
	"github.com/yuluo688/credit-manager/internal/store"
)

// Authenticate verifies a client plugin key and returns a stable principal.
// query and rawPath accept the same native credential locations as CLIProxyAPI
// (X-Goog-Api-Key, X-Api-Key, ?key=, ?auth_token=, or a query string still
// attached to the path).
func (s *Service) Authenticate(ctx context.Context, headers http.Header, query url.Values, rawPath string) (principal string, metadata map[string]string, ok bool) {
	raw := PluginKeyFromRequest(headers, query, rawPath)
	if raw == "" {
		return "", nil, false
	}
	kid, err := keys.Parse(raw)
	if err != nil {
		return "", nil, false
	}
	key, err := s.store.GetPluginKeyByKid(ctx, kid)
	if err != nil {
		return "", nil, false
	}
	if err := store.EnsurePluginKeyUsable(key, time.Now().UTC()); err != nil {
		return "", nil, false
	}
	if _, ok := keys.Verify(raw, key.KeyHash, key.PepperID, s.peppers); !ok {
		return "", nil, false
	}
	caller, err := s.store.GetCaller(ctx, key.CallerID)
	if err != nil || !caller.Enabled {
		return "", nil, false
	}
	_ = s.store.TouchPluginKeyUsed(ctx, key.ID)
	return key.Principal, map[string]string{
		"plugin":      PluginID,
		"kid":         key.Kid,
		"caller_id":   key.CallerID,
		"fingerprint": key.Fingerprint,
	}, true
}

// LookupPluginKey verifies a plaintext plugin key for its self-service usage page.
// It intentionally returns one generic error for invalid, unavailable, or disabled
// keys so the public endpoint does not expose key state to unauthenticated callers.
func (s *Service) LookupPluginKey(ctx context.Context, raw string) (store.PluginKey, error) {
	raw = strings.TrimSpace(raw)
	kid, err := keys.Parse(raw)
	if err != nil {
		return store.PluginKey{}, keys.ErrInvalidKey
	}
	key, err := s.store.GetPluginKeyByKid(ctx, kid)
	if err != nil {
		return store.PluginKey{}, keys.ErrInvalidKey
	}
	if err := store.EnsurePluginKeyUsable(key, time.Now().UTC()); err != nil {
		return store.PluginKey{}, keys.ErrInvalidKey
	}
	if _, ok := keys.Verify(raw, key.KeyHash, key.PepperID, s.peppers); !ok {
		return store.PluginKey{}, keys.ErrInvalidKey
	}
	caller, err := s.store.GetCaller(ctx, key.CallerID)
	if err != nil || !caller.Enabled {
		return store.PluginKey{}, keys.ErrInvalidKey
	}
	return key, nil
}

func (s *Service) LookupPluginKeyFromHeaders(ctx context.Context, headers http.Header, query url.Values) (store.PluginKey, error) {
	return s.LookupPluginKey(ctx, PluginKeyFromRequest(headers, query, ""))
}

// PluginKeyFromRequest extracts a client plugin key.
// Priority matches CLIProxyAPI's built-in API-key provider:
// Authorization: Bearer, X-Goog-Api-Key, X-Api-Key, ?key=, then ?auth_token=.
// rawPath is consulted only when it still carries a query string.
func PluginKeyFromRequest(headers http.Header, query url.Values, rawPath string) string {
	if token := bearerToken(headers); token != "" {
		return token
	}
	for _, name := range []string{"X-Goog-Api-Key", "X-Api-Key"} {
		if token := stripOptionalBearer(headerValue(headers, name)); token != "" {
			return token
		}
	}
	if token := credentialFromQuery(query); token != "" {
		return token
	}
	return credentialFromQuery(queryFromRawPath(rawPath))
}

func bearerToken(headers http.Header) string {
	auth := headerValue(headers, "Authorization")
	if len(auth) < 7 || !strings.EqualFold(auth[:7], "bearer ") {
		return ""
	}
	return strings.TrimSpace(auth[7:])
}

func credentialFromQuery(query url.Values) string {
	for _, name := range []string{"key", "auth_token"} {
		if token := stripOptionalBearer(queryValue(query, name)); token != "" {
			return token
		}
	}
	return ""
}

func queryValue(query url.Values, name string) string {
	if len(query) == 0 {
		return ""
	}
	for _, value := range query[name] {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	for key, values := range query {
		if !strings.EqualFold(key, name) {
			continue
		}
		for _, value := range values {
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		}
	}
	return ""
}

func queryFromRawPath(rawPath string) url.Values {
	rawPath = strings.TrimSpace(rawPath)
	if i := strings.IndexByte(rawPath, '#'); i >= 0 {
		rawPath = rawPath[:i]
	}
	i := strings.IndexByte(rawPath, '?')
	if i < 0 || i+1 >= len(rawPath) {
		return nil
	}
	values, err := url.ParseQuery(rawPath[i+1:])
	if err != nil {
		return nil
	}
	return values
}

func headerValue(headers http.Header, name string) string {
	if headers == nil {
		return ""
	}
	for _, value := range headers.Values(name) {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	for key, values := range headers {
		if !strings.EqualFold(key, name) {
			continue
		}
		for _, value := range values {
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		}
	}
	return ""
}

func stripOptionalBearer(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 7 && strings.EqualFold(value[:7], "bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return value
}

// ResolveIdentity finds the plugin key for an authenticated execution request.
func (s *Service) ResolveIdentity(ctx context.Context, headers http.Header, query url.Values, metadata map[string]any) (store.PluginKey, store.Caller, error) {
	if raw := PluginKeyFromRequest(headers, query, ""); raw != "" {
		if kid, err := keys.Parse(raw); err == nil {
			key, err := s.store.GetPluginKeyByKid(ctx, kid)
			if err == nil {
				if err := store.EnsurePluginKeyUsable(key, time.Now().UTC()); err != nil {
					return store.PluginKey{}, store.Caller{}, err
				}
				caller, err := s.store.GetCaller(ctx, key.CallerID)
				if err != nil {
					return store.PluginKey{}, store.Caller{}, err
				}
				if !caller.Enabled {
					return store.PluginKey{}, store.Caller{}, store.ErrCallerDisabled
				}
				return key, caller, nil
			}
		}
	}
	scope := metadataString(metadata, CallerScopeMetadataKey)
	if scope == "" {
		return store.PluginKey{}, store.Caller{}, fmt.Errorf("%w: missing caller identity", store.ErrInvalidArgument)
	}
	key, err := s.store.GetPluginKeyByCallerScope(ctx, scope)
	if err != nil {
		return store.PluginKey{}, store.Caller{}, err
	}
	if err := store.EnsurePluginKeyUsable(key, time.Now().UTC()); err != nil {
		return store.PluginKey{}, store.Caller{}, err
	}
	caller, err := s.store.GetCaller(ctx, key.CallerID)
	if err != nil {
		return store.PluginKey{}, store.Caller{}, err
	}
	if !caller.Enabled {
		return store.PluginKey{}, store.Caller{}, store.ErrCallerDisabled
	}
	return key, caller, nil
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	if value, ok := metadata[key]; ok && value != nil {
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
		return strings.TrimSpace(fmt.Sprint(value))
	}
	// try common casing variants
	for k, value := range metadata {
		if strings.EqualFold(k, key) && value != nil {
			if text, ok := value.(string); ok {
				return strings.TrimSpace(text)
			}
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return ""
}
