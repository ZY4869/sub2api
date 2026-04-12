package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	opsRequestTraceTimeout      = 5 * time.Second
	opsRequestTraceWorkerMin    = 2
	opsRequestTraceWorkerMax    = 16
	opsRequestTraceQueueMin     = 256
	opsRequestTraceQueueMax     = 4096
	opsRequestTraceQueuePerWork = 64
	opsRequestTraceBodyLimit    = 1024 * 1024
	opsRequestTraceSampleRate   = 0.1
	opsRequestTraceSlowMs       = int64(3000)
)

type opsRequestTraceJob struct {
	ops   *service.OpsService
	input *service.OpsRecordRequestTraceInput
}

type opsRequestTraceCaptureWriter struct {
	gin.ResponseWriter
	limit     int
	total     int
	truncated bool
	buf       bytes.Buffer
}

var (
	opsRequestTraceOnce  sync.Once
	opsRequestTraceQueue chan opsRequestTraceJob

	opsRequestTraceQueueLen atomic.Int64
	opsRequestTraceDropped  atomic.Int64
	opsRequestTraceStop     atomic.Bool
	opsRequestTraceWorkers  sync.WaitGroup

	opsRequestTraceWriterPool = sync.Pool{
		New: func() any {
			return &opsRequestTraceCaptureWriter{limit: opsRequestTraceBodyLimit}
		},
	}
)

func OpsRequestTraceMiddleware(ops *service.OpsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ops == nil {
			c.Next()
			return
		}

		startedAt := time.Now()
		originalWriter := c.Writer
		writer := acquireOpsRequestTraceCaptureWriter(originalWriter)
		c.Writer = writer
		defer func() {
			if c.Writer == writer {
				c.Writer = originalWriter
			}
			releaseOpsRequestTraceCaptureWriter(writer)
		}()

		c.Next()

		if !ops.IsMonitoringEnabled(c.Request.Context()) {
			return
		}

		input := buildOpsRequestTraceInput(c, writer, startedAt)
		if !shouldQueueOpsRequestTrace(input) {
			return
		}
		enqueueOpsRequestTrace(ops, input)
	}
}

func acquireOpsRequestTraceCaptureWriter(rw gin.ResponseWriter) *opsRequestTraceCaptureWriter {
	writer, ok := opsRequestTraceWriterPool.Get().(*opsRequestTraceCaptureWriter)
	if !ok || writer == nil {
		writer = &opsRequestTraceCaptureWriter{}
	}
	writer.ResponseWriter = rw
	writer.limit = opsRequestTraceBodyLimit
	writer.total = 0
	writer.truncated = false
	writer.buf.Reset()
	return writer
}

func releaseOpsRequestTraceCaptureWriter(writer *opsRequestTraceCaptureWriter) {
	if writer == nil {
		return
	}
	writer.ResponseWriter = nil
	writer.limit = opsRequestTraceBodyLimit
	writer.total = 0
	writer.truncated = false
	writer.buf.Reset()
	opsRequestTraceWriterPool.Put(writer)
}

func (w *opsRequestTraceCaptureWriter) Write(data []byte) (int, error) {
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *opsRequestTraceCaptureWriter) WriteString(value string) (int, error) {
	w.capture([]byte(value))
	return w.ResponseWriter.WriteString(value)
}

func (w *opsRequestTraceCaptureWriter) capture(data []byte) {
	if w == nil || len(data) == 0 {
		return
	}
	w.total += len(data)
	if w.limit <= 0 || w.buf.Len() >= w.limit {
		w.truncated = true
		return
	}
	remaining := w.limit - w.buf.Len()
	if len(data) > remaining {
		_, _ = w.buf.Write(data[:remaining])
		w.truncated = true
		return
	}
	_, _ = w.buf.Write(data)
}

func (w *opsRequestTraceCaptureWriter) BytesCopy() []byte {
	if w == nil || w.buf.Len() == 0 {
		return nil
	}
	return append([]byte(nil), w.buf.Bytes()...)
}

