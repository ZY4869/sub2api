package service

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
)

func (s *GeminiMessagesCompatService) handleNativeStreamingResponse(c *gin.Context, resp *http.Response, startTime time.Time, isOAuth bool) (*geminiNativeStreamResult, error) {
	if s.cfg != nil && s.cfg.Gateway.GeminiDebugResponseHeaders {
		logger.LegacyPrintf("service.gemini_messages_compat", "[GeminiAPI] ========== Streaming Response Headers ==========")
		for key, values := range resp.Header {
			if strings.HasPrefix(strings.ToLower(key), "x-ratelimit") {
				logger.LegacyPrintf("service.gemini_messages_compat", "[GeminiAPI] %s: %v", key, values)
			}
		}
		logger.LegacyPrintf("service.gemini_messages_compat", "[GeminiAPI] ====================================================")
	}
	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.Status(resp.StatusCode)
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/event-stream; charset=utf-8"
	}
	c.Header("Content-Type", contentType)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}
	reader := bufio.NewReader(resp.Body)
	usage := &ClaudeUsage{}
	var firstTokenMs *int
	responseID := ""
	var resolvedServiceTier *string
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			trimmed := strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(trimmed, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
				if payload == "" || payload == "[DONE]" {
					_, _ = io.WriteString(c.Writer, line)
					flusher.Flush()
				} else {
					var rawToWrite string
					rawToWrite = payload
					var rawBytes []byte
					if isOAuth {
						innerBytes, err := unwrapGeminiResponse([]byte(payload))
						if err == nil {
							rawToWrite = string(innerBytes)
							rawBytes = innerBytes
						}
					} else {
						rawBytes = []byte(payload)
					}
					var geminiResp map[string]any
					if json.Unmarshal(rawBytes, &geminiResp) == nil && geminiResp != nil {
						analysis := analyzeGeminiResponse(geminiResp, rawBytes)
						if analysis.Usage != nil {
							usage = analysis.Usage
						}
						if candidate := extractGeminiResolvedServiceTierFromResponse(rawBytes, nil); candidate != nil {
							resolvedServiceTier = candidate
						}
						if responseID == "" && strings.TrimSpace(analysis.ResponseID) != "" {
							responseID = analysis.ResponseID
						}
						if c.Writer.Header().Get("x-request-id") == "" && responseID != "" {
							c.Header("x-request-id", responseID)
						}
					} else if u := extractGeminiUsage(rawBytes); u != nil {
						usage = u
					}
					if firstTokenMs == nil {
						ms := int(time.Since(startTime).Milliseconds())
						firstTokenMs = &ms
					}
					if isOAuth {
						_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", rawToWrite)
					} else {
						_, _ = io.WriteString(c.Writer, line)
					}
					flusher.Flush()
				}
			} else {
				_, _ = io.WriteString(c.Writer, line)
				flusher.Flush()
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	if resolvedServiceTier == nil {
		resolvedServiceTier = extractGeminiResolvedServiceTierFromResponse(nil, resp.Header)
	}
	return &geminiNativeStreamResult{usage: usage, firstTokenMs: firstTokenMs, responseID: responseID, resolvedServiceTier: resolvedServiceTier}, nil
}
