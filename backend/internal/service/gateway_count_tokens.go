package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func (s *GatewayService) ForwardCountTokens(ctx context.Context, c *gin.Context, account *Account, parsed *ParsedRequest) error {
	if parsed == nil {
		s.countTokensError(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return fmt.Errorf("parse request: empty request")
	}
	if account != nil && account.IsAnthropicAPIKeyPassthroughEnabled() {
		passthroughBody := parsed.Body
		if reqModel := parsed.Model; reqModel != "" {
			if mappedModel := account.GetMappedModel(reqModel); mappedModel != reqModel {
				passthroughBody = s.replaceModelInBody(passthroughBody, mappedModel)
				logger.LegacyPrintf("service.gateway", "CountTokens passthrough model mapping: %s -> %s (account: %s)", reqModel, mappedModel, account.Name)
			}
		}
		return s.forwardCountTokensAnthropicAPIKeyPassthrough(ctx, c, account, passthroughBody)
	}
	body := parsed.Body
	reqModel := s.resolveCanonicalRequestModel(ctx, parsed.Model)
	if reqModel == "" {
		reqModel = parsed.Model
	}
	isClaudeCode := isClaudeCodeRequest(ctx, c, parsed)
	shouldMimicClaudeCode := IsClaudeClientMimicEnabled(account, PlatformAnthropic) && !isClaudeCode
	if shouldMimicClaudeCode {
		normalizeOpts := claudeOAuthNormalizeOptions{stripSystemCacheControl: true}
		body, reqModel = normalizeClaudeOAuthRequestBody(body, reqModel, normalizeOpts)
	}
	if account.Platform == PlatformAntigravity {
		s.countTokensError(c, http.StatusNotFound, "not_found_error", "count_tokens endpoint is not supported for this platform")
		return nil
	}
	if reqModel != "" {
		mappedModel := reqModel
		mappingSource := ""
		if account.Type == AccountTypeAPIKey {
			mappedModel = account.GetMappedModel(reqModel)
			if mappedModel != reqModel {
				mappingSource = "account"
			}
		}
		if mappingSource == "" {
			protocolModel := s.resolveUpstreamModelID(ctx, account, reqModel)
			if protocolModel != "" && protocolModel != reqModel {
				mappedModel = protocolModel
				mappingSource = "registry_protocol"
			}
		}
		if mappedModel != reqModel {
			body = s.replaceModelInBody(body, mappedModel)
			reqModel = mappedModel
			logger.LegacyPrintf("service.gateway", "CountTokens model mapping applied: %s -> %s (account: %s, source=%s)", parsed.RawModel, mappedModel, account.Name, mappingSource)
		}
	}
	token, tokenType, err := s.GetAccessToken(ctx, account)
	if err != nil {
		s.countTokensError(c, http.StatusBadGateway, "upstream_error", "Failed to get access token")
		return err
	}
	upstreamReq, err := s.buildCountTokensRequest(ctx, c, account, body, token, tokenType, reqModel, shouldMimicClaudeCode)
	if err != nil {
		s.countTokensError(c, http.StatusInternalServerError, "api_error", "Failed to build request")
		return err
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.DoWithTLS(upstreamReq, proxyURL, account.ID, account.Concurrency, account.IsTLSFingerprintEnabled())
	if err != nil {
		setOpsUpstreamError(c, 0, sanitizeUpstreamErrorMessage(err.Error()), "")
		s.countTokensError(c, http.StatusBadGateway, "upstream_error", "Request failed")
		return fmt.Errorf("upstream request failed: %w", err)
	}
	maxReadBytes := resolveUpstreamResponseReadLimit(s.cfg)
	respBody, err := readUpstreamResponseBodyLimited(resp.Body, maxReadBytes)
	_ = resp.Body.Close()
	if err != nil {
		if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
			setOpsUpstreamError(c, http.StatusBadGateway, "upstream response too large", "")
			s.countTokensError(c, http.StatusBadGateway, "upstream_error", "Upstream response too large")
			return err
		}
		s.countTokensError(c, http.StatusBadGateway, "upstream_error", "Failed to read response")
		return err
	}
	if resp.StatusCode == 400 && s.isThinkingBlockSignatureError(respBody) && s.settingService.IsSignatureRectifierEnabled(ctx) {
		logger.LegacyPrintf("service.gateway", "Account %d: detected thinking block signature error on count_tokens, retrying with filtered thinking blocks", account.ID)
		filteredBody := FilterThinkingBlocksForRetry(body)
		retryReq, buildErr := s.buildCountTokensRequest(ctx, c, account, filteredBody, token, tokenType, reqModel, shouldMimicClaudeCode)
		if buildErr == nil {
			retryResp, retryErr := s.httpUpstream.DoWithTLS(retryReq, proxyURL, account.ID, account.Concurrency, account.IsTLSFingerprintEnabled())
			if retryErr == nil {
				resp = retryResp
				respBody, err = readUpstreamResponseBodyLimited(resp.Body, maxReadBytes)
				_ = resp.Body.Close()
				if err != nil {
					if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
						setOpsUpstreamError(c, http.StatusBadGateway, "upstream response too large", "")
						s.countTokensError(c, http.StatusBadGateway, "upstream_error", "Upstream response too large")
						return err
					}
					s.countTokensError(c, http.StatusBadGateway, "upstream_error", "Failed to read response")
					return err
				}
			}
		}
	}
	if resp.StatusCode >= 400 {
		s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
		upstreamDetail := ""
		if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
			maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
			if maxBytes <= 0 {
				maxBytes = 2048
			}
			upstreamDetail = truncateString(string(respBody), maxBytes)
		}
		setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)
		if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
			logger.LegacyPrintf("service.gateway", "count_tokens upstream error %d (account=%d platform=%s type=%s): %s", resp.StatusCode, account.ID, account.Platform, account.Type, truncateForLog(respBody, s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes))
		}
		errMsg := "Upstream request failed"
		switch resp.StatusCode {
		case 429:
			errMsg = "Rate limit exceeded"
		case 529:
			errMsg = "Service overloaded"
		}
		s.countTokensError(c, resp.StatusCode, "upstream_error", errMsg)
		if upstreamMsg == "" {
			return fmt.Errorf("upstream error: %d", resp.StatusCode)
		}
		return fmt.Errorf("upstream error: %d message=%s", resp.StatusCode, upstreamMsg)
	}
	c.Data(resp.StatusCode, "application/json", respBody)
	return nil
}
func (s *GatewayService) forwardCountTokensAnthropicAPIKeyPassthrough(ctx context.Context, c *gin.Context, account *Account, body []byte) error {
	token, tokenType, err := s.GetAccessToken(ctx, account)
	if err != nil {
		s.countTokensError(c, http.StatusBadGateway, "upstream_error", "Failed to get access token")
		return err
	}
	if tokenType != "apikey" {
		s.countTokensError(c, http.StatusBadGateway, "upstream_error", "Invalid account token type")
		return fmt.Errorf("anthropic api key passthrough requires apikey token, got: %s", tokenType)
	}
	upstreamReq, err := s.buildCountTokensRequestAnthropicAPIKeyPassthrough(ctx, c, account, body, token)
	if err != nil {
		s.countTokensError(c, http.StatusInternalServerError, "api_error", "Failed to build request")
		return err
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.DoWithTLS(upstreamReq, proxyURL, account.ID, account.Concurrency, account.IsTLSFingerprintEnabled())
	if err != nil {
		setOpsUpstreamError(c, 0, sanitizeUpstreamErrorMessage(err.Error()), "")
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: 0, Passthrough: true, Kind: "request_error", Message: sanitizeUpstreamErrorMessage(err.Error())})
		s.countTokensError(c, http.StatusBadGateway, "upstream_error", "Request failed")
		return fmt.Errorf("upstream request failed: %w", err)
	}
	maxReadBytes := resolveUpstreamResponseReadLimit(s.cfg)
	respBody, err := readUpstreamResponseBodyLimited(resp.Body, maxReadBytes)
	_ = resp.Body.Close()
	if err != nil {
		if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
			setOpsUpstreamError(c, http.StatusBadGateway, "upstream response too large", "")
			s.countTokensError(c, http.StatusBadGateway, "upstream_error", "Upstream response too large")
			return err
		}
		s.countTokensError(c, http.StatusBadGateway, "upstream_error", "Failed to read response")
		return err
	}
	if resp.StatusCode >= 400 {
		if s.rateLimitService != nil {
			s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
		}
		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
		if isCountTokensUnsupported404(resp.StatusCode, respBody) {
			logger.LegacyPrintf("service.gateway", "[count_tokens] Upstream does not support count_tokens (404), returning 404: account=%d name=%s msg=%s", account.ID, account.Name, truncateString(upstreamMsg, 512))
			s.countTokensError(c, http.StatusNotFound, "not_found_error", "count_tokens endpoint is not supported by upstream")
			return nil
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
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Passthrough: true, Kind: "http_error", Message: upstreamMsg, Detail: upstreamDetail})
		errMsg := "Upstream request failed"
		switch resp.StatusCode {
		case 429:
			errMsg = "Rate limit exceeded"
		case 529:
			errMsg = "Service overloaded"
		}
		s.countTokensError(c, resp.StatusCode, "upstream_error", errMsg)
		if upstreamMsg == "" {
			return fmt.Errorf("upstream error: %d", resp.StatusCode)
		}
		return fmt.Errorf("upstream error: %d message=%s", resp.StatusCode, upstreamMsg)
	}
	writeAnthropicPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, respBody)
	return nil
}
func (s *GatewayService) buildCountTokensRequestAnthropicAPIKeyPassthrough(ctx context.Context, c *gin.Context, account *Account, body []byte, token string) (*http.Request, error) {
	targetURL := claudeAPICountTokensURL
	baseURL := account.GetBaseURL()
	if baseURL != "" {
		validatedURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		targetURL = validatedURL + "/v1/messages/count_tokens?beta=true"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if c != nil && c.Request != nil {
		for key, values := range c.Request.Header {
			lowerKey := strings.ToLower(strings.TrimSpace(key))
			if !allowedHeaders[lowerKey] {
				continue
			}
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
	}
	req.Header.Del("authorization")
	req.Header.Del("x-api-key")
	req.Header.Del("x-goog-api-key")
	req.Header.Del("cookie")
	req.Header.Set("x-api-key", token)
	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}
	if req.Header.Get("anthropic-version") == "" {
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	return req, nil
}
func (s *GatewayService) buildCountTokensRequest(ctx context.Context, c *gin.Context, account *Account, body []byte, token, tokenType, modelID string, mimicClaudeCode bool) (*http.Request, error) {
	targetURL := claudeAPICountTokensURL
	if account.Type == AccountTypeAPIKey {
		baseURL := account.GetBaseURL()
		if baseURL != "" {
			validatedURL, err := s.validateUpstreamBaseURL(baseURL)
			if err != nil {
				return nil, err
			}
			targetURL = validatedURL + "/v1/messages/count_tokens?beta=true"
		}
	}
	clientHeaders := http.Header{}
	if c != nil && c.Request != nil {
		clientHeaders = c.Request.Header
	}
	if mimicClaudeCode && s.identityService != nil {
		fp, err := s.identityService.GetOrCreateFingerprint(ctx, account.ID, clientHeaders)
		if err == nil {
			accountUUID := account.GetExtraString("account_uuid")
			if accountUUID != "" && fp.ClientID != "" {
				if newBody, err := s.identityService.RewriteUserIDWithMasking(ctx, body, account, accountUUID, fp.ClientID, fp.UserAgent); err == nil && len(newBody) > 0 {
					body = newBody
				}
			}
		}
	}
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if tokenType == "oauth" {
		req.Header.Set("authorization", "Bearer "+token)
	} else {
		req.Header.Set("x-api-key", token)
	}
	for key, values := range clientHeaders {
		lowerKey := strings.ToLower(key)
		if allowedHeaders[lowerKey] {
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
	}
	if mimicClaudeCode && s.identityService != nil {
		fp, _ := s.identityService.GetOrCreateFingerprint(ctx, account.ID, clientHeaders)
		if fp != nil {
			s.identityService.ApplyFingerprint(req, fp)
		}
	}
	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}
	if req.Header.Get("anthropic-version") == "" {
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	if tokenType == "oauth" || mimicClaudeCode {
		applyClaudeOAuthHeaderDefaults(req, false)
	}
	ctEffectiveDropSet := mergeDropSets(s.getBetaPolicyFilterSet(ctx, c, account))
	if tokenType == "oauth" {
		if mimicClaudeCode {
			applyClaudeCodeMimicHeaders(req, false)
			incomingBeta := req.Header.Get("anthropic-beta")
			requiredBetas := []string{claude.BetaClaudeCode, claude.BetaOAuth, claude.BetaInterleavedThinking, claude.BetaTokenCounting}
			req.Header.Set("anthropic-beta", mergeAnthropicBetaDropping(requiredBetas, incomingBeta, ctEffectiveDropSet))
		} else {
			clientBetaHeader := req.Header.Get("anthropic-beta")
			if clientBetaHeader == "" {
				req.Header.Set("anthropic-beta", claude.CountTokensBetaHeader)
			} else {
				beta := s.getBetaHeader(modelID, clientBetaHeader)
				if !strings.Contains(beta, claude.BetaTokenCounting) {
					beta = beta + "," + claude.BetaTokenCounting
				}
				req.Header.Set("anthropic-beta", stripBetaTokensWithSet(beta, ctEffectiveDropSet))
			}
		}
	} else {
		if mimicClaudeCode {
			applyClaudeCodeMimicHeaders(req, false)
			incomingBeta := req.Header.Get("anthropic-beta")
			requiredBetas := []string{claude.BetaClaudeCode, claude.BetaInterleavedThinking, claude.BetaFineGrainedToolStreaming, claude.BetaTokenCounting}
			req.Header.Set("anthropic-beta", mergeAnthropicBetaDropping(requiredBetas, incomingBeta, ctEffectiveDropSet))
		} else if existingBeta := req.Header.Get("anthropic-beta"); existingBeta != "" {
			req.Header.Set("anthropic-beta", stripBetaTokensWithSet(existingBeta, ctEffectiveDropSet))
		} else if s.cfg != nil && s.cfg.Gateway.InjectBetaForAPIKey {
			if requestNeedsBetaFeatures(body) {
				if beta := defaultAPIKeyBetaHeader(body); beta != "" {
					req.Header.Set("anthropic-beta", beta)
				}
			}
		}
	}
	if c != nil && (tokenType == "oauth" || mimicClaudeCode) {
		c.Set(claudeMimicDebugInfoKey, buildClaudeMimicDebugLine(req, body, account, tokenType, mimicClaudeCode))
	}
	if s.debugClaudeMimicEnabled() {
		logClaudeMimicDebug(req, body, account, tokenType, mimicClaudeCode)
	}
	return req, nil
}
