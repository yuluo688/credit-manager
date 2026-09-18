package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
	"github.com/yuluo688/credit-manager/internal/usageparse"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type imageHold struct {
	reservation store.Reservation
	plan        service.ReservePlan
	stopHeart   func()
}

var (
	imageHoldsMu sync.Mutex
	imageHolds   = map[string]imageHold{}
)

type hostModelExecutionRequest struct {
	pluginapi.HostModelExecutionRequest
	HostCallbackID string `json:"host_callback_id,omitempty"`
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
		if errors.Is(err, store.ErrModelDisabled) {
			return errorEnvelope("model_disabled", err.Error()), nil
		}
		return errorEnvelope("reserve_rejected", err.Error()), nil
	}
	reservation, err := svc.Reserve(ctx, key, plan, "")
	if err != nil {
		if errors.Is(err, store.ErrModelNotAllowed) {
			return errorEnvelope("model_not_allowed", err.Error()), nil
		}
		return errorEnvelope("limit_rejected", err.Error()), nil
	}
	svc.TrackAuthCapture(reservation.ID, plan.Model, req.Model)
	defer func() { _ = svc.FinishExecution(ctx, reservation.ID) }()
	if err := admitExecutorAuth(ctx, svc, reservation.ID, req.ExecutorRequest); err != nil {
		_ = svc.Release(ctx, reservation.ID, "auth_concurrency:"+err.Error())
		return errorEnvelope("limit_rejected", err.Error()), nil
	}
	stopHeartbeat := startReservationHeartbeat(svc, reservation.ID)
	defer stopHeartbeat()

	startedAt := time.Now()
	hostBody, headers, status, errHost := hostModelExecute(req.HostCallbackID, req.ExecutorRequest, body, false)
	completedAt := time.Now()
	metrics := usageMetricsFromRequest(body, startedAt, completedAt, resultFromStatus(status))
	if errHost != nil {
		_ = svc.Release(ctx, reservation.ID, "upstream_error:"+errHost.Error())
		if isAuthConcurrencyError(errHost) {
			return errorEnvelope("limit_rejected", errHost.Error()), nil
		}
		return errorEnvelope("upstream_error", errHost.Error()), nil
	}
	captureUpstreamResponseModel(svc, reservation.ID, hostBody, false)
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
		// Preserve the upstream response, but never silently retain a hold when
		// the final ledger write cannot complete.
		_ = svc.Release(ctx, reservation.ID, "settle_failed")
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
	lifecycleID := lifecycleIDFromHeaders(req.Headers)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				closePluginStream(streamID, fmt.Sprintf("panic: %v", recovered))
			}
		}()
		if err := runStream(context.Background(), svc, req, streamID, lifecycleID); err != nil {
			closePluginStream(streamID, err.Error())
			return
		}
		closePluginStream(streamID, "")
	}()
	return okEnvelope(map[string]any{
		"headers": http.Header{"Content-Type": []string{"text/event-stream"}},
	})
}

