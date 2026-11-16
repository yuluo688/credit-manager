package plugin

import (
	"context"
	"net/http"
	"strings"

	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
)

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
