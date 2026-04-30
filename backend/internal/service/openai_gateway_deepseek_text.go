package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

func (s *OpenAIGatewayService) forwardDeepSeekNativeChatCompletions(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()

	prepared, err := prepareDeepSeekNativeChatRequestBody(account, body, defaultMappedModel)
	if err != nil {
		var requestErr *deepSeekChatRequestError
		if errors.As(err, &requestErr) {
			writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", requestErr.message, requestErr.reason)
			return nil, err
		}
		return nil, fmt.Errorf("prepare deepseek native chat request: %w", err)
	}
	logDeepSeekBetaRouting(
		account,
		prepared.originalModel,
		prepared.mappedModel,
		EndpointChatCompletions,
		EndpointChatCompletions,
		prepared.explicitBetaSpecified,
		prepared.explicitBetaValue,
		prepared.autoBetaRequested,
		prepared.betaEnabled,
		prepared.betaStripped,
	)

	ctx = WithOpenAICodexRequestModel(ctx, prepared.mappedModel)
	if c != nil && c.Request != nil {
		c.Request = c.Request.WithContext(ctx)
	}
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	upstreamReq, err := s.buildDeepSeekChatCompletionsUpstreamRequest(ctx, c, account, prepared.body, token, prepared.stream, prepared.betaEnabled)
	if err != nil {
		return nil, fmt.Errorf("build deepseek native chat request: %w", err)
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
			Platform:           RoutingPlatformForAccount(account),
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
				Platform:           RoutingPlatformForAccount(account),
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
	if prepared.stream {
		streamResult, streamErr := s.handleNativeChatCompletionsStreamingResponse(
			ctx,
			resp,
			c,
			account,
			startTime,
			prepared.originalModel,
			prepared.mappedModel,
			prepared.clientRequestedUsage,
		)
		if streamErr != nil {
			return nil, streamErr
		}
		usage = streamResult.usage
		firstTokenMs = streamResult.firstTokenMs
	} else {
		usage, err = s.handleNativeChatCompletionsNonStreamingResponse(resp, c, prepared.originalModel, prepared.mappedModel)
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
		Model:           prepared.originalModel,
		BillingModel:    prepared.mappedModel,
		UpstreamModel:   prepared.mappedModel,
		ServiceTier:     extractOpenAIServiceTierFromBody(prepared.body),
		ReasoningEffort: extractOpenAIReasoningEffortFromBody(prepared.body, prepared.originalModel),
		Stream:          prepared.stream,
		Duration:        time.Since(startTime),
		FirstTokenMs:    firstTokenMs,
	}, nil
}

func (s *OpenAIGatewayService) ForwardDeepSeekCompletions(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()

	originalModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if originalModel == "" {
		return nil, fmt.Errorf("missing model")
	}
	mappedModel := resolveOpenAIForwardModel(account, originalModel, defaultMappedModel)
	if mappedModel != "" && mappedModel != originalModel {
		nextBody, err := sjson.SetBytes(body, "model", mappedModel)
		if err != nil {
			return nil, fmt.Errorf("rewrite deepseek completions model: %w", err)
		}
		body = nextBody
	}
	if !isDeepSeekFIMCompletionModel(mappedModel) {
		writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", "DeepSeek /v1/completions currently only supports deepseek-v4-flash or deepseek-v4-pro", "deepseek_fim_model_unsupported")
		return nil, fmt.Errorf("deepseek completions unsupported model: %s", mappedModel)
	}

	stream := gjson.GetBytes(body, "stream").Bool()
	logDeepSeekBetaRouting(account, originalModel, mappedModel, EndpointCompletions, EndpointCompletions, false, false, false, true, false)

	ctx = WithOpenAICodexRequestModel(ctx, mappedModel)
	if c != nil && c.Request != nil {
		c.Request = c.Request.WithContext(ctx)
	}
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	upstreamReq, err := s.buildDeepSeekCompletionsUpstreamRequest(ctx, c, account, body, token, stream)
	if err != nil {
		return nil, fmt.Errorf("build deepseek completions request: %w", err)
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
			Platform:           RoutingPlatformForAccount(account),
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
				Platform:           RoutingPlatformForAccount(account),
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
	if stream {
		streamResult, streamErr := s.handleDeepSeekCompletionsStreamingResponse(resp, c, startTime, originalModel, mappedModel)
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
		RequestID:     resp.Header.Get("x-request-id"),
		Usage:         *usage,
		Model:         originalModel,
		BillingModel:  mappedModel,
		UpstreamModel: mappedModel,
		Stream:        stream,
		Duration:      time.Since(startTime),
		FirstTokenMs:  firstTokenMs,
	}, nil
}

func (s *OpenAIGatewayService) buildDeepSeekChatCompletionsUpstreamRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	token string,
	isStream bool,
	beta bool,
) (*http.Request, error) {
	targetURL, err := resolveDeepSeekChatCompletionsTargetURL(account, s.validateUpstreamBaseURL, beta)
	if err != nil {
		return nil, err
	}
	return buildOpenAIStyleJSONRequest(ctx, c, account, body, token, targetURL, isStream)
}

func (s *OpenAIGatewayService) buildDeepSeekCompletionsUpstreamRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	token string,
	isStream bool,
) (*http.Request, error) {
	targetURL, err := resolveDeepSeekCompletionsTargetURL(account, s.validateUpstreamBaseURL)
	if err != nil {
		return nil, err
	}
	return buildOpenAIStyleJSONRequest(ctx, c, account, body, token, targetURL, isStream)
}

func buildOpenAIStyleJSONRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	token string,
	targetURL string,
	isStream bool,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(string(body)))
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

func (s *OpenAIGatewayService) handleDeepSeekCompletionsStreamingResponse(
	resp *http.Response,
	c *gin.Context,
	startTime time.Time,
	originalModel string,
	mappedModel string,
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
			parseOpenAIChatCompletionsSSEUsage(dataBytes, usage)
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
			logger.LegacyPrintf("service.openai_gateway", "Client disconnected during deepseek completions streaming, continuing to drain upstream for billing")
			continue
		}
		flusher.Flush()
	}

	if err := scanner.Err(); err != nil {
		if clientDisconnected || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, nil
		}
		return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, fmt.Errorf("deepseek completions stream read error: %w", err)
	}

	return &openaiStreamingResult{usage: usage, firstTokenMs: firstTokenMs}, nil
}

func logDeepSeekBetaRouting(account *Account, originalModel string, mappedModel string, inboundEndpoint string, upstreamEndpoint string, explicitBetaSpecified bool, explicitBetaValue bool, autoBetaRequested bool, betaEnabled bool, betaFieldsStripped bool) {
	logger.L().Debug(
		"deepseek.beta_routing",
		zap.Int64("account_id", account.ID),
		zap.String("account_name", account.Name),
		zap.String("runtime_platform", RoutingPlatformForAccount(account)),
		zap.String("original_model", strings.TrimSpace(originalModel)),
		zap.String("mapped_model", strings.TrimSpace(mappedModel)),
		zap.Bool("deepseek_beta_explicit_specified", explicitBetaSpecified),
		zap.Bool("deepseek_beta_explicit_value", explicitBetaValue),
		zap.Bool("deepseek_beta_auto_requested", autoBetaRequested),
		zap.Bool("deepseek_beta_enabled", betaEnabled),
		zap.Bool("deepseek_beta_fields_stripped", betaFieldsStripped),
		zap.String("inbound_endpoint", inboundEndpoint),
		zap.String("upstream_endpoint", upstreamEndpoint),
	)
}