func startOpsRequestTraceWorkers() {
	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < opsRequestTraceWorkerMin {
		workerCount = opsRequestTraceWorkerMin
	}
	if workerCount > opsRequestTraceWorkerMax {
		workerCount = opsRequestTraceWorkerMax
	}

	queueSize := workerCount * opsRequestTraceQueuePerWork
	if queueSize < opsRequestTraceQueueMin {
		queueSize = opsRequestTraceQueueMin
	}
	if queueSize > opsRequestTraceQueueMax {
		queueSize = opsRequestTraceQueueMax
	}

	opsRequestTraceQueue = make(chan opsRequestTraceJob, queueSize)
	opsRequestTraceWorkers.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer opsRequestTraceWorkers.Done()
			for job := range opsRequestTraceQueue {
				opsRequestTraceQueueLen.Add(-1)
				if job.ops == nil || job.input == nil {
					continue
				}
				ctx, cancel := context.WithTimeout(context.Background(), opsRequestTraceTimeout)
				_ = job.ops.RecordRequestTrace(ctx, job.input)
				cancel()
			}
		}()
	}
}

func enqueueOpsRequestTrace(ops *service.OpsService, input *service.OpsRecordRequestTraceInput) {
	if ops == nil || input == nil || opsRequestTraceStop.Load() {
		return
	}
	opsRequestTraceOnce.Do(startOpsRequestTraceWorkers)
	select {
	case opsRequestTraceQueue <- opsRequestTraceJob{ops: ops, input: input}:
		opsRequestTraceQueueLen.Add(1)
	default:
		opsRequestTraceDropped.Add(1)
		log.Printf("[OpsRequestTrace] queue is full; dropping request trace (dropped_total=%d)", opsRequestTraceDropped.Load())
	}
}

func buildOpsRequestTraceInput(c *gin.Context, writer *opsRequestTraceCaptureWriter, startedAt time.Time) *service.OpsRecordRequestTraceInput {
	if c == nil || c.Request == nil {
		return nil
	}

	apiKey, _ := middleware2.GetAPIKeyFromContext(c)
	subject, _ := middleware2.GetAuthSubjectFromContext(c)

	var userID *int64
	if subject.UserID > 0 {
		userID = &subject.UserID
	}
	var apiKeyID *int64
	var groupID *int64
	if apiKey != nil {
		apiKeyID = &apiKey.ID
		groupID = apiKey.GroupID
		if userID == nil && apiKey.User != nil && apiKey.User.ID > 0 {
			userID = &apiKey.User.ID
		}
	}

	var accountID *int64
	if value, ok := c.Get(opsAccountIDKey); ok {
		if parsed, ok := value.(int64); ok && parsed > 0 {
			accountID = &parsed
		}
	}

	requestBody := getOpsTraceRequestBody(c)
	responseBody := writer.BytesCopy()
	requestHeadersJSON := marshalOpsTraceHeaders(filterOpsTraceHeaders(c.Request.Header, opsTraceRequestHeaderAllowlist))
	responseHeadersJSON := marshalOpsTraceHeaders(filterOpsTraceHeaders(c.Writer.Header(), opsTraceResponseHeaderAllowlist))

	normalize, usage := buildOpsTraceNormalizeResult(c, apiKey, requestBody, responseBody)
	statusCode := c.Writer.Status()
	input := &service.OpsRecordRequestTraceInput{
		RequestID:          strings.TrimSpace(firstNonEmptyString(c.Writer.Header().Get("X-Request-Id"), c.Writer.Header().Get("x-request-id"))),
		ClientRequestID:    readContextString(c, ctxkey.ClientRequestID),
		UpstreamRequestID:  normalize.UpstreamRequestID,
		UserID:             userID,
		APIKeyID:           apiKeyID,
		AccountID:          accountID,
		GroupID:            groupID,
		StatusCode:         statusCode,
		UpstreamStatusCode: readOpsTraceStatusCode(c),
		DurationMs:         time.Since(startedAt).Milliseconds(),
		TTFTMs:             getContextLatencyMs(c, service.OpsTimeToFirstTokenMsKey),
		InputTokens:        usage.inputTokens,
		OutputTokens:       usage.outputTokens,
		TotalTokens:        usage.totalTokens,
		Trace: service.GatewayTraceContext{
			Normalize:           normalize,
			RequestHeadersJSON:  requestHeadersJSON,
			ResponseHeadersJSON: responseHeadersJSON,
			RawRequest:          requestBody,
			RawResponse:         responseBody,
		},
		CreatedAt: time.Now().UTC(),
	}
	return input
}

