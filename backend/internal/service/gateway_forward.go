package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"

	"io"
	"net/http"
	"strings"
	"time"
)

func (s *GatewayService) Forward(ctx context.Context, c *gin.Context, account *Account, parsed *ParsedRequest) (*ForwardResult, error) {
	account = ResolveProtocolGatewayInboundAccount(account, PlatformAnthropic)
	startTime := time.Now()
	if parsed == nil {
		return nil, fmt.Errorf("parse request: empty request")
	}
	reasoningEffort := NormalizeClaudeOutputEffort(parsed.OutputEffort)
	if account != nil && account.IsAnthropicAPIKeyPassthroughEnabled() {
		passthroughBody := parsed.Body
		requestedModel := parsed.Model
		passthroughModel := ResolveGatewaySelectionModelFromContext(ctx, parsed.Model)
		if passthroughModel == "" {
			passthroughModel = parsed.Model
		}
		if passthroughModel != "" {
			if mappedModel := account.GetMappedModel(passthroughModel); mappedModel != passthroughModel {
				passthroughBody = s.replaceModelInBody(passthroughBody, mappedModel)
				logger.LegacyPrintf("service.gateway", "Passthrough model mapping: %s -> %s (account: %s)", parsed.Model, mappedModel, account.Name)
				passthroughModel = mappedModel
			}
		}
		return s.forwardAnthropicAPIKeyPassthrough(ctx, c, account, passthroughBody, requestedModel, passthroughModel, parsed.Stream, startTime, reasoningEffort)
	}
	if account.IsAnthropic() && c != nil {
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
	reqModel := s.resolveCanonicalRequestModel(ctx, parsed.Model)
	if reqModel == "" {
		reqModel = parsed.Model
	}
	if selectionModel := ResolveGatewaySelectionModelFromContext(ctx, reqModel); selectionModel != "" {
		reqModel = selectionModel
	}
	reqStream := parsed.Stream
	originalModel := parsed.RawModel
	if originalModel == "" {
		originalModel = reqModel
	}
	isClaudeCode := isClaudeCodeRequest(ctx, c, parsed)
	shouldMimicClaudeCode := IsClaudeClientMimicEnabled(account, PlatformAnthropic) && !isClaudeCode
	if shouldMimicClaudeCode {
		if !strings.Contains(strings.ToLower(reqModel), "haiku") && !systemIncludesClaudeCodePrompt(parsed.System) {
			body = rewriteSystemForNonClaudeCode(body, parsed.System)
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
	body = StripEmptyTextBlocks(body)
	body = s.maybeInjectAnthropicCacheTTL1h(ctx, account, body)
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
		logger.LegacyPrintf("service.gateway", "Model mapping applied: %s -> %s (account: %s, source=%s)", originalModel, mappedModel, account.Name, mappingSource)
	}
	debugLogRequestBody("CLIENT_ORIGINAL", body)
	token := ""
	tokenType := ""
	var err error
	if account.Platform != PlatformKiro {
		token, tokenType, err = s.GetAccessToken(ctx, account)
		if err != nil {
			return nil, err
		}
	}
	proxyURL := resolveGatewayProxyURL(account)
	tlsProfile := resolveAccountTLSFingerprintProfile(account, s.tlsFingerprintProfileService)
	logger.LegacyPrintf("service.gateway", "[Forward] Using account: ID=%d Name=%s Platform=%s Type=%s TLSFingerprint=%v Proxy=%s", account.ID, account.Name, RoutingPlatformForAccount(account), account.Type, account.IsTLSFingerprintEnabled(), proxyURL)
	setOpsUpstreamRequestBody(c, body)
	var resp *http.Response
	var kiroRuntime *KiroRuntimeService
	if account.Platform == PlatformKiro {
		kiroRuntime = NewKiroRuntimeService(s.accountRepo, s.httpUpstream, s.claudeTokenProvider)
		kiroRuntime.SetTLSFingerprintProfileService(s.tlsFingerprintProfileService)
	}
	retryStart := time.Now()
	for attempt := 1; attempt <= maxRetryAttempts; attempt++ {
		if account.Platform == PlatformKiro {
			execResult, execErr := kiroRuntime.ExecuteClaude(ctx, account, KiroRuntimeExecuteInput{
				Body:           body,
				ModelID:        reqModel,
				Stream:         reqStream,
				RequestHeaders: c.Request.Header,
			})
			err = execErr
			if execResult != nil {
				resp = execResult.Response
			} else {
				resp = nil
			}
		} else {
			upstreamReq, buildErr := s.buildUpstreamRequest(ctx, c, account, body, token, tokenType, reqModel, reqStream, shouldMimicClaudeCode)
			if buildErr != nil {
				return nil, buildErr
			}
			resp, err = s.httpUpstream.DoWithTLS(upstreamReq, proxyURL, account.ID, account.Concurrency, tlsProfile)
		}
		if err != nil {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			safeErr := sanitizeUpstreamErrorMessage(err.Error())
			setOpsUpstreamError(c, 0, safeErr, "")
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: 0, Kind: "request_error", Message: safeErr})
			c.JSON(http.StatusBadGateway, gin.H{"type": "error", "error": gin.H{"type": "upstream_error", "message": "Upstream request failed"}})
			return nil, fmt.Errorf("upstream request failed: %s", safeErr)
		}
		if resp.StatusCode == 400 {
			respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			if readErr == nil {
				_ = resp.Body.Close()
				if s.isThinkingBlockSignatureError(respBody) && s.settingService.IsSignatureRectifierEnabled(ctx) {
					appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "signature_error", Message: extractUpstreamErrorMessage(respBody), Detail: func() string {
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
						retryResp, retryErr := s.httpUpstream.DoWithTLS(retryReq, proxyURL, account.ID, account.Concurrency, tlsProfile)
						if retryErr == nil {
							if retryResp.StatusCode < 400 {
								logger.LegacyPrintf("service.gateway", "Account %d: signature error retry succeeded (thinking downgraded)", account.ID)
								resp = retryResp
								break
							}
							retryRespBody, retryReadErr := io.ReadAll(io.LimitReader(retryResp.Body, 2<<20))
							_ = retryResp.Body.Close()
							if retryReadErr == nil && retryResp.StatusCode == 400 && s.isThinkingBlockSignatureError(retryRespBody) {
								appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: retryResp.StatusCode, UpstreamRequestID: retryResp.Header.Get("x-request-id"), Kind: "signature_retry_thinking", Message: extractUpstreamErrorMessage(retryRespBody), Detail: func() string {
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
										retryResp2, retryErr2 := s.httpUpstream.DoWithTLS(retryReq2, proxyURL, account.ID, account.Concurrency, tlsProfile)
										if retryErr2 == nil {
											resp = retryResp2
											break
										}
										if retryResp2 != nil && retryResp2.Body != nil {
											_ = retryResp2.Body.Close()
										}
										appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: 0, Kind: "signature_retry_tools_request_error", Message: sanitizeUpstreamErrorMessage(retryErr2.Error())})
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
					appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "budget_constraint_error", Message: errMsg, Detail: func() string {
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
							budgetRetryResp, retryErr := s.httpUpstream.DoWithTLS(budgetRetryReq, proxyURL, account.ID, account.Concurrency, tlsProfile)
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
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "retry", Message: extractUpstreamErrorMessage(respBody), Detail: func() string {
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
		if account.IsGemini() && resp.StatusCode < 400 && s.cfg != nil && s.cfg.Gateway.GeminiDebugResponseHeaders {
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
			appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "retry_exhausted_failover", Message: extractUpstreamErrorMessage(respBody), Detail: func() string {
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
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "failover", Message: extractUpstreamErrorMessage(respBody), Detail: func() string {
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
				appendOpsUpstreamError(c, OpsUpstreamErrorEvent{Platform: RoutingPlatformForAccount(account), AccountID: account.ID, AccountName: account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "failover_on_400", Message: upstreamMsg, Detail: upstreamDetail})
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
	debugLogRequestBody("UPSTREAM_FORWARD", body)
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
	return &ForwardResult{RequestID: resp.Header.Get("x-request-id"), Usage: *usage, Model: originalModel, UpstreamModel: reqModel, ReasoningEffort: reasoningEffort, Stream: reqStream, Duration: time.Since(startTime), FirstTokenMs: firstTokenMs, ClientDisconnect: clientDisconnect}, nil
}
