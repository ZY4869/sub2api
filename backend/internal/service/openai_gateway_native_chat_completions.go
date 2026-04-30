package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
)

func (s *OpenAIGatewayService) ForwardNativeChatCompletions(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	account = ResolveProtocolGatewayInboundAccount(account, PlatformOpenAI)
	if RoutingPlatformForAccount(account) == PlatformDeepSeek {
		return s.forwardDeepSeekNativeChatCompletions(ctx, c, account, body, defaultMappedModel)
	}

	startTime := time.Now()

	var chatReq apicompat.ChatCompletionsRequest
	if err := json.Unmarshal(body, &chatReq); err != nil {
		return nil, fmt.Errorf("parse native chat completions request: %w", err)
	}

	originalModel := strings.TrimSpace(chatReq.Model)
	clientRequestedUsage := chatReq.StreamOptions != nil && chatReq.StreamOptions.IncludeUsage
	mappedModel := resolveOpenAIForwardModel(account, originalModel, defaultMappedModel)
	chatReq.Model = mappedModel
	if chatReq.Stream && !clientRequestedUsage {
		if chatReq.StreamOptions == nil {
			chatReq.StreamOptions = &apicompat.ChatStreamOptions{}
		}
		chatReq.StreamOptions.IncludeUsage = true
	}

	requestBody, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshal native chat completions request: %w", err)
	}

	ctx = WithOpenAICodexRequestModel(ctx, mappedModel)
	if c != nil && c.Request != nil {
		c.Request = c.Request.WithContext(ctx)
	}
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	upstreamReq, err := s.buildNativeChatCompletionsUpstreamRequest(ctx, c, account, requestBody, token, chatReq.Stream)
	if err != nil {
		return nil, fmt.Errorf("build native chat completions request: %w", err)
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		writeChatCompletionsError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", "")
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(strings.NewReader(string(respBody)))

		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			upstreamDetail := ""
			if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
				maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
				if maxBytes <= 0 {
					maxBytes = 2048
				}
				upstreamDetail = truncateString(string(respBody), maxBytes)
			}
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Kind:               "failover",
				Message:            upstreamMsg,
				Detail:             upstreamDetail,
			})
			if s.rateLimitService != nil {
				s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			}
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: account.IsPoolMode() && (isPoolModeRetryableStatus(resp.StatusCode) || isOpenAITransientProcessingError(resp.StatusCode, upstreamMsg, respBody)),
			}
		}
		return s.handleChatCompletionsErrorResponse(resp, c, account)
	}

	var usage *OpenAIUsage
	var firstTokenMs *int
	if chatReq.Stream {
		streamResult, streamErr := s.handleNativeChatCompletionsStreamingResponse(
			ctx,
			resp,
			c,
			account,
			startTime,
			originalModel,
			mappedModel,
			clientRequestedUsage,
		)
		if streamErr != nil {
			return nil, streamErr
		}
		usage = streamResult.usage
		firstTokenMs = streamResult.firstTokenMs
	} else {
		usage, err = s.handleNativeChatCompletionsNonStreamingResponse(resp, c, originalModel, mappedModel)
		if err != nil {
			return nil, err
		}
	}
	if usage == nil {
		usage = &OpenAIUsage{}
	}

	return &OpenAIForwardResult{
		RequestID:       resp.Header.Get("x-request-id"),
		Usage:           *usage,
		Model:           originalModel,
		BillingModel:    mappedModel,
		UpstreamModel:   mappedModel,
		ServiceTier:     extractOpenAIServiceTierFromBody(requestBody),
		ReasoningEffort: extractOpenAIReasoningEffortFromBody(requestBody, originalModel),
		Stream:          chatReq.Stream,
		Duration:        time.Since(startTime),
		FirstTokenMs:    firstTokenMs,
	}, nil
}

func (s *OpenAIGatewayService) buildNativeChatCompletionsUpstreamRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	token string,
	isStream bool,
) (*http.Request, error) {
	targetURL, err := resolveOpenAIChatCompletionsTargetURL(account, s.validateUpstreamBaseURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("authorization", "Bearer "+token)
	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			if !openaiAllowedHeaders[strings.ToLower(key)] {
				continue
			}
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}
	if req.Header.Get("accept") == "" {
		if isStream {
			req.Header.Set("accept", "text/event-stream")
		} else {
			req.Header.Set("accept", "application/json")
		}
	}
	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		req.Header.Set("user-agent", customUA)
	}
	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}
	return req, nil
}

