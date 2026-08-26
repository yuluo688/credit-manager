package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func readCredentials(raw []byte) (quotaCredentials, bool, error) {
	var v map[string]any
	if json.Unmarshal(raw, &v) != nil {
		return quotaCredentials{}, true, fmt.Errorf("auth configuration is invalid")
	}
	token, evidence, stale := oauthToken(v)
	if !evidence {
		return quotaCredentials{}, false, nil
	}
	if token == "" {
		if stale {
			return quotaCredentials{}, true, fmt.Errorf("OAuth access token is unavailable")
		}
		return quotaCredentials{}, true, fmt.Errorf("OAuth access token is unavailable")
	}
	return quotaCredentials{token: token, accountID: findText(v, "account_id", "accountId", "chatgpt_account_id"), projectID: findText(v, "project_id", "projectId", "project"), userID: findText(v, "user_id", "userId", "userid", "sub", "subject")}, true, nil
}
func oauthToken(v any) (string, bool, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return "", false, false
	}
	evidence, stale := false, false
	for k, x := range m {
		key := strings.ToLower(k)
		switch key {
		case "tokens", "token", "oauth", "claudeaioauth":
			evidence = true
			t, e, s := oauthToken(x)
			if t != "" {
				return t, true, s
			}
			evidence = evidence || e
			stale = stale || s
		case "access_token", "accesstoken":
			evidence = true
			if t, ok := x.(string); ok && strings.TrimSpace(t) != "" {
				return strings.TrimSpace(t), true, stale
			}
		case "refresh_token", "refreshtoken", "id_token", "idtoken":
			evidence = true
			stale = true
		case "type", "auth_type":
			if t, ok := x.(string); ok && strings.Contains(strings.ToLower(t), "oauth") {
				evidence = true
			}
		}
	}
	return "", evidence, stale
}
func findText(v any, keys ...string) string {
	wanted := map[string]bool{}
	for _, k := range keys {
		wanted[strings.ToLower(k)] = true
	}
	var walk func(any) string
	walk = func(x any) string {
		switch y := x.(type) {
		case map[string]any:
			for k, v := range y {
				if wanted[strings.ToLower(k)] {
					if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
						return strings.TrimSpace(s)
					}
				}
			}
			for _, v := range y {
				if r := walk(v); r != "" {
					return r
				}
			}
		case []any:
			for _, v := range y {
				if r := walk(v); r != "" {
					return r
				}
			}
		}
		return ""
	}
	return walk(v)
}

func quotaPlanText(v any, keys ...string) string {
	for _, key := range keys {
		if s := findText(v, key); s != "" {
			return s
		}
	}
	return ""
}

func findBool(v any, keys ...string) (bool, bool) {
	wanted := map[string]bool{}
	for _, k := range keys {
		wanted[strings.ToLower(k)] = true
	}
	var walk func(any) (bool, bool)
	walk = func(x any) (bool, bool) {
		switch y := x.(type) {
		case map[string]any:
			for k, val := range y {
				if !wanted[strings.ToLower(k)] {
					continue
				}
				switch t := val.(type) {
				case bool:
					return t, true
				case string:
					switch strings.ToLower(strings.TrimSpace(t)) {
					case "true", "1", "yes":
						return true, true
					case "false", "0", "no":
						return false, true
					}
				}
			}
			for _, val := range y {
				if b, ok := walk(val); ok {
					return b, true
				}
			}
		case []any:
			for _, val := range y {
				if b, ok := walk(val); ok {
					return b, true
				}
			}
		}
		return false, false
	}
	return walk(v)
}

func quotaPlanFromAuthJSON(raw []byte) string {
	var v map[string]any
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	if plan := quotaPlanText(v, "rate_limit_tier", "rateLimitTier", "subscriptionType", "subscription_type", "chatgpt_plan_type", "plan_type", "tier_id", "tierId"); plan != "" {
		return plan
	}
	if token := findText(v, "id_token", "idToken"); token != "" {
		if claims := decodeJWTPayload(token); claims != nil {
			return quotaPlanText(claims, "chatgpt_plan_type", "plan_type", "subscriptionType", "subscription_type")
		}
	}
	return ""
}

func decodeJWTPayload(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	payload := parts[1]
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		if m := len(payload) % 4; m != 0 {
			payload += strings.Repeat("=", 4-m)
		}
		raw, err = base64.URLEncoding.DecodeString(payload)
		if err != nil {
			return nil
		}
	}
	var v map[string]any
	if json.Unmarshal(raw, &v) != nil {
		return nil
	}
	return v
}

func normalizeQuotaPlan(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}
	compact := strings.ReplaceAll(strings.ReplaceAll(s, "_", "-"), " ", "-")
	switch {
	case strings.Contains(compact, "max-20") || strings.Contains(compact, "max20"):
		return "max_20x"
	case strings.Contains(compact, "max-5") || strings.Contains(compact, "max5"):
		return "max_5x"
	case compact == "max" || strings.Contains(compact, "claude-max"):
		return "max"
	case compact == "pro" || strings.Contains(compact, "claude-pro"):
		return "pro"
	case compact == "plus":
		return "plus"
	case strings.Contains(compact, "team"):
		return "team"
	case strings.Contains(compact, "enterprise"):
		return "enterprise"
	case strings.Contains(compact, "business"):
		return "business"
	case compact == "go":
		return "go"
	case strings.Contains(compact, "standard"):
		return "standard"
	case strings.Contains(compact, "legacy"):
		return "legacy"
	case strings.Contains(compact, "free"):
		return "free"
	case strings.Contains(compact, "supergrok") || strings.Contains(compact, "super-grok"):
		return "super_grok"
	default:
		return strings.TrimSpace(raw)
	}
}

func (s *Service) fetch(ctx context.Context, source AuthQuotaSource, callback, p string, c quotaCredentials) (quotaSnapshot, error) {
	switch p {
	case "codex":
		return codex(ctx, source, callback, c)
	case "claude":
		return claude(ctx, source, callback, c)
	case "antigravity":
		return antigravity(ctx, source, callback, c)
	case "kimi":
		return kimi(ctx, source, callback, c)
	case "xai":
		return xai(ctx, source, callback, c)
	}
	return quotaSnapshot{}, fmt.Errorf("unsupported auth quota provider")
}
func headers(token string) http.Header {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+token)
	h.Set("Accept", "application/json")
	return h
}
func request(ctx context.Context, s AuthQuotaSource, callback, method, url string, h http.Header, b []byte) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r, e := s.DoAuthQuotaHTTP(ctx, callback, AuthQuotaHTTPRequest{Method: method, URL: url, Header: h, Body: b})
	if e != nil {
		return nil, e
	}
	if r.StatusCode < 200 || r.StatusCode > 299 {
		return nil, fmt.Errorf("quota endpoint returned status %d", r.StatusCode)
	}
	var out map[string]any
	if json.Unmarshal(r.Body, &out) != nil {
		return nil, fmt.Errorf("decode quota response")
	}
	return out, nil
}
