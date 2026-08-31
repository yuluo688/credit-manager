package usageparse

import (
	"encoding/json"
	"strings"

	"github.com/yuluo688/credit-manager/internal/money"
)

// Result is a best-effort extraction of official token usage fields.
type Result struct {
	Usage       money.TokenUsage
	Found       bool
	Partial     bool
	Source      string
	ServiceTier string
}

func (r Result) withServiceTier(root map[string]any) Result {
	if r.ServiceTier == "" {
		r.ServiceTier = serviceTierFrom(root)
	}
	return r
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
			return r.withServiceTier(root)
		}
	case strings.Contains(format, "gemini"):
		if r := fromGemini(root); r.Found {
			return r.withServiceTier(root)
		}
	case strings.Contains(format, "response"), strings.Contains(format, "codex"):
		if r := fromResponses(root); r.Found {
			return r.withServiceTier(root)
		}
	}
	if r := fromOpenAI(root); r.Found {
		return r.withServiceTier(root)
	}
	if r := fromClaude(root); r.Found {
		return r.withServiceTier(root)
	}
	if r := fromGemini(root); r.Found {
		return r.withServiceTier(root)
	}
	if r := fromResponses(root); r.Found {
		return r.withServiceTier(root)
	}
	return Result{}.withServiceTier(root)
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
	lastTier := serviceTierFromCandidates(candidates)
	var best Result
	for i := len(candidates) - 1; i >= 0; i-- {
		r := FromResponseBody(candidates[i], format)
		if r.Found {
			return r.withLastServiceTier(lastTier)
		}
		// Responses API often nests usage under response.
		var root map[string]any
		if err := json.Unmarshal(candidates[i], &root); err != nil {
			continue
		}
		if nested, ok := root["response"].(map[string]any); ok {
			if r := fromResponses(nested); r.Found {
				return r.withLastServiceTier(lastTier)
			}
			if r := fromOpenAI(nested); r.Found {
				return r.withLastServiceTier(lastTier)
			}
		}
		if usage, ok := root["usage"].(map[string]any); ok {
			r := mapUsage(usage, "stream")
			if r.Found {
				return r.withLastServiceTier(lastTier)
			}
		}
		if r := mapUsage(root, "stream"); r.Found {
			return r.withLastServiceTier(lastTier)
		}
		if r.Partial && !best.Found {
			best = r
		}
	}
	return best.withLastServiceTier(lastTier)
}

func extractJSONCandidates(text string) [][]byte {
	spaced := insertSSEBreaks(text)
	var out [][]byte
	for _, line := range strings.Split(spaced, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "[DONE]" || line == "data: [DONE]" {
			continue
		}
		if strings.HasPrefix(line, "data:") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		if strings.HasPrefix(line, "{") {
			out = append(out, []byte(line))
		}
	}
	trimmed := strings.TrimSpace(spaced)
	if strings.HasPrefix(trimmed, "{") {
		out = append(out, []byte(trimmed))
	}
	out = append(out, extractJSONObjects(text)...)
	return out
}

func insertSSEBreaks(text string) string {
	replacer := strings.NewReplacer("event:", "\nevent:", "data:", "\ndata:")
	return replacer.Replace(text)
}

func extractJSONObjects(text string) [][]byte {
	var out [][]byte
	for i := 0; i < len(text); i++ {
		if text[i] != '{' {
			continue
		}
		depth := 0
		inString := false
		escape := false
		for j := i; j < len(text); j++ {
			c := text[j]
			if inString {
				if escape {
					escape = false
					continue
				}
				if c == '\\' {
					escape = true
					continue
				}
				if c == '"' {
					inString = false
				}
				continue
			}
			switch c {
			case '"':
				inString = true
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					out = append(out, []byte(text[i:j+1]))
					i = j
				}
			}
			if depth == 0 && c == '}' {
				break
			}
		}
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
	r.Usage.ReportedTotal = int64Field(usage, "total_tokens")
	r.Found = usagePresent(r.Usage)
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
	r.Usage.ReportedTotal = int64Field(usage, "totalTokenCount", "total_token_count")
	r.Found = usagePresent(r.Usage)
	r.Partial = r.Found && (r.Usage.Input == 0 || r.Usage.Output == 0)
	return r
}

