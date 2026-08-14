package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef int (*cliproxy_host_call_fn)(void*, const char*, const uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_host_free_fn)(void*, size_t);

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	cliproxy_host_call_fn call;
	cliproxy_host_free_fn free_buffer;
} cliproxy_host_api;

typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);

typedef struct {
	uint32_t abi_version;
	cliproxy_plugin_call_fn call;
	cliproxy_plugin_free_fn free_buffer;
	cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

extern int cliproxyPluginCall(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cliproxyPluginFree(void*, size_t);
extern void cliproxyPluginShutdown(void);

static const cliproxy_host_api* stored_host;

static void store_host_api(const cliproxy_host_api* host) {
	stored_host = host;
}

static int call_host_api(const char* method, const uint8_t* request, size_t request_len, cliproxy_buffer* response) {
	if (stored_host == NULL || stored_host->call == NULL) {
		return 1;
	}
	return stored_host->call(stored_host->host_ctx, method, request, request_len, response);
}

static void free_host_buffer(void* ptr, size_t len) {
	if (stored_host != NULL && stored_host->free_buffer != NULL && ptr != NULL) {
		stored_host->free_buffer(ptr, len);
	}
}
*/
import "C"

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unsafe"

	"github.com/yuluo688/credit-manager/internal/management"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
	"github.com/yuluo688/credit-manager/internal/usageparse"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *envelopeError  `json:"error,omitempty"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type lifecycleRequest struct {
	ConfigYAML []byte `json:"config_yaml"`
}

type registration struct {
	SchemaVersion uint32                 `json:"schema_version"`
	Metadata      pluginapi.Metadata     `json:"metadata"`
	Capabilities  registrationCapability `json:"capabilities"`
}

type registrationCapability struct {
	FrontendAuthProvider          bool     `json:"frontend_auth_provider"`
	FrontendAuthProviderExclusive bool     `json:"frontend_auth_provider_exclusive"`
	ModelRouter                   bool     `json:"model_router"`
	Executor                      bool     `json:"executor"`
	ExecutorModelScope            string   `json:"executor_model_scope"`
	ExecutorInputFormats          []string `json:"executor_input_formats"`
	ExecutorOutputFormats         []string `json:"executor_output_formats"`
	UsagePlugin                   bool     `json:"usage_plugin"`
	ManagementAPI                 bool     `json:"management_api"`
}

type rpcModelRouteRequest struct {
	pluginapi.ModelRouteRequest
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

type rpcExecutorRequest struct {
	pluginapi.ExecutorRequest
	StreamID       string `json:"stream_id,omitempty"`
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

type hostModelExecutionRequest struct {
	pluginapi.HostModelExecutionRequest
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

type rpcStreamEmitRequest struct {
	StreamID string `json:"stream_id"`
	Payload  []byte `json:"payload,omitempty"`
	Error    string `json:"error,omitempty"`
}

type rpcStreamCloseRequest struct {
	StreamID string `json:"stream_id"`
	Error    string `json:"error,omitempty"`
}

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(host *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if plugin == nil {
		return 1
	}
	C.store_host_api(host)
	plugin.abi_version = C.uint32_t(pluginabi.ABIVersion)
	plugin.call = C.cliproxy_plugin_call_fn(C.cliproxyPluginCall)
	plugin.free_buffer = C.cliproxy_plugin_free_fn(C.cliproxyPluginFree)
	plugin.shutdown = C.cliproxy_plugin_shutdown_fn(C.cliproxyPluginShutdown)
	return 0
}

//export cliproxyPluginCall
func cliproxyPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response != nil {
		response.ptr = nil
		response.len = 0
	}
	if method == nil {
		writeResponse(response, errorEnvelope("invalid_method", "method is required"))
		return 1
	}
	var requestBytes []byte
	if request != nil && requestLen > 0 {
		requestBytes = C.GoBytes(unsafe.Pointer(request), C.int(requestLen))
	}
	raw, errHandle := handleMethod(C.GoString(method), requestBytes)
	if errHandle != nil {
		writeResponse(response, errorEnvelope("plugin_error", errHandle.Error()))
		return 1
	}
	writeResponse(response, raw)
	return 0
}

//export cliproxyPluginFree
func cliproxyPluginFree(ptr unsafe.Pointer, _ C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
}

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() {
	service.Shutdown()
}

func handleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case pluginabi.MethodPluginRegister, pluginabi.MethodPluginReconfigure:
		if err := configure(request); err != nil {
			return nil, err
		}
		return okEnvelope(pluginRegistration())
	case pluginabi.MethodFrontendAuthIdentifier:
		return okEnvelope(map[string]string{"identifier": service.PluginID})
	case pluginabi.MethodFrontendAuthAuthenticate:
		return authenticate(request)
	case pluginabi.MethodModelRoute:
		return routeModel(request)
	case pluginabi.MethodExecutorIdentifier:
		return okEnvelope(map[string]string{"identifier": service.PluginID})
	case pluginabi.MethodExecutorExecute:
		return execute(request)
	case pluginabi.MethodExecutorExecuteStream:
		return executeStream(request)
	case pluginabi.MethodExecutorCountTokens:
		return okEnvelope(pluginapi.ExecutorResponse{Payload: []byte(`{"input_tokens":0}`)})
	case pluginabi.MethodManagementRegister:
		return okEnvelope(pluginapi.ManagementRegistrationResponse{
			Routes:    management.Routes(),
			Resources: management.Resources(),
		})
	case pluginabi.MethodManagementHandle:
		return handleManagement(request)
	case pluginabi.MethodUsageHandle:
		return handleUsage(request)
	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

func configure(raw []byte) error {
	var req lifecycleRequest
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &req); err != nil {
			return err
		}
	}
	return service.Configure(context.Background(), req.ConfigYAML)
}

func pluginRegistration() registration {
	formats := []string{
		"openai", "chat-completions", "claude", "gemini", "openai-response", "responses", "codex",
	}
	return registration{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             service.PluginName,
			Version:          service.PluginVersion,
			Author:           "yuluo688",
			GitHubRepository: "https://github.com/yuluo688/credit-manager",
			ConfigFields: []pluginapi.ConfigField{
				{Name: "config_file", Type: pluginapi.ConfigFieldTypeString, Description: "External YAML path (or set CREDIT_MANAGER_CONFIG_FILE). Host may only set this."},
				{Name: "data_dir", Type: pluginapi.ConfigFieldTypeString, Description: "Plugin-managed data directory for SQLite and lock files."},
				{Name: "database_file", Type: pluginapi.ConfigFieldTypeString, Description: "SQLite filename under data_dir (default credit-manager.db)."},
				{Name: "busy_timeout", Type: pluginapi.ConfigFieldTypeString, Description: "SQLite busy timeout duration, e.g. 5s."},
				{Name: "keys.pepper_env", Type: pluginapi.ConfigFieldTypeString, Description: "Optional env override for id:pepper list; wins over pepper file when set."},
				{Name: "keys.pepper_file", Type: pluginapi.ConfigFieldTypeString, Description: "Pepper file under data_dir (default key-peppers). Auto-created on first run."},
				{Name: "keys.active_pepper_id", Type: pluginapi.ConfigFieldTypeString, Description: "Pepper id used when minting new keys."},
				{Name: "limits.max_token_estimate", Type: pluginapi.ConfigFieldTypeInteger, Description: "Strict maximum tokens reserved per request."},
				{Name: "limits.default_output_reserve", Type: pluginapi.ConfigFieldTypeInteger, Description: "Default output token reserve when body omits max_tokens."},
				{Name: "pricing.unknown_policy", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{"deny", "allow", "default"}, Description: "Behavior when no price rule matches."},
				{Name: "settlement.missing_usage", Type: pluginapi.ConfigFieldTypeEnum, EnumValues: []string{"settle_reserved", "release"}, Description: "Settlement when upstream returns no usage."},
			},
		},
		Capabilities: registrationCapability{
			FrontendAuthProvider:          true,
			FrontendAuthProviderExclusive: true,
			ModelRouter:                   true,
			Executor:                      true,
			ExecutorModelScope:            string(pluginapi.ExecutorModelScopeBoth),
			ExecutorInputFormats:          formats,
			ExecutorOutputFormats:         formats,
			UsagePlugin:                   true,
			ManagementAPI:                 true,
		},
	}
}

func authenticate(raw []byte) ([]byte, error) {
	svc := service.Current()
	if svc == nil {
		return okEnvelope(pluginapi.FrontendAuthResponse{Authenticated: false})
	}
	var req pluginapi.FrontendAuthRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return okEnvelope(pluginapi.FrontendAuthResponse{Authenticated: false})
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

func routeModel(raw []byte) ([]byte, error) {
	var req rpcModelRouteRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	// Only handle authenticated plugin-key traffic; exclusive auth already rejected others.
	scope := metadataString(req.Metadata, service.CallerScopeMetadataKey)
	if scope == "" && bearerPresent(req.Headers) {
		// Host may not yet inject caller_scope into router metadata; still claim authenticated Bearer traffic.
		return okEnvelope(pluginapi.ModelRouteResponse{
			Handled:    true,
			TargetKind: pluginapi.ModelRouteTargetSelf,
			Reason:     "credit_manager_authenticated",
		})
	}
	if scope == "" {
		return okEnvelope(pluginapi.ModelRouteResponse{Handled: false})
	}
	return okEnvelope(pluginapi.ModelRouteResponse{
		Handled:    true,
		TargetKind: pluginapi.ModelRouteTargetSelf,
		Reason:     "credit_manager_caller_scope",
	})
}

func execute(raw []byte) ([]byte, error) {
	svc := service.Current()
	if svc == nil {
		return errorEnvelope("service_unavailable", "credit manager is not configured"), nil
	}
	var req rpcExecutorRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	ctx := context.Background()
	key, _, err := svc.ResolveIdentity(ctx, req.Headers, req.Metadata)
	if err != nil {
		return errorEnvelope("unauthorized", err.Error()), nil
	}
	body := requestBody(req.ExecutorRequest)
	plan, err := svc.BuildReservePlan(ctx, req.Model, body)
	if err != nil {
		return errorEnvelope("reserve_rejected", err.Error()), nil
	}
	reservation, err := svc.Reserve(ctx, key, plan, "")
	if err != nil {
		return errorEnvelope("insufficient_quota", err.Error()), nil
	}
	svc.TrackAuthCapture(reservation.ID, plan.Model)

	startedAt := time.Now()
	hostBody, headers, status, errHost := hostModelExecute(req.HostCallbackID, req.ExecutorRequest, body, false)
	completedAt := time.Now()
	metrics := usageMetricsFromRequest(body, startedAt, completedAt, resultFromStatus(status))
	if errHost != nil {
		_ = svc.Release(ctx, reservation.ID, "upstream_error:"+errHost.Error())
		return errorEnvelope("upstream_error", errHost.Error()), nil
	}
	if status >= 400 {
		// Upstream executed; settle conservatively unless body has usage.
		parsed := usageparse.FromResponseBody(hostBody, req.SourceFormat)
		if settleErr := svc.SettleFromUsage(ctx, reservation, plan, parsed, req.SourceFormat, metrics); settleErr != nil {
			_ = svc.Release(ctx, reservation.ID, "settle_failed")
		}
		return okEnvelope(pluginapi.ExecutorResponse{Payload: hostBody, Headers: headers})
	}
	parsed := usageparse.FromResponseBody(hostBody, firstNonEmpty(req.Format, req.SourceFormat))
	if settleErr := svc.SettleFromUsage(ctx, reservation, plan, parsed, firstNonEmpty(req.Format, req.SourceFormat), metrics); settleErr != nil {
		// Prefer not to drop real response; attempt reserved settle already handled inside.
		_ = settleErr
	}
	return okEnvelope(pluginapi.ExecutorResponse{Payload: hostBody, Headers: headers})
}

func executeStream(raw []byte) ([]byte, error) {
	svc := service.Current()
	if svc == nil {
		return errorEnvelope("service_unavailable", "credit manager is not configured"), nil
	}
	var req rpcExecutorRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	streamID := strings.TrimSpace(req.StreamID)
	if streamID == "" {
		return errorEnvelope("executor_error", "stream_id is required"), nil
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				closePluginStream(streamID, fmt.Sprintf("panic: %v", recovered))
			}
		}()
		if err := runStream(context.Background(), svc, req, streamID); err != nil {
			closePluginStream(streamID, err.Error())
			return
		}
		closePluginStream(streamID, "")
	}()
	return okEnvelope(map[string]any{
		"headers": http.Header{"Content-Type": []string{"text/event-stream"}},
	})
}

func runStream(ctx context.Context, svc *service.Service, req rpcExecutorRequest, pluginStreamID string) error {
	key, _, err := svc.ResolveIdentity(ctx, req.Headers, req.Metadata)
	if err != nil {
		return err
	}
	body := requestBody(req.ExecutorRequest)
	plan, err := svc.BuildReservePlan(ctx, req.Model, body)
	if err != nil {
		return err
	}
	reservation, err := svc.Reserve(ctx, key, plan, "")
	if err != nil {
		return err
	}
	svc.TrackAuthCapture(reservation.ID, plan.Model)

	startedAt := time.Now()
	raw, err := callHost(pluginabi.MethodHostModelExecuteStream, hostModelExecutionRequest{
		HostModelExecutionRequest: pluginapi.HostModelExecutionRequest{
			EntryProtocol: firstNonEmpty(req.SourceFormat, "openai"),
			ExitProtocol:  firstNonEmpty(req.Format, req.SourceFormat, "openai"),
			Model:         strings.TrimSpace(req.Model),
			Stream:        true,
			Body:          body,
			Headers:       req.Headers,
			Query:         req.Query,
			Alt:           req.Alt,
		},
		HostCallbackID: req.HostCallbackID,
	})
	if err != nil {
		_ = svc.Release(ctx, reservation.ID, "upstream_stream_error")
		return err
	}
	var stream pluginapi.HostModelStreamResponse
	if err := json.Unmarshal(raw, &stream); err != nil {
		_ = svc.Release(ctx, reservation.ID, "bad_host_stream")
		return err
	}
	initialCompletedAt := time.Now()
	if stream.StatusCode >= 400 {
		_ = closeHostModelStream(stream.StreamID)
		parsed := usageparse.Result{}
		_ = svc.SettleFromUsage(ctx, reservation, plan, parsed, req.SourceFormat,
			usageMetricsFromStream(body, startedAt, time.Time{}, initialCompletedAt, "failed"))
		return fmt.Errorf("host model status %d", stream.StatusCode)
	}
	if strings.TrimSpace(stream.StreamID) == "" {
		_ = svc.Release(ctx, reservation.ID, "empty_stream_id")
		return fmt.Errorf("empty host stream id")
	}
	defer func() { _ = closeHostModelStream(stream.StreamID) }()

	firstChunkAt := time.Time{}
	completedAt := time.Time{}
	var buffer bytes.Buffer
	maxBuffer := svc.Config().Stream.MaxBufferBytes
	for {
		chunkRaw, errRead := callHost(pluginabi.MethodHostModelStreamRead, pluginapi.HostModelStreamReadRequest{StreamID: stream.StreamID})
		if errRead != nil {
			completedAt = time.Now()
			parsed := usageparse.FromStreamBuffer(buffer.Bytes(), firstNonEmpty(req.Format, req.SourceFormat))
			_ = svc.SettleFromUsage(ctx, reservation, plan, parsed, firstNonEmpty(req.Format, req.SourceFormat),
				usageMetricsFromStream(body, startedAt, firstChunkAt, completedAt, "failed"))
			return errRead
		}
		var chunk pluginapi.HostModelStreamReadResponse
		if err := json.Unmarshal(chunkRaw, &chunk); err != nil {
			completedAt = time.Now()
			_ = svc.SettleFromUsage(ctx, reservation, plan, usageparse.Result{}, req.SourceFormat,
				usageMetricsFromStream(body, startedAt, firstChunkAt, completedAt, "failed"))
			return err
		}
		if chunk.Error != "" {
			completedAt = time.Now()
			parsed := usageparse.FromStreamBuffer(buffer.Bytes(), firstNonEmpty(req.Format, req.SourceFormat))
			_ = svc.SettleFromUsage(ctx, reservation, plan, parsed, firstNonEmpty(req.Format, req.SourceFormat),
				usageMetricsFromStream(body, startedAt, firstChunkAt, completedAt, "failed"))
			return fmt.Errorf("%s", chunk.Error)
		}
		if len(chunk.Payload) > 0 {
			if buffer.Len() < maxBuffer {
				remain := min(maxBuffer-buffer.Len(), len(chunk.Payload))
				_, _ = buffer.Write(chunk.Payload[:remain])
			}
			if firstChunkAt.IsZero() {
				firstChunkAt = time.Now()
			}
			if err := emitPluginStreamChunk(pluginStreamID, bytes.Clone(chunk.Payload)); err != nil {
				completedAt = time.Now()
				parsed := usageparse.FromStreamBuffer(buffer.Bytes(), firstNonEmpty(req.Format, req.SourceFormat))
				_ = svc.SettleFromUsage(ctx, reservation, plan, parsed, firstNonEmpty(req.Format, req.SourceFormat),
					usageMetricsFromStream(body, startedAt, firstChunkAt, completedAt, "cancelled"))
				return err
			}
		}
		if chunk.Done {
			break
		}
	}
	completedAt = time.Now()
	parsed := usageparse.FromStreamBuffer(buffer.Bytes(), firstNonEmpty(req.Format, req.SourceFormat))
	return svc.SettleFromUsage(ctx, reservation, plan, parsed, firstNonEmpty(req.Format, req.SourceFormat),
		usageMetricsFromStream(body, startedAt, firstChunkAt, completedAt, "success"))
}

func hostModelExecute(hostCallbackID string, req pluginapi.ExecutorRequest, body []byte, stream bool) ([]byte, http.Header, int, error) {
	raw, err := callHost(pluginabi.MethodHostModelExecute, hostModelExecutionRequest{
		HostModelExecutionRequest: pluginapi.HostModelExecutionRequest{
			EntryProtocol: firstNonEmpty(req.SourceFormat, "openai"),
			ExitProtocol:  firstNonEmpty(req.Format, req.SourceFormat, "openai"),
			Model:         strings.TrimSpace(req.Model),
			Stream:        stream,
			Body:          body,
			Headers:       req.Headers,
			Query:         req.Query,
			Alt:           req.Alt,
		},
		HostCallbackID: hostCallbackID,
	})
	if err != nil {
		return nil, nil, 0, err
	}
	var resp pluginapi.HostModelExecutionResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, nil, 0, err
	}
	return resp.Body, resp.Headers, resp.StatusCode, nil
}

func handleManagement(raw []byte) ([]byte, error) {
	var req pluginapi.ManagementRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil, err
	}
	resp, err := management.Handle(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return okEnvelope(resp)
}

func handleUsage(raw []byte) ([]byte, error) {
	svc := service.Current()
	if svc == nil || len(raw) == 0 {
		return okEnvelope(map[string]any{})
	}
	var record pluginapi.UsageRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return okEnvelope(map[string]any{})
	}
	auth := authIdentityFromUsage(record)
	if auth.Empty() {
		return okEnvelope(map[string]any{})
	}
	auth = enrichAuthIdentity(auth)
	ledgerID, ok := svc.ObserveSelectedAuth(record.RequestedAt, auth, record.Model, record.Alias)
	if ok && strings.TrimSpace(ledgerID) != "" {
		_ = svc.Store().UpdateUsageAuth(context.Background(), ledgerID, auth)
	}
	return okEnvelope(map[string]any{})
}

func authIdentityFromUsage(record pluginapi.UsageRecord) store.AuthIdentity {
	return store.AuthIdentity{
		AuthID:    strings.TrimSpace(record.AuthID),
		AuthIndex: strings.TrimSpace(record.AuthIndex),
		Type:      strings.TrimSpace(record.AuthType),
		Provider:  strings.TrimSpace(record.Provider),
	}
}

func enrichAuthIdentity(auth store.AuthIdentity) store.AuthIdentity {
	if strings.TrimSpace(auth.AuthIndex) != "" {
		if enriched, ok := authIdentityFromHostIndex(auth.AuthIndex); ok {
			return mergeAuthIdentity(auth, enriched)
		}
	}
	if strings.TrimSpace(auth.AuthID) != "" {
		if enriched, ok := authIdentityFromHostID(auth.AuthID); ok {
			return mergeAuthIdentity(auth, enriched)
		}
	}
	return auth
}

func authIdentityFromHostIndex(authIndex string) (store.AuthIdentity, bool) {
	raw, err := callHost(pluginabi.MethodHostAuthGetRuntime, pluginapi.HostAuthGetRequest{AuthIndex: authIndex})
	if err != nil || len(raw) == 0 {
		return store.AuthIdentity{}, false
	}
	var resp pluginapi.HostAuthGetRuntimeResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return store.AuthIdentity{}, false
	}
	identity := authIdentityFromHostEntry(resp.Auth)
	if identity.Empty() {
		return store.AuthIdentity{}, false
	}
	return identity, true
}

func authIdentityFromHostID(authID string) (store.AuthIdentity, bool) {
	raw, err := callHost(pluginabi.MethodHostAuthList, map[string]any{})
	if err != nil || len(raw) == 0 {
		return store.AuthIdentity{}, false
	}
	var resp struct {
		Files []pluginapi.HostAuthFileEntry `json:"files"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return store.AuthIdentity{}, false
	}
	authID = strings.TrimSpace(authID)
	for _, entry := range resp.Files {
		if strings.TrimSpace(entry.ID) == authID || strings.TrimSpace(entry.AuthIndex) == authID || strings.TrimSpace(entry.Name) == authID {
			return authIdentityFromHostEntry(entry), true
		}
	}
	return store.AuthIdentity{}, false
}

