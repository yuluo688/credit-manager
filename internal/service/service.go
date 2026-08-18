package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yuluo688/credit-manager/internal/config"
	"github.com/yuluo688/credit-manager/internal/keys"
	"github.com/yuluo688/credit-manager/internal/lockfile"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
	"github.com/yuluo688/credit-manager/internal/usageparse"
)

const (
	PluginID      = "credit-manager"
	PluginName    = "CPA Credit Manager"
	PluginVersion = "1.2.0"
	// CallerScopeMetadataKey mirrors sdk/cliproxy/executor.CallerScopeMetadataKey.
	CallerScopeMetadataKey = "caller_scope"
)

const staleCleanupInterval = time.Minute

// Service is the process-wide plugin runtime.
type Service struct {
	cfg                config.Config
	peppers            config.PepperSet
	store              *store.Store
	authMu             sync.Mutex
	authPending        map[string]*pendingAuthCapture
	cleanupMu          sync.Mutex
	lastCleanup        time.Time
	authQuotaMu        sync.RWMutex
	authQuotaSource    AuthQuotaSource
	authQuotaRefreshMu sync.Mutex
}

var current atomic.Pointer[Service]

func Current() *Service { return current.Load() }

func Replace(svc *Service) {
	if old := current.Swap(svc); old != nil {
		_ = old.Close()
	}
}

func Shutdown() {
	if old := current.Swap(nil); old != nil {
		_ = old.Close()
	}
}

func Open(ctx context.Context, cfg config.Config) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	peppers, err := cfg.LoadPeppers()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	dbPath := cfg.DatabasePath()
	st, err := store.OpenLocked(ctx, dbPath, store.OpenOptions{BusyTimeout: cfg.BusyTimeout}, lockfile.New())
	if err != nil {
		return nil, err
	}
	svc := &Service{cfg: cfg, peppers: peppers, store: st}
	if err := svc.ensureBootstrap(ctx); err != nil {
		_ = svc.Close()
		return nil, err
	}
	if _, err := svc.cleanupStaleReservations(ctx, true); err != nil {
		_ = svc.Close()
		return nil, fmt.Errorf("release stale reservations: %w", err)
	}
	return svc, nil
}

func (s *Service) Close() error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.store.Close()
}

func (s *Service) Config() config.Config { return s.cfg }
func (s *Service) Store() *store.Store   { return s.store }
func (s *Service) Peppers() config.PepperSet {
	return s.peppers
}

// SetAuthQuotaSource attaches the host bridge used to inspect auth files and
// make authenticated quota requests. It may be called after Open or Configure.
func (s *Service) SetAuthQuotaSource(source AuthQuotaSource) {
	if s == nil {
		return
	}
	s.authQuotaMu.Lock()
	s.authQuotaSource = source
	s.authQuotaMu.Unlock()
}

// Authenticate verifies a Bearer plugin key and returns a stable principal.
func (s *Service) Authenticate(ctx context.Context, headers http.Header) (principal string, metadata map[string]string, ok bool) {
	raw := bearerToken(headers)
	if raw == "" {
		return "", nil, false
	}
	kid, err := keys.Parse(raw)
	if err != nil {
		return "", nil, false
	}
	key, err := s.store.GetPluginKeyByKid(ctx, kid)
	if err != nil {
		return "", nil, false
	}
	if err := store.EnsurePluginKeyUsable(key, time.Now().UTC()); err != nil {
		return "", nil, false
	}
	if _, ok := keys.Verify(raw, key.KeyHash, key.PepperID, s.peppers); !ok {
		return "", nil, false
	}
	caller, err := s.store.GetCaller(ctx, key.CallerID)
	if err != nil || !caller.Enabled {
		return "", nil, false
	}
	_ = s.store.TouchPluginKeyUsed(ctx, key.ID)
	return key.Principal, map[string]string{
		"plugin":      PluginID,
		"kid":         key.Kid,
		"caller_id":   key.CallerID,
		"fingerprint": key.Fingerprint,
	}, true
}

