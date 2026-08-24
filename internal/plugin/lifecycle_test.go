package plugin

import (
	"encoding/json"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
)

func TestDecodeLifecycleAcceptsConfigYAMLRepresentations(t *testing.T) {
	yamlConfig := []byte("data_dir: ./data/credit-manager\nbusy_timeout: 5s\n")
	standard, err := json.Marshal(lifecycleRequest{ConfigYAML: yamlConfig, SchemaVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := json.Marshal(map[string]any{
		"config_yaml":    string(yamlConfig),
		"schema_version": 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	byteValues := make([]int, len(yamlConfig))
	for i, value := range yamlConfig {
		byteValues[i] = int(value)
	}
	array, err := json.Marshal(map[string]any{
		"config_yaml":    byteValues,
		"schema_version": 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	for name, raw := range map[string][]byte{
		"base64": standard,
		"plain":  plain,
		"array":  array,
	} {
		t.Run(name, func(t *testing.T) {
			request, decodeErr := decodeLifecycle(raw)
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			if string(request.ConfigYAML) != string(yamlConfig) {
				t.Fatalf("config yaml = %q", request.ConfigYAML)
			}
			if request.SchemaVersion != 1 {
				t.Fatalf("schema = %d", request.SchemaVersion)
			}
		})
	}
}

func TestNegotiateRPCSchemaCapsAtSupportedVersion(t *testing.T) {
	if got := negotiateRPCSchema(0); got != 1 {
		t.Fatalf("empty host schema = %d, want 1", got)
	}
	if got := negotiateRPCSchema(2); got != 2 {
		t.Fatalf("older host schema = %d, want 2", got)
	}
	if got := negotiateRPCSchema(pluginabi.SchemaVersion + 4); got != pluginabi.SchemaVersion {
		t.Fatalf("newer host schema = %d, want %d", got, pluginabi.SchemaVersion)
	}
}
