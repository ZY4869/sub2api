package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type OpenAIAlphaSearchForwardResult struct {
	RequestID       string
	StatusCode      int
	ResponseHeaders http.Header
	Duration        time.Duration
}

func (s *OpenAIGatewayService) ForwardAlphaSearch(ctx context.Context, c *gin.Context, account *Account, body []byte) (*OpenAIAlphaSearchForwardResult, error) {
	account = ResolveProtocolGatewayInboundAccount(account, PlatformOpenAI)
	startTime := time.Now()
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	targetURL, err := resolveOpenAIAlphaSearchTargetURL(account, s.validateUpstreamBaseURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("authorization", "Bearer "+token)
	applyOpenRouterAttributionRequestHeaders(account, req.Header)
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
	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		req.Header.Set("user-agent", customUA)
	}
	ApplyAccountRequestHeaderOverrides(req, account)
	setOpsUpstreamRequestBody(c, body)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(MarkOpenAIHTTPUpstreamRequest(req), proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		return nil, newOpenAITransportFailoverError(c, account, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	statusCode := resp.StatusCode
	c.Status(statusCode)
	if _, copyErr := io.Copy(c.Writer, resp.Body); copyErr != nil {
		return nil, copyErr
	}
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
	result := &OpenAIAlphaSearchForwardResult{
		RequestID:       resp.Header.Get("x-request-id"),
		StatusCode:      statusCode,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
	}
	return result, nil
}
