package usageparse

import "testing"

func TestResponseModelFromBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{name: "openai", body: `{"model":"gpt-6-sol","usage":{"total_tokens":1}}`, want: "gpt-6-sol"},
		{name: "responses", body: `{"type":"response.completed","response":{"model":"gpt-6-sol"}}`, want: "gpt-6-sol"},
		{name: "anthropic", body: `{"type":"message_start","message":{"model":"claude-sonnet-4"}}`, want: "claude-sonnet-4"},
		{name: "gemini", body: `{"modelVersion":"gemini-3-pro"}`, want: "gemini-3-pro"},
		{name: "missing", body: `{"usage":{"total_tokens":1}}`, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResponseModelFromBody([]byte(tc.body)); got != tc.want {
				t.Fatalf("ResponseModelFromBody() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResponseModelFromStreamBufferPrefersTerminalResponse(t *testing.T) {
	stream := []byte("data: {\"type\":\"response.created\",\"response\":{\"model\":\"gpt-5.6-sol\"}}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"model\":\"gpt-6-sol\"}}\n\n")
	if got := ResponseModelFromStreamBuffer(stream); got != "gpt-6-sol" {
		t.Fatalf("ResponseModelFromStreamBuffer() = %q, want gpt-6-sol", got)
	}
}

func TestResponseModelFromStreamBufferReadsAnthropicMessageStart(t *testing.T) {
	stream := []byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"claude-sonnet-4\"}}\n\n")
	if got := ResponseModelFromStreamBuffer(stream); got != "claude-sonnet-4" {
		t.Fatalf("ResponseModelFromStreamBuffer() = %q, want claude-sonnet-4", got)
	}
}
