package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/yuluo688/credit-manager/internal/service"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type hostAuthQuotaSource struct{}

type hostAuthQuotaHTTPDoStreamRequest struct {
	Method         string      `json:"method"`
	URL            string      `json:"url"`
	Headers        http.Header `json:"headers,omitempty"`
	Body           []byte      `json:"body,omitempty"`
	HostCallbackID string      `json:"host_callback_id,omitempty"`
}

type hostAuthQuotaHTTPDoStreamResponse struct {
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers"`
	StreamID   string      `json:"stream_id"`
}

type hostAuthQuotaHTTPStreamReadRequest struct {
	StreamID       string `json:"stream_id"`
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

type hostAuthQuotaHTTPStreamReadResponse struct {
	Payload []byte `json:"payload"`
	Error   string `json:"error"`
	Done    bool   `json:"done"`
}

type hostAuthQuotaHTTPStreamCloseRequest struct {
	StreamID       string `json:"stream_id"`
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

const maxAuthQuotaHTTPResponseBytes = 1 << 20

func (hostAuthQuotaSource) ListAuthQuotaFiles(context.Context) ([]service.AuthQuotaFile, error) {
	raw, err := callHost(pluginabi.MethodHostAuthList, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("list auth quota files failed")
	}
	var response struct {
		Files []pluginapi.HostAuthFileEntry `json:"files"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode auth quota file list: %w", err)
	}
	files := make([]service.AuthQuotaFile, 0, len(response.Files))
	for _, entry := range response.Files {
		files = append(files, service.AuthQuotaFile{ID: entry.ID, AuthIndex: entry.AuthIndex, Name: entry.Name, Label: entry.Label, Provider: entry.Provider, Type: entry.Type, Email: entry.Email, Account: entry.Account, ModTime: entry.ModTime})
	}
	return files, nil
}

func (hostAuthQuotaSource) GetAuthQuotaJSON(_ context.Context, authIndex string) ([]byte, error) {
	raw, err := callHost(pluginabi.MethodHostAuthGet, pluginapi.HostAuthGetRequest{AuthIndex: authIndex})
	if err != nil {
		return nil, fmt.Errorf("read auth quota configuration failed")
	}
	var response pluginapi.HostAuthGetResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode auth quota configuration: %w", err)
	}
	return append([]byte(nil), response.JSON...), nil
}

func (hostAuthQuotaSource) DoAuthQuotaHTTP(ctx context.Context, hostCallbackID string, request service.AuthQuotaHTTPRequest) (service.AuthQuotaHTTPResponse, error) {
	if err := ctx.Err(); err != nil {
		return service.AuthQuotaHTTPResponse{}, err
	}
	if !allowedAuthQuotaRequest(request.Method, request.URL) {
		return service.AuthQuotaHTTPResponse{}, fmt.Errorf("auth quota request is not allowed")
	}
	raw, err := callHost(pluginabi.MethodHostHTTPDoStream, hostAuthQuotaHTTPDoStreamRequest{Method: request.Method, URL: request.URL, Headers: request.Header, Body: request.Body, HostCallbackID: hostCallbackID})
	if err != nil {
		return service.AuthQuotaHTTPResponse{}, fmt.Errorf("start auth quota request failed")
	}
	var response hostAuthQuotaHTTPDoStreamResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return service.AuthQuotaHTTPResponse{}, fmt.Errorf("decode auth quota response: %w", err)
	}
	if strings.TrimSpace(response.StreamID) == "" {
		return service.AuthQuotaHTTPResponse{}, fmt.Errorf("auth quota response did not provide a stream")
	}
	defer func() {
		_, _ = callHost(pluginabi.MethodHostHTTPStreamClose, hostAuthQuotaHTTPStreamCloseRequest{StreamID: response.StreamID, HostCallbackID: hostCallbackID})
	}()

	var body bytes.Buffer
	for {
		if err := ctx.Err(); err != nil {
			return service.AuthQuotaHTTPResponse{}, err
		}
		chunkRaw, err := callHost(pluginabi.MethodHostHTTPStreamRead, hostAuthQuotaHTTPStreamReadRequest{StreamID: response.StreamID, HostCallbackID: hostCallbackID})
		if err != nil {
			return service.AuthQuotaHTTPResponse{}, fmt.Errorf("read auth quota response failed")
		}
		var chunk hostAuthQuotaHTTPStreamReadResponse
		if err := json.Unmarshal(chunkRaw, &chunk); err != nil {
			return service.AuthQuotaHTTPResponse{}, fmt.Errorf("decode auth quota response chunk: %w", err)
		}
		if chunk.Error != "" {
			return service.AuthQuotaHTTPResponse{}, fmt.Errorf("auth quota response stream failed")
		}
		if len(chunk.Payload) > maxAuthQuotaHTTPResponseBytes-body.Len() {
			return service.AuthQuotaHTTPResponse{}, fmt.Errorf("auth quota response exceeds 1 MiB")
		}
		_, _ = body.Write(chunk.Payload)
		if chunk.Done {
			return service.AuthQuotaHTTPResponse{StatusCode: response.StatusCode, Header: response.Headers, Body: body.Bytes()}, nil
		}
	}
}

func allowedAuthQuotaRequest(method, rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" {
		return false
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	switch parsed.Host {
	case "chatgpt.com":
		return method == http.MethodGet && (parsed.Path == "/backend-api/wham/usage" || parsed.Path == "/backend-api/wham/rate-limit-reset-credits")
	case "api.anthropic.com":
		return method == http.MethodGet && (parsed.Path == "/api/oauth/usage" || parsed.Path == "/api/oauth/profile")
	case "cloudcode-pa.googleapis.com", "daily-cloudcode-pa.googleapis.com", "daily-cloudcode-pa.sandbox.googleapis.com", "autopush-cloudcode-pa.googleapis.com":
		return method == http.MethodPost && parsed.Path == "/v1internal:retrieveUserQuotaSummary"
	case "api.kimi.com":
		return method == http.MethodGet && parsed.Path == "/coding/v1/usages"
	case "cli-chat-proxy.grok.com":
		return method == http.MethodGet && parsed.Path == "/v1/billing"
	case "api.x.ai":
		return method == http.MethodGet && parsed.Path == "/v1/billing"
	}
	return false
}
