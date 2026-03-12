package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type antigravityStreamResult struct {
	usage            *ClaudeUsage
	firstTokenMs     *int
	clientDisconnect bool
}
type antigravityClientWriter struct {
	w            gin.ResponseWriter
	flusher      http.Flusher
	disconnected bool
	prefix       string
}

func newAntigravityClientWriter(w gin.ResponseWriter, flusher http.Flusher, prefix string) *antigravityClientWriter {
	return &antigravityClientWriter{w: w, flusher: flusher, prefix: prefix}
}
func (cw *antigravityClientWriter) Write(p []byte) bool {
	if cw.disconnected {
		return false
	}
	if _, err := cw.w.Write(p); err != nil {
		cw.markDisconnected()
		return false
	}
	cw.flusher.Flush()
	return true
}
func (cw *antigravityClientWriter) Fprintf(format string, args ...any) bool {
	if cw.disconnected {
		return false
	}
	if _, err := fmt.Fprintf(cw.w, format, args...); err != nil {
		cw.markDisconnected()
		return false
	}
	cw.flusher.Flush()
	return true
}
func (cw *antigravityClientWriter) Disconnected() bool {
	return cw.disconnected
}
func (cw *antigravityClientWriter) markDisconnected() {
	cw.disconnected = true
	logger.LegacyPrintf("service.antigravity_gateway", "Client disconnected during streaming (%s), continuing to drain upstream for billing", cw.prefix)
}
func handleStreamReadError(err error, clientDisconnected bool, prefix string) (disconnect bool, handled bool) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		logger.LegacyPrintf("service.antigravity_gateway", "Context canceled during streaming (%s), returning collected usage", prefix)
		return true, true
	}
	if clientDisconnected {
		logger.LegacyPrintf("service.antigravity_gateway", "Upstream read error after client disconnect (%s): %v, returning collected usage", prefix, err)
		return true, true
	}
	return false, false
}
func (s *AntigravityGatewayService) handleGeminiStreamingResponse(c *gin.Context, resp *http.Response, startTime time.Time) (*antigravityStreamResult, error) {
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
	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.settingService.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	scanner.Buffer(scanBuf[:0], maxLineSize)
	usage := &ClaudeUsage{}
	var firstTokenMs *int
	type scanEvent struct {
		line string
		err  error
	}
	events := make(chan scanEvent, 16)
	done := make(chan struct{})
	sendEvent := func(ev scanEvent) bool {
		select {
		case events <- ev:
			return true
		case <-done:
			return false
		}
	}
	var lastReadAt int64
	atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
	go func(scanBuf *sseScannerBuf64K) {
		defer putSSEScannerBuf64K(scanBuf)
		defer close(events)
		for scanner.Scan() {
			atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
			if !sendEvent(scanEvent{line: scanner.Text()}) {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			_ = sendEvent(scanEvent{err: err})
		}
	}(scanBuf)
	defer close(done)
	streamInterval := time.Duration(0)
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		streamInterval = time.Duration(s.settingService.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
	}
	var intervalTicker *time.Ticker
	if streamInterval > 0 {
		intervalTicker = time.NewTicker(streamInterval)
		defer intervalTicker.Stop()
	}
	var intervalCh <-chan time.Time
	if intervalTicker != nil {
		intervalCh = intervalTicker.C
	}
	cw := newAntigravityClientWriter(c.Writer, flusher, "antigravity gemini")
	errorEventSent := false
	sendErrorEvent := func(reason string) {
		if errorEventSent || cw.Disconnected() {
			return
		}
		errorEventSent = true
		_, _ = fmt.Fprintf(c.Writer, "event: error\ndata: {\"error\":\"%s\"}\n\n", reason)
		flusher.Flush()
	}
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: cw.Disconnected()}, nil
			}
			if ev.err != nil {
				if disconnect, handled := handleStreamReadError(ev.err, cw.Disconnected(), "antigravity gemini"); handled {
					return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: disconnect}, nil
				}
				if errors.Is(ev.err, bufio.ErrTooLong) {
					logger.LegacyPrintf("service.antigravity_gateway", "SSE line too long (antigravity): max_size=%d error=%v", maxLineSize, ev.err)
					sendErrorEvent("response_too_large")
					return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs}, ev.err
				}
				sendErrorEvent("stream_read_error")
				return nil, ev.err
			}
			line := ev.line
			trimmed := strings.TrimRight(line, "\r\n")
			if strings.HasPrefix(trimmed, "data:") {
				payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
				if payload == "" || payload == "[DONE]" {
					cw.Fprintf("%s\n", line)
					continue
				}
				inner, parseErr := s.unwrapV1InternalResponse([]byte(payload))
				if parseErr == nil && inner != nil {
					payload = string(inner)
				}
				if u := extractGeminiUsage(inner); u != nil {
					usage = u
				}
				var parsed map[string]any
				if json.Unmarshal(inner, &parsed) == nil {
					if candidates, ok := parsed["candidates"].([]any); ok && len(candidates) > 0 {
						if cand, ok := candidates[0].(map[string]any); ok {
							if fr, ok := cand["finishReason"].(string); ok && fr == "MALFORMED_FUNCTION_CALL" {
								logger.LegacyPrintf("service.antigravity_gateway", "[Antigravity] MALFORMED_FUNCTION_CALL detected in forward stream")
								if content, ok := cand["content"]; ok {
									if b, err := json.Marshal(content); err == nil {
										logger.LegacyPrintf("service.antigravity_gateway", "[Antigravity] Malformed content: %s", string(b))
									}
								}
							}
						}
					}
				}
				if firstTokenMs == nil {
					ms := int(time.Since(startTime).Milliseconds())
					firstTokenMs = &ms
				}
				cw.Fprintf("data: %s\n\n", payload)
				continue
			}
			cw.Fprintf("%s\n", line)
		case <-intervalCh:
			lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
			if time.Since(lastRead) < streamInterval {
				continue
			}
			if cw.Disconnected() {
				logger.LegacyPrintf("service.antigravity_gateway", "Upstream timeout after client disconnect (antigravity gemini), returning collected usage")
				return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}, nil
			}
			logger.LegacyPrintf("service.antigravity_gateway", "Stream data interval timeout (antigravity)")
			sendErrorEvent("stream_timeout")
			return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs}, fmt.Errorf("stream data interval timeout")
		}
	}
}
func (s *AntigravityGatewayService) handleGeminiStreamToNonStreaming(c *gin.Context, resp *http.Response, startTime time.Time) (*antigravityStreamResult, error) {
	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.settingService.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	scanner.Buffer(scanBuf[:0], maxLineSize)
	usage := &ClaudeUsage{}
	var firstTokenMs *int
	var last map[string]any
	var lastWithParts map[string]any
	var collectedImageParts []map[string]any
	var collectedTextParts []string
	type scanEvent struct {
		line string
		err  error
	}
	events := make(chan scanEvent, 16)
	done := make(chan struct{})
	sendEvent := func(ev scanEvent) bool {
		select {
		case events <- ev:
			return true
		case <-done:
			return false
		}
	}
	var lastReadAt int64
	atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
	go func(scanBuf *sseScannerBuf64K) {
		defer putSSEScannerBuf64K(scanBuf)
		defer close(events)
		for scanner.Scan() {
			atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
			if !sendEvent(scanEvent{line: scanner.Text()}) {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			_ = sendEvent(scanEvent{err: err})
		}
	}(scanBuf)
	defer close(done)
	streamInterval := time.Duration(0)
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		streamInterval = time.Duration(s.settingService.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
	}
	var intervalTicker *time.Ticker
	if streamInterval > 0 {
		intervalTicker = time.NewTicker(streamInterval)
		defer intervalTicker.Stop()
	}
	var intervalCh <-chan time.Time
	if intervalTicker != nil {
		intervalCh = intervalTicker.C
	}
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				goto returnResponse
			}
			if ev.err != nil {
				if errors.Is(ev.err, bufio.ErrTooLong) {
					logger.LegacyPrintf("service.antigravity_gateway", "SSE line too long (antigravity non-stream): max_size=%d error=%v", maxLineSize, ev.err)
				}
				return nil, ev.err
			}
			line := ev.line
			trimmed := strings.TrimRight(line, "\r\n")
			if !strings.HasPrefix(trimmed, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
			if payload == "" || payload == "[DONE]" {
				continue
			}
			inner, parseErr := s.unwrapV1InternalResponse([]byte(payload))
			if parseErr != nil {
				continue
			}
			var parsed map[string]any
			if err := json.Unmarshal(inner, &parsed); err != nil {
				continue
			}
			if firstTokenMs == nil {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}
			last = parsed
			if u := extractGeminiUsage(inner); u != nil {
				usage = u
			}
			if candidates, ok := parsed["candidates"].([]any); ok && len(candidates) > 0 {
				if cand, ok := candidates[0].(map[string]any); ok {
					if fr, ok := cand["finishReason"].(string); ok && fr == "MALFORMED_FUNCTION_CALL" {
						logger.LegacyPrintf("service.antigravity_gateway", "[Antigravity] MALFORMED_FUNCTION_CALL detected in forward non-stream collect")
						if content, ok := cand["content"]; ok {
							if b, err := json.Marshal(content); err == nil {
								logger.LegacyPrintf("service.antigravity_gateway", "[Antigravity] Malformed content: %s", string(b))
							}
						}
					}
				}
			}
			if parts := extractGeminiParts(parsed); len(parts) > 0 {
				lastWithParts = parsed
				for _, part := range parts {
					if inlineData, ok := part["inlineData"].(map[string]any); ok {
						collectedImageParts = append(collectedImageParts, part)
						_ = inlineData
					}
					if text, ok := part["text"].(string); ok && text != "" {
						collectedTextParts = append(collectedTextParts, text)
					}
				}
			}
		case <-intervalCh:
			lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
			if time.Since(lastRead) < streamInterval {
				continue
			}
			logger.LegacyPrintf("service.antigravity_gateway", "Stream data interval timeout (antigravity non-stream)")
			return nil, fmt.Errorf("stream data interval timeout")
		}
	}
