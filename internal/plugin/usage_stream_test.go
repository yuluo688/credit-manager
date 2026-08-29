package plugin

import (
	"encoding/json"
	"testing"
)

func TestRequestBodyWithStreamUsage(t *testing.T) {
	t.Run("adds usage request for openai streams", func(t *testing.T) {
		body := requestBodyWithStreamUsage([]byte(`{"model":"test","stream":true}`), "openai", "openai")
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		options, ok := payload["stream_options"].(map[string]any)
		if !ok || options["include_usage"] != true {
			t.Fatalf("stream_options = %#v, want include_usage=true", payload["stream_options"])
		}
	})

	t.Run("forces include_usage for ledger settlement", func(t *testing.T) {
		body := requestBodyWithStreamUsage([]byte(`{"stream_options":{"include_usage":false,"extra":1}}`), "openai", "openai")
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		options := payload["stream_options"].(map[string]any)
		if options["include_usage"] != true || options["extra"] != float64(1) {
			t.Fatalf("stream_options = %#v, want include_usage=true with extra preserved", options)
		}
	})

	t.Run("leaves claude payload untouched", func(t *testing.T) {
		original := []byte(`{"stream":true}`)
		if got := string(requestBodyWithStreamUsage(original, "claude", "claude")); got != string(original) {
			t.Fatalf("body = %s, want unchanged", got)
		}
	})

	t.Run("does not inject stream_options for responses api", func(t *testing.T) {
		original := []byte(`{"model":"gpt-5.6-sol","input":"hello","stream":true}`)
		for _, formats := range [][2]string{
			{"openai-response", "openai-response"},
			{"responses", "responses"},
			{"codex", "codex"},
			{"openai-response", "codex"},
			{"openai-response", "openai"},
		} {
			got := requestBodyWithStreamUsage(original, formats[0], formats[1])
			if string(got) != string(original) {
				t.Fatalf("formats %s/%s body = %s, want unchanged", formats[0], formats[1], got)
			}
			var payload map[string]any
			if err := json.Unmarshal(got, &payload); err != nil {
				t.Fatal(err)
			}
			if _, ok := payload["stream_options"]; ok {
				t.Fatalf("formats %s/%s injected stream_options = %#v", formats[0], formats[1], payload["stream_options"])
			}
		}
	})

	t.Run("does not inject stream_options for responses-shaped openai body", func(t *testing.T) {
		original := []byte(`{"model":"gpt-5.6-sol","input":"hello","stream":true}`)
		got := requestBodyWithStreamUsage(original, "openai", "openai")
		if string(got) != string(original) {
			t.Fatalf("body = %s, want unchanged", got)
		}
	})

	t.Run("still injects include_usage for chat completions", func(t *testing.T) {
		body := requestBodyWithStreamUsage([]byte(`{"model":"test","messages":[{"role":"user","content":"hi"}],"stream":true}`), "chat-completions", "openai")
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		options, ok := payload["stream_options"].(map[string]any)
		if !ok || options["include_usage"] != true {
			t.Fatalf("stream_options = %#v, want include_usage=true", payload["stream_options"])
		}
	})

	t.Run("treats input as responses even when messages is also present", func(t *testing.T) {
		original := []byte(`{"model":"gpt-5.6-sol","input":"hello","messages":[{"role":"user","content":"hi"}],"stream":true}`)
		got := requestBodyWithStreamUsage(original, "openai", "openai")
		var payload map[string]any
		if err := json.Unmarshal(got, &payload); err != nil {
			t.Fatal(err)
		}
		if _, ok := payload["stream_options"]; ok {
			t.Fatalf("stream_options = %#v, want omitted", payload["stream_options"])
		}
	})

	t.Run("strips include_usage from responses payload and keeps other stream_options", func(t *testing.T) {
		original := []byte(`{"model":"gpt-5.6-sol","input":"hello","stream":true,"stream_options":{"include_usage":true,"reasoning_summary_delivery":"sequential_cutoff"}}`)
		got := requestBodyWithStreamUsage(original, "openai-response", "codex")
		var payload map[string]any
		if err := json.Unmarshal(got, &payload); err != nil {
			t.Fatal(err)
		}
		options, ok := payload["stream_options"].(map[string]any)
		if !ok {
			t.Fatal("stream_options missing")
		}
		if _, exists := options["include_usage"]; exists {
			t.Fatalf("include_usage still present: %#v", options)
		}
		if options["reasoning_summary_delivery"] != "sequential_cutoff" {
			t.Fatalf("stream_options = %#v, want reasoning_summary_delivery preserved", options)
		}
	})

	t.Run("does not inject stream_options for responses websocket create", func(t *testing.T) {
		original := []byte(`{"type":"response.create","model":"gpt-5.6-sol","input":"hello"}`)
		got := requestBodyWithStreamUsage(original, "openai-response", "openai-response")
		if string(got) != string(original) {
			t.Fatalf("body = %s, want unchanged", got)
		}
	})

	t.Run("leaves interactions payload untouched", func(t *testing.T) {
		original := []byte(`{"model":"gpt-5.6-sol","input":"hello","stream":true}`)
		if got := string(requestBodyWithStreamUsage(original, "interactions", "interactions")); got != string(original) {
			t.Fatalf("body = %s, want unchanged", got)
		}
	})
}
