package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

func (s *GrokGatewayService) forwardAPIKeyChatCompletions(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardOfficialChatCompletions(ctx, c, account, body)
}

func (s *GrokGatewayService) forwardOAuthBuildChatCompletions(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardOfficialChatCompletions(ctx, c, account, body)
}

func (s *GrokGatewayService) forwardOfficialChatCompletions(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	reqModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	mappedModel, mappedBody := grokApplyMappedModel(account, reqModel, body)
	if grokIsVideoRequestModel(reqModel, mappedModel) {
		return s.forwardGrokVideoChatCompletions(ctx, c, account, body)
	}
	stream := gjson.GetBytes(mappedBody, "stream").Bool()
	startTime := time.Now()

	resp, meta, err := s.doGrokOfficialRequest(ctx, c, account, http.MethodPost, grokEndpointChatCompletions, mappedBody, withGrokConversationID(c))
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, s.handleHTTPError(ctx, resp, c, account, meta)
	}

	forwardResult, upstreamRequestID, err := s.handleAPIKeyOpenAIResponse(resp, c, grokOpenAIResponseOptions{
		OriginalModel: reqModel,
		MappedModel:   mappedModel,
		Stream:        stream,
		StartTime:     startTime,
	})
	if err != nil {
		return nil, err
	}
	forwardResult.UpstreamModel = mappedModel
	return &GrokGatewayForwardResult{
		Result:            forwardResult,
		RouteMode:         meta.RouteMode,
		Endpoint:          grokEndpointChatCompletions,
		EffectiveHost:     meta.EffectiveHost,
		EffectiveEndpoint: meta.EffectiveEndpoint,
		MediaType:         "",
		UpstreamRequestID: upstreamRequestID,
	}, nil
}

func (s *GrokGatewayService) forwardAPIKeyMessagesCompat(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardOfficialMessagesCompat(ctx, c, account, body)
}

func (s *GrokGatewayService) forwardOAuthBuildMessagesCompat(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardOfficialMessagesCompat(ctx, c, account, body)
}

func (s *GrokGatewayService) forwardOfficialMessagesCompat(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	responsesBody, originalModel, mappedModel, stream, err := buildGrokMessagesCompatResponsesBody(account, body)
	if err != nil {
		writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body", "")
		return nil, err
	}
	startTime := time.Now()
	resp, meta, err := s.doGrokOfficialRequest(ctx, c, account, http.MethodPost, grokEndpointResponses, responsesBody)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, s.handleHTTPError(ctx, resp, c, account, meta)
	}
	result, upstreamRequestID, err := s.handleAPIKeyAnthropicResponse(resp, c, grokAnthropicResponseOptions{
		OriginalModel: originalModel,
		MappedModel:   mappedModel,
		Stream:        stream,
		StartTime:     startTime,
	})
	if err != nil {
		return nil, err
	}
	return &GrokGatewayForwardResult{
		Result:            result,
		RouteMode:         meta.RouteMode,
		Endpoint:          grokEndpointResponses,
		EffectiveHost:     meta.EffectiveHost,
		EffectiveEndpoint: meta.EffectiveEndpoint,
		UpstreamRequestID: upstreamRequestID,
	}, nil
}

func (s *GrokGatewayService) forwardAPIKeyResponses(ctx context.Context, c *gin.Context, account *Account, body []byte, method string, subpath string) (*GrokGatewayForwardResult, error) {
	return s.forwardOfficialResponses(ctx, c, account, body, method, subpath)
}

func (s *GrokGatewayService) forwardOAuthBuildResponses(ctx context.Context, c *gin.Context, account *Account, body []byte, method string, subpath string) (*GrokGatewayForwardResult, error) {
	return s.forwardOfficialResponses(ctx, c, account, body, method, subpath)
}