type opsTraceUsage struct {
	inputTokens  int
	outputTokens int
	totalTokens  int
}

func buildOpsTraceNormalizeResult(c *gin.Context, apiKey *service.APIKey, requestBody []byte, responseBody []byte) (service.ProtocolNormalizeResult, opsTraceUsage) {
	result := service.ProtocolNormalizeResult{}
	usage := opsTraceUsage{}

	requestJSON := parseOpsTraceJSON(requestBody)
	responseJSON := parseOpsTraceResponseJSON(responseBody)

	result.Platform = resolveOpsTracePlatform(c, apiKey)
	result.ProtocolIn = inferOpsTraceProtocolIn(c)
	result.ProtocolOut = inferOpsTraceProtocolOut(result.ProtocolIn, result.Platform)
	result.Channel = inferOpsTraceChannel(c, result.ProtocolIn, result.ProtocolOut)
	result.RoutePath = inferOpsTraceRoutePath(c)
	result.RequestType = inferOpsTraceRequestType(c)
	result.RequestedModel = strings.TrimSpace(readOpsTraceModel(c))
	result.UpstreamModel = strings.TrimSpace(resolveOpsUpstreamModel(c))
	if result.UpstreamModel == "" {
		result.UpstreamModel = result.RequestedModel
	}

	if requestJSON != nil {
		enrichOpsTraceRequestMetadata(requestJSON, &result)
	}
	if responseJSON != nil {
		enrichOpsTraceResponseMetadata(responseJSON, result.ProtocolOut, &result, &usage)
	}

	if result.ActualUpstreamModel == "" {
		result.ActualUpstreamModel = firstNonEmptyString(result.UpstreamModel, result.RequestedModel)
	}
	if result.UpstreamRequestID == "" {
		result.UpstreamRequestID = strings.TrimSpace(firstNonEmptyString(
			c.Writer.Header().Get("X-Request-Id"),
			c.Writer.Header().Get("x-request-id"),
			c.Writer.Header().Get("x-goog-request-id"),
		))
	}
	if geminiSurface, ok := service.GeminiSurfaceMetadataFromContext(c.Request.Context()); ok {
		result.GeminiSurface = geminiSurface
	} else if result.Platform == service.PlatformGemini {
		classifier := service.NewGeminiRequestClassifier()
		classification := classifier.ClassifyRequest(service.GeminiBillingCalculationInput{
			InboundEndpoint: c.Request.URL.Path,
			RequestBody:     requestBody,
		})
		if classification != nil {
			result.GeminiSurface = classification.Surface
		}
	}
	if billingRuleID, ok := service.BillingRuleIDMetadataFromContext(c.Request.Context()); ok {
		result.BillingRuleID = billingRuleID
	}
	if probeAction, ok := service.ProbeActionMetadataFromContext(c.Request.Context()); ok {
		result.ProbeAction = probeAction
	}
	if headerValue := strings.TrimSpace(c.Writer.Header().Get("X-Sub2api-CountTokens-Source")); headerValue != "" {
		result.CountTokensSource = headerValue
	}
	return result, usage
}

