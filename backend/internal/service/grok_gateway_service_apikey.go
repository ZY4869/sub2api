package service

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func (s *GrokGatewayService) forwardAPIKeyChatCompletions(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	reqModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	mappedModel, mappedBody := grokApplyMappedModel(account, reqModel, body)
	stream := gjson.GetBytes(mappedBody, "stream").Bool()
	startTime := time.Now()

	resp, err := s.doAPIKeyRequest(ctx, c, account, http.MethodPost, grokEndpointChatCompletions, mappedBody)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, s.handleHTTPError(ctx, resp, c, account, GrokRouteModeAPIKey)
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
		RouteMode:         GrokRouteModeAPIKey,
		Endpoint:          grokEndpointChatCompletions,
		MediaType:         "",
		UpstreamRequestID: upstreamRequestID,
	}, nil
}

func (s *GrokGatewayService) forwardAPIKeyResponses(ctx context.Context, c *gin.Context, account *Account, body []byte, method string, subpath string) (*GrokGatewayForwardResult, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	endpoint := grokEndpointResponses + normalizeResponsesSubpath(subpath)
	reqModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	mappedModel, mappedBody := grokApplyMappedModel(account, reqModel, body)
	stream := method == http.MethodPost && gjson.GetBytes(mappedBody, "stream").Bool()
	startTime := time.Now()

	resp, err := s.doAPIKeyRequest(ctx, c, account, method, endpoint, mappedBody)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, s.handleHTTPError(ctx, resp, c, account, GrokRouteModeAPIKey)
	}

	if method != http.MethodPost {
		bodyBytes, readErr := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
		if readErr != nil {
			return nil, readErr
		}
		s.writeJSONResponse(c, resp, bodyBytes)
		return &GrokGatewayForwardResult{
			Result: &ForwardResult{
				RequestID:     resp.Header.Get("x-request-id"),
				Model:         reqModel,
				UpstreamModel: mappedModel,
				Stream:        false,
				Duration:      time.Since(startTime),
			},
			RouteMode:         GrokRouteModeAPIKey,
			Endpoint:          grokEndpointResponses,
			UpstreamRequestID: resp.Header.Get("x-request-id"),
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
		RouteMode:         GrokRouteModeAPIKey,
		Endpoint:          grokEndpointResponses,
		UpstreamRequestID: upstreamRequestID,
	}, nil
}

func (s *GrokGatewayService) forwardAPIKeyImagesGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardAPIKeyImageRequest(ctx, c, account, body, grokEndpointImagesGen)
}

func (s *GrokGatewayService) forwardAPIKeyImagesEdits(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	return s.forwardAPIKeyImageRequest(ctx, c, account, body, grokEndpointImagesEdits)
}

func (s *GrokGatewayService) forwardAPIKeyImageRequest(ctx context.Context, c *gin.Context, account *Account, body []byte, endpoint string) (*GrokGatewayForwardResult, error) {
	reqModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	mappedModel, mappedBody := grokApplyMappedModel(account, reqModel, body)
	startTime := time.Now()

	resp, err := s.doAPIKeyRequest(ctx, c, account, http.MethodPost, endpoint, mappedBody)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, s.handleHTTPError(ctx, resp, c, account, GrokRouteModeAPIKey)
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
		RequestID:     resp.Header.Get("x-request-id"),
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
		RouteMode:         GrokRouteModeAPIKey,
		Endpoint:          endpoint,
		MediaType:         "image",
		UpstreamRequestID: resp.Header.Get("x-request-id"),
	}, nil
}

func (s *GrokGatewayService) forwardAPIKeyVideosGeneration(ctx context.Context, c *gin.Context, account *Account, body []byte) (*GrokGatewayForwardResult, error) {
	reqModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	mappedModel, mappedBody := grokApplyMappedModel(account, reqModel, body)
	startTime := time.Now()
	endpoint := grokAPIKeyVideoGenerationEndpoint(mappedBody)

	resp, err := s.doAPIKeyRequest(ctx, c, account, http.MethodPost, endpoint, mappedBody)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, s.handleHTTPError(ctx, resp, c, account, GrokRouteModeAPIKey)
	}

	bodyBytes, err := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if err != nil {
		return nil, err
	}
	s.writeJSONResponse(c, resp, bodyBytes)
	requestID := strings.TrimSpace(gjson.GetBytes(bodyBytes, "request_id").String())
	if requestID == "" {
		requestID = strings.TrimSpace(gjson.GetBytes(bodyBytes, "id").String())
	}
	return &GrokGatewayForwardResult{
		Result: &ForwardResult{
			RequestID:     requestID,
			Model:         reqModel,
			UpstreamModel: mappedModel,
			Stream:        false,
			Duration:      time.Since(startTime),
			MediaType:     "video",
		},
		RouteMode:         GrokRouteModeAPIKey,
		Endpoint:          endpoint,
		MediaType:         "video",
		UpstreamRequestID: requestID,
		SkipUsageRecord:   true,
	}, nil
}