func (s *GrokGatewayService) forwardOfficialResponses(ctx context.Context, c *gin.Context, account *Account, body []byte, method string, subpath string) (*GrokGatewayForwardResult, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	endpoint := grokEndpointResponses + normalizeResponsesSubpath(subpath)
	reqModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	mappedModel, mappedBody := grokApplyMappedModel(account, reqModel, body)
	if method == http.MethodPost && grokIsVideoRequestModel(reqModel, mappedModel) {
		return s.forwardGrokVideoResponses(ctx, c, account, body)
	}
	if method == http.MethodPost {
		mappedBody = sanitizeGrokOpenAICompatibleRequestBody(mappedBody)
	}
	stream := method == http.MethodPost && gjson.GetBytes(mappedBody, "stream").Bool()
	startTime := time.Now()

	resp, meta, err := s.doGrokOfficialRequest(ctx, c, account, method, endpoint, mappedBody)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, s.handleHTTPError(ctx, resp, c, account, meta)
	}

	if method != http.MethodPost {
		bodyBytes, readErr := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
		if readErr != nil {
			return nil, readErr
		}
		s.writeJSONResponse(c, resp, bodyBytes)
		return &GrokGatewayForwardResult{
			Result: &ForwardResult{
				RequestID:     grokUpstreamRequestID(resp.Header),
				Model:         reqModel,
				UpstreamModel: mappedModel,
				Stream:        false,
				Duration:      time.Since(startTime),
			},
			RouteMode:         meta.RouteMode,
			Endpoint:          grokEndpointResponses,
			EffectiveHost:     meta.EffectiveHost,
			EffectiveEndpoint: meta.EffectiveEndpoint,
			UpstreamRequestID: grokUpstreamRequestID(resp.Header),
			SkipUsageRecord:   true,
		}, nil
	}

	forwardResult, upstreamRequestID, err := s.handleAPIKeyOpenAIResponse(resp, c, grokOpenAIResponseOptions{
		OriginalModel: reqModel,
		MappedModel:   mappedModel,
		Stream:        stream,
		StartTime:     startTime,
	})
	if err != nil {
		return nil, err
	}
	forwardResult.UpstreamModel = mappedModel
	return &GrokGatewayForwardResult{
		Result:            forwardResult,
		RouteMode:         meta.RouteMode,
		Endpoint:          grokEndpointResponses,
		EffectiveHost:     meta.EffectiveHost,
		EffectiveEndpoint: meta.EffectiveEndpoint,
		UpstreamRequestID: upstreamRequestID,
	}, nil
}

func (s *GrokGatewayService) forwardAPIKeyAnthropicCountTokensCompat(ctx context.Context, c *gin.Context, account *Account, body []byte) (*AnthropicCountTokensBridgeResult, error) {
	return s.forwardOfficialAnthropicCountTokensCompat(ctx, c, account, body)
}

func (s *GrokGatewayService) forwardOAuthBuildAnthropicCountTokensCompat(ctx context.Context, c *gin.Context, account *Account, body []byte) (*AnthropicCountTokensBridgeResult, error) {
	return s.forwardOfficialAnthropicCountTokensCompat(ctx, c, account, body)
}

func (s *GrokGatewayService) forwardOfficialAnthropicCountTokensCompat(ctx context.Context, c *gin.Context, account *Account, body []byte) (*AnthropicCountTokensBridgeResult, error) {
	countBody, err := buildResponsesInputTokensBody(account, body, "")
	if err != nil {
		writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body", "")
		return nil, err
	}
	countBody = sanitizeOpenAIResponsesInputTokensBody(countBody)
	reqModel := strings.TrimSpace(gjson.GetBytes(countBody, "model").String())
	_, mappedBody := grokApplyMappedModel(account, reqModel, countBody)

	resp, _, err := s.doGrokOfficialRequest(ctx, c, account, http.MethodPost, grokEndpointResponses+openAIResponsesInputTokensPath, mappedBody)
	if err != nil {
		writeAnthropicError(c, http.StatusBadGateway, "upstream_error", "Request failed", "")
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	result, err := readResponsesInputTokensResult(resp, c, resolveUpstreamResponseReadLimit(s.cfg), func(status int, errType string, message string) {
		writeAnthropicError(c, status, errType, message, "")
	})
	if err != nil {
		return nil, err
	}
	c.JSON(http.StatusOK, result)
	return result, nil
}

func (s *GrokGatewayService) forwardAPIKeyImagesGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardOfficialImageRequest(ctx, c, account, body, grokEndpointImagesGen)
}

