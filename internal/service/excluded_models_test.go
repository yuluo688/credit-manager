package service

import (
	"encoding/json"
	"testing"
)

func mergeFields(t *testing.T, out []byte) map[string]json.RawMessage {
	t.Helper()
	var meta map[string]json.RawMessage
	if err := json.Unmarshal(out, &meta); err != nil {
		t.Fatalf("decode merged auth: %v", err)
	}
	return meta
}

func TestMergeAuthExcludedModelsKeepsUserRemovalAcrossSyncs(t *testing.T) {
	// File state after a sync where the plugin managed "old-hidden".
	edited := []byte(`{"type":"devin","excluded_models":["keep-me"],"credit_manager_excluded_models":["old-hidden"]}`)

	// User removes the plugin-managed entry.
	out, changed, err := MergeAuthExcludedModels(edited, []string{"old-hidden"})
	if err != nil || !changed {
		t.Fatalf("removal merge changed=%t err=%v", changed, err)
	}
	meta := mergeFields(t, out)
	if got := stringSliceFromRaw(meta["excluded_models"]); !stringSlicesEqual(got, []string{"keep-me"}) {
		t.Fatalf("excluded_models = %#v", got)
	}
	if got := stringSliceFromRaw(meta[managedRemovedModelsKey]); !stringSlicesEqual(got, []string{"old-hidden"}) {
		t.Fatalf("removed = %#v", got)
	}
	if _, ok := meta[managedExcludedModelsKey]; ok {
		t.Fatalf("managed key should be removed: %s", out)
	}

	// Next sync must not re-add the removed model but must add new disabled ones.
	out, changed, err = MergeAuthExcludedModels(out, []string{"new-hidden", "old-hidden"})
	if err != nil || !changed {
		t.Fatalf("second sync changed=%t err=%v", changed, err)
	}
	meta = mergeFields(t, out)
	if got := stringSliceFromRaw(meta["excluded_models"]); !stringSlicesEqual(got, []string{"keep-me", "new-hidden"}) {
		t.Fatalf("excluded_models = %#v", got)
	}
	if got := stringSliceFromRaw(meta[managedRemovedModelsKey]); !stringSlicesEqual(got, []string{"old-hidden"}) {
		t.Fatalf("removed = %#v", got)
	}
}

func TestMergeAuthExcludedModelsUserReaddClearsRemoval(t *testing.T) {
	raw := []byte(`{"type":"devin","excluded_models":["m1"],"credit_manager_removed_models":["m1"]}`)
	out, changed, err := MergeAuthExcludedModels(raw, []string{"m1"})
	if err != nil || !changed {
		t.Fatalf("changed=%t err=%v", changed, err)
	}
	meta := mergeFields(t, out)
	if _, ok := meta[managedRemovedModelsKey]; ok {
		t.Fatalf("removed key should be cleared: %s", out)
	}
	if _, ok := meta[managedExcludedModelsKey]; ok {
		t.Fatalf("re-added model must stay user-owned: %s", out)
	}
	// A later global re-enable must keep the user's own exclusion.
	out, changed, err = MergeAuthExcludedModels(out, nil)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatalf("re-enable rewrote user-owned exclusion: %s", out)
	}
}

func TestMergeAuthExcludedModelsDoesNotHijackUserExclusion(t *testing.T) {
	raw := []byte(`{"type":"devin","excluded_models":["u1"]}`)
	out, changed, err := MergeAuthExcludedModels(raw, []string{"u1"})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatalf("user-owned exclusion already covering disabled set must be a no-op: %s", out)
	}
	out, changed, err = MergeAuthExcludedModels(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatalf("empty disabled rewrote user exclusion: %s", out)
	}
}

func TestMergeAuthExcludedModelsGlobalReenableCleanup(t *testing.T) {
	raw := []byte(`{"type":"devin","excluded_models":["m1","u1"],"credit_manager_excluded_models":["m1"]}`)
	out, changed, err := MergeAuthExcludedModels(raw, nil)
	if err != nil || !changed {
		t.Fatalf("changed=%t err=%v", changed, err)
	}
	meta := mergeFields(t, out)
	if got := stringSliceFromRaw(meta["excluded_models"]); !stringSlicesEqual(got, []string{"u1"}) {
		t.Fatalf("excluded_models = %#v", got)
	}
	if _, ok := meta[managedExcludedModelsKey]; ok {
		t.Fatalf("managed key should be removed: %s", out)
	}
}