// LookupPluginKey verifies a plaintext plugin key for its self-service usage page.
// It intentionally returns one generic error for invalid, unavailable, or disabled
// keys so the public endpoint does not expose key state to unauthenticated callers.
func (s *Service) LookupPluginKey(ctx context.Context, raw string) (store.PluginKey, error) {
	raw = strings.TrimSpace(raw)
	kid, err := keys.Parse(raw)
	if err != nil {
		return store.PluginKey{}, keys.ErrInvalidKey
	}
	key, err := s.store.GetPluginKeyByKid(ctx, kid)
	if err != nil {
		return store.PluginKey{}, keys.ErrInvalidKey
	}
	if err := store.EnsurePluginKeyUsable(key, time.Now().UTC()); err != nil {
		return store.PluginKey{}, keys.ErrInvalidKey
	}
	if _, ok := keys.Verify(raw, key.KeyHash, key.PepperID, s.peppers); !ok {
		return store.PluginKey{}, keys.ErrInvalidKey
	}
	caller, err := s.store.GetCaller(ctx, key.CallerID)
	if err != nil || !caller.Enabled {
		return store.PluginKey{}, keys.ErrInvalidKey
	}
	return key, nil
}

func (s *Service) LookupPluginKeyFromHeaders(ctx context.Context, headers http.Header) (store.PluginKey, error) {
	return s.LookupPluginKey(ctx, bearerToken(headers))
}

func bearerToken(headers http.Header) string {
	if headers == nil {
		return ""
	}
	auth := strings.TrimSpace(headers.Get("Authorization"))
	if auth == "" {
		// case-insensitive fallback
		for k, values := range headers {
			if strings.EqualFold(k, "Authorization") && len(values) > 0 {
				auth = strings.TrimSpace(values[0])
				break
			}
		}
	}
	if len(auth) < 8 || !strings.EqualFold(auth[:7], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(auth[7:])
}

// ResolveIdentity finds the plugin key for an authenticated execution request.
func (s *Service) ResolveIdentity(ctx context.Context, headers http.Header, metadata map[string]any) (store.PluginKey, store.Caller, error) {
	if raw := bearerToken(headers); raw != "" {
		if kid, err := keys.Parse(raw); err == nil {
			key, err := s.store.GetPluginKeyByKid(ctx, kid)
			if err == nil {
				if err := store.EnsurePluginKeyUsable(key, time.Now().UTC()); err != nil {
					return store.PluginKey{}, store.Caller{}, err
				}
				caller, err := s.store.GetCaller(ctx, key.CallerID)
				if err != nil {
					return store.PluginKey{}, store.Caller{}, err
				}
				if !caller.Enabled {
					return store.PluginKey{}, store.Caller{}, store.ErrCallerDisabled
				}
				return key, caller, nil
			}
		}
	}
	scope := metadataString(metadata, CallerScopeMetadataKey)
	if scope == "" {
		return store.PluginKey{}, store.Caller{}, fmt.Errorf("%w: missing caller identity", store.ErrInvalidArgument)
	}
	key, err := s.store.GetPluginKeyByCallerScope(ctx, scope)
	if err != nil {
		return store.PluginKey{}, store.Caller{}, err
	}
	if err := store.EnsurePluginKeyUsable(key, time.Now().UTC()); err != nil {
		return store.PluginKey{}, store.Caller{}, err
	}
	caller, err := s.store.GetCaller(ctx, key.CallerID)
	if err != nil {
		return store.PluginKey{}, store.Caller{}, err
	}
	if !caller.Enabled {
		return store.PluginKey{}, store.Caller{}, store.ErrCallerDisabled
	}
	return key, caller, nil
}

type ReservePlan struct {
	Model          string
	InputEstimate  int64
	OutputEstimate int64
	TokenEstimate  int64
	Price          money.PricePerMTok
	PricingRuleID  *string
	Amount         money.MicroUSD
	AllowUnpriced  bool
}

func (s *Service) BuildReservePlan(ctx context.Context, model string, body []byte) (ReservePlan, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		model = extractModel(body)
	}
	if model == "" {
		return ReservePlan{}, fmt.Errorf("%w: model is required", store.ErrInvalidArgument)
	}
	inputEst, outputEst := estimateTokens(body, s.cfg.Limits.DefaultOutputReserve, s.cfg.Limits.MaxTokenEstimate)
	if err := s.cfg.ValidateTokenEstimate(inputEst + outputEst); err != nil {
		return ReservePlan{}, err
	}
	plan := ReservePlan{
		Model:          model,
		InputEstimate:  inputEst,
		OutputEstimate: outputEst,
		TokenEstimate:  inputEst + outputEst,
	}
	rule, err := s.store.ResolvePricingRule(ctx, model)
	switch {
	case err == nil:
		plan.Price = rule.Price
		id := rule.ID
		plan.PricingRuleID = &id
	case errors.Is(err, store.ErrPricingRuleNotFound):
		switch s.cfg.Pricing.UnknownPolicy {
		case config.UnknownPricingDeny:
			return ReservePlan{}, fmt.Errorf("no pricing rule for model %q", model)
		case config.UnknownPricingAllow:
			plan.AllowUnpriced = true
			plan.Amount = 0
			return plan, nil
		case config.UnknownPricingDefault:
			if s.cfg.Pricing.Default == nil {
				return ReservePlan{}, errors.New("default pricing missing")
			}
			plan.Price = money.PricePerMTok{
				Input: money.MicroUSD(s.cfg.Pricing.Default.Input), Output: money.MicroUSD(s.cfg.Pricing.Default.Output),
				Reasoning: money.MicroUSD(s.cfg.Pricing.Default.Reasoning), Cached: money.MicroUSD(s.cfg.Pricing.Default.Cached),
				CacheRead: money.MicroUSD(s.cfg.Pricing.Default.CacheRead), CacheCreation: money.MicroUSD(s.cfg.Pricing.Default.CacheCreation),
			}
		}
	default:
		return ReservePlan{}, err
	}
	cost, err := money.Cost(money.TokenUsage{Input: inputEst, Output: outputEst}, plan.Price)
	if err != nil {
		return ReservePlan{}, err
	}
	plan.Amount = cost
	return plan, nil
}