func (s *GrokGatewayService) forwardAPIKeyImagesEdits(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardOfficialImageRequest(ctx, c, account, body, grokEndpointImagesEdits)
}

func (s *GrokGatewayService) forwardOAuthBuildImagesGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardOfficialImageRequest(ctx, c, account, body, grokEndpointImagesGen)
}

func (s *GrokGatewayService) forwardOAuthBuildImagesEdits(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardOfficialImageRequest(ctx, c, account, body, grokEndpointImagesEdits)
}

func (s *GrokGatewayService) forwardOfficialImageRequest(ctx context.Context, c *gin.Context, account *Account, body []byte, endpoint string) (*GrokGatewayForwardResult, error) {
	reqModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	mappedModel, mappedBody := grokApplyMappedModel(account, reqModel, body)
	startTime := time.Now()

	resp, meta, err := s.doGrokOfficialRequest(ctx, c, account, http.MethodPost, endpoint, mappedBody)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, s.handleHTTPError(ctx, resp, c, account, meta)
	}

	bodyBytes, err := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if err != nil {
		return nil, err
	}
	s.writeJSONResponse(c, resp, bodyBytes)
	imageCount, imageURL := grokExtractImageResponse(bodyBytes)
	imageSize := strings.TrimSpace(gjson.GetBytes(mappedBody, "size").String())
	if imageCount == 0 {
		imageCount = 1
	}
	forwardResult := &ForwardResult{
		RequestID:     grokUpstreamRequestID(resp.Header),
		Model:         reqModel,
		UpstreamModel: mappedModel,
		Stream:        false,
		Duration:      time.Since(startTime),
		MediaType:     "image",
		ImageCount:    imageCount,
		ImageSize:     imageSize,
		MediaURL:      imageURL,
	}
	return &GrokGatewayForwardResult{
		Result:            forwardResult,
		RouteMode:         meta.RouteMode,
		Endpoint:          endpoint,
		EffectiveHost:     meta.EffectiveHost,
		EffectiveEndpoint: meta.EffectiveEndpoint,
		MediaType:         "image",
		UpstreamRequestID: grokUpstreamRequestID(resp.Header),
	}, nil
}

func (s *GrokGatewayService) forwardAPIKeyVideosGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardGrokVideoCreate(ctx, c, account, body)
}

func (s *GrokGatewayService) forwardOAuthBuildVideosGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardGrokVideoCreate(ctx, c, account, body)
}

func (s *GrokGatewayService) forwardAPIKeyVideosEdit(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardGrokVideoCreateWithOperation(ctx, c, account, body, grokVideoOperationEdit)
}

func (s *GrokGatewayService) forwardOAuthBuildVideosEdit(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardGrokVideoCreateWithOperation(ctx, c, account, body, grokVideoOperationEdit)
}

func (s *GrokGatewayService) forwardAPIKeyVideosExtension(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardGrokVideoCreateWithOperation(ctx, c, account, body, grokVideoOperationExtension)
}

func (s *GrokGatewayService) forwardOAuthBuildVideosExtension(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardGrokVideoCreateWithOperation(ctx, c, account, body, grokVideoOperationExtension)
}

func (s *GrokGatewayService) forwardAPIKeyVideoStatus(ctx context.Context, c *gin.Context, account *Account, requestID string) (*GrokGatewayForwardResult, error) {
	return s.forwardOfficialVideoStatus(ctx, c, account, requestID)
}

