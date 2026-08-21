package management

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func putPricing(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID        string `json:"id"`
		MatchKind string `json:"match_kind"`
		Pattern   string `json:"pattern"`
		Priority  int    `json:"priority"`
		Enabled   *bool  `json:"enabled"`
		Price     struct {
			Input          int64  `json:"input"`
			Output         int64  `json:"output"`
			Reasoning      int64  `json:"reasoning"`
			Cached         int64  `json:"cached"`
			CacheRead      int64  `json:"cache_read"`
			CacheCreation  int64  `json:"cache_creation"`
			AccountingMode string `json:"accounting_mode"`
			BillingMode    string `json:"billing_mode"`
			PerImage       int64  `json:"per_image"`
		} `json:"price"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	enabled := true
	if id := strings.TrimSpace(req.ID); id != "" {
		if existing, err := svc.Store().GetPricingRule(ctx, id); err == nil {
			enabled = existing.Enabled
		} else if !errors.Is(err, store.ErrPricingRuleNotFound) {
			return jsonErr(http.StatusInternalServerError, err.Error()), nil
		}
	}
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rule := store.PricingRule{
		ID: req.ID, MatchKind: store.MatchKind(req.MatchKind), Pattern: req.Pattern, Priority: req.Priority, Enabled: enabled,
		Price: money.PricePerMTok{
			Input: money.MicroUSD(req.Price.Input), Output: money.MicroUSD(req.Price.Output),
			Reasoning: money.MicroUSD(req.Price.Reasoning), Cached: money.MicroUSD(req.Price.Cached),
			CacheRead: money.MicroUSD(req.Price.CacheRead), CacheCreation: money.MicroUSD(req.Price.CacheCreation),
			AccountingMode: strings.TrimSpace(req.Price.AccountingMode),
			BillingMode:    strings.TrimSpace(req.Price.BillingMode),
			PerImage:       money.MicroUSD(req.Price.PerImage),
		},
	}
	if err := svc.Store().PutPricingRule(ctx, rule); err != nil {
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	if err := svc.RefreshModelDirectory(ctx); err != nil {
		return jsonErr(http.StatusBadGateway, err.Error()), nil
	}
	return jsonOK(map[string]any{"id": rule.ID, "saved": true}), nil
}

func setPricingEnabled(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID      string `json:"id"`
		Enabled *bool  `json:"enabled"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	if strings.TrimSpace(req.ID) == "" {
		return jsonErr(http.StatusBadRequest, "id is required"), nil
	}
	if req.Enabled == nil {
		return jsonErr(http.StatusBadRequest, "enabled is required"), nil
	}
	if err := svc.Store().SetPricingRuleEnabled(ctx, req.ID, *req.Enabled); err != nil {
		if errors.Is(err, store.ErrPricingRuleNotFound) {
			return jsonErr(http.StatusNotFound, err.Error()), nil
		}
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	if err := svc.RefreshModelDirectory(ctx); err != nil {
		return jsonErr(http.StatusBadGateway, err.Error()), nil
	}
	return jsonOK(map[string]any{"id": strings.TrimSpace(req.ID), "enabled": *req.Enabled}), nil
}

func listPricing(ctx context.Context, svc *service.Service) (pluginapi.ManagementResponse, error) {
	items, err := svc.Store().ListPricingRules(ctx)
	if err != nil {
		return jsonErr(http.StatusInternalServerError, err.Error()), nil
	}
	return jsonOK(map[string]any{"items": items}), nil
}

func deletePricing(ctx context.Context, svc *service.Service, body []byte) (pluginapi.ManagementResponse, error) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return jsonErr(http.StatusBadRequest, "invalid json"), nil
	}
	if err := svc.Store().DeletePricingRule(ctx, req.ID); err != nil {
		if errors.Is(err, store.ErrPricingRuleNotFound) {
			return jsonErr(http.StatusNotFound, err.Error()), nil
		}
		return jsonErr(http.StatusBadRequest, err.Error()), nil
	}
	if err := svc.RefreshModelDirectory(ctx); err != nil {
		return jsonErr(http.StatusBadGateway, err.Error()), nil
	}
	return jsonOK(map[string]any{"id": req.ID, "deleted": true}), nil
}
