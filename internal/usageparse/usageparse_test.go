package usageparse

import (
	"encoding/json"
	"testing"
)

func TestFromResponseBodyFindsOpenAITotalTokens(t *testing.T) {
	got := FromResponseBody([]byte(`{"usage":{"total_tokens":42}}`), "openai")
	if !got.Found || got.Usage.ReportedTotal != 42 {
		t.Fatalf("got %#v, want found total_tokens=42", got)
	}
}

func TestFromStreamBufferFindsResponsesCompletedUsage(t *testing.T) {
	buf := []byte("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":12,\"output_tokens\":7,\"total_tokens\":19}}}\n\n")
	got := FromStreamBuffer(buf, "openai-response")
	if !got.Found || got.Usage.Input != 12 || got.Usage.Output != 7 || got.Usage.ReportedTotal != 19 {
		t.Fatalf("got %#v, want responses completed usage", got)
	}
}

func TestFromStreamBufferFindsSplitSSEUsage(t *testing.T) {
	buf := []byte("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":9,\"output_tokens\":4,\"total_tokens\":13}}}\n\n")
	got := FromStreamBuffer(buf, "openai-response")
	if !got.Found || got.Usage.Input != 9 || got.Usage.Output != 4 {
		t.Fatalf("got %#v, want usage from SSE data line", got)
	}
}

func TestFromStreamBufferFindsWebsocketJSONEvent(t *testing.T) {
	buf := []byte(`{"type":"response.completed","response":{"usage":{"input_tokens":21,"output_tokens":3,"total_tokens":24}}}`)
	got := FromStreamBuffer(buf, "openai-response")
	if !got.Found || got.Usage.Input != 21 || got.Usage.Output != 3 || got.Usage.ReportedTotal != 24 {
		t.Fatalf("got %#v, want websocket JSON usage", got)
	}
}

func TestFromStreamBufferFindsConcatenatedResponsesSSE(t *testing.T) {
	buf := []byte(`event: response.createddata: {"type":"response.created","response":{"usage":null}}event: response.completeddata: {"type":"response.completed","response":{"usage":{"input_tokens":317,"output_tokens":5,"total_tokens":322}}}`)
	got := FromStreamBuffer(buf, "openai-response")
	if !got.Found || got.Usage.Input != 317 || got.Usage.Output != 5 || got.Usage.ReportedTotal != 322 {
		t.Fatalf("got %#v, want concatenated responses usage", got)
	}
}

func TestFromStreamBufferFindsOpenAIChatUsageChunk(t *testing.T) {
	buf := []byte("data: {\"object\":\"chat.completion.chunk\",\"choices\":[{\"delta\":{\"content\":\"pong\"}}]}\n\ndata: {\"object\":\"chat.completion.chunk\",\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":217,\"completion_tokens\":68,\"total_tokens\":285,\"prompt_tokens_details\":{\"cached_tokens\":128},\"completion_tokens_details\":{\"reasoning_tokens\":67}}}\n\ndata: [DONE]\n")
	got := FromStreamBuffer(buf, "openai")
	if !got.Found || got.Usage.Input != 217 || got.Usage.Output != 68 || got.Usage.CacheRead != 128 || got.Usage.Reasoning != 67 {
		t.Fatalf("got %#v, want chat usage chunk", got)
	}
}

func TestFromResponseBodyReadsServiceTier(t *testing.T) {
	got := FromResponseBody([]byte(`{"service_tier":"priority","usage":{"prompt_tokens":12,"completion_tokens":3,"total_tokens":15}}`), "openai")
	if !got.Found || got.ServiceTier != "priority" || got.Usage.Input != 12 {
		t.Fatalf("got %#v, want priority service_tier", got)
	}
}

func TestFromResponseBodyPrefersNestedResponsesServiceTier(t *testing.T) {
	got := FromResponseBody([]byte(`{"type":"response.completed","response":{"service_tier":"default","usage":{"input_tokens":9,"output_tokens":2,"total_tokens":11}}}`), "openai-response")
	if !got.Found || got.ServiceTier != "default" {
		t.Fatalf("got %#v, want nested default service_tier", got)
	}
}

func TestFromResponsesCapsNestedResponseDepth(t *testing.T) {
	nested := any(map[string]any{"usage": map[string]any{"input_tokens": 3.0, "output_tokens": 1.0, "total_tokens": 4.0}})
	for i := 0; i < 12; i++ {
		nested = map[string]any{"response": nested}
	}
	raw, err := json.Marshal(nested)
	if err != nil {
		t.Fatal(err)
	}
	got := FromResponseBody(raw, "openai-response")
	if got.Found {
		t.Fatalf("deeply nested response must not recurse unbounded: %#v", got)
	}
}

func TestFromStreamBufferReadsCompletedServiceTier(t *testing.T) {
	buf := []byte("data: {\"service_tier\":\"priority\",\"usage\":null}\n\ndata: {\"service_tier\":\"default\",\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":1,\"total_tokens\":5}}\n\n")
	got := FromStreamBuffer(buf, "openai")
	if !got.Found || got.ServiceTier != "default" || got.Usage.Input != 4 {
		t.Fatalf("got %#v, want last completed default tier", got)
	}
}