func fromResponses(root map[string]any) Result {
	for depth := 0; depth < 4 && root != nil; depth++ {
		usage, _ := root["usage"].(map[string]any)
		if usage != nil {
			r := Result{Source: "responses"}
			r.Usage.Input = int64Field(usage, "input_tokens", "prompt_tokens")
			r.Usage.Output = int64Field(usage, "output_tokens", "completion_tokens")
			if details, ok := usage["input_tokens_details"].(map[string]any); ok {
				r.Usage.CacheRead = int64Field(details, "cached_tokens")
			}
			if details, ok := usage["output_tokens_details"].(map[string]any); ok {
				r.Usage.Reasoning = int64Field(details, "reasoning_tokens")
			}
			r.Usage.ReportedTotal = int64Field(usage, "total_tokens")
			r.Found = usagePresent(r.Usage)
			r.Partial = r.Found && (r.Usage.Input == 0 || r.Usage.Output == 0)
			return r
		}
		nested, ok := root["response"].(map[string]any)
		if !ok {
			return Result{}
		}
		root = nested
	}
	return Result{}
}

func mapUsage(usage map[string]any, source string) Result {
	r := Result{Source: source}
	r.Usage.Input = int64Field(usage, "prompt_tokens", "input_tokens")
	r.Usage.Output = int64Field(usage, "completion_tokens", "output_tokens")
	r.Usage.Reasoning = int64Field(usage, "reasoning_tokens")
	r.Usage.Cached = int64Field(usage, "cached_tokens")
	if details, ok := usage["prompt_tokens_details"].(map[string]any); ok {
		cached := int64Field(details, "cached_tokens")
		if r.Usage.Cached == 0 {
			r.Usage.Cached = cached
		}
		r.Usage.CacheRead = cached
	}
	if details, ok := usage["completion_tokens_details"].(map[string]any); ok {
		if r.Usage.Reasoning == 0 {
			r.Usage.Reasoning = int64Field(details, "reasoning_tokens")
		}
	}
	r.Usage.ReportedTotal = int64Field(usage, "total_tokens")
	r.Found = usagePresent(r.Usage)
	r.Partial = r.Found && (r.Usage.Input == 0 || r.Usage.Output == 0)
	return r
}

func usagePresent(usage money.TokenUsage) bool {
	return usage.HasTokens()
}

func (r Result) withLastServiceTier(tier string) Result {
	if strings.TrimSpace(tier) != "" {
		r.ServiceTier = strings.TrimSpace(tier)
	}
	return r
}

func serviceTierFromCandidates(candidates [][]byte) string {
	last := ""
	for _, candidate := range candidates {
		var root map[string]any
		if json.Unmarshal(candidate, &root) != nil {
			continue
		}
		if t := serviceTierFrom(root); t != "" {
			last = t
		}
	}
	return last
}

func serviceTierFrom(root map[string]any) string {
	if root == nil {
		return ""
	}
	if t := stringField(root, "service_tier", "serviceTier", "tier"); t != "" {
		return t
	}
	if nested, ok := root["response"].(map[string]any); ok {
		if t := stringField(nested, "service_tier", "serviceTier", "tier"); t != "" {
			return t
		}
	}
	if usage, ok := root["usage"].(map[string]any); ok {
		if t := stringField(usage, "service_tier", "serviceTier"); t != "" {
			return t
		}
	}
	return ""
}

func stringField(object map[string]any, keys ...string) string {
	for _, key := range keys {
		if text, ok := object[key].(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
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
