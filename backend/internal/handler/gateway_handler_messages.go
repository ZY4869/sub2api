package handler

import (
	"context"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

func forwardFailedLogFields(account *service.Account, wroteFallback bool, err error) []zap.Field {
	fields := []zap.Field{
		zap.Bool("fallback_error_response_written", wroteFallback),
		zap.Error(err),
	}
	if account == nil {
		return fields
	}
	fields = append(fields,
		zap.Int64("account_id", account.ID),
		zap.String("account_name", account.Name),
		zap.String("account_platform", account.Platform),
	)
	if account.Proxy != nil {
		fields = append(fields,
			zap.Int64("proxy_id", account.Proxy.ID),
			zap.String("proxy_name", account.Proxy.Name),
			zap.String("proxy_host", account.Proxy.Host),
			zap.Int("proxy_port", account.Proxy.Port),
		)
	} else if account.ProxyID != nil {
		fields = append(fields, zap.Int64p("proxy_id", account.ProxyID))
	}
	return fields
}

func (h *GatewayHandler) Messages(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(c, "handler.gateway.messages", zap.Int64("user_id", subject.UserID), zap.Int64("api_key_id", apiKey.ID), zap.Any("group_id", apiKey.GroupID))
	defer h.maybeLogCompatibilityFallbackMetrics(reqLog)
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
	parsedReq, err := service.ParseGatewayRequest(body, domain.PlatformAnthropic)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	h.resolveParsedRequestModel(c.Request.Context(), parsedReq)
	reqModel := parsedReq.Model
	reqStream := parsedReq.Stream
	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", reqStream))
	if isMaxTokensOneHaikuRequest(reqModel, parsedReq.MaxTokens, reqStream) {
		ctx := service.WithIsMaxTokensOneHaikuRequest(c.Request.Context(), true, h.metadataBridgeEnabled())
		c.Request = c.Request.WithContext(ctx)
	}
	SetClaudeCodeClientContext(c, body, parsedReq)
	isClaudeCodeClient := service.IsClaudeCodeClient(c.Request.Context())
	if !h.checkClaudeCodeVersion(c) {
		return
	}
	c.Request = c.Request.WithContext(service.WithThinkingEnabled(c.Request.Context(), parsedReq.ThinkingEnabled, h.metadataBridgeEnabled()))
	setOpsRequestContext(c, reqModel, reqStream, body)
	if reqModel == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	selectionModel := h.gatewayService.ResolveAPIKeySelectionModel(c.Request.Context(), apiKey, "", reqModel)
	streamStarted := false
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	maxWait := service.CalculateMaxWait(subject.Concurrency)
	canWait, err := h.concurrencyHelper.IncrementWaitCount(c.Request.Context(), subject.UserID, maxWait)
	waitCounted := false
	if err != nil {
		reqLog.Warn("gateway.user_wait_counter_increment_failed", zap.Error(err))
	} else if !canWait {
		reqLog.Info("gateway.user_wait_queue_full", zap.Int("max_wait", maxWait))
		h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later")
		return
	}
	if err == nil && canWait {
		waitCounted = true
	}
	defer func() {
		if waitCounted {
			h.concurrencyHelper.DecrementWaitCount(c.Request.Context(), subject.UserID)
		}
	}()
	userReleaseFunc, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, subject.UserID, subject.Concurrency, reqStream, &streamStarted)
	if err != nil {
		reqLog.Warn("gateway.user_slot_acquire_failed", zap.Error(err))
		h.handleConcurrencyError(c, err, "user", streamStarted)
		return
	}
	if waitCounted {
		h.concurrencyHelper.DecrementWaitCount(c.Request.Context(), subject.UserID)
		waitCounted = false
	}
	userReleaseFunc = wrapReleaseOnDone(c.Request.Context(), userReleaseFunc)
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}
	parsedReq.SessionContext = &service.SessionContext{ClientIP: ip.GetClientIP(c), UserAgent: c.GetHeader("User-Agent"), APIKeyID: apiKey.ID}
	selectedSessionHash := h.gatewayService.GenerateSessionHash(parsedReq)
	forcePlatform, hasForcePlatform := middleware2.GetForcePlatformFromContext(c)
	allowedPlatforms := gatewayCompatiblePlatforms
	if hasForcePlatform && strings.TrimSpace(forcePlatform) != "" {
		allowedPlatforms = []string{forcePlatform}
	}
	excludedGroupIDs := make(map[int64]struct{})

	for {
		currentAPIKey, currentSubscription, err := resolveSelectedGatewayAPIKey(
			c,
			h.settingService,
			h.gatewayService,
			h.billingCacheService,
			apiKey,
			subscription,
			selectionModel,
			allowedPlatforms,
			excludedGroupIDs,
		)
		if err != nil {
			reqLog.Info("gateway.group_selection_failed", zap.Error(err))
			status, code, message := groupSelectionErrorDetails(err)
			h.handleStreamingAwareError(c, status, code, message, streamStarted)
			return
		}

		currentPlatform := ""
		if currentAPIKey.Group != nil {
			currentPlatform = currentAPIKey.Group.Platform
		}
		if currentPlatform != service.PlatformGemini &&
			currentPlatform != service.PlatformAnthropic &&
			currentPlatform != service.PlatformAntigravity &&
			currentPlatform != service.PlatformKiro {
			if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
				continue
			}
			h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "This endpoint does not support the selected platform", streamStarted)
			return
		}
		runtimeSelectionModel, channelState, err := bindGatewayChannelState(c, h.gatewayService, currentAPIKey.Group, selectionModel)
		if err != nil {
			if errors.Is(err, service.ErrChannelModelNotAllowed) {
				h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "Requested model is not allowed by the bound channel", streamStarted)
				return
			}
			h.handleStreamingAwareError(c, http.StatusInternalServerError, "api_error", "Failed to resolve channel routing", streamStarted)
			return
		}

		sessionKey := selectedSessionHash
		if currentPlatform == service.PlatformGemini && selectedSessionHash != "" {
			sessionKey = "gemini:" + selectedSessionHash
		}
		var sessionBoundAccountID int64
		if sessionKey != "" {
			sessionBoundAccountID, _ = h.gatewayService.GetCachedSessionAccountID(c.Request.Context(), currentAPIKey.GroupID, sessionKey)
			if sessionBoundAccountID > 0 {
				prefetchedGroupID := int64(0)
				if currentAPIKey.GroupID != nil {
					prefetchedGroupID = *currentAPIKey.GroupID
				}
				ctx := service.WithPrefetchedStickySession(c.Request.Context(), sessionBoundAccountID, prefetchedGroupID, h.metadataBridgeEnabled())
				c.Request = c.Request.WithContext(ctx)
			}
		}
		hasBoundSession := sessionKey != "" && sessionBoundAccountID > 0

		if currentPlatform == service.PlatformGemini {
			fs := NewFailoverState(h.maxAccountSwitchesGemini, hasBoundSession)
			if h.gatewayService.IsSingleAntigravityAccountGroup(c.Request.Context(), currentAPIKey.GroupID) {
				ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
				c.Request = c.Request.WithContext(ctx)
			}
			for {
				selection, err := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), currentAPIKey.GroupID, sessionKey, runtimeSelectionModel, fs.FailedAccountIDs, "")
				if err != nil {
					if len(fs.FailedAccountIDs) == 0 {
						if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
							break
						}
						h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts: "+err.Error(), streamStarted)
						return
					}
					action := fs.HandleSelectionExhausted(c.Request.Context())
					switch action {
					case FailoverContinue:
						ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
						c.Request = c.Request.WithContext(ctx)
						continue
					case FailoverCanceled:
						return
					default:
						if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
							break
						}
						if fs.LastFailoverErr != nil {
							h.handleFailoverExhausted(c, fs.LastFailoverErr, service.PlatformGemini, streamStarted)
						} else {
							h.handleFailoverExhaustedSimple(c, 502, streamStarted)
						}
						return
					}
				}
				account := selection.Account
				setOpsSelectedAccountDetails(c, account)
				setOpsEndpointContext(c, account.GetMappedModel(runtimeSelectionModel), service.RequestTypeFromLegacy(reqStream, false))
				if account.IsInterceptWarmupEnabled() {
					interceptType := detectInterceptType(body, reqModel, parsedReq.MaxTokens, reqStream, isClaudeCodeClient)
					if interceptType != InterceptTypeNone {
						if selection.Acquired && selection.ReleaseFunc != nil {
							selection.ReleaseFunc()
						}
						if reqStream {
							sendMockInterceptStream(c, reqModel, interceptType)
						} else {
							sendMockInterceptResponse(c, reqModel, interceptType)
						}
						return
					}
				}
				accountReleaseFunc := selection.ReleaseFunc
				if !selection.Acquired {
					if selection.WaitPlan == nil {
						if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
							break
						}
						h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
						return
					}
					accountWaitCounted := false
					canWait, err := h.concurrencyHelper.IncrementAccountWaitCount(c.Request.Context(), account.ID, selection.WaitPlan.MaxWaiting)
					if err != nil {
						reqLog.Warn("gateway.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", currentAPIKey.GroupID), zap.Error(err))
					} else if !canWait {
						reqLog.Info("gateway.account_wait_queue_full", zap.Int64("account_id", account.ID), zap.Any("group_id", currentAPIKey.GroupID), zap.Int("max_waiting", selection.WaitPlan.MaxWaiting))
						h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", streamStarted)
						return
					}
					if err == nil && canWait {
						accountWaitCounted = true
					}
					releaseWait := func() {
						if accountWaitCounted {
							h.concurrencyHelper.DecrementAccountWaitCount(c.Request.Context(), account.ID)
							accountWaitCounted = false
						}
					}
					accountReleaseFunc, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(c, account.ID, selection.WaitPlan.MaxConcurrency, selection.WaitPlan.Timeout, reqStream, &streamStarted)
					if err != nil {
						reqLog.Warn("gateway.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", currentAPIKey.GroupID), zap.Error(err))
						releaseWait()
						h.handleConcurrencyError(c, err, "account", streamStarted)
						return
					}
					releaseWait()
					if err := h.gatewayService.BindStickySession(c.Request.Context(), currentAPIKey.GroupID, sessionKey, account.ID); err != nil {
						reqLog.Warn("gateway.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", currentAPIKey.GroupID), zap.Error(err))
					}
				}
				accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)
				var result *service.ForwardResult
				requestCtx := c.Request.Context()
				if fs.SwitchCount > 0 {
					requestCtx = service.WithAccountSwitchCount(requestCtx, fs.SwitchCount, h.metadataBridgeEnabled())
				}
				if account.Platform == service.PlatformAntigravity {
					result, err = h.antigravityGatewayService.ForwardGemini(requestCtx, c, account, reqModel, "generateContent", reqStream, body, hasBoundSession)
				} else {
					result, err = h.geminiCompatService.Forward(requestCtx, c, account, body)
				}
				if accountReleaseFunc != nil {
					accountReleaseFunc()
				}
				if err != nil {
					var failoverErr *service.UpstreamFailoverError
					if errors.As(err, &failoverErr) {
						action := fs.HandleFailoverError(c.Request.Context(), h.gatewayService, account.ID, account.Platform, failoverErr)
						switch action {
						case FailoverContinue:
							continue
						case FailoverExhausted:
							if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
								break
							}
							h.submitFailedUsageRecordTask(
								"handler.gateway.messages",
								c,
								currentAPIKey,
								currentSubscription,
								account,
								reqModel,
								reqStream,
								0,
								service.EffectiveProtocol(account),
								fs.LastFailoverErr,
								err,
							)
							h.handleFailoverExhausted(c, fs.LastFailoverErr, service.PlatformGemini, streamStarted)
							return
						case FailoverCanceled:
							return
						}
					}
					wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
					h.submitFailedUsageRecordTask(
						"handler.gateway.messages",
						c,
						currentAPIKey,
						currentSubscription,
						account,
						reqModel,
						reqStream,
						0,
						service.EffectiveProtocol(account),
						nil,
						err,
					)
					reqLog.Error("gateway.forward_failed", append([]zap.Field{zap.Any("group_id", currentAPIKey.GroupID)}, forwardFailedLogFields(account, wroteFallback, err)...)...)
					return
				}
				if account.IsAnthropicOAuthOrSetupToken() && account.GetBaseRPM() > 0 {
					if err := h.gatewayService.IncrementAccountRPM(c.Request.Context(), account.ID); err != nil {
						reqLog.Warn("gateway.rpm_increment_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", currentAPIKey.GroupID), zap.Error(err))
					}
				}
				userAgent := c.GetHeader("User-Agent")
				clientIP := ip.GetClientIP(c)
				h.submitUsageRecordTask(func(ctx context.Context) {
					ctx = reattachGatewayChannelState(ctx, channelState)
					if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{Result: result, APIKey: currentAPIKey, User: currentAPIKey.User, Account: account, Subscription: currentSubscription, ThinkingEnabled: service.ParseExplicitThinkingEnabledValue(body), InboundEndpoint: GetInboundEndpoint(c), UpstreamEndpoint: GetUpstreamEndpointForAccount(c, account), UserAgent: userAgent, IPAddress: clientIP, ForceCacheBilling: fs.ForceCacheBilling, APIKeyService: h.apiKeyService}); err != nil {
						logger.L().With(zap.String("component", "handler.gateway.messages"), zap.Int64("user_id", subject.UserID), zap.Int64("api_key_id", currentAPIKey.ID), zap.Any("group_id", currentAPIKey.GroupID), zap.String("model", reqModel), zap.Int64("account_id", account.ID)).Error("gateway.record_usage_failed", zap.Error(err))
					}
				})
				return
			}
			continue
		}

		runtimeAPIKey := currentAPIKey
		runtimeSubscription := currentSubscription
		runtimeSelectionModel, channelState, err = bindGatewayChannelState(c, h.gatewayService, runtimeAPIKey.Group, selectionModel)
		if err != nil {
			if errors.Is(err, service.ErrChannelModelNotAllowed) {
				h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "Requested model is not allowed by the bound channel", streamStarted)
				return
			}
			h.handleStreamingAwareError(c, http.StatusInternalServerError, "api_error", "Failed to resolve channel routing", streamStarted)
			return
		}
		var fallbackGroupID *int64
		if runtimeAPIKey.Group != nil {
			fallbackGroupID = runtimeAPIKey.Group.FallbackGroupIDOnInvalidRequest
		}
		fallbackUsed := false
		if h.gatewayService.IsSingleAntigravityAccountGroup(c.Request.Context(), runtimeAPIKey.GroupID) {
			ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
			c.Request = c.Request.WithContext(ctx)
		}
		for {
			fs := NewFailoverState(h.maxAccountSwitches, hasBoundSession)
			retryWithFallback := false
			for {
				selection, err := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), runtimeAPIKey.GroupID, sessionKey, runtimeSelectionModel, fs.FailedAccountIDs, parsedReq.MetadataUserID)
				if err != nil {
					if len(fs.FailedAccountIDs) == 0 {
						if excludeSelectedGroup(excludedGroupIDs, runtimeAPIKey) {
							break
						}
						h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts: "+err.Error(), streamStarted)
						return
					}
					action := fs.HandleSelectionExhausted(c.Request.Context())
					switch action {
					case FailoverContinue:
						ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
						c.Request = c.Request.WithContext(ctx)
						continue
					case FailoverCanceled:
						return
					default:
						if excludeSelectedGroup(excludedGroupIDs, runtimeAPIKey) {
							break
						}
						if fs.LastFailoverErr != nil {
							h.handleFailoverExhausted(c, fs.LastFailoverErr, currentPlatform, streamStarted)
						} else {
							h.handleFailoverExhaustedSimple(c, 502, streamStarted)
						}
						return
					}
				}
				account := selection.Account
				setOpsSelectedAccountDetails(c, account)
				setOpsEndpointContext(c, account.GetMappedModel(runtimeSelectionModel), service.RequestTypeFromLegacy(reqStream, false))
				if account.IsInterceptWarmupEnabled() {
					interceptType := detectInterceptType(body, reqModel, parsedReq.MaxTokens, reqStream, isClaudeCodeClient)
					if interceptType != InterceptTypeNone {
						if selection.Acquired && selection.ReleaseFunc != nil {
							selection.ReleaseFunc()
						}
						if reqStream {
							sendMockInterceptStream(c, reqModel, interceptType)
						} else {
							sendMockInterceptResponse(c, reqModel, interceptType)
						}
						return
					}
				}
				accountReleaseFunc := selection.ReleaseFunc
				if !selection.Acquired {
					if selection.WaitPlan == nil {
						if excludeSelectedGroup(excludedGroupIDs, runtimeAPIKey) {
							break
						}
						h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
						return
					}
					accountWaitCounted := false
					canWait, err := h.concurrencyHelper.IncrementAccountWaitCount(c.Request.Context(), account.ID, selection.WaitPlan.MaxWaiting)
					if err != nil {
						reqLog.Warn("gateway.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", runtimeAPIKey.GroupID), zap.Error(err))
					} else if !canWait {
						reqLog.Info("gateway.account_wait_queue_full", zap.Int64("account_id", account.ID), zap.Any("group_id", runtimeAPIKey.GroupID), zap.Int("max_waiting", selection.WaitPlan.MaxWaiting))
						h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", streamStarted)
						return
					}
					if err == nil && canWait {
						accountWaitCounted = true
					}
					releaseWait := func() {
						if accountWaitCounted {
							h.concurrencyHelper.DecrementAccountWaitCount(c.Request.Context(), account.ID)
							accountWaitCounted = false
						}
					}
					accountReleaseFunc, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(c, account.ID, selection.WaitPlan.MaxConcurrency, selection.WaitPlan.Timeout, reqStream, &streamStarted)
					if err != nil {
						reqLog.Warn("gateway.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", runtimeAPIKey.GroupID), zap.Error(err))
						releaseWait()
						h.handleConcurrencyError(c, err, "account", streamStarted)
						return
					}
					releaseWait()
					if err := h.gatewayService.BindStickySession(c.Request.Context(), runtimeAPIKey.GroupID, sessionKey, account.ID); err != nil {
						reqLog.Warn("gateway.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", runtimeAPIKey.GroupID), zap.Error(err))
					}
				}
				accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)
				var queueRelease func()
				umqMode := h.getUserMsgQueueMode(account, parsedReq)
				switch umqMode {
				case config.UMQModeSerialize:
					baseRPM := account.GetBaseRPM()
					release, qErr := h.userMsgQueueHelper.AcquireWithWait(c, account.ID, baseRPM, reqStream, &streamStarted, h.cfg.Gateway.UserMessageQueue.WaitTimeout(), reqLog)
					if qErr != nil {
						reqLog.Warn("gateway.umq_acquire_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", runtimeAPIKey.GroupID), zap.Error(qErr))
					} else {
						queueRelease = release
					}
				case config.UMQModeThrottle:
					baseRPM := account.GetBaseRPM()
					if tErr := h.userMsgQueueHelper.ThrottleWithPing(c, account.ID, baseRPM, reqStream, &streamStarted, h.cfg.Gateway.UserMessageQueue.WaitTimeout(), reqLog); tErr != nil {
						reqLog.Warn("gateway.umq_throttle_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", runtimeAPIKey.GroupID), zap.Error(tErr))
					}
				default:
					if umqMode != "" {
						reqLog.Warn("gateway.umq_unknown_mode", zap.String("mode", umqMode), zap.Int64("account_id", account.ID), zap.Any("group_id", runtimeAPIKey.GroupID))
					}
				}
				queueRelease = wrapReleaseOnDone(c.Request.Context(), queueRelease)
				parsedReq.OnUpstreamAccepted = queueRelease
				var result *service.ForwardResult
				requestCtx := c.Request.Context()
				if fs.SwitchCount > 0 {
					requestCtx = service.WithAccountSwitchCount(requestCtx, fs.SwitchCount, h.metadataBridgeEnabled())
				}
				if account.Platform == service.PlatformAntigravity && account.Type != service.AccountTypeAPIKey {
					result, err = h.antigravityGatewayService.Forward(requestCtx, c, account, body, hasBoundSession)
				} else {
					result, err = h.gatewayService.Forward(requestCtx, c, account, parsedReq)
				}
				if queueRelease != nil {
					queueRelease()
				}
				parsedReq.OnUpstreamAccepted = nil
				if accountReleaseFunc != nil {
					accountReleaseFunc()
				}
				if err != nil {
					var betaBlockedErr *service.BetaBlockedError
					if errors.As(err, &betaBlockedErr) {
						h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", betaBlockedErr.Message)
						return
					}
					var promptTooLongErr *service.PromptTooLongError
					if errors.As(err, &promptTooLongErr) {
						reqLog.Warn("gateway.prompt_too_long_from_antigravity", zap.Any("current_group_id", runtimeAPIKey.GroupID), zap.Any("fallback_group_id", fallbackGroupID), zap.Bool("fallback_used", fallbackUsed))
						if !fallbackUsed && fallbackGroupID != nil && *fallbackGroupID > 0 {
							fallbackGroup, err := h.gatewayService.ResolveGroupByID(c.Request.Context(), *fallbackGroupID)
							if err != nil {
								reqLog.Warn("gateway.resolve_fallback_group_failed", zap.Int64("fallback_group_id", *fallbackGroupID), zap.Error(err))
								_ = h.antigravityGatewayService.WriteMappedClaudeError(c, account, promptTooLongErr.StatusCode, promptTooLongErr.RequestID, promptTooLongErr.Body)
								return
							}
							if fallbackGroup.Platform != service.PlatformAnthropic || fallbackGroup.SubscriptionType == service.SubscriptionTypeSubscription || fallbackGroup.FallbackGroupIDOnInvalidRequest != nil {
								reqLog.Warn("gateway.fallback_group_invalid", zap.Int64("fallback_group_id", fallbackGroup.ID), zap.String("fallback_platform", fallbackGroup.Platform), zap.String("fallback_subscription_type", fallbackGroup.SubscriptionType))
								_ = h.antigravityGatewayService.WriteMappedClaudeError(c, account, promptTooLongErr.StatusCode, promptTooLongErr.RequestID, promptTooLongErr.Body)
								return
							}
							fallbackAPIKey := cloneAPIKeyWithGroup(runtimeAPIKey, fallbackGroup)
							if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), fallbackAPIKey.User, fallbackAPIKey, fallbackGroup, nil); err != nil {
								status, code, message := billingErrorDetails(err)
								h.handleStreamingAwareError(c, status, code, message, streamStarted)
								return
							}
							ctx := context.WithValue(c.Request.Context(), ctxkey.ForcePlatform, "")
							c.Request = c.Request.WithContext(ctx)
							runtimeAPIKey = fallbackAPIKey
							runtimeSubscription = nil
							fallbackUsed = true
							retryWithFallback = true
							break
						}
						_ = h.antigravityGatewayService.WriteMappedClaudeError(c, account, promptTooLongErr.StatusCode, promptTooLongErr.RequestID, promptTooLongErr.Body)
						return
					}
					var failoverErr *service.UpstreamFailoverError
					if errors.As(err, &failoverErr) {
						action := fs.HandleFailoverError(c.Request.Context(), h.gatewayService, account.ID, account.Platform, failoverErr)
						switch action {
						case FailoverContinue:
							continue
						case FailoverExhausted:
							if excludeSelectedGroup(excludedGroupIDs, runtimeAPIKey) {
								break
							}
							h.submitFailedUsageRecordTask(
								"handler.gateway.messages",
								c,
								runtimeAPIKey,
								runtimeSubscription,
								account,
								reqModel,
								reqStream,
								0,
								service.EffectiveProtocol(account),
								fs.LastFailoverErr,
								err,
							)
							h.handleFailoverExhausted(c, fs.LastFailoverErr, account.Platform, streamStarted)
							return
						case FailoverCanceled:
							return
						}
					}
					wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
					h.submitFailedUsageRecordTask(
						"handler.gateway.messages",
						c,
						runtimeAPIKey,
						runtimeSubscription,
						account,
						reqModel,
						reqStream,
						0,
						service.EffectiveProtocol(account),
						nil,
						err,
					)
					reqLog.Error("gateway.forward_failed", append([]zap.Field{zap.Any("group_id", runtimeAPIKey.GroupID)}, forwardFailedLogFields(account, wroteFallback, err)...)...)
					return
				}
				if account.IsAnthropicOAuthOrSetupToken() && account.GetBaseRPM() > 0 {
					if err := h.gatewayService.IncrementAccountRPM(c.Request.Context(), account.ID); err != nil {
						reqLog.Warn("gateway.rpm_increment_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", runtimeAPIKey.GroupID), zap.Error(err))
					}
				}
				userAgent := c.GetHeader("User-Agent")
				clientIP := ip.GetClientIP(c)
				h.submitUsageRecordTask(func(ctx context.Context) {
					ctx = reattachGatewayChannelState(ctx, channelState)
					if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{Result: result, APIKey: runtimeAPIKey, User: runtimeAPIKey.User, Account: account, Subscription: runtimeSubscription, ThinkingEnabled: service.ParseExplicitThinkingEnabledValue(body), InboundEndpoint: GetInboundEndpoint(c), UpstreamEndpoint: GetUpstreamEndpointForAccount(c, account), UserAgent: userAgent, IPAddress: clientIP, ForceCacheBilling: fs.ForceCacheBilling, APIKeyService: h.apiKeyService}); err != nil {
						logger.L().With(zap.String("component", "handler.gateway.messages"), zap.Int64("user_id", subject.UserID), zap.Int64("api_key_id", runtimeAPIKey.ID), zap.Any("group_id", runtimeAPIKey.GroupID), zap.String("model", reqModel), zap.Int64("account_id", account.ID)).Error("gateway.record_usage_failed", zap.Error(err))
					}
				})
				return
			}
			if !retryWithFallback {
				break
			}
		}
	}
}

