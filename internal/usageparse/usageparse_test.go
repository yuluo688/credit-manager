package usageparse

import "testing"

func TestFromResponseBodyFindsOpenAITotalTokens(t *testing.T) {
	got := FromResponseBody([]byte(`{"usage":{"total_tokens":42}}`), "openai")
	if !got.Found || got.Usage.ReportedTotal != 42 {
		t.Fatalf("got %#v, want found total_tokens=42", got)
	}
}
