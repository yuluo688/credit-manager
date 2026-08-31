package service

import (
	"encoding/json"
	"strings"
)

func estimateTokens(body []byte, defaultOutput, maxTotal int64) (input, output int64) {
	output = defaultOutput
	if body == nil {
		input = 1
	} else {
		// Conservative character-based upper bound: 1 token per rune-ish byte pair, floor 1.
		input = int64(len(body)/2) + 1
	}
	// Prefer explicit protocol limits when present.
	if explicit := extractOutputLimit(body); explicit > 0 {
		output = explicit
	}
	if input > maxTotal {
		input = maxTotal
	}
	if output > maxTotal {
		output = maxTotal
	}
	if input+output > maxTotal {
		// Keep output preference and shrink input.
		if output >= maxTotal {
			output = maxTotal
			input = 0
		} else {
			input = maxTotal - output
		}
	}
	if input <= 0 {
		input = 1
	}
	if output <= 0 {
		output = 1
	}
	return input, output
}

func extractModel(body []byte) string {
	return extractString(body, "model")
}

func extractServiceTier(body []byte) string {
	return extractString(body, "service_tier", "tier")
}

func extractString(body []byte, keys ...string) string {
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return ""
	}
	for _, key := range keys {
		if value, ok := root[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func extractOutputLimit(body []byte) int64 {
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return 0
	}
	for _, key := range []string{"max_tokens", "max_output_tokens", "max_completion_tokens"} {
		if value, ok := root[key]; ok {
			switch typed := value.(type) {
			case float64:
				if typed > 0 {
					return int64(typed)
				}
			case int64:
				if typed > 0 {
					return typed
				}
			case int:
				if typed > 0 {
					return int64(typed)
				}
			}
		}
	}
	if gen, ok := root["generationConfig"].(map[string]any); ok {
		if value, ok := gen["maxOutputTokens"].(float64); ok && value > 0 {
			return int64(value)
		}
	}
	return 0
}

func extractImageCount(body []byte) int64 {
	const maxImages int64 = 32
	if len(body) == 0 {
		return 1
	}
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return 1
	}
	for _, key := range []string{"n", "n_images", "num_images", "image_count"} {
		if value, ok := root[key]; ok {
			if count := positiveInt64(value); count > 0 {
				if count > maxImages {
					return maxImages
				}
				return count
			}
		}
	}
	return 1
}

func positiveInt64(value any) int64 {
	switch typed := value.(type) {
	case float64:
		if typed > 0 {
			return int64(typed)
		}
	case int64:
		if typed > 0 {
			return typed
		}
	case int:
		if typed > 0 {
			return int64(typed)
		}
	case json.Number:
		if parsed, err := typed.Int64(); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 0
}