func (s *Service) Reserve(ctx context.Context, key store.PluginKey, plan ReservePlan, idempotency string) (store.Reservation, error) {
	if _, err := s.cleanupStaleReservations(ctx, false); err != nil {
		return store.Reservation{}, fmt.Errorf("release stale reservations: %w", err)
	}
	if idempotency == "" {
		idempotency = newIdempotency(key.ID, plan.Model, plan.TokenEstimate, plan.Amount)
	}
	return s.store.Reserve(ctx, store.ReserveRequest{
		CallerID:             key.CallerID,
		PluginKeyID:          key.ID,
		IdempotencyKey:       idempotency,
		Model:                plan.Model,
		RequestTokenEstimate: plan.TokenEstimate,
		AmountMicroUSD:       plan.Amount,
		RequestSummary:       fmt.Sprintf("model=%s in=%d out=%d", plan.Model, plan.InputEstimate, plan.OutputEstimate),
	})
}

func (s *Service) cleanupStaleReservations(ctx context.Context, force bool) (int64, error) {
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()
	now := time.Now().UTC()
	if !force && !s.lastCleanup.IsZero() && now.Sub(s.lastCleanup) < staleCleanupInterval {
		return 0, nil
	}
	released, err := s.store.ReleaseStaleReservations(ctx, now.Add(-s.cfg.Stream.StaleReservationTimeout))
	if err != nil {
		return 0, err
	}
	s.lastCleanup = now
	return released, nil
}

func (s *Service) TouchReservation(ctx context.Context, reservationID string) error {
	return s.store.TouchReservation(ctx, reservationID)
}