func enrichOpsTraceRequestMetadata(payload map[string]any, result *service.ProtocolNormalizeResult) {
	if payload == nil || result == nil {
		return
	}

	if result.RequestedModel == "" {
		result.RequestedModel = strings.TrimSpace(stringValueFromMap(payload, "model"))
	}
	if stream, ok := payload["stream"].(bool); ok {
		result.Stream = stream
	}

	if generationConfig, ok := payload["generationConfig"].(map[string]any); ok && generationConfig != nil {
		if mediaResolution := strings.TrimSpace(stringValueFromMap(generationConfig, "mediaResolution")); mediaResolution != "" {
			result.MediaResolution = mediaResolution
		}
		if thinkingConfig, ok := generationConfig["thinkingConfig"].(map[string]any); ok && thinkingConfig != nil {
			if thinkingLevel := strings.TrimSpace(stringValueFromMap(thinkingConfig, "thinkingLevel")); thinkingLevel != "" {
				result.HasThinking = true
				result.ThinkingSource = "thinking_level"
				result.ThinkingLevel = thinkingLevel
			}
			if thinkingBudget := intValueFromMap(thinkingConfig, "thinkingBudget"); thinkingBudget != nil {
				result.HasThinking = true
				if result.ThinkingSource == "" {
					result.ThinkingSource = "thinking_budget"
				}
				result.ThinkingBudget = thinkingBudget
			}
		}
	}

	if thinking, ok := payload["thinking"].(map[string]any); ok && thinking != nil {
		result.HasThinking = true
		if result.ThinkingSource == "" {
			result.ThinkingSource = "compat_thinking"
		}
		if level := strings.TrimSpace(stringValueFromMap(thinking, "type")); level != "" && result.ThinkingLevel == "" {
			result.ThinkingLevel = strings.ToUpper(level)
		}
		if budget := firstNonNilInt(thinking["budget_tokens"], thinking["budgetTokens"]); budget != nil && result.ThinkingBudget == nil {
			result.ThinkingBudget = budget
		}
	}

	if reasoningEffort := strings.TrimSpace(stringValueFromMap(payload, "reasoning_effort")); reasoningEffort != "" {
		result.HasThinking = true
		result.ThinkingSource = "mapped_reasoning_effort"
		switch strings.ToLower(reasoningEffort) {
		case "low":
			result.ThinkingLevel = "LOW"
		case "medium":
			result.ThinkingLevel = "MEDIUM"
		case "high", "xhigh":
			result.ThinkingLevel = "HIGH"
		case "none":
			result.ThinkingLevel = "MINIMAL"
		}
	}

	if mediaResolution := strings.TrimSpace(stringValueFromMap(payload, "media_resolution")); mediaResolution != "" && result.MediaResolution == "" {
		result.MediaResolution = mediaResolution
	}
	if mediaResolution := strings.TrimSpace(stringValueFromMap(payload, "mediaResolution")); mediaResolution != "" && result.MediaResolution == "" {
		result.MediaResolution = mediaResolution
	}

	toolKinds := make([]string, 0, 4)
	switch tools := payload["tools"].(type) {
	case []any:
		for _, tool := range tools {
			toolKinds = append(toolKinds, inferOpsTraceToolKinds(tool)...)
		}
	case []map[string]any:
		for _, tool := range tools {
			toolKinds = append(toolKinds, inferOpsTraceToolKinds(tool)...)
		}
	}
	toolKinds = dedupeTraceStrings(toolKinds)
	result.ToolKinds = toolKinds
	result.HasTools = len(toolKinds) > 0
}

