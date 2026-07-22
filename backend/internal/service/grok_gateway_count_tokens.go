package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func (s *GrokGatewayService) readGrokResponsesInputTokensResult(
	resp *http.Response,
	c *gin.Context,
	account *Account,
	anthropicBody []byte,
	maxReadBytes int64,
) (*AnthropicCountTokensBridgeResult, error) {
	if resp == nil {
		writeAnthropicError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", "")
		return nil, fmt.Errorf("upstream response is nil")
	}
	body, err := readUpstreamResponseBodyLimited(resp.Body, maxReadBytes)
	if err != nil {
		writeAnthropicError(c, http.StatusBadGateway, "upstream_error", "Failed to read response", "")
		return nil, err
	}
	if resp.StatusCode < 400 {
		inputTokens := int(firstPositiveInt64(
			gjson.GetBytes(body, "input_tokens").Int(),
			gjson.GetBytes(body, "usage.input_tokens").Int(),
			gjson.GetBytes(body, "total_tokens").Int(),
		))
		return &AnthropicCountTokensBridgeResult{InputTokens: inputTokens}, nil
	}

	upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(body)))
	setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, "")
	if isGrokResponsesInputTokensUnsupported(resp.StatusCode, body) {
		return s.finishGrokEstimatedCountTokensResponse(c, account, anthropicBody, resp.StatusCode, upstreamMsg), nil
	}

	message := "Upstream request failed"
	if resp.StatusCode == http.StatusTooManyRequests {
		message = "Rate limit exceeded"
	}
	writeAnthropicError(c, resp.StatusCode, "upstream_error", message, "")
	if upstreamMsg == "" {
		return nil, fmt.Errorf("upstream input_tokens error: %d", resp.StatusCode)
	}
	return nil, fmt.Errorf("upstream input_tokens error: %d message=%s", resp.StatusCode, upstreamMsg)
}

func isGrokResponsesInputTokensUnsupported(statusCode int, body []byte) bool {
	if statusCode != http.StatusNotFound && statusCode != http.StatusBadRequest && statusCode != http.StatusNotImplemented {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(extractUpstreamErrorMessage(body)))
	if msg == "" {
		return statusCode == http.StatusNotFound
	}
	if strings.Contains(msg, "input_tokens") || strings.Contains(msg, "count_tokens") {
		return strings.Contains(msg, "not found") ||
			strings.Contains(msg, "unsupported") ||
			strings.Contains(msg, "not supported") ||
			strings.Contains(msg, "unknown endpoint") ||
			strings.Contains(msg, "does not exist")
	}
	return statusCode == http.StatusNotFound && strings.Contains(msg, "not found")
}

func (s *GrokGatewayService) finishGrokEstimatedCountTokensResponse(
	c *gin.Context,
	account *Account,
	anthropicBody []byte,
	upstreamStatusCode int,
	message string,
) *AnthropicCountTokensBridgeResult {
	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" {
		message = "Grok input_tokens upstream unavailable; estimated fallback used"
	}
	if c != nil {
		c.Header(geminiCountTokensSourceHeader, string(geminiCountTokensSourceEstimated))
	}
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           PlatformGrok,
		AccountID:          accountIDForOps(account),
		AccountName:        accountNameForOps(account),
		UpstreamStatusCode: upstreamStatusCode,
		Kind:               "count_tokens_estimated",
		Message:            message,
	})
	result := &AnthropicCountTokensBridgeResult{
		InputTokens: estimateGrokAnthropicCountTokens(anthropicBody),
		Estimated:   true,
		Source:      string(geminiCountTokensSourceEstimated),
	}
	return result
}

func estimateGrokAnthropicCountTokens(body []byte) int {
	var req apicompat.AnthropicRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return estimateTokensForText(string(body))
	}
	total := 0
	total += estimateGrokRawTextTokens(req.System)
	for _, message := range req.Messages {
		total += estimateTokensForText(message.Role)
		total += estimateGrokRawTextTokens(message.Content)
	}
	for _, tool := range req.Tools {
		total += estimateTokensForText(tool.Type)
		total += estimateTokensForText(tool.Name)
		total += estimateTokensForText(tool.Description)
		total += estimateGrokRawTextTokens(tool.InputSchema)
	}
	if total < 0 {
		return 0
	}
	return total
}

func estimateGrokRawTextTokens(raw []byte) int {
	if len(raw) == 0 {
		return 0
	}
	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return estimateTokensForText(plain)
	}
	var blocks []apicompat.AnthropicContentBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		total := 0
		for _, block := range blocks {
			total += estimateTokensForText(block.Type)
			total += estimateTokensForText(block.Text)
			total += estimateTokensForText(block.Thinking)
			total += estimateTokensForText(block.Name)
			total += estimateGrokRawTextTokens(block.Input)
			total += estimateGrokRawTextTokens(block.Content)
			if block.Source != nil {
				total += estimateTokensForText(block.Source.MediaType)
			}
		}
		return total
	}
	return estimateTokensForText(string(raw))
}

func accountIDForOps(account *Account) int64 {
	if account == nil {
		return 0
	}
	return account.ID
}

func accountNameForOps(account *Account) string {
	if account == nil {
		return ""
	}
	return account.Name
}
