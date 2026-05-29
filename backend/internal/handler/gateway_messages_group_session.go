package handler

import (
	"errors"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *GatewayHandler) runGatewayMessages(c *gin.Context, req *gatewayMessagesRequest) {
	for {
		route, retryGroup, ok := h.resolveGatewayMessagesRoute(c, req)
		if !ok {
			return
		}
		if retryGroup {
			continue
		}
		if route.platform == service.PlatformGemini {
			if h.forwardGatewayMessagesGemini(c, req, route) {
				return
			}
			continue
		}
		if h.forwardGatewayMessagesClaude(c, req, route) {
			return
		}
	}
}

func (h *GatewayHandler) resolveGatewayMessagesRoute(c *gin.Context, req *gatewayMessagesRequest) (*gatewayMessagesRoute, bool, bool) {
	currentAPIKey, currentSubscription, err := resolveSelectedGatewayAPIKey(
		c,
		h.settingService,
		h.gatewayService,
		h.billingCacheService,
		req.apiKey,
		req.subscription,
		req.bindingSelectionModel,
		req.allowedPlatforms,
		req.excludedGroupIDs,
	)
	if err != nil {
		req.reqLog.Info("gateway.group_selection_failed", zap.Error(err))
		status, code, message := groupSelectionErrorDetails(err)
		h.handleStreamingAwareError(c, status, code, message, req.streamStarted)
		return nil, false, false
	}

	currentPlatform := ""
	if currentAPIKey.Group != nil {
		currentPlatform = currentAPIKey.Group.Platform
	}
	if !gatewayMessagesPlatformSupported(currentPlatform) {
		if excludeSelectedGroup(req.excludedGroupIDs, currentAPIKey) {
			releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, currentAPIKey)
			return nil, true, true
		}
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
		h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "This endpoint does not support the selected platform", req.streamStarted)
		return nil, false, false
	}

	channelSelectionModel, channelState, err := bindGatewayChannelState(c, h.gatewayService, currentAPIKey.Group, req.bindingSelectionModel)
	if err != nil {
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
		if errors.Is(err, service.ErrChannelModelNotAllowed) {
			h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "Requested model is not allowed by the bound channel", req.streamStarted)
			return nil, false, false
		}
		if errors.Is(err, service.ErrModelHardRemoved) {
			h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "Requested model is no longer available", req.streamStarted)
			return nil, false, false
		}
		h.handleStreamingAwareError(c, http.StatusInternalServerError, "api_error", "Failed to resolve channel routing", req.streamStarted)
		return nil, false, false
	}

	runtimeSelectionModel := channelSelectionModel
	if req.publicCatalogEntry != nil {
		runtimeSelectionModel = req.reqModel
	}
	sessionKey, hasBoundSession := h.resolveGatewayMessagesSession(c, req, currentAPIKey, currentPlatform)
	return &gatewayMessagesRoute{
		apiKey:                currentAPIKey,
		subscription:          currentSubscription,
		platform:              currentPlatform,
		channelState:          channelState,
		runtimeSelectionModel: runtimeSelectionModel,
		sessionKey:            sessionKey,
		hasBoundSession:       hasBoundSession,
	}, false, true
}

func gatewayMessagesPlatformSupported(platform string) bool {
	switch platform {
	case service.PlatformGemini,
		service.PlatformAnthropic,
		service.PlatformDeepSeek,
		service.PlatformAntigravity,
		service.PlatformKiro,
		service.PlatformProtocolGateway:
		return true
	default:
		return false
	}
}

func (h *GatewayHandler) resolveGatewayMessagesSession(c *gin.Context, req *gatewayMessagesRequest, apiKey *service.APIKey, platform string) (string, bool) {
	sessionKey := req.selectedSessionHash
	if platform == service.PlatformGemini && req.selectedSessionHash != "" {
		sessionKey = "gemini:" + req.selectedSessionHash
	} else if platform == service.PlatformDeepSeek && req.selectedSessionHash != "" {
		sessionKey = "deepseek:" + req.selectedSessionHash
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
	return sessionKey, sessionKey != "" && sessionBoundAccountID > 0
}

func (h *GatewayHandler) rebindGatewayMessagesRouteChannel(c *gin.Context, req *gatewayMessagesRequest, route *gatewayMessagesRoute) bool {
	channelSelectionModel, channelState, err := bindGatewayChannelState(c, h.gatewayService, route.apiKey.Group, req.bindingSelectionModel)
	if err != nil {
		if errors.Is(err, service.ErrChannelModelNotAllowed) {
			releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
			h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "Requested model is not allowed by the bound channel", req.streamStarted)
			return false
		}
		if errors.Is(err, service.ErrModelHardRemoved) {
			releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
			h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", "Requested model is no longer available", req.streamStarted)
			return false
		}
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
		h.handleStreamingAwareError(c, http.StatusInternalServerError, "api_error", "Failed to resolve channel routing", req.streamStarted)
		return false
	}
	route.channelState = channelState
	route.runtimeSelectionModel = channelSelectionModel
	if req.publicCatalogEntry != nil {
		route.runtimeSelectionModel = req.reqModel
	}
	return true
}
