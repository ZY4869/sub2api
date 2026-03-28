package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// Responses handles OpenAI Responses API endpoint
// POST /openai/v1/responses
func (h *OpenAIGatewayHandler) Responses(c *gin.Context) {
	// 局部兜底：确保该 handler 内部任何 panic 都不会击穿到进程级。
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)
	compactStartedAt := time.Now()
	defer h.logOpenAIRemoteCompactOutcome(c, compactStartedAt)
	setOpenAIClientTransportHTTP(c)

	requestStart := time.Now()

	// Get apiKey and user from context (set by ApiKeyAuth middleware)
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	if apiKey.Group != nil {
		applyOpenAIPlatformContext(c, apiKey.Group.Platform)
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.responses",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	// Read request body
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}

	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	setOpsRequestContext(c, "", false, body)
	sessionHashBody := body
	if service.IsOpenAIResponsesCompactPathForTest(c) {
		if compactSeed := strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String()); compactSeed != "" {
			c.Set(service.OpenAICompactSessionSeedKeyForTest(), compactSeed)
		}
		normalizedCompactBody, normalizedCompact, compactErr := service.NormalizeOpenAICompactRequestBodyForTest(body)
		if compactErr != nil {
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to normalize compact request body")
			return
		}
		if normalizedCompact {
			body = normalizedCompactBody
		}
	}

	// 校验请求体 JSON 合法性
	if !gjson.ValidBytes(body) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}

	// 使用 gjson 只读提取字段做校验，避免完整 Unmarshal
	modelResult := gjson.GetBytes(body, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String || modelResult.String() == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	reqModel := modelResult.String()

	streamResult := gjson.GetBytes(body, "stream")
	if streamResult.Exists() && streamResult.Type != gjson.True && streamResult.Type != gjson.False {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "invalid stream field type")
		return
	}
	reqStream := streamResult.Bool()
	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", reqStream))
	previousResponseID := strings.TrimSpace(gjson.GetBytes(body, "previous_response_id").String())
	if previousResponseID != "" {
		previousResponseIDKind := service.ClassifyOpenAIPreviousResponseIDKind(previousResponseID)
		reqLog = reqLog.With(
			zap.Bool("has_previous_response_id", true),
			zap.String("previous_response_id_kind", previousResponseIDKind),
			zap.Int("previous_response_id_len", len(previousResponseID)),
		)
		if previousResponseIDKind == service.OpenAIPreviousResponseIDKindMessageID {
			reqLog.Warn("openai.request_validation_failed",
				zap.String("reason", "previous_response_id_looks_like_message_id"),
			)
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "previous_response_id must be a response.id (resp_*), not a message id")
			return
		}
	}

	setOpsRequestContext(c, reqModel, reqStream, body)

	// 提前校验 function_call_output 是否具备可关联上下文，避免上游 400。
	if !h.validateFunctionCallOutputRequest(c, body, reqLog) {
		return
	}

	// 绑定错误透传服务，允许 service 层在非 failover 错误场景复用规则。
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}

	// Get subscription info (may be nil)
	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, reqStream, &streamStarted, reqLog)
	if !acquired {
		return
	}
	// 确保请求取消时也会释放槽位，避免长连接被动中断造成泄漏
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	// Generate session hash (header first; fallback to prompt_cache_key)
	sessionHash := h.gatewayService.GenerateSessionHash(c, sessionHashBody)
	excludedGroupIDs := make(map[int64]struct{})
	maxAccountSwitches := h.maxAccountSwitches

	for {
		if isRequestCanceled(c.Request.Context(), nil) {
			return
		}
		currentAPIKey, currentSubscription, err := resolveSelectedOpenAIAPIKey(
			c,
			h.settingService,
			h.gatewayService,
			h.billingCacheService,
			apiKey,
			subscription,
			reqModel,
			openAICompatiblePlatforms,
			excludedGroupIDs,
		)
		if err != nil {
			if isRequestCanceled(c.Request.Context(), err) {
				return
			}
			reqLog.Info("openai.group_selection_failed", zap.Error(err))
			status, code, message := groupSelectionErrorDetails(err)
			h.handleStreamingAwareError(c, status, code, message, streamStarted)
			return
		}
		if currentAPIKey.Group != nil {
			applyOpenAIPlatformContext(c, currentAPIKey.Group.Platform)
		}

		switchCount := 0
		failedAccountIDs := make(map[int64]struct{})
		sameAccountRetryCount := make(map[int64]int)
		var lastFailoverErr *service.UpstreamFailoverError

		for {
			if isRequestCanceled(c.Request.Context(), nil) {
				return
			}
			// Select account supporting the requested model
			reqLog.Debug("openai.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
			selection, scheduleDecision, err := h.gatewayService.SelectAccountWithScheduler(
				c.Request.Context(),
				currentAPIKey.GroupID,
				previousResponseID,
				sessionHash,
				reqModel,
				failedAccountIDs,
				service.OpenAIUpstreamTransportAny,
			)
			if err != nil {
				if isRequestCanceled(c.Request.Context(), err) {
					return
				}
				reqLog.Warn("openai.account_select_failed",
					zap.Error(err),
					zap.Int("excluded_account_count", len(failedAccountIDs)),
				)
				if len(failedAccountIDs) == 0 {
					if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
						break
					}
					h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable", streamStarted)
					return
				}
				if lastFailoverErr != nil {
					if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
						break
					}
					h.handleFailoverExhausted(c, lastFailoverErr, streamStarted)
				} else {
					if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
						break
					}
					h.handleFailoverExhaustedSimple(c, 502, streamStarted)
				}
				return
			}
			if selection == nil || selection.Account == nil {
				if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
					break
				}
				h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
				return
			}
			if previousResponseID != "" && selection != nil && selection.Account != nil {
				reqLog.Debug("openai.account_selected_with_previous_response_id", zap.Int64("account_id", selection.Account.ID))
			}
			reqLog.Debug("openai.account_schedule_decision",
				zap.String("layer", scheduleDecision.Layer),
				zap.Bool("sticky_previous_hit", scheduleDecision.StickyPreviousHit),
				zap.Bool("sticky_session_hit", scheduleDecision.StickySessionHit),
				zap.Int("candidate_count", scheduleDecision.CandidateCount),
				zap.Int("top_k", scheduleDecision.TopK),
				zap.Int64("latency_ms", scheduleDecision.LatencyMs),
				zap.Float64("load_skew", scheduleDecision.LoadSkew),
			)
			account := selection.Account
			sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
			reqLog.Debug("openai.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
			setOpsSelectedAccount(c, account.ID, account.Platform)
			setOpsEndpointContext(c, account.GetMappedModel(reqModel), service.RequestTypeFromLegacy(reqStream, false))

			accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, currentAPIKey.GroupID, sessionHash, selection, reqStream, &streamStarted, reqLog)
			if !acquired {
				return
			}

			// Forward request
			service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
			forwardStart := time.Now()
			result, err := h.gatewayService.Forward(c.Request.Context(), c, account, body)
			forwardDurationMs := time.Since(forwardStart).Milliseconds()
			if accountReleaseFunc != nil {
				accountReleaseFunc()
			}
			upstreamLatencyMs, _ := getContextInt64(c, service.OpsUpstreamLatencyMsKey)
			responseLatencyMs := forwardDurationMs
			if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
				responseLatencyMs = forwardDurationMs - upstreamLatencyMs
			}
			service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, responseLatencyMs)
			if err == nil && result != nil && result.FirstTokenMs != nil {
				service.SetOpsLatencyMs(c, service.OpsTimeToFirstTokenMsKey, int64(*result.FirstTokenMs))
			}
			if err != nil {
				var failoverErr *service.UpstreamFailoverError
				if errors.As(err, &failoverErr) {
					h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
					// 池模式：同账号重试
					if failoverErr.RetryableOnSameAccount {
						retryLimit := account.GetPoolModeRetryCount()
						if sameAccountRetryCount[account.ID] < retryLimit {
							sameAccountRetryCount[account.ID]++
							reqLog.Warn("openai.pool_mode_same_account_retry",
								zap.Int64("account_id", account.ID),
								zap.Int("upstream_status", failoverErr.StatusCode),
								zap.Int("retry_limit", retryLimit),
								zap.Int("retry_count", sameAccountRetryCount[account.ID]),
							)
							select {
							case <-c.Request.Context().Done():
								return
							case <-time.After(sameAccountRetryDelay):
							}
							continue
						}
					}
					h.gatewayService.RecordOpenAIAccountSwitch()
					failedAccountIDs[account.ID] = struct{}{}
					lastFailoverErr = failoverErr
					if switchCount >= maxAccountSwitches {
						if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
							break
						}
						h.submitFailedUsageRecordTask(
							"handler.openai_gateway.responses",
							c,
							currentAPIKey,
							currentSubscription,
							account,
							reqModel,
							reqStream,
							time.Since(forwardStart),
							failoverErr,
							err,
						)
						h.handleFailoverExhausted(c, failoverErr, streamStarted)
						return
					}
					switchCount++
					reqLog.Warn("openai.upstream_failover_switching",
						zap.Int64("account_id", account.ID),
						zap.Int("upstream_status", failoverErr.StatusCode),
						zap.Int("switch_count", switchCount),
						zap.Int("max_switches", maxAccountSwitches),
					)
					continue
				}
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
				h.submitFailedUsageRecordTask(
					"handler.openai_gateway.responses",
					c,
					currentAPIKey,
					currentSubscription,
					account,
					reqModel,
					reqStream,
					time.Since(forwardStart),
					nil,
					err,
				)
				fields := []zap.Field{
					zap.Int64("account_id", account.ID),
					zap.Bool("fallback_error_response_written", wroteFallback),
					zap.Error(err),
				}
				if shouldLogOpenAIForwardFailureAsWarn(c, wroteFallback) {
					reqLog.Warn("openai.forward_failed", fields...)
					return
				}
				reqLog.Error("openai.forward_failed", fields...)
				return
			}
			if result != nil {
				if account.Type == service.AccountTypeOAuth {
					h.gatewayService.UpdateCodexUsageSnapshotFromHeaders(c.Request.Context(), account.ID, result.ResponseHeaders)
				}
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
			} else {
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
			}

			// 捕获请求信息（用于异步记录，避免在 goroutine 中访问 gin.Context）
			userAgent := c.GetHeader("User-Agent")
			clientIP := ip.GetClientIP(c)

			// 使用量记录通过有界 worker 池提交，避免请求热路径创建无界 goroutine。
			h.submitUsageRecordTask(func(ctx context.Context) {
				if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
					Result:        result,
					APIKey:        currentAPIKey,
					User:          currentAPIKey.User,
					Account:       account,
					Subscription:  currentSubscription,
					UserAgent:     userAgent,
					IPAddress:     clientIP,
					APIKeyService: h.apiKeyService,
				}); err != nil {
					logger.L().With(
						zap.String("component", "handler.openai_gateway.responses"),
						zap.Int64("user_id", subject.UserID),
						zap.Int64("api_key_id", currentAPIKey.ID),
						zap.Any("group_id", currentAPIKey.GroupID),
						zap.String("model", reqModel),
						zap.Int64("account_id", account.ID),
					).Error("openai.record_usage_failed", zap.Error(err))
				}
			})
			reqLog.Debug("openai.request_completed",
				zap.Int64("account_id", account.ID),
				zap.Int("switch_count", switchCount),
			)
			return
		}
	}
}

