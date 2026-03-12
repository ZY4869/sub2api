package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

func (s *GatewayService) Forward(ctx context.Context, c *gin.Context, account *Account, parsed *ParsedRequest) (*ForwardResult, error) {
	startTime := time.Now()
	if parsed == nil {
		return nil, fmt.Errorf("parse request: empty request")
	}
	if account != nil && account.IsAnthropicAPIKeyPassthroughEnabled() {
		passthroughBody := parsed.Body
		passthroughModel := parsed.Model
		if passthroughModel != "" {
			if mappedModel := account.GetMappedModel(passthroughModel); mappedModel != passthroughModel {
				passthroughBody = s.replaceModelInBody(passthroughBody, mappedModel)
				logger.LegacyPrintf("service.gateway", "Passthrough model mapping: %s -> %s (account: %s)", parsed.Model, mappedModel, account.Name)
				passthroughModel = mappedModel
			}
		}
		return s.forwardAnthropicAPIKeyPassthrough(ctx, c, account, passthroughBody, passthroughModel, parsed.Stream, startTime)
	}
	if account.Platform == PlatformAnthropic && c != nil {
		policy := s.evaluateBetaPolicy(ctx, c.GetHeader("anthropic-beta"), account)
		if policy.blockErr != nil {
			return nil, policy.blockErr
		}
		filterSet := policy.filterSet
		if filterSet == nil {
			filterSet = map[string]struct{}{}
		}
		c.Set(betaPolicyFilterSetKey, filterSet)
	}
	body := parsed.Body
	reqModel := parsed.Model
	reqStream := parsed.Stream
	originalModel := reqModel
	isClaudeCode := isClaudeCodeRequest(ctx, c, parsed)
	shouldMimicClaudeCode := account.IsOAuth() && !isClaudeCode
	if shouldMimicClaudeCode {
		if !strings.Contains(strings.ToLower(reqModel), "haiku") && !systemIncludesClaudeCodePrompt(parsed.System) {
			body = injectClaudeCodePrompt(body, parsed.System)
		}
		normalizeOpts := claudeOAuthNormalizeOptions{stripSystemCacheControl: true}
		if s.identityService != nil {
			fp, err := s.identityService.GetOrCreateFingerprint(ctx, account.ID, c.Request.Header)
			if err == nil && fp != nil {
				if metadataUserID := s.buildOAuthMetadataUserID(parsed, account, fp); metadataUserID != "" {
					normalizeOpts.injectMetadata = true
					normalizeOpts.metadataUserID = metadataUserID
				}
			}
		}
		body, reqModel = normalizeClaudeOAuthRequestBody(body, reqModel, normalizeOpts)
	}
	if account.IsOAuth() {
		body = filterSystemBlocksByPrefix(body)
	}
	body = enforceCacheControlLimit(body)
	mappedModel := reqModel
	mappingSource := ""
	if account.Type == AccountTypeAPIKey {
		mappedModel = account.GetMappedModel(reqModel)
		if mappedModel != reqModel {
			mappingSource = "account"
		}
	}
	if mappingSource == "" && account.Platform == PlatformAnthropic && account.Type != AccountTypeAPIKey {
		normalized := claude.NormalizeModelID(reqModel)
		if normalized != reqModel {
			mappedModel = normalized
			mappingSource = "prefix"
		}
	}
	if mappedModel != reqModel {
		body = s.replaceModelInBody(body, mappedModel)
		reqModel = mappedModel
		logger.LegacyPrintf("service.gateway", "Model mapping applied: %s -> %s (account: %s, source=%s)", originalModel, mappedModel, account.Name, mappingSource)
	}
	token, tokenType, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	logger.LegacyPrintf("service.gateway", "[Forward] Using account: ID=%d Name=%s Platform=%s Type=%s TLSFingerprint=%v Proxy=%s", account.ID, account.Name, account.Platform, account.Type, account.IsTLSFingerprintEnabled(), proxyURL)
	setOpsUpstreamRequestBody(c, body)
	var resp *http.Response
	retryStart := time.Now()
	for attempt := 1; attempt <= maxRetryAttempts; attempt++ {
		upstreamReq, err := s.buildUpstreamRequest(ctx, c, account, body, token, tokenType, reqModel, reqStream, shouldMimicClaudeCode)
		if err != nil {
			return nil, err
		}
		resp, err = s.httpUpstream.DoWithTLS(upstreamReq, proxyURL, account.ID, account.Concurrency, account.IsTLSFingerprintEnabled())
		if err != nil {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			safeErr := sanitizeUpstreamErrorMessage(err.Error())
			setOpsUpstreamError(c, 0, safeErr, "")
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: 0, Kind: "request_error", Message: safeErr})
			c.JSON(http.StatusBadGateway, gin.H{"type": "error", "error": gin.H{"type": "upstream_error", "message": "Upstream request failed"}})
			return nil, fmt.Errorf("upstream request failed: %s", safeErr)
		}
		if resp.StatusCode == 400 {
			respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			if readErr == nil {
				_ = resp.Body.Close()
				if s.isThinkingBlockSignatureError(respBody) && s.settingService.IsSignatureRectifierEnabled(ctx) {
					appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "signature_error", Message: extractUpstreamErrorMessage(respBody), Detail: func() string {
						if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
							return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
						}
						return ""
					}()})
					looksLikeToolSignatureError := func(msg string) bool {
						m := strings.ToLower(msg)
						return strings.Contains(m, "tool_use") || strings.Contains(m, "tool_result") || strings.Contains(m, "functioncall") || strings.Contains(m, "function_call") || strings.Contains(m, "functionresponse") || strings.Contains(m, "function_response")
					}
					if time.Since(retryStart) >= maxRetryElapsed {
						resp.Body = io.NopCloser(bytes.NewReader(respBody))
						break
					}
					logger.LegacyPrintf("service.gateway", "Account %d: detected thinking block signature error, retrying with filtered thinking blocks", account.ID)
					filteredBody := FilterThinkingBlocksForRetry(body)
					retryReq, buildErr := s.buildUpstreamRequest(ctx, c, account, filteredBody, token, tokenType, reqModel, reqStream, shouldMimicClaudeCode)
					if buildErr == nil {
						retryResp, retryErr := s.httpUpstream.DoWithTLS(retryReq, proxyURL, account.ID, account.Concurrency, account.IsTLSFingerprintEnabled())
						if retryErr == nil {
							if retryResp.StatusCode < 400 {
								logger.LegacyPrintf("service.gateway", "Account %d: signature error retry succeeded (thinking downgraded)", account.ID)
								resp = retryResp
								break
							}
							retryRespBody, retryReadErr := io.ReadAll(io.LimitReader(retryResp.Body, 2<<20))
							_ = retryResp.Body.Close()
							if retryReadErr == nil && retryResp.StatusCode == 400 && s.isThinkingBlockSignatureError(retryRespBody) {
								appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: retryResp.StatusCode, UpstreamRequestID: retryResp.Header.Get("x-request-id"), Kind: "signature_retry_thinking", Message: extractUpstreamErrorMessage(retryRespBody), Detail: func() string {
									if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
										return truncateString(string(retryRespBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
									}
									return ""
								}()})
								msg2 := extractUpstreamErrorMessage(retryRespBody)
								if looksLikeToolSignatureError(msg2) && time.Since(retryStart) < maxRetryElapsed {
									logger.LegacyPrintf("service.gateway", "Account %d: signature retry still failing and looks tool-related, retrying with tool blocks downgraded", account.ID)
									filteredBody2 := FilterSignatureSensitiveBlocksForRetry(body)
									retryReq2, buildErr2 := s.buildUpstreamRequest(ctx, c, account, filteredBody2, token, tokenType, reqModel, reqStream, shouldMimicClaudeCode)
									if buildErr2 == nil {
										retryResp2, retryErr2 := s.httpUpstream.DoWithTLS(retryReq2, proxyURL, account.ID, account.Concurrency, account.IsTLSFingerprintEnabled())
										if retryErr2 == nil {
											resp = retryResp2
											break
										}
										if retryResp2 != nil && retryResp2.Body != nil {
											_ = retryResp2.Body.Close()
										}
										appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: 0, Kind: "signature_retry_tools_request_error", Message: sanitizeUpstreamErrorMessage(retryErr2.Error())})
										logger.LegacyPrintf("service.gateway", "Account %d: tool-downgrade signature retry failed: %v", account.ID, retryErr2)
									} else {
										logger.LegacyPrintf("service.gateway", "Account %d: tool-downgrade signature retry build failed: %v", account.ID, buildErr2)
									}
								}
							}
							resp = &http.Response{StatusCode: retryResp.StatusCode, Header: retryResp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(retryRespBody))}
							break
						}
						if retryResp != nil && retryResp.Body != nil {
							_ = retryResp.Body.Close()
						}
						logger.LegacyPrintf("service.gateway", "Account %d: signature error retry failed: %v", account.ID, retryErr)
					} else {
						logger.LegacyPrintf("service.gateway", "Account %d: signature error retry build request failed: %v", account.ID, buildErr)
					}
					resp.Body = io.NopCloser(bytes.NewReader(respBody))
					break
				}
				errMsg := extractUpstreamErrorMessage(respBody)
				if isThinkingBudgetConstraintError(errMsg) && s.settingService.IsBudgetRectifierEnabled(ctx) {
					appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "budget_constraint_error", Message: errMsg, Detail: func() string {
						if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
							return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
						}
						return ""
					}()})
					rectifiedBody, applied := RectifyThinkingBudget(body)
					if applied && time.Since(retryStart) < maxRetryElapsed {
						logger.LegacyPrintf("service.gateway", "Account %d: detected budget_tokens constraint error, retrying with rectified budget (budget_tokens=%d, max_tokens=%d)", account.ID, BudgetRectifyBudgetTokens, BudgetRectifyMaxTokens)
						budgetRetryReq, buildErr := s.buildUpstreamRequest(ctx, c, account, rectifiedBody, token, tokenType, reqModel, reqStream, shouldMimicClaudeCode)
						if buildErr == nil {
							budgetRetryResp, retryErr := s.httpUpstream.DoWithTLS(budgetRetryReq, proxyURL, account.ID, account.Concurrency, account.IsTLSFingerprintEnabled())
							if retryErr == nil {
								resp = budgetRetryResp
								break
							}
							if budgetRetryResp != nil && budgetRetryResp.Body != nil {
								_ = budgetRetryResp.Body.Close()
							}
							logger.LegacyPrintf("service.gateway", "Account %d: budget rectifier retry failed: %v", account.ID, retryErr)
						} else {
							logger.LegacyPrintf("service.gateway", "Account %d: budget rectifier retry build failed: %v", account.ID, buildErr)
						}
					}
				}
				resp.Body = io.NopCloser(bytes.NewReader(respBody))
			}
		}
		if resp.StatusCode >= 400 && resp.StatusCode != 400 && s.shouldRetryUpstreamError(account, resp.StatusCode) {
			if attempt < maxRetryAttempts {
				elapsed := time.Since(retryStart)
				if elapsed >= maxRetryElapsed {
					break
				}
				delay := retryBackoffDelay(attempt)
				remaining := maxRetryElapsed - elapsed
				if delay > remaining {
					delay = remaining
				}
				if delay <= 0 {
					break
				}
				respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
				_ = resp.Body.Close()
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "retry", Message: extractUpstreamErrorMessage(respBody), Detail: func() string {
					if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
						return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
					}
					return ""
				}()})
				logger.LegacyPrintf("service.gateway", "Account %d: upstream error %d, retry %d/%d after %v (elapsed=%v/%v)", account.ID, resp.StatusCode, attempt, maxRetryAttempts, delay, elapsed, maxRetryElapsed)
				if err := sleepWithContext(ctx, delay); err != nil {
					return nil, err
				}
				continue
			}
			break
		}
		if account.Platform == PlatformGemini && resp.StatusCode < 400 && s.cfg != nil && s.cfg.Gateway.GeminiDebugResponseHeaders {
			logger.LegacyPrintf("service.gateway", "[DEBUG] Gemini API Response Headers for account %d:", account.ID)
			for k, v := range resp.Header {
				logger.LegacyPrintf("service.gateway", "[DEBUG]   %s: %v", k, v)
			}
		}
		break
	}
	if resp == nil || resp.Body == nil {
		return nil, errors.New("upstream request failed: empty response")
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 400 && s.shouldRetryUpstreamError(account, resp.StatusCode) {
		if s.shouldFailoverUpstreamError(resp.StatusCode) {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			_ = resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
			logger.LegacyPrintf("service.gateway", "[Forward] Upstream error (retry exhausted, failover): Account=%d(%s) Status=%d RequestID=%s Body=%s", account.ID, account.Name, resp.StatusCode, resp.Header.Get("x-request-id"), truncateString(string(respBody), 1000))
			s.handleRetryExhaustedSideEffects(ctx, resp, account)
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "retry_exhausted_failover", Message: extractUpstreamErrorMessage(respBody), Detail: func() string {
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
				}
				return ""
			}()})
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, RetryableOnSameAccount: account.IsPoolMode() && isPoolModeRetryableStatus(resp.StatusCode)}
		}
		return s.handleRetryExhaustedError(ctx, resp, c, account)
	}
	if resp.StatusCode >= 400 && s.shouldFailoverUpstreamError(resp.StatusCode) {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		logger.LegacyPrintf("service.gateway", "[Forward] Upstream error (failover): Account=%d(%s) Status=%d RequestID=%s Body=%s", account.ID, account.Name, resp.StatusCode, resp.Header.Get("x-request-id"), truncateString(string(respBody), 1000))
		s.handleFailoverSideEffects(ctx, resp, account)
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "failover", Message: extractUpstreamErrorMessage(respBody), Detail: func() string {
			if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
				return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
			}
			return ""
		}()})
		return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, RetryableOnSameAccount: account.IsPoolMode() && isPoolModeRetryableStatus(resp.StatusCode)}
	}
	if resp.StatusCode >= 400 {
		if resp.StatusCode == 400 && s.cfg != nil && s.cfg.Gateway.FailoverOn400 {
			respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			if readErr != nil {
				return s.handleErrorResponse(ctx, resp, c, account)
			}
			_ = resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
			if s.shouldFailoverOn400(respBody) {
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
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "failover_on_400", Message: upstreamMsg, Detail: upstreamDetail})
				if s.cfg.Gateway.LogUpstreamErrorBody {
					logger.LegacyPrintf("service.gateway", "Account %d: 400 error, attempting failover: %s", account.ID, truncateForLog(respBody, s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes))
				} else {
					logger.LegacyPrintf("service.gateway", "Account %d: 400 error, attempting failover", account.ID)
				}
				s.handleFailoverSideEffects(ctx, resp, account)
				return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody}
			}
		}
		return s.handleErrorResponse(ctx, resp, c, account)
	}
	if parsed.OnUpstreamAccepted != nil {
		parsed.OnUpstreamAccepted()
	}
	var usage *ClaudeUsage
	var firstTokenMs *int
	var clientDisconnect bool
	if reqStream {
		streamResult, err := s.handleStreamingResponse(ctx, resp, c, account, startTime, originalModel, reqModel, shouldMimicClaudeCode)
		if err != nil {
			if err.Error() == "have error in stream" {
				return nil, &UpstreamFailoverError{StatusCode: 403}
			}
			return nil, err
		}
		usage = streamResult.usage
		firstTokenMs = streamResult.firstTokenMs
		clientDisconnect = streamResult.clientDisconnect
	} else {
		usage, err = s.handleNonStreamingResponse(ctx, resp, c, account, originalModel, reqModel)
		if err != nil {
			return nil, err
		}
	}
	return &ForwardResult{RequestID: resp.Header.Get("x-request-id"), Usage: *usage, Model: originalModel, Stream: reqStream, Duration: time.Since(startTime), FirstTokenMs: firstTokenMs, ClientDisconnect: clientDisconnect}, nil
}
func (s *GatewayService) forwardAnthropicAPIKeyPassthrough(ctx context.Context, c *gin.Context, account *Account, body []byte, reqModel string, reqStream bool, startTime time.Time) (*ForwardResult, error) {
	token, tokenType, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	if tokenType != "apikey" {
		return nil, fmt.Errorf("anthropic api key passthrough requires apikey token, got: %s", tokenType)
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	logger.LegacyPrintf("service.gateway", "[Anthropic 自动透传] 命中 API Key 透传分支: account=%d name=%s model=%s stream=%v", account.ID, account.Name, reqModel, reqStream)
	if c != nil {
		c.Set("anthropic_passthrough", true)
	}
	setOpsUpstreamRequestBody(c, body)
	var resp *http.Response
	retryStart := time.Now()
	for attempt := 1; attempt <= maxRetryAttempts; attempt++ {
		upstreamReq, err := s.buildUpstreamRequestAnthropicAPIKeyPassthrough(ctx, c, account, body, token)
		if err != nil {
			return nil, err
		}
		resp, err = s.httpUpstream.DoWithTLS(upstreamReq, proxyURL, account.ID, account.Concurrency, account.IsTLSFingerprintEnabled())
		if err != nil {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			safeErr := sanitizeUpstreamErrorMessage(err.Error())
			setOpsUpstreamError(c, 0, safeErr, "")
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: 0, Passthrough: true, Kind: "request_error", Message: safeErr})
			c.JSON(http.StatusBadGateway, gin.H{"type": "error", "error": gin.H{"type": "upstream_error", "message": "Upstream request failed"}})
			return nil, fmt.Errorf("upstream request failed: %s", safeErr)
		}
		if resp.StatusCode >= 400 && resp.StatusCode != 400 && s.shouldRetryUpstreamError(account, resp.StatusCode) {
			if attempt < maxRetryAttempts {
				elapsed := time.Since(retryStart)
				if elapsed >= maxRetryElapsed {
					break
				}
				delay := retryBackoffDelay(attempt)
				remaining := maxRetryElapsed - elapsed
				if delay > remaining {
					delay = remaining
				}
				if delay <= 0 {
					break
				}
				respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
				_ = resp.Body.Close()
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Passthrough: true, Kind: "retry", Message: extractUpstreamErrorMessage(respBody), Detail: func() string {
					if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
						return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
					}
					return ""
				}()})
				logger.LegacyPrintf("service.gateway", "Anthropic passthrough account %d: upstream error %d, retry %d/%d after %v (elapsed=%v/%v)", account.ID, resp.StatusCode, attempt, maxRetryAttempts, delay, elapsed, maxRetryElapsed)
				if err := sleepWithContext(ctx, delay); err != nil {
					return nil, err
				}
				continue
			}
			break
		}
		break
	}
	if resp == nil || resp.Body == nil {
		return nil, errors.New("upstream request failed: empty response")
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode >= 400 && s.shouldRetryUpstreamError(account, resp.StatusCode) {
		if s.shouldFailoverUpstreamError(resp.StatusCode) {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			_ = resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
			logger.LegacyPrintf("service.gateway", "[Anthropic Passthrough] Upstream error (retry exhausted, failover): Account=%d(%s) Status=%d RequestID=%s Body=%s", account.ID, account.Name, resp.StatusCode, resp.Header.Get("x-request-id"), truncateString(string(respBody), 1000))
			s.handleRetryExhaustedSideEffects(ctx, resp, account)
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Passthrough: true, Kind: "retry_exhausted_failover", Message: extractUpstreamErrorMessage(respBody), Detail: func() string {
				if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
					return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
				}
				return ""
			}()})
			return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, RetryableOnSameAccount: account.IsPoolMode() && isPoolModeRetryableStatus(resp.StatusCode)}
		}
		return s.handleRetryExhaustedError(ctx, resp, c, account)
	}
	if resp.StatusCode >= 400 && s.shouldFailoverUpstreamError(resp.StatusCode) {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		logger.LegacyPrintf("service.gateway", "[Anthropic Passthrough] Upstream error (failover): Account=%d(%s) Status=%d RequestID=%s Body=%s", account.ID, account.Name, resp.StatusCode, resp.Header.Get("x-request-id"), truncateString(string(respBody), 1000))
		s.handleFailoverSideEffects(ctx, resp, account)
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: account.Platform, AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Passthrough: true, Kind: "failover", Message: extractUpstreamErrorMessage(respBody), Detail: func() string {
			if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
				return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
			}
			return ""
		}()})
		return nil, &UpstreamFailoverError{StatusCode: resp.StatusCode, ResponseBody: respBody, RetryableOnSameAccount: account.IsPoolMode() && isPoolModeRetryableStatus(resp.StatusCode)}
	}
	if resp.StatusCode >= 400 {
		return s.handleErrorResponse(ctx, resp, c, account)
	}
	var usage *ClaudeUsage
	var firstTokenMs *int
	var clientDisconnect bool
	if reqStream {
		streamResult, err := s.handleStreamingResponseAnthropicAPIKeyPassthrough(ctx, resp, c, account, startTime, reqModel)
		if err != nil {
			return nil, err
		}
		usage = streamResult.usage
		firstTokenMs = streamResult.firstTokenMs
		clientDisconnect = streamResult.clientDisconnect
	} else {
		usage, err = s.handleNonStreamingResponseAnthropicAPIKeyPassthrough(ctx, resp, c, account)
		if err != nil {
			return nil, err
		}
	}
	if usage == nil {
		usage = &ClaudeUsage{}
	}
	return &ForwardResult{RequestID: resp.Header.Get("x-request-id"), Usage: *usage, Model: reqModel, Stream: reqStream, Duration: time.Since(startTime), FirstTokenMs: firstTokenMs, ClientDisconnect: clientDisconnect}, nil
}
func (s *GatewayService) buildUpstreamRequestAnthropicAPIKeyPassthrough(ctx context.Context, c *gin.Context, account *Account, body []byte, token string) (*http.Request, error) {
	targetURL := claudeAPIURL
	baseURL := account.GetBaseURL()
	if baseURL != "" {
		validatedURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		targetURL = validatedURL + "/v1/messages?beta=true"
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
func (s *GatewayService) handleStreamingResponseAnthropicAPIKeyPassthrough(ctx context.Context, resp *http.Response, c *gin.Context, account *Account, startTime time.Time, model string) (*streamingResult, error) {
	if s.rateLimitService != nil {
		s.rateLimitService.UpdateSessionWindow(ctx, account, resp.Header)
	}
	writeAnthropicPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "text/event-stream"
	}
	c.Header("Content-Type", contentType)
	if c.Writer.Header().Get("Cache-Control") == "" {
		c.Header("Cache-Control", "no-cache")
	}
	if c.Writer.Header().Get("Connection") == "" {
		c.Header("Connection", "keep-alive")
	}
	c.Header("X-Accel-Buffering", "no")
	if v := resp.Header.Get("x-request-id"); v != "" {
		c.Header("x-request-id", v)
	}
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}
	usage := &ClaudeUsage{}
	var firstTokenMs *int
	clientDisconnected := false
	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	scanner.Buffer(scanBuf[:0], maxLineSize)
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
	go func(scanBuf *sseScannerBuf64K) {
		defer putSSEScannerBuf64K(scanBuf)
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
	}(scanBuf)
	defer close(done)
	streamInterval := time.Duration(0)
	if s.cfg != nil && s.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		streamInterval = time.Duration(s.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
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
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				if !clientDisconnected {
					flusher.Flush()
				}
				return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: clientDisconnected}, nil
			}
			if ev.err != nil {
				if clientDisconnected {
					logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] Upstream read error after client disconnect: account=%d err=%v", account.ID, ev.err)
					return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}, nil
				}
				if errors.Is(ev.err, context.Canceled) || errors.Is(ev.err, context.DeadlineExceeded) {
					logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] 流读取被取消: account=%d request_id=%s err=%v ctx_err=%v", account.ID, resp.Header.Get("x-request-id"), ev.err, ctx.Err())
					return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}, nil
				}
				if errors.Is(ev.err, bufio.ErrTooLong) {
					logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] SSE line too long: account=%d max_size=%d error=%v", account.ID, maxLineSize, ev.err)
					return &streamingResult{usage: usage, firstTokenMs: firstTokenMs}, ev.err
				}
				return &streamingResult{usage: usage, firstTokenMs: firstTokenMs}, fmt.Errorf("stream read error: %w", ev.err)
			}
			line := ev.line
			if data, ok := extractAnthropicSSEDataLine(line); ok {
				trimmed := strings.TrimSpace(data)
				if firstTokenMs == nil && trimmed != "" && trimmed != "[DONE]" {
					ms := int(time.Since(startTime).Milliseconds())
					firstTokenMs = &ms
				}
				s.parseSSEUsagePassthrough(data, usage)
			}
			if !clientDisconnected {
				if _, err := io.WriteString(w, line); err != nil {
					clientDisconnected = true
					logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] Client disconnected during streaming, continue draining upstream for usage: account=%d", account.ID)
				} else if _, err := io.WriteString(w, "\n"); err != nil {
					clientDisconnected = true
					logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] Client disconnected during streaming, continue draining upstream for usage: account=%d", account.ID)
				} else if line == "" {
					flusher.Flush()
				}
			}
		case <-intervalCh:
			lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
			if time.Since(lastRead) < streamInterval {
				continue
			}
			if clientDisconnected {
				logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] Upstream timeout after client disconnect: account=%d model=%s", account.ID, model)
				return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}, nil
			}
			logger.LegacyPrintf("service.gateway", "[Anthropic passthrough] Stream data interval timeout: account=%d model=%s interval=%s", account.ID, model, streamInterval)
			if s.rateLimitService != nil {
				s.rateLimitService.HandleStreamTimeout(ctx, account, model)
			}
			return &streamingResult{usage: usage, firstTokenMs: firstTokenMs}, fmt.Errorf("stream data interval timeout")
		}
	}
}
func extractAnthropicSSEDataLine(line string) (string, bool) {
	if !strings.HasPrefix(line, "data:") {
		return "", false
	}
	start := len("data:")
	for start < len(line) {
		if line[start] != ' ' && line[start] != '\t' {
			break
		}
		start++
	}
	return line[start:], true
}
func (s *GatewayService) parseSSEUsagePassthrough(data string, usage *ClaudeUsage) {
	if usage == nil || data == "" || data == "[DONE]" {
		return
	}
	parsed := gjson.Parse(data)
	switch parsed.Get("type").String() {
	case "message_start":
		msgUsage := parsed.Get("message.usage")
		if msgUsage.Exists() {
			usage.InputTokens = int(msgUsage.Get("input_tokens").Int())
			usage.CacheCreationInputTokens = int(msgUsage.Get("cache_creation_input_tokens").Int())
			usage.CacheReadInputTokens = int(msgUsage.Get("cache_read_input_tokens").Int())
			cc5m := msgUsage.Get("cache_creation.ephemeral_5m_input_tokens")
			cc1h := msgUsage.Get("cache_creation.ephemeral_1h_input_tokens")
			if cc5m.Exists() || cc1h.Exists() {
				usage.CacheCreation5mTokens = int(cc5m.Int())
				usage.CacheCreation1hTokens = int(cc1h.Int())
			}
		}
	case "message_delta":
		deltaUsage := parsed.Get("usage")
		if deltaUsage.Exists() {
			if v := deltaUsage.Get("input_tokens").Int(); v > 0 {
				usage.InputTokens = int(v)
			}
			if v := deltaUsage.Get("output_tokens").Int(); v > 0 {
				usage.OutputTokens = int(v)
			}
			if v := deltaUsage.Get("cache_creation_input_tokens").Int(); v > 0 {
				usage.CacheCreationInputTokens = int(v)
			}
			if v := deltaUsage.Get("cache_read_input_tokens").Int(); v > 0 {
				usage.CacheReadInputTokens = int(v)
			}
			cc5m := deltaUsage.Get("cache_creation.ephemeral_5m_input_tokens")
			cc1h := deltaUsage.Get("cache_creation.ephemeral_1h_input_tokens")
			if cc5m.Exists() && cc5m.Int() > 0 {
				usage.CacheCreation5mTokens = int(cc5m.Int())
			}
			if cc1h.Exists() && cc1h.Int() > 0 {
				usage.CacheCreation1hTokens = int(cc1h.Int())
			}
		}
	}
	if usage.CacheReadInputTokens == 0 {
		if cached := parsed.Get("message.usage.cached_tokens").Int(); cached > 0 {
			usage.CacheReadInputTokens = int(cached)
		}
		if cached := parsed.Get("usage.cached_tokens").Int(); usage.CacheReadInputTokens == 0 && cached > 0 {
			usage.CacheReadInputTokens = int(cached)
		}
	}
	if usage.CacheCreationInputTokens == 0 {
		cc5m := parsed.Get("message.usage.cache_creation.ephemeral_5m_input_tokens").Int()
		cc1h := parsed.Get("message.usage.cache_creation.ephemeral_1h_input_tokens").Int()
		if cc5m == 0 && cc1h == 0 {
			cc5m = parsed.Get("usage.cache_creation.ephemeral_5m_input_tokens").Int()
			cc1h = parsed.Get("usage.cache_creation.ephemeral_1h_input_tokens").Int()
		}
		total := cc5m + cc1h
		if total > 0 {
			usage.CacheCreationInputTokens = int(total)
		}
	}
}
func parseClaudeUsageFromResponseBody(body []byte) *ClaudeUsage {
	usage := &ClaudeUsage{}
	if len(body) == 0 {
		return usage
	}
	parsed := gjson.ParseBytes(body)
	usageNode := parsed.Get("usage")
	if !usageNode.Exists() {
		return usage
	}
	usage.InputTokens = int(usageNode.Get("input_tokens").Int())
	usage.OutputTokens = int(usageNode.Get("output_tokens").Int())
	usage.CacheCreationInputTokens = int(usageNode.Get("cache_creation_input_tokens").Int())
	usage.CacheReadInputTokens = int(usageNode.Get("cache_read_input_tokens").Int())
	cc5m := usageNode.Get("cache_creation.ephemeral_5m_input_tokens").Int()
	cc1h := usageNode.Get("cache_creation.ephemeral_1h_input_tokens").Int()
	if cc5m > 0 || cc1h > 0 {
		usage.CacheCreation5mTokens = int(cc5m)
		usage.CacheCreation1hTokens = int(cc1h)
	}
	if usage.CacheCreationInputTokens == 0 && (cc5m > 0 || cc1h > 0) {
		usage.CacheCreationInputTokens = int(cc5m + cc1h)
	}
	if usage.CacheReadInputTokens == 0 {
		if cached := usageNode.Get("cached_tokens").Int(); cached > 0 {
			usage.CacheReadInputTokens = int(cached)
		}
	}
	return usage
}
func (s *GatewayService) handleNonStreamingResponseAnthropicAPIKeyPassthrough(ctx context.Context, resp *http.Response, c *gin.Context, account *Account) (*ClaudeUsage, error) {
	if s.rateLimitService != nil {
		s.rateLimitService.UpdateSessionWindow(ctx, account, resp.Header)
	}
	maxBytes := resolveUpstreamResponseReadLimit(s.cfg)
	body, err := readUpstreamResponseBodyLimited(resp.Body, maxBytes)
	if err != nil {
		if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
			setOpsUpstreamError(c, http.StatusBadGateway, "upstream response too large", "")
			c.JSON(http.StatusBadGateway, gin.H{"type": "error", "error": gin.H{"type": "upstream_error", "message": "Upstream response too large"}})
		}
		return nil, err
	}
	usage := parseClaudeUsageFromResponseBody(body)
	writeAnthropicPassthroughResponseHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(resp.StatusCode, contentType, body)
	return usage, nil
}
func writeAnthropicPassthroughResponseHeaders(dst http.Header, src http.Header, filter *responseheaders.CompiledHeaderFilter) {
	if dst == nil || src == nil {
		return
	}
	if filter != nil {
		responseheaders.WriteFilteredHeaders(dst, src, filter)
		return
	}
	if v := strings.TrimSpace(src.Get("Content-Type")); v != "" {
		dst.Set("Content-Type", v)
	}
	if v := strings.TrimSpace(src.Get("x-request-id")); v != "" {
		dst.Set("x-request-id", v)
	}
}
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

