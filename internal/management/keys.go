package management

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/yuluo688/credit-manager/internal/keys"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func createKey(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		CallerID              string                  `json:"caller_id"`
		Label                 string                  `json:"label"`
		KeyMaterial           string                  `json:"key_material"`
		ExpiresAt             string                  `json:"expires_at"`
		QuotaMicroUSD         *int64                  `json:"quota_micro_usd"`
		TotalQuotaMicroUSD    *int64                  `json:"total_quota_micro_usd"`
		DailyQuotaMicroUSD    *int64                  `json:"daily_quota_micro_usd"`
		WeeklyQuotaMicroUSD   *int64                  `json:"weekly_quota_micro_usd"`
		MonthlyQuotaMicroUSD  *int64                  `json:"monthly_quota_micro_usd"`
		MaxConcurrentRequests *int64                  `json:"max_concurrent_requests"`
		AllowedModels         []string                `json:"allowed_models"`
		ModelTokenLimits      []store.ModelTokenLimit `json:"model_token_limits"`
		UnmatchedModelsMode   string                  `json:"unmatched_models_mode"`
		Enabled               *bool                   `json:"enabled"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	var expires *time.Time
	if strings.TrimSpace(req.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return jsonErr(http.StatusBadRequest, "expires_at must be RFC3339"), nil
		}
		expires = &parsed
	}
	// nil or 0 = unlimited
	quota := money.MicroUSD(0)
	if req.QuotaMicroUSD != nil {
		if *req.QuotaMicroUSD < 0 {
			return jsonErr(http.StatusBadRequest, "quota_micro_usd must not be negative"), nil
		}
		quota = money.MicroUSD(*req.QuotaMicroUSD)
	}
	if req.TotalQuotaMicroUSD != nil {
		if *req.TotalQuotaMicroUSD < 0 {
			return jsonErr(http.StatusBadRequest, "total_quota_micro_usd must not be negative"), nil
		}
		quota = money.MicroUSD(*req.TotalQuotaMicroUSD)
	}
	if err := validateKeyLimitFields(req.DailyQuotaMicroUSD, req.WeeklyQuotaMicroUSD, req.MonthlyQuotaMicroUSD, req.MaxConcurrentRequests); err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	key, material, err := svc.MintKeyWithPolicy(ctx, service.MintKeyRequest{
		CallerID:              req.CallerID,
		Label:                 req.Label,
		KeyMaterial:           req.KeyMaterial,
		ExpiresAt:             expires,
		QuotaMicroUSD:         quota,
		DailyQuotaMicroUSD:    optionalMicroUSD(req.DailyQuotaMicroUSD),
		WeeklyQuotaMicroUSD:   optionalMicroUSD(req.WeeklyQuotaMicroUSD),
		MonthlyQuotaMicroUSD:  optionalMicroUSD(req.MonthlyQuotaMicroUSD),
		MaxConcurrentRequests: optionalInt64(req.MaxConcurrentRequests),
		AllowedModels:         req.AllowedModels,
		ModelTokenLimits:      req.ModelTokenLimits,
		UnmatchedModelsMode:   req.UnmatchedModelsMode,
		Enabled:               req.Enabled,
	})
	if err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	headers := http.Header{}
	headers.Set("Cache-Control", "no-store")
	view := keyView(key)
	view["plaintext"] = material.Plaintext
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       mustJSON(view),
	}, nil
}

func rotateKey(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID          string `json:"id"`
		KeyMaterial string `json:"key_material"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	key, material, err := svc.RotateKey(ctx, req.ID, req.KeyMaterial)
	if err != nil {
		if errors.Is(err, store.ErrPluginKeyNotFound) {
			return jsonErr(http.StatusNotFound, err.Error()), nil
		}
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	headers := http.Header{}
	headers.Set("Cache-Control", "no-store")
	view := keyView(key)
	view["plaintext"] = material.Plaintext
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       mustJSON(view),
	}, nil
}

func listKeys(ctx context.Context, svc *service.Service, query map[string][]string) (pluginapi.ManagementResponse, error) {
	callerID := firstQuery(query, "caller_id")
	pageSize := queryInt(query, "page_size", queryInt(query, "limit", 10))
	if pageSize > 100 {
		pageSize = 100
	}
	page := queryInt(query, "page", 1)
	activeOnly := firstQuery(query, "active_only") == "1"
	total, err := svc.Store().CountPluginKeys(ctx, callerID, activeOnly)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	if totalPages == 0 {
		page = 1
	} else if int64(page) > totalPages {
		page = int(totalPages)
	}
	items, err := svc.Store().ListPluginKeysPage(ctx, callerID, activeOnly, pageSize, (page-1)*pageSize)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, keyView(item))
	}
	return jsonOK(map[string]any{
		"items":       out,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	}), nil
}