func (h *OpenAIGatewayHandler) validateFunctionCallOutputRequest(c *gin.Context, body []byte, reqLog *zap.Logger) bool {
	if !gjson.GetBytes(body, `input.#(type=="function_call_output")`).Exists() {
		return true
	}

	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		// 保持原有容错语义：解析失败时跳过预校验，沿用后续上游校验结果。
		return true
	}

	c.Set(service.OpenAIParsedRequestBodyKey, reqBody)
	validation := service.ValidateFunctionCallOutputContext(reqBody)
	if !validation.HasFunctionCallOutput {
		return true
	}

	previousResponseID, _ := reqBody["previous_response_id"].(string)
	if strings.TrimSpace(previousResponseID) != "" || validation.HasToolCallContext {
		return true
	}

	if validation.HasFunctionCallOutputMissingCallID {
		reqLog.Warn("openai.request_validation_failed",
			zap.String("reason", "function_call_output_missing_call_id"),
		)
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "function_call_output requires call_id or previous_response_id; if relying on history, ensure store=true and reuse previous_response_id")
		return false
	}
	if validation.HasItemReferenceForAllCallIDs {
		return true
	}

	reqLog.Warn("openai.request_validation_failed",
		zap.String("reason", "function_call_output_missing_item_reference"),
	)
	h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "function_call_output requires item_reference ids matching each call_id, or previous_response_id/tool_call context; if relying on history, ensure store=true and reuse previous_response_id")
	return false
}