func (s *GrokGatewayService) forwardOAuthBuildVideoStatus(ctx context.Context, c *gin.Context, account *Account, requestID string) (*GrokGatewayForwardResult, error) {
	return s.forwardOfficialVideoStatus(ctx, c, account, requestID)
}

func (s *GrokGatewayService) forwardOfficialVideoStatus(ctx context.Context, c *gin.Context, account *Account, requestID string) (*GrokGatewayForwardResult, error) {
	requestID = strings.TrimSpace(requestID)
	startTime := time.Now()
	endpoint := "/v1/videos/" + requestID

	resp, meta, err := s.doGrokOfficialRequest(ctx, c, account, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, s.handleHTTPError(ctx, resp, c, account, meta)
	}

	bodyBytes, err := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if err != nil {
		return nil, err
	}
	videoResult := grokParseVideoResultBody(bodyBytes, "", "")
	if videoResult == nil {
		videoResult = &grokVideoResult{
			RequestID: requestID,
			Status:    "processing",
			Provider:  "grok",
		}
	}
	videoResult.RequestID = firstNonEmptyString(strings.TrimSpace(videoResult.RequestID), requestID)
	c.JSON(http.StatusOK, grokBuildVideoStatusResponse(videoResult))

	model := strings.TrimSpace(videoResult.Model)
	status := strings.ToLower(strings.TrimSpace(videoResult.Status))
	stableRequestID := "grok-video:" + requestID
	result := &GrokGatewayForwardResult{
		Result: &ForwardResult{
			RequestID:     stableRequestID,
			Model:         model,
			UpstreamModel: firstNonEmptyString(strings.TrimSpace(videoResult.UpstreamModel), model),
			Stream:        false,
			Duration:      time.Since(startTime),
			MediaType:     "video",
			MediaURL:      strings.TrimSpace(videoResult.URL),
		},
		RouteMode:         meta.RouteMode,
		Endpoint:          grokEndpointVideosStatus,
		EffectiveHost:     meta.EffectiveHost,
		EffectiveEndpoint: meta.EffectiveEndpoint,
		MediaType:         "video",
		UpstreamRequestID: requestID,
		SkipUsageRecord:   !grokIsTerminalVideoStatus(status),
	}
	if status == "failed" || status == "error" || status == "cancelled" {
		result.SkipUsageRecord = true
		result.FailedUsage = &GrokFailedUsageInfo{
			RequestID:     stableRequestID,
			Model:         model,
			UpstreamModel: model,
			ErrorCode:     status,
			ErrorMessage:  strings.TrimSpace(firstNonEmptyString(gjson.GetBytes(bodyBytes, "error.message").String(), gjson.GetBytes(bodyBytes, "message").String())),
			MediaType:     "video",
			Duration:      time.Since(startTime),
		}
	}
	return result, nil
}

type grokOpenAIResponseOptions struct {
	OriginalModel string
	MappedModel   string
	Stream        bool
	StartTime     time.Time
}

type grokAnthropicResponseOptions struct {
	OriginalModel string
	MappedModel   string
	Stream        bool
	StartTime     time.Time
}

func (s *GrokGatewayService) handleAPIKeyAnthropicResponse(resp *http.Response, c *gin.Context, opts grokAnthropicResponseOptions) (*ForwardResult, string, error) {
	upstreamRequestID := strings.TrimSpace(firstNonEmptyString(resp.Header.Get("x-request-id"), resp.Header.Get("X-Request-Id")))
	openAISvc := &OpenAIGatewayService{
		cfg:                  s.cfg,
		responseHeaderFilter: s.responseHeaderFilter,
	}
	if opts.Stream {
		result, err := openAISvc.handleAnthropicStreamingResponse(resp, c, opts.OriginalModel, opts.MappedModel, opts.StartTime)
		if result != nil && upstreamRequestID == "" {
			upstreamRequestID = strings.TrimSpace(result.RequestID)
		}
		if result == nil {
			return nil, upstreamRequestID, err
		}
		return openAIResultToForwardResult(result), upstreamRequestID, err
	}
	result, err := openAISvc.handleAnthropicBufferedStreamingResponse(resp, c, opts.OriginalModel, opts.MappedModel, opts.StartTime)
	if result != nil && upstreamRequestID == "" {
		upstreamRequestID = strings.TrimSpace(result.RequestID)
	}
	if result == nil {
		return nil, upstreamRequestID, err
	}
	return openAIResultToForwardResult(result), upstreamRequestID, err
}

