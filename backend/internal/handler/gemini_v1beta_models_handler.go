package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GeminiV1BetaModels proxies Gemini native REST endpoints like:
// POST /v1beta/models/{model}:generateContent
// POST /v1beta/models/{model}:streamGenerateContent?alt=sse
func (h *GatewayHandler) GeminiV1BetaModels(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		googleErrorKey(c, http.StatusUnauthorized, "gateway.gemini.invalid_api_key", "Invalid API key")
		return
	}
	authSubject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		googleErrorKey(c, http.StatusInternalServerError, "gateway.gemini.user_context_missing", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.gemini_v1beta.models",
		zap.Int64("user_id", authSubject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)

	if !middleware.HasForcePlatform(c) && !multiGroupRoutingEnabled(c.Request.Context(), apiKey, h.settingService) {
		if apiKey.Group != nil && (apiKey.Group.Platform == service.PlatformKiro || service.IsUnsupportedRuntimePlatform(apiKey.Group.Platform)) {
			googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.unsupported_platform", "Gemini protocol is not supported for this platform")
			return
		}
		if apiKey.Group == nil || (apiKey.Group.Platform != service.PlatformGemini && apiKey.Group.Platform != service.PlatformProtocolGateway) {
			googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.group_platform_invalid", "API key group platform is not gemini")
			return
		}
	}

	modelName, action, ok := h.prepareGeminiModelRoute(c, apiKey)
	if !ok {
		return
	}
	selectionPlatform := selectionPlatformForGeminiRoute(c, apiKey)
	if !h.ensureGeminiProtocolCapability(c, selectionPlatform, modelName, action) {
		return
	}
	if service.GeminiActionEndpoint(action) == service.EndpointGeminiBatches {
		h.GeminiV1BetaBatches(c)
		return
	}
	if service.GeminiActionEndpoint(action) == service.EndpointGeminiEmbeddings {
		h.GeminiV1BetaEmbeddings(c, modelName)
		return
	}

	modelRuntime, ok := h.resolveGeminiModelRuntime(c, reqLog, apiKey, selectionPlatform, modelName)
	if !ok {
		return
	}

	stream := action == "streamGenerateContent"
	reqLog = reqLog.With(zap.String("model", modelRuntime.publicModelName), zap.String("action", action), zap.Bool("stream", stream))

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			googleErrorBodyTooLarge(c, maxErr.Limit)
			return
		}
		googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.read_body_failed", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.body_empty", "Request body is empty")
		return
	}
	moderationInput := buildContentModerationRecordInput(c, service.ContentModerationSourceGeminiGenerate, service.PlatformGemini, modelRuntime.publicModelName, body)
	if decision, err := checkContentModerationKeywordBlock(c.Request.Context(), h.contentModerationService, moderationInput); err != nil {
		reqLog.Warn("gemini.content_moderation_keyword_check_failed", zap.Error(err))
	} else if decision != nil {
		contentModerationGeminiBlockResponse(c, decision)
		return
	}
	submitContentModerationAudit(c.Request.Context(), h.contentModerationService, moderationInput)

	setOpsRequestContext(c, modelName, stream, body)

	subscription, _ := middleware.GetSubscriptionFromContext(c)
	geminiConcurrency := NewConcurrencyHelper(h.concurrencyHelper.concurrencyService, SSEPingFormatNone, 0)
	maxWait := service.CalculateMaxWait(authSubject.Concurrency)
	canWait, err := geminiConcurrency.IncrementWaitCount(c.Request.Context(), authSubject.UserID, maxWait)
	waitCounted := false
	if err != nil {
		reqLog.Warn("gemini.user_wait_counter_increment_failed", zap.Error(err))
	} else if !canWait {
		reqLog.Info("gemini.user_wait_queue_full", zap.Int("max_wait", maxWait))
		googleErrorKey(c, http.StatusTooManyRequests, "gateway.gemini.pending_requests", "Too many pending requests, please retry later")
		return
	}
	if err == nil && canWait {
		waitCounted = true
	}
	defer func() {
		if waitCounted {
			geminiConcurrency.DecrementWaitCount(c.Request.Context(), authSubject.UserID)
		}
	}()

	streamStarted := false
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}
	userReleaseFunc, err := geminiConcurrency.AcquireUserSlotWithWait(c, authSubject.UserID, authSubject.Concurrency, stream, &streamStarted)
	if err != nil {
		reqLog.Warn("gemini.user_slot_acquire_failed", zap.Error(err))
		googleErrorPendingRequests(c)
		return
	}
	if waitCounted {
		geminiConcurrency.DecrementWaitCount(c.Request.Context(), authSubject.UserID)
		waitCounted = false
	}
	userReleaseFunc = wrapReleaseOnDone(c.Request.Context(), userReleaseFunc)
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	selectedForcePlatform, hasSelectedForcePlatform := middleware.GetForcePlatformFromContext(c)
	if hasSelectedForcePlatform && (selectedForcePlatform == service.PlatformKiro || service.IsUnsupportedRuntimePlatform(selectedForcePlatform)) {
		googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.unsupported_platform", "Gemini protocol is not supported for this platform")
		return
	}
	selectedAllowedPlatforms := geminiCompatiblePlatforms
	if hasSelectedForcePlatform && strings.TrimSpace(selectedForcePlatform) != "" {
		selectedAllowedPlatforms = []string{selectedForcePlatform}
	}
	excludedGroupIDs := make(map[int64]struct{})