func (h *OpenAIGatewayHandler) acquireResponsesUserSlot(
	c *gin.Context,
	userID int64,
	userConcurrency int,
	reqStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), bool) {
	ctx := c.Request.Context()
	userReleaseFunc, userAcquired, err := h.concurrencyHelper.TryAcquireUserSlot(ctx, userID, userConcurrency)
	if err != nil {
		reqLog.Warn("openai.user_slot_acquire_failed", zap.Error(err))
		h.handleConcurrencyError(c, err, "user", *streamStarted)
		return nil, false
	}
	if userAcquired {
		return wrapReleaseOnDone(ctx, userReleaseFunc), true
	}

	maxWait := service.CalculateMaxWait(userConcurrency)
	canWait, waitErr := h.concurrencyHelper.IncrementWaitCount(ctx, userID, maxWait)
	if waitErr != nil {
		reqLog.Warn("openai.user_wait_counter_increment_failed", zap.Error(waitErr))
		// 按现有降级语义：等待计数异常时放行后续抢槽流程
	} else if !canWait {
		reqLog.Info("openai.user_wait_queue_full", zap.Int("max_wait", maxWait))
		h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later")
		return nil, false
	}

	waitCounted := waitErr == nil && canWait
	defer func() {
		if waitCounted {
			h.concurrencyHelper.DecrementWaitCount(ctx, userID)
		}
	}()

	userReleaseFunc, err = h.concurrencyHelper.AcquireUserSlotWithWait(c, userID, userConcurrency, reqStream, streamStarted)
	if err != nil {
		reqLog.Warn("openai.user_slot_acquire_failed_after_wait", zap.Error(err))
		h.handleConcurrencyError(c, err, "user", *streamStarted)
		return nil, false
	}

	// 槽位获取成功后，立刻退出等待计数。
	if waitCounted {
		h.concurrencyHelper.DecrementWaitCount(ctx, userID)
		waitCounted = false
	}
	return wrapReleaseOnDone(ctx, userReleaseFunc), true
}

