package plugin

import "testing"

func TestDecodeHostUsageRecordAcceptsSnakeCase(t *testing.T) {
	raw := []byte(`{"provider":"xai","model":"grok-4.6","auth_index":"idx-1","auth_id":"auth-1","detail":{"input_tokens":11,"output_tokens":3}}`)
	record, ok := decodeHostUsageRecord(raw)
	if !ok {
		t.Fatal("expected snake_case usage record")
	}
	if record.Provider != "xai" || record.Model != "grok-4.6" || record.AuthIndex != "idx-1" || record.Detail.InputTokens != 11 {
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
