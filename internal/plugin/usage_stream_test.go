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

	t.Run("preserves client usage option", func(t *testing.T) {
		body := requestBodyWithStreamUsage([]byte(`{"stream_options":{"include_usage":false,"extra":1}}`), "openai", "openai")
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatal(err)
		}
		options := payload["stream_options"].(map[string]any)
		if options["include_usage"] != false || options["extra"] != float64(1) {
			t.Fatalf("stream_options = %#v, want client options unchanged", options)
		}
	})

	t.Run("leaves claude payload untouched", func(t *testing.T) {
		original := []byte(`{"stream":true}`)
		if got := string(requestBodyWithStreamUsage(original, "claude", "claude")); got != string(original) {
			t.Fatalf("body = %s, want unchanged", got)
		}
	})
}
