package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v8/sdk/pluginapi"
	"github.com/yuluo688/credit-manager/internal/config"
	"github.com/yuluo688/credit-manager/internal/money"
	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
)

func TestTerminalEventReleasesSlotBeforeForwarding(t *testing.T) {
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
		CallerID: service.BootstrapCallerID, Label: "test", QuotaMicroUSD: 10_000_000, MaxConcurrentRequests: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"model":"test-model","stream":true,"input":"test","max_output_tokens":16}`)
	plan, err := svc.BuildReservePlan(ctx, "test-model", body)
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
				// This is the newline-free concatenation that the host can produce.
				result = pluginapi.HostModelStreamReadResponse{Payload: []byte("event: response.createddata: {\"type\":\"response.created\"}event: response.completeddata: {\"type\":\"response.completed\"}")}
			} else {
				result = pluginapi.HostModelStreamReadResponse{Done: true}
			}
		case pluginabi.MethodHostStreamEmit:
			overview, err := svc.Store().GetKeyUsageOverview(ctx, key.ID, time.Now())
			if err != nil || overview.ActiveReservations != 0 {
				return nil, 0, fmt.Errorf("terminal forwarded with active slot: %v", err)
			}
			got, err := svc.Store().GetPluginKey(ctx, key.ID)
			if err != nil || got.HeldAmountMicroUSD <= 0 {
				return nil, 0, fmt.Errorf("terminal forwarding lost financial hold: %v", err)
			}
			if _, err := svc.Reserve(ctx, key, plan, "next-turn"); err != nil {
				return nil, 0, fmt.Errorf("next request could not reserve: %w", err)
			}
			if _, err := svc.Reserve(ctx, key, plan, "over-limit"); !errors.Is(err, store.ErrConcurrentLimit) {
				return nil, 0, fmt.Errorf("concurrency enforcement lost: %v", err)
			}
		case pluginabi.MethodHostModelStreamClose:
		default:
			return nil, 0, fmt.Errorf("unexpected host method %s", method)
		}
		encoded, err := okEnvelope(result)
		return encoded, 0, err
	}
	if err := runStream(ctx, svc, rpcExecutorRequest{ExecutorRequest: pluginapi.ExecutorRequest{
		Model: "test-model", SourceFormat: "openai-response", Format: "openai-response", Stream: true,
		Headers: http.Header{"Authorization": []string{"Bearer " + material.Plaintext}}, Payload: body,
	}}, "downstream", ""); err != nil {
		t.Fatal(err)
	}
}

func TestRunStreamSettlesWhenHostReadReturnsCanceled(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	svc, err := service.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if err := svc.Store().PutPricingRule(ctx, store.PricingRule{
		ID: "test", MatchKind: store.MatchGlob, Pattern: "*", Priority: 10, Enabled: true,
		Price: money.PricePerMTok{Input: 1_000_000, Output: 3_000_000},
	}); err != nil {
		t.Fatal(err)
	}
	key, material, err := svc.MintKey(ctx, service.BootstrapCallerID, "cancel-stream", 10_000_000, nil)
	if err != nil {
		t.Fatal(err)
	}
	oldHost := HostCall
	defer func() { HostCall = oldHost }()
	// Identity and reserve must succeed first. Cancel only when the host read
	// returns, which is the 300s upstream-cap path from the issue.
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	HostCall = func(method string, raw []byte) ([]byte, int, error) {
		switch method {
		case pluginabi.MethodHostModelExecuteStream:
			encoded, err := okEnvelope(pluginapi.HostModelStreamResponse{StreamID: "upstream", StatusCode: http.StatusOK})
			return encoded, 0, err
		case pluginabi.MethodHostModelStreamRead:
			cancel()
			return nil, 0, context.Canceled
		case pluginabi.MethodHostModelStreamClose, pluginabi.MethodHostStreamEmit, pluginabi.MethodHostStreamClose:
			encoded, err := okEnvelope(map[string]any{})
			return encoded, 0, err
		default:
			return nil, 0, fmt.Errorf("unexpected host method %s", method)
		}
	}
	body := []byte(`{"model":"test-model","stream":true,"input":"test","max_output_tokens":16}`)
	err = runStream(streamCtx, svc, rpcExecutorRequest{ExecutorRequest: pluginapi.ExecutorRequest{
		Model: "test-model", SourceFormat: "openai", Format: "openai", Stream: true,
		Headers: http.Header{"Authorization": []string{"Bearer " + material.Plaintext}}, Payload: body,
	}}, "downstream", "")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runStream error = %v, want context.Canceled", err)
	}
	overview, err := svc.Store().GetKeyUsageOverview(ctx, key.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if overview.ActiveReservations != 0 {
		t.Fatalf("active reservations = %d, want 0", overview.ActiveReservations)
	}
	got, err := svc.Store().GetPluginKey(ctx, key.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.HeldAmountMicroUSD != 0 {
		t.Fatalf("held amount = %d, want 0", got.HeldAmountMicroUSD)
	}
	entries, err := svc.Store().ListUsage(ctx, store.UsageFilter{PluginKeyID: key.ID, Limit: 5})
	if err != nil || len(entries) != 1 || entries[0].ReservationID == "" {
		t.Fatalf("usage after canceled host read: %v %#v", err, entries)
	}
}

func TestReservationHeartbeatTouchesImmediately(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	cfg.DataDir = t.TempDir()
	cfg.Stream.StaleReservationTimeout = time.Minute
	svc, err := service.Open(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	key, _, err := svc.MintKey(ctx, service.BootstrapCallerID, "heartbeat", 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := svc.Store().Reserve(ctx, store.ReserveRequest{
		CallerID: key.CallerID, PluginKeyID: key.ID, IdempotencyKey: "heartbeat", Model: "model", RequestTokenEstimate: 1, AmountMicroUSD: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	stop := startReservationHeartbeat(svc, reservation.ID)
	defer stop()
	got, err := svc.Store().GetReservation(ctx, reservation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.UpdatedAt.After(reservation.UpdatedAt) {
		t.Fatalf("heartbeat did not touch immediately: before=%s after=%s", reservation.UpdatedAt, got.UpdatedAt)
	}
}

func TestStreamTerminalDetector(t *testing.T) {
	cases := []struct {
		name   string
		chunks []string
		want   bool
	}{
		{"complete host frames without blank line", []string{
			"event: response.created\ndata: {\"type\":\"response.created\"}",
			"event: response.completed\ndata: {\"type\":\"response.completed\"}",
		}, true},
		// CLIProxyAPI's usage parser already supports this wire form. The detector
		// must release the slot before EOF for it as well.
		{"concatenated stripped response frames", []string{
			"event: response.createddata: {\"type\":\"response.created\"}event: response.completeddata: {\"type\":\"response.completed\"}",
		}, true},
		{"split response event", []string{"event: response.com", "pleted\r\ndata: {}\n\n"}, true},
		{"incomplete terminal event prefix", []string{"event: response.completed", "_extra\n\n"}, false},
		{"incomplete error event prefix", []string{"event: error", "_timeout\n\n"}, false},
		{"multiple chat choices", []string{
			"data: {\"choices\":[{\"index\":0,\"finish_reason\":\"stop\"}]}\n\n",
			"data: {\"choices\":[{\"index\":1,\"finish_reason\":\"stop\"}]}\n\n",
		}, true},
		{"non-terminal data", []string{"data: {\"type\":\"response.output_text.delta\",\"delta\":\"response.completed\"}\n\n"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			detector := newStreamTerminalDetector([]byte(`{"n":2}`))
			if tc.name != "multiple chat choices" {
				detector = newStreamTerminalDetector(nil)
			}
			count := 0
			for _, chunk := range tc.chunks {
				if detector.Feed([]byte(chunk)) {
					count++
				}
			}
			if got := count == 1; got != tc.want {
				t.Fatalf("terminal notifications = %d, want terminal=%t", count, tc.want)
			}
			if tc.want && detector.Feed([]byte("data: [DONE]\n\n")) {
				t.Fatal("duplicate terminal notification")
			}
		})
	}
}

func TestStreamTerminalDetectorWaitsForCompleteDataField(t *testing.T) {
	detector := newStreamTerminalDetector([]byte(`{"n":2}`))
	if detector.Feed([]byte(`data: {"choices":[{"index":0,"finish_reason":"stop"}]}`)) {
		t.Fatal("incomplete data field released concurrency")
	}
	if len(detector.finishedChoices) != 0 {
		t.Fatal("incomplete data field updated choice state")
	}
	if detector.Feed([]byte("\n\n")) {
		t.Fatal("first complete choice released concurrency")
	}
	if !detector.Feed([]byte("data: {\"choices\":[{\"index\":1,\"finish_reason\":\"stop\"}]}\n\n")) {
		t.Fatal("all complete choices should release concurrency")
	}
}

func TestStreamTerminalDetectorPreservesSSEMarkersInsideJSONStrings(t *testing.T) {
	cases := []string{
		`data: {"type":"response.completed","response":{"output_text":"literal data: and event: text"}}` + "\n\n",
		`data: {"choices":[{"index":0,"delta":{"content":"event: data:"},"finish_reason":"stop"}]}` + "\n\n",
	}
	for _, payload := range cases {
		detector := newStreamTerminalDetector(nil)
		if !detector.Feed([]byte(payload)) {
			t.Fatalf("terminal marker inside JSON text broke detection for %q", payload)
		}
	}
}

func TestStreamTerminalDetectorRecognizesRawHostJSONChunks(t *testing.T) {
	cases := []string{
		`{"type":"response.completed"}`,
		`{"choices":[{"index":0,"finish_reason":"stop"}]}`,
		`[DONE]`,
	}
	for _, payload := range cases {
		detector := newStreamTerminalDetector(nil)
		if !detector.Feed([]byte(payload)) {
			t.Fatalf("raw host payload did not release concurrency: %q", payload)
		}
	}
}
