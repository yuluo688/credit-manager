package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"

	"github.com/yuluo688/credit-manager/internal/store"
)

const (
	managedExcludedModelsKey = "credit_manager_excluded_models"
	managedRemovedModelsKey  = "credit_manager_removed_models"
	managedOptOutKey         = "credit_manager_managed"
)

type ModelDirectorySyncer interface {
	SyncExcludedModels(ctx context.Context, disabled []string) error
}

func (s *Service) SetModelDirectorySyncer(syncer ModelDirectorySyncer) {
	if s == nil {
		return
	}
	s.directorySyncer = syncer
}

func (s *Service) RefreshModelDirectory(ctx context.Context) error {
	if s == nil || s.directorySyncer == nil {
		return nil
	}
	disabled, err := s.disabledDirectoryModels(ctx)
	if err != nil {
		return err
	}
	return s.directorySyncer.SyncExcludedModels(ctx, disabled)
}

func (s *Service) rememberDirectoryIDs(ids []string) {
	if s == nil {
		return
	}
	copied := uniqueSortedStrings(ids)
	s.directoryIDsMu.Lock()
	s.lastDirectoryIDs = copied
	s.directoryIDsMu.Unlock()
}

func (s *Service) snapshotDirectoryIDs() []string {
	if s == nil {
		return nil
	}
	s.directoryIDsMu.Lock()
	defer s.directoryIDsMu.Unlock()
	if len(s.lastDirectoryIDs) == 0 {
		return nil
	}
	out := make([]string, len(s.lastDirectoryIDs))
	copy(out, s.lastDirectoryIDs)
	return out
}

func (s *Service) disabledDirectoryModels(ctx context.Context) ([]string, error) {
	if s == nil || s.store == nil {
		return nil, nil
	}
	rules, err := s.store.ListPricingRules(ctx)
	if err != nil {
		return nil, err
	}
	hidden := hiddenModelsByRules(rules, s.snapshotDirectoryIDs())
	for _, rule := range rules {
		if rule.Enabled || rule.MatchKind != store.MatchExact {
			continue
		}
		if id := strings.TrimSpace(rule.Pattern); id != "" {
			hidden = append(hidden, id)
		}
	}
	return uniqueSortedStrings(hidden), nil
}

func hiddenModelsByRules(rules []store.PricingRule, catalog []string) []string {
	hidden := make([]string, 0, len(catalog))
	for _, id := range catalog {
		if modelDisabledByRules(id, rules) {
			hidden = append(hidden, id)
		}
	}
	return hidden
}

func modelDisabledByRules(model string, rules []store.PricingRule) bool {
	rule, err := store.WinningPricingRule(rules, model)
	if err != nil {
		if errors.Is(err, store.ErrPricingRuleNotFound) {
			trimmed := strings.TrimPrefix(model, "models/")
			if trimmed != model {
				return modelDisabledByRules(trimmed, rules)
			}
			return false
		}
		return false
	}
	return !rule.Enabled
}

