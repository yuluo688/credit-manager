package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func handleUsage(raw []byte) ([]byte, error) {
	svc := service.Current()
	if svc == nil || len(raw) == 0 {
		return okEnvelope(map[string]any{})
	}
	record, ok := decodeHostUsageRecord(raw)
	if !ok {
		return okEnvelope(map[string]any{})
	}
	auth := authIdentityFromUsage(record)
	if !auth.Empty() {
		auth = enrichAuthIdentity(auth)
	}
	usage := usageFromHostRecord(record)
	ledgerID, matched := svc.ObserveHostUsage(record.RequestedAt, auth, usage, record.Model, record.Alias)
	if matched && strings.TrimSpace(ledgerID) == "" {
		return okEnvelope(map[string]any{})
	}
	if strings.TrimSpace(ledgerID) == "" {
		if entry, found, err := svc.Store().FindRecentFallback(context.Background(), []string{record.Model, record.Alias}, 15*time.Minute); err == nil && found {
			ledgerID = entry.ID
		}
	}
	if strings.TrimSpace(ledgerID) != "" {
		if !auth.Empty() {
			_ = svc.Store().UpdateUsageAuth(context.Background(), ledgerID, auth)
		}
		if hostUsageFound(usage) {
			_ = svc.ApplyHostUsage(context.Background(), ledgerID, usage)
		}
	}
	return okEnvelope(map[string]any{})
}

func usageFromHostRecord(record pluginapi.UsageRecord) money.TokenUsage {
	return money.TokenUsage{
		Input:         record.Detail.InputTokens,
		Output:        record.Detail.OutputTokens,
		Reasoning:     record.Detail.ReasoningTokens,
		Cached:        record.Detail.CachedTokens,
		CacheRead:     record.Detail.CacheReadTokens,
		CacheCreation: record.Detail.CacheCreationTokens,
		ReportedTotal: record.Detail.TotalTokens,
	}
}

func hostUsageFound(usage money.TokenUsage) bool {
	return usage.HasTokens()
}

func authIdentityFromUsage(record pluginapi.UsageRecord) store.AuthIdentity {
	return store.AuthIdentity{
		AuthID:    strings.TrimSpace(record.AuthID),
		AuthIndex: strings.TrimSpace(record.AuthIndex),
		Type:      strings.TrimSpace(record.AuthType),
		Provider:  strings.TrimSpace(record.Provider),
	}
}

func enrichAuthIdentity(auth store.AuthIdentity) store.AuthIdentity {
	if strings.TrimSpace(auth.AuthIndex) != "" {
		if enriched, ok := authIdentityFromHostIndex(auth.AuthIndex); ok {
			return mergeAuthIdentity(auth, enriched)
		}
	}
	if strings.TrimSpace(auth.AuthID) != "" {
		if enriched, ok := authIdentityFromHostID(auth.AuthID); ok {
			return mergeAuthIdentity(auth, enriched)
		}
	}
	return auth
}

func authIdentityFromHostIndex(authIndex string) (store.AuthIdentity, bool) {
	raw, err := callHost(pluginabi.MethodHostAuthGetRuntime, pluginapi.HostAuthGetRequest{AuthIndex: authIndex})
	if err != nil || len(raw) == 0 {
		return store.AuthIdentity{}, false
	}
	var resp pluginapi.HostAuthGetRuntimeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return store.AuthIdentity{}, false
	}
	identity := authIdentityFromHostEntry(resp.Auth)
	if identity.Empty() {
		return store.AuthIdentity{}, false
	}
	return identity, true
}