func updateKey(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID                    string                  `json:"id"`
		Label                 *string                 `json:"label"`
		Enabled               *bool                   `json:"enabled"`
		QuotaMicroUSD         *int64                  `json:"quota_micro_usd"`
		TotalQuotaMicroUSD    *int64                  `json:"total_quota_micro_usd"`
		DailyQuotaMicroUSD    *int64                  `json:"daily_quota_micro_usd"`
		WeeklyQuotaMicroUSD   *int64                  `json:"weekly_quota_micro_usd"`
		MonthlyQuotaMicroUSD  *int64                  `json:"monthly_quota_micro_usd"`
		MaxConcurrentRequests *int64                  `json:"max_concurrent_requests"`
		AllowedModels         []string                `json:"allowed_models"`
		SetAllowedModels      bool                    `json:"set_allowed_models"`
		ModelTokenLimits      []store.ModelTokenLimit `json:"model_token_limits"`
		SetModelTokenLimits   bool                    `json:"set_model_token_limits"`
		UnmatchedModelsMode   *string                 `json:"unmatched_models_mode"`
		ExpiresAt             string                  `json:"expires_at"`
		ClearExpiresAt        bool                    `json:"clear_expires_at"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	update := store.PluginKeyPolicyUpdate{
		ID:             req.ID,
		Label:          req.Label,
		Enabled:        req.Enabled,
		ClearExpiresAt: req.ClearExpiresAt,
	}
	if req.QuotaMicroUSD != nil {
		q := money.MicroUSD(*req.QuotaMicroUSD)
		update.QuotaMicroUSD = &q
	}
	if req.TotalQuotaMicroUSD != nil {
		q := money.MicroUSD(*req.TotalQuotaMicroUSD)
		update.QuotaMicroUSD = &q
	}
	if err := validateKeyLimitFields(req.DailyQuotaMicroUSD, req.WeeklyQuotaMicroUSD, req.MonthlyQuotaMicroUSD, req.MaxConcurrentRequests); err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	if req.DailyQuotaMicroUSD != nil {
		q := money.MicroUSD(*req.DailyQuotaMicroUSD)
		update.DailyQuotaMicroUSD = &q
	}
	if req.WeeklyQuotaMicroUSD != nil {
		q := money.MicroUSD(*req.WeeklyQuotaMicroUSD)
		update.WeeklyQuotaMicroUSD = &q
	}
	if req.MonthlyQuotaMicroUSD != nil {
		q := money.MicroUSD(*req.MonthlyQuotaMicroUSD)
		update.MonthlyQuotaMicroUSD = &q
	}
	if req.MaxConcurrentRequests != nil {
		q := *req.MaxConcurrentRequests
		update.MaxConcurrentRequests = &q
	}
	if req.SetAllowedModels {
		models := append([]string(nil), req.AllowedModels...)
		update.AllowedModels = &models
	}
	if req.SetModelTokenLimits {
		limits := append([]store.ModelTokenLimit(nil), req.ModelTokenLimits...)
		update.ModelTokenLimits = &limits
	}
	if req.UnmatchedModelsMode != nil {
		mode := *req.UnmatchedModelsMode
		update.UnmatchedModelsMode = &mode
	}
	if strings.TrimSpace(req.ExpiresAt) != "" && !req.ClearExpiresAt {
		parsed, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return jsonErr(http.StatusBadRequest, "expires_at must be RFC3339"), nil
		}
		update.ExpiresAt = &parsed
	}
	key, err := svc.Store().UpdatePluginKeyPolicy(ctx, update)
	if err != nil {
		if errors.Is(err, store.ErrPluginKeyNotFound) {
			return jsonErr(http.StatusNotFound, err.Error()), nil
		}
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	return jsonOK(keyView(key)), nil
}

func revokeKey(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	if err := svc.Store().RevokePluginKey(ctx, req.ID); err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	return jsonOK(map[string]any{"id": req.ID, "revoked": true}), nil
}

func revealKey(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	key, err := svc.Store().GetPluginKey(ctx, req.ID)
	if err != nil {
		if errors.Is(err, store.ErrPluginKeyNotFound) {
			return jsonErr(http.StatusNotFound, err.Error()), nil
		}
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	if len(key.EncryptedKeyMaterial) == 0 {
		return jsonErr(http.StatusConflict, "this key was created before recoverable storage; rotate it to create a viewable replacement"), nil
	}
	plaintext, err := keys.DecryptPlaintext(key.EncryptedKeyMaterial, svc.Peppers())
	if err != nil {
		return jsonErr(http.StatusInternalServerError, "decrypt plugin key: "+err.Error()), nil
	}
	if _, ok := keys.Verify(plaintext, key.KeyHash, key.PepperID, svc.Peppers()); !ok {
		return jsonErr(http.StatusInternalServerError, "stored plugin key material does not match its verification hash"), nil
	}
	headers := http.Header{}
	headers.Set("Cache-Control", "no-store")
	return pluginapi.ManagementResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       mustJSON(map[string]any{"id": key.ID, "plaintext": plaintext}),
	}, nil
}

func resetKeySpend(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID      string   `json:"id"`
		IDs     []string `json:"ids"`
		All     bool     `json:"all"`
		Total   bool     `json:"total"`
		Daily   bool     `json:"daily"`
		Weekly  bool     `json:"weekly"`
		Monthly bool     `json:"monthly"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	ids := append([]string(nil), req.IDs...)
	if strings.TrimSpace(req.ID) != "" {
		ids = append(ids, req.ID)
	}
	reset, err := svc.Store().ResetPluginKeySpend(ctx, ids, req.All, store.SpendResetScopes{
		Total:   req.Total,
		Daily:   req.Daily,
		Weekly:  req.Weekly,
		Monthly: req.Monthly,
	})
	if err != nil {
		if errors.Is(err, store.ErrPluginKeyNotFound) {
			return jsonErr(http.StatusNotFound, err.Error()), nil
		}
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	return jsonOK(map[string]any{"reset": reset, "total": req.Total, "daily": req.Daily, "weekly": req.Weekly, "monthly": req.Monthly}), nil
}

func deleteKeyPermanently(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	if strings.TrimSpace(req.ID) == "" {
		return jsonErr(http.StatusBadRequest, "id is required"), nil
	}
	if err := svc.Store().DeletePluginKey(ctx, req.ID); err != nil {
		if errors.Is(err, store.ErrPluginKeyNotFound) {
			return jsonErr(http.StatusNotFound, err.Error()), nil
		}
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	return jsonOK(map[string]any{"id": req.ID, "deleted": true, "history_retained": true}), nil
}