func (h *OpenAIGatewayHandler) acquireResponsesAccountSlot(
	c *gin.Context,
	groupID *int64,
	sessionHash string,
	selection *service.AccountSelectionResult,
	reqStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), bool) {
	if selection == nil || selection.Account == nil {
		h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", *streamStarted)
		return nil, false
	}

	ctx := c.Request.Context()
	account := selection.Account
	if selection.Acquired {
		return wrapReleaseOnDone(ctx, selection.ReleaseFunc), true
	}
	if selection.WaitPlan == nil {
		h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", *streamStarted)
		return nil, false
	}

	fastReleaseFunc, fastAcquired, err := h.concurrencyHelper.TryAcquireAccountSlot(
		ctx,
		account.ID,
		selection.WaitPlan.MaxConcurrency,
	)
	if err != nil {
		reqLog.Warn("openai.account_slot_quick_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		h.handleConcurrencyError(c, err, "account", *streamStarted)
		return nil, false
	}
	if fastAcquired {
		if err := h.gatewayService.BindStickySession(ctx, groupID, sessionHash, account.ID); err != nil {
			reqLog.Warn("openai.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		}
		return wrapReleaseOnDone(ctx, fastReleaseFunc), true
	}

	canWait, waitErr := h.concurrencyHelper.IncrementAccountWaitCount(ctx, account.ID, selection.WaitPlan.MaxWaiting)
	if waitErr != nil {
		reqLog.Warn("openai.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Error(waitErr))
	} else if !canWait {
		reqLog.Info("openai.account_wait_queue_full",
			zap.Int64("account_id", account.ID),
			zap.Int("max_waiting", selection.WaitPlan.MaxWaiting),
		)
		h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", *streamStarted)
		return nil, false
	}

	accountWaitCounted := waitErr == nil && canWait
	releaseWait := func() {
		if accountWaitCounted {
			h.concurrencyHelper.DecrementAccountWaitCount(ctx, account.ID)
			accountWaitCounted = false
		}
	}
	defer releaseWait()

	accountReleaseFunc, err := h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(
		c,
		account.ID,
		selection.WaitPlan.MaxConcurrency,
		selection.WaitPlan.Timeout,
		reqStream,
		streamStarted,
	)
	if err != nil {
		reqLog.Warn("openai.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		h.handleConcurrencyError(c, err, "account", *streamStarted)
		return nil, false
	}

	// Slot acquired: no longer waiting in queue.
	releaseWait()
	if err := h.gatewayService.BindStickySession(ctx, groupID, sessionHash, account.ID); err != nil {
		reqLog.Warn("openai.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
	}
	return wrapReleaseOnDone(ctx, accountReleaseFunc), true
}

func getContextInt64(c *gin.Context, key string) (int64, bool) {
	if c == nil || key == "" {
		return 0, false
	}
	v, ok := c.Get(key)
	if !ok {
		return 0, false
	}
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case int32:
		return int64(t), true
	case float64:
		return int64(t), true
	default:
		return 0, false
	}
}

func ensureOpenAIPoolModeSessionHash(sessionHash string, account *service.Account) string {
	if sessionHash != "" || account == nil || !account.IsPoolMode() {
		return sessionHash
	}
	// 为当前请求生成一次性粘性会话键，确保同账号重试不会重新负载均衡到其他账号。
	return "openai-pool-retry-" + uuid.NewString()
}
