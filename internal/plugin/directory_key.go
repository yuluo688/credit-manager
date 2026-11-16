package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

const (
	directoryIntentTTL  = 5 * time.Second
	maxDirectoryIntents = 256
)

// directoryIntent remembers a model-list authentication that carried its Key
// only in the query string. The response interceptor sees headers, not the URL.
// Header credentials are resolved directly and are not queued.
//
// Anonymous directory requests are queued too. A response is trimmed only when
// exactly one intent matches its headers, so overlapping clients fail open
// instead of seeing each other's allowlist.
type directoryIntent struct {
	fingerprint string
	key         store.PluginKey
	restricted  bool
	expires     time.Time
}

var (
	directoryIntentMu sync.Mutex
	directoryIntents  []directoryIntent
)

func noteModelDirectoryAuth(req pluginapi.FrontendAuthRequest) {
	if service.PluginKeyFromRequest(req.Headers, nil, "") != "" {
		return
	}
	key, restricted := directoryKeyFromQuery(req)
	rememberDirectoryIntent(directoryHeaderFingerprint(req.Headers), key, restricted, time.Now())
}

func directoryKeyFromQuery(req pluginapi.FrontendAuthRequest) (store.PluginKey, bool) {
	svc := service.Current()
	if svc == nil {
		return store.PluginKey{}, false
	}
	raw := service.PluginKeyFromRequest(nil, req.Query, req.Path)
	if raw == "" {
		return store.PluginKey{}, false
	}
	key, err := svc.LookupPluginKey(context.Background(), raw)
	if err != nil {
		return store.PluginKey{}, false
	}
	return directoryFilterKey(key), true
}

func modelDirectoryKey(ctx context.Context, headers http.Header) store.PluginKey {
	svc := service.Current()
	if svc == nil {
		return store.PluginKey{}
	}
	if raw := service.PluginKeyFromRequest(headers, nil, ""); raw != "" {
		key, err := svc.LookupPluginKey(ctx, raw)
		if err != nil {
			return store.PluginKey{}
		}
		return key
	}
	key, ok := takeDirectoryIntent(directoryHeaderFingerprint(headers), time.Now())
	if !ok {
		return store.PluginKey{}
	}
	return key
}

func directoryFilterKey(key store.PluginKey) store.PluginKey {
	return store.PluginKey{
		AllowedModels:       append([]string(nil), key.AllowedModels...),
		ModelTokenLimits:    append([]store.ModelTokenLimit(nil), key.ModelTokenLimits...),
		UnmatchedModelsMode: key.UnmatchedModelsMode,
	}
}

func rememberDirectoryIntent(fingerprint string, key store.PluginKey, restricted bool, now time.Time) {
	directoryIntentMu.Lock()
	defer directoryIntentMu.Unlock()
	purgeDirectoryIntentsLocked(now)
	directoryIntents = append(directoryIntents, directoryIntent{
		fingerprint: fingerprint,
		key:         key,
		restricted:  restricted,
		expires:     now.Add(directoryIntentTTL),
	})
	if len(directoryIntents) > maxDirectoryIntents {
		directoryIntents = directoryIntents[len(directoryIntents)-maxDirectoryIntents:]
	}
}

func takeDirectoryIntent(fingerprint string, now time.Time) (store.PluginKey, bool) {
	directoryIntentMu.Lock()
	defer directoryIntentMu.Unlock()
	purgeDirectoryIntentsLocked(now)
	match := -1
	for i := range directoryIntents {
		if directoryIntents[i].fingerprint != fingerprint {
			continue
		}
		if match >= 0 {
			return store.PluginKey{}, false
		}
		match = i
	}
	if match < 0 {
		return store.PluginKey{}, false
	}
	intent := directoryIntents[match]
	directoryIntents = append(directoryIntents[:match], directoryIntents[match+1:]...)
	if !intent.restricted {
		return store.PluginKey{}, false
	}
	return intent.key, true
}

func purgeDirectoryIntentsLocked(now time.Time) {
	kept := directoryIntents[:0]
	for _, intent := range directoryIntents {
		if now.Before(intent.expires) {
			kept = append(kept, intent)
		}
	}
	directoryIntents = kept
}

func resetDirectoryIntents() {
	directoryIntentMu.Lock()
	directoryIntents = nil
	directoryIntentMu.Unlock()
}

func directoryHeaderFingerprint(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}
	merged := make(map[string][]string, len(headers))
	for key, values := range headers {
		canon := strings.ToLower(key)
		merged[canon] = append(merged[canon], values...)
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hash := sha256.New()
	for _, key := range keys {
		hash.Write([]byte(key))
		hash.Write([]byte{0})
		for _, value := range merged[key] {
			hash.Write([]byte(value))
			hash.Write([]byte{0})
		}
		hash.Write([]byte{1})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
