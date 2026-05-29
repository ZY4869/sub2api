package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
)

type UpstreamHTTPResult struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

func (s *GeminiMessagesCompatService) handleNativeNonStreamingResponse(c *gin.Context, resp *http.Response, isOAuth bool) (*ClaudeUsage, *string, error) {
	if s.cfg != nil && s.cfg.Gateway.GeminiDebugResponseHeaders {
		logger.LegacyPrintf("service.gemini_messages_compat", "[GeminiAPI] ========== Response Headers ==========")
		for key, values := range resp.Header {
			if strings.HasPrefix(strings.ToLower(key), "x-ratelimit") {
				logger.LegacyPrintf("service.gemini_messages_compat", "[GeminiAPI] %s: %v", key, values)
			}
		}
		logger.LegacyPrintf("service.gemini_messages_compat", "[GeminiAPI] ========================================")
	}
	maxBytes := resolveUpstreamResponseReadLimit(s.cfg)
	respBody, err := readUpstreamResponseBodyLimited(resp.Body, maxBytes)
	if err != nil {
		if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
			setOpsUpstreamError(c, http.StatusBadGateway, "upstream response too large", "")
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "upstream_error", "message": "Upstream response too large"}})
		}
		return nil, nil, err
	}
	if isOAuth {
		unwrappedBody, uwErr := unwrapGeminiResponse(respBody)
		if uwErr == nil {
			respBody = unwrappedBody
		}
	}
	resolvedServiceTier := extractGeminiResolvedServiceTierFromResponse(respBody, resp.Header)
	var geminiResp map[string]any
	_ = json.Unmarshal(respBody, &geminiResp)
	analysis := analyzeGeminiResponse(geminiResp, respBody)
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	if c.Writer.Header().Get("x-request-id") == "" && strings.TrimSpace(analysis.ResponseID) != "" {
		c.Header("x-request-id", analysis.ResponseID)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, respBody)
	if analysis.Usage != nil {
		return analysis.Usage, resolvedServiceTier, nil
	}
	return &ClaudeUsage{}, resolvedServiceTier, nil
}

func getGeminiUpstreamRequestID(header http.Header, primaryKey string) string {
	return firstNonEmptyString(
		getGeminiHeaderValue(header, primaryKey),
		getGeminiHeaderValue(header, "x-goog-request-id"),
	)
}

func getGeminiHeaderValue(header http.Header, key string) string {
	key = strings.TrimSpace(key)
	if header == nil || key == "" {
		return ""
	}
	if value := strings.TrimSpace(header.Get(key)); value != "" {
		return value
	}
	for headerKey, values := range header {
		if !strings.EqualFold(headerKey, key) {
			continue
		}
		for _, value := range values {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}
