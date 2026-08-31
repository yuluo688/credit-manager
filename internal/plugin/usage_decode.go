package plugin

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
	applyUsageTokenBreakdown(&record.Detail, firstJSONObject(detail, "TokenBreakdown", "token_breakdown"))
	if !hostUsageRecordUseful(record) {
		return pluginapi.UsageRecord{}, false
	}
	return record, true
}

func applyUsageTokenBreakdown(detail *pluginapi.UsageDetail, breakdown map[string]json.RawMessage) {
	if detail == nil || len(breakdown) == 0 {
		return
	}
	input := firstJSONObject(breakdown, "Input", "input")
	output := firstJSONObject(breakdown, "Output", "output")
	if detail.InputTokens == 0 {
		detail.InputTokens = firstJSONInt64(input, "TotalTokens", "total_tokens")
	}
	if detail.OutputTokens == 0 {
		detail.OutputTokens = firstJSONInt64(output, "TotalTokens", "total_tokens")
	}
	if detail.ReasoningTokens == 0 {
		detail.ReasoningTokens = firstJSONInt64(output, "ReasoningTokens", "reasoning_tokens")
	}
	if detail.CacheReadTokens == 0 {
		detail.CacheReadTokens = firstJSONInt64(input, "CacheReadTokens", "cache_read_tokens")
	}
	if detail.CacheCreationTokens == 0 {
		detail.CacheCreationTokens = firstJSONInt64(input, "CacheWriteTokens", "cache_write_tokens")
	}
	if detail.TotalTokens == 0 {
		detail.TotalTokens = firstJSONInt64(breakdown, "TotalTokens", "total_tokens")
	}
}

func hostServiceTier(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(raw, &root) != nil {
		return ""
	}
	if tier := firstJSONString(root, "ServiceTier", "service_tier", "Tier", "tier"); tier != "" {
		return tier
	}
	detail := firstJSONObject(root, "Detail", "detail")
	return firstJSONString(detail, "ServiceTier", "service_tier", "Tier", "tier")
}

func hostUsageRecordUseful(record pluginapi.UsageRecord) bool {
	if strings.TrimSpace(record.Model) != "" || strings.TrimSpace(record.Alias) != "" {
		return true
	}
	if strings.TrimSpace(record.AuthID) != "" || strings.TrimSpace(record.AuthIndex) != "" {
		return true
	}
	return hostUsageFound(usageFromHostRecord(record))
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
			if json.Unmarshal(value, &result) == nil && result >= 0 {
				return result
			}
			var numbered float64
			if json.Unmarshal(value, &numbered) == nil && numbered >= 0 {
				return int64(numbered)
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
