package usageparse

import (
	"encoding/json"
	"strings"
)

// ResponseModelFromBody extracts the model identifier declared by an upstream
// response without changing the response sent to the client.
func ResponseModelFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var root map[string]any
	if json.Unmarshal(body, &root) != nil {
		return ""
	}
	return responseModelFromRoot(root)
}

// ResponseModelFromStreamBuffer extracts the declared model from SSE or other
// streamed JSON frames. A terminal Responses API event wins because it carries
// the authoritative completed response after a model may have been updated.
func ResponseModelFromStreamBuffer(buf []byte) string {
	var first, terminal string
	for _, candidate := range extractJSONCandidates(string(buf)) {
		var root map[string]any
		if json.Unmarshal(candidate, &root) != nil {
			continue
		}
		model := responseModelFromRoot(root)
		if model == "" {
			continue
		}
		if responseModelIsTerminal(root) {
			terminal = model
			continue
		}
		if first == "" {
			first = model
		}
	}
	if terminal != "" {
		return terminal
	}
	return first
}

func responseModelFromRoot(root map[string]any) string {
	if root == nil {
		return ""
	}
	if response, ok := root["response"].(map[string]any); ok {
		if model := responseModelFromObject(response); model != "" {
			return model
		}
	}
	if message, ok := root["message"].(map[string]any); ok {
		if model := responseModelFromObject(message); model != "" {
			return model
		}
	}
	return responseModelFromObject(root)
}

func responseModelFromObject(root map[string]any) string {
	return stringField(root, "model", "model_id", "modelId", "model_version", "modelVersion")
}

func responseModelIsTerminal(root map[string]any) bool {
	typ := strings.ToLower(strings.TrimSpace(stringField(root, "type", "event")))
	return strings.HasSuffix(typ, ".completed") ||
		strings.HasSuffix(typ, ".failed") ||
		strings.HasSuffix(typ, ".incomplete") ||
		typ == "message_stop"
}
