package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
	"github.com/yuluo688/credit-manager/internal/usageparse"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestExecuteRecordsUpstreamResponseModel(t *testing.T) {
	ctx := context.Background()
	svc := newLifecycleTestService(t)
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "all", MatchKind: store.MatchGlob, Pattern: "*", Priority: 1, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 1_000_000},
	}); err != nil {
		t.Fatal(err)
	}
	key, material, err := svc.MintKey(ctx, service.BootstrapCallerID, "upstream-model", 10_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	HostCall = func(method string, raw []byte) ([]byte, int, error) {
		if method != pluginabi.MethodHostModelExecute {
			return nil, 0, fmt.Errorf("unexpected host method %s", method)
		}
		encoded, err := okEnvelope(pluginapi.HostModelExecutionResponse{
			StatusCode: http.StatusOK,
			Body:       []byte(`{"model":"gpt-6-sol","usage":{"prompt_tokens":2,"completion_tokens":1}}`),
		})
		return encoded, 0, err
	}
	body := []byte(`{"model":"gpt-5.6-sol","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`)
	raw, err := json.Marshal(rpcExecutorRequest{ExecutorRequest: pluginapi.ExecutorRequest{
		Model: "gpt-5.6-sol", SourceFormat: "openai", Format: "openai",
		Headers: http.Header{"Authorization": []string{"Bearer " + material.Plaintext}}, Payload: body,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := execute(raw); err != nil {
		t.Fatal(err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 1})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	if got := entries[0].UpstreamResponseModel; got != "gpt-6-sol" {
		t.Fatalf("upstream response model = %q, want gpt-6-sol", got)
	}
}

func TestRunStreamRecordsTerminalUpstreamResponseModel(t *testing.T) {
	ctx := context.Background()
	svc := newLifecycleTestService(t)
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "all", MatchKind: store.MatchGlob, Pattern: "*", Priority: 1, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 1_000_000},
	}); err != nil {
		t.Fatal(err)
	}
	key, material, err := svc.MintKey(ctx, service.BootstrapCallerID, "upstream-stream-model", 10_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	reads := 0
	HostCall = func(method string, raw []byte) ([]byte, int, error) {
		var result any = map[string]any{}
		switch method {
		case pluginabi.MethodHostModelExecuteStream:
			result = pluginapi.HostModelStreamResponse{StreamID: "upstream", StatusCode: http.StatusOK}
		case pluginabi.MethodHostModelStreamRead:
			reads++
			if reads == 1 {
				result = pluginapi.HostModelStreamReadResponse{Payload: []byte("data: {\"type\":\"response.created\",\"response\":{\"model\":\"gpt-5.6-sol\"}}\n\n")}
			} else if reads == 2 {
				result = pluginapi.HostModelStreamReadResponse{Payload: []byte("data: {\"type\":\"response.completed\",\"response\":{\"model\":\"gpt-6-sol\"}}\n\n")}
			} else {
				result = pluginapi.HostModelStreamReadResponse{Done: true}
			}
		case pluginabi.MethodHostStreamEmit, pluginabi.MethodHostModelStreamClose:
		default:
			return nil, 0, fmt.Errorf("unexpected host method %s", method)
		}
		encoded, err := okEnvelope(result)
		return encoded, 0, err
	}
	body := []byte(`{"model":"gpt-5.6-sol","stream":true,"input":"hi","max_output_tokens":1}`)
	if err := runStream(ctx, svc, rpcExecutorRequest{ExecutorRequest: pluginapi.ExecutorRequest{
		Model: "gpt-5.6-sol", SourceFormat: "openai-response", Format: "openai-response", Stream: true,
		Headers: http.Header{"Authorization": []string{"Bearer " + material.Plaintext}}, Payload: body,
	}}, "downstream", ""); err != nil {
		t.Fatal(err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 1})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	if got := entries[0].UpstreamResponseModel; got != "gpt-6-sol" {
		t.Fatalf("upstream response model = %q, want gpt-6-sol", got)
	}
}

func TestHandleUsageRecordsHostModelWhenResponseModelIsUnavailable(t *testing.T) {
	ctx := context.Background()
	svc := newLifecycleTestService(t)
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "all", MatchKind: store.MatchGlob, Pattern: "*", Priority: 1, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 1_000_000},
	}); err != nil {
		t.Fatal(err)
	}
	key, _, err := svc.MintKey(ctx, service.BootstrapCallerID, "host-usage-model", 10_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := svc.BuildReservePlan(ctx, "gpt-5.6-sol", []byte(`{"model":"gpt-5.6-sol","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := svc.Reserve(ctx, key, plan, "host-usage-model")
	if err != nil {
		t.Fatal(err)
	}
	svc.TrackAuthCapture(reservation.ID, plan.Model)
	if err := svc.SettleFromUsage(ctx, reservation, plan, usageparse.Result{}, "openai", store.UsageMetrics{}); err != nil {
		t.Fatal(err)
	}
	svc.CaptureUpstreamResponseModel(reservation.ID, "gpt-5.6-sol")
	if _, err := handleUsage([]byte(`{"model":"gpt-6-sol","alias":"gpt-5.6-sol","detail":{"input_tokens":2,"output_tokens":1}}`)); err != nil {
		t.Fatal(err)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 1})
	if err != nil || len(entries) != 1 {
		t.Fatalf("list usage: %v %#v", err, entries)
	}
	if got := entries[0].UpstreamResponseModel; got != "gpt-6-sol" {
		t.Fatalf("upstream response model = %q, want gpt-6-sol", got)
	}
}
