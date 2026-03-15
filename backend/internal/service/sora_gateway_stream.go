package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

var soraImageSizeMap = map[string]string{
	"gpt-image":           "360",
	"gpt-image-landscape": "540",
	"gpt-image-portrait":  "540",
}

func (s *SoraGatewayService) shouldFailoverUpstreamError(statusCode int) bool {
	switch statusCode {
	case 401, 402, 403, 404, 429, 529:
		return true
	default:
		return statusCode >= 500
	}
}

func buildSoraNonStreamResponse(content, model string) map[string]any {
	return map[string]any{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{
			map[string]any{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
	}
}

func soraImageSizeFromModel(model string) string {
	modelLower := strings.ToLower(model)
	if size, ok := soraImageSizeMap[modelLower]; ok {
		return size
	}
	if strings.Contains(modelLower, "landscape") || strings.Contains(modelLower, "portrait") {
		return "540"
	}
	return "360"
}

func soraProErrorMessage(model, upstreamMsg string) string {
	modelLower := strings.ToLower(model)
	if strings.Contains(modelLower, "sora2pro-hd") {
		return "当前账号无法使用 Sora Pro-HD 模型，请更换模型或账号"
	}
	if strings.Contains(modelLower, "sora2pro") {
		return "当前账号无法使用 Sora Pro 模型，请更换模型或账号"
	}
	return ""
}

func firstMediaURL(urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

func (s *SoraGatewayService) buildSoraMediaURL(path string, rawQuery string) string {
	if path == "" {
		return path
	}
	prefix := "/sora/media"
	values := url.Values{}
	if rawQuery != "" {
		if parsed, err := url.ParseQuery(rawQuery); err == nil {
			values = parsed
		}
	}

	signKey := ""
	ttlSeconds := 0
	if s != nil && s.cfg != nil {
		signKey = strings.TrimSpace(s.cfg.Gateway.SoraMediaSigningKey)
		ttlSeconds = s.cfg.Gateway.SoraMediaSignedURLTTLSeconds
	}
	values.Del("sig")
	values.Del("expires")
	signingQuery := values.Encode()
	if signKey != "" && ttlSeconds > 0 {
		expires := time.Now().Add(time.Duration(ttlSeconds) * time.Second).Unix()
		signature := SignSoraMediaURL(path, signingQuery, expires, signKey)
		if signature != "" {
			values.Set("expires", strconv.FormatInt(expires, 10))
			values.Set("sig", signature)
			prefix = "/sora/media-signed"
		}
	}

	encoded := values.Encode()
	if encoded == "" {
		return prefix + path
	}
	return prefix + path + "?" + encoded
}

func (s *SoraGatewayService) prepareSoraStream(c *gin.Context, requestID string) {
	if c == nil {
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	if strings.TrimSpace(requestID) != "" {
		c.Header("x-request-id", requestID)
	}
}

func (s *SoraGatewayService) writeSoraStream(c *gin.Context, model, content string, startTime time.Time) (*int, error) {
	if c == nil {
		return nil, nil
	}
	writer := c.Writer
	flusher, _ := writer.(http.Flusher)

	chunk := map[string]any{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{
			map[string]any{
				"index": 0,
				"delta": map[string]any{
					"content": content,
				},
			},
		},
	}
	encoded, _ := jsonMarshalRaw(chunk)
	if _, err := fmt.Fprintf(writer, "data: %s\n\n", encoded); err != nil {
		return nil, err
	}
	if flusher != nil {
		flusher.Flush()
	}
	ms := int(time.Since(startTime).Milliseconds())
	finalChunk := map[string]any{
		"id":      chunk["id"],
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{
			map[string]any{
				"index":         0,
				"delta":         map[string]any{},
				"finish_reason": "stop",
			},
		},
	}
	finalEncoded, _ := jsonMarshalRaw(finalChunk)
	if _, err := fmt.Fprintf(writer, "data: %s\n\n", finalEncoded); err != nil {
		return &ms, err
	}
	if _, err := fmt.Fprint(writer, "data: [DONE]\n\n"); err != nil {
		return &ms, err
	}
	if flusher != nil {
		flusher.Flush()
	}
	return &ms, nil
}

func (s *SoraGatewayService) writeSoraError(c *gin.Context, status int, errType, message string, stream bool) {
	if c == nil {
		return
	}
	if stream {
		flusher, _ := c.Writer.(http.Flusher)
		errorData := map[string]any{
			"error": map[string]string{
				"type":    errType,
				"message": message,
			},
		}
		jsonBytes, err := json.Marshal(errorData)
		if err != nil {
			_ = c.Error(err)
			return
		}
		errorEvent := fmt.Sprintf("event: error\ndata: %s\n\n", string(jsonBytes))
		_, _ = fmt.Fprint(c.Writer, errorEvent)
		_, _ = fmt.Fprint(c.Writer, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		return
	}
	c.JSON(status, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}

func (s *SoraGatewayService) handleSoraRequestError(ctx context.Context, account *Account, err error, model string, c *gin.Context, stream bool) error {
	if err == nil {
		return nil
	}
	var upstreamErr *SoraUpstreamError
	if errors.As(err, &upstreamErr) {
		accountID := int64(0)
		if account != nil {
			accountID = account.ID
		}
		logger.LegacyPrintf(
			"service.sora",
			"[SoraRawError] account_id=%d model=%s status=%d request_id=%s cf_ray=%s message=%s raw_body=%s",
			accountID,
			model,
			upstreamErr.StatusCode,
			strings.TrimSpace(upstreamErr.Headers.Get("x-request-id")),
			strings.TrimSpace(upstreamErr.Headers.Get("cf-ray")),
			strings.TrimSpace(upstreamErr.Message),
			truncateForLog(upstreamErr.Body, 1024),
		)
		if s.rateLimitService != nil && account != nil {
			s.rateLimitService.HandleUpstreamError(ctx, account, upstreamErr.StatusCode, upstreamErr.Headers, upstreamErr.Body)
		}
		if s.shouldFailoverUpstreamError(upstreamErr.StatusCode) {
			var responseHeaders http.Header
			if upstreamErr.Headers != nil {
				responseHeaders = upstreamErr.Headers.Clone()
			}
			return &UpstreamFailoverError{
				StatusCode:      upstreamErr.StatusCode,
				ResponseBody:    upstreamErr.Body,
				ResponseHeaders: responseHeaders,
			}
		}
		msg := upstreamErr.Message
		if override := soraProErrorMessage(model, msg); override != "" {
			msg = override
		}
		s.writeSoraError(c, upstreamErr.StatusCode, "upstream_error", msg, stream)
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		s.writeSoraError(c, http.StatusGatewayTimeout, "timeout_error", "Sora generation timeout", stream)
		return err
	}
	s.writeSoraError(c, http.StatusBadGateway, "api_error", err.Error(), stream)
	return err
}

func (s *SoraGatewayService) maybeSendPing(c *gin.Context, lastPing *time.Time) {
	if c == nil {
		return
	}
	interval := 10 * time.Second
	if s != nil && s.cfg != nil && s.cfg.Concurrency.PingInterval > 0 {
		interval = time.Duration(s.cfg.Concurrency.PingInterval) * time.Second
	}
	if time.Since(*lastPing) < interval {
		return
	}
	if _, err := fmt.Fprint(c.Writer, ":\n\n"); err == nil {
		if flusher, ok := c.Writer.(http.Flusher); ok {
			flusher.Flush()
		}
		*lastPing = time.Now()
	}
}

func (s *SoraGatewayService) normalizeSoraMediaURLs(urls []string) []string {
	if len(urls) == 0 {
		return urls
	}
	output := make([]string, 0, len(urls))
	for _, raw := range urls {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
			output = append(output, raw)
			continue
		}
		pathVal := raw
		if !strings.HasPrefix(pathVal, "/") {
			pathVal = "/" + pathVal
		}
		output = append(output, s.buildSoraMediaURL(pathVal, ""))
	}
	return output
}

// jsonMarshalRaw 序列化 JSON，不转义 &、<、> 等 HTML 字符，
// 避免 URL 中的 & 被转义为 \u0026 导致客户端无法直接使用。
func jsonMarshalRaw(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// Encode 会追加换行符，去掉它
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}

func buildSoraContent(mediaType string, urls []string) string {
	switch mediaType {
	case "image":
		parts := make([]string, 0, len(urls))
		for _, u := range urls {
			parts = append(parts, fmt.Sprintf("![image](%s)", u))
		}
		return strings.Join(parts, "\n")
	case "video":
		if len(urls) == 0 {
			return ""
		}
		return fmt.Sprintf("```html\n<video src='%s' controls></video>\n```", urls[0])
	default:
		return ""
	}
}