func authIdentityFromHostEntry(entry pluginapi.HostAuthFileEntry) store.AuthIdentity {
	return store.AuthIdentity{
		AuthID:    firstNonEmpty(entry.ID, entry.Name),
		AuthIndex: strings.TrimSpace(entry.AuthIndex),
		Name:      firstNonEmpty(entry.Name, entry.Path),
		Label:     firstNonEmpty(entry.Label, entry.Account, entry.Email),
		Provider:  strings.TrimSpace(entry.Provider),
		Type:      strings.TrimSpace(entry.Type),
		Email:     strings.TrimSpace(entry.Email),
		Path:      strings.TrimSpace(entry.Path),
	}
}

func mergeAuthIdentity(base, extra store.AuthIdentity) store.AuthIdentity {
	return store.AuthIdentity{
		AuthID:    firstNonEmpty(base.AuthID, extra.AuthID),
		AuthIndex: firstNonEmpty(base.AuthIndex, extra.AuthIndex),
		Name:      firstNonEmpty(extra.Name, base.Name),
		Label:     firstNonEmpty(extra.Label, base.Label),
		Provider:  firstNonEmpty(extra.Provider, base.Provider),
		Type:      firstNonEmpty(extra.Type, base.Type),
		Email:     firstNonEmpty(extra.Email, base.Email),
		Path:      firstNonEmpty(extra.Path, base.Path),
	}
}

