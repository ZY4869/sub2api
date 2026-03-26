package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	mathrand "math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type antigravityRetryLoopParams struct {
	ctx             context.Context
	prefix          string
	account         *Account
	proxyURL        string
	accessToken     string
	action          string
	body            []byte
	c               *gin.Context
	httpUpstream    HTTPUpstream
	settingService  *SettingService
	accountRepo     AccountRepository
	handleError     func(ctx context.Context, prefix string, account *Account, statusCode int, headers http.Header, body []byte, requestedModel string, groupID int64, sessionHash string, isStickySession bool) *handleModelRateLimitResult
	requestedModel  string
	isStickySession bool
	groupID         int64
	sessionHash     string
}
type antigravityRetryLoopResult struct{ resp *http.Response }

func resolveAntigravityForwardBaseURL() string {
	baseURLs := antigravity.ForwardBaseURLs()
	if len(baseURLs) == 0 {
		return ""
	}
	mode := strings.ToLower(strings.TrimSpace(os.Getenv(antigravityForwardBaseURLEnv)))
	if mode == "prod" && len(baseURLs) > 1 {
		return baseURLs[1]
	}
	return baseURLs[0]
}

type smartRetryAction int

const (
	smartRetryActionContinue smartRetryAction = iota
	smartRetryActionBreakWithResp
	smartRetryActionContinueURL
)

type smartRetryResult struct {
	action      smartRetryAction
	resp        *http.Response
	err         error
	switchError *AntigravityAccountSwitchError
}

