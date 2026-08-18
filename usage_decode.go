package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func decodeHostUsageRecord(raw []byte) (pluginapi.UsageRecord, bool) {
	if len(raw) == 0 {
		return pluginapi.UsageRecord{}, false
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(raw, &root) != nil {
		return pluginapi.UsageRecord{}, false
	}
	detail := firstJSONObject(root, "Detail", "detail")
	record := pluginapi.UsageRecord{
		Provider:     firstJSONString(root, "Provider", "provider"),
		ExecutorType: firstJSONString(root, "ExecutorType", "executor_type"),
		Model:        firstJSONString(root, "Model", "model"),
		Alias:        firstJSONString(root, "Alias", "alias"),
		APIKey:       firstJSONString(root, "APIKey", "api_key"),
		AuthID:       firstJSONString(root, "AuthID", "auth_id"),
		AuthIndex:    firstJSONString(root, "AuthIndex", "auth_index"),
		AuthType:     firstJSONString(root, "AuthType", "auth_type"),
		Source:       firstJSONString(root, "Source", "source"),
		RequestedAt:  firstJSONTime(root, "RequestedAt", "requested_at"),
		Detail: pluginapi.UsageDetail{
			InputTokens:         firstJSONInt64(detail, "InputTokens", "input_tokens"),
			OutputTokens:        firstJSONInt64(detail, "OutputTokens", "output_tokens"),
			ReasoningTokens:     firstJSONInt64(detail, "ReasoningTokens", "reasoning_tokens"),
			CachedTokens:        firstJSONInt64(detail, "CachedTokens", "cached_tokens"),
			CacheReadTokens:     firstJSONInt64(detail, "CacheReadTokens", "cache_read_tokens"),
			CacheCreationTokens: firstJSONInt64(detail, "CacheCreationTokens", "cache_creation_tokens"),
			TotalTokens:         firstJSONInt64(detail, "TotalTokens", "total_tokens"),
		},
	}
	if !hostUsageRecordUseful(record) {
		return pluginapi.UsageRecord{}, false
	}
	return record, true
}

func hostUsageRecordUseful(record pluginapi.UsageRecord) bool {
	if strings.TrimSpace(record.Model) != "" || strings.TrimSpace(record.Alias) != "" {
		return true
	}
	if strings.TrimSpace(record.AuthID) != "" || strings.TrimSpace(record.AuthIndex) != "" {
		return true
	}
	usage := usageFromHostRecord(record)
	return usage.Input > 0 || usage.Output > 0 || usage.Reasoning > 0 || usage.Cached > 0 || usage.CacheRead > 0 || usage.CacheCreation > 0
}

func firstJSONObject(root map[string]json.RawMessage, keys ...string) map[string]json.RawMessage {
	for _, key := range keys {
		if value, ok := root[key]; ok {
			var result map[string]json.RawMessage
			if json.Unmarshal(value, &result) == nil {
				return result
			}
		}
	}
	return map[string]json.RawMessage{}
}

func firstJSONString(root map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		if value, ok := root[key]; ok {
			var result string
			if json.Unmarshal(value, &result) == nil && strings.TrimSpace(result) != "" {
				return strings.TrimSpace(result)
			}
		}
	}
	return ""
}

func firstJSONInt64(root map[string]json.RawMessage, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := root[key]; ok {
			var result int64
			if json.Unmarshal(value, &result) == nil {
				return result
			}
		}
	}
	return 0
}

func firstJSONTime(root map[string]json.RawMessage, keys ...string) time.Time {
	for _, key := range keys {
		if value, ok := root[key]; ok {
			var result time.Time
			if json.Unmarshal(value, &result) == nil && !result.IsZero() {
				return result
			}
		}
	}
	return time.Time{}
}