func buildGrokMessagesCompatResponsesBody(account *Account, body []byte) ([]byte, string, string, bool, error) {
	var anthropicReq apicompat.AnthropicRequest
	if err := json.Unmarshal(body, &anthropicReq); err != nil {
		return nil, "", "", false, fmt.Errorf("parse grok messages compat request: %w", err)
	}
	originalModel := strings.TrimSpace(anthropicReq.Model)
	applyOpenAICompatModelNormalization(&anthropicReq)
	responsesReq, _, err := ConvertAnthropicMessagesToResponsesRuntime(&anthropicReq)
	if err != nil {
		return nil, "", "", false, fmt.Errorf("convert grok messages compat request: %w", err)
	}
	requestModel := strings.TrimSpace(responsesReq.Model)
	if requestModel == "" {
		requestModel = strings.TrimSpace(anthropicReq.Model)
	}
	mappedModel, _ := grokApplyMappedModel(account, requestModel, nil)
	if strings.TrimSpace(mappedModel) != "" {
		responsesReq.Model = mappedModel
	}
	clientStream := anthropicReq.Stream
	responsesReq.Stream = true
	responsesReq.Store = nil
	responsesReq.PromptCacheKey = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
	responsesBody, err := json.Marshal(responsesReq)
	if err != nil {
		return nil, "", "", false, fmt.Errorf("marshal grok messages compat request: %w", err)
	}
	responsesBody = sanitizeGrokOpenAICompatibleRequestBody(responsesBody)
	return responsesBody, originalModel, responsesReq.Model, clientStream, nil
}