func (s *AntigravityGatewayService) handleSmartRetry(p antigravityRetryLoopParams, resp *http.Response, respBody []byte, baseURL string, urlIdx int, availableURLs []string) *smartRetryResult {
	if resp.StatusCode == http.StatusTooManyRequests && isURLLevelRateLimit(respBody) && urlIdx < len(availableURLs)-1 {
		logger.LegacyPrintf("service.antigravity_gateway", "%s URL fallback (429): %s -> %s", p.prefix, baseURL, availableURLs[urlIdx+1])
		return &smartRetryResult{action: smartRetryActionContinueURL}
	}
	shouldSmartRetry, shouldRateLimitModel, waitDuration, modelName, isModelCapacityExhausted := shouldTriggerAntigravitySmartRetry(p.account, respBody)
	if shouldRateLimitModel {
		if resp.StatusCode == http.StatusServiceUnavailable && isSingleAccountRetry(p.ctx) {
			return s.handleSingleAccountRetryInPlace(p, resp, respBody, baseURL, waitDuration, modelName)
		}
		rateLimitDuration := waitDuration
		if rateLimitDuration <= 0 {
			rateLimitDuration = antigravityDefaultRateLimitDuration
		}
		logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d oauth_long_delay model=%s account=%d upstream_retry_delay=%v body=%s (model rate limit, switch account)", p.prefix, resp.StatusCode, modelName, p.account.ID, rateLimitDuration, truncateForLog(respBody, 200))
		resetAt := time.Now().Add(rateLimitDuration)
		if !setModelRateLimitByModelName(p.ctx, p.accountRepo, p.account.ID, modelName, p.prefix, resp.StatusCode, resetAt, false) {
			p.handleError(p.ctx, p.prefix, p.account, resp.StatusCode, resp.Header, respBody, p.requestedModel, p.groupID, p.sessionHash, p.isStickySession)
			logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d rate_limited account=%d (no model mapping)", p.prefix, resp.StatusCode, p.account.ID)
		} else {
			s.updateAccountModelRateLimitInCache(p.ctx, p.account, modelName, resetAt)
		}
		return &smartRetryResult{action: smartRetryActionBreakWithResp, switchError: &AntigravityAccountSwitchError{OriginalAccountID: p.account.ID, RateLimitedModel: modelName, IsStickySession: p.isStickySession}}
	}
	if shouldSmartRetry {
		var lastRetryResp *http.Response
		var lastRetryBody []byte
		maxAttempts := antigravitySmartRetryMaxAttempts
		if isModelCapacityExhausted {
			maxAttempts = antigravityModelCapacityRetryMaxAttempts
			waitDuration = antigravityModelCapacityRetryWait
			if modelName != "" {
				modelCapacityExhaustedMu.RLock()
				cooldownUntil, exists := modelCapacityExhaustedUntil[modelName]
				modelCapacityExhaustedMu.RUnlock()
				if exists && time.Now().Before(cooldownUntil) {
					log.Printf("%s status=%d model_capacity_exhausted_dedup model=%s account=%d cooldown_until=%v (skip retry)", p.prefix, resp.StatusCode, modelName, p.account.ID, cooldownUntil.Format("15:04:05"))
					return &smartRetryResult{action: smartRetryActionBreakWithResp, resp: &http.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(respBody))}}
				}
			}
		}
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			log.Printf("%s status=%d oauth_smart_retry attempt=%d/%d delay=%v model=%s account=%d", p.prefix, resp.StatusCode, attempt, maxAttempts, waitDuration, modelName, p.account.ID)
			timer := time.NewTimer(waitDuration)
			select {
			case <-p.ctx.Done():
				timer.Stop()
				log.Printf("%s status=context_canceled_during_smart_retry", p.prefix)
				return &smartRetryResult{action: smartRetryActionBreakWithResp, err: p.ctx.Err()}
			case <-timer.C:
			}
			retryReq, err := antigravity.NewAPIRequestWithURL(p.ctx, baseURL, p.action, p.accessToken, p.body)
			if err != nil {
				logger.LegacyPrintf("service.antigravity_gateway", "%s status=smart_retry_request_build_failed error=%v", p.prefix, err)
				p.handleError(p.ctx, p.prefix, p.account, resp.StatusCode, resp.Header, respBody, p.requestedModel, p.groupID, p.sessionHash, p.isStickySession)
				return &smartRetryResult{action: smartRetryActionBreakWithResp, resp: &http.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(respBody))}}
			}
			retryResp, retryErr := p.httpUpstream.Do(retryReq, p.proxyURL, p.account.ID, p.account.Concurrency)
			if retryErr == nil && retryResp != nil && retryResp.StatusCode != http.StatusTooManyRequests && retryResp.StatusCode != http.StatusServiceUnavailable {
				log.Printf("%s status=%d smart_retry_success attempt=%d/%d", p.prefix, retryResp.StatusCode, attempt, maxAttempts)
				if isModelCapacityExhausted && modelName != "" {
					modelCapacityExhaustedMu.Lock()
					delete(modelCapacityExhaustedUntil, modelName)
					modelCapacityExhaustedMu.Unlock()
				}
				return &smartRetryResult{action: smartRetryActionBreakWithResp, resp: retryResp}
			}
			if retryErr != nil || retryResp == nil {
				log.Printf("%s status=smart_retry_network_error attempt=%d/%d error=%v", p.prefix, attempt, maxAttempts, retryErr)
				continue
			}
			if lastRetryResp != nil {
				_ = lastRetryResp.Body.Close()
			}
			lastRetryResp = retryResp
			lastRetryBody, _ = io.ReadAll(io.LimitReader(retryResp.Body, 8<<10))
			_ = retryResp.Body.Close()
			if !isModelCapacityExhausted && attempt < maxAttempts && lastRetryBody != nil {
				newShouldRetry, _, newWaitDuration, _, _ := shouldTriggerAntigravitySmartRetry(p.account, lastRetryBody)
				if newShouldRetry && newWaitDuration > 0 {
					waitDuration = newWaitDuration
				}
			}
		}
		rateLimitDuration := waitDuration
		if rateLimitDuration <= 0 {
			rateLimitDuration = antigravityDefaultRateLimitDuration
		}
		retryBody := lastRetryBody
		if retryBody == nil {
			retryBody = respBody
		}
		if isModelCapacityExhausted {
			if modelName != "" {
				modelCapacityExhaustedMu.Lock()
				modelCapacityExhaustedUntil[modelName] = time.Now().Add(antigravityModelCapacityCooldown)
				modelCapacityExhaustedMu.Unlock()
			}
			log.Printf("%s status=%d smart_retry_exhausted_model_capacity attempts=%d model=%s account=%d body=%s (model capacity exhausted, not switching account)", p.prefix, resp.StatusCode, maxAttempts, modelName, p.account.ID, truncateForLog(retryBody, 200))
			return &smartRetryResult{action: smartRetryActionBreakWithResp, resp: &http.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(retryBody))}}
		}
		if resp.StatusCode == http.StatusServiceUnavailable && isSingleAccountRetry(p.ctx) {
			logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d smart_retry_exhausted_single_account attempts=%d model=%s account=%d body=%s (return 503 directly)", p.prefix, resp.StatusCode, antigravitySmartRetryMaxAttempts, modelName, p.account.ID, truncateForLog(retryBody, 200))
			return &smartRetryResult{action: smartRetryActionBreakWithResp, resp: &http.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(retryBody))}}
		}
		log.Printf("%s status=%d smart_retry_exhausted attempts=%d model=%s account=%d upstream_retry_delay=%v body=%s (switch account)", p.prefix, resp.StatusCode, maxAttempts, modelName, p.account.ID, rateLimitDuration, truncateForLog(retryBody, 200))
		resetAt := time.Now().Add(rateLimitDuration)
		if p.accountRepo != nil && modelName != "" {
			if err := p.accountRepo.SetModelRateLimit(p.ctx, p.account.ID, modelName, resetAt); err != nil {
				logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d model_rate_limit_failed model=%s error=%v", p.prefix, resp.StatusCode, modelName, err)
			} else {
				logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d model_rate_limited_after_smart_retry model=%s account=%d reset_in=%v", p.prefix, resp.StatusCode, modelName, p.account.ID, rateLimitDuration)
				s.updateAccountModelRateLimitInCache(p.ctx, p.account, modelName, resetAt)
			}
		}
		if s.cache != nil && p.sessionHash != "" {
			_ = s.cache.DeleteSessionAccountID(p.ctx, p.groupID, p.sessionHash)
		}
		return &smartRetryResult{action: smartRetryActionBreakWithResp, switchError: &AntigravityAccountSwitchError{OriginalAccountID: p.account.ID, RateLimitedModel: modelName, IsStickySession: p.isStickySession}}
	}
	return &smartRetryResult{action: smartRetryActionContinue}
}
func (s *AntigravityGatewayService) handleSingleAccountRetryInPlace(p antigravityRetryLoopParams, resp *http.Response, respBody []byte, baseURL string, waitDuration time.Duration, modelName string) *smartRetryResult {
	if waitDuration > antigravitySingleAccountSmartRetryMaxWait {
		waitDuration = antigravitySingleAccountSmartRetryMaxWait
	}
	if waitDuration < antigravitySmartRetryMinWait {
		waitDuration = antigravitySmartRetryMinWait
	}
	logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d single_account_503_retry_in_place model=%s account=%d upstream_retry_delay=%v (retrying in-place instead of rate-limiting)", p.prefix, resp.StatusCode, modelName, p.account.ID, waitDuration)
	var lastRetryResp *http.Response
	var lastRetryBody []byte
	totalWaited := time.Duration(0)
	for attempt := 1; attempt <= antigravitySingleAccountSmartRetryMaxAttempts; attempt++ {
		if totalWaited+waitDuration > antigravitySingleAccountSmartRetryTotalMaxWait {
			remaining := antigravitySingleAccountSmartRetryTotalMaxWait - totalWaited
			if remaining <= 0 {
				logger.LegacyPrintf("service.antigravity_gateway", "%s single_account_503_retry: total_wait_exceeded total=%v max=%v, giving up", p.prefix, totalWaited, antigravitySingleAccountSmartRetryTotalMaxWait)
				break
			}
			waitDuration = remaining
		}
		logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d single_account_503_retry attempt=%d/%d delay=%v total_waited=%v model=%s account=%d", p.prefix, resp.StatusCode, attempt, antigravitySingleAccountSmartRetryMaxAttempts, waitDuration, totalWaited, modelName, p.account.ID)
		timer := time.NewTimer(waitDuration)
		select {
		case <-p.ctx.Done():
			timer.Stop()
			logger.LegacyPrintf("service.antigravity_gateway", "%s status=context_canceled_during_single_account_retry", p.prefix)
			return &smartRetryResult{action: smartRetryActionBreakWithResp, err: p.ctx.Err()}
		case <-timer.C:
		}
		totalWaited += waitDuration
		retryReq, err := antigravity.NewAPIRequestWithURL(p.ctx, baseURL, p.action, p.accessToken, p.body)
		if err != nil {
			logger.LegacyPrintf("service.antigravity_gateway", "%s single_account_503_retry: request_build_failed error=%v", p.prefix, err)
			break
		}
		retryResp, retryErr := p.httpUpstream.Do(retryReq, p.proxyURL, p.account.ID, p.account.Concurrency)
		if retryErr == nil && retryResp != nil && retryResp.StatusCode != http.StatusTooManyRequests && retryResp.StatusCode != http.StatusServiceUnavailable {
			logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d single_account_503_retry_success attempt=%d/%d total_waited=%v", p.prefix, retryResp.StatusCode, attempt, antigravitySingleAccountSmartRetryMaxAttempts, totalWaited)
			if lastRetryResp != nil {
				_ = lastRetryResp.Body.Close()
			}
			return &smartRetryResult{action: smartRetryActionBreakWithResp, resp: retryResp}
		}
		if retryErr != nil || retryResp == nil {
			logger.LegacyPrintf("service.antigravity_gateway", "%s single_account_503_retry: network_error attempt=%d/%d error=%v", p.prefix, attempt, antigravitySingleAccountSmartRetryMaxAttempts, retryErr)
			continue
		}
		if lastRetryResp != nil {
			_ = lastRetryResp.Body.Close()
		}
		lastRetryResp = retryResp
		lastRetryBody, _ = io.ReadAll(io.LimitReader(retryResp.Body, 8<<10))
		_ = retryResp.Body.Close()
		if attempt < antigravitySingleAccountSmartRetryMaxAttempts && lastRetryBody != nil {
			_, _, newWaitDuration, _, _ := shouldTriggerAntigravitySmartRetry(p.account, lastRetryBody)
			if newWaitDuration > 0 {
				waitDuration = newWaitDuration
				if waitDuration > antigravitySingleAccountSmartRetryMaxWait {
					waitDuration = antigravitySingleAccountSmartRetryMaxWait
				}
				if waitDuration < antigravitySmartRetryMinWait {
					waitDuration = antigravitySmartRetryMinWait
				}
			}
		}
	}
	retryBody := lastRetryBody
	if retryBody == nil {
		retryBody = respBody
	}
	logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d single_account_503_retry_exhausted attempts=%d total_waited=%v model=%s account=%d body=%s (return 503 directly)", p.prefix, resp.StatusCode, antigravitySingleAccountSmartRetryMaxAttempts, totalWaited, modelName, p.account.ID, truncateForLog(retryBody, 200))
	return &smartRetryResult{action: smartRetryActionBreakWithResp, resp: &http.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(retryBody))}}
}
func (s *AntigravityGatewayService) antigravityRetryLoop(p antigravityRetryLoopParams) (*antigravityRetryLoopResult, error) {
	if p.requestedModel != "" {
		if remaining := p.account.GetRateLimitRemainingTimeWithContext(p.ctx, p.requestedModel); remaining > 0 {
			if isSingleAccountRetry(p.ctx) {
				logger.LegacyPrintf("service.antigravity_gateway", "%s pre_check: single_account_retry skipping rate_limit remaining=%v model=%s account=%d (will retry in-place if 503)", p.prefix, remaining.Truncate(time.Millisecond), p.requestedModel, p.account.ID)
			} else {
				logger.LegacyPrintf("service.antigravity_gateway", "%s pre_check: rate_limit_switch remaining=%v model=%s account=%d", p.prefix, remaining.Truncate(time.Millisecond), p.requestedModel, p.account.ID)
				return nil, &AntigravityAccountSwitchError{OriginalAccountID: p.account.ID, RateLimitedModel: p.requestedModel, IsStickySession: p.isStickySession}
			}
		}
	}
	baseURL := resolveAntigravityForwardBaseURL()
	if baseURL == "" {
		return nil, errors.New("no antigravity forward base url configured")
	}
	availableURLs := []string{baseURL}
	var resp *http.Response
	var usedBaseURL string
	logBody := p.settingService != nil && p.settingService.cfg != nil && p.settingService.cfg.Gateway.LogUpstreamErrorBody
	maxBytes := 2048
	if p.settingService != nil && p.settingService.cfg != nil && p.settingService.cfg.Gateway.LogUpstreamErrorBodyMaxBytes > 0 {
		maxBytes = p.settingService.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
	}
	getUpstreamDetail := func(body []byte) string {
		if !logBody {
			return ""
		}
		return truncateString(string(body), maxBytes)
	}
urlFallbackLoop:
	for urlIdx, baseURL := range availableURLs {
		usedBaseURL = baseURL
		for attempt := 1; attempt <= antigravityMaxRetries; attempt++ {
			select {
			case <-p.ctx.Done():
				logger.LegacyPrintf("service.antigravity_gateway", "%s status=context_canceled error=%v", p.prefix, p.ctx.Err())
				return nil, p.ctx.Err()
			default:
			}
			upstreamReq, err := antigravity.NewAPIRequestWithURL(p.ctx, baseURL, p.action, p.accessToken, p.body)
			if err != nil {
				return nil, err
			}
			if p.c != nil && len(p.body) > 0 {
				p.c.Set(OpsUpstreamRequestBodyKey, string(p.body))
			}
			resp, err = p.httpUpstream.Do(upstreamReq, p.proxyURL, p.account.ID, p.account.Concurrency)
			if err == nil && resp == nil {
				err = errors.New("upstream returned nil response")
			}
			if err != nil {
				safeErr := sanitizeUpstreamErrorMessage(err.Error())
				appendOpsUpstreamError(p.c, OpsUpstreamErrorEvent{Platform: p.account.Platform, AccountID: p.account.ID, AccountName: p.account.Name, UpstreamStatusCode: 0, Kind: "request_error", Message: safeErr})
				if shouldAntigravityFallbackToNextURL(err, 0) && urlIdx < len(availableURLs)-1 {
					logger.LegacyPrintf("service.antigravity_gateway", "%s URL fallback (connection error): %s -> %s", p.prefix, baseURL, availableURLs[urlIdx+1])
					continue urlFallbackLoop
				}
				if attempt < antigravityMaxRetries {
					logger.LegacyPrintf("service.antigravity_gateway", "%s status=request_failed retry=%d/%d error=%v", p.prefix, attempt, antigravityMaxRetries, err)
					if !sleepAntigravityBackoffWithContext(p.ctx, attempt) {
						logger.LegacyPrintf("service.antigravity_gateway", "%s status=context_canceled_during_backoff", p.prefix)
						return nil, p.ctx.Err()
					}
					continue
				}
				logger.LegacyPrintf("service.antigravity_gateway", "%s status=request_failed retries_exhausted error=%v", p.prefix, err)
				setOpsUpstreamError(p.c, 0, safeErr, "")
				return nil, fmt.Errorf("upstream request failed after retries: %w", err)
			}
			if resp.StatusCode >= 400 {
				respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
				_ = resp.Body.Close()
				if handled, outStatus, policyErr := s.applyErrorPolicy(p, resp.StatusCode, resp.Header, respBody); handled {
					if policyErr != nil {
						return nil, policyErr
					}
					resp = &http.Response{StatusCode: outStatus, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(respBody))}
					break urlFallbackLoop
				}
				if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
					smartResult := s.handleSmartRetry(p, resp, respBody, baseURL, urlIdx, availableURLs)
					switch smartResult.action {
					case smartRetryActionContinueURL:
						continue urlFallbackLoop
					case smartRetryActionBreakWithResp:
						if smartResult.err != nil {
							return nil, smartResult.err
						}
						if smartResult.switchError != nil {
							return nil, smartResult.switchError
						}
						resp = smartResult.resp
						break urlFallbackLoop
					}
					if attempt < antigravityMaxRetries {
						upstreamMsg := strings.TrimSpace(extractAntigravityErrorMessage(respBody))
						upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
						appendOpsUpstreamError(p.c, OpsUpstreamErrorEvent{Platform: p.account.Platform, AccountID: p.account.ID, AccountName: p.account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "retry", Message: upstreamMsg, Detail: getUpstreamDetail(respBody)})
						logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d retry=%d/%d body=%s", p.prefix, resp.StatusCode, attempt, antigravityMaxRetries, truncateForLog(respBody, 200))
						if !sleepAntigravityBackoffWithContext(p.ctx, attempt) {
							logger.LegacyPrintf("service.antigravity_gateway", "%s status=context_canceled_during_backoff", p.prefix)
							return nil, p.ctx.Err()
						}
						continue
					}
					p.handleError(p.ctx, p.prefix, p.account, resp.StatusCode, resp.Header, respBody, p.requestedModel, p.groupID, p.sessionHash, p.isStickySession)
					logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d rate_limited base_url=%s body=%s", p.prefix, resp.StatusCode, baseURL, truncateForLog(respBody, 200))
					resp = &http.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(respBody))}
					break urlFallbackLoop
				}
				if shouldRetryAntigravityError(resp.StatusCode) {
					if attempt < antigravityMaxRetries {
						upstreamMsg := strings.TrimSpace(extractAntigravityErrorMessage(respBody))
						upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
						appendOpsUpstreamError(p.c, OpsUpstreamErrorEvent{Platform: p.account.Platform, AccountID: p.account.ID, AccountName: p.account.Name, UpstreamStatusCode: resp.StatusCode, UpstreamRequestID: resp.Header.Get("x-request-id"), Kind: "retry", Message: upstreamMsg, Detail: getUpstreamDetail(respBody)})
						logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d retry=%d/%d body=%s", p.prefix, resp.StatusCode, attempt, antigravityMaxRetries, truncateForLog(respBody, 500))
						if !sleepAntigravityBackoffWithContext(p.ctx, attempt) {
							logger.LegacyPrintf("service.antigravity_gateway", "%s status=context_canceled_during_backoff", p.prefix)
							return nil, p.ctx.Err()
						}
						continue
					}
				}
				resp = &http.Response{StatusCode: resp.StatusCode, Header: resp.Header.Clone(), Body: io.NopCloser(bytes.NewReader(respBody))}
				break urlFallbackLoop
			}
			break urlFallbackLoop
		}
	}
	if resp != nil && resp.StatusCode < 400 && usedBaseURL != "" {
		antigravity.DefaultURLAvailability.MarkSuccess(usedBaseURL)
	}
	return &antigravityRetryLoopResult{resp: resp}, nil
}
func (s *AntigravityGatewayService) shouldFailoverUpstreamError(statusCode int) bool {
	switch statusCode {
	case 401, 403, 429, 529:
		return true
	default:
		return statusCode >= 500
	}
}
func isGoogleProjectConfigError(lowerMsg string) bool {
	return strings.Contains(lowerMsg, "invalid project resource name")
}

