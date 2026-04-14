package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

func (s *AntigravityGatewayService) Forward(ctx context.Context, c *gin.Context, account *Account, body []byte, isStickySession bool) (*ForwardResult, error) {
	if account.Type == AccountTypeUpstream {
		return s.ForwardUpstream(ctx, c, account, body)
	}
	startTime := time.Now()
	sessionID := getSessionID(c)
	prefix := logPrefix(sessionID, account.Name)
	var claudeReq antigravity.ClaudeRequest
	if err := json.Unmarshal(body, &claudeReq); err != nil {
		return nil, s.writeClaudeError(c, http.StatusBadRequest, "invalid_request_error", "Invalid request body")
	}
	if strings.TrimSpace(claudeReq.Model) == "" {
		return nil, s.writeClaudeError(c, http.StatusBadRequest, "invalid_request_error", "Missing model")
	}
	originalModel := claudeReq.Model
	mappedModel := s.getMappedModel(account, claudeReq.Model)
	if mappedModel == "" {
		return nil, s.writeClaudeError(c, http.StatusForbidden, "permission_error", fmt.Sprintf("model %s not in whitelist", claudeReq.Model))
	}
	thinkingEnabled := claudeReq.Thinking != nil && (claudeReq.Thinking.Type == "enabled" || claudeReq.Thinking.Type == "adaptive")
	mappedModel = applyThinkingModelSuffix(mappedModel, thinkingEnabled)
	billingModel := mappedModel
	if s.tokenProvider == nil {
		return nil, s.writeClaudeError(c, http.StatusBadGateway, "api_error", "Antigravity token provider not configured")
	}
	accessToken, err := s.tokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return nil, s.writeClaudeError(c, http.StatusBadGateway, "authentication_error", "Failed to get upstream access token")
	}
	projectID := strings.TrimSpace(account.GetCredential("project_id"))
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	transformOpts := s.getClaudeTransformOptions(ctx)
	transformOpts.EnableIdentityPatch = true
	geminiBody, err := antigravity.TransformClaudeToGeminiWithOptions(&claudeReq, projectID, mappedModel, transformOpts)
	if err != nil {
		return nil, s.writeClaudeError(c, http.StatusBadRequest, "invalid_request_error", "Invalid request")
	}
	action := "streamGenerateContent"
	result, err := s.antigravityRetryLoop(antigravityRetryLoopParams{ctx: ctx, prefix: prefix, account: account, proxyURL: proxyURL, accessToken: accessToken, action: action, body: geminiBody, c: c, httpUpstream: s.httpUpstream, settingService: s.settingService, accountRepo: s.accountRepo, handleError: s.handleUpstreamError, requestedModel: originalModel, isStickySession: isStickySession, groupID: 0, sessionHash: ""})
	if err != nil {
		if switchErr, ok := IsAntigravityAccountSwitchError(err); ok {
			return nil, &UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable, ForceCacheBilling: switchErr.IsStickySession}
		}
		if c.Request.Context().Err() != nil {
			return nil, s.writeClaudeError(c, http.StatusBadGateway, "client_disconnected", "Client disconnected before upstream response")
		}
		return nil, s.writeClaudeError(c, http.StatusBadGateway, "upstream_error", "Upstream request failed after retries")
	}
	resp := result.resp
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if resp.StatusCode == http.StatusBadRequest && isSignatureRelatedError(respBody) && s.settingService.IsSignatureRectifierEnabled(ctx) {
			upstreamMsg := strings.TrimSpace(extractAntigravityErrorMessage(respBody))
			upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
			logBody, maxBytes := s.getLogConfig()
			upstreamDetail := s.getUpstreamErrorDetail(respBody)
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "signature_error", Message: upstreamMsg, Detail: upstreamDetail})
			retryStages := []struct {
				name  string
				strip func(*antigravity.ClaudeRequest) (bool, error)
			}{{name: "thinking-only", strip: stripThinkingFromClaudeRequest}, {name: "thinking+tools", strip: stripSignatureSensitiveBlocksFromClaudeRequest}}
			for _, stage := range retryStages {
				retryClaudeReq := claudeReq
				retryClaudeReq.Messages = append([]antigravity.ClaudeMessage(nil), claudeReq.Messages...)
				stripped, stripErr := stage.strip(&retryClaudeReq)
				if stripErr != nil || !stripped {
					continue
				}
				logger.LegacyPrintf("service.antigravity_gateway", "Antigravity account %d: detected signature-related 400, retrying once (%s)", account.ID, stage.name)
				retryGeminiBody, txErr := antigravity.TransformClaudeToGeminiWithOptions(&retryClaudeReq, projectID, mappedModel, s.getClaudeTransformOptions(ctx))
				if txErr != nil {
					continue
				}
				retryResult, retryErr := s.antigravityRetryLoop(antigravityRetryLoopParams{ctx: ctx, prefix: prefix, account: account, proxyURL: proxyURL, accessToken: accessToken, action: action, body: retryGeminiBody, c: c, httpUpstream: s.httpUpstream, settingService: s.settingService, accountRepo: s.accountRepo, handleError: s.handleUpstreamError, requestedModel: originalModel, isStickySession: isStickySession, groupID: 0, sessionHash: ""})
				if retryErr != nil {
					appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: 0, Kind: "signature_retry_request_error", Message: sanitizeUpstreamErrorMessage(retryErr.Error())})
					logger.LegacyPrintf("service.antigravity_gateway", "Antigravity account %d: signature retry request failed (%s): %v", account.ID, stage.name, retryErr)
					continue
				}
				retryResp := retryResult.resp
				if retryResp.StatusCode < 400 {
					_ = resp.Body.Close()
					resp = retryResp
					respBody = nil
					break
				}
				retryBody, _ := io.ReadAll(io.LimitReader(retryResp.Body, 8<<10))
				_ = retryResp.Body.Close()
				if retryResp.StatusCode == http.StatusTooManyRequests {
					retryBaseURL := ""
					if retryResp.Request != nil && retryResp.Request.URL != nil {
						retryBaseURL = retryResp.Request.URL.Scheme + "://" + retryResp.Request.URL.Host
					}
					logger.LegacyPrintf("service.antigravity_gateway", "%s status=429 rate_limited base_url=%s retry_stage=%s body=%s", prefix, retryBaseURL, stage.name, truncateForLog(retryBody, 200))
				}
				kind := "signature_retry"
				if strings.TrimSpace(stage.name) != "" {
					kind = "signature_retry_" + strings.ReplaceAll(stage.name, "+", "_")
				}
				retryUpstreamMsg := strings.TrimSpace(extractAntigravityErrorMessage(retryBody))
				retryUpstreamMsg = sanitizeUpstreamErrorMessage(retryUpstreamMsg)
				retryUpstreamDetail := ""
				if logBody {
					retryUpstreamDetail = truncateString(string(retryBody), maxBytes)
				}
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: retryResp.StatusCode, UpstreamRequestID: retryResp.Header.Get("x-request-id"), Kind: kind, Message: retryUpstreamMsg, Detail: retryUpstreamDetail})
				if retryResp.StatusCode != http.StatusBadRequest || !isSignatureRelatedError(retryBody) {
					respBody = retryBody
					resp = &http.Response{StatusCode: retryResp.StatusCode, Header: retryResp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(retryBody))}
					break
				}
				respBody = retryBody
				resp = &http.Response{StatusCode: retryResp.StatusCode, Header: retryResp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(retryBody))}
			}
		}
		if resp.StatusCode == http.StatusBadRequest && respBody != nil && !isSignatureRelatedError(respBody) {
			errMsg := strings.TrimSpace(extractAntigravityErrorMessage(respBody))
			if isThinkingBudgetConstraintError(errMsg) && s.settingService.IsBudgetRectifierEnabled(ctx) {
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "budget_constraint_error", Message: errMsg, Detail: s.getUpstreamErrorDetail(respBody)})
				if claudeReq.Thinking == nil || claudeReq.Thinking.Type != "adaptive" {
					retryClaudeReq := claudeReq
					retryClaudeReq.Messages = append([]antigravity.ClaudeMessage(nil), claudeReq.Messages...)
					retryClaudeReq.Thinking = &antigravity.ThinkingConfig{Type: "enabled", BudgetTokens: BudgetRectifyBudgetTokens}
					if retryClaudeReq.MaxTokens < BudgetRectifyMinMaxTokens {
						retryClaudeReq.MaxTokens = BudgetRectifyMaxTokens
					}
					logger.LegacyPrintf("service.antigravity_gateway", "Antigravity account %d: detected budget_tokens constraint error, retrying with rectified budget (budget_tokens=%d, max_tokens=%d)", account.ID, BudgetRectifyBudgetTokens, BudgetRectifyMaxTokens)
					retryGeminiBody, txErr := antigravity.TransformClaudeToGeminiWithOptions(&retryClaudeReq, projectID, mappedModel, transformOpts)
					if txErr == nil {
						retryResult, retryErr := s.antigravityRetryLoop(antigravityRetryLoopParams{ctx: ctx, prefix: prefix, account: account, proxyURL: proxyURL, accessToken: accessToken, action: action, body: retryGeminiBody, c: c, httpUpstream: s.httpUpstream, settingService: s.settingService, accountRepo: s.accountRepo, handleError: s.handleUpstreamError, requestedModel: originalModel, isStickySession: isStickySession, groupID: 0, sessionHash: ""})
						if retryErr == nil {
							retryResp := retryResult.resp
							if retryResp.StatusCode < 400 {
								_ = resp.Body.Close()
								resp = retryResp
								respBody = nil
							} else {
								retryBody, _ := io.ReadAll(io.LimitReader(retryResp.Body, 2<<20))
								_ = retryResp.Body.Close()
								respBody = retryBody
								resp = &http.Response{StatusCode: retryResp.StatusCode, Header: retryResp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(retryBody))}
							}
						} else {
							logger.LegacyPrintf("service.antigravity_gateway", "Antigravity account %d: budget rectifier retry failed: %v", account.ID, retryErr)
						}
					}
				}
			}
		}
		if resp.StatusCode >= 400 {
			if resp.StatusCode == http.StatusBadRequest && isPromptTooLongError(respBody) {
				upstreamMsg := strings.TrimSpace(extractAntigravityErrorMessage(respBody))
				upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
				upstreamDetail := s.getUpstreamErrorDetail(respBody)
				logBody, maxBytes := s.getLogConfig()
				if logBody {
					logger.LegacyPrintf("service.antigravity_gateway", "%s status=400 prompt_too_long=true upstream_message=%q request_id=%s body=%s", prefix, upstreamMsg, resp.Header.Get("x-request-id"), truncateForLog(respBody, maxBytes))
				}
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "prompt_too_long", Message: upstreamMsg, Detail: upstreamDetail})
				return nil, &PromptTooLongError{StatusCode: resp.StatusCode, RequestID: resp.Header.Get("x-request-id"), Body: respBody}
			}
			s.handleUpstreamError(ctx, prefix, account, resp.StatusCode, resp.Header, respBody, originalModel, 0, "", isStickySession)
			if resp.StatusCode == http.StatusBadRequest {
				msg := strings.ToLower(strings.TrimSpace(extractAntigravityErrorMessage(respBody)))
				if isGoogleProjectConfigError(msg) {
					upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractAntigravityErrorMessage(respBody)))
					upstreamDetail := s.getUpstreamErrorDetail(respBody)
					log.Printf("%s status=400 google_config_error failover=true upstream_message=%q account=%d", prefix, upstreamMsg, account.ID)
					appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "failover", Message: upstreamMsg, Detail: upstreamDetail})
					return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, RetryableOnSameAccount: true}
				}
			}
			if s.shouldFailoverUpstreamError(resp.StatusCode) {
				upstreamMsg := strings.TrimSpace(extractAntigravityErrorMessage(respBody))
				upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
				upstreamDetail := s.getUpstreamErrorDetail(respBody)
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "failover", Message: upstreamMsg, Detail: upstreamDetail})
				return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody}
			}
			return nil, s.writeMappedClaudeError(c, account, resp.StatusCode, resp.Header.Get("x-request-id"), respBody)
		}
	}
	requestID := resp.Header.Get("x-request-id")
	if requestID != "" {
		c.Header("x-request-id", requestID)
	}
	var usage *ClaudeUsage
	var firstTokenMs *int
	var clientDisconnect bool
	if claudeReq.Stream {
		streamRes, err := s.handleClaudeStreamingResponse(c, resp, startTime, originalModel)
		if err != nil {
			logger.LegacyPrintf("service.antigravity_gateway", "%s status=stream_error error=%v", prefix, err)
			return nil, err
		}
		usage = streamRes.usage
		firstTokenMs = streamRes.firstTokenMs
		clientDisconnect = streamRes.clientDisconnect
	} else {
		streamRes, err := s.handleClaudeStreamToNonStreaming(c, resp, startTime, originalModel)
		if err != nil {
			logger.LegacyPrintf("service.antigravity_gateway", "%s status=stream_collect_error error=%v", prefix, err)
			return nil, err
		}
		usage = streamRes.usage
		firstTokenMs = streamRes.firstTokenMs
	}
	return &ForwardResult{RequestID: requestID, Usage: *usage, Model: originalModel, UpstreamModel: billingModel, Stream: claudeReq.Stream, Duration: time.Since(startTime), FirstTokenMs: firstTokenMs, ClientDisconnect: clientDisconnect}, nil
}
func (s *AntigravityGatewayService) ForwardGemini(ctx context.Context, c *gin.Context, account *Account, originalModel string, action string, stream bool, body []byte, isStickySession bool) (*ForwardResult, error) {
	startTime := time.Now()
	sessionID := getSessionID(c)
	prefix := logPrefix(sessionID, account.Name)
	if strings.TrimSpace(originalModel) == "" {
		return nil, s.writeGoogleError(c, http.StatusBadRequest, "Missing model in URL")
	}
	if strings.TrimSpace(action) == "" {
		return nil, s.writeGoogleError(c, http.StatusBadRequest, "Missing action in URL")
	}
	if len(body) == 0 {
		return nil, s.writeGoogleError(c, http.StatusBadRequest, "Request body is empty")
	}
	imageSize := s.extractImageSize(body)
	switch action {
	case "generateContent", "streamGenerateContent":
	case "countTokens":
		c.JSON(http.StatusOK, map[string]any{"totalTokens": 0})
		return &ForwardResult{RequestID: "", Usage: ClaudeUsage{}, Model: originalModel, ServiceTier: extractGeminiRequestedServiceTierFromBody(body), Stream: false, Duration: time.Since(startTime), FirstTokenMs: nil}, nil
	default:
		return nil, s.writeGoogleError(c, http.StatusNotFound, "Unsupported action: "+action)
	}
	mappedModel := s.getMappedModel(account, originalModel)
	if mappedModel == "" {
		return nil, s.writeGoogleError(c, http.StatusForbidden, fmt.Sprintf("model %s not in whitelist", originalModel))
	}
	billingModel := mappedModel
	if s.tokenProvider == nil {
		return nil, s.writeGoogleError(c, http.StatusBadGateway, "Antigravity token provider not configured")
	}
	accessToken, err := s.tokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return nil, s.writeGoogleError(c, http.StatusBadGateway, "Failed to get upstream access token")
	}
	projectID := strings.TrimSpace(account.GetCredential("project_id"))
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	injectedBody, err := injectIdentityPatchToGeminiRequest(body)
	if err != nil {
		return nil, s.writeGoogleError(c, http.StatusBadRequest, "Invalid request body")
	}
	if cleanedBody, err := cleanGeminiRequest(injectedBody); err == nil {
		injectedBody = cleanedBody
		logger.LegacyPrintf("service.antigravity_gateway", "[Antigravity] Cleaned request schema in forwarded request for account %s", account.Name)
	} else {
		logger.LegacyPrintf("service.antigravity_gateway", "[Antigravity] Failed to clean schema: %v", err)
	}
	wrappedBody, err := s.wrapV1InternalRequest(projectID, mappedModel, injectedBody)
	if err != nil {
		return nil, s.writeGoogleError(c, http.StatusInternalServerError, "Failed to build upstream request")
	}
	upstreamAction := "streamGenerateContent"
	result, err := s.antigravityRetryLoop(antigravityRetryLoopParams{ctx: ctx, prefix: prefix, account: account, proxyURL: proxyURL, accessToken: accessToken, action: upstreamAction, body: wrappedBody, c: c, httpUpstream: s.httpUpstream, settingService: s.settingService, accountRepo: s.accountRepo, handleError: s.handleUpstreamError, requestedModel: originalModel, isStickySession: isStickySession, groupID: 0, sessionHash: ""})
	if err != nil {
		if switchErr, ok := IsAntigravityAccountSwitchError(err); ok {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: 0,
				Kind:               "failover",
				Message:            "rate_limit_switch",
				Detail:             switchErr.RateLimitedModel,
			})
			return nil, &UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable, ForceCacheBilling: switchErr.IsStickySession}
		}
		if c.Request.Context().Err() != nil {
			return nil, s.writeGoogleError(c, http.StatusBadGateway, "Client disconnected before upstream response")
		}
		return nil, s.writeGoogleError(c, http.StatusBadGateway, "Upstream request failed after retries")
	}
	resp := result.resp
	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()
	signatureRetried := false
	for resp != nil && resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		contentType := resp.Header.Get("Content-Type")
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		if s.settingService != nil && s.settingService.IsModelFallbackEnabled(ctx) && isModelNotFoundError(resp.StatusCode, respBody) {
			fallbackModel := s.settingService.GetFallbackModel(ctx, PlatformAntigravity)
			if fallbackModel != "" && fallbackModel != mappedModel {
				logger.LegacyPrintf("service.antigravity_gateway", "[Antigravity] Model not found (%s), retrying with fallback model %s (account: %s)", mappedModel, fallbackModel, account.Name)
				fallbackWrapped, err := s.wrapV1InternalRequest(projectID, fallbackModel, injectedBody)
				if err == nil {
					fallbackReq, err := antigravity.NewAPIRequest(ctx, upstreamAction, accessToken, fallbackWrapped)
					if err == nil {
						fallbackResp, err := s.httpUpstream.Do(fallbackReq, proxyURL, account.ID, account.Concurrency)
						if err == nil && fallbackResp.StatusCode < 400 {
							_ = resp.Body.Close()
							resp = fallbackResp
						} else if fallbackResp != nil {
							_ = fallbackResp.Body.Close()
						}
					}
				}
			}
		}
		if resp.StatusCode < 400 {
			break
		}
		requestID := resp.Header.Get("x-request-id")
		if requestID != "" {
			c.Header("x-request-id", requestID)
		}
		unwrapped, unwrapErr := s.unwrapV1InternalResponse(respBody)
		unwrappedForOps := unwrapped
		if unwrapErr != nil || len(unwrappedForOps) == 0 {
			unwrappedForOps = respBody
		}
		if resp.StatusCode == http.StatusBadRequest &&
			!signatureRetried &&
			s.settingService != nil &&
			s.settingService.IsSignatureRectifierEnabled(ctx) &&
			isSignatureRelatedError(unwrappedForOps) {
			upstreamMsg := strings.TrimSpace(extractAntigravityErrorMessage(unwrappedForOps))
			upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
			upstreamDetail := s.getUpstreamErrorDetail(unwrappedForOps)
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  requestID,
				Kind:               "signature_error",
				Message:            upstreamMsg,
				Detail:             upstreamDetail,
			})
			rectified, changed, rectErr := rectifyGeminiThoughtSignatures(injectedBody)
			if rectErr == nil && changed {
				retryWrapped, wrapErr := s.wrapV1InternalRequest(projectID, mappedModel, rectified)
				if wrapErr == nil {
					signatureRetried = true
					injectedBody = rectified
					retryResult, retryErr := s.antigravityRetryLoop(antigravityRetryLoopParams{ctx: ctx, prefix: prefix, account: account, proxyURL: proxyURL, accessToken: accessToken, action: upstreamAction, body: retryWrapped, c: c, httpUpstream: s.httpUpstream, settingService: s.settingService, accountRepo: s.accountRepo, handleError: s.handleUpstreamError, requestedModel: originalModel, isStickySession: isStickySession, groupID: 0, sessionHash: ""})
					if retryErr != nil {
						if switchErr, ok := IsAntigravityAccountSwitchError(retryErr); ok {
							appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
								Platform:           account.Platform,
								AccountID:          account.ID,
								AccountName:        account.Name,
								UpstreamStatusCode: 0,
								Kind:               "failover",
								Message:            "rate_limit_switch",
								Detail:             switchErr.RateLimitedModel,
							})
							return nil, &UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable, ForceCacheBilling: switchErr.IsStickySession}
						}
						appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
							Platform:           account.Platform,
							AccountID:          account.ID,
							AccountName:        account.Name,
							UpstreamStatusCode: 0,
							Kind:               "signature_retry_request_error",
							Message:            sanitizeUpstreamErrorMessage(retryErr.Error()),
						})
						if c.Request.Context().Err() != nil {
							return nil, s.writeGoogleError(c, http.StatusBadGateway, "Client disconnected before upstream response")
						}
						return nil, s.writeGoogleError(c, http.StatusBadGateway, "Upstream request failed after retries")
					}
					resp = retryResult.resp
					continue
				}
			}
		}
		s.handleUpstreamError(ctx, prefix, account, resp.StatusCode, resp.Header, respBody, originalModel, 0, "", isStickySession)
		upstreamMsg := strings.TrimSpace(extractAntigravityErrorMessage(unwrappedForOps))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
		upstreamDetail := s.getUpstreamErrorDetail(unwrappedForOps)
		setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)
		if resp.StatusCode == http.StatusBadRequest && isGoogleProjectConfigError(strings.ToLower(upstreamMsg)) {
			log.Printf("%s status=400 google_config_error failover=true upstream_message=%q account=%d", prefix, upstreamMsg, account.ID)
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: requestID, Kind: "failover", Message: upstreamMsg, Detail: upstreamDetail})
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: unwrappedForOps, RetryableOnSameAccount: true}
		}
		if s.shouldFailoverUpstreamError(resp.StatusCode) {
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: requestID, Kind: "failover", Message: upstreamMsg, Detail: upstreamDetail})
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: unwrappedForOps}
		}
		if contentType == "" {
			contentType = "application/json"
		}
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: requestID, Kind: "http_error", Message: upstreamMsg, Detail: upstreamDetail})
		logger.LegacyPrintf("service.antigravity_gateway", "[antigravity-Forward] upstream error status=%d body=%s", resp.StatusCode, truncateForLog(unwrappedForOps, 500))
		c.Data(resp.StatusCode, contentType, unwrappedForOps)
		return nil, fmt.Errorf("antigravity upstream error: %d", resp.StatusCode)
	}
	requestID := resp.Header.Get("x-request-id")
	if requestID != "" {
		c.Header("x-request-id", requestID)
	}
	var usage *ClaudeUsage
	var firstTokenMs *int
	var clientDisconnect bool
	if stream {
		streamRes, err := s.handleGeminiStreamingResponse(c, resp, startTime)
		if err != nil {
			logger.LegacyPrintf("service.antigravity_gateway", "%s status=stream_error error=%v", prefix, err)
			return nil, err
		}
		usage = streamRes.usage
		firstTokenMs = streamRes.firstTokenMs
		clientDisconnect = streamRes.clientDisconnect
	} else {
		streamRes, err := s.handleGeminiStreamToNonStreaming(c, resp, startTime)
		if err != nil {
			logger.LegacyPrintf("service.antigravity_gateway", "%s status=stream_collect_error error=%v", prefix, err)
			return nil, err
		}
		usage = streamRes.usage
		firstTokenMs = streamRes.firstTokenMs
	}
	if usage == nil {
		usage = &ClaudeUsage{}
	}
	imageCount := 0
	if isImageGenerationModel(mappedModel) {
		imageCount = 1
	}
	return &ForwardResult{RequestID: requestID, Usage: *usage, Model: originalModel, UpstreamModel: billingModel, ServiceTier: extractGeminiRequestedServiceTierFromBody(body), Stream: stream, Duration: time.Since(startTime), FirstTokenMs: firstTokenMs, ClientDisconnect: clientDisconnect, ImageCount: imageCount, ImageSize: imageSize}, nil
}
func (s *AntigravityGatewayService) ForwardUpstream(ctx context.Context, c *gin.Context, account *Account, body []byte) (*ForwardResult, error) {
	startTime := time.Now()
	sessionID := getSessionID(c)
	prefix := logPrefix(sessionID, account.Name)
	baseURL := strings.TrimSpace(account.GetCredential("base_url"))
	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("upstream account missing base_url or api_key")
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	var claudeReq antigravity.ClaudeRequest
	if err := json.Unmarshal(body, &claudeReq); err != nil {
		return nil, fmt.Errorf("parse claude request: %w", err)
	}
	if strings.TrimSpace(claudeReq.Model) == "" {
		return nil, fmt.Errorf("missing model")
	}
	originalModel := claudeReq.Model
	upstreamURL := baseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create upstream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("x-api-key", apiKey)
	if v := c.GetHeader("anthropic-version"); v != "" {
		req.Header.Set("anthropic-version", v)
	}
	if v := c.GetHeader("anthropic-beta"); v != "" {
		req.Header.Set("anthropic-beta", v)
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		logger.LegacyPrintf("service.antigravity_gateway", "%s upstream request failed: %v", prefix, err)
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if resp.StatusCode == http.StatusTooManyRequests {
			s.handleUpstreamError(ctx, prefix, account, resp.StatusCode, resp.Header, respBody, originalModel, 0, "", false)
		}
		c.Header("Content-Type", resp.Header.Get("Content-Type"))
		c.Status(resp.StatusCode)
		_, _ = c.Writer.Write(respBody)
		return &ForwardResult{Model: originalModel}, nil
	}
	var usage *ClaudeUsage
	var firstTokenMs *int
	var clientDisconnect bool
	if claudeReq.Stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)
		streamRes := s.streamUpstreamResponse(c, resp, startTime)
		usage = streamRes.usage
		firstTokenMs = streamRes.firstTokenMs
		clientDisconnect = streamRes.clientDisconnect
	} else {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read upstream response: %w", err)
		}
		usage = s.extractClaudeUsage(respBody)
		c.Header("Content-Type", resp.Header.Get("Content-Type"))
		c.Status(http.StatusOK)
		_, _ = c.Writer.Write(respBody)
	}
	duration := time.Since(startTime)
	logger.LegacyPrintf("service.antigravity_gateway", "%s status=success duration_ms=%d", prefix, duration.Milliseconds())
	return &ForwardResult{Model: originalModel, Stream: claudeReq.Stream, Duration: duration, FirstTokenMs: firstTokenMs, ClientDisconnect: clientDisconnect, Usage: ClaudeUsage{InputTokens: usage.InputTokens, OutputTokens: usage.OutputTokens, CacheReadInputTokens: usage.CacheReadInputTokens, CacheCreationInputTokens: usage.CacheCreationInputTokens}}, nil
}
func (s *AntigravityGatewayService) streamUpstreamResponse(c *gin.Context, resp *http.Response, startTime time.Time) *antigravityStreamResult {
	usage := &ClaudeUsage{}
	var firstTokenMs *int
	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.settingService.cfg.Gateway.MaxLineSize
	}
	scanner.Buffer(make([]byte, 64*1024), maxLineSize)
	type scanEvent struct {
		line string
		err  error
	}
	events := make(chan scanEvent, 16)
	done := make(chan struct{})
	sendEvent := func(ev scanEvent) bool {
		select {
		case events <- ev:
			return true
		case <-done:
			return false
		}
	}
	var lastReadAt int64
	atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
	go func() {
		defer close(events)
		for scanner.Scan() {
			atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
			if !sendEvent(scanEvent{line: scanner.Text()}) {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			_ = sendEvent(scanEvent{err: err})
		}
	}()
	defer close(done)
	streamInterval := time.Duration(0)
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		streamInterval = time.Duration(s.settingService.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
	}
	var intervalTicker *time.Ticker
	if streamInterval > 0 {
		intervalTicker = time.NewTicker(streamInterval)
		defer intervalTicker.Stop()
	}
	var intervalCh <-chan time.Time
	if intervalTicker != nil {
		intervalCh = intervalTicker.C
	}
	flusher, _ := c.Writer.(http.Flusher)
	cw := newAntigravityClientWriter(c.Writer, flusher, "antigravity upstream")
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: cw.Disconnected()}
			}
			if ev.err != nil {
				if disconnect, handled := handleStreamReadError(ev.err, cw.Disconnected(), "antigravity upstream"); handled {
					return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: disconnect}
				}
				logger.LegacyPrintf("service.antigravity_gateway", "Stream read error (antigravity upstream): %v", ev.err)
				return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs}
			}
			line := ev.line
			if firstTokenMs == nil && len(line) > 0 {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}
			s.extractSSEUsage(line, usage)
			cw.Fprintf("%s\n", line)
		case <-intervalCh:
			lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
			if time.Since(lastRead) < streamInterval {
				continue
			}
			if cw.Disconnected() {
				logger.LegacyPrintf("service.antigravity_gateway", "Upstream timeout after client disconnect (antigravity upstream), returning collected usage")
				return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}
			}
			logger.LegacyPrintf("service.antigravity_gateway", "Stream data interval timeout (antigravity upstream)")
			return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs}
		}
	}
}
