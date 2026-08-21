package service

import (
	"context"
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
