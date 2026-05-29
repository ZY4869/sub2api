package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/gin-gonic/gin"
)

func cloneAccountForBackgroundProbe(account *Account) *Account {
	if account == nil {
		return nil
	}

	cloned := *account
	cloned.Credentials = cloneStringAnyMap(account.Credentials)
	cloned.Extra = cloneStringAnyMap(account.Extra)
	cloned.Proxy = nil
	if account.Proxy != nil {
		proxy := *account.Proxy
		cloned.Proxy = &proxy
	}

	cloned.modelMappingCache = nil
	cloned.modelMappingCacheReady = false
	cloned.modelMappingCacheCredentialsPtr = 0
	cloned.modelMappingCacheRawPtr = 0
	cloned.modelMappingCacheRawLen = 0
	cloned.modelMappingCacheRawSig = 0

	return &cloned
}

func (s *AccountTestService) refreshOpenAIKnownModelsSnapshot(account *Account) {
	if s == nil || s.accountRepo == nil || s.accountModelImportService == nil || account == nil || !account.IsOpenAIOAuth() {
		return
	}

	cloned := cloneAccountForBackgroundProbe(account)
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	probeResult, err := s.accountModelImportService.ProbeAccountModels(ctx, cloned)
	if err != nil {
		slog.Warn(
			"openai_known_models_probe_failed",
			"account_id", account.ID,
			"duration_ms", time.Since(start).Milliseconds(),
			"error", err,
		)
		return
	}
	if probeResult == nil || len(probeResult.DetectedModels) == 0 {
		slog.Info(
			"openai_known_models_probe_empty",
			"account_id", account.ID,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return
	}

	updatedAt := time.Now().UTC()
	updates := MergeStringAnyMap(
		BuildOpenAIKnownModelsExtra(
			probeResult.DetectedModels,
			updatedAt,
			OpenAIKnownModelsSourceTestProbe,
		),
		BuildAccountModelAvailabilitySnapshotExtra(
			BuildAccountModelProjection(ctx, account, s.modelRegistryService),
			probeResult.DetectedModels,
			updatedAt,
			AccountModelProbeSnapshotSourceTestProbe,
			probeResult.ProbeSource,
		),
	)
	if err := s.accountRepo.UpdateExtra(ctx, account.ID, updates); err != nil {
		slog.Warn(
			"openai_known_models_snapshot_update_failed",
			"account_id", account.ID,
			"duration_ms", time.Since(start).Milliseconds(),
			"error", err,
		)
		return
	}
	mergeAccountExtra(account, updates)

	slog.Info(
		"openai_known_models_snapshot_updated",
		"account_id", account.ID,
		"model_count", len(probeResult.DetectedModels),
		"probe_source", probeResult.ProbeSource,
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

func precheckOpenAIAccountTestRuntimeQuota(account *Account, candidates ...string) error {
	status := openAIRuntimeQuotaStatusForCandidates(account, candidates...)
	if !status.Limited() {
		return nil
	}
	message := openAIAdminQuotaCooldownMessage(status)
	slog.Info(
		"openai_account_test_runtime_quota_blocked",
		"account_id", account.ID,
		"scope", status.Scope,
		"scope_remaining_seconds", int(status.ScopeRemaining.Seconds()),
		"account_reset_at", status.AccountResetAt,
		"candidates", candidates,
	)
	return infraerrors.BadRequest("TEST_OPENAI_RUNTIME_QUOTA_COOLDOWN", message)
}

// testClaudeAccountConnection tests an Anthropic Claude account's connection

func (s *AccountTestService) testOpenAIAccountConnection(c *gin.Context, account *Account, modelID string, prompt string, sourceProtocol string, simulatedClient string) error {
	ctx := c.Request.Context()
	requestFormat := ResolveOpenAITextRequestFormatForAccount(account, "")

	testModelID := resolveOpenAITestModelID(ctx, account, modelID, s.modelRegistryService)

	// For API Key accounts with model mapping, map the model
	if account.Type == "apikey" {
		mapping := account.GetModelMapping()
		if len(mapping) > 0 {
			if mappedModel, exists := mapping[testModelID]; exists {
				testModelID = mappedModel
			}
		}
	}

	if err := precheckOpenAIAccountTestRuntimeQuota(account, openAIRuntimeQuotaModelCandidates(account, testModelID, modelID)...); err != nil {
		return err
	}

	// Determine authentication method and API URL
	var authToken string
	var apiURL string
	var isOAuth bool
	var err error
	useChatGPTOAuth := isChatGPTOpenAIOAuthAccount(account)
	var chatgptAccountID string

	if account.IsOAuth() {
		isOAuth = true
		if s.openAITokenProvider != nil {
			authToken, err = s.openAITokenProvider.GetAccessToken(ctx, account)
			if err != nil {
				return s.sendErrorAndEnd(c, fmt.Sprintf("Failed to get access token: %s", err.Error()))
			}
		} else {
			authToken = account.GetOpenAIAccessToken()
		}
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No access token available")
		}

		apiURL, err = resolveOpenAITargetURLForRequestFormat(account, requestFormat, s.validateUpstreamBaseURL)
		if err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid base URL: %s", err.Error()))
		}
		if useChatGPTOAuth {
			chatgptAccountID = account.GetChatGPTAccountID()
		}
	} else if account.Type == "apikey" {
		// API Key - use Platform API
		authToken = account.GetOpenAIApiKey()
		if account.Platform == PlatformDeepSeek {
			authToken = strings.TrimSpace(account.GetCredential("api_key"))
		}
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No API key available")
		}

		baseURL := resolveOpenAICompatibleBaseURL(account)
		if baseURL == "" {
			baseURL = "https://api.openai.com"
		}
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid base URL: %s", err.Error()))
		}
		apiURL = buildOpenAIResponsesURLForPlatform(normalizedBaseURL, account.Platform)
		if requestFormat == GatewayOpenAIRequestFormatChatCompletions {
			apiURL = buildOpenAIChatCompletionsURLForPlatform(normalizedBaseURL, account.Platform)
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

	payload := createOpenAITestPayloadForRequestFormat(testModelID, prompt, requestFormat, useChatGPTOAuth)
	payloadBytes, _ := json.Marshal(payload)

	// Send test_start event
	s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	s.sendResolvedTestRuntimeMetaEvents(c)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}

	// Set common headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	if simulatedClient == GatewayClientProfileCodex {
		req.Header.Set("Originator", resolveOpenAIUpstreamOriginator(c, true))
		req.Header.Set("User-Agent", codexCLIUserAgent)
	}

	// Set OAuth-specific headers for ChatGPT internal API
	if useChatGPTOAuth {
		req.Host = "chatgpt.com"
		req.Header.Set("accept", "text/event-stream")
		if chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
		if s.openAIGatewayService != nil {
			s.openAIGatewayService.applyCodexOAuthUserAgentPolicy(ctx, req.Header, account)
		}
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

	probeCtx := WithOpenAICodexRequestModel(ctx, testModelID)
	var codexState *openAICodexRateLimitState
	if useChatGPTOAuth && s.accountRepo != nil {
		probeScope := openAICodexScopeNormal
		if resolvedScope, ok := resolveOpenAICodexQuotaScopeFromContext(probeCtx, account); ok && strings.TrimSpace(resolvedScope) != "" {
			probeScope = resolvedScope
		}
		probeCtx = withOpenAICodexResolvedQuotaScope(probeCtx, probeScope)
		if updates, err := extractOpenAICodexProbeUpdatesForScope(resp, probeScope); err == nil && len(updates) > 0 {
			codexState = syncOpenAICodexRateLimitState(probeCtx, s.accountRepo, account, updates, time.Now())
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if useChatGPTOAuth && s.accountRepo != nil {
			if resp.StatusCode == http.StatusTooManyRequests && (codexState == nil || (codexState.AccountResetAt == nil && codexState.ScopeResetAt == nil)) {
				if resetAt := (&RateLimitService{}).calculateOpenAI429ResetTime(resp.Header); resetAt != nil {
					_ = setAccountRateLimited(ctx, s.accountRepo, account.ID, *resetAt, AccountRateLimitReason429)
					account.RateLimitResetAt = resetAt
				}
			}
			if resp.StatusCode == http.StatusUnauthorized && isOpenAIPermanentUnauthorizedDetail(body) {
				errMsg := fmt.Sprintf("Authentication failed (401): %s", string(body))
				_ = s.accountRepo.SetError(ctx, account.ID, errMsg)
			}
		}
		return s.sendFailedTestResponse(c, ctx, account, resp.StatusCode, body, "API returned")
	}

	if isOAuth && s.accountModelImportService != nil && s.accountRepo != nil {
		accountForProbe := cloneAccountForBackgroundProbe(account)
		s.runBackgroundTask(func() {
			s.refreshOpenAIKnownModelsSnapshot(accountForProbe)
		})
	}

	// Process SSE stream
	if requestFormat == GatewayOpenAIRequestFormatChatCompletions {
		return s.processOpenAIChatCompletionsStream(c, resp.Body)
	}
	return s.processOpenAIStream(c, resp.Body)
}

func createOpenAITestPayloadForRequestFormat(modelID string, prompt string, requestFormat string, isOAuth bool) map[string]any {
	if NormalizeGatewayOpenAIRequestFormat(requestFormat) == GatewayOpenAIRequestFormatChatCompletions {
		return createOpenAIChatCompletionsTestPayload(modelID, prompt)
	}
	return createOpenAITestPayload(modelID, prompt, isOAuth)
}

// createOpenAITestPayload creates a test payload for OpenAI Responses API
func createOpenAITestPayload(modelID string, prompt string, isOAuth bool) map[string]any {
	textPrompt := strings.TrimSpace(prompt)
	if textPrompt == "" {
		textPrompt = accountDaily5HPrompt
	}
	payload := map[string]any{
		"model": modelID,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": textPrompt,
					},
				},
			},
		},
		"stream": true,
	}

	// OAuth accounts using ChatGPT internal API require store: false
	if isOAuth {
		payload["store"] = false
	}

	// All accounts require instructions for Responses API
	payload["instructions"] = openai.DefaultInstructions

	return payload
}

func createOpenAIChatCompletionsTestPayload(modelID string, prompt string) map[string]any {
	textPrompt := strings.TrimSpace(prompt)
	if textPrompt == "" {
		textPrompt = accountDaily5HPrompt
	}
	return map[string]any{
		"model": modelID,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": textPrompt,
			},
		},
		"stream": true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
	}
}

// processClaudeStream processes the SSE stream from Claude API
