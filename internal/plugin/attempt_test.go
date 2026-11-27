package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
)

func TestImageFailureSettlesZeroCostIntoLedger(t *testing.T) {
	ctx := context.Background()
	svc := newLifecycleTestService(t)
	key, _, err := svc.MintKey(ctx, service.BootstrapCallerID, "image-fail", 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := svc.Store().Reserve(ctx, store.ReserveRequest{
		CallerID: key.CallerID, PluginKeyID: key.ID, IdempotencyKey: "image-fail",
		Model: "image-model", RequestTokenEstimate: 1, AmountMicroUSD: money.MicroUSD(10),
	})
	if err != nil {
		t.Fatal(err)
	}
	const requestID = "image-fail"
	imageHoldsMu.Lock()
	imageHolds[requestID] = imageHold{
		reservation: reservation,
		plan: service.ReservePlan{
			Model: "image-model", ImageCount: 2,
			Price: money.PricePerMTok{BillingMode: money.BillingPerImage, PerImage: 5_000_000},
		},
	}
	imageHoldsMu.Unlock()
	t.Cleanup(func() {
		imageHoldsMu.Lock()
		delete(imageHolds, requestID)
		imageHoldsMu.Unlock()
	})

	raw, err := json.Marshal(pluginapi.RequestCompletion{
		RequestID: requestID, Outcome: pluginapi.RequestCompletionFailed, StatusCode: 500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := completeInterceptedRequest(raw); err != nil {
		t.Fatal(err)
	}

	gotKey, err := svc.Store().GetPluginKey(ctx, key.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotKey.HeldAmountMicroUSD != 0 || gotKey.SettledSpendMicroUSD != 0 {
		t.Fatalf("quota: held=%d settled=%d (must not charge images on failure)", gotKey.HeldAmountMicroUSD, gotKey.SettledSpendMicroUSD)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 5})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	if entries[0].CostMicroUSD != 0 || entries[0].Source != service.SourceAttemptFailed {
		t.Fatalf("entry = %#v", entries[0])
	}
	final, err := svc.Store().GetReservation(ctx, reservation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.Status != store.ReservationSettled {
		t.Fatalf("status = %s, want settled", final.Status)
	}
}

func TestPreUpstreamCancelSettlesCancelledIntoLedger(t *testing.T) {
	ctx := context.Background()
	svc := newLifecycleTestService(t)
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "lifecycle", MatchKind: store.MatchGlob, Pattern: "*", Priority: 10, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 1_000_000},
	}); err != nil {
		t.Fatal(err)
	}
	key, material, err := svc.MintKey(ctx, service.BootstrapCallerID, "cancel-ledger", 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	const requestID = "cancel-into-ledger"
	trackStreamLifecycle(requestID, svc)
	completeStreamLifecycle(pluginapi.RequestCompletion{RequestID: requestID, Outcome: pluginapi.RequestCompletionCanceled})
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	HostCall = func(method string, raw []byte) ([]byte, int, error) {
		return nil, 0, fmt.Errorf("host must not start: %s", method)
	}
	body := []byte(`{"model":"cancel-ledger","stream":true,"max_tokens":1,"messages":[{"role":"user","content":"test"}]}`)
	err = runStream(ctx, svc, rpcExecutorRequest{ExecutorRequest: pluginapi.ExecutorRequest{
		Model: "cancel-ledger", SourceFormat: "openai", Format: "openai", Stream: true,
		Headers: http.Header{"Authorization": []string{"Bearer " + material.Plaintext}}, Payload: body,
	}}, "downstream", requestID)
	if !errors.Is(err, errClientDisconnectedBeforeUpstream) {
		t.Fatalf("runStream error = %v", err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 5})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	if entries[0].CostMicroUSD != 0 || entries[0].Source != service.SourceAttemptCancelled {
		t.Fatalf("entry = %#v", entries[0])
	}
	if entries[0].Metrics.Result == nil || *entries[0].Metrics.Result != "cancelled" {
		t.Fatalf("result = %#v", entries[0].Metrics.Result)
	}
}

func TestExecuteUpstreamErrorSettlesZeroCost(t *testing.T) {
	ctx := context.Background()
	svc := newLifecycleTestService(t)
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "all", MatchKind: store.MatchGlob, Pattern: "*", Priority: 10, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 1_000_000},
	}); err != nil {
		t.Fatal(err)
	}
	key, material, err := svc.MintKey(ctx, service.BootstrapCallerID, "upstream-err", 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	HostCall = func(method string, raw []byte) ([]byte, int, error) {
		return nil, 0, fmt.Errorf("upstream boom")
	}
	body := []byte(`{"model":"upstream-err","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`)
	raw, err := json.Marshal(rpcExecutorRequest{ExecutorRequest: pluginapi.ExecutorRequest{
		Model: "upstream-err", SourceFormat: "openai", Format: "openai",
		Headers: http.Header{"Authorization": []string{"Bearer " + material.Plaintext}}, Payload: body,
	}})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := execute(raw)
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(resp, &envelope); err != nil {
		t.Fatal(err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 5})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v envelope=%v", err, entries, envelope)
	}
	if entries[0].CostMicroUSD != 0 || entries[0].Source != service.SourceAttemptFailed {
		t.Fatalf("entry = %#v", entries[0])
	}
}

