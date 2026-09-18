package management

import (
	"strings"
	"testing"

	"github.com/yuluo688/credit-manager/internal/store"
)

func TestUsageViewsExposeUpstreamResponseModelMismatch(t *testing.T) {
	entry := store.UsageEntry{Model: "gpt-5.6-sol", UpstreamResponseModel: "gpt-6-sol"}
	for name, view := range map[string]map[string]any{
		"management": usageView(entry),
		"lookup":     publicUsageView(entry),
	} {
		if got := view["upstream_response_model"]; got != "gpt-6-sol" {
			t.Fatalf("%s upstream_response_model = %#v, want gpt-6-sol", name, got)
		}
		if got, _ := view["upstream_model_mismatch"].(bool); !got {
			t.Fatalf("%s upstream_model_mismatch = %#v, want true", name, view["upstream_model_mismatch"])
		}
		if got, _ := view["upstream_model_variant"].(bool); got {
			t.Fatalf("%s upstream_model_variant = %#v, want false", name, view["upstream_model_variant"])
		}
	}
}

func TestUsageViewsClassifyBuildSuffixAsVariant(t *testing.T) {
	view := usageView(store.UsageEntry{Model: "grok-4.6", UpstreamResponseModel: "grok-4.6-build"})
	if got, _ := view["upstream_model_mismatch"].(bool); !got {
		t.Fatalf("upstream_model_mismatch = %#v, want true", view["upstream_model_mismatch"])
	}
	if got, _ := view["upstream_model_variant"].(bool); !got {
		t.Fatalf("upstream_model_variant = %#v, want true", view["upstream_model_variant"])
	}
}

func TestConsoleRendersUpstreamModelMismatch(t *testing.T) {
	page := string(consolePage().Body)
	for _, text := range []string{
		"const usageModelCell",
		"upstream_response_model",
		"upstream_model_mismatch",
		"上游响应",
		"模型不一致",
		"疑似版本变体",
	} {
		if !strings.Contains(page, text) {
			t.Fatalf("console page is missing upstream model display: %q", text)
		}
	}
}
