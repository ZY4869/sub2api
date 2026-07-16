package service

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type openaiStreamingResult struct {
	usage        *OpenAIUsage
	firstTokenMs *int
}

func (s *OpenAIGatewayService) handleStreamingResponse(ctx context.Context, resp *http.Response, c *gin.Context, account *Account, startTime time.Time, originalModel, mappedModel string, reasoningEffortOpt ...string) (*openaiStreamingResult, error) {
	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	if v := resp.Header.Get("x-request-id"); v != "" {
		c.Header("x-request-id", v)
	}
	reasoningEffort := ""
	if len(reasoningEffortOpt) > 0 {
		reasoningEffort = strings.TrimSpace(reasoningEffortOpt[0])
	}
	firstOutputTimeout := s.openAIFirstOutputTimeout(reasoningEffort)
	firstOutputDeadline := time.Time{}
	if firstOutputTimeout > 0 {
		firstOutputDeadline = startTime.Add(firstOutputTimeout)
	}
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}
	bufferedWriter := bufio.NewWriterSize(w, 4*1024)
	flushBuffered := func() error {
		if err := bufferedWriter.Flush(); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}
	usage := &OpenAIUsage{}
	var firstTokenMs *int
	shouldCountImageOutputs := false
	if surface, ok := ImageRequestSurfaceMetadataFromContext(ctx); ok && surface == "responses_tool" {
		shouldCountImageOutputs = true
	}
	seenOutputIndex := map[int64]struct{}{}
	fallbackImageCount := 0
	if shouldCountImageOutputs {
		seenOutputIndex = make(map[int64]struct{}, 4)
	}
	finalizeImageOutputCount := func() {
		if !shouldCountImageOutputs {
			return
		}
		outputCount := len(seenOutputIndex)
		if outputCount == 0 {
			outputCount = fallbackImageCount
		}
		if outputCount <= 0 {
			return
		}
		if c != nil && c.Request != nil {
			mdCtx := EnsureRequestMetadata(c.Request.Context())
			SetImageOutputCountMetadata(mdCtx, outputCount)
			c.Request = c.Request.WithContext(mdCtx)
			return
		}
		SetImageOutputCountMetadata(ctx, outputCount)
	}
	defer finalizeImageOutputCount()
	reader, cleanup, err := upstreamResponseBodyReader(resp)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	scanner := bufio.NewScanner(reader)
	maxLineSize := defaultMaxLineSize
	if s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	scanner.Buffer(scanBuf[:0], maxLineSize)
	streamInterval := time.Duration(0)
	if s.cfg != nil && s.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		streamInterval = time.Duration(s.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
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
	keepaliveInterval := time.Duration(0)
	if s.cfg != nil && s.cfg.Gateway.StreamKeepaliveInterval > 0 {
		keepaliveInterval = time.Duration(s.cfg.Gateway.StreamKeepaliveInterval) * time.Second
	}
	var keepaliveTicker *time.Ticker
	if keepaliveInterval > 0 {
		keepaliveTicker = time.NewTicker(keepaliveInterval)
		defer keepaliveTicker.Stop()
	}
	var keepaliveCh <-chan time.Time
	if keepaliveTicker != nil {
		keepaliveCh = keepaliveTicker.C
	}
	var firstOutputTimer *time.Timer
	if !firstOutputDeadline.IsZero() {
		remaining := time.Until(firstOutputDeadline)
		if remaining <= 0 {
			remaining = time.Nanosecond
		}
		firstOutputTimer = time.NewTimer(remaining)
		defer firstOutputTimer.Stop()
	}
	var firstOutputCh <-chan time.Time
	if firstOutputTimer != nil {
		firstOutputCh = firstOutputTimer.C
	}
	const firstOutputStageLimit = 1024 * 1024
	var firstOutputStage strings.Builder
	firstOutputStageActive := firstOutputTimeout > 0
	lastDataAt := time.Now()
	errorEventSent := false
	clientDisconnected := false
	upstreamErrorForwarded := false
	sendErrorEvent := func(reason string) {
		if errorEventSent || clientDisconnected {
			return
		}
		errorEventSent = true
		payload := `{"type":"error","sequence_number":0,"error":{"type":"upstream_error","message":` + strconv.Quote(reason) + `,"code":` + strconv.Quote(reason) + `}}`
		if err := flushBuffered(); err != nil {
			clientDisconnected = true
			return
		}
		if _, err := bufferedWriter.WriteString("data: " + payload + "\n\n"); err != nil {
			clientDisconnected = true
			return
		}
		if err := flushBuffered(); err != nil {
			clientDisconnected = true
		}
	}
	sendFailureEvent := func(reason string) {
		if errorEventSent || clientDisconnected {
			return
		}
		errorEventSent = true
		msg := reason
		if msg == "" {
			msg = "Upstream request failed"
		}
		payload := `event: response.failed` + "\n" +
			`data: {"type":"response.failed","response":{"status":"failed","output":[],"error":{"code":` + strconv.Quote(reason) +
			`,"message":` + strconv.Quote(msg) + `}}}` + "\n\n"
		if _, err := bufferedWriter.WriteString(payload); err != nil {
			clientDisconnected = true
			return
		}
		if err := flushBuffered(); err != nil {
			clientDisconnected = true
		}
	}
	needModelReplace := originalModel != mappedModel
	resultWithUsage := func() *openaiStreamingResult {
		return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}
	}
	flushFirstOutputStage := func() bool {
		if firstOutputStage.Len() == 0 || clientDisconnected {
			firstOutputStage.Reset()
			return true
		}
		if _, err := bufferedWriter.WriteString(firstOutputStage.String()); err != nil {
			clientDisconnected = true
			firstOutputStage.Reset()
			logger.LegacyPrintf("service.openai_gateway", "Client disconnected while flushing staged OpenAI SSE events")
			return false
		}
		firstOutputStage.Reset()
		return true
	}
	finalizeStream := func() (*openaiStreamingResult, error) {
		_ = flushFirstOutputStage()
		if !clientDisconnected {
			if err := flushBuffered(); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.openai_gateway", "Client disconnected during final flush, returning collected usage")
			}
		}
		return resultWithUsage(), nil
	}
	handleScanErr := func(scanErr error) (*openaiStreamingResult, error, bool) {
		if scanErr == nil {
			return nil, nil, false
		}
		if errors.Is(scanErr, context.Canceled) || errors.Is(scanErr, context.DeadlineExceeded) {
			logger.LegacyPrintf("service.openai_gateway", "Context canceled during streaming, returning collected usage")
			return resultWithUsage(), nil, true
		}
		if clientDisconnected {
			logger.LegacyPrintf("service.openai_gateway", "Upstream read error after client disconnect: %v, returning collected usage", scanErr)
			return resultWithUsage(), nil, true
		}
		if upstreamErrorForwarded {
			logger.LegacyPrintf("service.openai_gateway", "Upstream read error after upstream SSE error was forwarded: account=%d error=%v", account.ID, scanErr)
			return resultWithUsage(), nil, true
		}
		if errors.Is(scanErr, bufio.ErrTooLong) {
			logger.LegacyPrintf("service.openai_gateway", "SSE line too long: account=%d max_size=%d error=%v", account.ID, maxLineSize, scanErr)
			if isOpenAIResponsesCompactPath(c) {
				sendFailureEvent("response_too_large")
			} else {
				sendErrorEvent("response_too_large")
			}
			return resultWithUsage(), scanErr, true
		}
		if isOpenAIResponsesCompactPath(c) {
			sendFailureEvent("stream_read_error")
		} else {
			sendErrorEvent("stream_read_error")
		}
		return resultWithUsage(), fmt.Errorf("stream read error: %w", scanErr), true
	}
	processSSELine := func(line string, queueDrained bool) {
		lastDataAt = time.Now()
		if isOpenAIUpstreamSSEErrorLine(line) {
			upstreamErrorForwarded = true
		}
		if data, ok := extractOpenAISSEDataLine(line); ok {
			if needModelReplace && mappedModel != "" && strings.Contains(data, mappedModel) {
				line = s.replaceModelInSSELine(line, mappedModel, originalModel)
			}
			dataBytes := []byte(data)
			if correctedData, corrected := s.toolCorrector.CorrectToolCallsInSSEBytes(dataBytes); corrected {
				dataBytes = correctedData
				data = string(correctedData)
				line = "data: " + data
			}
			if shouldCountImageOutputs {
				switch strings.TrimSpace(gjson.GetBytes(dataBytes, "type").String()) {
				case "response.image_generation_call.partial_image":
					if idx := gjson.GetBytes(dataBytes, "output_index"); idx.Exists() {
						seenOutputIndex[idx.Int()] = struct{}{}
					}
				case "response.completed", "response.done":
					if len(seenOutputIndex) == 0 && fallbackImageCount == 0 {
						fallbackImageCount = countOpenAIResponsesOutputImagesFromTerminalEvent(dataBytes)
					}
				}
			}
			semanticOutput := openAISSEDataHasSemanticOutput(dataBytes)
			if semanticOutput && firstOutputTimer != nil {
				if firstOutputTimer.Stop() {
					firstOutputCh = nil
				}
				firstOutputTimer = nil
			}
			if !clientDisconnected {
				shouldFlush := queueDrained
				if firstTokenMs == nil && semanticOutput {
					shouldFlush = true
				}
				if semanticOutput && firstOutputStageActive {
					firstOutputStageActive = false
					_ = flushFirstOutputStage()
				}
				if firstOutputStageActive && !semanticOutput && firstOutputStage.Len()+len(line)+1 <= firstOutputStageLimit {
					firstOutputStage.WriteString(line)
					firstOutputStage.WriteByte('\n')
				} else if firstOutputStageActive && !semanticOutput {
					firstOutputStageActive = false
					_ = flushFirstOutputStage()
				}
				if firstOutputStageActive && !semanticOutput {
					// Hold pre-output status frames until the first semantic event
					// arrives so a first-output timeout can cleanly retry another account.
				} else if _, err := bufferedWriter.WriteString(line); err != nil {
					clientDisconnected = true
					logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming, continuing to drain upstream for billing")
				} else if _, err := bufferedWriter.WriteString("\n"); err != nil {
					clientDisconnected = true
					logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming, continuing to drain upstream for billing")
				} else if shouldFlush {
					if err := flushBuffered(); err != nil {
						clientDisconnected = true
						logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming flush, continuing to drain upstream for billing")
					}
				}
			}
			if firstTokenMs == nil && semanticOutput {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}
			if semanticOutput {
				markOpenAIRealSSEStarted(c)
			}
			s.parseSSEUsageBytes(dataBytes, usage)
			return
		}
		if !firstOutputStageActive && !isOpenAIKeepaliveSSELine(line) {
			markOpenAIRealSSEStarted(c)
		}
		if !clientDisconnected {
			if firstOutputStageActive && firstOutputStage.Len()+len(line)+1 <= firstOutputStageLimit {
				firstOutputStage.WriteString(line)
				firstOutputStage.WriteByte('\n')
			} else if firstOutputStageActive {
				firstOutputStageActive = false
				_ = flushFirstOutputStage()
			}
			if firstOutputStageActive {
				// Hold comments/event-name lines paired with pre-output data.
			} else if _, err := bufferedWriter.WriteString(line); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming, continuing to drain upstream for billing")
			} else if _, err := bufferedWriter.WriteString("\n"); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming, continuing to drain upstream for billing")
			} else if queueDrained {
				if err := flushBuffered(); err != nil {
					clientDisconnected = true
					logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming flush, continuing to drain upstream for billing")
				}
			}
		}
	}
	if streamInterval <= 0 && keepaliveInterval <= 0 && firstOutputTimeout <= 0 {
		defer putSSEScannerBuf64K(scanBuf)
		for scanner.Scan() {
			processSSELine(scanner.Text(), true)
		}
		if result, err, done := handleScanErr(scanner.Err()); done {
			return result, err
		}
		return finalizeStream()
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
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return finalizeStream()
			}
			if result, err, done := handleScanErr(ev.err); done {
				return result, err
			}
			processSSELine(ev.line, len(events) == 0)
		case <-intervalCh:
			lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
			if time.Since(lastRead) < streamInterval {
				continue
			}
			if clientDisconnected {
				logger.LegacyPrintf("service.openai_gateway", "Upstream timeout after client disconnect, returning collected usage")
				return resultWithUsage(), nil
			}
			logger.LegacyPrintf("service.openai_gateway", "Stream data interval timeout: account=%d model=%s interval=%s", account.ID, originalModel, streamInterval)
			if s.rateLimitService != nil {
				s.rateLimitService.HandleStreamTimeout(ctx, account, originalModel)
			}
			if isOpenAIResponsesCompactPath(c) {
				sendFailureEvent("stream_timeout")
			} else {
				sendErrorEvent("stream_timeout")
			}
			return resultWithUsage(), fmt.Errorf("stream data interval timeout")
		case <-keepaliveCh:
			if clientDisconnected {
				continue
			}
			if time.Since(lastDataAt) < keepaliveInterval {
				continue
			}
			if _, err := bufferedWriter.WriteString(":\n\n"); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.openai_gateway", "Client disconnected during streaming, continuing to drain upstream for billing")
				continue
			}
			if err := flushBuffered(); err != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.openai_gateway", "Client disconnected during keepalive flush, continuing to drain upstream for billing")
			}
		case <-firstOutputCh:
			if clientDisconnected {
				logger.LegacyPrintf("service.openai_gateway", "First output timeout after client disconnect, returning collected usage")
				return resultWithUsage(), nil
			}
			if c != nil && c.Writer != nil && c.Writer.Written() {
				if isOpenAIResponsesCompactPath(c) {
					sendFailureEvent("first_output_timeout")
				} else {
					sendErrorEvent("first_output_timeout")
				}
			}
			return resultWithUsage(), s.newOpenAIFirstOutputTimeoutError(ctx, c, account, startTime, originalModel, reasoningEffort, firstOutputTimeout, resp.Header)
		}
	}
}