func runStream(ctx context.Context, svc *service.Service, req rpcExecutorRequest, pluginStreamID, lifecycleID string) error {
	defer clearStreamLifecycle(lifecycleID)
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
	if bindStreamLifecycle(lifecycleID, reservation.ID) {
		if releaseErr := svc.Release(ctx, reservation.ID, "client_disconnected_before_upstream"); releaseErr != nil {
			// Do not strand the Key slot if SQLite's final release briefly fails.
			// The financial hold remains recoverable through the normal stale path.
			_ = svc.FinishClientExecution(ctx, reservation.ID)
			return fmt.Errorf("release pre-upstream cancellation: %w", releaseErr)
		}
		return errClientDisconnectedBeforeUpstream
	}
	svc.TrackAuthCapture(reservation.ID, plan.Model, req.Model)
	defer func() { _ = svc.FinishExecution(ctx, reservation.ID) }()
	if err := admitExecutorAuth(ctx, svc, reservation.ID, req.ExecutorRequest); err != nil {
		_ = svc.Release(ctx, reservation.ID, "auth_concurrency:"+err.Error())
		return err
	}
	if !beginStreamUpstream(lifecycleID) {
		if releaseErr := svc.Release(ctx, reservation.ID, "client_disconnected_before_upstream"); releaseErr != nil {
			_ = svc.FinishClientExecution(ctx, reservation.ID)
			return fmt.Errorf("release pre-upstream cancellation: %w", releaseErr)
		}
		return errClientDisconnectedBeforeUpstream
	}
	stopHeartbeat := startReservationHeartbeat(svc, reservation.ID)
	defer stopHeartbeat()

	startedAt := time.Now()
	body = requestBodyWithStreamUsage(body, req.SourceFormat, req.Format)
	raw, err := callHost(pluginabi.MethodHostModelExecuteStream, hostModelExecutionRequest{
		HostModelExecutionRequest: pluginapi.HostModelExecutionRequest{
			EntryProtocol: firstNonEmpty(req.SourceFormat, "openai"),
			ExitProtocol:  firstNonEmpty(req.SourceFormat, req.Format, "openai"),
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
		if settleErr := svc.SettleFromUsage(ctx, reservation, plan, parsed, req.SourceFormat,
			usageMetricsFromStream(body, startedAt, time.Time{}, initialCompletedAt, "failed")); settleErr != nil {
			_ = svc.Release(ctx, reservation.ID, "settle_failed")
		}
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
	captureStreamResponseModel := func() {
		captureUpstreamResponseModel(svc, reservation.ID, buffer.Bytes(), true)
	}
	terminal := newStreamTerminalDetector(body)
	for {
		chunkRaw, errRead := callHost(pluginabi.MethodHostModelStreamRead, pluginapi.HostModelStreamReadRequest{StreamID: stream.StreamID})
		if errRead != nil {
			completedAt = time.Now()
			parsed := parseExecutorStreamUsage(buffer.Bytes(), req)
			captureStreamResponseModel()
			if settleErr := svc.SettleFromUsage(ctx, reservation, plan, parsed, firstNonEmpty(req.SourceFormat, req.Format),
				usageMetricsFromStream(body, startedAt, firstChunkAt, completedAt, "failed")); settleErr != nil {
				_ = svc.Release(ctx, reservation.ID, "settle_failed")
			}
			return errRead
		}
		var chunk pluginapi.HostModelStreamReadResponse
		if err := json.Unmarshal(chunkRaw, &chunk); err != nil {
			completedAt = time.Now()
			captureStreamResponseModel()
			if settleErr := svc.SettleFromUsage(ctx, reservation, plan, usageparse.Result{}, req.SourceFormat,
				usageMetricsFromStream(body, startedAt, firstChunkAt, completedAt, "failed")); settleErr != nil {
				_ = svc.Release(ctx, reservation.ID, "settle_failed")
			}
			return err
		}
		if chunk.Error != "" {
			completedAt = time.Now()
			parsed := parseExecutorStreamUsage(buffer.Bytes(), req)
			captureStreamResponseModel()
			if settleErr := svc.SettleFromUsage(ctx, reservation, plan, parsed, firstNonEmpty(req.SourceFormat, req.Format),
				usageMetricsFromStream(body, startedAt, firstChunkAt, completedAt, "failed")); settleErr != nil {
				_ = svc.Release(ctx, reservation.ID, "settle_failed")
			}
			return fmt.Errorf("%s", chunk.Error)
		}
		if len(chunk.Payload) > 0 {
			// Observe each frame so an early model declaration survives a capped
			// usage buffer; the final buffered parse below can still prefer a
			// terminal response.completed declaration.
			captureUpstreamResponseModel(svc, reservation.ID, chunk.Payload, true)
			if terminal.Feed(chunk.Payload) {
				// A failed optimization must not hide completion from the client or
				// interrupt the financial settlement performed below.
				_ = svc.FinishExecution(ctx, reservation.ID)
			}
			if buffer.Len() < maxBuffer {
				remain := min(maxBuffer-buffer.Len(), len(chunk.Payload))
				_, _ = buffer.Write(chunk.Payload[:remain])
			}
			if firstChunkAt.IsZero() {
				firstChunkAt = time.Now()
			}
			if err := emitPluginStreamChunk(pluginStreamID, bytes.Clone(chunk.Payload)); err != nil {
				completedAt = time.Now()
				parsed := parseExecutorStreamUsage(buffer.Bytes(), req)
				captureStreamResponseModel()
				if settleErr := svc.SettleFromUsage(ctx, reservation, plan, parsed, firstNonEmpty(req.SourceFormat, req.Format),
					usageMetricsFromStream(body, startedAt, firstChunkAt, completedAt, "cancelled")); settleErr != nil {
					_ = svc.Release(ctx, reservation.ID, "settle_failed")
				}
				return err
			}
		}
		if chunk.Done {
			break
		}
	}
	completedAt = time.Now()
	parsed := parseExecutorStreamUsage(buffer.Bytes(), req)
	captureStreamResponseModel()
	if err := svc.SettleFromUsage(ctx, reservation, plan, parsed, firstNonEmpty(req.SourceFormat, req.Format),
		usageMetricsFromStream(body, startedAt, firstChunkAt, completedAt, "success")); err != nil {
		_ = svc.Release(ctx, reservation.ID, "settle_failed")
		return err
	}
	return nil
}

func parseExecutorStreamUsage(buf []byte, req rpcExecutorRequest) usageparse.Result {
	var best usageparse.Result
	for _, format := range []string{req.SourceFormat, req.Format, "openai-response", "openai", "codex"} {
		format = strings.TrimSpace(format)
		if format == "" {
			continue
		}
		got := usageparse.FromStreamBuffer(buf, format)
		if got.Found && !got.Partial {
			return got
		}
		if got.Found && !best.Found {
			best = got
		}
	}
	if !best.Found {
		best = usageparse.FromResponseBody(buf, firstNonEmpty(req.SourceFormat, req.Format))
	}
	return best
}

func captureUpstreamResponseModel(svc *service.Service, reservationID string, body []byte, stream bool) {
	if svc == nil || len(body) == 0 {
		return
	}
	model := usageparse.ResponseModelFromBody(body)
	if stream {
		model = usageparse.ResponseModelFromStreamBuffer(body)
	}
	svc.CaptureUpstreamResponseModel(reservationID, model)
}

func admitExecutorAuth(ctx context.Context, svc *service.Service, reservationID string, req pluginapi.ExecutorRequest) error {
	auth := store.AuthIdentity{
		AuthID:    strings.TrimSpace(req.AuthID),
		Provider:  strings.TrimSpace(req.AuthProvider),
		AuthIndex: firstNonEmpty(metadataString(req.Metadata, "selected_auth_index"), metadataString(req.Metadata, "auth_index")),
	}
	if auth.Empty() {
		return nil
	}
	return svc.AdmitAuth(ctx, reservationID, auth)
}

func isAuthConcurrencyError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, store.ErrConcurrentLimit) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "maximum concurrent") || strings.Contains(msg, "limit_rejected")
}

func startReservationHeartbeat(svc *service.Service, reservationID string) func() {
	done := make(chan struct{})
	var stopOnce sync.Once
	if svc == nil || strings.TrimSpace(reservationID) == "" {
		return func() { stopOnce.Do(func() { close(done) }) }
	}
	_ = svc.TouchReservation(context.Background(), reservationID)
	interval := time.Minute
	if timeout := svc.Config().Stream.StaleReservationTimeout; timeout > 0 && timeout/3 < interval {
		interval = timeout / 3
	}
	if interval < time.Second {
		interval = time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_ = svc.TouchReservation(context.Background(), reservationID)
			}
		}
	}()
	return func() { stopOnce.Do(func() { close(done) }) }
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