func enrichOpsTraceResponseMetadata(payload map[string]any, protocolOut string, result *service.ProtocolNormalizeResult, usage *opsTraceUsage) {
	if payload == nil || result == nil || usage == nil {
		return
	}

	switch strings.ToLower(strings.TrimSpace(protocolOut)) {
	case "gemini", "vertex":
		if responseID := strings.TrimSpace(stringValueFromMap(payload, "responseId")); responseID != "" {
			result.UpstreamRequestID = responseID
		}
		if modelVersion := strings.TrimSpace(stringValueFromMap(payload, "modelVersion")); modelVersion != "" {
			result.ActualUpstreamModel = modelVersion
		}
		if promptFeedback, ok := payload["promptFeedback"].(map[string]any); ok && promptFeedback != nil {
			result.PromptBlockReason = strings.TrimSpace(stringValueFromMap(promptFeedback, "blockReason"))
		}
		if candidates, ok := payload["candidates"].([]any); ok && len(candidates) > 0 {
			if candidate, ok := candidates[0].(map[string]any); ok && candidate != nil {
				result.FinishReason = strings.TrimSpace(stringValueFromMap(candidate, "finishReason"))
				if content, ok := candidate["content"].(map[string]any); ok && content != nil {
					if parts, ok := content["parts"].([]any); ok {
						for _, part := range parts {
							if pm, ok := part.(map[string]any); ok && pm != nil {
								if functionCall, ok := pm["functionCall"].(map[string]any); ok && functionCall != nil {
									result.HasTools = true
									result.ToolKinds = dedupeTraceStrings(append(result.ToolKinds, "function"))
								}
							}
						}
					}
				}
			}
		}
		if usageMetadata, ok := payload["usageMetadata"].(map[string]any); ok && usageMetadata != nil {
			if promptTokens := intValueFromMap(usageMetadata, "promptTokenCount"); promptTokens != nil {
				usage.inputTokens = *promptTokens
			}
			if completionTokens := intValueFromMap(usageMetadata, "candidatesTokenCount"); completionTokens != nil {
				usage.outputTokens = *completionTokens
			}
			if totalTokens := intValueFromMap(usageMetadata, "totalTokenCount"); totalTokens != nil {
				usage.totalTokens = *totalTokens
			}
		}
	default:
		if model := strings.TrimSpace(stringValueFromMap(payload, "model")); model != "" {
			result.ActualUpstreamModel = model
		}
		if responseID := strings.TrimSpace(stringValueFromMap(payload, "id")); responseID != "" {
			result.UpstreamRequestID = responseID
		}
		if stopReason := strings.TrimSpace(stringValueFromMap(payload, "stop_reason")); stopReason != "" {
			result.FinishReason = stopReason
		}
		if usageMap, ok := payload["usage"].(map[string]any); ok && usageMap != nil {
			if inputTokens := firstNonNilInt(usageMap["input_tokens"], usageMap["prompt_tokens"]); inputTokens != nil {
				usage.inputTokens = *inputTokens
			}
			if outputTokens := firstNonNilInt(usageMap["output_tokens"], usageMap["completion_tokens"]); outputTokens != nil {
				usage.outputTokens = *outputTokens
			}
			if totalTokens := firstNonNilInt(usageMap["total_tokens"], usageMap["totalTokens"]); totalTokens != nil {
				usage.totalTokens = *totalTokens
			}
		}
		if choices, ok := payload["choices"].([]any); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]any); ok && choice != nil {
				if finishReason := strings.TrimSpace(stringValueFromMap(choice, "finish_reason")); finishReason != "" && result.FinishReason == "" {
					result.FinishReason = finishReason
				}
			}
		}
	}
}

func shouldQueueOpsRequestTrace(input *service.OpsRecordRequestTraceInput) bool {
	if input == nil {
		return false
	}
	normalize := input.Trace.Normalize
	switch {
	case input.StatusCode >= 400:
		return true
	case input.DurationMs >= opsRequestTraceSlowMs:
		return true
	case normalize.Stream:
		return true
	case normalize.HasTools:
		return true
	case normalize.HasThinking:
		return true
	case isGoogleTraceForQueue(normalize):
		return true
	case normalize.ProtocolIn != "" && normalize.ProtocolOut != "" && normalize.ProtocolIn != normalize.ProtocolOut:
		return true
	default:
		return shouldSampleOpsTrace(opsRequestTraceSampleRate, input)
	}
}

func isGoogleTraceForQueue(normalize service.ProtocolNormalizeResult) bool {
	for _, value := range []string{normalize.Platform, normalize.ProtocolIn, normalize.ProtocolOut, normalize.Channel} {
		value = strings.ToLower(strings.TrimSpace(value))
		if strings.Contains(value, "gemini") || strings.Contains(value, "vertex") || strings.Contains(value, "google") {
			return true
		}
	}
	return false
}

