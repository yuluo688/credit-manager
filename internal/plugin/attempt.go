package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
)

const maxAttemptSummaryDetailRunes = 512

// formatAttemptSummary builds a bounded, single-line summary preserving
// failure details without letting large upstream responses bloat SQLite.
func formatAttemptSummary(prefix, detail string) string {
	prefix = strings.TrimSpace(prefix)
	clean := strings.Join(strings.Fields(detail), " ")
	if clean == "" {
		return prefix
	}
	if utf8.RuneCountInString(clean) > maxAttemptSummaryDetailRunes {
		runes := []rune(clean)
		clean = string(runes[:maxAttemptSummaryDetailRunes]) + "..."
	}
	if prefix == "" {
		return clean
	}
	return prefix + ":" + clean
}

func formatAttemptError(prefix string, err error) string {
	if err == nil {
		return strings.TrimSpace(prefix)
	}
	return formatAttemptSummary(prefix, err.Error())
}

func httpStatusDetail(statusCode int) string {
	if statusCode < http.StatusBadRequest {
		return ""
	}
	return fmt.Sprintf("http_%d", statusCode)
}

func jsonStringField(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		if raw, ok := obj[key].(string); ok && strings.TrimSpace(raw) != "" {
			return strings.TrimSpace(raw)
		}
	}
	return ""
}

func extractErrorSummary(body []byte, statusCode int) string {
	var payload map[string]any
	if len(body) > 0 && json.Unmarshal(body, &payload) == nil {
		var code, message string
		if errObj, ok := payload["error"].(map[string]any); ok {
			code = jsonStringField(errObj, "code", "type")
			message = jsonStringField(errObj, "message")
		}
		if message == "" {
			message = jsonStringField(payload, "message", "error")
		}
		if code != "" && message != "" {
			return code + ": " + message
		}
		if message != "" {
			return message
		}
		if code != "" {
			return code
		}
	}
	return httpStatusDetail(statusCode)
}

// settleAttemptOrRelease writes a zero-cost usage_ledger row for a post-reserve
// failure. Falls back to Release only if settle itself cannot complete.
func settleAttemptOrRelease(ctx context.Context, svc *service.Service, reservation store.Reservation, model, result, source, summary string) {
	if svc == nil {
		return
	}
	if err := svc.SettleZeroCostAttempt(ctx, reservation, model, service.AttemptMetrics(result), source, summary); err != nil {
		_ = svc.Release(ctx, reservation.ID, summary)
	}
}

// recordRejectedAttempt best-effort records a pre-reserve rejection into usage stats.
func recordRejectedAttempt(ctx context.Context, svc *service.Service, key store.PluginKey, model, result, source, summary string) {
	if svc == nil || strings.TrimSpace(key.ID) == "" {
		return
	}
	_ = svc.RecordRejectedAttempt(ctx, key, model, result, source, summary)
}

func recordRejectedFromHeaders(ctx context.Context, svc *service.Service, headers map[string][]string, metadata map[string]any, model, result, source, summary string) {
	if svc == nil {
		return
	}
	key, _, err := svc.ResolveIdentity(ctx, http.Header(headers), nil, metadata)
	if err != nil {
		return
	}
	recordRejectedAttempt(ctx, svc, key, model, result, source, summary)
}