func TestReserveRejectedRecordsLedger(t *testing.T) {
	ctx := context.Background()
	svc := newLifecycleTestService(t)
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "all", MatchKind: store.MatchGlob, Pattern: "*", Priority: 10, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 1_000_000},
	}); err != nil {
		t.Fatal(err)
	}
	key, material, err := svc.MintKeyWithPolicy(ctx, service.MintKeyRequest{
		CallerID: service.BootstrapCallerID, Label: "allowlist", QuotaMicroUSD: 10_000_000,
		AllowedModels: []string{"only-this"},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"model":"not-allowed","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`)
	raw, err := json.Marshal(rpcExecutorRequest{ExecutorRequest: pluginapi.ExecutorRequest{
		Model: "not-allowed", SourceFormat: "openai", Format: "openai",
		Headers: http.Header{"Authorization": []string{"Bearer " + material.Plaintext}}, Payload: body,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := execute(raw); err != nil {
		t.Fatal(err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 5})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	if entries[0].Source != service.SourceAttemptRejected || entries[0].CostMicroUSD != 0 {
		t.Fatalf("entry = %#v", entries[0])
	}
}

func TestRunStreamUpstreamErrorPreservesDetailInSettlementSummary(t *testing.T) {
	ctx := context.Background()
	svc := newLifecycleTestService(t)
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "all", MatchKind: store.MatchGlob, Pattern: "*", Priority: 10, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 1_000_000},
	}); err != nil {
		t.Fatal(err)
	}
	key, material, err := svc.MintKey(ctx, service.BootstrapCallerID, "stream-cooldown", 10_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	HostCall = func(method string, raw []byte) ([]byte, int, error) {
		if method == pluginabi.MethodHostModelExecuteStream {
			return nil, 0, fmt.Errorf(`host_call_failed: {"error":{"code":"model_cooldown","message":"All credentials for model gpt-6.2-sol are cooling down"}}`)
		}
		return nil, 0, fmt.Errorf("unexpected host method %s", method)
	}
	body := []byte(`{"model":"gpt-6.2-sol","stream":true,"max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`)
	err = runStream(ctx, svc, rpcExecutorRequest{ExecutorRequest: pluginapi.ExecutorRequest{
		Model: "gpt-6.2-sol", SourceFormat: "openai-response", Format: "openai-response", Stream: true,
		Headers: http.Header{"Authorization": []string{"Bearer " + material.Plaintext}}, Payload: body,
	}}, "downstream", "")
	if err == nil || !strings.Contains(err.Error(), "model_cooldown") {
		t.Fatalf("runStream error = %v, want model_cooldown", err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 5})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	if entries[0].CostMicroUSD != 0 || entries[0].Source != service.SourceAttemptFailed {
		t.Fatalf("entry = %#v", entries[0])
	}
	if !strings.HasPrefix(entries[0].SettlementSummary, "upstream_stream_error:") || !strings.Contains(entries[0].SettlementSummary, "model_cooldown") {
		t.Fatalf("settlement_summary = %q, want upstream_stream_error with model_cooldown", entries[0].SettlementSummary)
	}
}

func TestRunStreamHTTPStatusErrorPreservesStatusSummary(t *testing.T) {
	ctx := context.Background()
	svc := newLifecycleTestService(t)
	key, material, err := svc.MintKey(ctx, service.BootstrapCallerID, "stream-http-500", 10_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	HostCall = func(method string, raw []byte) ([]byte, int, error) {
		switch method {
		case pluginabi.MethodHostModelExecuteStream:
			result, _ := json.Marshal(pluginapi.HostModelStreamResponse{StatusCode: http.StatusTooManyRequests, StreamID: "s-429"})
			return mustMarshalEnvelopeOK(result), 0, nil
		case pluginabi.MethodHostModelStreamClose:
			return mustMarshalEnvelopeOK([]byte(`{}`)), 0, nil
		default:
			return nil, 0, fmt.Errorf("unexpected host method %s", method)
		}
	}
	body := []byte(`{"model":"gpt-6.2-sol","stream":true,"max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`)
	err = runStream(ctx, svc, rpcExecutorRequest{ExecutorRequest: pluginapi.ExecutorRequest{
		Model: "gpt-6.2-sol", SourceFormat: "openai-response", Format: "openai-response", Stream: true,
		Headers: http.Header{"Authorization": []string{"Bearer " + material.Plaintext}}, Payload: body,
	}}, "downstream", "")
	if err == nil {
		t.Fatal("expected error for 429 stream status")
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 5})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	if entries[0].SettlementSummary != "upstream_stream_status:http_429" {
		t.Fatalf("settlement_summary = %q, want upstream_stream_status:http_429", entries[0].SettlementSummary)
	}
}

func TestFormatAttemptSummaryNormalizesAndCaps(t *testing.T) {
	long := strings.Repeat("x", maxAttemptSummaryDetailRunes+50)
	got := formatAttemptSummary("upstream_stream_error", "  line1\n\tline2  "+long)
	if strings.ContainsAny(got, "\n\t") {
		t.Fatalf("summary contains raw whitespace: %q", got)
	}
	if !strings.HasPrefix(got, "upstream_stream_error:line1 line2 ") || !strings.HasSuffix(got, "...") {
		t.Fatalf("summary = %q", got)
	}
	if got := formatAttemptSummary("upstream_stream_error", "   "); got != "upstream_stream_error" {
		t.Fatalf("empty detail summary = %q", got)
	}
}

func mustMarshalEnvelopeOK(result json.RawMessage) []byte {
	raw, _ := json.Marshal(envelope{OK: true, Result: result})
	return raw
}
