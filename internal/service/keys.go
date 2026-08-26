package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/keys"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/store"
)

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
	ModelTokenLimits      []store.ModelTokenLimit
	UnmatchedModelsMode   string
	Enabled               *bool
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
		Enabled:               req.Enabled == nil || *req.Enabled,
		ExpiresAt:             req.ExpiresAt,
		QuotaMicroUSD:         req.QuotaMicroUSD,
		DailyQuotaMicroUSD:    req.DailyQuotaMicroUSD,
		WeeklyQuotaMicroUSD:   req.WeeklyQuotaMicroUSD,
		MonthlyQuotaMicroUSD:  req.MonthlyQuotaMicroUSD,
		MaxConcurrentRequests: req.MaxConcurrentRequests,
		AllowedModels:         req.AllowedModels,
		ModelTokenLimits:      req.ModelTokenLimits,
		UnmatchedModelsMode:   req.UnmatchedModelsMode,
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
		ModelTokenLimits:      oldKey.ModelTokenLimits,
		UnmatchedModelsMode:   oldKey.UnmatchedModelsMode,
	})
	if err != nil {
		return store.PluginKey{}, keys.Material{}, err
	}
	return newKey, material, nil
}