func (s *Service) SettleFromUsage(ctx context.Context, reservation store.Reservation, plan ReservePlan, parsed usageparse.Result, format string, metrics store.UsageMetrics) error {
	if hostUsage, ok := s.CapturedHostUsage(reservation.ID); ok {
		cost, err := money.CostFor(hostUsage, plan.Price, plan.Model, "")
		if err != nil {
			return err
		}
		if plan.AllowUnpriced {
			cost = 0
		}
		metrics.TokensPerSecond = tokensPerSecond(hostUsage.Output, metrics.GenerationDuration)
		return s.settleWithAuth(ctx, store.Settlement{
			ReservationID:         reservation.ID,
			Model:                 plan.Model,
			PricingRuleID:         plan.PricingRuleID,
			Usage:                 hostUsage,
			CostMicroUSD:          cost,
			EstimatedCostMicroUSD: reservation.HeldMicroUSD,
			Source:                "host_usage",
			Metrics:               metrics,
			SettlementSummary:     "host_usage_callback",
		})
	}
	if parsed.Found {
		cost, err := money.CostFor(parsed.Usage, plan.Price, plan.Model, "")
		if err != nil {
			return err
		}
		if plan.AllowUnpriced {
			cost = 0
		}
		metrics.TokensPerSecond = tokensPerSecond(parsed.Usage.Output, metrics.GenerationDuration)
		return s.settleWithAuth(ctx, store.Settlement{
			ReservationID:         reservation.ID,
			Model:                 plan.Model,
			PricingRuleID:         plan.PricingRuleID,
			Usage:                 parsed.Usage,
			CostMicroUSD:          cost,
			EstimatedCostMicroUSD: reservation.HeldMicroUSD,
			Source:                parsed.Source,
			Metrics:               metrics,
			SettlementSummary:     fmt.Sprintf("format=%s source=%s", format, parsed.Source),
		})
	}
	switch s.cfg.Settlement.MissingUsage {
	case config.MissingUsageRelease:
		s.CancelAuthCapture(reservation.ID)
		_, err := s.store.Release(ctx, reservation.ID, "missing_usage")
		return err
	default:
		usage := money.TokenUsage{Input: plan.InputEstimate, Output: plan.OutputEstimate}
		metrics.TokensPerSecond = tokensPerSecond(usage.Output, metrics.GenerationDuration)
		return s.settleWithAuth(ctx, store.Settlement{
			ReservationID:         reservation.ID,
			Model:                 plan.Model,
			PricingRuleID:         plan.PricingRuleID,
			Usage:                 usage,
			CostMicroUSD:          reservation.HeldMicroUSD,
			EstimatedCostMicroUSD: reservation.HeldMicroUSD,
			Source:                "reserved_fallback",
			Metrics:               metrics,
			SettlementSummary:     "missing_usage_settle_reserved",
		})
	}
}

func (s *Service) ApplyHostUsage(ctx context.Context, ledgerID string, usage money.TokenUsage) error {
	if !usageFound(usage) {
		return nil
	}
	entry, err := s.store.GetUsage(ctx, ledgerID)
	if err != nil {
		return err
	}
	price, err := s.priceForUsage(ctx, entry)
	if err != nil {
		return err
	}
	cost, err := money.CostFor(usage, price, entry.Model, entry.Auth.Provider)
	if err != nil {
		return err
	}
	if price == (money.PricePerMTok{}) && entry.PricingRuleID == nil {
		cost = 0
	}
	return s.store.UpdateUsageDetail(ctx, ledgerID, usage, cost)
}