func shouldSampleOpsTrace(rate float64, input *service.OpsRecordRequestTraceInput) bool {
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	key := strings.TrimSpace(firstNonEmptyString(input.RequestID, input.ClientRequestID, input.UpstreamRequestID))
	if key == "" {
		key = input.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	sum := serviceHashString(key)
	return float64(sum%10000)/10000.0 < rate
}

func resolveOpsTracePlatform(c *gin.Context, apiKey *service.APIKey) string {
	if c != nil && c.Request != nil {
		if value, ok := c.Request.Context().Value(ctxkey.Platform).(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if forced, ok := middleware2.GetForcePlatformFromContext(c); ok && strings.TrimSpace(forced) != "" {
		return strings.TrimSpace(forced)
	}
	if apiKey != nil && apiKey.Group != nil && strings.TrimSpace(apiKey.Group.Platform) != "" {
		return strings.TrimSpace(apiKey.Group.Platform)
	}
	return guessPlatformFromPath(c.Request.URL.Path)
}

func inferOpsTraceProtocolIn(c *gin.Context) string {
	path := strings.ToLower(inferOpsTraceRoutePath(c))
	switch {
	case strings.Contains(path, "/publishers/google/models/"):
		return "vertex"
	case strings.HasPrefix(path, "/v1beta"), strings.HasPrefix(path, "/upload/v1beta"), strings.HasPrefix(path, "/download/v1beta"), strings.HasPrefix(path, "/google/batch/archive/v1beta"):
		return "gemini"
	case strings.Contains(path, "/messages"):
		return "claude"
	case strings.Contains(path, "/responses"), strings.Contains(path, "/chat/completions"), strings.Contains(path, "/images/"), strings.Contains(path, "/videos"):
		return "openai"
	default:
		return "gateway"
	}
}

func inferOpsTraceProtocolOut(protocolIn string, platform string) string {
	switch strings.TrimSpace(platform) {
	case service.PlatformGemini:
		return "gemini"
	case service.PlatformAntigravity:
		if protocolIn == "gemini" || protocolIn == "vertex" {
			return protocolIn
		}
		return "anthropic"
	case service.PlatformOpenAI, service.PlatformCopilot, service.PlatformKiro, service.PlatformGrok:
		return "openai"
	default:
		if protocolIn != "" {
			return protocolIn
		}
		return "gateway"
	}
}

func inferOpsTraceChannel(c *gin.Context, protocolIn, protocolOut string) string {
	if c != nil && c.Request != nil {
		if state, ok := service.GatewayChannelStateFromContext(c.Request.Context()); ok && state != nil {
			if channelName := strings.TrimSpace(state.ChannelName()); channelName != "" {
				return channelName
			}
		}
	}
	path := strings.ToLower(inferOpsTraceRoutePath(c))
	switch {
	case strings.Contains(path, "/publishers/google/models/"):
		return "vertex"
	case protocolOut == "gemini" && protocolIn != "gemini":
		return "gemini_compat"
	case protocolOut == "gemini":
		return "ai_studio"
	case protocolOut == "anthropic":
		return "anthropic"
	case protocolOut == "openai":
		return "openai_compat"
	default:
		return protocolOut
	}
}

func inferOpsTraceRoutePath(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if fullPath := strings.TrimSpace(c.FullPath()); fullPath != "" {
		return fullPath
	}
	if c.Request.URL != nil {
		return strings.TrimSpace(c.Request.URL.Path)
	}
	return ""
}

func inferOpsTraceRequestType(c *gin.Context) string {
	stream := false
	if value, ok := c.Get(opsStreamKey); ok {
		if parsed, ok := value.(bool); ok {
			stream = parsed
		}
	}
	if requestType := resolveOpsRequestType(c, stream); requestType != nil {
		return service.RequestTypeFromInt16(*requestType).String()
	}
	return service.RequestTypeFromLegacy(stream, false).String()
}

func getOpsTraceRequestBody(c *gin.Context) []byte {
	if c == nil {
		return nil
	}
	value, ok := c.Get(opsRequestBodyKey)
	if !ok {
		return nil
	}
	raw, ok := value.([]byte)
	if !ok || len(raw) == 0 {
		return nil
	}
	if len(raw) > opsRequestTraceBodyLimit {
		return append([]byte(nil), raw[:opsRequestTraceBodyLimit]...)
	}
	return append([]byte(nil), raw...)
}

func readOpsTraceStatusCode(c *gin.Context) *int {
	if c == nil {
		return nil
	}
	value, ok := c.Get(service.OpsUpstreamStatusCodeKey)
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return &typed
		}
	case int64:
		if typed > 0 {
			value := int(typed)
			return &value
		}
	}
	return nil
}

func readContextString(c *gin.Context, key ctxkey.Key) string {
	if c == nil || c.Request == nil {
		return ""
	}
	value, _ := c.Request.Context().Value(key).(string)
	return strings.TrimSpace(value)
}

func readOpsTraceModel(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if value, ok := c.Get(opsModelKey); ok {
		if model, ok := value.(string); ok {
			return strings.TrimSpace(model)
		}
	}
	return ""
}

func parseOpsTraceJSON(payload []byte) map[string]any {
	if len(payload) == 0 || !json.Valid(payload) {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil
	}
	return out
}

func parseOpsTraceResponseJSON(payload []byte) map[string]any {
	if parsed := parseOpsTraceJSON(payload); parsed != nil {
		return parsed
	}
	lines := strings.Split(string(payload), "\n")
	var last map[string]any
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		if parsed := parseOpsTraceJSON([]byte(data)); parsed != nil {
			last = parsed
		}
	}
	return last
}