func sanitizeGrokOpenAICompatibleRequestBody(body []byte) []byte {
	if len(body) == 0 {
		return body
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	for _, key := range []string{
		"previous_response_id",
		"safety_identifier",
		"service_tier",
		"metadata",
		"include",
	} {
		delete(payload, key)
	}
	next, err := json.Marshal(payload)
	if err != nil {
		return body
	}
	return next
}

func openAIResultToForwardResult(result *OpenAIForwardResult) *ForwardResult {
	if result == nil {
		return nil
	}
	return &ForwardResult{
		RequestID: result.RequestID,
		Usage: ClaudeUsage{
			InputTokens:              result.Usage.InputTokens,
			OutputTokens:             result.Usage.OutputTokens,
			CacheCreationInputTokens: result.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     result.Usage.CacheReadInputTokens,
		},
		Model:         result.Model,
		UpstreamModel: result.UpstreamModel,
		Stream:        result.Stream,
		Duration:      result.Duration,
		FirstTokenMs:  result.FirstTokenMs,
	}
}

func (s *GrokGatewayService) handleAPIKeyOpenAIResponse(resp *http.Response, c *gin.Context, opts grokOpenAIResponseOptions) (*ForwardResult, string, error) {
	upstreamRequestID := grokUpstreamRequestID(resp.Header)
	if opts.Stream {
		usage, firstTokenMs, requestID, err := s.proxyAPIKeyStream(resp, c, opts.StartTime)
		if err != nil {
			return nil, upstreamRequestID, err
		}
		if requestID == "" {
			requestID = upstreamRequestID
		}
		if upstreamRequestID == "" {
			upstreamRequestID = requestID
		}
		return &ForwardResult{
			RequestID:     requestID,
			Usage:         usage,
			Model:         opts.OriginalModel,
			UpstreamModel: opts.MappedModel,
			Stream:        true,
			Duration:      time.Since(opts.StartTime),
			FirstTokenMs:  firstTokenMs,
		}, upstreamRequestID, nil
	}

	bodyBytes, err := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if err != nil {
		return nil, upstreamRequestID, err
	}
	s.writeJSONResponse(c, resp, bodyBytes)
	requestID := strings.TrimSpace(firstNonEmptyString(
		upstreamRequestID,
		gjson.GetBytes(bodyBytes, "id").String(),
		gjson.GetBytes(bodyBytes, "response.id").String(),
	))
	if upstreamRequestID == "" {
		upstreamRequestID = requestID
	}
	return &ForwardResult{
		RequestID:     requestID,
		Usage:         grokExtractUsageFromJSON(bodyBytes),
		Model:         opts.OriginalModel,
		UpstreamModel: opts.MappedModel,
		Stream:        false,
		Duration:      time.Since(opts.StartTime),
	}, upstreamRequestID, nil
}

func (s *GrokGatewayService) proxyAPIKeyStream(resp *http.Response, c *gin.Context, startTime time.Time) (ClaudeUsage, *int, string, error) {
	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return ClaudeUsage{}, nil, "", fmt.Errorf("streaming not supported")
	}

	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)

	var usage ClaudeUsage
	var firstTokenMs *int
	requestID := grokUpstreamRequestID(resp.Header)
	for scanner.Scan() {
		line := scanner.Text()
		if data, ok := extractOpenAISSEDataLine(line); ok {
			trimmed := strings.TrimSpace(data)
			if firstTokenMs == nil && trimmed != "" && trimmed != "[DONE]" {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}
			grokParseSSEUsage([]byte(trimmed), &usage)
			if requestID == "" {
				requestID = strings.TrimSpace(firstNonEmptyString(
					gjson.Get(trimmed, "id").String(),
					gjson.Get(trimmed, "response.id").String(),
				))
			}
		}
		if _, err := io.WriteString(c.Writer, line+"\n"); err != nil {
			return usage, firstTokenMs, requestID, nil
		}
		if line == "" {
			flusher.Flush()
		}
	}
	if err := scanner.Err(); err != nil {
		return usage, firstTokenMs, requestID, err
	}
	flusher.Flush()
	return usage, firstTokenMs, requestID, nil
}

type grokAPIKeyRequestOption func(*http.Request)

func withGrokConversationID(c *gin.Context) grokAPIKeyRequestOption {
	return func(req *http.Request) {
		if req == nil || c == nil || c.Request == nil {
			return
		}
		if conversationID := strings.TrimSpace(c.GetHeader("x-grok-conv-id")); conversationID != "" {
			req.Header.Set("x-grok-conv-id", conversationID)
		}
	}
}

func (s *GrokGatewayService) doGrokOfficialRequest(ctx context.Context, c *gin.Context, account *Account, method string, endpoint string, body []byte, opts ...grokAPIKeyRequestOption) (*http.Response, grokUpstreamRequestMetadata, error) {
	switch {
	case account != nil && account.IsGrokOAuth():
		return s.doOAuthBuildRequest(ctx, c, account, method, endpoint, body, opts...)
	default:
		return s.doAPIKeyRequest(ctx, c, account, method, endpoint, body, opts...)
	}
}

func (s *GrokGatewayService) doAPIKeyRequest(ctx context.Context, c *gin.Context, account *Account, method string, endpoint string, body []byte, opts ...grokAPIKeyRequestOption) (*http.Response, grokUpstreamRequestMetadata, error) {
	meta := grokUpstreamRequestMetadata{RouteMode: GrokRouteModeAPIKey, EffectiveEndpoint: strings.TrimSpace(endpoint)}
	if account == nil || !account.IsGrokAPIKey() {
		return nil, meta, fmt.Errorf("grok api key account is required")
	}
	token := strings.TrimSpace(account.GetGrokAPIKey())
	if token == "" {
		return nil, meta, fmt.Errorf("grok api key token is missing")
	}
	baseURL := account.GetBaseURL()
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultGrokAPIBaseURL
	}
	baseURL, err := s.validatedBaseURL(baseURL, defaultGrokAPIBaseURL)
	if err != nil {
		return nil, meta, err
	}
	return s.doGrokOfficialBearerRequest(ctx, c, account, method, endpoint, body, baseURL, token, GrokRouteModeAPIKey, false, opts...)
}