func (s *Service) priceForUsage(ctx context.Context, entry store.UsageEntry) (money.PricePerMTok, error) {
	if entry.PricingRuleID != nil {
		if rule, err := s.store.GetPricingRule(ctx, *entry.PricingRuleID); err == nil {
			return rule.Price, nil
		} else if !errors.Is(err, store.ErrPricingRuleNotFound) {
			return money.PricePerMTok{}, err
		}
	}
	rule, err := s.store.ResolvePricingRule(ctx, entry.Model)
	if err == nil {
		return rule.Price, nil
	}
	if !errors.Is(err, store.ErrPricingRuleNotFound) {
		return money.PricePerMTok{}, err
	}
	if s.cfg.Pricing.UnknownPolicy == config.UnknownPricingDefault && s.cfg.Pricing.Default != nil {
		return money.PricePerMTok{
			Input: money.MicroUSD(s.cfg.Pricing.Default.Input), Output: money.MicroUSD(s.cfg.Pricing.Default.Output),
			Reasoning: money.MicroUSD(s.cfg.Pricing.Default.Reasoning), Cached: money.MicroUSD(s.cfg.Pricing.Default.Cached),
			CacheRead: money.MicroUSD(s.cfg.Pricing.Default.CacheRead), CacheCreation: money.MicroUSD(s.cfg.Pricing.Default.CacheCreation),
		}, nil
	}
	return money.PricePerMTok{}, nil
}

func (s *Service) settleWithAuth(ctx context.Context, settlement store.Settlement) error {
	if strings.TrimSpace(settlement.LedgerID) == "" {
		settlement.LedgerID = store.NewID()
	}
	settlement.Auth = s.AuthForSettlement(settlement.ReservationID, settlement.LedgerID)
	_, err := s.store.Settle(ctx, settlement)
	return err
}

func tokensPerSecond(output int64, generationDuration *time.Duration) *float64 {
	if output <= 0 || generationDuration == nil || *generationDuration <= 0 {
		return nil
	}
	value := float64(output) / generationDuration.Seconds()
	return &value
}

func (s *Service) Release(ctx context.Context, reservationID, reason string) error {
	s.CancelAuthCapture(reservationID)
	_, err := s.store.Release(ctx, reservationID, reason)
	return err
}

func (s *Service) MintKey(ctx context.Context, callerID, label string, quota money.MicroUSD, expiresAt *time.Time) (store.PluginKey, keys.Material, error) {
	return s.MintKeyWithPolicy(ctx, MintKeyRequest{
		CallerID:      callerID,
		Label:         label,
		ExpiresAt:     expiresAt,
		QuotaMicroUSD: quota,
	})
}

type MintKeyRequest struct {
	CallerID              string
	Label                 string
	ExpiresAt             *time.Time
	QuotaMicroUSD         money.MicroUSD
	DailyQuotaMicroUSD    money.MicroUSD
	WeeklyQuotaMicroUSD   money.MicroUSD
	MonthlyQuotaMicroUSD  money.MicroUSD
	MaxConcurrentRequests int64
	AllowedModels         []string
	KeyMaterial           string
}

func (s *Service) MintKeyWithPolicy(ctx context.Context, req MintKeyRequest) (store.PluginKey, keys.Material, error) {
	if req.QuotaMicroUSD < 0 || req.DailyQuotaMicroUSD < 0 || req.WeeklyQuotaMicroUSD < 0 || req.MonthlyQuotaMicroUSD < 0 || req.MaxConcurrentRequests < 0 {
		return store.PluginKey{}, keys.Material{}, fmt.Errorf("%w: key limits must not be negative", store.ErrInvalidArgument)
	}
	if strings.TrimSpace(req.CallerID) == "" {
		req.CallerID = BootstrapCallerID
	}
	var (
		material keys.Material
		err      error
	)
	if strings.TrimSpace(req.KeyMaterial) == "" {
		material, err = keys.Mint(s.peppers)
	} else {
		material, err = keys.MaterialFromPlaintext(req.KeyMaterial, s.peppers)
	}
	if err != nil {
		return store.PluginKey{}, keys.Material{}, err
	}
	encryptedMaterial, err := keys.EncryptPlaintext(material.Plaintext, s.peppers)
	if err != nil {
		return store.PluginKey{}, keys.Material{}, err
	}
	key, err := s.store.CreatePluginKey(ctx, store.PluginKeySpec{
		CallerID:              req.CallerID,
		Kid:                   material.Kid,
		KeyHash:               material.KeyHash,
		EncryptedKeyMaterial:  encryptedMaterial,
		PepperID:              material.PepperID,
		Fingerprint:           material.Fingerprint,
		Label:                 req.Label,
		Principal:             material.Principal,
		CallerScope:           material.CallerScope,
		Enabled:               true,
		ExpiresAt:             req.ExpiresAt,
		QuotaMicroUSD:         req.QuotaMicroUSD,
		DailyQuotaMicroUSD:    req.DailyQuotaMicroUSD,
		WeeklyQuotaMicroUSD:   req.WeeklyQuotaMicroUSD,
		MonthlyQuotaMicroUSD:  req.MonthlyQuotaMicroUSD,
		MaxConcurrentRequests: req.MaxConcurrentRequests,
		AllowedModels:         req.AllowedModels,
	})
	if err != nil {
		return store.PluginKey{}, keys.Material{}, err
	}
	return key, material, nil
}

