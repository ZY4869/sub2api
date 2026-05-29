package service

import (
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func estimateGeminiCountTokens(reqBody []byte) int {
	total := 0
	gjson.GetBytes(reqBody, "systemInstruction.parts").ForEach(func(_, part gjson.Result) bool {
		if t := strings.TrimSpace(part.Get("text").String()); t != "" {
			total += estimateTokensForText(t)
		}
		return true
	})
	gjson.GetBytes(reqBody, "contents").ForEach(func(_, content gjson.Result) bool {
		content.Get("parts").ForEach(func(_, part gjson.Result) bool {
			if t := strings.TrimSpace(part.Get("text").String()); t != "" {
				total += estimateTokensForText(t)
			}
			return true
		})
		return true
	})
	if total < 0 {
		return 0
	}
	return total
}

func estimateTokensForText(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	runes := []rune(s)
	if len(runes) == 0 {
		return 0
	}
	ascii := 0
	for _, r := range runes {
		if r <= 0x7f {
			ascii++
		}
	}
	asciiRatio := float64(ascii) / float64(len(runes))
	if asciiRatio >= 0.8 {
		return (len(runes) + 3) / 4
	}
	return len(runes)
}

func setGeminiCountTokensSourceHeader(c *gin.Context, source geminiCountTokensSource) {
	if c == nil {
		return
	}
	if value := strings.TrimSpace(string(source)); value != "" {
		c.Header(geminiCountTokensSourceHeader, value)
	}
}

func (s *GeminiMessagesCompatService) finishGeminiEstimatedCountTokensResponse(
	c *gin.Context,
	account *Account,
	originalModel string,
	mappedModel string,
	simulatedClient string,
	requestID string,
	body []byte,
	upstreamStatusCode int,
	message string,
	detail string,
	startTime time.Time,
) (*ForwardResult, error) {
	requestID = strings.TrimSpace(requestID)
	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" {
		message = "countTokens upstream unavailable; estimated fallback used"
	}
	setGeminiCountTokensSourceHeader(c, geminiCountTokensSourceEstimated)
	if c != nil && requestID != "" && c.Writer != nil && c.Writer.Header().Get("x-request-id") == "" {
		c.Header("x-request-id", requestID)
	}
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           RoutingPlatformForAccount(account),
		AccountID:          account.ID,
		AccountName:        account.Name,
		UpstreamStatusCode: upstreamStatusCode,
		UpstreamRequestID:  requestID,
		Kind:               "count_tokens_estimated",
		Message:            message,
		Detail:             strings.TrimSpace(detail),
	})
	logger.LegacyPrintf(
		"service.gemini_messages_compat",
		"Gemini account %d: countTokens fallback source=estimated status=%d request_id=%s model=%s upstream_model=%s reason=%s",
		account.ID,
		upstreamStatusCode,
		requestID,
		originalModel,
		mappedModel,
		message,
	)
	if c != nil {
		c.JSON(http.StatusOK, map[string]any{"totalTokens": estimateGeminiCountTokens(body)})
	}
	result := &ForwardResult{
		RequestID:            requestID,
		Usage:                ClaudeUsage{},
		Model:                originalModel,
		UpstreamModel:        mappedModel,
		RequestedServiceTier: extractGeminiRequestedServiceTierFromBody(body),
		ServiceTier:          extractGeminiRequestedServiceTierFromBody(body),
		SimulatedClient:      simulatedClient,
		Stream:               false,
		Duration:             time.Since(startTime),
		FirstTokenMs:         nil,
	}
	applyClaudeCapabilityToForwardResult(result, ParseClaudeRequestCapability(originalModel, strings.TrimSpace(gjson.GetBytes(body, "effortLevel").String())))
	return result, nil
}
