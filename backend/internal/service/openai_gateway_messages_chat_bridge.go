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

func shouldForwardAnthropicMessagesViaChatCompletions(account *Account) bool {
	if !IsProtocolGatewayAccount(account) {
		return false
	}
	openAIAccount := ResolveProtocolGatewayInboundAccount(account, PlatformOpenAI)
	return openAIAccount != nil &&
		GetAccountGatewayProtocol(openAIAccount) == PlatformOpenAI &&
		GetAccountGatewayOpenAIRequestFormat(openAIAccount) == GatewayOpenAIRequestFormatChatCompletions
}

func (s *OpenAIGatewayService) forwardAnthropicViaRawChatCompletions(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	anthropicReq *apicompat.AnthropicRequest,
	originalModel string,
	topLevelEffort string,
	promptCacheKey string,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()
	ctx = EnsureRequestMetadata(ctx)
	account = ResolveProtocolGatewayInboundAccount(account, PlatformOpenAI)

	runtimeRequestedModel := anthropicReq.Model
	if entry, ok := PublishedPublicCatalogEntryFromContext(ctx); ok && entry != nil {
		if sourceModel := strings.TrimSpace(entry.SourceModelID); sourceModel != "" {
			runtimeRequestedModel = sourceModel
		}
	}
	anthropicReq.Model = runtimeRequestedModel
	claudeCapability := RecordClaudeCapabilityMetadataRequestedOnly(ctx, runtimeRequestedModel, topLevelEffort)
	anthropicReq.Model = firstNonEmptyString(claudeCapability.RequestedModelNormalized, anthropicReq.Model)
	applyOpenAICompatModelNormalization(anthropicReq)
	entryEffortResolution := ResolveAnthropicEffortForOpenAI(func() string {
		if anthropicReq.OutputConfig == nil {
			return ""
		}
		return anthropicReq.OutputConfig.Effort
	}(), topLevelEffort)

	chatReq, toolProxies, err := apicompat.AnthropicToChatCompletionsRequest(anthropicReq)
	if err != nil {
		if writeLocalizedCompatError(c, writeAnthropicError, "invalid_request_error", err) {
			return nil, err
		}
		return nil, fmt.Errorf("convert anthropic to chat completions: %w", err)
	}

	originalChatModel := strings.TrimSpace(chatReq.Model)
	mappedModel := normalizeOpenAIModelForUpstream(account, resolveOpenAIForwardModel(account, originalChatModel, defaultMappedModel))
	chatReq.Model = mappedModel
	if chatReq.Stream && chatReq.StreamOptions == nil {
		chatReq.StreamOptions = &apicompat.ChatStreamOptions{IncludeUsage: true}
	} else if chatReq.Stream {
		chatReq.StreamOptions.IncludeUsage = true
	}
	if entryEffortResolution.Effective != nil {
		chatReq.ReasoningEffort = *entryEffortResolution.Effective
	}

	requestBody, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic chat bridge request: %w", err)
	}
	logger.L().Debug("openai messages force-chat bridge: model mapping applied",
		zap.Int64("account_id", account.ID),
		zap.String("original_model", originalModel),
		zap.String("mapped_model", mappedModel),
		zap.Bool("stream", chatReq.Stream),
	)

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
		return nil, fmt.Errorf("build anthropic chat bridge request: %w", err)
	}
	if promptCacheKey != "" && isChatGPTOpenAIOAuthAccount(account) {
		apiKeyID := getAPIKeyIDFromContext(c)
		upstreamReq.Header.Set("session_id", generateSessionUUID(isolateOpenAISessionID(apiKeyID, promptCacheKey)))
	}

	proxyURL := ""
	if account != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return nil, newOpenAITransportFailoverError(c, account, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Kind:               "failover",
				Message:            upstreamMsg,
			})
			if s.rateLimitService != nil {
				s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			}
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: account.IsPoolMode() && (account.IsPoolModeRetryableStatus(resp.StatusCode) || isOpenAITransientProcessingError(resp.StatusCode, upstreamMsg, respBody)),
			}
		}
		return s.handleAnthropicErrorResponse(resp, c, account)
	}

	var usage *OpenAIUsage
	var firstTokenMs *int
	if chatReq.Stream {
		streamResult, streamErr := s.streamChatCompletionsAsAnthropic(resp, c, startTime, originalModel, mappedModel, toolProxies)
		if streamErr != nil {
			return nil, streamErr
		}
		usage = streamResult.usage
		firstTokenMs = streamResult.firstTokenMs
	} else {
		usage, err = s.bufferChatCompletionsAsAnthropic(resp, c, originalModel, mappedModel, toolProxies)
		if err != nil {
			return nil, err
		}
	}
	if usage == nil {
		usage = &OpenAIUsage{}
	}
	result := &OpenAIForwardResult{
		RequestID:                resp.Header.Get("x-request-id"),
		Usage:                    *usage,
		Model:                    originalModel,
		BillingModel:             mappedModel,
		UpstreamModel:            mappedModel,
		ServiceTier:              extractOpenAIServiceTierFromBody(requestBody),
		ReasoningEffort:          entryEffortResolution.Effective,
		ReasoningEffortRaw:       entryEffortResolution.Raw,
		ReasoningEffortEffective: entryEffortResolution.Effective,
		ReasoningEffortSource:    entryEffortResolution.Source,
		Stream:                   chatReq.Stream,
		Duration:                 time.Since(startTime),
		FirstTokenMs:             firstTokenMs,
	}
	applyClaudeCapabilityToOpenAIForwardResult(result, claudeCapability)
	return result, nil
}