func TestMergeAuthExcludedModelsRemovedSurvivesColdStart(t *testing.T) {
	raw := []byte(`{"type":"devin","excluded_models":["u1"],"credit_manager_removed_models":["claude-opus"]}`)
	out, changed, err := MergeAuthExcludedModels(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatalf("cold start must not rewrite the file: %s", out)
	}
	out, changed, err = MergeAuthExcludedModels(raw, []string{"claude-opus"})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatalf("removed model re-added: %s", out)
	}
}

func TestMergeAuthExcludedModelsOptOut(t *testing.T) {
	raw := []byte(`{"type":"devin","excluded_models":["u1"],"credit_manager_managed":false}`)
	out, changed, err := MergeAuthExcludedModels(raw, []string{"m1", "m2"})
	if err != nil || changed {
		t.Fatalf("opt-out changed=%t err=%v out=%s", changed, err, out)
	}
	if string(out) != string(raw) {
		t.Fatalf("opt-out rewrote file: %s", out)
	}
}

func TestMergeAuthExcludedModelsCanonicalEmptyBeatsLegacy(t *testing.T) {
	raw := []byte(`{"type":"devin","excluded_models":[],"excluded-models":["legacy"]}`)
	out, changed, err := MergeAuthExcludedModels(raw, nil)
	if err != nil || !changed {
		t.Fatalf("changed=%t err=%v", changed, err)
	}
	meta := mergeFields(t, out)
	if _, ok := meta["excluded-models"]; ok {
		t.Fatalf("legacy key should be removed: %s", out)
	}
	if _, ok := meta["excluded_models"]; ok {
		t.Fatalf("empty canonical key should be removed: %s", out)
	}
}

func TestMergeAuthExcludedModelsDeletingBookkeepingResets(t *testing.T) {
	raw := []byte(`{"type":"devin","excluded_models":["u1"]}`)
	out, changed, err := MergeAuthExcludedModels(raw, []string{"m1"})
	if err != nil || !changed {
		t.Fatalf("changed=%t err=%v", changed, err)
	}
	meta := mergeFields(t, out)
	if got := stringSliceFromRaw(meta["excluded_models"]); !stringSlicesEqual(got, []string{"m1", "u1"}) {
		t.Fatalf("excluded_models = %#v", got)
	}
	if got := stringSliceFromRaw(meta[managedExcludedModelsKey]); !stringSlicesEqual(got, []string{"m1"}) {
		t.Fatalf("managed = %#v", got)
	}
}

func catalogFor(cred, provider []string, providers ...string) AuthModelCatalog {
	credSet := make(map[string]struct{}, len(cred))
	for _, m := range cred {
		credSet[m] = struct{}{}
	}
	provSet := make(map[string]struct{}, len(provider))
	for _, m := range provider {
		provSet[m] = struct{}{}
	}
	known := make(map[string]struct{}, len(providers))
	for _, p := range providers {
		known[NormalizeProviderKey(p)] = struct{}{}
	}
	return AuthModelCatalog{
		CredModels:     credSet,
		ProviderModels: provSet,
		KnownProviders: known,
		Available:      true,
	}
}

func TestScopeDisabledModelsForAuthDropsForeignProviderPrefix(t *testing.T) {
	raw := []byte(`{"type":"claude","credit_manager_excluded_models":["devin/swe-1-6-slow"]}`)
	cat := catalogFor([]string{"claude-opus-5"}, []string{"claude-opus-5"}, "claude", "devin")
	got := ScopeDisabledModelsForAuth(raw, "claude", []string{"devin/swe-1-6-slow", "claude-opus-5"}, cat)
	if !stringSlicesEqual(got, []string{"claude-opus-5"}) {
		t.Fatalf("scoped = %#v", got)
	}
}

func TestScopeDisabledModelsForAuthKeepsCredentialAndProviderModels(t *testing.T) {
	raw := []byte(`{"type":"devin"}`)
	cat := catalogFor([]string{"swe-2"}, []string{"claude-opus-5", "swe-2"}, "claude", "devin")
	got := ScopeDisabledModelsForAuth(raw, "devin", []string{"swe-2", "claude-opus-5", "kimi-k2"}, cat)
	if !stringSlicesEqual(got, []string{"swe-2", "claude-opus-5"}) {
		t.Fatalf("scoped = %#v", got)
	}
}

func TestScopeDisabledModelsForAuthAntiOscillation(t *testing.T) {
	// Model excluded on the credential is no longer in its registered catalog;
	// it must stay scoped in or every second sync would un-exclude it.
	raw := []byte(`{"type":"devin","credit_manager_excluded_models":["swe-2"]}`)
	cat := catalogFor(nil, nil, "claude", "devin")
	got := ScopeDisabledModelsForAuth(raw, "devin", []string{"swe-2", "other"}, cat)
	if !stringSlicesEqual(got, []string{"swe-2"}) {
		t.Fatalf("scoped = %#v", got)
	}
}

