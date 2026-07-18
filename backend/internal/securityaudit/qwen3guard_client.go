package securityaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const maxGuardResponseBytes = 256 * 1024

type OpenAICompatibleScanner struct {
	clients sync.Map
}

func NewOpenAICompatibleScanner() *OpenAICompatibleScanner {
	return &OpenAICompatibleScanner{}
}

func (s *OpenAICompatibleScanner) Scan(ctx context.Context, endpoint ActiveEndpoint, chunk string, enabledScanners []string) (*NormalizedResult, error) {
	client := s.clientFor(endpoint)
	requestURL, err := ChatCompletionsURL(endpoint.BaseURL)
	if err != nil {
		return nil, &GuardError{Code: ErrorCodeUnavailable, Cause: err}
	}
	req, err := newGuardRequest(ctx, requestURL, endpoint, chunk)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, networkGuardError(err)
	}
	defer func() { _ = resp.Body.Close() }()
	result, err := parseGuardHTTPResponse(resp, enabledScanners)
	if err != nil {
		return nil, err
	}
	result.GuardEndpointID = endpoint.ID
	result.ScannerVersion = endpoint.Model
	return result, nil
}

func (s *OpenAICompatibleScanner) clientFor(endpoint ActiveEndpoint) *http.Client {
	key := fmt.Sprintf("%s|%s|%d", endpoint.ID, endpoint.BaseURL, endpoint.TimeoutMS)
	if cached, ok := s.clients.Load(key); ok {
		if client, ok := cached.(*http.Client); ok {
			return client
		}
	}
	client, err := NewSecureHTTPClient(endpoint)
	if err != nil {
		timeout := time.Duration(DefaultTimeoutMS) * time.Millisecond
		client = &http.Client{Timeout: timeout}
	}
	actual, _ := s.clients.LoadOrStore(key, client)
	if typed, ok := actual.(*http.Client); ok {
		return typed
	}
	return client
}

func newGuardRequest(ctx context.Context, requestURL string, endpoint ActiveEndpoint, chunk string) (*http.Request, error) {
	payload := map[string]any{
		"model":       strings.TrimSpace(endpoint.Model),
		"messages":    []map[string]string{{"role": "user", "content": chunk}},
		"temperature": 0,
		"max_tokens":  64,
		"seed":        42,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, &GuardError{Code: ErrorCodeInvalidResponse, Cause: err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, &GuardError{Code: ErrorCodeUnavailable, Cause: err}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token := strings.TrimSpace(endpoint.Token); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

func parseGuardHTTPResponse(resp *http.Response, enabledScanners []string) (*NormalizedResult, error) {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &GuardError{Code: ErrorCodeUnavailable, HTTPStatus: resp.StatusCode, Retryable: isRetryableGuardStatus(resp.StatusCode)}
	}
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxGuardResponseBytes+1))
	if err != nil {
		return nil, &GuardError{Code: ErrorCodeUnavailable, Retryable: true, Cause: err}
	}
	if int64(len(responseBody)) > maxGuardResponseBytes {
		return nil, &GuardError{Code: ErrorCodeInvalidResponse}
	}
	content, err := extractOpenAIContent(responseBody)
	if err != nil {
		return nil, &GuardError{Code: ErrorCodeInvalidResponse, Cause: err}
	}
	return ParseQwen3Guard(content, enabledScanners)
}

func isRetryableGuardStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}
