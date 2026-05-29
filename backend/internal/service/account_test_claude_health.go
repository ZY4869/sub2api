package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	testClaudeAPIURL = "https://api.anthropic.com/v1/messages?beta=true"
)

func generateSessionString() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	hex64 := hex.EncodeToString(b)
	sessionUUID := uuid.New().String()
	uaVersion := ExtractCLIVersion(claude.DefaultHeaders["User-Agent"])
	return FormatMetadataUserID(hex64, "", sessionUUID, uaVersion), nil
}

// createTestPayload creates a Claude Code style test request payload
func createTestPayload(modelID string) (map[string]any, error) {
	sessionID, err := generateSessionString()
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"model": modelID,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": accountDaily5HPrompt,
						"cache_control": map[string]string{
							"type": "ephemeral",
						},
					},
				},
			},
		},
		"system": []map[string]any{
			{
				"type": "text",
				"text": claudeCodeSystemPrompt,
				"cache_control": map[string]string{
					"type": "ephemeral",
				},
			},
		},
		"metadata": map[string]string{
			"user_id": sessionID,
		},
		"max_tokens":  1024,
		"temperature": 1,
		"stream":      true,
	}, nil
}

func createAnthropicStandardTestPayload(modelID string) map[string]any {
	return map[string]any{
		"model": modelID,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": accountDaily5HPrompt,
			},
		},
		"max_tokens": 64,
		"stream":     true,
	}
}

func (s *AccountTestService) testClaudeAccountConnection(c *gin.Context, account *Account, modelID string, sourceProtocol string, simulatedClient string) error {
	if account != nil && RoutingPlatformForAccount(account) == PlatformKiro {
		return s.testKiroAccountConnection(c, account, modelID)
	}
	ctx := c.Request.Context()
	shouldMimicClaudeClient := IsClaudeClientMimicEnabled(account, sourceProtocol)

	// Determine the model to use
	testModelID := modelID
	if testModelID == "" {
		testModelID = claude.DefaultTestModel
	}

	// API Key 账号测试连接时也需要应用通配符模型映射。
	if account.Type == "apikey" {
		testModelID = account.GetMappedModel(testModelID)
	}

	// Bedrock accounts use a separate test path
	if account.IsBedrock() {
		return s.testBedrockAccountConnection(c, ctx, account, testModelID)
	}
	testModelID = s.resolveTestModelID(ctx, account, testModelID)

	// Determine authentication method and API URL
	var authToken string
	var useBearer bool
	var apiURL string
	var err error

	if account.IsOAuth() {
		// OAuth or Setup Token - use Bearer token
		useBearer = true
		apiURL = testClaudeAPIURL
		if account.Type == AccountTypeOAuth && s.claudeTokenProvider != nil {
			authToken, err = s.claudeTokenProvider.GetAccessToken(ctx, account)
			if err != nil {
				return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to get access token: %s", err.Error()))
			}
		} else {
			authToken = account.GetCredential("access_token")
		}
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No access token available")
		}
	} else if account.Type == "apikey" {
		// API Key - use x-api-key header
		useBearer = false
		authToken = account.GetCredential("api_key")
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No API key available")
		}

		baseURL := account.GetBaseURL()
		if baseURL == "" {
			baseURL = "https://api.anthropic.com"
		}
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid base URL: %s", err.Error()))
		}
		apiURL = strings.TrimSuffix(normalizedBaseURL, "/") + "/v1/messages"
		if account.Platform != PlatformDeepSeek {
			apiURL += "?beta=true"
		}
	} else {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported account type: %s", account.Type))
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	var payloadBytes []byte
	if shouldMimicClaudeClient || useBearer {
		payload, payloadErr := createTestPayload(testModelID)
		if payloadErr != nil {
			return s.sendErrorAndEnd(c, "Failed to create test payload")
		}
		payloadBytes, _ = json.Marshal(payload)
	} else {
		payloadBytes, _ = json.Marshal(createAnthropicStandardTestPayload(testModelID))
	}

	// Send test_start event
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	s.sendResolvedTestRuntimeMetaEvents(c)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}

	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	if shouldMimicClaudeClient || useBearer {
		for key, value := range claude.DefaultHeaders {
			req.Header.Set(key, value)
		}
	}

	// Set authentication header
	if useBearer {
		req.Header.Set("anthropic-beta", claude.DefaultBetaHeader)
		req.Header.Set("Authorization", "Bearer "+authToken)
	} else {
		if shouldMimicClaudeClient {
			req.Header.Set("anthropic-beta", claude.APIKeyBetaHeader)
		}
		req.Header.Set("x-api-key", authToken)
	}

	// Get proxy URL
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
	return s.processClaudeStream(c, resp.Body)
}