func extractOpenAISSEDataLine(line string) (string, bool) {
	if !strings.HasPrefix(line, "data:") {
		return "", false
	}
	start := len("data:")
	for start < len(line) {
		if line[start] != ' ' && line[start] != '\t' {
			break
		}
		start++
	}
	return line[start:], true
}

func isOpenAIUpstreamSSEErrorLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "event:") {
		eventName := strings.TrimSpace(trimmed[len("event:"):])
		return strings.EqualFold(eventName, "error")
	}
	data, ok := extractOpenAISSEDataLine(trimmed)
	if !ok {
		return false
	}
	data = strings.TrimSpace(data)
	if data == "" || data == "[DONE]" || !gjson.Valid(data) {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(gjson.Get(data, "type").String()), "error") {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(gjson.Get(data, "type").String()), "response.failed") &&
		gjson.Get(data, "response.error").Exists() {
		return true
	}
	errObj := gjson.Get(data, "error")
	if !errObj.Exists() || errObj.Type == gjson.Null {
		return false
	}
	if errObj.IsObject() {
		return gjson.Get(data, "error.message").Exists() ||
			gjson.Get(data, "error.type").Exists() ||
			gjson.Get(data, "error.code").Exists()
	}
	return errObj.String() != ""
}

func (s *OpenAIGatewayService) replaceModelInSSELine(line, fromModel, toModel string) string {
	data, ok := extractOpenAISSEDataLine(line)
	if !ok {
		return line
	}
	if data == "" || data == "[DONE]" {
		return line
	}
	if m := gjson.Get(data, "model"); m.Exists() && m.Str == fromModel {
		newData, err := sjson.Set(data, "model", toModel)
		if err != nil {
			return line
		}
		return "data: " + newData
	}
	if m := gjson.Get(data, "response.model"); m.Exists() && m.Str == fromModel {
		newData, err := sjson.Set(data, "response.model", toModel)
		if err != nil {
			return line
		}
		return "data: " + newData
	}
	return line
}