func grokAPIKeyVideoGenerationEndpoint(body []byte) string {
	if grokVideoEditRequested(body) {
		return "/v1/videos/edits"
	}
	return grokEndpointVideosGen
}

func grokVideoEditRequested(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	return strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(body, "video").String(),
		gjson.GetBytes(body, "video.url").String(),
		gjson.GetBytes(body, "video_url").String(),
		gjson.GetBytes(body, "source_video").String(),
		gjson.GetBytes(body, "source_video.url").String(),
		gjson.GetBytes(body, "source_video_url").String(),
		gjson.GetBytes(body, "input_video").String(),
		gjson.GetBytes(body, "input_video.url").String(),
	)) != ""
}

func (s *GrokGatewayService) forwardAPIKeyVideoStatus(ctx context.Context, c *gin.Context, account *Account, requestID string) (*GrokGatewayForwardResult, error) {
	requestID = strings.TrimSpace(requestID)
	startTime := time.Now()
	endpoint := "/v1/videos/" + requestID

	resp, err := s.doAPIKeyRequest(ctx, c, account, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil, s.handleHTTPError(ctx, resp, c, account, GrokRouteModeAPIKey)
	}

	bodyBytes, err := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if err != nil {
		return nil, err
	}
	s.writeJSONResponse(c, resp, bodyBytes)

	model := strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(bodyBytes, "model").String(),
		gjson.GetBytes(bodyBytes, "data.model").String(),
	))
	status := strings.ToLower(strings.TrimSpace(firstNonEmptyString(
		gjson.GetBytes(bodyBytes, "status").String(),
		gjson.GetBytes(bodyBytes, "data.status").String(),
	)))
	stableRequestID := "grok-video:" + requestID
	result := &GrokGatewayForwardResult{
		Result: &ForwardResult{
			RequestID:     stableRequestID,
			Model:         model,
			UpstreamModel: model,
			Stream:        false,
			Duration:      time.Since(startTime),
			MediaType:     "video",
		},
		RouteMode:         GrokRouteModeAPIKey,
		Endpoint:          grokEndpointVideosStatus,
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

func (s *GrokGatewayService) handleAPIKeyOpenAIResponse(resp *http.Response, c *gin.Context, opts grokOpenAIResponseOptions) (*ForwardResult, string, error) {
	upstreamRequestID := strings.TrimSpace(firstNonEmptyString(resp.Header.Get("x-request-id"), resp.Header.Get("X-Request-Id")))
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
	requestID := strings.TrimSpace(resp.Header.Get("x-request-id"))
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

func (s *GrokGatewayService) doAPIKeyRequest(ctx context.Context, c *gin.Context, account *Account, method string, endpoint string, body []byte) (*http.Response, error) {
	if account == nil || !account.IsGrokAPIKey() {
		return nil, fmt.Errorf("grok apikey account is required")
	}
	token := strings.TrimSpace(account.GetGrokAPIKey())
	if token == "" {
		return nil, fmt.Errorf("grok api key is missing")
	}
	baseURL, err := s.validatedBaseURL(account.GetBaseURL(), defaultGrokAPIBaseURL)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(baseURL, "/") + endpoint

	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("grok upstream request failed: %s", safeErr)
	}
	return resp, nil
}

func grokApplyMappedModel(account *Account, requestedModel string, body []byte) (string, []byte) {
	mappedModel := strings.TrimSpace(requestedModel)
	if account != nil && requestedModel != "" {
		mappedModel = account.GetMappedModel(requestedModel)
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
		CacheReadInputTokens:     int(firstPositiveInt64(gjson.GetBytes(body, "usage.input_tokens_details.cached_tokens").Int(), gjson.GetBytes(body, "usage.prompt_tokens_details.cached_tokens").Int())),
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
		usage.CacheReadInputTokens = int(firstPositiveInt64(gjson.GetBytes(data, "response.usage.input_tokens_details.cached_tokens").Int(), int64(usage.CacheReadInputTokens)))
		return
	}
	if gjson.GetBytes(data, "usage").Exists() {
		usage.InputTokens = int(firstPositiveInt64(gjson.GetBytes(data, "usage.input_tokens").Int(), gjson.GetBytes(data, "usage.prompt_tokens").Int(), int64(usage.InputTokens)))
		usage.OutputTokens = int(firstPositiveInt64(gjson.GetBytes(data, "usage.output_tokens").Int(), gjson.GetBytes(data, "usage.completion_tokens").Int(), int64(usage.OutputTokens)))
		usage.CacheReadInputTokens = int(firstPositiveInt64(gjson.GetBytes(data, "usage.input_tokens_details.cached_tokens").Int(), gjson.GetBytes(data, "usage.prompt_tokens_details.cached_tokens").Int(), int64(usage.CacheReadInputTokens)))
	}
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
	case "completed", "succeeded", "failed", "error", "cancelled":
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
