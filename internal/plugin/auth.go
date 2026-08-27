package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
	"github.com/yuluo688/credit-manager/internal/usageparse"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func authenticate(raw []byte) ([]byte, error) {
	svc := service.Current()
	if svc == nil {
		return okEnvelope(pluginapi.FrontendAuthResponse{Authenticated: false})
	}
	var req pluginapi.FrontendAuthRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return okEnvelope(pluginapi.FrontendAuthResponse{Authenticated: false})
	}
	// CLIProxyAPI's management center queries the read-only model directory
	// without a client Key. Keep model execution exclusively Key-authenticated.
	// Globally disabled models are stripped by the response interceptor; Key
	// allowlists are enforced at reserve/execute time.
	if publicModelDirectoryRequest(req.Method, req.Path) {
		return okEnvelope(pluginapi.FrontendAuthResponse{
			Authenticated: true,
			Principal:     "credit-manager:model-directory",
			Metadata:      map[string]string{"plugin": service.PluginID, "public_models": "true"},
		})
	}
	principal, meta, ok := svc.Authenticate(context.Background(), req.Headers)
	if !ok {
		return okEnvelope(pluginapi.FrontendAuthResponse{Authenticated: false})
	}
	return okEnvelope(pluginapi.FrontendAuthResponse{
		Authenticated: true,
		Principal:     principal,
		Metadata:      meta,
	})
}

func publicModelDirectoryRequest(method, path string) bool {
	if !strings.EqualFold(strings.TrimSpace(method), http.MethodGet) {
		return false
	}
	switch strings.TrimRight(strings.TrimSpace(path), "/") {
	case "/v1/models", "/v1beta/models":
		return true
	default:
		return false
	}
}

func interceptRequestAfterAuth(raw []byte) ([]byte, error) {
	svc := service.Current()
	if svc == nil {
		return okEnvelope(pluginapi.RequestInterceptResponse{})
	}
	var req pluginapi.RequestInterceptRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	if !isNativeImageProtocol(req.SourceFormat) && !isImageOnlyModel(firstNonEmpty(req.RequestedModel, req.Model)) {
		return okEnvelope(pluginapi.RequestInterceptResponse{})
	}
	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		return okEnvelope(pluginapi.RequestInterceptResponse{})
	}
	ctx := context.Background()
	key, _, err := svc.ResolveIdentity(ctx, req.Headers, req.Metadata)
	if err != nil {
		return okEnvelope(quotaRejectResponse(http.StatusUnauthorized, err.Error()))
	}
	plan, err := svc.BuildReservePlan(ctx, firstNonEmpty(req.RequestedModel, req.Model), req.Body)
	if err != nil {
		if errors.Is(err, store.ErrModelDisabled) {
			return okEnvelope(modelDisabledRejectResponse(err.Error()))
		}
		return okEnvelope(quotaRejectResponse(http.StatusPaymentRequired, err.Error()))
	}
	reservation, err := svc.Reserve(ctx, key, plan, "")
	if err != nil {
		if errors.Is(err, store.ErrModelNotAllowed) {
			return okEnvelope(modelNotAllowedRejectResponse(err.Error()))
		}
		return okEnvelope(quotaRejectResponse(http.StatusTooManyRequests, err.Error()))
	}
	svc.TrackAuthCapture(reservation.ID, plan.Model, req.Model, req.RequestedModel)
	if err := svc.AdmitAuth(ctx, reservation.ID, authIdentityFromIntercept(req)); err != nil {
		_ = svc.Release(ctx, reservation.ID, "auth_concurrency:"+err.Error())
		return okEnvelope(quotaRejectResponse(http.StatusTooManyRequests, err.Error()))
	}
	imageHoldsMu.Lock()
	imageHolds[requestID] = imageHold{reservation: reservation, plan: plan, stopHeart: startReservationHeartbeat(svc, reservation.ID)}
	imageHoldsMu.Unlock()
	return okEnvelope(pluginapi.RequestInterceptResponse{})
}

func completeInterceptedRequest(raw []byte) ([]byte, error) {
	svc := service.Current()
	if svc == nil {
		return okEnvelope(map[string]any{})
	}
	var req pluginapi.RequestCompletion
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	requestID := strings.TrimSpace(req.RequestID)
	imageHoldsMu.Lock()
	hold, ok := imageHolds[requestID]
	if ok {
		delete(imageHolds, requestID)
	}
	imageHoldsMu.Unlock()
	if !ok {
		return okEnvelope(map[string]any{})
	}
	if hold.stopHeart != nil {
		hold.stopHeart()
	}
	ctx := context.Background()
	if req.Outcome != pluginapi.RequestCompletionSucceeded {
		_ = svc.Release(ctx, hold.reservation.ID, "image_"+string(req.Outcome))
		return okEnvelope(map[string]any{})
	}
	metrics := usageMetricsFromRequest(nil, req.StartedAt, req.CompletedAt, resultFromStatus(req.StatusCode))
	_ = svc.SettleFromUsage(ctx, hold.reservation, hold.plan, usageparse.Result{}, firstNonEmpty(req.SourceFormat, "openai-image"), metrics)
	return okEnvelope(map[string]any{})
}

func interceptModelDirectoryResponse(raw []byte) ([]byte, error) {
	svc := service.Current()
	if svc == nil {
		return okEnvelope(pluginapi.ResponseInterceptResponse{})
	}
	var req pluginapi.ResponseInterceptRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	filtered, changed, err := svc.FilterModelDirectory(context.Background(), req.Body)
	if err != nil {
		return nil, err
	}
	if !changed {
		return okEnvelope(pluginapi.ResponseInterceptResponse{})
	}
	return okEnvelope(pluginapi.ResponseInterceptResponse{Body: filtered})
}

func modelNotAllowedRejectResponse(message string) pluginapi.RequestInterceptResponse {
	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{"message": message, "type": "model_not_allowed", "code": "model_not_allowed"},
	})
	return pluginapi.RequestInterceptResponse{
		Terminate:    true,
		StatusCode:   http.StatusForbidden,
		ResponseBody: body,
	}
}

func modelDisabledRejectResponse(message string) pluginapi.RequestInterceptResponse {
	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{"message": message, "type": "model_disabled", "code": "model_disabled"},
	})
	return pluginapi.RequestInterceptResponse{
		Terminate:    true,
		StatusCode:   http.StatusForbidden,
		ResponseBody: body,
	}
}

func authIdentityFromIntercept(req pluginapi.RequestInterceptRequest) store.AuthIdentity {
	meta := req.Metadata
	return store.AuthIdentity{
		AuthID:    firstNonEmpty(metadataString(meta, "selected_auth_id"), metadataString(meta, "auth_id")),
		AuthIndex: firstNonEmpty(metadataString(meta, "selected_auth_index"), metadataString(meta, "auth_index")),
		Provider:  firstNonEmpty(metadataString(meta, "selected_auth_provider"), metadataString(meta, "auth_provider"), metadataString(meta, "provider")),
	}
}

func quotaRejectResponse(status int, message string) pluginapi.RequestInterceptResponse {
	if status <= 0 {
		status = http.StatusForbidden
	}
	body, _ := json.Marshal(map[string]any{
		"error": map[string]any{"message": message, "type": "insufficient_quota", "code": "limit_rejected"},
	})
	return pluginapi.RequestInterceptResponse{
		Terminate:    true,
		StatusCode:   status,
		ResponseBody: body,
	}
}
