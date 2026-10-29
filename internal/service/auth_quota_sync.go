package service

import (
	"context"
	"strings"
	"time"
)

// Background quota sync: periodically refreshes every quota-capable auth so
// window starts/resets are observed close to when they happen. Read-only
// upstream calls only; never consumes reset credits.

const authQuotaSyncSpacing = 3 * time.Second

func (s *Service) authQuotaSyncInterval() time.Duration {
	return s.cfg.AuthQuota.SyncInterval
}

func (s *Service) StartAuthQuotaSync() {
	if s == nil || s.authQuotaSourceValue() == nil || s.authQuotaSyncInterval() <= 0 {
		s.StopAuthQuotaSync()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.warmupMu.Lock()
	previous := s.quotaSyncCancel
	s.quotaSyncCancel = cancel
	s.warmupMu.Unlock()
	if previous != nil {
		previous()
	}
	go s.runAuthQuotaSync(ctx, s.authQuotaSyncInterval())
}

func (s *Service) StopAuthQuotaSync() {
	if s == nil {
		return
	}
	s.warmupMu.Lock()
	cancel := s.quotaSyncCancel
	s.quotaSyncCancel = nil
	s.warmupMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Service) runAuthQuotaSync(ctx context.Context, interval time.Duration) {
	// Let the plugin settle after start/reload before the first pass.
	timer := time.NewTimer(2 * time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		s.syncAllAuthQuotas(ctx, interval, time.Now().UTC())
		timer.Reset(interval)
	}
}

func (s *Service) syncAllAuthQuotas(ctx context.Context, interval time.Duration, now time.Time) {
	_, files, err := s.authQuotaFiles(ctx)
	if err != nil {
		return
	}
	for _, file := range files {
		if ctx.Err() != nil {
			return
		}
		provider := quotaProvider(file.Provider)
		if provider == "" {
			continue
		}
		id := first(file.ID, file.Name, file.AuthIndex)
		if !authQuotaSyncDue(s, ctx, provider, id, interval, now) {
			continue
		}
		_, _ = s.RefreshAuthQuota(ctx, "", provider, id, file.AuthIndex)
		select {
		case <-ctx.Done():
			return
		case <-time.After(authQuotaSyncSpacing):
		}
	}
}

// authQuotaSyncDue skips auths refreshed recently (manual refresh, warmup) and
// backs off after rate-limit / auth errors.
func authQuotaSyncDue(s *Service, ctx context.Context, provider, id string, interval time.Duration, now time.Time) bool {
	saved, err := s.store.GetAuthQuotaSnapshot(ctx, provider, id)
	if err != nil {
		return true
	}
	if !saved.LastAttemptAt.IsZero() && now.Sub(saved.LastAttemptAt) < interval*3/4 {
		return false
	}
	if saved.LastError != "" && saved.LastErrorAt != nil {
		msg := strings.ToLower(saved.LastError)
		backoff := time.Duration(0)
		switch {
		case strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many"):
			backoff = 4 * interval
		case strings.Contains(msg, "401") || strings.Contains(msg, "403") || strings.Contains(msg, "token"):
			backoff = 8 * interval
		}
		if backoff > 0 && now.Sub(*saved.LastErrorAt) < backoff {
			return false
		}
	}
	return true
}