func (s *OpenAIGatewayService) bufferChatCompletionsAsAnthropic(
	resp *http.Response,
	c *gin.Context,
	originalModel string,
	mappedModel string,
	toolProxies map[string]apicompat.ResponsesToolProxy,
) (*OpenAIUsage, error) {
	body, err := readUpstreamResponseBodyLimited(resp.Body, resolveUpstreamResponseReadLimit(s.cfg))
	if err != nil {
		return nil, err
	}
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("parse chat completions anthropic bridge response: invalid json response")
	}
	var chatResp apicompat.ChatCompletionsResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("parse chat completions anthropic bridge response: %w", err)
	}
	if mappedModel != "" && chatResp.Model == mappedModel {
		chatResp.Model = originalModel
	}
	anthropicResp := apicompat.ChatCompletionsResponseToAnthropic(&chatResp, originalModel, toolProxies)
	responseBody, err := json.Marshal(anthropicResp)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic bridge response: %w", err)
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	c.Data(resp.StatusCode, "application/json", responseBody)
	usage := extractOpenAIChatUsage(chatResp.Usage)
	return &usage, nil
}

func (s *OpenAIGatewayService) streamChatCompletionsAsAnthropic(
	resp *http.Response,
	c *gin.Context,
	startTime time.Time,
	originalModel string,
	mappedModel string,
	toolProxies map[string]apicompat.ResponsesToolProxy,
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
	maxLineSize := defaultMaxLineSize
	if s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)
	state := apicompat.NewChatCompletionsToAnthropicStreamState(originalModel, toolProxies)
	usage := &OpenAIUsage{}
	var firstTokenMs *int

	writeEvents := func(events []apicompat.AnthropicStreamEvent) bool {
		for _, event := range events {
			if firstTokenMs == nil && event.Type == "content_block_delta" && event.Delta != nil {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}
			sse, err := apicompat.ResponsesAnthropicEventToSSE(event)
			if err != nil {
				logger.L().Warn("openai messages force-chat stream: failed to marshal event", zap.Error(err))
				continue
			}
			if _, err := fmt.Fprint(c.Writer, sse); err != nil {
				return false
			}
		}
		if len(events) > 0 {
			flusher.Flush()
		}
		return true
	}

	for scanner.Scan() {
		data, ok := extractOpenAISSEDataLine(scanner.Text())
		if !ok || strings.TrimSpace(data) == "" || strings.TrimSpace(data) == "[DONE]" {
			continue
		}
		var chunk apicompat.ChatCompletionsChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			logger.L().Warn("openai messages force-chat bridge: failed to parse chat chunk", zap.Error(err))
			continue
		}
		if mappedModel != "" && chunk.Model == mappedModel {
			chunk.Model = originalModel
		}
		if chunk.Usage != nil {
			*usage = extractOpenAIChatUsage(chunk.Usage)
		}
		if !writeEvents(apicompat.ChatCompletionsChunkToAnthropicEvents(&chunk, state)) {
			return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, fmt.Errorf("chat completions anthropic stream read error: %w", err)
	}
	if !writeEvents(apicompat.FinalizeChatCompletionsAnthropicStream(state)) {
		return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, nil
	}
	return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, nil
}

func extractOpenAIChatUsage(usage *apicompat.ChatUsage) OpenAIUsage {
	if usage == nil {
		return OpenAIUsage{}
	}
	out := OpenAIUsage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
	}
	if usage.PromptTokensDetails != nil {
		out.CacheReadInputTokens = usage.PromptTokensDetails.CachedTokens
	}
	return out
}