// RotateKey creates and enables a credential with the existing key policy while
// atomically revoking the old credential. Historical usage stays attached to it.
func (s *Service) RotateKey(ctx context.Context, keyID, keyMaterial string) (store.PluginKey, keys.Material, error) {
	oldKey, err := s.store.GetPluginKey(ctx, keyID)
	if err != nil {
		return store.PluginKey{}, keys.Material{}, err
	}
	if oldKey.RevokedAt != nil {
		return store.PluginKey{}, keys.Material{}, store.ErrPluginKeyRevoked
	}
	// nil or 0 quota = unlimited; carry forward as 0
	quota := money.MicroUSD(0)
	if oldKey.QuotaMicroUSD != nil {
		quota = *oldKey.QuotaMicroUSD
	}
	var material keys.Material
	if strings.TrimSpace(keyMaterial) == "" {
		material, err = keys.Mint(s.peppers)
	} else {
		material, err = keys.MaterialFromPlaintext(keyMaterial, s.peppers)
	}
	if err != nil {
		return store.PluginKey{}, keys.Material{}, err
	}
	encryptedMaterial, err := keys.EncryptPlaintext(material.Plaintext, s.peppers)
	if err != nil {
		return store.PluginKey{}, keys.Material{}, err
	}
	newKey, err := s.store.RotatePluginKey(ctx, oldKey.ID, store.PluginKeySpec{
		CallerID:              oldKey.CallerID,
		Kid:                   material.Kid,
		KeyHash:               material.KeyHash,
		EncryptedKeyMaterial:  encryptedMaterial,
		PepperID:              material.PepperID,
		Fingerprint:           material.Fingerprint,
		Label:                 oldKey.Label,
		Principal:             material.Principal,
		CallerScope:           material.CallerScope,
		Enabled:               true,
		ExpiresAt:             oldKey.ExpiresAt,
		QuotaMicroUSD:         quota,
		DailyQuotaMicroUSD:    oldKey.DailyQuotaMicroUSD,
		WeeklyQuotaMicroUSD:   oldKey.WeeklyQuotaMicroUSD,
		MonthlyQuotaMicroUSD:  oldKey.MonthlyQuotaMicroUSD,
		MaxConcurrentRequests: oldKey.MaxConcurrentRequests,
		AllowedModels:         oldKey.AllowedModels,
	})
	if err != nil {
		return store.PluginKey{}, keys.Material{}, err
	}
	return newKey, material, nil
}

func estimateTokens(body []byte, defaultOutput, maxTotal int64) (input, output int64) {
	output = defaultOutput
	if body == nil {
		input = 1
	} else {
		// Conservative character-based upper bound: 1 token per rune-ish byte pair, floor 1.
		input = int64(len(body)/2) + 1
	}
	// Prefer explicit protocol limits when present.
	if explicit := extractOutputLimit(body); explicit > 0 {
		output = explicit
	}
	if input > maxTotal {
		input = maxTotal
	}
	if output > maxTotal {
		output = maxTotal
	}
	if input+output > maxTotal {
		// Keep output preference and shrink input.
		if output >= maxTotal {
			output = maxTotal
			input = 0
		} else {
			input = maxTotal - output
		}
	}
	if input <= 0 {
		input = 1
	}
	if output <= 0 {
		output = 1
	}
	return input, output
}