func TestScopeDisabledModelsForAuthEmptyCatalogSkipsNew(t *testing.T) {
	raw := []byte(`{"type":"antigravity"}`)
	cat := catalogFor(nil, nil, "antigravity", "devin")
	got := ScopeDisabledModelsForAuth(raw, "antigravity", []string{"devin-only", "swe-2"}, cat)
	if len(got) != 0 {
		t.Fatalf("scoped = %#v", got)
	}
}

func TestScopeDisabledModelsForAuthUnavailableCatalog(t *testing.T) {
	raw := []byte(`{"type":"claude"}`)
	cat := AuthModelCatalog{
		KnownProviders: map[string]struct{}{"claude": {}, "devin": {}},
	}
	got := ScopeDisabledModelsForAuth(raw, "claude", []string{"devin/swe-1-6-slow", "claude-opus-5"}, cat)
	if !stringSlicesEqual(got, []string{"claude-opus-5"}) {
		t.Fatalf("scoped = %#v", got)
	}
}

func TestMergeAuthExcludedModelsManagedFlagVariants(t *testing.T) {
	// Only the JSON literal false opts out; null/true/strings/numbers/objects
	// are all treated as managed.
	for _, v := range []string{"null", "true", `"false"`, "0", "1", "{}"} {
		raw := []byte(`{"type":"devin","credit_manager_managed":` + v + `}`)
		out, changed, err := MergeAuthExcludedModels(raw, []string{"m1"})
		if err != nil || !changed {
			t.Fatalf("managed=%s changed=%t err=%v", v, changed, err)
		}
		meta := mergeFields(t, out)
		if got := stringSliceFromRaw(meta["excluded_models"]); !stringSlicesEqual(got, []string{"m1"}) {
			t.Fatalf("managed=%s excluded_models = %#v", v, got)
		}
	}
}

func TestMergeAuthExcludedModelsMalformedBookkeepingKeys(t *testing.T) {
	for _, raw := range []string{
		`{"type":"devin","credit_manager_excluded_models":123,"excluded_models":["u1"]}`,
		`{"type":"devin","credit_manager_excluded_models":{"a":1},"credit_manager_removed_models":[1,true],"excluded_models":["u1"]}`,
		`{"type":"devin","excluded_models":{"a":1}}`,
		`{"type":"devin","excluded_models":"u1,u2"}`,
	} {
		out, changed, err := MergeAuthExcludedModels([]byte(raw), []string{"m1"})
		if err != nil {
			t.Fatalf("raw=%s err=%v", raw, err)
		}
		if !changed {
			t.Fatalf("raw=%s expected change", raw)
		}
		var meta map[string]json.RawMessage
		if err := json.Unmarshal(out, &meta); err != nil {
			t.Fatalf("raw=%s produced invalid json: %v", raw, err)
		}
	}
}

func TestMergeAuthExcludedModelsTruncatedJSON(t *testing.T) {
	for _, raw := range []string{
		`{"type":"devin","excluded_models":["u1"]`,
		`{"type":"devin"}}`,
		`{"type":"devin"}]`,
		`{"type":"devin"} garbage`,
	} {
		if _, _, err := MergeAuthExcludedModels([]byte(raw), nil); err == nil {
			t.Fatalf("malformed auth json must error: %s", raw)
		}
	}
}

func TestScopeDisabledModelsForAuthGeminiNormalization(t *testing.T) {
	// File/provider "gemini" normalizes to "gemini-cli"; a gemini-cli-prefixed
	// disabled model must be treated as same-provider, not foreign.
	raw := []byte(`{"type":"gemini"}`)
	cat := catalogFor([]string{"gemini-cli/x", "m2"}, nil, "gemini-cli", "devin")
	got := ScopeDisabledModelsForAuth(raw, "gemini", []string{"gemini-cli/x", "devin/y"}, cat)
	if !stringSlicesEqual(got, []string{"gemini-cli/x"}) {
		t.Fatalf("scoped = %#v", got)
	}
}

func TestScopeDisabledModelsForAuthModelsPathPrefix(t *testing.T) {
	// "models/gemini-pro" is a Gemini API path form, not a provider-qualified
	// ID — "models" is never a known provider so the entry stays eligible.
	raw := []byte(`{"type":"gemini-cli"}`)
	cat := catalogFor([]string{"models/gemini-pro"}, nil, "gemini-cli", "devin")
	got := ScopeDisabledModelsForAuth(raw, "gemini-cli", []string{"models/gemini-pro"}, cat)
	if !stringSlicesEqual(got, []string{"models/gemini-pro"}) {
		t.Fatalf("scoped = %#v", got)
	}
}
