package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/yuluo688/credit-manager/internal/store"
)

const managedExcludedModelsKey = "credit_manager_excluded_models"

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

func MergeAuthExcludedModels(raw []byte, disabled []string) ([]byte, bool, error) {
	src := bytes.TrimSpace(raw)
	if len(src) == 0 {
		src = []byte("{}")
	}
	_, fields, err := decodeTopLevelObject(src)
	if err != nil {
		return nil, false, err
	}
	current := uniqueSortedStrings(disabled)
	previous := stringSliceFromRaw(fields[managedExcludedModelsKey])
	existing := stringSliceFromRaw(fields["excluded_models"])
	if len(existing) == 0 {
		existing = stringSliceFromRaw(fields["excluded-models"])
	}
	user := subtractStrings(existing, previous)
	next := uniqueSortedStrings(append(user, current...))
	if stringSlicesEqual(existing, next) && stringSlicesEqual(previous, current) {
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
	if len(current) == 0 {
		remove = append(remove, managedExcludedModelsKey)
	} else {
		encoded, err := json.Marshal(current)
		if err != nil {
			return nil, false, err
		}
		set[managedExcludedModelsKey] = encoded
	}
	out, err := patchTopLevelJSONFields(src, set, remove)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
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