/*
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
		reqLog.Info("gateway.billing_eligibility_check_failed", zap.Error(err))
		status, code, message := billingErrorDetails(err)
		h.handleStreamingAwareError(c, status, code, message, streamStarted)
		return
	}
	parsedReq.SessionContext = &service.SessionContext{ClientIP: ip.GetClientIP(c), UserAgent: c.GetHeader("User-Agent"), APIKeyID: apiKey.ID}
	sessionHash := h.gatewayService.GenerateSessionHash(parsedReq)
	platform := ""
	if forcePlatform, ok := middleware2.GetForcePlatformFromContext(c); ok {
		platform = forcePlatform
	} else if apiKey.Group != nil {
		platform = apiKey.Group.Platform
	}
	sessionKey := sessionHash
	if platform == service.PlatformGemini && sessionHash != "" {
		sessionKey = "gemini:" + sessionHash
	}
	var sessionBoundAccountID int64
	if sessionKey != "" {
		sessionBoundAccountID, _ = h.gatewayService.GetCachedSessionAccountID(c.Request.Context(), apiKey.GroupID, sessionKey)
		if sessionBoundAccountID > 0 {
			prefetchedGroupID := int64(0)
			if apiKey.GroupID != nil {
				prefetchedGroupID = *apiKey.GroupID
			}
			ctx := service.WithPrefetchedStickySession(c.Request.Context(), sessionBoundAccountID, prefetchedGroupID, h.metadataBridgeEnabled())
			c.Request = c.Request.WithContext(ctx)
		}
	}
	hasBoundSession := sessionKey != "" && sessionBoundAccountID > 0
	if platform == service.PlatformGemini {
		fs := NewFailoverState(h.maxAccountSwitchesGemini, hasBoundSession)
		if h.gatewayService.IsSingleAntigravityAccountGroup(c.Request.Context(), apiKey.GroupID) {
			ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
			c.Request = c.Request.WithContext(ctx)
		}
		for {
			selection, err := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), apiKey.GroupID, sessionKey, selectionModel, fs.FailedAccountIDs, "")
			if err != nil {
				if len(fs.FailedAccountIDs) == 0 {
					h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts: "+err.Error(), streamStarted)
					return
				}
				action := fs.HandleSelectionExhausted(c.Request.Context())
				switch action {
				case FailoverContinue:
					ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
					c.Request = c.Request.WithContext(ctx)
					continue
				case FailoverCanceled:
					return
				default:
					if fs.LastFailoverErr != nil {
						h.handleFailoverExhausted(c, fs.LastFailoverErr, service.PlatformGemini, streamStarted)
					} else {
						h.handleFailoverExhaustedSimple(c, 502, streamStarted)
					}
					return
				}
			}
			account := selection.Account
			setOpsSelectedAccountDetails(c, account)
			setOpsEndpointContext(c, account.GetMappedModel(selectionModel), service.RequestTypeFromLegacy(reqStream, false))
			if account.IsInterceptWarmupEnabled() {
				interceptType := detectInterceptType(body, reqModel, parsedReq.MaxTokens, reqStream, isClaudeCodeClient)
				if interceptType != InterceptTypeNone {
					if selection.Acquired && selection.ReleaseFunc != nil {
						selection.ReleaseFunc()
					}
					if reqStream {
						sendMockInterceptStream(c, reqModel, interceptType)
					} else {
						sendMockInterceptResponse(c, reqModel, interceptType)
					}
					return
				}
			}
			accountReleaseFunc := selection.ReleaseFunc
			if !selection.Acquired {
				if selection.WaitPlan == nil {
					h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
					return
				}
				accountWaitCounted := false
				canWait, err := h.concurrencyHelper.IncrementAccountWaitCount(c.Request.Context(), account.ID, selection.WaitPlan.MaxWaiting)
				if err != nil {
					reqLog.Warn("gateway.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				} else if !canWait {
					reqLog.Info("gateway.account_wait_queue_full", zap.Int64("account_id", account.ID), zap.Int("max_waiting", selection.WaitPlan.MaxWaiting))
					h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", streamStarted)
					return
				}
				if err == nil && canWait {
					accountWaitCounted = true
				}
				releaseWait := func() {
					if accountWaitCounted {
						h.concurrencyHelper.DecrementAccountWaitCount(c.Request.Context(), account.ID)
						accountWaitCounted = false
					}
				}
				accountReleaseFunc, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(c, account.ID, selection.WaitPlan.MaxConcurrency, selection.WaitPlan.Timeout, reqStream, &streamStarted)
				if err != nil {
					reqLog.Warn("gateway.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
					releaseWait()
					h.handleConcurrencyError(c, err, "account", streamStarted)
					return
				}
				releaseWait()
				if err := h.gatewayService.BindStickySession(c.Request.Context(), apiKey.GroupID, sessionKey, account.ID); err != nil {
					reqLog.Warn("gateway.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				}
			}
			accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)
			var result *service.ForwardResult
			requestCtx := c.Request.Context()
			if fs.SwitchCount > 0 {
				requestCtx = service.WithAccountSwitchCount(requestCtx, fs.SwitchCount, h.metadataBridgeEnabled())
			}
			if account.Platform == service.PlatformAntigravity {
				result, err = h.antigravityGatewayService.ForwardGemini(requestCtx, c, account, reqModel, "generateContent", reqStream, body, hasBoundSession)
			} else {
				result, err = h.geminiCompatService.Forward(requestCtx, c, account, body)
			}
			if accountReleaseFunc != nil {
				accountReleaseFunc()
			}
			if err != nil {
				var failoverErr *service.UpstreamFailoverError
				if errors.As(err, &failoverErr) {
					action := fs.HandleFailoverError(c.Request.Context(), h.gatewayService, account.ID, account.Platform, failoverErr)
					switch action {
					case FailoverContinue:
						continue
					case FailoverExhausted:
						h.submitFailedUsageRecordTask(
							"handler.gateway.messages",
							c,
							apiKey,
							subscription,
							account,
							reqModel,
							reqStream,
							0,
							service.EffectiveProtocol(account),
							fs.LastFailoverErr,
							err,
						)
						h.handleFailoverExhausted(c, fs.LastFailoverErr, service.PlatformGemini, streamStarted)
						return
					case FailoverCanceled:
						return
					}
				}
				wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
				h.submitFailedUsageRecordTask(
					"handler.gateway.messages",
					c,
					apiKey,
					subscription,
					account,
					reqModel,
					reqStream,
					0,
					service.EffectiveProtocol(account),
					nil,
					err,
				)
				reqLog.Error("gateway.forward_failed", forwardFailedLogFields(account, wroteFallback, err)...)
				return
			}
			if account.IsAnthropicOAuthOrSetupToken() && account.GetBaseRPM() > 0 {
				if err := h.gatewayService.IncrementAccountRPM(c.Request.Context(), account.ID); err != nil {
					reqLog.Warn("gateway.rpm_increment_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				}
			}
			userAgent := c.GetHeader("User-Agent")
			clientIP := ip.GetClientIP(c)
			h.submitUsageRecordTask(func(ctx context.Context) {
				if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{Result: result, APIKey: apiKey, User: apiKey.User, Account: account, Subscription: subscription, ThinkingEnabled: service.ParseExplicitThinkingEnabledValue(body), InboundEndpoint: GetInboundEndpoint(c), UpstreamEndpoint: GetUpstreamEndpointForAccount(c, account), UserAgent: userAgent, IPAddress: clientIP, ForceCacheBilling: fs.ForceCacheBilling, APIKeyService: h.apiKeyService}); err != nil {
					logger.L().With(zap.String("component", "handler.gateway.messages"), zap.Int64("user_id", subject.UserID), zap.Int64("api_key_id", apiKey.ID), zap.Any("group_id", apiKey.GroupID), zap.String("model", reqModel), zap.Int64("account_id", account.ID)).Error("gateway.record_usage_failed", zap.Error(err))
				}
			})
			return
		}
	}
	currentAPIKey := apiKey
	currentSubscription := subscription
	var fallbackGroupID *int64
	if apiKey.Group != nil {
		fallbackGroupID = apiKey.Group.FallbackGroupIDOnInvalidRequest
	}
	fallbackUsed := false
	if h.gatewayService.IsSingleAntigravityAccountGroup(c.Request.Context(), currentAPIKey.GroupID) {
		ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
		c.Request = c.Request.WithContext(ctx)
	}
	for {
		fs := NewFailoverState(h.maxAccountSwitches, hasBoundSession)
		retryWithFallback := false
		for {
			selection, err := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), currentAPIKey.GroupID, sessionKey, selectionModel, fs.FailedAccountIDs, parsedReq.MetadataUserID)
			if err != nil {
				if len(fs.FailedAccountIDs) == 0 {
					h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts: "+err.Error(), streamStarted)
					return
				}
				action := fs.HandleSelectionExhausted(c.Request.Context())
				switch action {
				case FailoverContinue:
					ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
					c.Request = c.Request.WithContext(ctx)
					continue
				case FailoverCanceled:
					return
				default:
					if fs.LastFailoverErr != nil {
						h.handleFailoverExhausted(c, fs.LastFailoverErr, platform, streamStarted)
					} else {
						h.handleFailoverExhaustedSimple(c, 502, streamStarted)
					}
					return
				}
			}
			account := selection.Account
			setOpsSelectedAccountDetails(c, account)
			setOpsEndpointContext(c, account.GetMappedModel(selectionModel), service.RequestTypeFromLegacy(reqStream, false))
			if account.IsInterceptWarmupEnabled() {
				interceptType := detectInterceptType(body, reqModel, parsedReq.MaxTokens, reqStream, isClaudeCodeClient)
				if interceptType != InterceptTypeNone {
					if selection.Acquired && selection.ReleaseFunc != nil {
						selection.ReleaseFunc()
					}
					if reqStream {
						sendMockInterceptStream(c, reqModel, interceptType)
					} else {
						sendMockInterceptResponse(c, reqModel, interceptType)
					}
					return
				}
			}
			accountReleaseFunc := selection.ReleaseFunc
			if !selection.Acquired {
				if selection.WaitPlan == nil {
					h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
					return
				}
				accountWaitCounted := false
				canWait, err := h.concurrencyHelper.IncrementAccountWaitCount(c.Request.Context(), account.ID, selection.WaitPlan.MaxWaiting)
				if err != nil {
					reqLog.Warn("gateway.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				} else if !canWait {
					reqLog.Info("gateway.account_wait_queue_full", zap.Int64("account_id", account.ID), zap.Int("max_waiting", selection.WaitPlan.MaxWaiting))
					h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", streamStarted)
					return
				}
				if err == nil && canWait {
					accountWaitCounted = true
				}
				releaseWait := func() {
					if accountWaitCounted {
						h.concurrencyHelper.DecrementAccountWaitCount(c.Request.Context(), account.ID)
						accountWaitCounted = false
					}
				}
				accountReleaseFunc, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(c, account.ID, selection.WaitPlan.MaxConcurrency, selection.WaitPlan.Timeout, reqStream, &streamStarted)
				if err != nil {
					reqLog.Warn("gateway.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
					releaseWait()
					h.handleConcurrencyError(c, err, "account", streamStarted)
					return
				}
				releaseWait()
				if err := h.gatewayService.BindStickySession(c.Request.Context(), currentAPIKey.GroupID, sessionKey, account.ID); err != nil {
					reqLog.Warn("gateway.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				}
			}
			accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)
			var queueRelease func()
			umqMode := h.getUserMsgQueueMode(account, parsedReq)
			switch umqMode {
			case config.UMQModeSerialize:
				baseRPM := account.GetBaseRPM()
				release, qErr := h.userMsgQueueHelper.AcquireWithWait(c, account.ID, baseRPM, reqStream, &streamStarted, h.cfg.Gateway.UserMessageQueue.WaitTimeout(), reqLog)
				if qErr != nil {
					reqLog.Warn("gateway.umq_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(qErr))
				} else {
					queueRelease = release
				}
			case config.UMQModeThrottle:
				baseRPM := account.GetBaseRPM()
				if tErr := h.userMsgQueueHelper.ThrottleWithPing(c, account.ID, baseRPM, reqStream, &streamStarted, h.cfg.Gateway.UserMessageQueue.WaitTimeout(), reqLog); tErr != nil {
					reqLog.Warn("gateway.umq_throttle_failed", zap.Int64("account_id", account.ID), zap.Error(tErr))
				}
			default:
				if umqMode != "" {
					reqLog.Warn("gateway.umq_unknown_mode", zap.String("mode", umqMode), zap.Int64("account_id", account.ID))
				}
			}
			queueRelease = wrapReleaseOnDone(c.Request.Context(), queueRelease)
			parsedReq.OnUpstreamAccepted = queueRelease
			var result *service.ForwardResult
			requestCtx := c.Request.Context()
			if fs.SwitchCount > 0 {
				requestCtx = service.WithAccountSwitchCount(requestCtx, fs.SwitchCount, h.metadataBridgeEnabled())
			}
			if account.Platform == service.PlatformAntigravity && account.Type != service.AccountTypeAPIKey {
				result, err = h.antigravityGatewayService.Forward(requestCtx, c, account, body, hasBoundSession)
			} else {
				result, err = h.gatewayService.Forward(requestCtx, c, account, parsedReq)
			}
			if queueRelease != nil {
				queueRelease()
			}
			parsedReq.OnUpstreamAccepted = nil
			if accountReleaseFunc != nil {
				accountReleaseFunc()
			}
			if err != nil {
				var betaBlockedErr *service.BetaBlockedError
				if errors.As(err, &betaBlockedErr) {
					h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", betaBlockedErr.Message)
					return
				}
				var promptTooLongErr *service.PromptTooLongError
				if errors.As(err, &promptTooLongErr) {
					reqLog.Warn("gateway.prompt_too_long_from_antigravity", zap.Any("current_group_id", currentAPIKey.GroupID), zap.Any("fallback_group_id", fallbackGroupID), zap.Bool("fallback_used", fallbackUsed))
					if !fallbackUsed && fallbackGroupID != nil && *fallbackGroupID > 0 {
						fallbackGroup, err := h.gatewayService.ResolveGroupByID(c.Request.Context(), *fallbackGroupID)
						if err != nil {
							reqLog.Warn("gateway.resolve_fallback_group_failed", zap.Int64("fallback_group_id", *fallbackGroupID), zap.Error(err))
							_ = h.antigravityGatewayService.WriteMappedClaudeError(c, account, promptTooLongErr.StatusCode, promptTooLongErr.RequestID, promptTooLongErr.Body)
							return
						}
						if fallbackGroup.Platform != service.PlatformAnthropic || fallbackGroup.SubscriptionType == service.SubscriptionTypeSubscription || fallbackGroup.FallbackGroupIDOnInvalidRequest != nil {
							reqLog.Warn("gateway.fallback_group_invalid", zap.Int64("fallback_group_id", fallbackGroup.ID), zap.String("fallback_platform", fallbackGroup.Platform), zap.String("fallback_subscription_type", fallbackGroup.SubscriptionType))
							_ = h.antigravityGatewayService.WriteMappedClaudeError(c, account, promptTooLongErr.StatusCode, promptTooLongErr.RequestID, promptTooLongErr.Body)
							return
						}
						fallbackAPIKey := cloneAPIKeyWithGroup(apiKey, fallbackGroup)
						if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), fallbackAPIKey.User, fallbackAPIKey, fallbackGroup, nil); err != nil {
							status, code, message := billingErrorDetails(err)
							h.handleStreamingAwareError(c, status, code, message, streamStarted)
							return
						}
						ctx := context.WithValue(c.Request.Context(), ctxkey.ForcePlatform, "")
						c.Request = c.Request.WithContext(ctx)
						currentAPIKey = fallbackAPIKey
						currentSubscription = nil
						fallbackUsed = true
						retryWithFallback = true
						break
					}
					_ = h.antigravityGatewayService.WriteMappedClaudeError(c, account, promptTooLongErr.StatusCode, promptTooLongErr.RequestID, promptTooLongErr.Body)
					return
				}
				var failoverErr *service.UpstreamFailoverError
				if errors.As(err, &failoverErr) {
					action := fs.HandleFailoverError(c.Request.Context(), h.gatewayService, account.ID, account.Platform, failoverErr)
					switch action {
					case FailoverContinue:
						continue
					case FailoverExhausted:
						h.submitFailedUsageRecordTask(
							"handler.gateway.messages",
							c,
							currentAPIKey,
							currentSubscription,
							account,
							reqModel,
							reqStream,
							0,
							service.EffectiveProtocol(account),
							fs.LastFailoverErr,
							err,
						)
						h.handleFailoverExhausted(c, fs.LastFailoverErr, account.Platform, streamStarted)
						return
					case FailoverCanceled:
						return
					}
				}
				wroteFallback := h.ensureForwardErrorResponse(c, streamStarted)
				h.submitFailedUsageRecordTask(
					"handler.gateway.messages",
					c,
					currentAPIKey,
					currentSubscription,
					account,
					reqModel,
					reqStream,
					0,
					service.EffectiveProtocol(account),
					nil,
					err,
				)
				reqLog.Error("gateway.forward_failed", forwardFailedLogFields(account, wroteFallback, err)...)
				return
			}
			if account.IsAnthropicOAuthOrSetupToken() && account.GetBaseRPM() > 0 {
				if err := h.gatewayService.IncrementAccountRPM(c.Request.Context(), account.ID); err != nil {
					reqLog.Warn("gateway.rpm_increment_failed", zap.Int64("account_id", account.ID), zap.Error(err))
				}
			}
			userAgent := c.GetHeader("User-Agent")
			clientIP := ip.GetClientIP(c)
			h.submitUsageRecordTask(func(ctx context.Context) {
				if err := h.gatewayService.RecordUsage(ctx, &service.RecordUsageInput{Result: result, APIKey: currentAPIKey, User: currentAPIKey.User, Account: account, Subscription: currentSubscription, ThinkingEnabled: service.ParseExplicitThinkingEnabledValue(body), InboundEndpoint: GetInboundEndpoint(c), UpstreamEndpoint: GetUpstreamEndpointForAccount(c, account), UserAgent: userAgent, IPAddress: clientIP, ForceCacheBilling: fs.ForceCacheBilling, APIKeyService: h.apiKeyService}); err != nil {
					logger.L().With(zap.String("component", "handler.gateway.messages"), zap.Int64("user_id", subject.UserID), zap.Int64("api_key_id", currentAPIKey.ID), zap.Any("group_id", currentAPIKey.GroupID), zap.String("model", reqModel), zap.Int64("account_id", account.ID)).Error("gateway.record_usage_failed", zap.Error(err))
				}
			})
			return
		}
		if !retryWithFallback {
			return
		}
	}
}
*/