func inferOpsTraceToolKinds(tool any) []string {
	payload, ok := tool.(map[string]any)
	if !ok || payload == nil {
		return nil
	}
	kinds := make([]string, 0, 4)
	if typeValue := strings.TrimSpace(stringValueFromMap(payload, "type")); typeValue != "" {
		kinds = append(kinds, typeValue)
	}
	for key, mapped := range map[string]string{
		"googleSearch":          "googleSearch",
		"googleSearchRetrieval": "googleSearch",
		"codeExecution":         "codeExecution",
		"googleMaps":            "googleMaps",
		"fileSearch":            "fileSearch",
		"urlContext":            "urlContext",
		"functionDeclarations":  "function",
	} {
		if _, exists := payload[key]; exists {
			kinds = append(kinds, mapped)
		}
	}
	return dedupeTraceStrings(kinds)
}

func dedupeTraceStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func stringValueFromMap(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	return stringValueFromAny(payload[key])
}

func stringValueFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strings.TrimSpace(strings.TrimRight(strings.TrimRight(strconvFloat(typed), "0"), "."))
	default:
		return ""
	}
}

func intValueFromMap(payload map[string]any, key string) *int {
	if payload == nil {
		return nil
	}
	return firstNonNilInt(payload[key])
}

func firstNonNilInt(values ...any) *int {
	for _, value := range values {
		switch typed := value.(type) {
		case int:
			v := typed
			return &v
		case int32:
			v := int(typed)
			return &v
		case int64:
			v := int(typed)
			return &v
		case float64:
			v := int(typed)
			return &v
		case json.Number:
			if parsed, err := typed.Int64(); err == nil {
				v := int(parsed)
				return &v
			}
		}
	}
	return nil
}

func filterOpsTraceHeaders(headers map[string][]string, allowlist []string) map[string]string {
	if len(headers) == 0 || len(allowlist) == 0 {
		return nil
	}
	allowed := make(map[string]string, len(allowlist))
	for _, key := range allowlist {
		values, ok := headers[key]
		if !ok || len(values) == 0 {
			values = headers[strings.ToLower(key)]
		}
		if len(values) == 0 {
			continue
		}
		allowed[strings.ToLower(key)] = truncateString(strings.Join(values, ", "), 1024)
	}
	if len(allowed) == 0 {
		return nil
	}
	return allowed
}

func marshalOpsTraceHeaders(headers map[string]string) *string {
	if len(headers) == 0 {
		return nil
	}
	raw, err := json.Marshal(headers)
	if err != nil {
		return nil
	}
	value := string(raw)
	return &value
}

func serviceHashString(value string) uint64 {
	var out uint64
	for _, ch := range []byte(value) {
		out = out*131 + uint64(ch)
	}
	return out
}

func strconvFloat(value float64) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

var opsTraceRequestHeaderAllowlist = []string{
	"Content-Type",
	"Accept",
	"User-Agent",
	"anthropic-version",
	"anthropic-beta",
	"openai-beta",
	"x-request-id",
}

var opsTraceResponseHeaderAllowlist = []string{
	"Content-Type",
	"X-Request-Id",
	"x-goog-request-id",
	"X-Sub2api-CountTokens-Source",
	"x-ratelimit-limit-requests",
	"x-ratelimit-remaining-requests",
	"x-ratelimit-reset-requests",
}