func authIdentityFromHostID(authID string) (store.AuthIdentity, bool) {
	raw, err := callHost(pluginabi.MethodHostAuthList, map[string]any{})
	if err != nil || len(raw) == 0 {
		return store.AuthIdentity{}, false
	}
	var resp struct {
		Files []pluginapi.HostAuthFileEntry `json:"files"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return store.AuthIdentity{}, false
	}
	authID = strings.TrimSpace(authID)
	for _, entry := range resp.Files {
		if strings.TrimSpace(entry.ID) == authID || strings.TrimSpace(entry.AuthIndex) == authID || strings.TrimSpace(entry.Name) == authID {
			return authIdentityFromHostEntry(entry), true
		}
	}
	return store.AuthIdentity{}, false
}

func authIdentityFromHostEntry(entry pluginapi.HostAuthFileEntry) store.AuthIdentity {
	return store.AuthIdentity{
		AuthID:    firstNonEmpty(entry.ID, entry.Name),
		AuthIndex: strings.TrimSpace(entry.AuthIndex),
		Name:      firstNonEmpty(entry.Name, entry.Path),
		Label:     firstNonEmpty(entry.Label, entry.Account, entry.Email),
		Provider:  strings.TrimSpace(entry.Provider),
		Type:      strings.TrimSpace(entry.Type),
		Email:     strings.TrimSpace(entry.Email),
		Path:      strings.TrimSpace(entry.Path),
	}
}

func mergeAuthIdentity(base, extra store.AuthIdentity) store.AuthIdentity {
	return store.AuthIdentity{
		AuthID:    firstNonEmpty(base.AuthID, extra.AuthID),
		AuthIndex: firstNonEmpty(base.AuthIndex, extra.AuthIndex),
		Name:      firstNonEmpty(extra.Name, base.Name),
		Label:     firstNonEmpty(extra.Label, base.Label),
		Provider:  firstNonEmpty(extra.Provider, base.Provider),
		Type:      firstNonEmpty(extra.Type, base.Type),
		Email:     firstNonEmpty(extra.Email, base.Email),
		Path:      firstNonEmpty(extra.Path, base.Path),
	}
}

func requestBody(req pluginapi.ExecutorRequest) []byte {
	if len(req.OriginalRequest) > 0 {
		return req.OriginalRequest
	}
	return req.Payload
}

// OpenAI Chat Completions SSE streams omit the terminal usage chunk unless
// include_usage is requested. Force it so AI-provider streams still settle
// official tokens, while preserving other client stream_options.
//
// Responses API (/v1/responses, openai-response, codex) rejects
// stream_options.include_usage with HTTP 400. Usage for those streams arrives
// on response.completed without this flag.
func requestBodyWithStreamUsage(body []byte, sourceFormat, outputFormat string) []byte {
	var payload map[string]any
	if len(body) == 0 || json.Unmarshal(body, &payload) != nil {
		return body
	}
	if !supportsChatCompletionsStreamUsage(sourceFormat, outputFormat) || isResponsesAPIPayload(payload) {
		return stripStreamUsageOption(body, payload)
	}
	options, ok := payload["stream_options"].(map[string]any)
	if !ok {
		options = make(map[string]any)
		payload["stream_options"] = options
	}
	options["include_usage"] = true
	updated, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return updated
}

func supportsChatCompletionsStreamUsage(formats ...string) bool {
	for _, format := range formats {
		format = strings.ToLower(strings.TrimSpace(format))
		if format == "" {
			continue
		}
		if strings.Contains(format, "claude") ||
			strings.Contains(format, "gemini") ||
			strings.Contains(format, "response") ||
			strings.Contains(format, "codex") ||
			strings.Contains(format, "interaction") ||
			strings.Contains(format, "image") ||
			strings.Contains(format, "video") {
			return false
		}
	}
	return true
}

func isResponsesAPIPayload(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	if _, hasInput := payload["input"]; hasInput {
		return true
	}
	switch typ := payload["type"].(type) {
	case string:
		typ = strings.ToLower(strings.TrimSpace(typ))
		if typ == "response.create" || typ == "response.append" || strings.HasPrefix(typ, "response.") {
			return true
		}
	}
	if nested, ok := payload["response"].(map[string]any); ok {
		if _, hasInput := nested["input"]; hasInput {
			return true
		}
	}
	return false
}

func stripStreamUsageOption(original []byte, payload map[string]any) []byte {
	options, ok := payload["stream_options"].(map[string]any)
	if !ok {
		return original
	}
	if _, exists := options["include_usage"]; !exists {
		return original
	}
	delete(options, "include_usage")
	if len(options) == 0 {
		delete(payload, "stream_options")
	}
	updated, err := json.Marshal(payload)
	if err != nil {
		return original
	}
	return updated
}

func usageMetricsFromRequest(body []byte, startedAt, completedAt time.Time, result string) store.UsageMetrics {
	return usageMetrics(body, startedAt, time.Time{}, completedAt, result)
}

func usageMetricsFromStream(body []byte, startedAt, firstChunkAt, completedAt time.Time, result string) store.UsageMetrics {
	return usageMetrics(body, startedAt, firstChunkAt, completedAt, result)
}

func usageMetrics(body []byte, startedAt, firstChunkAt, completedAt time.Time, result string) store.UsageMetrics {
	metrics := store.UsageMetrics{
		Tier:              optionalString(requestString(body, "service_tier", "tier")),
		Result:            optionalString(result),
		ThinkingIntensity: optionalString(requestNestedString(body, []string{"reasoning", "effort"}, []string{"thinking", "effort"}, []string{"reasoning_effort"})),
	}
	if !startedAt.IsZero() && !completedAt.IsZero() && !completedAt.Before(startedAt) {
		generationDuration := completedAt.Sub(startedAt)
		metrics.GenerationDuration = &generationDuration
	}
	if !startedAt.IsZero() && !firstChunkAt.IsZero() && !firstChunkAt.Before(startedAt) {
		firstTokenLatency := firstChunkAt.Sub(startedAt)
		metrics.FirstTokenLatency = &firstTokenLatency
	}
	return metrics
}

func requestString(body []byte, keys ...string) string {
	var value map[string]any
	if len(body) == 0 || json.Unmarshal(body, &value) != nil {
		return ""
	}
	for _, key := range keys {
		if text, ok := value[key].(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func requestNestedString(body []byte, paths ...[]string) string {
	var value any
	if len(body) == 0 || json.Unmarshal(body, &value) != nil {
		return ""
	}
	for _, path := range paths {
		current := value
		for _, key := range path {
			object, ok := current.(map[string]any)
			if !ok {
				current = nil
				break
			}
			current = object[key]
		}
		if text, ok := current.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func resultFromStatus(status int) string {
	if status >= http.StatusBadRequest {
		return "failed"
	}
	return "success"
}

func bearerPresent(headers http.Header) bool {
	if headers == nil {
		return false
	}
	auth := headers.Get("Authorization")
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(auth)), "bearer ")
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
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