const googleConfigErrorCooldown = 1 * time.Minute

func tempUnscheduleGoogleConfigError(ctx context.Context, repo AccountRepository, accountID int64, logPrefix string) {
	until := time.Now().Add(googleConfigErrorCooldown)
	reason := "400: invalid project resource name (auto temp-unschedule 1m)"
	if err := repo.SetTempUnschedulable(ctx, accountID, until, reason); err != nil {
		log.Printf("%s temp_unschedule_failed account=%d error=%v", logPrefix, accountID, err)
	} else {
		log.Printf("%s temp_unscheduled account=%d until=%v reason=%q", logPrefix, accountID, until.Format("15:04:05"), reason)
	}
}

const emptyResponseCooldown = 1 * time.Minute

func tempUnscheduleEmptyResponse(ctx context.Context, repo AccountRepository, accountID int64, logPrefix string) {
	until := time.Now().Add(emptyResponseCooldown)
	reason := "empty stream response (auto temp-unschedule 1m)"
	if err := repo.SetTempUnschedulable(ctx, accountID, until, reason); err != nil {
		log.Printf("%s temp_unschedule_failed account=%d error=%v", logPrefix, accountID, err)
	} else {
		log.Printf("%s temp_unscheduled account=%d until=%v reason=%q", logPrefix, accountID, until.Format("15:04:05"), reason)
	}
}
func sleepAntigravityBackoffWithContext(ctx context.Context, attempt int) bool {
	delay := antigravityRetryBaseDelay * time.Duration(1<<uint(attempt-1))
	if delay > antigravityRetryMaxDelay {
		delay = antigravityRetryMaxDelay
	}
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	jitter := time.Duration(float64(delay) * 0.2 * (r.Float64()*2 - 1))
	sleepFor := delay + jitter
	if sleepFor < 0 {
		sleepFor = 0
	}
	timer := time.NewTimer(sleepFor)
	select {
	case <-ctx.Done():
		timer.Stop()
		return false
	case <-timer.C:
		return true
	}
}
func isSingleAccountRetry(ctx context.Context) bool {
	v, _ := SingleAccountRetryFromContext(ctx)
	return v
}
func setModelRateLimitByModelName(ctx context.Context, repo AccountRepository, accountID int64, modelName, prefix string, statusCode int, resetAt time.Time, afterSmartRetry bool) bool {
	if repo == nil || modelName == "" {
		return false
	}
	if err := repo.SetModelRateLimit(ctx, accountID, modelName, resetAt); err != nil {
		logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d model_rate_limit_failed model=%s error=%v", prefix, statusCode, modelName, err)
		return false
	}
	if afterSmartRetry {
		logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d model_rate_limited_after_smart_retry model=%s account=%d reset_in=%v", prefix, statusCode, modelName, accountID, time.Until(resetAt).Truncate(time.Second))
	} else {
		logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d model_rate_limited model=%s account=%d reset_in=%v", prefix, statusCode, modelName, accountID, time.Until(resetAt).Truncate(time.Second))
	}
	return true
}
func antigravityFallbackCooldownSeconds() (time.Duration, bool) {
	raw := strings.TrimSpace(os.Getenv(antigravityFallbackSecondsEnv))
	if raw == "" {
		return 0, false
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		return 0, false
	}
	return time.Duration(seconds) * time.Second, true
}