func (s *OpenAIGatewayService) handleNativeChatCompletionsStreamingResponse(
	ctx context.Context,
	resp *http.Response,
	c *gin.Context,
	account *Account,
	startTime time.Time,
	originalModel string,
	mappedModel string,
	clientRequestedUsage bool,
) (*openaiStreamingResult, error) {
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

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}

	usage := &OpenAIUsage{}
	var firstTokenMs *int
	clientDisconnected := false
	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	scanner.Buffer(scanBuf[:0], maxLineSize)
	defer putSSEScannerBuf64K(scanBuf)

	for scanner.Scan() {
		line := scanner.Text()
		if data, ok := extractOpenAISSEDataLine(line); ok {
			if mappedModel != "" && mappedModel != originalModel && strings.Contains(data, mappedModel) {
				line = s.replaceModelInSSELine(line, mappedModel, originalModel)
				data, _ = extractOpenAISSEDataLine(line)
			}

			dataBytes := []byte(data)
			if correctedData, corrected := s.toolCorrector.CorrectToolCallsInSSEBytes(dataBytes); corrected {
				dataBytes = correctedData
				line = "data: " + string(correctedData)
				data = string(correctedData)
			}

			parseOpenAIChatCompletionsSSEUsage(dataBytes, usage)
			if !clientRequestedUsage && isOpenAIChatCompletionsUsageOnlyChunk(dataBytes) {
				continue
			}
			if firstTokenMs == nil && strings.TrimSpace(data) != "" && strings.TrimSpace(data) != "[DONE]" && !isOpenAIChatCompletionsUsageOnlyChunk(dataBytes) {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}
		}

		if clientDisconnected {
			continue
		}
		if _, err := fmt.Fprintln(c.Writer, line); err != nil {
			clientDisconnected = true
			logger.LegacyPrintf("service.openai_gateway", "Client disconnected during native chat streaming, continuing to drain upstream for billing")
			continue
		}
		flusher.Flush()
	}

	if err := scanner.Err(); err != nil {
		if clientDisconnected || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, nil
		}
		return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, fmt.Errorf("native chat stream read error: %w", err)
	}

	return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, nil
}

func (s *OpenAIGatewayService) handleNativeChatCompletionsNonStreamingResponse(
	resp *http.Response,
	c *gin.Context,
	originalModel string,
	mappedModel string,
) (*OpenAIUsage, error) {
	maxBytes := resolveUpstreamResponseReadLimit(s.cfg)
	body, err := readUpstreamResponseBodyLimited(resp.Body, maxBytes)
	if err != nil {
		return nil, err
	}

	usageValue, ok := extractOpenAIChatCompletionsUsageFromJSONBytes(body)
	if !ok && !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("parse native chat completions response: invalid json response")
	}
	if mappedModel != "" && mappedModel != originalModel {
		body = s.replaceModelInResponseBody(body, mappedModel, originalModel)
	}
	body = s.correctToolCallsInResponseBody(body)

	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := "application/json"
	if s.cfg != nil && !s.cfg.Security.ResponseHeaders.Enabled {
		if upstreamType := resp.Header.Get("Content-Type"); upstreamType != "" {
			contentType = upstreamType
		}
	}
	c.Data(resp.StatusCode, contentType, body)

	usage := usageValue
	return &usage, nil
}

func extractOpenAIChatCompletionsUsageFromJSONBytes(body []byte) (OpenAIUsage, bool) {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return OpenAIUsage{}, false
	}
	return OpenAIUsage{
		InputTokens:          int(firstPositiveInt64(gjson.GetBytes(body, "usage.input_tokens").Int(), gjson.GetBytes(body, "usage.prompt_tokens").Int())),
		OutputTokens:         int(firstPositiveInt64(gjson.GetBytes(body, "usage.output_tokens").Int(), gjson.GetBytes(body, "usage.completion_tokens").Int())),
		CacheReadInputTokens: int(firstPositiveInt64(gjson.GetBytes(body, "usage.input_tokens_details.cached_tokens").Int(), gjson.GetBytes(body, "usage.prompt_tokens_details.cached_tokens").Int())),
	}, true
}

func parseOpenAIChatCompletionsSSEUsage(data []byte, usage *OpenAIUsage) {
	if usage == nil || len(data) == 0 || string(data) == "[DONE]" {
		return
	}
	if !gjson.GetBytes(data, "usage").Exists() {
		return
	}
	usage.InputTokens = int(firstPositiveInt64(gjson.GetBytes(data, "usage.input_tokens").Int(), gjson.GetBytes(data, "usage.prompt_tokens").Int(), int64(usage.InputTokens)))
	usage.OutputTokens = int(firstPositiveInt64(gjson.GetBytes(data, "usage.output_tokens").Int(), gjson.GetBytes(data, "usage.completion_tokens").Int(), int64(usage.OutputTokens)))
	usage.CacheReadInputTokens = int(firstPositiveInt64(gjson.GetBytes(data, "usage.input_tokens_details.cached_tokens").Int(), gjson.GetBytes(data, "usage.prompt_tokens_details.cached_tokens").Int(), int64(usage.CacheReadInputTokens)))
}

func isOpenAIChatCompletionsUsageOnlyChunk(data []byte) bool {
	if len(data) == 0 || !gjson.GetBytes(data, "usage").Exists() {
		return false
	}
	choices := gjson.GetBytes(data, "choices")
	return !choices.Exists() || len(choices.Array()) == 0
}