func (s *GrokGatewayService) doOAuthBuildRequest(ctx context.Context, c *gin.Context, account *Account, method string, endpoint string, body []byte, opts ...grokAPIKeyRequestOption) (*http.Response, grokUpstreamRequestMetadata, error) {
	meta := grokUpstreamRequestMetadata{RouteMode: GrokRouteModeOAuthBuild, EffectiveEndpoint: strings.TrimSpace(endpoint)}
	if account == nil || !account.IsGrokOAuth() {
		return nil, meta, fmt.Errorf("grok oauth build account is required")
	}
	token := ""
	if s.tokenProvider != nil {
		refreshedToken, err := s.tokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return nil, meta, s.handleGrokCredentialFailure(ctx, c, account, err)
		}
		token = strings.TrimSpace(refreshedToken)
	} else {
		token = strings.TrimSpace(account.GetGrokOAuthAccessToken())
	}
	if token == "" {
		return nil, meta, s.handleGrokCredentialFailure(ctx, c, account, errGrokOAuthAccessTokenMissing)
	}
	return s.doGrokOfficialBearerRequest(ctx, c, account, method, endpoint, body, defaultGrokCLIBaseURL, token, GrokRouteModeOAuthBuild, true, opts...)
}

func (s *GrokGatewayService) doGrokOfficialBearerRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	method string,
	endpoint string,
	body []byte,
	baseURL string,
	token string,
	routeMode string,
	cliHeaders bool,
	opts ...grokAPIKeyRequestOption,
) (*http.Response, grokUpstreamRequestMetadata, error) {
	meta := grokUpstreamRequestMetadata{RouteMode: strings.TrimSpace(routeMode), EffectiveEndpoint: strings.TrimSpace(endpoint)}
	url, err := grokJoinVersionedEndpoint(baseURL, endpoint)
	if err != nil {
		return nil, meta, err
	}
	meta = newGrokUpstreamRequestMetadata(account, url, endpoint)
	if strings.TrimSpace(routeMode) != "" {
		meta.RouteMode = strings.TrimSpace(routeMode)
	}

	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, meta, err
	}
	if c != nil && c.Request != nil {
		contentType := strings.TrimSpace(c.GetHeader("Content-Type"))
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
	}
	if req.Header.Get("Content-Type") == "" && len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if cliHeaders {
		applyGrokCLIHeaders(req.Header)
	} else {
		ApplyAccountRequestHeaderOverrides(req, account)
	}
	for _, opt := range opts {
		if opt != nil {
			opt(req)
		}
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           PlatformGrok,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		logger.FromContext(ctx).Warn("grok.upstream_request_error", append(grokUpstreamLogFields(account, meta, 0, ""), zap.String("error", safeErr))...)
		return nil, meta, fmt.Errorf("grok upstream request failed: %s", safeErr)
	}
	if resp != nil {
		s.observeGrokQuotaHeaders(ctx, account, resp.Header, resp.StatusCode, "gateway_response")
		logger.FromContext(ctx).Debug("grok.upstream_response", grokUpstreamLogFields(account, meta, resp.StatusCode, grokUpstreamRequestID(resp.Header))...)
	}
	return resp, meta, nil
}

func grokApplyMappedModel(account *Account, requestedModel string, body []byte) (string, []byte) {
	mappedModel := strings.TrimSpace(requestedModel)
	if account != nil && requestedModel != "" {
		mappedModel = account.GetMappedModel(requestedModel)
		if mappedModel == requestedModel && (account.IsGrokAPIKey() || account.IsGrokOAuth()) {
			if resolved := GrokAPIKeyResolvedUpstreamModel(requestedModel); resolved != "" {
				mappedModel = resolved
			}
		}
	}
	if requestedModel == "" || mappedModel == "" || mappedModel == requestedModel || len(body) == 0 {
		return mappedModel, body
	}
	nextBody, err := sjson.SetBytes(body, "model", mappedModel)
	if err != nil {
		return mappedModel, body
	}
	return mappedModel, nextBody
}