type antigravitySmartRetryInfo struct {
	RetryDelay               time.Duration
	ModelName                string
	IsModelCapacityExhausted bool
}

func parseAntigravitySmartRetryInfo(body []byte) *antigravitySmartRetryInfo {
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil
	}
	errObj, ok := parsed["error"].(map[string]any)
	if !ok {
		return nil
	}
	status, _ := errObj["status"].(string)
	isResourceExhausted := status == googleRPCStatusResourceExhausted
	isUnavailable := status == googleRPCStatusUnavailable
	if !isResourceExhausted && !isUnavailable {
		return nil
	}
	details, ok := errObj["details"].([]any)
	if !ok {
		return nil
	}
	var retryDelay time.Duration
	var modelName string
	var hasRateLimitExceeded bool
	var hasModelCapacityExhausted bool
	for _, d := range details {
		dm, ok := d.(map[string]any)
		if !ok {
			continue
		}
		atType, _ := dm["@type"].(string)
		if atType == googleRPCTypeErrorInfo {
			if meta, ok := dm["metadata"].(map[string]any); ok {
				if model, ok := meta["model"].(string); ok {
					modelName = model
				}
			}
			if reason, ok := dm["reason"].(string); ok {
				if reason == googleRPCReasonModelCapacityExhausted {
					hasModelCapacityExhausted = true
				}
				if reason == googleRPCReasonRateLimitExceeded {
					hasRateLimitExceeded = true
				}
			}
			continue
		}
		if atType == googleRPCTypeRetryInfo {
			delay, ok := dm["retryDelay"].(string)
			if !ok || delay == "" {
				continue
			}
			dur, err := time.ParseDuration(delay)
			if err != nil {
				logger.LegacyPrintf("service.antigravity_gateway", "[Antigravity] failed to parse retryDelay: %s error=%v", delay, err)
				continue
			}
			retryDelay = dur
		}
	}
	if isResourceExhausted && !hasRateLimitExceeded {
		return nil
	}
	if isUnavailable && !hasModelCapacityExhausted {
		return nil
	}
	if modelName == "" {
		return nil
	}
	if retryDelay <= 0 {
		retryDelay = antigravityDefaultRateLimitDuration
	}
	return &antigravitySmartRetryInfo{RetryDelay: retryDelay, ModelName: modelName, IsModelCapacityExhausted: hasModelCapacityExhausted}
}
func shouldTriggerAntigravitySmartRetry(account *Account, respBody []byte) (shouldRetry bool, shouldRateLimitModel bool, waitDuration time.Duration, modelName string, isModelCapacityExhausted bool) {
	if account.Platform != PlatformAntigravity {
		return false, false, 0, "", false
	}
	info := parseAntigravitySmartRetryInfo(respBody)
	if info == nil {
		return false, false, 0, "", false
	}
	if info.IsModelCapacityExhausted {
		return true, false, antigravityModelCapacityRetryWait, info.ModelName, true
	}
	if info.RetryDelay >= antigravityRateLimitThreshold {
		return false, true, info.RetryDelay, info.ModelName, false
	}
	waitDuration = info.RetryDelay
	if waitDuration < antigravitySmartRetryMinWait {
		waitDuration = antigravitySmartRetryMinWait
	}
	return true, false, waitDuration, info.ModelName, false
}