type streamingResult struct {
	usage            *ClaudeUsage
	firstTokenMs     *int
	clientDisconnect bool
}

func (s *GatewayService) handleStreamingResponse(ctx context.Context, resp *http.Response, c *gin.Context, account *Account, startTime time.Time, originalModel, mappedModel string, mimicClaudeCode bool) (*streamingResult, error) {
	s.rateLimitService.UpdateSessionWindow(ctx, account, resp.Header)
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
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("streaming not supported")
	}
	usage := &ClaudeUsage{}
	var firstTokenMs *int
	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	scanner.Buffer(scanBuf[:0], maxLineSize)
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
	go func(scanBuf *sseScannerBuf64K) {
		defer putSSEScannerBuf64K(scanBuf)
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
	}(scanBuf)
	defer close(done)
	streamInterval := time.Duration(0)
	if s.cfg != nil && s.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		streamInterval = time.Duration(s.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
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
	keepaliveInterval := time.Duration(0)
	if s.cfg != nil && s.cfg.Gateway.StreamKeepaliveInterval > 0 {
		keepaliveInterval = time.Duration(s.cfg.Gateway.StreamKeepaliveInterval) * time.Second
	}
	var keepaliveTicker *time.Ticker
	if keepaliveInterval > 0 {
		keepaliveTicker = time.NewTicker(keepaliveInterval)
		defer keepaliveTicker.Stop()
	}
	var keepaliveCh <-chan time.Time
	if keepaliveTicker != nil {
		keepaliveCh = keepaliveTicker.C
	}
	lastDataAt := time.Now()
	errorEventSent := false
	sendErrorEvent := func(reason string) {
		if errorEventSent {
			return
		}
		errorEventSent = true
		_, _ = fmt.Fprintf(w, "event: error\ndata: {\"error\":\"%s\"}\n\n", reason)
		flusher.Flush()
	}
	needModelReplace := originalModel != mappedModel
	clientDisconnected := false
	pendingEventLines := make([]string, 0, 4)
	processSSEEvent := func(lines []string) ([]string, string, *sseUsagePatch, error) {
		if len(lines) == 0 {
			return nil, "", nil, nil
		}
		eventName := ""
		dataLine := ""
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "event:") {
				eventName = strings.TrimSpace(strings.TrimPrefix(trimmed, "event:"))
				continue
			}
			if dataLine == "" && sseDataRe.MatchString(trimmed) {
				dataLine = sseDataRe.ReplaceAllString(trimmed, "")
			}
		}
		if eventName == "error" {
			return nil, dataLine, nil, errors.New("have error in stream")
		}
		if dataLine == "" {
			return []string{strings.Join(lines, "\n") + "\n\n"}, "", nil, nil
		}
		if dataLine == "[DONE]" {
			block := ""
			if eventName != "" {
				block = "event: " + eventName + "\n"
			}
			block += "data: " + dataLine + "\n\n"
			return []string{block}, dataLine, nil, nil
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(dataLine), &event); err != nil {
			block := ""
			if eventName != "" {
				block = "event: " + eventName + "\n"
			}
			block += "data: " + dataLine + "\n\n"
			return []string{block}, dataLine, nil, nil
		}
		eventType, _ := event["type"].(string)
		if eventName == "" {
			eventName = eventType
		}
		eventChanged := false
		if eventType == "message_start" {
			if msg, ok := event["message"].(map[string]any); ok {
				if u, ok := msg["usage"].(map[string]any); ok {
					eventChanged = reconcileCachedTokens(u) || eventChanged
				}
			}
		}
		if eventType == "message_delta" {
			if u, ok := event["usage"].(map[string]any); ok {
				eventChanged = reconcileCachedTokens(u) || eventChanged
			}
		}
		if account.IsCacheTTLOverrideEnabled() {
			overrideTarget := account.GetCacheTTLOverrideTarget()
			if eventType == "message_start" {
				if msg, ok := event["message"].(map[string]any); ok {
					if u, ok := msg["usage"].(map[string]any); ok {
						eventChanged = rewriteCacheCreationJSON(u, overrideTarget) || eventChanged
					}
				}
			}
			if eventType == "message_delta" {
				if u, ok := event["usage"].(map[string]any); ok {
					eventChanged = rewriteCacheCreationJSON(u, overrideTarget) || eventChanged
				}
			}
		}
		if needModelReplace {
			if msg, ok := event["message"].(map[string]any); ok {
				if model, ok := msg["model"].(string); ok && model == mappedModel {
					msg["model"] = originalModel
					eventChanged = true
				}
			}
		}
		usagePatch := s.extractSSEUsagePatch(event)
		if !eventChanged {
			block := ""
			if eventName != "" {
				block = "event: " + eventName + "\n"
			}
			block += "data: " + dataLine + "\n\n"
			return []string{block}, dataLine, usagePatch, nil
		}
		newData, err := json.Marshal(event)
		if err != nil {
			block := ""
			if eventName != "" {
				block = "event: " + eventName + "\n"
			}
			block += "data: " + dataLine + "\n\n"
			return []string{block}, dataLine, usagePatch, nil
		}
		block := ""
		if eventName != "" {
			block = "event: " + eventName + "\n"
		}
		block += "data: " + string(newData) + "\n\n"
		return []string{block}, string(newData), usagePatch, nil
	}
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: clientDisconnected}, nil
			}
			if ev.err != nil {
				if errors.Is(ev.err, context.Canceled) || errors.Is(ev.err, context.DeadlineExceeded) {
					logger.LegacyPrintf("service.gateway", "Context canceled during streaming, returning collected usage")
					return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}, nil
				}
				if clientDisconnected {
					logger.LegacyPrintf("service.gateway", "Upstream read error after client disconnect: %v, returning collected usage", ev.err)
					return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}, nil
				}
				if errors.Is(ev.err, bufio.ErrTooLong) {
					logger.LegacyPrintf("service.gateway", "SSE line too long: account=%d max_size=%d error=%v", account.ID, maxLineSize, ev.err)
					sendErrorEvent("response_too_large")
					return &streamingResult{usage: usage, firstTokenMs: firstTokenMs}, ev.err
				}
				sendErrorEvent("stream_read_error")
				return &streamingResult{usage: usage, firstTokenMs: firstTokenMs}, fmt.Errorf("stream read error: %w", ev.err)
			}
			line := ev.line
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				if len(pendingEventLines) == 0 {
					continue
				}
				outputBlocks, data, usagePatch, err := processSSEEvent(pendingEventLines)
				pendingEventLines = pendingEventLines[:0]
				if err != nil {
					if clientDisconnected {
						return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}, nil
					}
					return nil, err
				}
				for _, block := range outputBlocks {
					if !clientDisconnected {
						if _, werr := fmt.Fprint(w, block); werr != nil {
							clientDisconnected = true
							logger.LegacyPrintf("service.gateway", "Client disconnected during streaming, continuing to drain upstream for billing")
							break
						}
						flusher.Flush()
						lastDataAt = time.Now()
					}
					if data != "" {
						if firstTokenMs == nil && data != "[DONE]" {
							ms := int(time.Since(startTime).Milliseconds())
							firstTokenMs = &ms
						}
						if usagePatch != nil {
							mergeSSEUsagePatch(usage, usagePatch)
						}
					}
				}
				continue
			}
			pendingEventLines = append(pendingEventLines, line)
		case <-intervalCh:
			lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
			if time.Since(lastRead) < streamInterval {
				continue
			}
			if clientDisconnected {
				logger.LegacyPrintf("service.gateway", "Upstream timeout after client disconnect, returning collected usage")
				return &streamingResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}, nil
			}
			logger.LegacyPrintf("service.gateway", "Stream data interval timeout: account=%d model=%s interval=%s", account.ID, originalModel, streamInterval)
			if s.rateLimitService != nil {
				s.rateLimitService.HandleStreamTimeout(ctx, account, originalModel)
			}
			sendErrorEvent("stream_timeout")
			return &streamingResult{usage: usage, firstTokenMs: firstTokenMs}, fmt.Errorf("stream data interval timeout")
		case <-keepaliveCh:
			if clientDisconnected {
				continue
			}
			if time.Since(lastDataAt) < keepaliveInterval {
				continue
			}
			if _, werr := fmt.Fprint(w, "event: ping\ndata: {\"type\": \"ping\"}\n\n"); werr != nil {
				clientDisconnected = true
				logger.LegacyPrintf("service.gateway", "Client disconnected during keepalive ping, continuing to drain upstream for billing")
				continue
			}
			flusher.Flush()
		}
	}
}
func (s *GatewayService) parseSSEUsage(data string, usage *ClaudeUsage) {
	if usage == nil {
		return
	}
	var event map[string]any
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return
	}
	if patch := s.extractSSEUsagePatch(event); patch != nil {
		mergeSSEUsagePatch(usage, patch)
	}
}

