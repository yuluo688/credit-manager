package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/yuluo688/credit-manager/internal/store"
)

func (s *Service) ModelDisabled(ctx context.Context, model string) (bool, error) {
	if s == nil || s.store == nil {
		return false, errors.New("service not open")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return false, nil
	}
	rule, err := s.store.ResolvePricingRule(ctx, model)
	if err == nil {
		return !rule.Enabled, nil
	}
	if errors.Is(err, store.ErrPricingRuleNotFound) {
		trimmed := strings.TrimPrefix(model, "models/")
		if trimmed != model {
			return s.ModelDisabled(ctx, trimmed)
		}
		return false, nil
	}
	return false, err
}

func ParseModelDirectoryIDs(body []byte) []string {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return nil
	}
	ids := make([]string, 0)
	for _, field := range []string{"data", "models"} {
		raw, exists := root[field]
		if !exists {
			continue
		}
		items, isList := raw.([]any)
		if !isList {
			continue
		}
		for _, item := range items {
			if id := modelIDFromEntry(item); id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func (s *Service) HiddenDirectoryModels(ctx context.Context, catalog []string) []string {
	if s == nil || s.store == nil {
		return uniqueSortedStrings(nil)
	}
	rules, err := s.store.ListPricingRules(ctx)
	if err != nil {
		return uniqueSortedStrings(nil)
	}
	hidden := hiddenModelsByRules(rules, catalog)
	for _, rule := range rules {
		if rule.Enabled || rule.MatchKind != store.MatchExact {
			continue
		}
		if id := strings.TrimSpace(rule.Pattern); id != "" {
			hidden = append(hidden, id)
		}
	}
	return uniqueSortedStrings(hidden)
}

func (s *Service) FilterModelDirectory(ctx context.Context, body []byte) ([]byte, bool, error) {
	return s.filterModelDirectory(ctx, body, store.PluginKey{})
}

// FilterModelDirectoryForKey removes globally disabled models and, when the key
// has an allowlist or disables unmatched models, models that key cannot call.
// A zero key leaves the public catalog unchanged aside from disabled models.
func (s *Service) FilterModelDirectoryForKey(ctx context.Context, body []byte, key store.PluginKey) ([]byte, bool, error) {
	return s.filterModelDirectory(ctx, body, key)
}

// IsModelDirectoryBody reports whether body is an OpenAI, Gemini, or Codex model list.
func IsModelDirectoryBody(body []byte) bool {
	root, ok := decodeModelDirectory(body)
	return ok && looksLikeModelDirectory(root)
}

func (s *Service) filterModelDirectory(ctx context.Context, body []byte, key store.PluginKey) ([]byte, bool, error) {
	root, ok := decodeModelDirectory(body)
	if !ok || !looksLikeModelDirectory(root) {
		return body, false, nil
	}
	s.rememberDirectoryIDs(ParseModelDirectoryIDs(body))
	rules, err := s.store.ListPricingRules(ctx)
	if err != nil {
		return nil, false, err
	}
	changed := false
	for _, field := range []string{"data", "models"} {
		raw, exists := root[field]
		if !exists {
			continue
		}
		items, isList := raw.([]any)
		if !isList {
			continue
		}
		kept := make([]any, 0, len(items))
		for _, item := range items {
			id := modelIDFromEntry(item)
			if id == "" {
				kept = append(kept, item)
				continue
			}
			if modelDisabledByRules(id, rules) || !directoryModelVisible(key, id) {
				changed = true
				continue
			}
			kept = append(kept, item)
		}
		root[field] = kept
	}
	if !changed {
		return body, false, nil
	}
	out, err := json.Marshal(root)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func decodeModelDirectory(body []byte) (map[string]any, bool) {
	if len(body) == 0 {
		return nil, false
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false
	}
	root, ok := payload.(map[string]any)
	return root, ok
}

func looksLikeModelDirectory(root map[string]any) bool {
	if obj, _ := root["object"].(string); obj == "list" {
		return true
	}
	if _, hasModels := root["models"]; hasModels {
		if _, hasChoices := root["choices"]; !hasChoices {
			return true
		}
	}
	return false
}

// directoryModelVisible reports whether this key may call id.
// An empty allowlist still means every model, matching reserve. Patterns may
// be written with or without the Gemini "models/" catalog prefix.
func directoryModelVisible(key store.PluginKey, id string) bool {
	if !directoryAllowlistAllows(key, id) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(key.UnmatchedModelsMode), store.UnmatchedModelsDisabled) {
		return true
	}
	if _, ok := store.MatchModelTokenLimit(key.ModelTokenLimits, id); ok {
		return true
	}
	if !strings.HasPrefix(id, "models/") {
		if _, ok := store.MatchModelTokenLimit(key.ModelTokenLimits, "models/"+id); ok {
			return true
		}
	}
	return false
}

func directoryAllowlistAllows(key store.PluginKey, id string) bool {
	if len(key.AllowedModels) == 0 {
		return true
	}
	if store.ModelAllowed(key, id) {
		return true
	}
	if strings.HasPrefix(id, "models/") {
		return store.ModelAllowed(store.PluginKey{AllowedModels: key.AllowedModels}, strings.TrimPrefix(id, "models/"))
	}
	return store.ModelAllowed(store.PluginKey{AllowedModels: key.AllowedModels}, "models/"+id)
}

func modelIDFromEntry(item any) string {
	switch value := item.(type) {
	case string:
		return normalizeModelListID(value)
	case map[string]any:
		for _, key := range []string{"id", "name", "model", "slug"} {
			raw, _ := value[key].(string)
			if id := normalizeModelListID(raw); id != "" {
				return id
			}
		}
	}
	return ""
}

func normalizeModelListID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, "models/")
	return strings.TrimSpace(id)
}