func requestBody(req pluginapi.ExecutorRequest) []byte {
	if len(req.OriginalRequest) > 0 {
		return req.OriginalRequest
	}
	return req.Payload
}

func usageMetricsFromRequest(body []byte, startedAt, completedAt time.Time, result string) store.UsageMetrics {
	return usageMetrics(body, startedAt, time.Time{}, completedAt, result)
}

func usageMetricsFromStream(body []byte, startedAt, firstChunkAt, completedAt time.Time, result string) store.UsageMetrics {
	return usageMetrics(body, startedAt, firstChunkAt, completedAt, result)
}

func usageMetrics(body []byte, startedAt, firstChunkAt, completedAt time.Time, result string) store.UsageMetrics {
	metrics := store.UsageMetrics{
		Tier:              optionalString(requestString(body, "service_tier", "tier")),
		Result:            optionalString(result),
		ThinkingIntensity: optionalString(requestNestedString(body, []string{"reasoning", "effort"}, []string{"thinking", "effort"}, []string{"reasoning_effort"})),
	}
	if !startedAt.IsZero() && !completedAt.IsZero() && !completedAt.Before(startedAt) {
		generationDuration := completedAt.Sub(startedAt)
		metrics.GenerationDuration = &generationDuration
	}
	if !startedAt.IsZero() && !firstChunkAt.IsZero() && !firstChunkAt.Before(startedAt) {
		firstTokenLatency := firstChunkAt.Sub(startedAt)
		metrics.FirstTokenLatency = &firstTokenLatency
	}
	return metrics
}

