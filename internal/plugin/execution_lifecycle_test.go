package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
	"github.com/yuluo688/credit-manager/internal/config"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
)

func TestTerminalEventReleasesSlotBeforeForwardingAndPreservesBilling(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	cfg.Settlement.HostUsageWait = 40 * time.Millisecond
	svc, err := service.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{ID: "test", MatchKind: store.MatchGlob, Pattern: "*", Priority: 10, Enabled: true, Price: money.PricePerMTok{Input: 1_000_000, Output: 3_000_000}}); err != nil {
		t.Fatal(err)
	}
	key, material, err := svc.MintKey(ctx, service.BootstrapCallerID, "test", 10_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	limit := int64(1)
	if _, err := svc.Store().UpdatePluginKeyPolicy(ctx, store.PluginKeyPolicyUpdate{ID: key.ID, MaxConcurrentRequests: &limit}); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"model":"test-model","stream":true,"input":"test","max_output_tokens":16}`)
	plan, err := svc.BuildReservePlan(ctx, "test-model", body)
	if err != nil {
		t.Fatal(err)
	}
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	reads, emits := 0, 0
	var next store.Reservation
	HostCall = func(method string, raw []byte) ([]byte, int, error) {
		var result any = map[string]any{}
		switch method {
		case pluginabi.MethodHostModelExecuteStream:
			result = pluginapi.HostModelStreamResponse{StreamID: "upstream", StatusCode: 200}
		case pluginabi.MethodHostModelStreamRead:
			reads++
			if reads == 1 {
				result = pluginapi.HostModelStreamReadResponse{Payload: []byte("event: response.completed\ndata: {\"type\":\"response.completed\"}\n\n")}
			} else {
				result = pluginapi.HostModelStreamReadResponse{Done: true}
			}
		case pluginabi.MethodHostStreamEmit:
			emits++
			// This is precisely when the client can see completion and submit
			// another turn; upstream EOF has not been requested yet.
			overview, err := svc.Store().GetKeyUsageOverview(ctx, key.ID, time.Now())
			if err != nil || overview.ActiveReservations != 0 {
				return nil, 0, fmt.Errorf("terminal event forwarded with slot occupied: %v", err)
			}
			got, err := svc.Store().GetPluginKey(ctx, key.ID)
			if err != nil || got.HeldAmountMicroUSD <= 0 {
				return nil, 0, fmt.Errorf("financial hold disappeared: %v", err)
			}
			next, err = svc.Reserve(ctx, key, plan, "next-turn")
			if err != nil {
				return nil, 0, err
			}
			if _, err := svc.Reserve(ctx, key, plan, "over-limit"); !errors.Is(err, store.ErrConcurrentLimit) {
				return nil, 0, fmt.Errorf("concurrency enforcement lost: %v", err)
			}
		case pluginabi.MethodHostModelStreamClose:
		default:
			return nil, 0, fmt.Errorf("unexpected host method %s", method)
		}
		b, err := okEnvelope(result)
		return b, 0, err
	}
	err = runStream(ctx, svc, rpcExecutorRequest{ExecutorRequest: pluginapi.ExecutorRequest{
		Model: "test-model", SourceFormat: "openai-response", Format: "openai-response", Stream: true,
		Headers: http.Header{"Authorization": []string{"Bearer " + material.Plaintext}}, Payload: body,
	}}, "downstream")
	if err != nil {
		t.Fatal(err)
	}
	if emits != 1 || reads != 2 {
		t.Fatalf("unexpected stream flow %d/%d", emits, reads)
	}
	rows, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 10})
	if err != nil || len(rows) != 1 || rows[0].Source != "reserved_fallback" {
		t.Fatalf("fallback ledger: %v %+v", err, rows)
	}
	for i := 0; i < 2; i++ {
		if err := svc.ApplyHostUsage(ctx, rows[0].ID, money.TokenUsage{Input: 9, Output: 3}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := svc.Store().GetPluginKey(ctx, key.ID)
	if err != nil || got.SettledSpendMicroUSD != 18 || got.HeldAmountMicroUSD != next.HeldMicroUSD {
		t.Fatalf("late billing duplicated or lost: %v", err)
	}
}

func TestStreamTerminalDetector(t *testing.T) {
	cases := []struct {
		name   string
		chunks []string
		want   bool
	}{
		{"host frames without blank line", []string{
			"event: response.created\ndata: {\"type\":\"response.created\"}",
			"event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\"}",
			"event: response.completed\ndata: {\"type\":\"response.completed\"}",
		}, true},
		{"chat translated terminal", []string{"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n"}, true},
		{"split responses", []string{"event: response.com", "pleted\r\n", "data: {}\n\n"}, true},
		{"json terminal", []string{"data: {\"type\":\"response.failed\"}\n\n"}, true},
		{"anthropic", []string{"event: message_stop\n\n"}, true},
		{"chat done", []string{"data: [DO", "NE]\n\n"}, true},
		{"tool complete is not turn complete", []string{"event: response.output_item.done\n\n"}, false},
		{"content not an event", []string{"data: {\"type\":\"response.output_text.delta\",\"delta\":\"response.completed\"}\n\n"}, false},
		{"comment", []string{": event: response.completed\n\n"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var d streamTerminalDetector
			count := 0
			for _, s := range tc.chunks {
				if d.Feed([]byte(s)) {
					count++
				}
			}
			if (count == 1) != tc.want {
				t.Fatalf("terminal notifications = %d", count)
			}
			if tc.want && d.Feed([]byte("data: [DONE]\n\n")) {
				t.Fatal("duplicate terminal notification")
			}
		})
	}
	// A huge data line must not make the detector retain arbitrary response data.
	var d streamTerminalDetector
	d.Feed(make([]byte, 128*1024))
	if len(d.line) > 64*1024 {
		t.Fatal("unbounded line buffer")
	}
	if !d.Feed([]byte("\nevent: response.completed\n\n")) {
		t.Fatal("detector failed after oversized line")
	}
	multi := newStreamTerminalDetector([]byte(`{"n":2}`))
	if multi.Feed([]byte("data: {\"choices\":[{\"index\":0,\"finish_reason\":\"stop\"}]}\n\n")) {
		t.Fatal("released before all choices finished")
	}
	if !multi.Feed([]byte("data: {\"choices\":[{\"index\":1,\"finish_reason\":\"stop\"}]}\n\n")) {
		t.Fatal("did not release after all choices finished")
	}
}

func TestCancelledStartupKeepsLateUsageAndReleasesOnlyItsOwnSlot(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	svc, err := service.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{ID: "test", MatchKind: store.MatchGlob, Pattern: "*", Priority: 10, Enabled: true, Price: money.PricePerMTok{Input: 1_000_000, Output: 3_000_000}}); err != nil {
		t.Fatal(err)
	}
	key, material, err := svc.MintKeyWithPolicy(ctx, service.MintKeyRequest{
		CallerID: service.BootstrapCallerID, Label: "cancel-test", QuotaMicroUSD: 10_000_000, MaxConcurrentRequests: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	HostCall = func(method string, raw []byte) ([]byte, int, error) {
		return nil, 0, fmt.Errorf("host model execution: context canceled")
	}
	body := []byte(`{"model":"cancel-model","stream":true,"input":"test","max_output_tokens":16}`)
	err = runStream(ctx, svc, rpcExecutorRequest{ExecutorRequest: pluginapi.ExecutorRequest{
		Model: "cancel-model", SourceFormat: "openai-response", Format: "openai-response", Stream: true,
		Headers: http.Header{"Authorization": []string{"Bearer " + material.Plaintext}}, Payload: body,
	}}, "cancelled-downstream")
	if !isCancelledHostCall(err) {
		t.Fatalf("expected cancelled startup: %v", err)
	}
	rows, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 10})
	if err != nil || len(rows) != 1 || rows[0].Source != "reserved_fallback" || rows[0].Metrics.Result == nil || *rows[0].Metrics.Result != "cancelled" {
		t.Fatalf("cancelled request lost its fallback ledger: %v, rows=%d", err, len(rows))
	}
	plan, err := svc.BuildReservePlan(ctx, "cancel-model", body)
	if err != nil {
		t.Fatal(err)
	}
	next, err := svc.Reserve(ctx, key, plan, "next")
	if err != nil {
		t.Fatalf("cancelled request still uses a slot: %v", err)
	}
	usage := money.TokenUsage{Input: 9, Output: 3}
	ledgerID, matched := svc.ObserveHostUsage(rows[0].CreatedAt, store.AuthIdentity{}, usage, "cancel-model")
	if !matched || ledgerID != rows[0].ID {
		t.Fatal("cancelled startup lost its late usage association")
	}
	for i := 0; i < 2; i++ {
		if err := svc.ApplyHostUsage(ctx, ledgerID, usage); err != nil {
			t.Fatal(err)
		}
	}
	got, err := svc.Store().GetPluginKey(ctx, key.ID)
	if err != nil || got.SettledSpendMicroUSD != 18 || got.HeldAmountMicroUSD != next.HeldMicroUSD {
		t.Fatalf("late usage changed another request's hold or duplicated billing: %v, spend=%d held=%d wantHeld=%d", err, got.SettledSpendMicroUSD, got.HeldAmountMicroUSD, next.HeldMicroUSD)
	}
	if _, err := svc.Reserve(ctx, key, plan, "third"); !errors.Is(err, store.ErrConcurrentLimit) {
		t.Fatalf("late cancellation released another request's slot: %v", err)
	}
}
