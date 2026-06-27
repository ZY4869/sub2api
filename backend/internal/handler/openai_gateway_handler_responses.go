package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
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
	if apiKey.ImageOnlyEnabled && c.Request != nil && strings.ToUpper(strings.TrimSpace(c.Request.Method)) != http.MethodPost {
		h.errorResponseWithCode(c, http.StatusForbidden, "forbidden_error", "IMAGE_ONLY_KEY_REQUEST_NOT_IMAGE", "生图专用 Key 仅允许生图请求")
		return
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

	contentType := strings.TrimSpace(c.GetHeader("Content-Type"))
	setOpsRequestContext(c, "", false, body)
	var sessionHashBody []byte
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
	sessionHashBody = body

	compatResult, compatErr := service.NormalizeOpenAIResponsesImageGenCompat(body, contentType)
	if compatErr != nil {
		var requestErr *service.OpenAIResponsesCompatError
		if errors.As(compatErr, &requestErr) {
			compatMetadata := requestErr.Metadata
			compatMetadata.Enabled = false
			compatMetadata.Rejected = true
			if strings.TrimSpace(compatMetadata.RejectCode) == "" {
				compatMetadata.RejectCode = strings.TrimSpace(requestErr.Code)
			}
			if strings.TrimSpace(compatMetadata.SourceGuess) == "" {
				compatMetadata.SourceGuess = detectOpenAIResponsesCompatSourceGuess(body, contentType)
			}
			if c.Request != nil {
				ctx := service.EnsureRequestMetadata(c.Request.Context())
				service.SetOpenAIResponsesImageGenCompatMetadata(ctx, compatMetadata)
				c.Request = c.Request.WithContext(ctx)
			}
			protocolruntime.RecordResponsesImagegenReject(compatMetadata.RejectCode)
			requestModel := detectOpenAIResponsesCompatRequestModel(body, contentType)
			setResponsesImagegenCompatTracePayload(c, requestModel, contentType, compatMetadata, nil)
			reqLog.Warn(
				"openai.responses_imagegen_compat_rejected",
				zap.String("request_id", openAIResponsesCompatRequestID(c)),
				zap.String("correlation_id", openAIResponsesCompatCorrelationID(c)),
				zap.String("code", compatMetadata.RejectCode),
				zap.String("source", compatMetadata.SourceGuess),
				zap.String("model", requestModel),
				zap.String("content_type", contentType),
				zap.Int("reference_image_count", compatMetadata.ReferenceImageCount),
			)
			h.errorResponseWithCode(c, requestErr.Status, requestErr.Type, requestErr.Code, requestErr.Message)
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to normalize responses image generation request")
		return
	}
	if compatResult != nil {
		body = compatResult.Body
		sessionHashBody = body
		if strings.TrimSpace(compatResult.ContentType) != "" && c.Request != nil {
			c.Request.Header.Set("Content-Type", compatResult.ContentType)
		}
		if compatResult.ParsedBody != nil {
			c.Set(service.OpenAIParsedRequestBodyKey, compatResult.ParsedBody)
		}
		if compatResult.Metadata.Enabled {
			if c.Request != nil {
				ctx := service.EnsureRequestMetadata(c.Request.Context())
				service.SetOpenAIResponsesImageGenCompatMetadata(ctx, compatResult.Metadata)
				c.Request = c.Request.WithContext(ctx)
			}
			protocolruntime.RecordResponsesImagegenCompat(compatResult.Metadata.Source)
			if compatResult.Metadata.ReferenceImagesNormalized {
				protocolruntime.RecordResponsesImagegenNormalized(compatResult.Metadata.Source)
			}
			setResponsesImagegenCompatTracePayload(
				c,
				detectOpenAIResponsesCompatRequestModel(body, compatResult.ContentType),
				compatResult.ContentType,
				compatResult.Metadata,
				compatResult.TraceTool,
			)
		}
	}
	setOpsRequestContext(c, "", false, body)

	// 校验请求体 JSON 合法性
	if !gjson.ValidBytes(body) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	modelHint := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	moderationInput := buildContentModerationRecordInput(c, service.ContentModerationSourceOpenAIResponses, service.PlatformOpenAI, modelHint, body)
	if decision, err := checkContentModerationKeywordBlock(c.Request.Context(), h.contentModerationService, moderationInput); err != nil {
		reqLog.Warn("openai.content_moderation_keyword_check_failed", zap.Error(err))
	} else if decision != nil {
		h.submitContentModerationFailedUsageRecordTask(
			"handler.openai_gateway.responses",
			c,
			apiKey,
			modelHint,
			gjson.GetBytes(body, "stream").Bool(),
			decision,
		)
		contentModerationOpenAIBlockResponse(c, decision)
		return
	}
	submitContentModerationAudit(c.Request.Context(), h.contentModerationService, moderationInput)

	// 使用 gjson 只读提取字段做校验，避免完整 Unmarshal
	modelResult := gjson.GetBytes(body, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String || modelResult.String() == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	reqModel := modelResult.String()
	var publicCatalogEntry *service.PublishedPublicCatalogEntry
	publicRequestModel := reqModel
	routingRequestModel := reqModel
	if entry, status, resolveErr := h.gatewayService.ResolveAPIKeyPublishedPublicCatalogRuntimeStatus(c.Request.Context(), apiKey, service.OpenAIPlatformFromContext(c.Request.Context()), reqModel); resolveErr != nil {
		reqLog.Warn("openai.responses.public_catalog_entry_resolve_failed", zap.Error(resolveErr))
	} else if status == service.PublicCatalogResolutionNoMatch || status == service.PublicCatalogResolutionTimeWindowDenied {
		h.publicCatalogUnavailableResponse(c, status)
		return
	} else if status == service.PublicCatalogResolutionMatched {
		publicCatalogEntry = entry
		routingRequestModel = service.NormalizeModelCatalogModelID(firstNonEmptyHandlerString(entry.SourceModelID, reqModel))
		c.Request = c.Request.WithContext(service.AttachPublishedPublicCatalogEntry(c.Request.Context(), entry))
	}
	imageToolModel, hasImageTool := detectResponsesImageToolRequest(body)
	if apiKey.ImageOnlyEnabled {
		if !hasImageTool {
			h.errorResponseWithCode(c, http.StatusForbidden, "forbidden_error", "IMAGE_ONLY_KEY_REQUEST_NOT_IMAGE", "生图专用 Key 仅允许生图请求，请使用 image_generation tool 或图片生成接口")
			return
		}
		imageOnlyModel := firstNonEmptyHandlerString(imageToolModel, routingRequestModel, reqModel)
		imageOnlyModelAllowed := service.APIKeyAllowsConfiguredModel(apiKey, imageOnlyModel)
		if imageToolModel == "" {
			imageOnlyModelAllowed = imageOnlyModelAllowed || service.APIKeyAllowsConfiguredModel(apiKey, reqModel)
			if routingRequestModel != reqModel {
				imageOnlyModelAllowed = imageOnlyModelAllowed || service.APIKeyAllowsConfiguredModel(apiKey, routingRequestModel)
			}
		}
		if !service.IsOpenAINativeImageModelID(imageOnlyModel) || !imageOnlyModelAllowed {
			h.errorResponseWithCode(c, http.StatusForbidden, "forbidden_error", "IMAGE_ONLY_KEY_MODEL_NOT_ALLOWED", "生图专用 Key 仅允许调用已授权的图片模型")
			return
		}
	}
	expectedImageCount := 1
	imageSizeTier := service.OpenAIImageSizeTier2K
	if hasImageTool {
		n := int(gjson.GetBytes(body, `tools.#(type=="image_generation").n`).Int())
		if n > 0 {
			expectedImageCount = n
		}
		imageSizeTier = service.ResolveOpenAIResponsesImageToolSizeTier(body)
	}
	reservedImageUnits := 0
	imageCountSettled := false

	reqStream, ok := parseOpenAIStreamFlag(body)
	if !ok {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "invalid stream field type")
		return
	}
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
	requestPayloadHash := service.HashUsageRequestPayload(body)
	if hasImageTool {
		applyResponsesImageToolTraceMetadata(c, service.PlatformOpenAI, reqModel, imageToolModel, service.PublicImageToolRouteReason)
	}

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

	if apiKey.EffectiveImageCountBillingEnabled() {
		reservedImageUnits = apiKey.ImageCountUnitsForTier(expectedImageCount, imageSizeTier)
		ok, reserveErr := h.apiKeyService.TryReserveImageCount(c.Request.Context(), apiKey.ID, reservedImageUnits)
		if reserveErr != nil {
			reqLog.Error("api_key_image_count_reserve_failed", zap.Error(reserveErr), zap.String("image_size_tier", imageSizeTier), zap.Int("image_count", expectedImageCount), zap.Int("reserved_units", reservedImageUnits))
			h.errorResponse(c, http.StatusInternalServerError, "api_error", "Failed to reserve image quota")
			return
		}
		if !ok {
			h.errorResponseWithCode(c, http.StatusTooManyRequests, "rate_limit_error", "IMAGE_ONLY_KEY_IMAGE_QUOTA_EXHAUSTED", "图片数量额度已用完")
			return
		}
		reqLog.Info("api_key_image_count_reserved", zap.String("image_size_tier", imageSizeTier), zap.Int("image_count", expectedImageCount), zap.Int("reserved_units", reservedImageUnits), zap.Int("max", apiKey.ImageMaxCount))
		defer func() {
			if reservedImageUnits <= 0 || imageCountSettled {
				return
			}
			if err := h.apiKeyService.RollbackImageCount(c.Request.Context(), apiKey.ID, reservedImageUnits); err != nil {
				reqLog.Error("api_key_image_count_rollback_failed", zap.Error(err), zap.String("image_size_tier", imageSizeTier), zap.Int("rollback_units", reservedImageUnits))
				return
			}
			reqLog.Info("api_key_image_count_rolled_back", zap.String("image_size_tier", imageSizeTier), zap.Int("rollback_units", reservedImageUnits))
		}()
	}

	// Generate session hash from stable, non-body-logging anchors.
	sessionHash, sessionHashSource, sessionHashSeedLen := h.gatewayService.ResolveSessionHashWithSource(c, sessionHashBody, "")
	if sessionHash != "" {
		reqLog.Debug(
			"openai.responses.session_hash_resolved",
			zap.String("source", sessionHashSource),
			zap.Int("seed_len", sessionHashSeedLen),
			zap.String("session_hash", sessionHash),
		)
	}
	excludedGroupIDs := make(map[int64]struct{})
	maxAccountSwitches := h.maxAccountSwitches
	sawQuotaOnlyGroupFailure := false
	sawNonQuotaGroupFailure := false

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
			publicRequestModel,
			openAICompatiblePlatforms,
			excludedGroupIDs,
		)
		if err != nil {
			if isRequestCanceled(c.Request.Context(), err) {
				return
			}
			releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, apiKey)
			reqLog.Info("openai.group_selection_failed", zap.Error(err))
			status, code, message := groupSelectionErrorDetails(err)
			h.handleStreamingAwareError(c, status, code, message, streamStarted)
			return
		}
		if currentAPIKey.Group != nil {
			applyOpenAIPlatformContext(c, currentAPIKey.Group.Platform)
		}
		channelSelectionModel, channelState, err := bindGatewayChannelState(c, h.gatewayService, currentAPIKey.Group, publicRequestModel)
		if err != nil {
			releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
			if errors.Is(err, service.ErrChannelModelNotAllowed) {
				h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "Requested model is not allowed by the bound channel", streamStarted)
				return
			}
			if errors.Is(err, service.ErrModelHardRemoved) {
				h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "Requested model is no longer available", streamStarted)
				return
			}
			h.handleStreamingAwareError(c, http.StatusInternalServerError, "api_error", "Failed to resolve channel routing", streamStarted)
			return
		}
		runtimeSelectionModel := channelSelectionModel
		if publicCatalogEntry != nil {
			runtimeSelectionModel = routingRequestModel
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
				runtimeSelectionModel,
				failedAccountIDs,
				service.OpenAIUpstreamTransportAny,
			)
			if err != nil {
				if isRequestCanceled(c.Request.Context(), err) {
					return
				}
				quotaOnlyGroupFailure := h.isOpenAIRuntimeQuotaOnlySelectionFailure(
					c.Request.Context(),
					currentAPIKey.GroupID,
					runtimeSelectionModel,
				)
				if quotaOnlyGroupFailure {
					sawQuotaOnlyGroupFailure = true
				} else {
					sawNonQuotaGroupFailure = true
				}
				reqLog.Warn("openai.account_select_failed",
					zap.Error(err),
					zap.Int("excluded_account_count", len(failedAccountIDs)),
					zap.Bool("quota_only_group_failure", quotaOnlyGroupFailure),
				)
				if len(failedAccountIDs) == 0 {
					if sawQuotaOnlyGroupFailure && !sawNonQuotaGroupFailure {
						releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
						h.handleOpenAIRuntimeQuotaUnavailable(c, streamStarted)
						return
					}
					if errors.Is(err, service.ErrOpenAIModelNotFound) {
						releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
						h.handleOpenAIModelNotFound(c, publicRequestModel, streamStarted)
						return
					}
					if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
						break
					}
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable", streamStarted)
					return
				}
				if lastFailoverErr != nil {
					if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
						break
					}
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					h.handleFailoverExhausted(c, lastFailoverErr, streamStarted)
				} else {
					if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
						break
					}
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					h.handleFailoverExhaustedSimple(c, 502, streamStarted)
				}
				return
			}
			if selection == nil || selection.Account == nil {
				quotaOnlyGroupFailure := h.isOpenAIRuntimeQuotaOnlySelectionFailure(
					c.Request.Context(),
					currentAPIKey.GroupID,
					runtimeSelectionModel,
				)
				if quotaOnlyGroupFailure {
					sawQuotaOnlyGroupFailure = true
				} else {
					sawNonQuotaGroupFailure = true
				}
				if sawQuotaOnlyGroupFailure && !sawNonQuotaGroupFailure {
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					h.handleOpenAIRuntimeQuotaUnavailable(c, streamStarted)
					return
				}
				if h.gatewayService.IsModelUnavailableBecauseUnsupported(c.Request.Context(), currentAPIKey.GroupID, runtimeSelectionModel, nil) {
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					h.handleOpenAIModelNotFound(c, publicRequestModel, streamStarted)
					return
				}
				if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
					break
				}
				releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
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
			if hasImageTool && !supportsResponsesImageToolPlatform(account.Platform) {
				applyResponsesImageToolTraceMetadata(
					c,
					account.Platform,
					reqModel,
					imageToolModel,
					service.PublicImageToolRouteReasonRejected,
				)
				reqLog.Warn(
					"openai.responses_image_tool_platform_rejected",
					zap.String("account_platform", account.Platform),
					zap.String("requested_model", reqModel),
					zap.String("tool_model", imageToolModel),
				)
				h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", responsesImageToolUnsupportedPlatformMessage())
				releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
				return
			}
			if hasImageTool {
				imageToolTargetModel, imageToolTargetSupported := resolveResponsesImageToolOpenAITargetModel(account, imageToolModel)
				if !imageToolTargetSupported {
					applyResponsesImageToolTraceMetadata(
						c,
						account.Platform,
						reqModel,
						imageToolModel,
						service.PublicImageToolRouteReasonRejected,
					)
					reqLog.Warn(
						"openai.responses_image_tool_model_rejected",
						zap.String("account_platform", account.Platform),
						zap.String("requested_model", reqModel),
						zap.String("tool_model", imageToolModel),
					)
					h.errorResponseWithCode(
						c,
						http.StatusBadRequest,
						"invalid_request_error",
						"image_tool_model_provider_unsupported",
						responsesImageToolUnsupportedModelMessage(imageToolModel),
					)
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					return
				}
				imageProtocolMode := service.ResolveEffectiveOpenAIImageProtocolMode(currentAPIKey.Group, account)
				if service.ResolveOpenAITextRequestFormatForAccount(account, service.EndpointResponses) == service.GatewayOpenAIRequestFormatChatCompletions {
					h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "Responses image_generation tool is not supported when the selected upstream is forced to chat completions", streamStarted)
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					return
				}
				if imageProtocolMode == service.OpenAIImageProtocolModeCompat && !service.IsOpenAIImageCompatAllowed(account) {
					h.errorResponseWithCode(c, http.StatusForbidden, "forbidden_error", "image_compat_not_allowed", "This account does not allow compat image generation")
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					return
				}
				normalizedImageToolRequest, normalizeErr := service.NormalizeOpenAIResponsesImageToolRequest(body)
				if normalizeErr != nil {
					h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to normalize image_generation tool request")
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					return
				}
				if strings.TrimSpace(imageToolTargetModel) != "" {
					normalizedImageToolRequest.TargetModelID = imageToolTargetModel
				}
				if imageProtocolMode == service.OpenAIImageProtocolModeCompat {
					normalizedImageToolRequest.TargetModelID = service.OpenAICompatImageTargetModel
				}
				if normalizedImageToolRequest.N != nil && *normalizedImageToolRequest.N > 1 {
					h.errorResponseWithCode(
						c,
						http.StatusBadRequest,
						"invalid_request_error",
						"image_n_not_supported",
						"Responses image_generation tool does not support n>1; call /v1/images/generations or repeat the request instead",
					)
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					return
				}
				capabilityProfile, capabilityErr := service.ValidateOpenAIImageCapabilities(normalizedImageToolRequest, imageProtocolMode, normalizedImageToolRequest.TargetModelID)
				if capabilityErr != nil {
					var imageRequestErr *service.OpenAIImageRequestError
					if errors.As(capabilityErr, &imageRequestErr) {
						h.errorResponseWithCode(c, imageRequestErr.Status, imageRequestErr.Type, imageRequestErr.Code, imageRequestErr.Message)
						releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
						return
					}
					h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", capabilityErr.Error())
					releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
					return
				}
				if compatResult == nil || !compatResult.Metadata.Enabled {
					service.SetOpenAIImageNormalizedTracePayload(c, "openai_responses_image_tool", normalizedImageToolRequest, capabilityProfile.ID)
				}
				if imageProtocolMode == service.OpenAIImageProtocolModeCompat {
					body, err = service.ForceOpenAIResponsesImageToolModel(body, service.OpenAICompatImageTargetModel)
					if err != nil {
						h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to normalize image_generation tool request")
						releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
						return
					}
					imageToolModel = service.OpenAICompatImageTargetModel
					setOpsRequestContext(c, reqModel, reqStream, body)
				} else if strings.TrimSpace(imageToolTargetModel) != "" && imageToolTargetModel != strings.TrimSpace(imageToolModel) {
					body, err = service.RewriteOpenAIResponsesImageToolModel(body, imageToolTargetModel)
					if err != nil {
						h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to normalize image_generation tool request")
						releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
						return
					}
					imageToolModel = imageToolTargetModel
					setOpsRequestContext(c, reqModel, reqStream, body)
				}
				applyResponsesImageToolRuntimeMetadata(
					c,
					account.Platform,
					reqModel,
					imageToolModel,
					imageProtocolMode,
					service.ResolveOpenAIResponsesImageToolAction(body),
					service.ResolveOpenAIResponsesImageToolSizeTier(body),
					capabilityProfile.ID,
				)
			}
			sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
			reqLog.Debug("openai.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
			setOpsSelectedAccountDetails(c, account)
			setOpsEndpointContext(c, account.GetMappedModel(runtimeSelectionModel), service.RequestTypeFromLegacy(reqStream, false))

			accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, currentAPIKey.GroupID, sessionHash, selection, reqStream, &streamStarted, reqLog)
			if !acquired {
				releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
				return
			}

			// Forward request
			service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
			forwardStart := time.Now()
			var result *service.OpenAIForwardResult
			if service.ResolveOpenAITextRequestFormatForAccount(account, service.EndpointResponses) == service.GatewayOpenAIRequestFormatChatCompletions {
				result, err = h.gatewayService.ForwardResponsesAsChatCompletions(c.Request.Context(), c, account, body, runtimeSelectionModel)
			} else {
				result, err = h.gatewayService.Forward(c.Request.Context(), c, account, body)
			}
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
					h.gatewayService.TempUnscheduleFailoverError(c.Request.Context(), account, failoverErr)
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
				if hasImageTool {
					if imageCount, ok := service.ImageOutputCountMetadataFromContext(c.Request.Context()); ok && imageCount > 0 {
						result.ImageCount = imageCount
						result.MediaType = "image"
					}
					if sizeTier, ok := service.ImageSizeTierMetadataFromContext(c.Request.Context()); ok && strings.TrimSpace(sizeTier) != "" {
						result.ImageSize = sizeTier
					}
					if targetModel, ok := service.ImageTargetModelIDMetadataFromContext(c.Request.Context()); ok && strings.TrimSpace(targetModel) != "" {
						result.BillingModel = targetModel
					}
				}
				if account.Type == service.AccountTypeOAuth {
					h.gatewayService.UpdateCodexUsageSnapshotFromHeaders(c.Request.Context(), account.ID, result.ResponseHeaders, result.UpstreamModel, result.Model)
				}
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
			} else {
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
			}

			// 捕获请求信息（用于异步记录，避免在 goroutine 中访问 gin.Context）
			userAgent := c.GetHeader("User-Agent")
			clientIP := ip.GetTrustedClientIP(c)

			if reservedImageUnits > 0 && !imageCountSettled {
				actualCount := expectedImageCount
				actualTier := imageSizeTier
				if result != nil && result.ImageCount > 0 {
					actualCount = result.ImageCount
				}
				if result != nil && strings.TrimSpace(result.ImageSize) != "" {
					actualTier = service.ResolveOpenAIImageSizeTier(result.ImageSize)
				}
				imageCountSettled = settleAPIKeyImageCountUnits(c.Request.Context(), reqLog, h.apiKeyService, apiKey, reservedImageUnits, actualCount, actualTier)
			}

			// 使用量记录通过有界 worker 池提交，避免请求热路径创建无界 goroutine。
			h.submitUsageRecordTask(func(ctx context.Context) {
				ctx = service.AttachPublishedPublicCatalogEntry(ctx, publicCatalogEntry)
				ctx = reattachGatewayChannelState(ctx, channelState)
				if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
					Result:             result,
					APIKey:             currentAPIKey,
					User:               currentAPIKey.User,
					Account:            account,
					Subscription:       currentSubscription,
					InboundEndpoint:    GetInboundEndpoint(c),
					UpstreamEndpoint:   GetUpstreamEndpointForAccount(c, account),
					UserAgent:          userAgent,
					IPAddress:          clientIP,
					RequestPayloadHash: requestPayloadHash,
					APIKeyService:      h.apiKeyService,
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
