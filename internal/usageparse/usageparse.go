package usageparse

import (
	"encoding/json"
	"strings"

	"github.com/yuluo688/credit-manager/internal/money"
)

// Result is a best-effort extraction of official token usage fields.
type Result struct {
	Usage   money.TokenUsage
	Found   bool
	Partial bool
	Source  string
}

// FromResponseBody extracts usage from non-streaming JSON bodies across major protocols.
func FromResponseBody(body []byte, format string) Result {
	if len(body) == 0 {
		return Result{}
	}
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return Result{}
	}
	format = strings.ToLower(strings.TrimSpace(format))
	switch {
	case strings.Contains(format, "claude"):
		if r := fromClaude(root); r.Found {
			return r
		}
	case strings.Contains(format, "gemini"):
		if r := fromGemini(root); r.Found {
			return r
		}
	case strings.Contains(format, "response"), strings.Contains(format, "codex"):
		if r := fromResponses(root); r.Found {
			return r
		}
	}
	if r := fromOpenAI(root); r.Found {
		return r
	}
	if r := fromClaude(root); r.Found {
		return r
	}
	if r := fromGemini(root); r.Found {
		return r
	}
	if r := fromResponses(root); r.Found {
		return r
	}
	return Result{}
}

// FromStreamBuffer extracts usage from accumulated SSE/stream text.
func FromStreamBuffer(buf []byte, format string) Result {
	if len(buf) == 0 {
		return Result{}
	}
	// Prefer the last complete JSON object that contains usage.
	text := string(buf)
	// Scan for data: lines and raw JSON blobs.
	candidates := extractJSONCandidates(text)
	var best Result
	for i := len(candidates) - 1; i >= 0; i-- {
		r := FromResponseBody(candidates[i], format)
		if r.Found {
			return r
		}
		// Responses API often nests usage under response.
		var root map[string]any
		if err := json.Unmarshal(candidates[i], &root); err != nil {
			continue
		}
		if nested, ok := root["response"].(map[string]any); ok {
			if r := fromResponses(nested); r.Found {
				return r
			}
			if r := fromOpenAI(nested); r.Found {
				return r
			}
		}
		if usage, ok := root["usage"].(map[string]any); ok {
			r := mapUsage(usage, "stream")
			if r.Found {
				return r
			}
		}
		if r.Partial && !best.Found {
			best = r
		}
	}
	return best
}

func extractJSONCandidates(text string) [][]byte {
	var out [][]byte
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "data: [DONE]" {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		if strings.HasPrefix(line, "{") {
			out = append(out, []byte(line))
		}
	}
	// Also try whole buffer as JSON.
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "{") {
		out = append(out, []byte(trimmed))
	}
	return out
}

func fromOpenAI(root map[string]any) Result {
	usage, _ := root["usage"].(map[string]any)
	if usage == nil {
		return Result{}
	}
	return mapUsage(usage, "openai")
}

func fromClaude(root map[string]any) Result {
	usage, _ := root["usage"].(map[string]any)
	if usage == nil {
		return Result{}
	}
	r := Result{Source: "claude"}
	r.Usage.Input = int64Field(usage, "input_tokens")
	r.Usage.Output = int64Field(usage, "output_tokens")
	r.Usage.CacheRead = int64Field(usage, "cache_read_input_tokens")
	r.Usage.CacheCreation = int64Field(usage, "cache_creation_input_tokens")
	if r.Usage.CacheRead == 0 {
		r.Usage.CacheRead = int64Field(usage, "cache_read_tokens")
	}
	if r.Usage.CacheCreation == 0 {
		r.Usage.CacheCreation = int64Field(usage, "cache_creation_tokens")
	}
	r.Found = r.Usage.Input > 0 || r.Usage.Output > 0 || r.Usage.CacheRead > 0 || r.Usage.CacheCreation > 0
	r.Partial = r.Found && (r.Usage.Input == 0 || r.Usage.Output == 0)
	return r
}

func fromGemini(root map[string]any) Result {
	usage, _ := root["usageMetadata"].(map[string]any)
	if usage == nil {
		usage, _ = root["usage_metadata"].(map[string]any)
	}
	if usage == nil {
		return Result{}
	}
	r := Result{Source: "gemini"}
	r.Usage.Input = int64Field(usage, "promptTokenCount", "prompt_token_count")
	r.Usage.Output = int64Field(usage, "candidatesTokenCount", "candidates_token_count")
	r.Usage.Reasoning = int64Field(usage, "thoughtsTokenCount", "thoughts_token_count")
	r.Usage.Cached = int64Field(usage, "cachedContentTokenCount", "cached_content_token_count")
	r.Found = r.Usage.Input > 0 || r.Usage.Output > 0 || r.Usage.Reasoning > 0 || r.Usage.Cached > 0
	r.Partial = r.Found && (r.Usage.Input == 0 || r.Usage.Output == 0)
	return r
}

func fromResponses(root map[string]any) Result {
	usage, _ := root["usage"].(map[string]any)
	if usage == nil {
		return Result{}
	}
	r := Result{Source: "responses"}
	r.Usage.Input = int64Field(usage, "input_tokens", "prompt_tokens")
	r.Usage.Output = int64Field(usage, "output_tokens", "completion_tokens")
	if details, ok := usage["input_tokens_details"].(map[string]any); ok {
		r.Usage.CacheRead = int64Field(details, "cached_tokens")
	}
	if details, ok := usage["output_tokens_details"].(map[string]any); ok {
		r.Usage.Reasoning = int64Field(details, "reasoning_tokens")
	}
	r.Found = r.Usage.Input > 0 || r.Usage.Output > 0 || r.Usage.CacheRead > 0 || r.Usage.Reasoning > 0
	r.Partial = r.Found && (r.Usage.Input == 0 || r.Usage.Output == 0)
	return r
}

func mapUsage(usage map[string]any, source string) Result {
	r := Result{Source: source}
	r.Usage.Input = int64Field(usage, "prompt_tokens", "input_tokens")
	r.Usage.Output = int64Field(usage, "completion_tokens", "output_tokens")
	r.Usage.Reasoning = int64Field(usage, "reasoning_tokens")
	r.Usage.Cached = int64Field(usage, "cached_tokens")
	if details, ok := usage["prompt_tokens_details"].(map[string]any); ok {
		if r.Usage.Cached == 0 {
			r.Usage.Cached = int64Field(details, "cached_tokens")
		}
		r.Usage.CacheRead = int64Field(details, "cached_tokens")
	}
	if details, ok := usage["completion_tokens_details"].(map[string]any); ok {
		if r.Usage.Reasoning == 0 {
			r.Usage.Reasoning = int64Field(details, "reasoning_tokens")
		}
	}
	r.Found = r.Usage.Input > 0 || r.Usage.Output > 0 || r.Usage.Cached > 0 || r.Usage.Reasoning > 0
	r.Partial = r.Found && (r.Usage.Input == 0 || r.Usage.Output == 0)
	return r
}

func int64Field(object map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := object[key]; ok {
			switch typed := value.(type) {
			case float64:
				if typed < 0 {
					return 0
				}
				return int64(typed)
			case int64:
				if typed < 0 {
					return 0
				}
				return typed
			case int:
				if typed < 0 {
					return 0
				}
				return int64(typed)
			case json.Number:
				n, err := typed.Int64()
				if err != nil || n < 0 {
					return 0
				}
				return n
			}
		}
	}
	return 0
}
