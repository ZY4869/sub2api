package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/gin-gonic/gin"
)

const (
	defaultGeminiTextTestPrompt  = accountDaily5HPrompt
	defaultGeminiImageTestPrompt = "Generate a cute orange cat astronaut sticker on a clean pastel background."
)

func defaultGeminiTestModelID(account *Account) string {
	if account != nil && account.IsGeminiVertexSource() {
		return defaultGeminiVertexValidationModel
	}
	return geminicli.DefaultTestModel
}

// testGeminiAccountConnection tests a Gemini account's connection
func (s *AccountTestService) testGeminiAccountConnection(c *gin.Context, account *Account, modelID string, prompt string, sourceProtocol string, simulatedClient string) error {
	ctx := c.Request.Context()
	shouldMimicGeminiCLI := simulatedClient == GatewayClientProfileGeminiCLI

	// Determine the model to use
	testModelID := modelID
	if testModelID == "" {
		testModelID = defaultGeminiTestModelID(account)
	}

	// For API Key accounts with model mapping, map the model
	if account.Type == AccountTypeAPIKey {
		mapping := account.GetModelMapping()
		if len(mapping) > 0 {
			if mappedModel, exists := mapping[testModelID]; exists {
				testModelID = mappedModel
			}
		}
	}
	testModelID = s.resolveTestModelID(ctx, account, testModelID)

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// Create test payload (Gemini format)
	payload := createGeminiTestPayload(testModelID, prompt)

	// Build request based on account type
	var req *http.Request
	var err error

	switch account.Type {
	case AccountTypeAPIKey:
		req, err = s.buildGeminiAPIKeyRequest(ctx, account, testModelID, payload, shouldMimicGeminiCLI)
	case AccountTypeOAuth:
		req, err = s.buildGeminiOAuthRequest(ctx, account, testModelID, payload, shouldMimicGeminiCLI)
	default:
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported account type: %s", account.Type))
	}

	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to build request: %s", err.Error()))
	}

	// Send test_start event
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	s.sendResolvedTestRuntimeMetaEvents(c)

	// Get proxy and execute request
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	tlsProfile := resolveAccountTLSFingerprintProfile(account, s.tlsFingerprintProfileService)

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return s.sendFailedTestResponse(c, ctx, account, resp.StatusCode, body, "API returned")
	}

	// Process SSE stream
	return s.processGeminiStream(c, resp.Body)
}

// routeAntigravityTest 路由 Antigravity 账号的测试请求。
// APIKey 类型走原生协议（与 gateway_handler 路由一致），OAuth/Upstream 走 CRS 中转。

func (s *AccountTestService) buildGeminiAPIKeyRequest(ctx context.Context, account *Account, modelID string, payload []byte, mimicGeminiCLI bool) (*http.Request, error) {
	apiKey := account.GetCredential("api_key")
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("no API key available")
	}

	baseURL := account.GetCredential("base_url")
	if baseURL == "" {
		if account.IsGeminiVertexExpress() {
			baseURL = geminicli.VertexAIBaseURL
		} else {
			baseURL = geminicli.AIStudioBaseURL
		}
	}
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	fullURL := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse",
		strings.TrimRight(normalizedBaseURL, "/"), modelID)
	if account.IsGeminiVertexExpress() {
		modelID = normalizeVertexUpstreamModelID(modelID)
		actionPath, err := account.GeminiVertexExpressModelActionPath(modelID, "streamGenerateContent")
		if err != nil {
			return nil, err
		}
		fullURL = strings.TrimRight(normalizedBaseURL, "/") + actionPath
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if account.IsGeminiVertexExpress() {
		query := req.URL.Query()
		query.Set("key", apiKey)
		query.Set("alt", "sse")
		req.URL.RawQuery = query.Encode()
	} else {
		req.Header.Set("x-goog-api-key", apiKey)
	}
	if mimicGeminiCLI {
		req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
	}

	return req, nil
}

