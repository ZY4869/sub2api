package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

func (s *GatewayService) handleErrorResponse(ctx context.Context, resp *http.Response, c *gin.Context, account *Account) (*ForwardResult, error) {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	logger.LegacyPrintf("service.gateway", "[Forward] Upstream error (non-retryable): Account=%d(%s) Status=%d RequestID=%s Body=%s", account.ID, account.Name, resp.StatusCode, resp.Header.Get("x-request-id"), truncateString(string(body), 1000))
	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(body))
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
	if isClaudeCodeCredentialScopeError(upstreamMsg) && c != nil {
		if v, ok := c.Get(claudeMimicDebugInfoKey); ok {
			if line, ok := v.(string); ok && strings.TrimSpace(line) != "" {
				logger.LegacyPrintf("service.gateway", "[ClaudeMimicDebugOnError] status=%d request_id=%s %s", resp.StatusCode, resp.Header.Get("x-request-id"), line)
			}
		}
	}
	upstreamDetail := ""
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		upstreamDetail = truncateString(string(body), maxBytes)
	}
	setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "http_error", Message: upstreamMsg, Detail: upstreamDetail})
	shouldDisable := false
	if s.rateLimitService != nil {
		shouldDisable = s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, body)
	}
	if shouldDisable {
		return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: body}
	}
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		logger.LegacyPrintf("service.gateway", "Upstream error %d (account=%d platform=%s type=%s): %s", resp.StatusCode, account.ID, account.Platform, account.Type, truncateForLog(body, s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes))
	}
	if status, errType, errMsg, matched := applyErrorPassthroughRule(c, account.Platform, resp.StatusCode, body, http.StatusBadGateway, "upstream_error", "Upstream request failed"); matched {
		c.JSON(status, gin.H{"type": "error", "error": gin.H{"type": errType, "message": errMsg}})
		summary := upstreamMsg
		if summary == "" {
			summary = errMsg
		}
		if summary == "" {
			return nil, fmt.Errorf("upstream error: %d (passthrough rule matched)", resp.StatusCode)
		}
		return nil, fmt.Errorf("upstream error: %d (passthrough rule matched) message=%s", resp.StatusCode, summary)
	}
	var errType, errMsg string
	var statusCode int
	switch resp.StatusCode {
	case 400:
		c.Data(http.StatusBadRequest, "application/json", body)
		summary := upstreamMsg
		if summary == "" {
			summary = truncateForLog(body, 512)
		}
		if summary == "" {
			return nil, fmt.Errorf("upstream error: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("upstream error: %d message=%s", resp.StatusCode, summary)
	case 401:
		statusCode = http.StatusBadGateway
		errType = "upstream_error"
		errMsg = "Upstream authentication failed, please contact administrator"
	case 403:
		statusCode = http.StatusBadGateway
		errType = "upstream_error"
		errMsg = "Upstream access forbidden, please contact administrator"
	case 429:
		statusCode = http.StatusTooManyRequests
		errType = "rate_limit_error"
		errMsg = "Upstream rate limit exceeded, please retry later"
	case 529:
		statusCode = http.StatusServiceUnavailable
		errType = "overloaded_error"
		errMsg = "Upstream service overloaded, please retry later"
	case 500, 502, 503, 504:
		statusCode = http.StatusBadGateway
		errType = "upstream_error"
		errMsg = "Upstream service temporarily unavailable"
	default:
		statusCode = http.StatusBadGateway
		errType = "upstream_error"
		errMsg = "Upstream request failed"
	}
	c.JSON(statusCode, gin.H{"type": "error", "error": gin.H{"type": errType, "message": errMsg}})
	if upstreamMsg == "" {
		return nil, fmt.Errorf("upstream error: %d", resp.StatusCode)
	}
	return nil, fmt.Errorf("upstream error: %d message=%s", resp.StatusCode, upstreamMsg)
}
func (s *GatewayService) handleRetryExhaustedSideEffects(ctx context.Context, resp *http.Response, account *Account) {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	statusCode := resp.StatusCode
	if account.IsOAuth() && statusCode == 403 {
		s.rateLimitService.HandleUpstreamError(ctx, account, statusCode, resp.Header, body)
		logger.LegacyPrintf("service.gateway", "Account %d: marked as error after %d retries for status %d", account.ID, maxRetryAttempts, statusCode)
	} else {
		logger.LegacyPrintf("service.gateway", "Account %d: upstream error %d after %d retries (not marking account)", account.ID, statusCode, maxRetryAttempts)
	}
}
func (s *GatewayService) handleFailoverSideEffects(ctx context.Context, resp *http.Response, account *Account) {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, body)
}
func (s *GatewayService) handleRetryExhaustedError(ctx context.Context, resp *http.Response, c *gin.Context, account *Account) (*ForwardResult, error) {
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(respBody))
	s.handleRetryExhaustedSideEffects(ctx, resp, account)
	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
	if isClaudeCodeCredentialScopeError(upstreamMsg) && c != nil {
		if v, ok := c.Get(claudeMimicDebugInfoKey); ok {
			if line, ok := v.(string); ok && strings.TrimSpace(line) != "" {
				logger.LegacyPrintf("service.gateway", "[ClaudeMimicDebugOnError] status=%d request_id=%s %s", resp.StatusCode, resp.Header.Get("x-request-id"), line)
			}
		}
	}
	upstreamDetail := ""
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		upstreamDetail = truncateString(string(respBody), maxBytes)
	}
	setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "retry_exhausted", Message: upstreamMsg, Detail: upstreamDetail})
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		logger.LegacyPrintf("service.gateway", "Upstream error %d retries_exhausted (account=%d platform=%s type=%s): %s", resp.StatusCode, account.ID, account.Platform, account.Type, truncateForLog(respBody, s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes))
	}
	if status, errType, errMsg, matched := applyErrorPassthroughRule(c, account.Platform, resp.StatusCode, respBody, http.StatusBadGateway, "upstream_error", "Upstream request failed after retries"); matched {
		c.JSON(status, gin.H{"type": "error", "error": gin.H{"type": errType, "message": errMsg}})
		summary := upstreamMsg
		if summary == "" {
			summary = errMsg
		}
		if summary == "" {
			return nil, fmt.Errorf("upstream error: %d (retries exhausted, passthrough rule matched)", resp.StatusCode)
		}
		return nil, fmt.Errorf("upstream error: %d (retries exhausted, passthrough rule matched) message=%s", resp.StatusCode, summary)
	}
	c.JSON(http.StatusBadGateway, gin.H{"type": "error", "error": gin.H{"type": "upstream_error", "message": "Upstream request failed after retries"}})
	if upstreamMsg == "" {
		return nil, fmt.Errorf("upstream error: %d (retries exhausted)", resp.StatusCode)
	}
	return nil, fmt.Errorf("upstream error: %d (retries exhausted) message=%s", resp.StatusCode, upstreamMsg)
}