type sseUsagePatch struct {
	inputTokens              int
	hasInputTokens           bool
	outputTokens             int
	hasOutputTokens          bool
	cacheCreationInputTokens int
	hasCacheCreationInput    bool
	cacheReadInputTokens     int
	hasCacheReadInput        bool
	cacheCreation5mTokens    int
	hasCacheCreation5m       bool
	cacheCreation1hTokens    int
	hasCacheCreation1h       bool
}

func (s *GatewayService) extractSSEUsagePatch(event map[string]any) *sseUsagePatch {
	if len(event) == 0 {
		return nil
	}
	eventType, _ := event["type"].(string)
	switch eventType {
	case "message_start":
		msg, _ := event["message"].(map[string]any)
		usageObj, _ := msg["usage"].(map[string]any)
		if len(usageObj) == 0 {
			return nil
		}
		patch := &sseUsagePatch{}
		patch.hasInputTokens = true
		if v, ok := parseSSEUsageInt(usageObj["input_tokens"]); ok {
			patch.inputTokens = v
		}
		patch.hasCacheCreationInput = true
		if v, ok := parseSSEUsageInt(usageObj["cache_creation_input_tokens"]); ok {
			patch.cacheCreationInputTokens = v
		}
		patch.hasCacheReadInput = true
		if v, ok := parseSSEUsageInt(usageObj["cache_read_input_tokens"]); ok {
			patch.cacheReadInputTokens = v
		}
		if cc, ok := usageObj["cache_creation"].(map[string]any); ok {
			if v, exists := parseSSEUsageInt(cc["ephemeral_5m_input_tokens"]); exists {
				patch.cacheCreation5mTokens = v
				patch.hasCacheCreation5m = true
			}
			if v, exists := parseSSEUsageInt(cc["ephemeral_1h_input_tokens"]); exists {
				patch.cacheCreation1hTokens = v
				patch.hasCacheCreation1h = true
			}
		}
		return patch
	case "message_delta":
		usageObj, _ := event["usage"].(map[string]any)
		if len(usageObj) == 0 {
			return nil
		}
		patch := &sseUsagePatch{}
		if v, ok := parseSSEUsageInt(usageObj["input_tokens"]); ok && v > 0 {
			patch.inputTokens = v
			patch.hasInputTokens = true
		}
		if v, ok := parseSSEUsageInt(usageObj["output_tokens"]); ok && v > 0 {
			patch.outputTokens = v
			patch.hasOutputTokens = true
		}
		if v, ok := parseSSEUsageInt(usageObj["cache_creation_input_tokens"]); ok && v > 0 {
			patch.cacheCreationInputTokens = v
			patch.hasCacheCreationInput = true
		}
		if v, ok := parseSSEUsageInt(usageObj["cache_read_input_tokens"]); ok && v > 0 {
			patch.cacheReadInputTokens = v
			patch.hasCacheReadInput = true
		}
		if cc, ok := usageObj["cache_creation"].(map[string]any); ok {
			if v, exists := parseSSEUsageInt(cc["ephemeral_5m_input_tokens"]); exists && v > 0 {
				patch.cacheCreation5mTokens = v
				patch.hasCacheCreation5m = true
			}
			if v, exists := parseSSEUsageInt(cc["ephemeral_1h_input_tokens"]); exists && v > 0 {
				patch.cacheCreation1hTokens = v
				patch.hasCacheCreation1h = true
			}
		}
		return patch
	}
	return nil
}
func mergeSSEUsagePatch(usage *ClaudeUsage, patch *sseUsagePatch) {
	if usage == nil || patch == nil {
		return
	}
	if patch.hasInputTokens {
		usage.InputTokens = patch.inputTokens
	}
	if patch.hasCacheCreationInput {
		usage.CacheCreationInputTokens = patch.cacheCreationInputTokens
	}
	if patch.hasCacheReadInput {
		usage.CacheReadInputTokens = patch.cacheReadInputTokens
	}
	if patch.hasOutputTokens {
		usage.OutputTokens = patch.outputTokens
	}
	if patch.hasCacheCreation5m {
		usage.CacheCreation5mTokens = patch.cacheCreation5mTokens
	}
	if patch.hasCacheCreation1h {
		usage.CacheCreation1hTokens = patch.cacheCreation1hTokens
	}
}
func parseSSEUsageInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case int32:
		return int(v), true
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i), true
		}
		if f, err := v.Float64(); err == nil {
			return int(f), true
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return parsed, true
		}
	}
	return 0, false
}
func applyCacheTTLOverride(usage *ClaudeUsage, target string) bool {
	if usage.CacheCreation5mTokens == 0 && usage.CacheCreation1hTokens == 0 && usage.CacheCreationInputTokens > 0 {
		usage.CacheCreation5mTokens = usage.CacheCreationInputTokens
	}
	total := usage.CacheCreation5mTokens + usage.CacheCreation1hTokens
	if total == 0 {
		return false
	}
	switch target {
	case "1h":
		if usage.CacheCreation1hTokens == total {
			return false
		}
		usage.CacheCreation1hTokens = total
		usage.CacheCreation5mTokens = 0
	default:
		if usage.CacheCreation5mTokens == total {
			return false
		}
		usage.CacheCreation5mTokens = total
		usage.CacheCreation1hTokens = 0
	}
	return true
}
func rewriteCacheCreationJSON(usageObj map[string]any, target string) bool {
	ccObj, ok := usageObj["cache_creation"].(map[string]any)
	if !ok {
		return false
	}
	v5m, _ := parseSSEUsageInt(ccObj["ephemeral_5m_input_tokens"])
	v1h, _ := parseSSEUsageInt(ccObj["ephemeral_1h_input_tokens"])
	total := v5m + v1h
	if total == 0 {
		return false
	}
	switch target {
	case "1h":
		if v1h == total {
			return false
		}
		ccObj["ephemeral_1h_input_tokens"] = float64(total)
		ccObj["ephemeral_5m_input_tokens"] = float64(0)
	default:
		if v5m == total {
			return false
		}
		ccObj["ephemeral_5m_input_tokens"] = float64(total)
		ccObj["ephemeral_1h_input_tokens"] = float64(0)
	}
	return true
}
func (s *GatewayService) handleNonStreamingResponse(ctx context.Context, resp *http.Response, c *gin.Context, account *Account, originalModel, mappedModel string) (*ClaudeUsage, error) {
	s.rateLimitService.UpdateSessionWindow(ctx, account, resp.Header)
	maxBytes := resolveUpstreamResponseReadLimit(s.cfg)
	body, err := readUpstreamResponseBodyLimited(resp.Body, maxBytes)
	if err != nil {
		if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
			setOpsUpstreamError(c, http.StatusBadGateway, "upstream response too large", "")
			c.JSON(http.StatusBadGateway, gin.H{"type": "error", "error": gin.H{"type": "upstream_error", "message": "Upstream response too large"}})
		}
		return nil, err
	}
	var response struct {
		Usage ClaudeUsage `json:"usage"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	cc5m := gjson.GetBytes(body, "usage.cache_creation.ephemeral_5m_input_tokens")
	cc1h := gjson.GetBytes(body, "usage.cache_creation.ephemeral_1h_input_tokens")
	if cc5m.Exists() || cc1h.Exists() {
		response.Usage.CacheCreation5mTokens = int(cc5m.Int())
		response.Usage.CacheCreation1hTokens = int(cc1h.Int())
	}
	if response.Usage.CacheReadInputTokens == 0 {
		cachedTokens := gjson.GetBytes(body, "usage.cached_tokens").Int()
		if cachedTokens > 0 {
			response.Usage.CacheReadInputTokens = int(cachedTokens)
			if newBody, err := sjson.SetBytes(body, "usage.cache_read_input_tokens", cachedTokens); err == nil {
				body = newBody
			}
		}
	}
	if account.IsCacheTTLOverrideEnabled() {
		overrideTarget := account.GetCacheTTLOverrideTarget()
		if applyCacheTTLOverride(&response.Usage, overrideTarget) {
			if newBody, err := sjson.SetBytes(body, "usage.cache_creation.ephemeral_5m_input_tokens", response.Usage.CacheCreation5mTokens); err == nil {
				body = newBody
			}
			if newBody, err := sjson.SetBytes(body, "usage.cache_creation.ephemeral_1h_input_tokens", response.Usage.CacheCreation1hTokens); err == nil {
				body = newBody
			}
		}
	}
	if originalModel != mappedModel {
		body = s.replaceModelInResponseBody(body, mappedModel, originalModel)
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := "application/json"
	if s.cfg != nil && !s.cfg.Security.ResponseHeaders.Enabled {
		if upstreamType := resp.Header.Get("Content-Type"); upstreamType != "" {
			contentType = upstreamType
		}
	}
	c.Data(resp.StatusCode, contentType, body)
	return &response.Usage, nil
}