type handleModelRateLimitParams struct {
	ctx             context.Context
	prefix          string
	account         *Account
	statusCode      int
	body            []byte
	cache           GatewayCache
	groupID         int64
	sessionHash     string
	isStickySession bool
}
type handleModelRateLimitResult struct {
	Handled      bool
	ShouldRetry  bool
	WaitDuration time.Duration
	SwitchError  *AntigravityAccountSwitchError
}

func (s *AntigravityGatewayService) handleModelRateLimit(p *handleModelRateLimitParams) *handleModelRateLimitResult {
	if p.statusCode != 429 && p.statusCode != 503 {
		return &handleModelRateLimitResult{Handled: false}
	}
	info := parseAntigravitySmartRetryInfo(p.body)
	if info == nil || info.ModelName == "" {
		return &handleModelRateLimitResult{Handled: false}
	}
	if info.IsModelCapacityExhausted {
		log.Printf("%s status=%d model_capacity_exhausted model=%s (not switching account, retry handled by smart retry)", p.prefix, p.statusCode, info.ModelName)
		return &handleModelRateLimitResult{Handled: true}
	}
	if info.RetryDelay < antigravityRateLimitThreshold {
		logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d model_rate_limit_wait model=%s wait=%v", p.prefix, p.statusCode, info.ModelName, info.RetryDelay)
		return &handleModelRateLimitResult{Handled: true, ShouldRetry: true, WaitDuration: info.RetryDelay}
	}
	s.setModelRateLimitAndClearSession(p, info)
	return &handleModelRateLimitResult{Handled: true, SwitchError: &AntigravityAccountSwitchError{OriginalAccountID: p.account.ID, RateLimitedModel: info.ModelName, IsStickySession: p.isStickySession}}
}
func (s *AntigravityGatewayService) setModelRateLimitAndClearSession(p *handleModelRateLimitParams, info *antigravitySmartRetryInfo) {
	resetAt := time.Now().Add(info.RetryDelay)
	logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d model_rate_limited model=%s account=%d reset_in=%v", p.prefix, p.statusCode, info.ModelName, p.account.ID, info.RetryDelay)
	if err := s.accountRepo.SetModelRateLimit(p.ctx, p.account.ID, info.ModelName, resetAt); err != nil {
		logger.LegacyPrintf("service.antigravity_gateway", "%s model_rate_limit_failed model=%s error=%v", p.prefix, info.ModelName, err)
	}
	s.updateAccountModelRateLimitInCache(p.ctx, p.account, info.ModelName, resetAt)
	if p.cache != nil && p.sessionHash != "" {
		_ = p.cache.DeleteSessionAccountID(p.ctx, p.groupID, p.sessionHash)
	}
}
func (s *AntigravityGatewayService) updateAccountModelRateLimitInCache(ctx context.Context, account *Account, modelKey string, resetAt time.Time) {
	if s.schedulerSnapshot == nil || account == nil || modelKey == "" {
		return
	}
	if account.Extra == nil {
		account.Extra = make(map[string]any)
	}
	limits, _ := account.Extra["model_rate_limits"].(map[string]any)
	if limits == nil {
		limits = make(map[string]any)
		account.Extra["model_rate_limits"] = limits
	}
	limits[modelKey] = map[string]any{"rate_limited_at": time.Now().UTC().Format(time.RFC3339), "rate_limit_reset_at": resetAt.UTC().Format(time.RFC3339)}
	if err := s.schedulerSnapshot.UpdateAccountInCache(ctx, account); err != nil {
		logger.LegacyPrintf("service.antigravity_gateway", "[antigravity-Forward] cache_update_failed account=%d model=%s err=%v", account.ID, modelKey, err)
	}
}
func (s *AntigravityGatewayService) handleUpstreamError(ctx context.Context, prefix string, account *Account, statusCode int, headers http.Header, body []byte, requestedModel string, groupID int64, sessionHash string, isStickySession bool) *handleModelRateLimitResult {
	if !account.ShouldHandleErrorCode(statusCode) {
		return nil
	}
	result := s.handleModelRateLimit(&handleModelRateLimitParams{ctx: ctx, prefix: prefix, account: account, statusCode: statusCode, body: body, cache: s.cache, groupID: groupID, sessionHash: sessionHash, isStickySession: isStickySession})
	if result.Handled {
		return result
	}
	if statusCode == 503 {
		return nil
	}
	if statusCode == 429 {
		if logBody, maxBytes := s.getLogConfig(); logBody {
			logger.LegacyPrintf("service.antigravity_gateway", "[Antigravity-Debug] 429 response body: %s", truncateString(string(body), maxBytes))
		}
		resetAt := ParseGeminiRateLimitResetTime(body)
		defaultDur := s.getDefaultRateLimitDuration()
		modelKey := resolveFinalAntigravityModelKey(ctx, account, requestedModel)
		if strings.TrimSpace(modelKey) == "" {
			modelKey = resolveAntigravityModelKey(requestedModel)
		}
		if modelKey != "" {
			ra := s.resolveResetTime(resetAt, defaultDur)
			if err := s.accountRepo.SetModelRateLimit(ctx, account.ID, modelKey, ra); err != nil {
				logger.LegacyPrintf("service.antigravity_gateway", "%s status=429 model_rate_limit_set_failed model=%s error=%v", prefix, modelKey, err)
			} else {
				logger.LegacyPrintf("service.antigravity_gateway", "%s status=429 model_rate_limited model=%s account=%d reset_at=%v reset_in=%v", prefix, modelKey, account.ID, ra.Format("15:04:05"), time.Until(ra).Truncate(time.Second))
				s.updateAccountModelRateLimitInCache(ctx, account, modelKey, ra)
			}
			return nil
		}
		ra := s.resolveResetTime(resetAt, defaultDur)
		logger.LegacyPrintf("service.antigravity_gateway", "%s status=429 rate_limited account=%d reset_at=%v reset_in=%v (fallback)", prefix, account.ID, ra.Format("15:04:05"), time.Until(ra).Truncate(time.Second))
		if err := setAccountRateLimited(ctx, s.accountRepo, account.ID, ra, AccountRateLimitReason429); err != nil {
			logger.LegacyPrintf("service.antigravity_gateway", "%s status=429 rate_limit_set_failed account=%d error=%v", prefix, account.ID, err)
		}
		return nil
	}
	if s.rateLimitService == nil {
		return nil
	}
	shouldDisable := s.rateLimitService.HandleUpstreamError(ctx, account, statusCode, headers, body)
	if shouldDisable {
		logger.LegacyPrintf("service.antigravity_gateway", "%s status=%d marked_error", prefix, statusCode)
	}
	return nil
}
