package management

import (
	"net/url"
	"strings"
	"testing"

	"github.com/yuluo688/credit-manager/internal/service"
	"github.com/yuluo688/credit-manager/internal/store"
)

func TestUsageViewExposesSettlementSummaryAndPublicViewSanitizesIt(t *testing.T) {
	failed := "failed"
	entry := store.UsageEntry{
		Model:             "gpt-6.2-sol",
		Source:            service.SourceAttemptFailed,
		SettlementSummary: `upstream_stream_error:host_call_failed: {"error":{"code":"model_cooldown","message":"All credentials for model gpt-6.2-sol are cooling down","auth":"secret@example.com"}}`,
		Metrics:           store.UsageMetrics{Result: &failed},
	}

	adminView := usageView(entry)
	if got := adminView["settlement_summary"]; got != entry.SettlementSummary {
		t.Fatalf("admin settlement_summary = %#v, want %q", got, entry.SettlementSummary)
	}

	lookupView := publicUsageView(entry)
	if _, leaked := lookupView["settlement_summary"]; leaked {
		t.Fatal("public usage view must not expose raw settlement_summary")
	}
	if got := lookupView["failure_reason"]; got != "model_cooldown" {
		t.Fatalf("public failure_reason = %#v, want model_cooldown", got)
	}

	success := "success"
	okView := publicUsageView(store.UsageEntry{
		Model:             "gpt-6.2-sol",
		SettlementSummary: "host_usage_callback",
		Metrics:           store.UsageMetrics{Result: &success},
	})
	if got := okView["failure_reason"]; got != "" {
		t.Fatalf("success public failure_reason = %#v, want empty", got)
	}
}

func TestUsageFilterFromQueryParsesResult(t *testing.T) {
	filter, err := usageFilterFromQuery(url.Values{"result": {"failed"}}, 25)
	if err != nil {
		t.Fatalf("usageFilterFromQuery: %v", err)
	}
	if filter.Result != "failed" {
		t.Fatalf("filter.Result = %q, want failed", filter.Result)
	}
	if got := usageFilterView(filter)["result"]; got != "failed" {
		t.Fatalf("usageFilterView result = %#v, want failed", got)
	}
}

func TestConsoleAndLookupRenderFailureReasons(t *testing.T) {
	console := string(consolePage().Body)
	for _, text := range []string{
		`id="usageResultFilter"`,
		`value="attempt_failed"`,
		"isUnassignedUsageAuth",
		"未分配账号",
		"parseUsageFailureSummary",
		"usageResultCell",
		"settlement_summary",
	} {
		if !strings.Contains(console, text) {
			t.Fatalf("console page is missing failure/account display: %q", text)
		}
	}

	lookup := string(lookupPage().Body)
	for _, text := range []string{
		"FAILURE_REASON_LABELS",
		"renderRecentResultCell",
		"failure_reason",
		"上游账号冷却中",
	} {
		if !strings.Contains(lookup, text) {
			t.Fatalf("lookup page is missing failure reason display: %q", text)
		}
	}
}