func grokExtractUsageFromJSON(body []byte) ClaudeUsage {
	return ClaudeUsage{
		InputTokens:              int(firstPositiveInt64(gjson.GetBytes(body, "usage.input_tokens").Int(), gjson.GetBytes(body, "usage.prompt_tokens").Int())),
		OutputTokens:             int(firstPositiveInt64(gjson.GetBytes(body, "usage.output_tokens").Int(), gjson.GetBytes(body, "usage.completion_tokens").Int())),
		CacheReadInputTokens:     int(grokCachedTokensFromUsage(body, "usage")),
		CacheCreationInputTokens: 0,
	}
}

func grokParseSSEUsage(data []byte, usage *ClaudeUsage) {
	if usage == nil || len(data) == 0 || bytes.Equal(data, []byte("[DONE]")) {
		return
	}
	if eventType := gjson.GetBytes(data, "type").String(); eventType == "response.completed" || eventType == "response.done" {
		usage.InputTokens = int(firstPositiveInt64(gjson.GetBytes(data, "response.usage.input_tokens").Int(), int64(usage.InputTokens)))
		usage.OutputTokens = int(firstPositiveInt64(gjson.GetBytes(data, "response.usage.output_tokens").Int(), int64(usage.OutputTokens)))
		usage.CacheReadInputTokens = int(firstPositiveInt64(grokCachedTokensFromUsage(data, "response.usage"), int64(usage.CacheReadInputTokens)))
		return
	}
	if gjson.GetBytes(data, "usage").Exists() {
		usage.InputTokens = int(firstPositiveInt64(gjson.GetBytes(data, "usage.input_tokens").Int(), gjson.GetBytes(data, "usage.prompt_tokens").Int(), int64(usage.InputTokens)))
		usage.OutputTokens = int(firstPositiveInt64(gjson.GetBytes(data, "usage.output_tokens").Int(), gjson.GetBytes(data, "usage.completion_tokens").Int(), int64(usage.OutputTokens)))
		usage.CacheReadInputTokens = int(firstPositiveInt64(grokCachedTokensFromUsage(data, "usage"), int64(usage.CacheReadInputTokens)))
	}
}

func grokCachedTokensFromUsage(body []byte, usagePath string) int64 {
	usagePath = strings.TrimSpace(usagePath)
	if usagePath == "" {
		return 0
	}
	return firstPositiveInt64(
		gjson.GetBytes(body, usagePath+".input_tokens_details.cached_tokens").Int(),
		gjson.GetBytes(body, usagePath+".prompt_tokens_details.cached_tokens").Int(),
		gjson.GetBytes(body, usagePath+".cached_tokens").Int(),
	)
}

func grokExtractImageResponse(body []byte) (int, string) {
	data := gjson.GetBytes(body, "data")
	if data.IsArray() {
		arr := data.Array()
		firstURL := ""
		if len(arr) > 0 {
			firstURL = strings.TrimSpace(firstNonEmptyString(arr[0].Get("url").String(), arr[0].Get("b64_json").String()))
		}
		return len(arr), firstURL
	}
	return 0, ""
}

func grokIsTerminalVideoStatus(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "completed", "done", "succeeded", "failed", "error", "cancelled", "canceled", "expired":
		return true
	default:
		return false
	}
}

func normalizeResponsesSubpath(subpath string) string {
	subpath = strings.TrimSpace(subpath)
	if subpath == "" || subpath == "/" {
		return ""
	}
	if !strings.HasPrefix(subpath, "/") {
		return "/" + subpath
	}
	return subpath
}

func firstPositiveInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