returnResponse:
	finalResponse := pickGeminiCollectResult(last, lastWithParts)
	if last == nil && lastWithParts == nil {
		logger.LegacyPrintf("service.antigravity_gateway", "[antigravity-Forward] warning: empty stream response (gemini non-stream), triggering failover")
		return nil, &UpstreamFailoverError{StatusCode: http.StatusBadGateway, ResponseBody: []byte(`{"error":"empty stream response from upstream"}`), RetryableOnSameAccount: true}
	}
	if len(collectedImageParts) > 0 {
		finalResponse = mergeImagePartsToResponse(finalResponse, collectedImageParts)
	}
	if len(collectedTextParts) > 0 {
		finalResponse = mergeTextPartsToResponse(finalResponse, collectedTextParts)
	}
	respBody, err := json.Marshal(finalResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}
	c.Data(http.StatusOK, "application/json", respBody)
	return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs}, nil
}
func getOrCreateGeminiParts(response map[string]any) (result map[string]any, existingParts []any, setParts func([]any)) {
	result = make(map[string]any)
	for k, v := range response {
		result[k] = v
	}
	candidates, ok := result["candidates"].([]any)
	if !ok || len(candidates) == 0 {
		candidates = []any{map[string]any{}}
	}
	candidate, ok := candidates[0].(map[string]any)
	if !ok {
		candidate = make(map[string]any)
		candidates[0] = candidate
	}
	content, ok := candidate["content"].(map[string]any)
	if !ok {
		content = map[string]any{"role": "model"}
		candidate["content"] = content
	}
	existingParts, ok = content["parts"].([]any)
	if !ok {
		existingParts = []any{}
	}
	setParts = func(newParts []any) {
		content["parts"] = newParts
		result["candidates"] = candidates
	}
	return result, existingParts, setParts
}
func mergeCollectedPartsToResponse(response map[string]any, collectedParts []map[string]any) map[string]any {
	if len(collectedParts) == 0 {
		return response
	}
	result, _, setParts := getOrCreateGeminiParts(response)
	var mergedParts []any
	var textBuffer strings.Builder
	flushTextBuffer := func() {
		if textBuffer.Len() > 0 {
			mergedParts = append(mergedParts, map[string]any{"text": textBuffer.String()})
			textBuffer.Reset()
		}
	}
	for _, part := range collectedParts {
		if text, ok := part["text"].(string); ok {
			if thought, _ := part["thought"].(bool); thought {
				flushTextBuffer()
				mergedParts = append(mergedParts, part)
			} else {
				_, _ = textBuffer.WriteString(text)
			}
		} else {
			flushTextBuffer()
			mergedParts = append(mergedParts, part)
		}
	}
	flushTextBuffer()
	setParts(mergedParts)
	return result
}
func mergeImagePartsToResponse(response map[string]any, imageParts []map[string]any) map[string]any {
	if len(imageParts) == 0 {
		return response
	}
	result, existingParts, setParts := getOrCreateGeminiParts(response)
	for _, p := range existingParts {
		if pm, ok := p.(map[string]any); ok {
			if _, hasInline := pm["inlineData"]; hasInline {
				return result
			}
		}
	}
	for _, imgPart := range imageParts {
		existingParts = append(existingParts, imgPart)
	}
	setParts(existingParts)
	return result
}
func mergeTextPartsToResponse(response map[string]any, textParts []string) map[string]any {
	if len(textParts) == 0 {
		return response
	}
	mergedText := strings.Join(textParts, "")
	result, existingParts, setParts := getOrCreateGeminiParts(response)
	newParts := make([]any, 0, len(existingParts)+1)
	textUpdated := false
	for _, p := range existingParts {
		pm, ok := p.(map[string]any)
		if !ok {
			newParts = append(newParts, p)
			continue
		}
		if _, hasText := pm["text"]; hasText && !textUpdated {
			newPart := make(map[string]any)
			for k, v := range pm {
				newPart[k] = v
			}
			newPart["text"] = mergedText
			newParts = append(newParts, newPart)
			textUpdated = true
		} else {
			newParts = append(newParts, pm)
		}
	}
	if !textUpdated {
		newParts = append([]any{map[string]any{"text": mergedText}}, newParts...)
	}
	setParts(newParts)
	return result
}
func (s *AntigravityGatewayService) writeClaudeError(c *gin.Context, status int, errType, message string) error {
	c.JSON(status, gin.H{"type": "error", "error": gin.H{"type": errType, "message": message}})
	return fmt.Errorf("%s", message)
}
func (s *AntigravityGatewayService) WriteMappedClaudeError(c *gin.Context, account *Account, upstreamStatus int, upstreamRequestID string, body []byte) error {
	return s.writeMappedClaudeError(c, account, upstreamStatus, upstreamRequestID, body)
}
func (s *AntigravityGatewayService) writeMappedClaudeError(c *gin.Context, account *Account, upstreamStatus int, upstreamRequestID string, body []byte) error {
	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(body))
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
	logBody, maxBytes := s.getLogConfig()
	upstreamDetail := s.getUpstreamErrorDetail(body)
	setOpsUpstreamError(c, upstreamStatus, upstreamMsg, upstreamDetail)
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: upstreamStatus, UpstreamRequestID: upstreamRequestID, Kind: "http_error", Message: upstreamMsg, Detail: upstreamDetail})
	if logBody {
		logger.LegacyPrintf("service.antigravity_gateway", "[antigravity-Forward] upstream_error status=%d body=%s", upstreamStatus, truncateForLog(body, maxBytes))
	}
	if ptStatus, ptErrType, ptErrMsg, matched := applyErrorPassthroughRule(c, account.Platform, upstreamStatus, body, 0, "", ""); matched {
		c.JSON(ptStatus, gin.H{"type": "error", "error": gin.H{"type": ptErrType, "message": ptErrMsg}})
		if upstreamMsg == "" {
			return fmt.Errorf("upstream error: %d", upstreamStatus)
		}
		return fmt.Errorf("upstream error: %d message=%s", upstreamStatus, upstreamMsg)
	}
	var statusCode int
	var errType, errMsg string
	switch upstreamStatus {
	case 400:
		statusCode = http.StatusBadRequest
		errType = "invalid_request_error"
		errMsg = getPassthroughOrDefault(upstreamMsg, "Invalid request")
	case 401:
		statusCode = http.StatusBadGateway
		errType = "authentication_error"
		errMsg = "Upstream authentication failed"
	case 403:
		statusCode = http.StatusBadGateway
		errType = "permission_error"
		errMsg = "Upstream access forbidden"
	case 429:
		statusCode = http.StatusTooManyRequests
		errType = "rate_limit_error"
		errMsg = "Upstream rate limit exceeded"
	case 529:
		statusCode = http.StatusServiceUnavailable
		errType = "overloaded_error"
		errMsg = "Upstream service overloaded"
	default:
		statusCode = http.StatusBadGateway
		errType = "upstream_error"
		errMsg = "Upstream request failed"
	}
	c.JSON(statusCode, gin.H{"type": "error", "error": gin.H{"type": errType, "message": errMsg}})
	if upstreamMsg == "" {
		return fmt.Errorf("upstream error: %d", upstreamStatus)
	}
	return fmt.Errorf("upstream error: %d message=%s", upstreamStatus, upstreamMsg)
}
func (s *AntigravityGatewayService) writeGoogleError(c *gin.Context, status int, message string) error {
	statusStr := "UNKNOWN"
	switch status {
	case 400:
		statusStr = "INVALID_ARGUMENT"
	case 404:
		statusStr = "NOT_FOUND"
	case 429:
		statusStr = "RESOURCE_EXHAUSTED"
	case 500:
		statusStr = "INTERNAL"
	case 502, 503:
		statusStr = "UNAVAILABLE"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": status, "message": message, "status": statusStr}})
	return fmt.Errorf("%s", message)
}
func (s *AntigravityGatewayService) handleClaudeStreamToNonStreaming(c *gin.Context, resp *http.Response, startTime time.Time, originalModel string) (*antigravityStreamResult, error) {
	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.settingService.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	scanner.Buffer(scanBuf[:0], maxLineSize)
	var firstTokenMs *int
	var last map[string]any
	var lastWithParts map[string]any
	var collectedParts []map[string]any
	type scanEvent struct {
		line string
		err  error
	}
	events := make(chan scanEvent, 16)
	done := make(chan struct{})
	sendEvent := func(ev scanEvent) bool {
		select {
		case events <- ev:
			return true
		case <-done:
			return false
		}
	}
	var lastReadAt int64
	atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
	go func(scanBuf *sseScannerBuf64K) {
		defer putSSEScannerBuf64K(scanBuf)
		defer close(events)
		for scanner.Scan() {
			atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
			if !sendEvent(scanEvent{line: scanner.Text()}) {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			_ = sendEvent(scanEvent{err: err})
		}
	}(scanBuf)
	defer close(done)
	streamInterval := time.Duration(0)
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		streamInterval = time.Duration(s.settingService.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
	}
	var intervalTicker *time.Ticker
	if streamInterval > 0 {
		intervalTicker = time.NewTicker(streamInterval)
		defer intervalTicker.Stop()
	}
	var intervalCh <-chan time.Time
	if intervalTicker != nil {
		intervalCh = intervalTicker.C
	}
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				goto returnResponse
			}
			if ev.err != nil {
				if errors.Is(ev.err, bufio.ErrTooLong) {
					logger.LegacyPrintf("service.antigravity_gateway", "SSE line too long (antigravity claude non-stream): max_size=%d error=%v", maxLineSize, ev.err)
				}
				return nil, ev.err
			}
			line := ev.line
			trimmed := strings.TrimRight(line, "\r\n")
			if !strings.HasPrefix(trimmed, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
			if payload == "" || payload == "[DONE]" {
				continue
			}
			inner, parseErr := s.unwrapV1InternalResponse([]byte(payload))
			if parseErr != nil {
				continue
			}
			var parsed map[string]any
			if err := json.Unmarshal(inner, &parsed); err != nil {
				continue
			}
			if firstTokenMs == nil {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}
			last = parsed
			if parts := extractGeminiParts(parsed); len(parts) > 0 {
				lastWithParts = parsed
				collectedParts = append(collectedParts, parts...)
			}
		case <-intervalCh:
			lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
			if time.Since(lastRead) < streamInterval {
				continue
			}
			logger.LegacyPrintf("service.antigravity_gateway", "Stream data interval timeout (antigravity claude non-stream)")
			return nil, fmt.Errorf("stream data interval timeout")
		}
	}
returnResponse:
	finalResponse := pickGeminiCollectResult(last, lastWithParts)
	if last == nil && lastWithParts == nil {
		logger.LegacyPrintf("service.antigravity_gateway", "[antigravity-Forward] warning: empty stream response (claude non-stream), triggering failover")
		return nil, &UpstreamFailoverError{StatusCode: http.StatusBadGateway, ResponseBody: []byte(`{"error":"empty stream response from upstream"}`), RetryableOnSameAccount: true}
	}
	if len(collectedParts) > 0 {
		finalResponse = mergeCollectedPartsToResponse(finalResponse, collectedParts)
	}
	geminiBody, err := json.Marshal(finalResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini response: %w", err)
	}
	claudeResp, agUsage, err := antigravity.TransformGeminiToClaude(geminiBody, originalModel)
	if err != nil {
		logger.LegacyPrintf("service.antigravity_gateway", "[antigravity-Forward] transform_error error=%v body=%s", err, string(geminiBody))
		return nil, s.writeClaudeError(c, http.StatusBadGateway, "upstream_error", "Failed to parse upstream response")
	}
	c.Data(http.StatusOK, "application/json", claudeResp)
	usage := &ClaudeUsage{InputTokens: agUsage.InputTokens, OutputTokens: agUsage.OutputTokens, CacheCreationInputTokens: agUsage.CacheCreationInputTokens, CacheReadInputTokens: agUsage.CacheReadInputTokens}
	return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs}, nil
}
func (s *AntigravityGatewayService) handleClaudeStreamingResponse(c *gin.Context, resp *http.Response, startTime time.Time, originalModel string) (*antigravityStreamResult, error) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}
	processor := antigravity.NewStreamingProcessor(originalModel)
	var firstTokenMs *int
	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.settingService.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	scanner.Buffer(scanBuf[:0], maxLineSize)
	convertUsage := func(agUsage *antigravity.ClaudeUsage) *ClaudeUsage {
		if agUsage == nil {
			return &ClaudeUsage{}
		}
		return &ClaudeUsage{InputTokens: agUsage.InputTokens, OutputTokens: agUsage.OutputTokens, CacheCreationInputTokens: agUsage.CacheCreationInputTokens, CacheReadInputTokens: agUsage.CacheReadInputTokens}
	}
	type scanEvent struct {
		line string
		err  error
	}
	events := make(chan scanEvent, 16)
	done := make(chan struct{})
	sendEvent := func(ev scanEvent) bool {
		select {
		case events <- ev:
			return true
		case <-done:
			return false
		}
	}
	var lastReadAt int64
	atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
	go func(scanBuf *sseScannerBuf64K) {
		defer putSSEScannerBuf64K(scanBuf)
		defer close(events)
		for scanner.Scan() {
			atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
			if !sendEvent(scanEvent{line: scanner.Text()}) {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			_ = sendEvent(scanEvent{err: err})
		}
	}(scanBuf)
	defer close(done)
	streamInterval := time.Duration(0)
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		streamInterval = time.Duration(s.settingService.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
	}
	var intervalTicker *time.Ticker
	if streamInterval > 0 {
		intervalTicker = time.NewTicker(streamInterval)
		defer intervalTicker.Stop()
	}
	var intervalCh <-chan time.Time
	if intervalTicker != nil {
		intervalCh = intervalTicker.C
	}
	cw := newAntigravityClientWriter(c.Writer, flusher, "antigravity claude")
	errorEventSent := false
	sendErrorEvent := func(reason string) {
		if errorEventSent || cw.Disconnected() {
			return
		}
		errorEventSent = true
		_, _ = fmt.Fprintf(c.Writer, "event: error\ndata: {\"error\":\"%s\"}\n\n", reason)
		flusher.Flush()
	}
	finishUsage := func() *ClaudeUsage {
		_, agUsage := processor.Finish()
		return convertUsage(agUsage)
	}
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				finalEvents, agUsage := processor.Finish()
				if len(finalEvents) > 0 {
					cw.Write(finalEvents)
				} else if !processor.MessageStartSent() && !cw.Disconnected() {
					logger.LegacyPrintf("service.antigravity_gateway", "[antigravity-Claude-Stream] empty stream response (no valid events parsed), triggering failover")
					return nil, &UpstreamFailoverError{StatusCode: http.StatusBadGateway, ResponseBody: []byte(`{"error":"empty stream response from upstream"}`), RetryableOnSameAccount: true}
				}
				return &antigravityStreamResult{usage: convertUsage(agUsage), firstTokenMs: firstTokenMs, clientDisconnect: cw.Disconnected()}, nil
			}
			if ev.err != nil {
				if disconnect, handled := handleStreamReadError(ev.err, cw.Disconnected(), "antigravity claude"); handled {
					return &antigravityStreamResult{usage: finishUsage(), firstTokenMs: firstTokenMs, clientDisconnect: disconnect}, nil
				}
				if errors.Is(ev.err, bufio.ErrTooLong) {
					logger.LegacyPrintf("service.antigravity_gateway", "SSE line too long (antigravity): max_size=%d error=%v", maxLineSize, ev.err)
					sendErrorEvent("response_too_large")
					return &antigravityStreamResult{usage: convertUsage(nil), firstTokenMs: firstTokenMs}, ev.err
				}
				sendErrorEvent("stream_read_error")
				return nil, fmt.Errorf("stream read error: %w", ev.err)
			}
			claudeEvents := processor.ProcessLine(strings.TrimRight(ev.line, "\r\n"))
			if len(claudeEvents) > 0 {
				if firstTokenMs == nil {
					ms := int(time.Since(startTime).Milliseconds())
					firstTokenMs = &ms
				}
				cw.Write(claudeEvents)
			}
		case <-intervalCh:
			lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
			if time.Since(lastRead) < streamInterval {
				continue
			}
			if cw.Disconnected() {
				logger.LegacyPrintf("service.antigravity_gateway", "Upstream timeout after client disconnect (antigravity claude), returning collected usage")
				return &antigravityStreamResult{usage: finishUsage(), firstTokenMs: firstTokenMs, clientDisconnect: true}, nil
			}
			logger.LegacyPrintf("service.antigravity_gateway", "Stream data interval timeout (antigravity)")
			sendErrorEvent("stream_timeout")
			return &antigravityStreamResult{usage: convertUsage(nil), firstTokenMs: firstTokenMs}, fmt.Errorf("stream data interval timeout")
		}
	}
}
func (s *AntigravityGatewayService) extractSSEUsage(line string, usage *ClaudeUsage) {
	if !strings.HasPrefix(line, "data: ") {
		return
	}
	dataStr := strings.TrimPrefix(line, "data: ")
	var event map[string]any
	if json.Unmarshal([]byte(dataStr), &event) != nil {
		return
	}
	u, ok := event["usage"].(map[string]any)
	if !ok {
		return
	}
	if v, ok := u["input_tokens"].(float64); ok && int(v) > 0 {
		usage.InputTokens = int(v)
	}
	if v, ok := u["output_tokens"].(float64); ok && int(v) > 0 {
		usage.OutputTokens = int(v)
	}
	if v, ok := u["cache_read_input_tokens"].(float64); ok && int(v) > 0 {
		usage.CacheReadInputTokens = int(v)
	}
	if v, ok := u["cache_creation_input_tokens"].(float64); ok && int(v) > 0 {
		usage.CacheCreationInputTokens = int(v)
	}
	if cc, ok := u["cache_creation"].(map[string]any); ok {
		if v, ok := cc["ephemeral_5m_input_tokens"].(float64); ok {
			usage.CacheCreation5mTokens = int(v)
		}
		if v, ok := cc["ephemeral_1h_input_tokens"].(float64); ok {
			usage.CacheCreation1hTokens = int(v)
		}
	}
}
func (s *AntigravityGatewayService) extractClaudeUsage(body []byte) *ClaudeUsage {
	usage := &ClaudeUsage{}
	var resp map[string]any
	if json.Unmarshal(body, &resp) != nil {
		return usage
	}
	if u, ok := resp["usage"].(map[string]any); ok {
		if v, ok := u["input_tokens"].(float64); ok {
			usage.InputTokens = int(v)
		}
		if v, ok := u["output_tokens"].(float64); ok {
			usage.OutputTokens = int(v)
		}
		if v, ok := u["cache_read_input_tokens"].(float64); ok {
			usage.CacheReadInputTokens = int(v)
		}
		if v, ok := u["cache_creation_input_tokens"].(float64); ok {
			usage.CacheCreationInputTokens = int(v)
		}
		if cc, ok := u["cache_creation"].(map[string]any); ok {
			if v, ok := cc["ephemeral_5m_input_tokens"].(float64); ok {
				usage.CacheCreation5mTokens = int(v)
			}
			if v, ok := cc["ephemeral_1h_input_tokens"].(float64); ok {
				usage.CacheCreation1hTokens = int(v)
			}
		}
	}
	return usage
}
