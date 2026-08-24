package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/keys"
	"github.com/yuluo688/credit-manager/internal/store"
)

// Authenticate verifies a Bearer plugin key and returns a stable principal.
func (s *Service) Authenticate(ctx context.Context, headers http.Header) (principal string, metadata map[string]string, ok bool) {
	raw := bearerToken(headers)
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

func (s *Service) LookupPluginKeyFromHeaders(ctx context.Context, headers http.Header) (store.PluginKey, error) {
	return s.LookupPluginKey(ctx, bearerToken(headers))
}

func bearerToken(headers http.Header) string {
	if headers == nil {
		return ""
	}
	auth := strings.TrimSpace(headers.Get("Authorization"))
	if auth == "" {
		// case-insensitive fallback
		for k, values := range headers {
			if strings.EqualFold(k, "Authorization") && len(values) > 0 {
				auth = strings.TrimSpace(values[0])
				break
			}
		}
	}
	if len(auth) < 8 || !strings.EqualFold(auth[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(auth[7:])
}

// ResolveIdentity finds the plugin key for an authenticated execution request.
func (s *Service) ResolveIdentity(ctx context.Context, headers http.Header, metadata map[string]any) (store.PluginKey, store.Caller, error) {
	if raw := bearerToken(headers); raw != "" {
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
