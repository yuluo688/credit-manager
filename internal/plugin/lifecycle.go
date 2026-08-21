package plugin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
)

func decodeLifecycle(raw []byte) (lifecycleRequest, error) {
	if len(raw) == 0 {
		return lifecycleRequest{}, nil
	}
	var envelope struct {
		ConfigYAML    json.RawMessage `json:"config_yaml"`
		SchemaVersion uint32          `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return lifecycleRequest{}, fmt.Errorf("decode lifecycle request: %w", err)
	}
	configYAML, err := decodeLifecycleConfigYAML(envelope.ConfigYAML)
	if err != nil {
		return lifecycleRequest{}, err
	}
	return lifecycleRequest{ConfigYAML: configYAML, SchemaVersion: envelope.SchemaVersion}, nil
}

func decodeLifecycleConfigYAML(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if decoded, decodeErr := base64.StdEncoding.DecodeString(text); decodeErr == nil && strings.Contains(string(decoded), ":") {
			return decoded, nil
		}
		return []byte(text), nil
	}
	var bytes []byte
	if err := json.Unmarshal(raw, &bytes); err == nil {
		return bytes, nil
	}
	return nil, fmt.Errorf("config_yaml must be a base64/plain string or byte array")
}

func negotiateRPCSchema(hostSchema uint32) uint32 {
	if hostSchema == 0 {
		return 1
	}
	if hostSchema < pluginabi.SchemaVersion {
		return hostSchema
	}
	return pluginabi.SchemaVersion
}