func (s *OpenAIGatewayService) correctToolCallsInResponseBody(body []byte) []byte {
	if len(body) == 0 {
		return body
	}
	corrected, changed := s.toolCorrector.CorrectToolCallsInSSEBytes(body)
	if changed {
		return corrected
	}
	return body
}

func (s *OpenAIGatewayService) parseSSEUsage(data string, usage *OpenAIUsage) {
	s.parseSSEUsageBytes([]byte(data), usage)
}

func (s *OpenAIGatewayService) parseSSEUsageBytes(data []byte, usage *OpenAIUsage) {
	if usage == nil || len(data) == 0 || bytes.Equal(data, []byte("[DONE]")) {
		return
	}
	if len(data) < 80 {
		return
	}
	if !bytes.Contains(data, []byte(`"response.completed"`)) && !bytes.Contains(data, []byte(`"response.done"`)) {
		return
	}
	eventType := gjson.GetBytes(data, "type").String()
	if eventType != "response.completed" && eventType != "response.done" {
		return
	}
	usage.InputTokens = int(gjson.GetBytes(data, "response.usage.input_tokens").Int())
	usage.OutputTokens = int(gjson.GetBytes(data, "response.usage.output_tokens").Int())
	if miss := gjson.GetBytes(data, "response.usage.prompt_cache_miss_tokens"); miss.Exists() && miss.Int() > 0 {
		usage.CacheCreationInputTokens = int(miss.Int())
	}
	usage.CacheReadInputTokens = int(firstPositiveInt64(
		gjson.GetBytes(data, "response.usage.prompt_cache_hit_tokens").Int(),
		gjson.GetBytes(data, "response.usage.input_tokens_details.cached_tokens").Int(),
	))
}

func countOpenAIResponsesOutputImagesFromTerminalEvent(data []byte) int {
	if len(data) == 0 || !gjson.ValidBytes(data) {
		return 0
	}
	count := 0
	for _, output := range gjson.GetBytes(data, "response.output").Array() {
		if strings.TrimSpace(output.Get("type").String()) != "message" {
			continue
		}
		for _, content := range output.Get("content").Array() {
			if strings.TrimSpace(content.Get("type").String()) != "output_image" {
				continue
			}
			if strings.TrimSpace(content.Get("image_url").String()) == "" {
				continue
			}
			count++
		}
	}
	return count
}
