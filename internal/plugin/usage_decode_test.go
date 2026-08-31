package plugin

import "testing"

func TestDecodeHostUsageRecordAcceptsSnakeCase(t *testing.T) {
	raw := []byte(`{"provider":"xai","executor_type":"XAIExecutor","model":"grok-4.6","auth_index":"idx-1","auth_id":"auth-1","detail":{"input_tokens":11,"output_tokens":3}}`)
	record, ok := decodeHostUsageRecord(raw)
	if !ok {
		t.Fatal("expected snake_case usage record")
	}
	if record.Provider != "xai" || record.ExecutorType != "XAIExecutor" || record.Model != "grok-4.6" || record.AuthIndex != "idx-1" || record.Detail.InputTokens != 11 {
		t.Fatalf("record = %#v", record)
	}
}

func TestDecodeHostUsageRecordAcceptsPascalCase(t *testing.T) {
	raw := []byte(`{"Provider":"xai","Model":"grok-4.6","AuthIndex":"idx-2","Detail":{"InputTokens":8,"OutputTokens":1}}`)
	record, ok := decodeHostUsageRecord(raw)
	if !ok || record.AuthIndex != "idx-2" || record.Detail.OutputTokens != 1 {
		t.Fatalf("record = %#v ok=%v", record, ok)
	}
}

func TestDecodeHostUsageRecordAcceptsTotalTokensOnly(t *testing.T) {
	raw := []byte(`{"provider":"openai-compatible-bigmodel","model":"glm-5.3-flash","detail":{"total_tokens":88}}`)
	record, ok := decodeHostUsageRecord(raw)
	if !ok || record.Detail.TotalTokens != 88 {
		t.Fatalf("record = %#v ok=%v", record, ok)
	}
	if !hostUsageFound(usageFromHostRecord(record)) {
		t.Fatal("official total_tokens must count as host usage")
	}
}

func TestHostServiceTierReadsSnakeAndDetail(t *testing.T) {
	if got := hostServiceTier([]byte(`{"service_tier":"priority","detail":{"input_tokens":1}}`)); got != "priority" {
		t.Fatalf("root tier = %q", got)
	}
	if got := hostServiceTier([]byte(`{"detail":{"service_tier":"default"}}`)); got != "default" {
		t.Fatalf("detail tier = %q", got)
	}
}

func TestDecodeHostUsageRecordReadsTokenBreakdown(t *testing.T) {
	raw := []byte(`{"model":"glm-5.3-flash","detail":{"token_breakdown":{"total_tokens":30,"input":{"total_tokens":10},"output":{"total_tokens":20,"reasoning_tokens":4}}}}`)
	record, ok := decodeHostUsageRecord(raw)
	if !ok || record.Detail.InputTokens != 10 || record.Detail.OutputTokens != 20 || record.Detail.ReasoningTokens != 4 || record.Detail.TotalTokens != 30 {
		t.Fatalf("record = %#v ok=%v", record, ok)
	}
}
