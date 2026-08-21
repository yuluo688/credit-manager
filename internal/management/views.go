package management

import (
	"errors"
	"time"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
)

func callerView(caller store.Caller) map[string]any {
	return map[string]any{
		"id":           caller.ID,
		"display_name": caller.DisplayName,
		"enabled":      caller.Enabled,
		"created_at":   caller.CreatedAt,
		"updated_at":   caller.UpdatedAt,
	}
}

func auditView(event store.AuditEvent) map[string]any {
	return map[string]any{
		"id":               event.ID,
		"caller_id":        event.CallerID,
		"plugin_key_id":    event.PluginKeyID,
		"reservation_id":   event.ReservationID,
		"event_type":       event.EventType,
		"amount_micro_usd": event.AmountMicroUSD,
		"details_json":     event.DetailsJSON,
		"created_at":       event.CreatedAt,
	}
}

func keyView(key store.PluginKey) map[string]any {
	quota := money.MicroUSD(0)
	if key.QuotaMicroUSD != nil {
		quota = *key.QuotaMicroUSD
	}
	return map[string]any{
		"id":                      key.ID,
		"kid":                     key.Kid,
		"fingerprint":             key.Fingerprint,
		"label":                   key.Label,
		"enabled":                 key.Enabled,
		"quota_micro_usd":         quota,
		"total_quota_micro_usd":   quota,
		"daily_quota_micro_usd":   key.DailyQuotaMicroUSD,
		"weekly_quota_micro_usd":  key.WeeklyQuotaMicroUSD,
		"monthly_quota_micro_usd": key.MonthlyQuotaMicroUSD,
		"max_concurrent_requests": key.MaxConcurrentRequests,
		"settled_spend_micro_usd": key.SettledSpendMicroUSD,
		"held_amount_micro_usd":   key.HeldAmountMicroUSD,
		"remaining_micro_usd":     key.RemainingMicroUSD(),
		"allowed_models":          key.AllowedModels,
		"revoked_at":              key.RevokedAt,
		"expires_at":              key.ExpiresAt,
		"last_used_at":            key.LastUsedAt,
		"created_at":              key.CreatedAt,
	}
}

func validateKeyLimitFields(limits ...*int64) error {
	for _, limit := range limits {
		if limit != nil && *limit < 0 {
			return errors.New("key limits must not be negative")
		}
	}
	return nil
}

func optionalMicroUSD(value *int64) money.MicroUSD {
	if value == nil {
		return 0
	}
	return money.MicroUSD(*value)
}

func optionalInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func usageView(entry store.UsageEntry) map[string]any {
	return map[string]any{
		"id":                       entry.ID,
		"plugin_key_id":            entry.PluginKeyID,
		"auth_id":                  entry.Auth.AuthID,
		"auth_index":               entry.Auth.AuthIndex,
		"auth_name":                entry.Auth.Name,
		"auth_label":               entry.Auth.Label,
		"auth_provider":            entry.Auth.Provider,
		"auth_type":                entry.Auth.Type,
		"auth_email":               entry.Auth.Email,
		"auth_path":                entry.Auth.Path,
		"model":                    entry.Model,
		"pricing_rule_id":          entry.PricingRuleID,
		"input_tokens":             entry.Usage.Input,
		"output_tokens":            entry.Usage.Output,
		"reasoning_tokens":         entry.Usage.Reasoning,
		"cached_tokens":            entry.Usage.Cached,
		"cache_read_tokens":        entry.Usage.CacheRead,
		"cache_creation_tokens":    entry.Usage.CacheCreation,
		"total_tokens":             money.ReportedTotal(entry.Usage),
		"cost_micro_usd":           entry.CostMicroUSD,
		"estimated_cost_micro_usd": entry.EstimatedCostMicroUSD,
		"source":                   entry.Source,
		"tier":                     entry.Metrics.Tier,
		"result":                   entry.Metrics.Result,
		"first_token_latency_ms":   durationMilliseconds(entry.Metrics.FirstTokenLatency),
		"generation_duration_ms":   durationMilliseconds(entry.Metrics.GenerationDuration),
		"tokens_per_second":        entry.Metrics.TokensPerSecond,
		"thinking_intensity":       entry.Metrics.ThinkingIntensity,
		"created_at":               entry.CreatedAt,
	}
}

func durationMilliseconds(value *time.Duration) *int64 {
	if value == nil {
		return nil
	}
	milliseconds := value.Milliseconds()
	return &milliseconds
}
