package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *OpenAIGatewayHandler) ImagesGeneration(c *gin.Context) {
	h.handleImagesRequest(c, "generation")
}

func (h *OpenAIGatewayHandler) ImagesEdits(c *gin.Context) {
	h.handleImagesRequest(c, "edits")
}

func (h *OpenAIGatewayHandler) handleImagesRequest(c *gin.Context, action string) {
	requestStart := time.Now()

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
		"handler.openai_gateway.images",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
		zap.String("action", action),
	)

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	reqModel, err := service.DetectOpenAIImageRequestModel(body, c.GetHeader("Content-Type"))
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	if strings.TrimSpace(reqModel) == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	setOpsRequestContext(c, reqModel, false, body)
	reqLog = reqLog.With(zap.String("model", reqModel))
	imageSizeTier := service.ResolveOpenAIImageSizeTier(service.DetectOpenAIImageRequestSize(body, c.GetHeader("Content-Type")))

	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()
	streamStarted := false

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	sessionHash := h.gatewayService.GenerateSessionHash(c, body)
	excludedGroupIDs := make(map[int64]struct{})

	for {
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
			status, code, message := groupSelectionErrorDetails(err)
			h.handleStreamingAwareError(c, status, code, message, false)
			return
		}
		if currentAPIKey.Group != nil {
			applyOpenAIPlatformContext(c, currentAPIKey.Group.Platform)
		}
		selectionModel := reqModel
		if currentAPIKey.Group != nil && service.NormalizeOpenAIGroupImageProtocolMode(currentAPIKey.Group.ImageProtocolMode) == service.OpenAIImageProtocolModeCompat {
			selectionModel = service.OpenAICompatImageTargetModel
		}
		runtimeSelectionModel, _, err := bindGatewayChannelState(c, h.gatewayService, currentAPIKey.Group, selectionModel)
		if err != nil {
			if errors.Is(err, service.ErrChannelModelNotAllowed) {
				h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Requested model is not allowed by the bound channel")
				return
			}
			h.errorResponse(c, http.StatusInternalServerError, "api_error", "Failed to resolve channel routing")
			return
		}

		selection, _, err := h.gatewayService.SelectAccountWithScheduler(
			c.Request.Context(),
			currentAPIKey.GroupID,
			"",
			sessionHash,
			runtimeSelectionModel,
			nil,
			service.OpenAIUpstreamTransportHTTPSSE,
		)
		if err != nil || selection == nil || selection.Account == nil {
			if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
				continue
			}
			h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", false)
			return
		}

		account := selection.Account
		imageProtocolMode := service.ResolveEffectiveOpenAIImageProtocolMode(currentAPIKey.Group, account)
		if imageProtocolMode == service.OpenAIImageProtocolModeCompat && !service.IsOpenAIImageCompatAllowed(account) {
			h.errorResponseWithCode(c, http.StatusForbidden, "forbidden_error", "image_compat_not_allowed", "This account does not allow compat image generation")
			return
		}
		setOpenAIImageTraceMetadata(
			c,
			action,
			imageProtocolMode,
			reqModel,
			reqModel,
			imageSizeTier,
			c.GetHeader("Content-Type"),
		)
		setOpsSelectedAccountDetails(c, account)
		setOpsEndpointContext(c, account.GetMappedModel(runtimeSelectionModel), service.RequestTypeSync)
		accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, currentAPIKey.GroupID, sessionHash, selection, false, &streamStarted, reqLog)
		if !acquired {
			return
		}
		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())

		var result *service.OpenAIForwardResult
		switch imageProtocolMode {
		case service.OpenAIImageProtocolModeCompat:
			result, err = h.gatewayService.ForwardCompatImages(
				c.Request.Context(),
				c,
				account,
				body,
				strings.TrimSpace(c.GetHeader("Content-Type")),
				action,
				reqModel,
			)
		default:
			switch action {
			case "edits":
				result, err = h.gatewayService.ForwardNativeImagesEdits(c.Request.Context(), c, account, body)
			default:
				result, err = h.gatewayService.ForwardNativeImagesGeneration(c.Request.Context(), c, account, body)
			}
		}
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			var requestErr *service.OpenAIImageRequestError
			if errors.As(err, &failoverErr) {
				if excludeSelectedGroup(excludedGroupIDs, currentAPIKey) {
					continue
				}
				h.submitFailedUsageRecordTask(
					"handler.openai_gateway.images",
					c,
					currentAPIKey,
					currentSubscription,
					account,
					reqModel,
					false,
					time.Since(requestStart),
					failoverErr,
					err,
				)
				h.handleFailoverExhausted(c, failoverErr, false)
				return
			}
			if errors.As(err, &requestErr) {
				h.errorResponseWithCode(c, requestErr.Status, requestErr.Type, requestErr.Code, requestErr.Message)
				return
			}
			if imageProtocolMode == service.OpenAIImageProtocolModeCompat && !c.Writer.Written() {
				errMessage := strings.TrimSpace(err.Error())
				if strings.HasPrefix(errMessage, "upstream request failed:") {
					h.errorResponse(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
				} else {
					h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", errMessage)
				}
				return
			}
			wroteFallback := h.ensureForwardErrorResponse(c, false)
			if !wroteFallback && !c.Writer.Written() {
				h.errorResponse(c, http.StatusBadGateway, "upstream_error", "Upstream request failed")
			}
			return
		}

		if result != nil && currentAPIKey.User != nil {
			userAgent := c.GetHeader("User-Agent")
			clientIP := ip.GetClientIP(c)
			requestPayloadHash := service.HashUsageRequestPayload(body)
			h.submitUsageRecordTask(func(ctx context.Context) {
				_ = h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
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
				})
			})
		}
		return
	}
}

func setOpenAIImageTraceMetadata(
	c *gin.Context,
	action string,
	protocolMode string,
	displayModel string,
	targetModel string,
	sizeTier string,
	contentType string,
) {
	if c == nil || c.Request == nil {
		return
	}
	ctx := service.EnsureRequestMetadata(c.Request.Context())
	service.SetImageRouteFamilyMetadata(ctx, service.PublicImageRouteFamily)
	if strings.TrimSpace(strings.ToLower(action)) == "edits" {
		service.SetImageActionMetadata(ctx, "edit")
		service.SetImageUpstreamEndpointMetadata(ctx, service.EndpointImagesEdits)
	} else {
		service.SetImageActionMetadata(ctx, "generate")
		service.SetImageUpstreamEndpointMetadata(ctx, service.EndpointImagesGen)
	}
	service.SetImageResolvedProviderMetadata(ctx, service.PlatformOpenAI)
	service.SetImageDisplayModelIDMetadata(ctx, displayModel)
	service.SetImageTargetModelIDMetadata(ctx, targetModel)
	service.SetImageRequestFormatMetadata(ctx, strings.TrimSpace(contentType))
	service.SetImageProtocolModeMetadata(ctx, protocolMode)
	if protocolMode == service.OpenAIImageProtocolModeCompat {
		service.SetImageRequestSurfaceMetadata(ctx, "images_bridge")
		service.SetImageTargetModelIDMetadata(ctx, service.OpenAICompatImageTargetModel)
		service.SetImageUpstreamEndpointMetadata(ctx, service.EndpointResponses)
	} else {
		service.SetImageRequestSurfaceMetadata(ctx, "images_api")
	}
	service.SetImageSizeTierMetadata(ctx, sizeTier)
	c.Request = c.Request.WithContext(ctx)
}