func extractModel(body []byte) string {
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return ""
	}
	if model, ok := root["model"].(string); ok {
		return strings.TrimSpace(model)
	}
	return ""
}

func extractOutputLimit(body []byte) int64 {
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return 0
	}
	for _, key := range []string{"max_tokens", "max_output_tokens", "max_completion_tokens"} {
		if value, ok := root[key]; ok {
			switch typed := value.(type) {
			case float64:
				if typed > 0 {
					return int64(typed)
				}
			case int64:
				if typed > 0 {
					return typed
				}
			case int:
				if typed > 0 {
					return int64(typed)
				}
			}
		}
	}
	if gen, ok := root["generationConfig"].(map[string]any); ok {
		if value, ok := gen["maxOutputTokens"].(float64); ok && value > 0 {
			return int64(value)
		}
	}
	return 0
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
	// try common casing variants
	for k, value := range metadata {
		if strings.EqualFold(k, key) && value != nil {
			if text, ok := value.(string); ok {
				return strings.TrimSpace(text)
			}
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return ""
}

func newIdempotency(keyID, model string, tokens int64, amount money.MicroUSD) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d|%d|%d", keyID, model, tokens, amount, time.Now().UnixNano())))
	return hex.EncodeToString(sum[:16])
}

// EnsureDataDir is exported for configuration validation helpers.
func EnsureDataDir(path string) error {
	return os.MkdirAll(filepath.Clean(path), 0o700)
}

// Guard serializes reconfigure so exclusive DB lock handoff cannot race itself.
var reconfigureMu sync.Mutex

// Configure applies host register/reconfigure YAML.
// Same database path reuses the open store (no second exclusive lock).
// Path changes close the old instance first, then open the new one.
func Configure(ctx context.Context, rawYAML []byte) error {
	reconfigureMu.Lock()
	defer reconfigureMu.Unlock()
	cfg, err := config.ParseYAML(rawYAML)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	peppers, err := cfg.LoadPeppers()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	dbPath := filepath.Clean(cfg.DatabasePath())

	if old := current.Load(); old != nil && filepath.Clean(old.cfg.DatabasePath()) == dbPath {
		// Keep the locked SQLite writer. Opening a second handle deadlocks on *.db.lock.
		next := &Service{cfg: cfg, peppers: peppers, store: old.store}
		next.SetAuthQuotaSource(old.authQuotaSourceValue())
		if err := next.ensureBootstrap(ctx); err != nil {
			return err
		}
		if _, err := next.cleanupStaleReservations(ctx, true); err != nil {
			return fmt.Errorf("release stale reservations: %w", err)
		}
		if !current.CompareAndSwap(old, next) {
			return fmt.Errorf("service replaced concurrently during reconfigure")
		}
		// Leave old.store attached: in-flight callers may still hold *old.
		// Ownership of Close stays with the published Service / Shutdown.
		return nil
	}

	// Different data path (or first start): release the previous exclusive lock first.
	if old := current.Swap(nil); old != nil {
		_ = old.Close()
	}
	st, err := store.OpenLocked(ctx, dbPath, store.OpenOptions{BusyTimeout: cfg.BusyTimeout}, lockfile.New())
	if err != nil {
		return err
	}
	svc := &Service{cfg: cfg, peppers: peppers, store: st}
	if err := svc.ensureBootstrap(ctx); err != nil {
		_ = svc.Close()
		return err
	}
	if _, err := svc.cleanupStaleReservations(ctx, true); err != nil {
		_ = svc.Close()
		return fmt.Errorf("release stale reservations: %w", err)
	}
	current.Store(svc)
	return nil
}