// MergeAuthExcludedModels rewrites an auth file's excluded_models so that
// plugin-managed entries track the disabled set while user edits survive.
//
// Ownership model (case-sensitive, exact model IDs):
//   - credit_manager_excluded_models (P): what the plugin added last sync.
//   - credit_manager_removed_models (R): managed models the user deleted from
//     excluded_models; they are never re-added until the user re-adds them.
//   - Everything else in excluded_models is user-owned and preserved.
//
// The merge formula is: userAdded = E \ P, newlyRemoved = P \ E,
// R_next = (R \ E) ∪ newlyRemoved, managed = (D \ R_next) \ userAdded,
// E_next = userAdded ∪ managed.
//
// R is intentionally not pruned against D: when the disabled catalog is
// temporarily empty (cold start before the first model directory snapshot),
// remembered removals must not be forgotten.
//
// An explicit "credit_manager_managed": false opts the file out entirely.
func MergeAuthExcludedModels(raw []byte, disabled []string) ([]byte, bool, error) {
	src := bytes.TrimSpace(raw)
	if len(src) == 0 {
		src = []byte("{}")
	}
	_, fields, err := decodeTopLevelObject(src)
	if err != nil {
		return nil, false, err
	}
	if boolFieldFalse(fields[managedOptOutKey]) {
		return raw, false, nil
	}
	current := uniqueSortedStrings(disabled)
	previous := stringSliceFromRaw(fields[managedExcludedModelsKey])
	removed := stringSliceFromRaw(fields[managedRemovedModelsKey])
	_, hasCanonical := fields["excluded_models"]
	_, hasLegacyAlias := fields["excluded-models"]
	var existing []string
	if hasCanonical {
		existing = stringSliceFromRaw(fields["excluded_models"])
	} else {
		existing = stringSliceFromRaw(fields["excluded-models"])
	}

	userAdded := subtractStrings(existing, previous)
	newlyRemoved := subtractStrings(previous, existing)
	// A model re-added by the user cancels its recorded removal.
	removedNext := uniqueSortedStrings(append(subtractStrings(removed, existing), newlyRemoved...))
	// managed stays disjoint from userAdded so pre-existing user exclusions are
	// never claimed into plugin bookkeeping (which would erase them when the
	// model is later re-enabled globally).
	managed := subtractStrings(subtractStrings(current, removedNext), userAdded)
	next := uniqueSortedStrings(append(userAdded, managed...))

	if stringSlicesEqual(existing, next) && stringSlicesEqual(previous, managed) && stringSlicesEqual(removed, removedNext) &&
		!(len(next) == 0 && (hasCanonical || hasLegacyAlias)) && !hasLegacyAlias {
		return raw, false, nil
	}
	set := map[string]json.RawMessage{}
	remove := []string{}
	if len(next) == 0 {
		remove = append(remove, "excluded_models", "excluded-models")
	} else {
		encoded, err := json.Marshal(next)
		if err != nil {
			return nil, false, err
		}
		set["excluded_models"] = encoded
		remove = append(remove, "excluded-models")
	}
	if len(managed) == 0 {
		remove = append(remove, managedExcludedModelsKey)
	} else {
		encoded, err := json.Marshal(managed)
		if err != nil {
			return nil, false, err
		}
		set[managedExcludedModelsKey] = encoded
	}
	if len(removedNext) == 0 {
		remove = append(remove, managedRemovedModelsKey)
	} else {
		encoded, err := json.Marshal(removedNext)
		if err != nil {
			return nil, false, err
		}
		set[managedRemovedModelsKey] = encoded
	}
	out, err := patchTopLevelJSONFields(src, set, remove)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

// AuthModelCatalog is the host model-registry view used to scope which disabled
// models belong on a given credential.
type AuthModelCatalog struct {
	// CredModels holds lowercase model IDs registered on this credential.
	CredModels map[string]struct{}
	// ProviderModels holds lowercase model IDs registered on any credential of
	// the same provider.
	ProviderModels map[string]struct{}
	// KnownProviders holds normalized provider keys the host knows about; it is
	// used to recognize provider-qualified model IDs like "devin/swe-1-6".
	KnownProviders map[string]struct{}
	// Available is false when the host could not report registered models; only
	// provider-prefix scoping is applied then.
	Available bool
}

// ScopeDisabledModelsForAuth filters the global disabled list down to the
// models relevant for one credential, so provider-specific exclusions never
// leak into unrelated auth files (which would also break CPA's
// oauth-excluded-models inheritance on those files).
//
// Rules, applied per model id:
//   - An explicit provider prefix ("devin/x" where "devin" is a known provider
//     different from this credential's provider) is always dropped. This also
//     cleans up prefix-qualified pollution written by older versions.
//   - Without a host catalog (Available == false), everything else passes.
//   - With a catalog, a model is kept when it is registered on the credential
//     or its provider, or already in credit_manager_excluded_models. Keeping
//     previously managed entries is required: registered catalogs exclude
//     currently-excluded models, so dropping them would oscillate
//     (exclude → unregister → un-exclude) on every sync.
func ScopeDisabledModelsForAuth(raw []byte, provider string, disabled []string, catalog AuthModelCatalog) []string {
	authProvider := NormalizeProviderKey(provider)
	var prevSet map[string]struct{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if _, fields, err := decodeTopLevelObject(raw); err == nil {
			previous := stringSliceFromRaw(fields[managedExcludedModelsKey])
			if len(previous) > 0 {
				prevSet = make(map[string]struct{}, len(previous))
				for _, m := range previous {
					prevSet[strings.ToLower(m)] = struct{}{}
				}
			}
		}
	}
	out := make([]string, 0, len(disabled))
	for _, m := range disabled {
		lower := strings.ToLower(strings.TrimSpace(m))
		if lower == "" {
			continue
		}
		if pfx, rest, ok := strings.Cut(m, "/"); ok && strings.TrimSpace(rest) != "" {
			np := NormalizeProviderKey(pfx)
			if _, known := catalog.KnownProviders[np]; known && np != authProvider {
				continue
			}
		}
		if !catalog.Available {
			out = append(out, m)
			continue
		}
		if _, ok := catalog.CredModels[lower]; ok {
			out = append(out, m)
			continue
		}
		if _, ok := catalog.ProviderModels[lower]; ok {
			out = append(out, m)
			continue
		}
		if _, ok := prevSet[lower]; ok {
			out = append(out, m)
		}
	}
	return out
}

// NormalizeProviderKey lowercases a provider identifier and maps the legacy
// "gemini" file type to its canonical "gemini-cli" provider.
func NormalizeProviderKey(provider string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "gemini" {
		return "gemini-cli"
	}
	return p
}

// boolFieldFalse reports whether raw is the JSON literal false. Anything else
// (null, true, strings, numbers, objects, missing) does not opt out.
func boolFieldFalse(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("false"))
}

