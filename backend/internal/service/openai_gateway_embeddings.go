package service

import (
	"bytes"
	"context"
	"errors"
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

func (s *OpenAIGatewayService) ForwardEmbeddings(ctx context.Context, c *gin.Context, account *Account, body []byte) (*OpenAIForwardResult, error) {
	account = ResolveProtocolGatewayInboundAccount(account, PlatformOpenAI)
	startTime := time.Now()
	ctx = EnsureRequestMetadata(ctx)
	if account == nil || !account.IsOpenAIApiKey() {
		msg := "OpenAI embeddings require an OpenAI API-key account"
		setOpsUpstreamError(c, http.StatusForbidden, msg, "")
		if c != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"type": "forbidden_error", "message": msg}})
		}
		return nil, errors.New("openai embeddings require openai apikey account")
	}

	originalModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	mappedModel := account.GetMappedModel(originalModel)
	if mappedModel != "" && mappedModel != originalModel {
		patched, err := sjson.SetBytes(body, "model", mappedModel)
		if err != nil {
			return nil, fmt.Errorf("patch embeddings model: %w", err)
		}
		body = patched
	}

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	targetURL, err := resolveOpenAIEmbeddingsTargetURL(account, s.validateUpstreamBaseURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("authorization", "Bearer "+token)
	for key, values := range c.Request.Header {
		lowerKey := strings.ToLower(key)
		if !openaiAllowedHeaders[lowerKey] {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	setOpsUpstreamRequestBody(c, body)
	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(MarkOpenAIHTTPUpstreamRequest(req), proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		return nil, newOpenAITransportFailoverError(c, account, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(respBody)))
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			upstreamDetail := ""
			if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
				maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
				if maxBytes <= 0 {
					maxBytes = 2048
				}
				upstreamDetail = truncateString(string(respBody), maxBytes)
			}
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "failover", Message: upstreamMsg, Detail: upstreamDetail})
			s.handleFailoverSideEffects(ctx, resp, account)
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode)}
		}
		nextResult, nextErr := s.handleErrorResponse(ctx, resp, c, account, body)
		return nextResult, nextErr
	}

	usage, err := s.handleEmbeddingsResponse(resp, c, account, originalModel, mappedModel)
	if err != nil {
		return nil, err
	}
	if usage == nil {
		usage = &OpenAIUsage{}
	}
	return &OpenAIForwardResult{
		RequestID:     resp.Header.Get("x-request-id"),
		Usage:         *usage,
		Model:         originalModel,
		UpstreamModel: mappedModel,
		Stream:        false,
		Duration:      time.Since(startTime),
	}, nil
}

func (s *OpenAIGatewayService) handleEmbeddingsResponse(resp *http.Response, c *gin.Context, account *Account, originalModel, mappedModel string) (*OpenAIUsage, error) {
	maxBytes := resolveUpstreamResponseReadLimit(s.cfg)
	body, err := readUpstreamResponseBodyLimited(resp.Body, maxBytes)
	if err != nil {
		if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
			setOpsUpstreamError(c, http.StatusBadGateway, "upstream response too large", "")
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "upstream_error", "message": "Upstream response too large"}})
		}
		return nil, err
	}
	SetOpsTraceUpstreamResponse(c, "openai_embeddings_upstream_response", body, resp.Header.Get("Content-Type"), false)
	usage := extractOpenAIEmbeddingsUsage(body)
	if originalModel != "" && mappedModel != "" && originalModel != mappedModel {
		body = s.replaceModelInResponseBody(body, mappedModel, originalModel)
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := "application/json"
	if s.cfg != nil && !s.cfg.Security.ResponseHeaders.Enabled {
		if upstreamType := resp.Header.Get("Content-Type"); upstreamType != "" {
			contentType = upstreamType
		}
	}
	SetOpsTraceGatewayResponse(c, "openai_embeddings_gateway_response", body, contentType, false)
	c.Data(resp.StatusCode, contentType, body)
	return &usage, nil
}

func extractOpenAIEmbeddingsUsage(body []byte) OpenAIUsage {
	inputTokens := int(firstPositiveInt64(
		gjson.GetBytes(body, "usage.input_tokens").Int(),
		gjson.GetBytes(body, "usage.prompt_tokens").Int(),
		gjson.GetBytes(body, "usage.total_tokens").Int(),
	))
	return OpenAIUsage{
		InputTokens:          inputTokens,
		CacheReadInputTokens: int(gjson.GetBytes(body, "usage.input_tokens_details.cached_tokens").Int()),
	}
}
