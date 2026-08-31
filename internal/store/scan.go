package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/yuluo688/credit-manager/internal/money"
)

type rowScanner interface{ Scan(...any) error }

func scanCaller(row rowScanner) (Caller, error) {
	var caller Caller
	var quota, settled, held int64
	var enabled int
	var created, updated int64
	if err := row.Scan(&caller.ID, &caller.DisplayName, &quota, &settled, &held, &enabled, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Caller{}, ErrCallerNotFound
		}
		return Caller{}, err
	}
	caller.QuotaMicroUSD = money.MicroUSD(quota)
	caller.SettledSpendMicroUSD = money.MicroUSD(settled)
	caller.HeldAmountMicroUSD = money.MicroUSD(held)
	caller.Enabled = enabled == 1
	caller.CreatedAt, caller.UpdatedAt = fromUnixMilli(created), fromUnixMilli(updated)
	return caller, nil
}

func scanPluginKey(row rowScanner) (PluginKey, error) {
	var key PluginKey
	var enabled int
	var revoked, expires, lastUsed, quota sql.NullInt64
	var dailyQuota, weeklyQuota, monthlyQuota, maxConcurrent, settled, held int64
	var allowedJSON, tokenLimitsJSON, unmatchedMode string
	var created, updated int64
	if err := row.Scan(&key.ID, &key.CallerID, &key.Kid, &key.KeyHash, &key.EncryptedKeyMaterial, &key.PepperID, &key.Fingerprint,
		&key.Label, &key.Principal, &key.CallerScope, &enabled, &revoked, &expires, &lastUsed, &created, &updated,
		&quota, &dailyQuota, &weeklyQuota, &monthlyQuota, &maxConcurrent, &settled, &held, &allowedJSON, &tokenLimitsJSON, &unmatchedMode); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PluginKey{}, ErrPluginKeyNotFound
		}
		return PluginKey{}, err
	}
	key.Enabled = enabled == 1
	if revoked.Valid {
		value := fromUnixMilli(revoked.Int64)
		key.RevokedAt = &value
	}
	if expires.Valid {
		value := fromUnixMilli(expires.Int64)
		key.ExpiresAt = &value
	}
	if lastUsed.Valid {
		value := fromUnixMilli(lastUsed.Int64)
		key.LastUsedAt = &value
	}
	if quota.Valid {
		value := money.MicroUSD(quota.Int64)
		key.QuotaMicroUSD = &value
	}
	key.DailyQuotaMicroUSD = money.MicroUSD(dailyQuota)
	key.WeeklyQuotaMicroUSD = money.MicroUSD(weeklyQuota)
	key.MonthlyQuotaMicroUSD = money.MicroUSD(monthlyQuota)
	key.MaxConcurrentRequests = maxConcurrent
	key.SettledSpendMicroUSD = money.MicroUSD(settled)
	key.HeldAmountMicroUSD = money.MicroUSD(held)
	models, err := unmarshalAllowedModels(allowedJSON)
	if err != nil {
		return PluginKey{}, err
	}
	key.AllowedModels = models
	tokenLimits, err := unmarshalModelTokenLimits(tokenLimitsJSON)
	if err != nil {
		return PluginKey{}, err
	}
	key.ModelTokenLimits = tokenLimits
	key.UnmatchedModelsMode = normalizeUnmatchedModelsMode(unmatchedMode)
	key.CreatedAt, key.UpdatedAt = fromUnixMilli(created), fromUnixMilli(updated)
	return key, nil
}

func marshalAllowedModels(models []string) (string, error) {
	clean := make([]string, 0, len(models))
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		clean = append(clean, m)
	}
	if clean == nil {
		clean = []string{}
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return "", fmt.Errorf("%w: encode allowed models: %v", ErrInvalidArgument, err)
	}
	return string(raw), nil
}

func unmarshalAllowedModels(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}, nil
	}
	var models []string
	if err := json.Unmarshal([]byte(raw), &models); err != nil {
		return nil, fmt.Errorf("%w: decode allowed models: %v", ErrInvalidArgument, err)
	}
	out := make([]string, 0, len(models))
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m != "" {
			out = append(out, m)
		}
	}
	return out, nil
}

func scanPricingRule(row rowScanner) (PricingRule, error) {
	var rule PricingRule
	var enabled int
	var created, updated int64
	var tiersJSON string
	if err := row.Scan(&rule.ID, &rule.MatchKind, &rule.Pattern, &rule.Priority,
		&rule.Price.Input, &rule.Price.Output, &rule.Price.Reasoning, &rule.Price.Cached,
		&rule.Price.CacheRead, &rule.Price.CacheCreation, &rule.Price.AccountingMode,
		&rule.Price.BillingMode, &rule.Price.PerImage, &tiersJSON, &enabled, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PricingRule{}, ErrPricingRuleNotFound
		}
		return PricingRule{}, err
	}
	tiers, err := unmarshalTiers(tiersJSON)
	if err != nil {
		return PricingRule{}, err
	}
	rule.Tiers = tiers
	rule.Enabled = enabled == 1
	rule.CreatedAt, rule.UpdatedAt = fromUnixMilli(created), fromUnixMilli(updated)
	return rule, nil
}

func scanReservation(row rowScanner) (Reservation, error) {
	var reservation Reservation
	var held int64
	var settled, settledAt, releasedAt sql.NullInt64
	var created, updated int64
	if err := row.Scan(&reservation.ID, &reservation.CallerID, &reservation.PluginKeyID, &reservation.IdempotencyKey,
		&reservation.Model, &reservation.RequestTokenEstimate, &held, &settled, &reservation.Status,
		&reservation.RequestSummary, &reservation.SettlementSummary, &created, &updated, &settledAt, &releasedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Reservation{}, ErrReservationNotFound
		}
		return Reservation{}, err
	}
	reservation.HeldMicroUSD = money.MicroUSD(held)
	if settled.Valid {
		value := money.MicroUSD(settled.Int64)
		reservation.SettledMicroUSD = &value
	}
	reservation.CreatedAt, reservation.UpdatedAt = fromUnixMilli(created), fromUnixMilli(updated)
	if settledAt.Valid {
		value := fromUnixMilli(settledAt.Int64)
		reservation.SettledAt = &value
	}
	if releasedAt.Valid {
		value := fromUnixMilli(releasedAt.Int64)
		reservation.ReleasedAt = &value
	}
	return reservation, nil
}

func nullableText(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return strings.TrimSpace(*value)
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableDurationMillis(value *time.Duration) any {
	if value == nil {
		return nil
	}
	return value.Milliseconds()
}

func requireOneRow(result sql.Result, zeroErr error) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return zeroErr
	}
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nowUnixMilli() int64                 { return time.Now().UTC().UnixMilli() }
func fromUnixMilli(value int64) time.Time { return time.UnixMilli(value).UTC() }

func newID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(value[:])
}

// NewID returns a new opaque entity identifier.
func NewID() string { return newID() }