func decodeTopLevelObject(raw []byte) ([]string, map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil, nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return nil, nil, errors.New("auth json must be an object")
	}
	fields := map[string]json.RawMessage{}
	order := make([]string, 0)
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		key, _ := keyTok.(string)
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, nil, err
		}
		if _, exists := fields[key]; !exists {
			order = append(order, key)
		}
		fields[key] = value
	}
	// The closing '}' must be present; dec.More() alone tolerates truncated input.
	tok, err = dec.Token()
	if err != nil {
		return nil, nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '}' {
		return nil, nil, errors.New("auth json must end with '}'")
	}
	// dec.More() cannot be used here: it also returns false before ']' and '}'.
	if _, err := dec.Token(); err != io.EOF {
		return nil, nil, errors.New("auth json has trailing data after '}'")
	}
	return order, fields, nil
}

func patchTopLevelJSONFields(raw []byte, set map[string]json.RawMessage, remove []string) ([]byte, error) {
	drop := map[string]struct{}{}
	for _, key := range remove {
		if _, updating := set[key]; updating {
			continue
		}
		drop[key] = struct{}{}
	}
	order, fields, err := decodeTopLevelObject(raw)
	if err != nil {
		return nil, err
	}
	for key := range drop {
		delete(fields, key)
	}
	for key, value := range set {
		fields[key] = value
		found := false
		for _, existing := range order {
			if existing == key {
				found = true
				break
			}
		}
		if !found {
			order = append(order, key)
		}
	}
	kept := make([]string, 0, len(order))
	seen := map[string]struct{}{}
	for _, key := range order {
		if _, gone := drop[key]; gone {
			continue
		}
		if _, ok := fields[key]; !ok {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		kept = append(kept, key)
	}
	var extra []string
	for key := range fields {
		if _, ok := seen[key]; ok {
			continue
		}
		extra = append(extra, key)
	}
	sort.Strings(extra)
	kept = append(kept, extra...)
	return encodeObjectPreserve(kept, fields), nil
}

func encodeObjectPreserve(order []string, fields map[string]json.RawMessage) []byte {
	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	for _, key := range order {
		value, ok := fields[key]
		if !ok {
			continue
		}
		if first {
			first = false
		} else {
			buf.WriteByte(',')
		}
		encodedKey, _ := json.Marshal(key)
		buf.Write(encodedKey)
		buf.WriteByte(':')
		buf.Write(value)
	}
	buf.WriteByte('}')
	return buf.Bytes()
}

func stringSliceFromRaw(raw json.RawMessage) []string {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var asList []string
	if err := json.Unmarshal(raw, &asList); err == nil {
		return uniqueSortedStrings(asList)
	}
	var asAny []any
	if err := json.Unmarshal(raw, &asAny); err == nil {
		out := make([]string, 0, len(asAny))
		for _, item := range asAny {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return uniqueSortedStrings(out)
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return uniqueSortedStrings(strings.Split(asString, ","))
	}
	return nil
}

func subtractStrings(values, remove []string) []string {
	drop := make(map[string]struct{}, len(remove))
	for _, item := range remove {
		drop[item] = struct{}{}
	}
	out := make([]string, 0, len(values))
	for _, item := range values {
		if _, found := drop[item]; !found {
			out = append(out, item)
		}
	}
	return out
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