func (s *AccountTestService) testKiroAccountConnection(c *gin.Context, account *Account, modelID string) error {
	ctx := c.Request.Context()
	testModelID := modelID
	if testModelID == "" {
		testModelID = claude.DefaultTestModel
	}
	testModelID = s.resolveTestModelID(ctx, account, testModelID)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	payload, err := createTestPayload(testModelID)
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create test payload")
	}
	payloadBytes, _ := json.Marshal(payload)
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})

	runtime := NewKiroRuntimeService(s.accountRepo, s.httpUpstream, s.claudeTokenProvider)
	runtime.SetTLSFingerprintProfileService(s.tlsFingerprintProfileService)
	result, err := runtime.ExecuteClaude(ctx, account, KiroRuntimeExecuteInput{
		Body:           payloadBytes,
		ModelID:        testModelID,
		Stream:         true,
		RequestHeaders: c.Request.Header,
	})
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	if result == nil || result.Response == nil {
		return s.sendErrorAndEnd(c, "Kiro runtime returned empty response")
	}

	resp := result.Response
	defer func() { _ = resp.Body.Close() }()
	s.sendKiroRuntimeMetaEvents(c, result)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return s.sendFailedTestResponse(c, ctx, account, resp.StatusCode, body, "API returned")
	}

	return s.processClaudeStream(c, resp.Body)
}

func (s *AccountTestService) sendKiroRuntimeMetaEvents(c *gin.Context, result *KiroRuntimeExecuteResult) {
	if c == nil || result == nil {
		return
	}
	emitMeta := func(text string, data map[string]any) {
		s.sendEvent(c, TestEvent{
			Type: "content",
			Text: text,
			Data: data,
		})
	}

	emitMeta(
		fmt.Sprintf("Kiro runtime region: %s", result.Region),
		map[string]any{
			"kind":   "runtime_meta",
			"key":    "resolved_region",
			"value":  result.Region,
			"source": "kiro_runtime",
		},
	)
	emitMeta(
		fmt.Sprintf("Kiro runtime endpoint: %s (%s)", result.Endpoint.Name, result.Endpoint.URL),
		map[string]any{
			"kind":   "runtime_meta",
			"key":    "endpoint",
			"value":  result.Endpoint.URL,
			"label":  result.Endpoint.Name,
			"source": "kiro_runtime",
		},
	)
	emitMeta(
		fmt.Sprintf("Kiro endpoint fallback: %t", result.FallbackUsed),
		map[string]any{
			"kind":   "runtime_meta",
			"key":    "fallback",
			"value":  result.FallbackUsed,
			"source": "kiro_runtime",
		},
	)
	emitMeta(
		fmt.Sprintf("Kiro profile ARN present: %t", strings.TrimSpace(result.ProfileARN) != ""),
		map[string]any{
			"kind":   "runtime_meta",
			"key":    "profile_arn_present",
			"value":  strings.TrimSpace(result.ProfileARN) != "",
			"source": "kiro_runtime",
		},
	)
}

// testBedrockAccountConnection tests a Bedrock (SigV4 or API Key) account using non-streaming invoke
func (s *AccountTestService) testBedrockAccountConnection(c *gin.Context, ctx context.Context, account *Account, testModelID string) error {
	region := bedrockRuntimeRegion(account)
	resolvedModelID, ok := ResolveBedrockModelID(account, testModelID)
	if !ok {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported Bedrock model: %s", testModelID))
	}
	testModelID = resolvedModelID

	// Set SSE headers (test UI expects SSE)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	// Create a minimal Bedrock-compatible payload (no stream, no cache_control)
	bedrockPayload := map[string]any{
		"anthropic_version": "bedrock-2023-05-31",
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": accountDaily5HPrompt,
					},
				},
			},
		},
		"max_tokens":  256,
		"temperature": 1,
	}
	bedrockBody, _ := json.Marshal(bedrockPayload)

	// Use non-streaming endpoint (response is standard Claude JSON)
	apiURL := BuildBedrockURL(region, testModelID, false)

	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bedrockBody))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	// Sign or set auth based on account type
	if account.IsBedrockAPIKey() {
		apiKey := account.GetCredential("api_key")
		if apiKey == "" {
			return s.sendErrorAndEnd(c, "No API key available")
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
	} else {
		signer, err := NewBedrockSignerFromAccount(account)
		if err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to create Bedrock signer: %s", err.Error()))
		}
		if err := signer.SignRequest(ctx, req, bedrockBody); err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to sign request: %s", err.Error()))
		}
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, nil)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return s.sendFailedTestResponse(c, ctx, account, resp.StatusCode, body, "API returned")
	}

	// Bedrock non-streaming response is standard Claude JSON, extract the text
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to parse response: %s", err.Error()))
	}

	text := ""
	if len(result.Content) > 0 {
		text = result.Content[0].Text
	}
	if text == "" {
		text = "(empty response)"
	}

	s.sendEvent(c, TestEvent{Type: "content", Text: text})
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}

// testOpenAIAccountConnection tests an OpenAI account's connection