func requestString(body []byte, keys ...string) string {
	var value map[string]any
	if len(body) == 0 || json.Unmarshal(body, &value) != nil {
		return ""
	}
	for _, key := range keys {
		if text, ok := value[key].(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func requestNestedString(body []byte, paths ...[]string) string {
	var value any
	if len(body) == 0 || json.Unmarshal(body, &value) != nil {
		return ""
	}
	for _, path := range paths {
		current := value
		for _, key := range path {
			object, ok := current.(map[string]any)
			if !ok {
				current = nil
				break
			}
			current = object[key]
		}
		if text, ok := current.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func resultFromStatus(status int) string {
	if status >= http.StatusBadRequest {
		return "failed"
	}
	return "success"
}

func bearerPresent(headers http.Header) bool {
	if headers == nil {
		return false
	}
	auth := headers.Get("Authorization")
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(auth)), "bearer ")
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
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func emitPluginStreamChunk(streamID string, payload []byte) error {
	_, err := callHost(pluginabi.MethodHostStreamEmit, rpcStreamEmitRequest{StreamID: streamID, Payload: payload})
	return err
}

func closePluginStream(streamID, errMsg string) {
	_, _ = callHost(pluginabi.MethodHostStreamClose, rpcStreamCloseRequest{StreamID: streamID, Error: strings.TrimSpace(errMsg)})
}

func closeHostModelStream(streamID string) error {
	if strings.TrimSpace(streamID) == "" {
		return nil
	}
	_, err := callHost(pluginabi.MethodHostModelStreamClose, pluginapi.HostModelStreamCloseRequest{StreamID: streamID})
	return err
}

func callHost(method string, payload any) (json.RawMessage, error) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal host callback %s: %w", method, err)
	}
	cMethod := C.CString(method)
	defer C.free(unsafe.Pointer(cMethod))

	var response C.cliproxy_buffer
	var requestPtr *C.uint8_t
	if len(rawPayload) > 0 {
		cPayload := C.CBytes(rawPayload)
		if cPayload == nil {
			return nil, fmt.Errorf("allocate host callback %s", method)
		}
		defer C.free(cPayload)
		requestPtr = (*C.uint8_t)(cPayload)
	}
	callCode := C.call_host_api(cMethod, requestPtr, C.size_t(len(rawPayload)), &response)
	var rawResponse []byte
	if response.ptr != nil && response.len > 0 {
		rawResponse = C.GoBytes(response.ptr, C.int(response.len))
	}
	if response.ptr != nil {
		C.free_host_buffer(response.ptr, response.len)
	}
	if len(rawResponse) == 0 {
		return nil, fmt.Errorf("host callback %s returned no response, code=%d", method, int(callCode))
	}
	var env envelope
	if err := json.Unmarshal(rawResponse, &env); err != nil {
		return nil, fmt.Errorf("decode host envelope %s: %w", method, err)
	}
	if !env.OK {
		if env.Error != nil {
			return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		return nil, fmt.Errorf("host callback %s failed", method)
	}
	if callCode != 0 {
		return nil, fmt.Errorf("host callback %s returned code=%d", method, int(callCode))
	}
	return append(json.RawMessage(nil), env.Result...), nil
}

func okEnvelope(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(envelope{OK: true, Result: raw})
}

func errorEnvelope(code, message string) []byte {
	raw, _ := json.Marshal(envelope{OK: false, Error: &envelopeError{Code: code, Message: message}})
	return raw
}

func writeResponse(response *C.cliproxy_buffer, raw []byte) {
	if response == nil || len(raw) == 0 {
		return
	}
	ptr := C.CBytes(raw)
	if ptr == nil {
		return
	}
	response.ptr = ptr
	response.len = C.size_t(len(raw))
}
