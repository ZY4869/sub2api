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
	"go.uber.org/zap"
)

func (s *OpenAIGatewayService) ForwardResponsesAsChatCompletions(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	var responsesReq apicompat.ResponsesRequest
	if err := json.Unmarshal(body, &responsesReq); err != nil {
		return nil, fmt.Errorf("parse responses request: %w", err)
	}
	chatReq, err := apicompat.ResponsesToChatCompletionsRequest(&responsesReq)
	if err != nil {
		return nil, fmt.Errorf("convert responses to chat completions: %w", err)
	}
	chatBody, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshal chat completions request: %w", err)
	}
	return s.forwardResponsesAsNativeChatCompletions(ctx, c, account, chatBody, responsesReq.Model, defaultMappedModel)
}

func (s *OpenAIGatewayService) forwardResponsesAsNativeChatCompletions(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	originalResponsesModel string,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	var chatReq apicompat.ChatCompletionsRequest
	if err := json.Unmarshal(body, &chatReq); err != nil {
		return nil, fmt.Errorf("parse native chat completions request: %w", err)
	}
	originalModel := strings.TrimSpace(originalResponsesModel)
	if originalModel == "" {
		originalModel = strings.TrimSpace(chatReq.Model)
	}
	mappedModel := resolveOpenAIForwardModel(account, chatReq.Model, defaultMappedModel)
	chatReq.Model = mappedModel
	if chatReq.Stream && chatReq.StreamOptions == nil {
		chatReq.StreamOptions = &apicompat.ChatStreamOptions{IncludeUsage: true}
	} else if chatReq.Stream {
		chatReq.StreamOptions.IncludeUsage = true
	}
	requestBody, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshal native chat completions request: %w", err)
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
	if account != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamError(c, 0, safeErr, "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: 0, Kind: "request_error", Message: safeErr})
		writeResponsesBridgeError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", "")
		return nil, fmt.Errorf("upstream request failed: %s", safeErr)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "failover", Message: upstreamMsg})
			if s.rateLimitService != nil {
				s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			}
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, RetryableOnSameAccount: account.IsPoolMode() && (isPoolModeRetryableStatus(resp.StatusCode) || isOpenAITransientProcessingError(resp.StatusCode, upstreamMsg, respBody))}
		}
		return s.handleCompatErrorResponse(resp, c, account, writeResponsesBridgeError)
	}

	var usage *OpenAIUsage
	var firstTokenMs *int
	if chatReq.Stream {
		streamResult, streamErr := s.handleNativeChatAsResponsesStreamingResponse(resp, c, startTime, originalModel, mappedModel)
		if streamErr != nil {
			return nil, streamErr
		}
		usage = streamResult.usage
		firstTokenMs = streamResult.firstTokenMs
	} else {
		usage, err = s.handleNativeChatAsResponsesNonStreamingResponse(resp, c, originalModel, mappedModel)
		if err != nil {
			return nil, err
		}
	}
	if usage == nil {
		usage = &OpenAIUsage{}
	}
	return &OpenAIForwardResult{
		RequestID:     resp.Header.Get("x-request-id"),
		Usage:         *usage,
		Model:         originalModel,
		BillingModel:  mappedModel,
		UpstreamModel: mappedModel,
		Stream:        chatReq.Stream,
		Duration:      time.Since(startTime),
		FirstTokenMs:  firstTokenMs,
	}, nil
}

func (s *OpenAIGatewayService) handleNativeChatAsResponsesNonStreamingResponse(
	resp *http.Response,
	c *gin.Context,
	originalModel string,
	mappedModel string,
) (*OpenAIUsage, error) {
	body, err := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if err != nil {
		return nil, err
	}
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("parse native chat completions response: invalid json response")
	}
	var chatResp apicompat.ChatCompletionsResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("parse native chat completions response: %w", err)
	}
	if mappedModel != "" && chatResp.Model == mappedModel {
		chatResp.Model = originalModel
	}
	responsesResp := apicompat.ChatCompletionsToResponsesResponse(&chatResp, originalModel)
	responseBody, err := json.Marshal(responsesResp)
	if err != nil {
		return nil, fmt.Errorf("marshal responses bridge response: %w", err)
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	c.Data(resp.StatusCode, "application/json", responseBody)
	usage := openAIUsageFromResponsesCompatUsage(responsesResp.Usage)
	return &usage, nil
}

func (s *OpenAIGatewayService) handleNativeChatAsResponsesStreamingResponse(
	resp *http.Response,
	c *gin.Context,
	startTime time.Time,
	originalModel string,
	mappedModel string,
) (*openaiStreamingResult, error) {
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
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
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), defaultMaxLineSize)
	state := &apicompat.ChatToResponsesStreamState{Model: originalModel}
	usage := &OpenAIUsage{}
	var firstTokenMs *int
	writeEvent := func(event apicompat.ResponsesStreamEvent) error {
		raw, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", raw); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}
	for scanner.Scan() {
		data, ok := extractOpenAISSEDataLine(scanner.Text())
		if !ok || strings.TrimSpace(data) == "" || strings.TrimSpace(data) == "[DONE]" {
			continue
		}
		var chunk apicompat.ChatCompletionsChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			logger.L().Warn("openai responses force-chat bridge: failed to parse chat chunk", zap.Error(err))
			continue
		}
		if mappedModel != "" && chunk.Model == mappedModel {
			chunk.Model = originalModel
		}
		for _, event := range apicompat.ChatCompletionsChunkToResponsesEvents(&chunk, state) {
			if firstTokenMs == nil && event.Type == "response.output_text.delta" {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}
			if err := writeEvent(event); err != nil {
				return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, nil
			}
		}
		if state.Usage != nil {
			*usage = openAIUsageFromResponsesCompatUsage(state.Usage)
		}
	}
	if err := scanner.Err(); err != nil {
		return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, fmt.Errorf("native chat stream read error: %w", err)
	}
	final := apicompat.FinalizeChatCompletionsToResponsesStream(state)
	if err := writeEvent(final); err != nil {
		return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, nil
	}
	_, _ = fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
	if state.Usage != nil {
		*usage = openAIUsageFromResponsesCompatUsage(state.Usage)
	}
	return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, nil
}

func writeResponsesBridgeError(c *gin.Context, statusCode int, errType, message, reason string) {
	errPayload := gin.H{"type": errType, "message": message}
	if strings.TrimSpace(reason) != "" {
		errPayload["code"] = strings.TrimSpace(reason)
	}
	c.JSON(statusCode, gin.H{"error": errPayload})
}

func openAIUsageFromResponsesCompatUsage(usage *apicompat.ResponsesUsage) OpenAIUsage {
	if usage == nil {
		return OpenAIUsage{}
	}
	out := OpenAIUsage{
		InputTokens:  usage.InputTokens,
		OutputTokens: usage.OutputTokens,
	}
	if usage.InputTokensDetails != nil {
		out.CacheReadInputTokens = usage.InputTokensDetails.CachedTokens
	}
	return out
}