groupSelectionLoop:
	for {
		currentAPIKey, currentSubscription, err := resolveSelectedGatewayAPIKey(
			c,
			h.settingService,
			h.gatewayService,
			h.billingCacheService,
			apiKey,
			subscription,
			modelRuntime.bindingSelectionModel,
			selectedAllowedPlatforms,
			excludedGroupIDs,
		)
		if err != nil {
			reqLog.Info("gemini.group_selection_failed", zap.Error(err))
			googleErrorFromServiceError(c, err)
			return
		}

		currentPlatform := ""
		if currentAPIKey.Group != nil {
			currentPlatform = currentAPIKey.Group.Platform
		}
		if currentPlatform == service.PlatformKiro || service.IsUnsupportedRuntimePlatform(currentPlatform) {
			if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
				releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, currentAPIKey)
				continue
			}
			releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
			googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.unsupported_platform", "Gemini protocol is not supported for this platform")
			return
		}
		if currentPlatform != service.PlatformGemini && currentPlatform != service.PlatformAntigravity && currentPlatform != service.PlatformProtocolGateway {
			if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
				releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, currentAPIKey)
				continue
			}
			releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
			googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.group_platform_invalid", "API key group platform is not gemini")
			return
		}
		runtimeSelectionModel, channelState, err := bindGatewayChannelState(c, h.gatewayService, currentAPIKey.Group, modelRuntime.bindingSelectionModel)
		if err != nil {
			releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
			if errors.Is(err, service.ErrChannelModelNotAllowed) {
				googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.channel_model_not_allowed", "Requested model is not allowed by the bound channel")
				return
			}
			if errors.Is(err, service.ErrModelHardRemoved) {
				googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.model_hard_removed", "Requested model is no longer available")
				return
			}
			googleErrorKey(c, http.StatusInternalServerError, "gateway.gemini.channel_routing_failed", "Failed to resolve channel routing")
			return
		}
		if modelRuntime.publicCatalogEntry != nil {
			runtimeSelectionModel = modelRuntime.upstreamModelName
		}

		sessionHash := extractGeminiCLISessionHash(c, body)
		if sessionHash == "" {
			parsedReq, _ := service.ParseGatewayRequest(body, domain.PlatformGemini)
			if parsedReq != nil {
				parsedReq.SessionContext = &service.SessionContext{
					ClientIP:  ip.GetTrustedClientIP(c),
					UserAgent: c.GetHeader("User-Agent"),
					APIKeyID:  currentAPIKey.ID,
				}
			}
			sessionHash = h.gatewayService.GenerateSessionHash(parsedReq)
		}
		sessionKey := ""
		if sessionHash != "" {
			sessionKey = "gemini:" + sessionHash
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

		var geminiDigestChain string
		var geminiPrefixHash string
		var geminiSessionUUID string
		var matchedDigestChain string
		useDigestFallback := sessionBoundAccountID == 0

		if useDigestFallback {
			var geminiReq antigravity.GeminiRequest
			if err := json.Unmarshal(body, &geminiReq); err == nil && len(geminiReq.Contents) > 0 {
				geminiDigestChain = service.BuildGeminiDigestChain(&geminiReq)
				if geminiDigestChain != "" {
					userAgent := c.GetHeader("User-Agent")
					clientIP := ip.GetTrustedClientIP(c)
					geminiPrefixHash = service.GenerateGeminiPrefixHash(
						authSubject.UserID,
						currentAPIKey.ID,
						clientIP,
						userAgent,
						currentPlatform,
						modelName,
					)

					foundUUID, foundAccountID, foundMatchedChain, found := h.gatewayService.FindGeminiSession(
						c.Request.Context(),
						derefGroupID(currentAPIKey.GroupID),
						geminiPrefixHash,
						geminiDigestChain,
					)
					if found {
						matchedDigestChain = foundMatchedChain
						sessionBoundAccountID = foundAccountID
						geminiSessionUUID = foundUUID
						reqLog.Info("gemini.digest_fallback_matched",
							zap.String("session_uuid_prefix", safeShortPrefix(foundUUID, 8)),
							zap.Int64("account_id", foundAccountID),
							zap.Any("group_id", currentAPIKey.GroupID),
							zap.String("digest_chain", truncateDigestChain(geminiDigestChain)),
						)

						if sessionKey == "" {
							sessionKey = service.GenerateGeminiDigestSessionKey(geminiPrefixHash, foundUUID)
						}
						_ = h.gatewayService.BindStickySession(c.Request.Context(), currentAPIKey.GroupID, sessionKey, foundAccountID)
					} else {
						geminiSessionUUID = uuid.New().String()
						if sessionKey == "" {
							sessionKey = service.GenerateGeminiDigestSessionKey(geminiPrefixHash, geminiSessionUUID)
						}
					}
				}
			}
		}

		hasBoundSession := sessionKey != "" && sessionBoundAccountID > 0
		cleanedForUnknownBinding := false
		fs := NewFailoverState(h.maxAccountSwitchesGemini, hasBoundSession)

		if h.gatewayService.IsSingleAntigravityAccountGroup(c.Request.Context(), currentAPIKey.GroupID) {
			ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
			c.Request = c.Request.WithContext(ctx)
		}

		for {
			selection, err := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), currentAPIKey.GroupID, sessionKey, runtimeSelectionModel, fs.FailedAccountIDs, "")
			if err != nil {
				if len(fs.FailedAccountIDs) == 0 {
					if multiGroupRoutingEnabled(c.Request.Context(), apiKey, h.settingService) && excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
						releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, currentAPIKey)
						continue groupSelectionLoop
					}
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					googleNoAvailableAccountsError(c, err)
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
					if multiGroupRoutingEnabled(c.Request.Context(), apiKey, h.settingService) && excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
						releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, currentAPIKey)
						continue groupSelectionLoop
					}
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					h.handleGeminiFailoverExhausted(c, fs.LastFailoverErr)
					return
				}
			}
			account := selection.Account
			setOpsSelectedAccountDetails(c, account)
			setOpsEndpointContext(c, account.GetMappedModel(runtimeSelectionModel), service.RequestTypeFromLegacy(stream, false))

			if sessionBoundAccountID > 0 && sessionBoundAccountID != account.ID {
				reqLog.Info("gemini.sticky_session_account_switched",
					zap.Int64("from_account_id", sessionBoundAccountID),
					zap.Int64("to_account_id", account.ID),
					zap.Bool("clean_thought_signature", true),
					zap.Any("group_id", currentAPIKey.GroupID),
				)
				body = service.CleanGeminiNativeThoughtSignatures(body)
				sessionBoundAccountID = account.ID
			} else if sessionKey != "" && sessionBoundAccountID == 0 && !cleanedForUnknownBinding && bytes.Contains(body, []byte(`"thoughtSignature"`)) {
				reqLog.Info("gemini.sticky_session_binding_missing",
					zap.Bool("clean_thought_signature", true),
					zap.Any("group_id", currentAPIKey.GroupID),
				)
				body = service.CleanGeminiNativeThoughtSignatures(body)
				cleanedForUnknownBinding = true
				sessionBoundAccountID = account.ID
			} else if sessionBoundAccountID == 0 {
				sessionBoundAccountID = account.ID
			}

			accountReleaseFunc := selection.ReleaseFunc
			if !selection.Acquired {
				if selection.WaitPlan == nil {
					if multiGroupRoutingEnabled(c.Request.Context(), apiKey, h.settingService) && excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
						releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, currentAPIKey)
						continue groupSelectionLoop
					}
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					googleErrorKey(c, http.StatusServiceUnavailable, "gateway.gemini.no_available_accounts", "No available Gemini accounts")
					return
				}
				accountWaitCounted := false
				canWait, err := geminiConcurrency.IncrementAccountWaitCount(c.Request.Context(), account.ID, selection.WaitPlan.MaxWaiting)
				if err != nil {
					reqLog.Warn("gemini.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", currentAPIKey.GroupID), zap.Error(err))
				} else if !canWait {
					reqLog.Info("gemini.account_wait_queue_full",
						zap.Int64("account_id", account.ID),
						zap.Any("group_id", currentAPIKey.GroupID),
						zap.Int("max_waiting", selection.WaitPlan.MaxWaiting),
					)
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					googleErrorKey(c, http.StatusTooManyRequests, "gateway.gemini.pending_requests", "Too many pending requests, please retry later")
					return
				}
				if err == nil && canWait {
					accountWaitCounted = true
				}
				defer func() {
					if accountWaitCounted {
						geminiConcurrency.DecrementAccountWaitCount(c.Request.Context(), account.ID)
					}
				}()

				accountReleaseFunc, err = geminiConcurrency.AcquireAccountSlotWithWaitTimeout(
					c,
					account.ID,
					selection.WaitPlan.MaxConcurrency,
					selection.WaitPlan.Timeout,
					stream,
					&streamStarted,
				)
				if err != nil {
					reqLog.Warn("gemini.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", currentAPIKey.GroupID), zap.Error(err))
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					googleErrorPendingRequests(c)
					return
				}
				if accountWaitCounted {
					geminiConcurrency.DecrementAccountWaitCount(c.Request.Context(), account.ID)
					accountWaitCounted = false
				}
				if err := h.gatewayService.BindStickySession(c.Request.Context(), currentAPIKey.GroupID, sessionKey, account.ID); err != nil {
					reqLog.Warn("gemini.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", currentAPIKey.GroupID), zap.Error(err))
				}
			}
			accountReleaseFunc = wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc)

			var result *service.ForwardResult
			requestCtx := c.Request.Context()
			if fs.SwitchCount > 0 {
				requestCtx = service.WithAccountSwitchCount(requestCtx, fs.SwitchCount, h.metadataBridgeEnabled())
			}
			if account.Platform == service.PlatformAntigravity && account.Type != service.AccountTypeAPIKey {
				result, err = h.antigravityGatewayService.ForwardGemini(requestCtx, c, account, modelRuntime.upstreamModelName, action, stream, body, hasBoundSession)
				if result != nil && modelRuntime.publicCatalogEntry != nil {
					result.Model = modelRuntime.publicModelName
				}
			} else {
				result, err = h.geminiNativeService.ForwardNative(requestCtx, c, account, modelRuntime.publicModelName, action, stream, body)
			}
			if accountReleaseFunc != nil {
				accountReleaseFunc()
			}
			if err != nil {
				var failoverErr *service.UpstreamFailoverError
				if errors.As(err, &failoverErr) {
					failoverAction := fs.HandleFailoverError(c.Request.Context(), h.gatewayService, account.ID, account.Platform, failoverErr)
					switch failoverAction {
					case FailoverContinue:
						continue
					case FailoverExhausted:
						if multiGroupRoutingEnabled(c.Request.Context(), apiKey, h.settingService) && excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
							releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, currentAPIKey)
							continue groupSelectionLoop
						}
						h.submitFailedUsageRecordTask(
							"handler.gemini_v1beta.models",
							c,
							currentAPIKey,
							currentSubscription,
							account,
							modelName,
							stream,
							0,
							service.PlatformGemini,
							fs.LastFailoverErr,
							err,
						)
						h.handleGeminiFailoverExhausted(c, fs.LastFailoverErr)
						return
					case FailoverCanceled:
						return
					}
				}
				h.submitFailedUsageRecordTask(
					"handler.gemini_v1beta.models",
					c,
					currentAPIKey,
					currentSubscription,
					account,
					modelName,
					stream,
					0,
					service.PlatformGemini,
					nil,
					err,
				)
				reqLog.Error("gemini.forward_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", currentAPIKey.GroupID), zap.Error(err))
				return
			}

			if useDigestFallback && geminiDigestChain != "" && geminiPrefixHash != "" {
				if err := h.gatewayService.SaveGeminiSession(
					c.Request.Context(),
					derefGroupID(currentAPIKey.GroupID),
					geminiPrefixHash,
					geminiDigestChain,
					geminiSessionUUID,
					account.ID,
					matchedDigestChain,
				); err != nil {
					reqLog.Warn("gemini.digest_session_save_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", currentAPIKey.GroupID), zap.Error(err))
				}
			}

			if !h.submitGeminiSuccessUsageRecord(
				c,
				reqLog,
				authSubject,
				currentAPIKey,
				currentSubscription,
				account,
				result,
				body,
				modelName,
				modelRuntime.publicModelName,
				modelRuntime.publicCatalogEntry,
				channelState,
				fs,
			) {
				return
			}
			return
		}
	}
}
