package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const openAIResponsesInputTokensPath = "/input_tokens"

type AnthropicCountTokensBridgeResult struct {
	InputTokens int `json:"input_tokens"`
}

func (s *OpenAIGatewayService) ForwardAnthropicCountTokensCompat(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	defaultMappedModel string,
) (*AnthropicCountTokensBridgeResult, error) {
	countBody, err := buildResponsesInputTokensBody(account, body, defaultMappedModel)
	if err != nil {
		writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body", "")
		return nil, err
	}
	countBody = sanitizeOpenAIResponsesInputTokensBody(countBody)
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		writeAnthropicError(c, http.StatusBadGateway, "upstream_error", "Failed to get access token", "")
		return nil, err
	}
	result, err := s.forwardResponsesInputTokens(ctx, c, account, countBody, token)
	if err != nil {
		return nil, err
	}
	c.JSON(http.StatusOK, result)
	return result, nil
}

func (s *OpenAIGatewayService) forwardResponsesInputTokens(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	token string,
) (*AnthropicCountTokensBridgeResult, error) {
	targetURL, err := resolveOpenAIResponsesTargetURL(account, s.validateUpstreamBaseURL)
	if err != nil {
		writeAnthropicError(c, http.StatusBadGateway, "upstream_error", "Failed to build request", "")
		return nil, err
	}
	targetURL = strings.TrimRight(targetURL, "/") + openAIResponsesInputTokensPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		writeAnthropicError(c, http.StatusInternalServerError, "api_error", "Failed to build request", "")
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			lowerKey := strings.ToLower(strings.TrimSpace(key))
			if !openaiAllowedHeaders[lowerKey] || lowerKey == "content-type" {
				continue
			}
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}
	if account != nil {
		if customUA := account.GetOpenAIUserAgent(); customUA != "" {
			req.Header.Set("User-Agent", customUA)
		}
	}
	proxyURL := ""
	if account != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	accountID, accountConcurrency := accountRuntimeRequestMeta(account)
	resp, err := s.httpUpstream.Do(MarkOpenAIHTTPUpstreamRequest(req), proxyURL, accountID, accountConcurrency)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		writeAnthropicError(c, http.StatusBadGateway, "upstream_error", "Request failed", "")
		return nil, fmt.Errorf("openai input_tokens request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()
	return readResponsesInputTokensResult(resp, c, resolveUpstreamResponseReadLimit(s.cfg), func(status int, errType string, message string) {
		writeAnthropicError(c, status, errType, message, "")
	})
}

func buildResponsesInputTokensBody(account *Account, anthropicBody []byte, defaultMappedModel string) ([]byte, error) {
	var anthropicReq apicompat.AnthropicRequest
	if err := json.Unmarshal(anthropicBody, &anthropicReq); err != nil {
		return nil, fmt.Errorf("parse anthropic count_tokens request: %w", err)
	}
	originalModel := anthropicReq.Model
	applyOpenAICompatModelNormalization(&anthropicReq)
	responsesReq, _, err := ConvertAnthropicMessagesToResponsesRuntime(&anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("convert anthropic count_tokens request: %w", err)
	}
	mappedModel := normalizeOpenAIModelForUpstream(account, resolveOpenAIForwardModel(account, anthropicReq.Model, defaultMappedModel))
	if strings.TrimSpace(mappedModel) == "" {
		mappedModel = strings.TrimSpace(firstNonEmptyString(anthropicReq.Model, originalModel))
	}
	responsesReq.Model = mappedModel
	responsesReq.Stream = false
	responsesReq.MaxOutputTokens = nil
	responsesReq.Temperature = nil
	responsesReq.TopP = nil
	responsesReq.Store = nil
	return json.Marshal(responsesReq)
}

func sanitizeOpenAIResponsesInputTokensBody(body []byte) []byte {
	if len(body) == 0 {
		return body
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	for _, key := range []string{
		"stream",
		"store",
		"max_output_tokens",
		"temperature",
		"top_p",
		"tool_choice",
		"parallel_tool_calls",
		"prompt_cache_key",
		"previous_response_id",
		"service_tier",
		"metadata",
	} {
		delete(payload, key)
	}
	next, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return next
}

func readResponsesInputTokensResult(
	resp *http.Response,
	c *gin.Context,
	maxReadBytes int64,
	writeError func(status int, errType string, message string),
) (*AnthropicCountTokensBridgeResult, error) {
	if resp == nil {
		writeError(http.StatusBadGateway, "upstream_error", "Upstream request failed")
		return nil, fmt.Errorf("upstream response is nil")
	}
	body, err := readUpstreamResponseBodyLimited(resp.Body, maxReadBytes)
	if err != nil {
		writeError(http.StatusBadGateway, "upstream_error", "Failed to read response")
		return nil, err
	}
	if resp.StatusCode >= 400 {
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(body)))
		setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, "")
		message := "Upstream request failed"
		if resp.StatusCode == http.StatusTooManyRequests {
			message = "Rate limit exceeded"
		}
		writeError(resp.StatusCode, "upstream_error", message)
		if upstreamMsg == "" {
			return nil, fmt.Errorf("upstream input_tokens error: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("upstream input_tokens error: %d message=%s", resp.StatusCode, upstreamMsg)
	}
	inputTokens := int(firstPositiveInt64(
		gjson.GetBytes(body, "input_tokens").Int(),
		gjson.GetBytes(body, "usage.input_tokens").Int(),
		gjson.GetBytes(body, "total_tokens").Int(),
	))
	return &AnthropicCountTokensBridgeResult{InputTokens: inputTokens}, nil
}

func accountRuntimeRequestMeta(account *Account) (int64, int) {
	if account == nil {
		return 0, 0
	}
	return account.ID, account.Concurrency
}
