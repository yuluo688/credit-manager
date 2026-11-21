package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/yuluo688/credit-manager/internal/service"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

type hostModelDirectorySyncer struct{}

func (hostModelDirectorySyncer) SyncExcludedModels(_ context.Context, disabled []string) error {
	raw, err := callHost(pluginabi.MethodHostAuthList, map[string]any{})
	if err != nil {
		return err
	}
	var listed struct {
		Files []pluginapi.HostAuthFileEntry `json:"files"`
	}
	if err := json.Unmarshal(raw, &listed); err != nil {
		return err
	}
	index := authModelCatalogs(listed.Files)
	var errs []string
	for _, file := range listed.Files {
		if file.RuntimeOnly || strings.TrimSpace(file.AuthIndex) == "" {
			continue
		}
		if err := syncAuthExcludedModels(file, disabled, index); err != nil {
			name := strings.TrimSpace(file.Name)
			if name == "" {
				name = strings.TrimSpace(file.AuthIndex)
			}
			errs = append(errs, name+": "+err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New("sync excluded models: " + strings.Join(errs, "; "))
	}
	return nil
}

// authCatalogIndex holds the host model-registry view indexed by credential,
// resolved lazily per auth file so provider fallbacks still reach the
// provider-wide model union.
type authCatalogIndex struct {
	credModels     map[string]map[string]struct{}
	credProviders  map[string]string
	providerModels map[string]map[string]struct{}
	knownProviders map[string]struct{}
	available      bool
}

func (idx authCatalogIndex) forAuth(authIndex, provider string) service.AuthModelCatalog {
	return service.AuthModelCatalog{
		CredModels:     idx.credModels[authIndex],
		ProviderModels: idx.providerModels[provider],
		KnownProviders: idx.knownProviders,
		Available:      idx.available,
	}
}

// syncAuthExcludedModels scopes the global disabled list to the models this
// credential actually serves, then merges it into the file's excluded_models.
// The JSON is re-read before saving so user edits landing mid-sync are merged
// rather than overwritten.
func syncAuthExcludedModels(file pluginapi.HostAuthFileEntry, disabled []string, index authCatalogIndex) error {
	authIndex := strings.TrimSpace(file.AuthIndex)
	resolveProvider := func(authJSON []byte) string {
		if p := authFileProvider(file, nil); p != "" {
			return p
		}
		if p := service.NormalizeProviderKey(index.credProviders[authIndex]); p != "" {
			return p
		}
		return authFileProvider(file, authJSON)
	}
	scopedFor := func(authJSON []byte) []string {
		provider := resolveProvider(authJSON)
		return service.ScopeDisabledModelsForAuth(authJSON, provider, disabled, index.forAuth(authIndex, provider))
	}
	load := func() (pluginapi.HostAuthGetResponse, error) {
		got, err := callHost(pluginabi.MethodHostAuthGet, pluginapi.HostAuthGetRequest{AuthIndex: authIndex})
		if err != nil {
			return pluginapi.HostAuthGetResponse{}, err
		}
		var auth pluginapi.HostAuthGetResponse
		if err := json.Unmarshal(got, &auth); err != nil {
			return pluginapi.HostAuthGetResponse{}, err
		}
		if len(auth.JSON) == 0 {
			return pluginapi.HostAuthGetResponse{}, errors.New("empty auth json")
		}
		return auth, nil
	}
	auth, err := load()
	if err != nil {
		return err
	}
	next, changed, err := service.MergeAuthExcludedModels(auth.JSON, scopedFor(auth.JSON))
	if err != nil || !changed {
		return err
	}
	fresh, err := load()
	if err != nil {
		return err
	}
	if !bytes.Equal(auth.JSON, fresh.JSON) {
		next, changed, err = service.MergeAuthExcludedModels(fresh.JSON, scopedFor(fresh.JSON))
		if err != nil || !changed {
			return err
		}
		auth = fresh
	}
	name := strings.TrimSpace(file.Name)
	if name == "" {
		name = strings.TrimSpace(auth.Name)
	}
	if name == "" || !strings.HasSuffix(strings.ToLower(name), ".json") {
		return errors.New("auth file name is required")
	}
	_, err = callHost(pluginabi.MethodHostAuthSave, pluginapi.HostAuthSaveRequest{Name: name, JSON: next})
	return err
}

// authFileProvider resolves the credential's provider from the list entry,
// falling back to the "type" field inside the auth JSON payload.
func authFileProvider(file pluginapi.HostAuthFileEntry, authJSON []byte) string {
	for _, candidate := range []string{file.Provider, file.Type} {
		if p := service.NormalizeProviderKey(candidate); p != "" {
			return p
		}
	}
	if len(authJSON) == 0 {
		return ""
	}
	var meta struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(authJSON, &meta); err != nil {
		return ""
	}
	return service.NormalizeProviderKey(meta.Type)
}

// authModelCatalogs builds the per-credential model view used for scoping. A
// failed host.auth.models call (older host, missing capability) degrades to
// prefix-only scoping instead of failing the sync.
func authModelCatalogs(files []pluginapi.HostAuthFileEntry) authCatalogIndex {
	index := authCatalogIndex{
		credModels:     map[string]map[string]struct{}{},
		credProviders:  map[string]string{},
		providerModels: map[string]map[string]struct{}{},
		knownProviders: map[string]struct{}{},
	}
	// Seed with providers the host API commonly reports so provider-qualified
	// model IDs are recognized even when that provider has no credentials.
	for _, p := range []string{
		"antigravity", "aistudio", "claude", "codex", "devin", "gemini-cli",
		"iflow", "kimi", "meta", "qwen", "vertex", "vertex-cli", "xai",
	} {
		index.knownProviders[service.NormalizeProviderKey(p)] = struct{}{}
	}
	for _, file := range files {
		if p := authFileProvider(file, nil); p != "" {
			index.knownProviders[p] = struct{}{}
		}
	}

	raw, err := callHost(pluginabi.MethodHostAuthModels, pluginapi.HostAuthModelsRequest{})
	if err != nil {
		return index
	}
	var modelsResp pluginapi.HostAuthModelsResponse
	if err := json.Unmarshal(raw, &modelsResp); err != nil {
		return index
	}
	index.available = true

	for _, cred := range modelsResp.Credentials {
		authIndex := strings.TrimSpace(cred.AuthIndex)
		if authIndex == "" {
			continue
		}
		set := make(map[string]struct{}, len(cred.Models))
		for _, m := range cred.Models {
			m = strings.ToLower(strings.TrimSpace(m))
			if m != "" {
				set[m] = struct{}{}
			}
		}
		index.credModels[authIndex] = set
		if p := service.NormalizeProviderKey(cred.Provider); p != "" {
			index.credProviders[authIndex] = p
			index.knownProviders[p] = struct{}{}
			union := index.providerModels[p]
			if union == nil {
				union = map[string]struct{}{}
				index.providerModels[p] = union
			}
			for m := range set {
				union[m] = struct{}{}
			}
		}
	}
	return index
}