// buildGeminiOAuthRequest builds request for Gemini OAuth accounts
func (s *AccountTestService) buildGeminiOAuthRequest(ctx context.Context, account *Account, modelID string, payload []byte, mimicGeminiCLI bool) (*http.Request, error) {
	if s.geminiTokenProvider == nil {
		return nil, fmt.Errorf("gemini token provider not configured")
	}

	// Get access token (auto-refreshes if needed)
	accessToken, err := s.geminiTokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	if account.IsGeminiVertexAI() {
		modelID = normalizeVertexUpstreamModelID(modelID)
		baseURL := account.GetGeminiVertexBaseURL(geminicli.VertexAIBaseURL)
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		actionPath, err := account.GeminiVertexModelActionPath(modelID, "streamGenerateContent")
		if err != nil {
			return nil, err
		}
		fullURL := fmt.Sprintf("%s%s?alt=sse", strings.TrimRight(normalizedBaseURL, "/"), actionPath)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		if mimicGeminiCLI {
			req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
		}
		return req, nil
	}

	projectID := strings.TrimSpace(account.GetCredential("project_id"))
	if projectID == "" {
		// AI Studio OAuth mode (no project_id): call generativelanguage API directly with Bearer token.
		baseURL := account.GetCredential("base_url")
		if strings.TrimSpace(baseURL) == "" {
			baseURL = geminicli.AIStudioBaseURL
		}
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		fullURL := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse", strings.TrimRight(normalizedBaseURL, "/"), modelID)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)
		if mimicGeminiCLI {
			req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)
		}
		return req, nil
	}

	// Code Assist mode (with project_id)
	return s.buildCodeAssistRequest(ctx, accessToken, projectID, modelID, payload)
}

// buildCodeAssistRequest builds request for Google Code Assist API (used by Gemini CLI and Antigravity)
func (s *AccountTestService) buildCodeAssistRequest(ctx context.Context, accessToken, projectID, modelID string, payload []byte) (*http.Request, error) {
	var inner map[string]any
	if err := json.Unmarshal(payload, &inner); err != nil {
		return nil, err
	}

	wrapped := map[string]any{
		"model":   modelID,
		"project": projectID,
		"request": inner,
	}
	wrappedBytes, _ := json.Marshal(wrapped)

	normalizedBaseURL, err := s.validateUpstreamBaseURL(geminicli.GeminiCliBaseURL)
	if err != nil {
		return nil, err
	}
	fullURL := fmt.Sprintf("%s/v1internal:streamGenerateContent?alt=sse", normalizedBaseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(wrappedBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", geminicli.GeminiCLIUserAgent)

	return req, nil
}

// createGeminiTestPayload creates a minimal test payload for Gemini API.
// Image models use the image-generation path so the frontend can preview the returned image.
func createGeminiTestPayload(modelID string, prompt string) []byte {
	if isImageGenerationModel(modelID) {
		imagePrompt := strings.TrimSpace(prompt)
		if imagePrompt == "" {
			imagePrompt = defaultGeminiImageTestPrompt
		}

		payload := map[string]any{
			"contents": []map[string]any{
				{
					"role": "user",
					"parts": []map[string]any{
						{"text": imagePrompt},
					},
				},
			},
			"generationConfig": map[string]any{
				"responseModalities": []string{"TEXT", "IMAGE"},
				"imageConfig": map[string]any{
					"aspectRatio": "1:1",
				},
			},
		}
		bytes, _ := json.Marshal(payload)
		return bytes
	}

	textPrompt := strings.TrimSpace(prompt)
	if textPrompt == "" {
		textPrompt = defaultGeminiTextTestPrompt
	}

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]any{
					{"text": textPrompt},
				},
			},
		},
		"systemInstruction": map[string]any{
			"parts": []map[string]any{
				{"text": "You are a helpful AI assistant."},
			},
		},
	}
	bytes, _ := json.Marshal(payload)
	return bytes
}

// processGeminiStream processes SSE stream from Gemini API
