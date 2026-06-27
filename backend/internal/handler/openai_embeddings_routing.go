package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *OpenAIGatewayHandler) runEmbeddingsRoutingLoop(
	c *gin.Context,
	req *openAIEmbeddingsRequest,
	subscription *service.UserSubscription,
	streamStarted *bool,
) {
	routingStart := time.Now()
	sessionHash := h.gatewayService.GenerateSessionHash(c, req.body)
	excludedGroupIDs := make(map[int64]struct{})

	for {
		currentAPIKey, currentSubscription, err := resolveSelectedOpenAIAPIKey(
			c,
			h.settingService,
			h.gatewayService,
			h.billingCacheService,
			req.apiKey,
			subscription,
			req.publicRequestModel,
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
		channelState, runtimeSelectionModel, ok := h.resolveEmbeddingsChannelState(c, currentAPIKey, req)
		if !ok {
			return
		}
		if h.tryEmbeddingsAccounts(c, openAIEmbeddingsForwardInput{
			req:                   req,
			currentAPIKey:         currentAPIKey,
			currentSubscription:   currentSubscription,
			channelState:          channelState,
			runtimeSelectionModel: runtimeSelectionModel,
			sessionHash:           sessionHash,
			excludedGroupIDs:      excludedGroupIDs,
			routingStart:          routingStart,
			streamStarted:         streamStarted,
		}) {
			return
		}
	}
}

func (h *OpenAIGatewayHandler) resolveEmbeddingsChannelState(
	c *gin.Context,
	currentAPIKey *service.APIKey,
	req *openAIEmbeddingsRequest,
) (*service.GatewayChannelState, string, bool) {
	channelSelectionModel, channelState, err := bindGatewayChannelState(c, h.gatewayService, currentAPIKey.Group, req.publicRequestModel)
	if err == nil {
		if req.publicCatalogEntry != nil {
			return channelState, req.runtimeRequestModel, true
		}
		return channelState, channelSelectionModel, true
	}
	releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
	if errors.Is(err, service.ErrChannelModelNotAllowed) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Requested model is not allowed by the bound channel")
		return nil, "", false
	}
	if errors.Is(err, service.ErrModelHardRemoved) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Requested model is no longer available")
		return nil, "", false
	}
	h.errorResponse(c, http.StatusInternalServerError, "api_error", "Failed to resolve channel routing")
	return nil, "", false
}

func (h *OpenAIGatewayHandler) tryEmbeddingsAccounts(c *gin.Context, input openAIEmbeddingsForwardInput) bool {
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	var lastFailoverErr *service.UpstreamFailoverError
	var lastSelectionErr error

	for {
		selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
			c.Request.Context(),
			input.currentAPIKey.GroupID,
			"",
			input.sessionHash,
			input.runtimeSelectionModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportHTTPSSE,
			service.OpenAIEndpointCapabilityEmbeddings,
		)
		if err != nil || selection == nil || selection.Account == nil {
			lastSelectionErr = err
			return h.handleEmbeddingsSelectionExhausted(c, input, lastFailoverErr, lastSelectionErr)
		}
		result, account, forwardDuration, err := h.forwardEmbeddingsWithAccount(c, input, selection)
		if err != nil {
			if h.handleEmbeddingsForwardError(c, input, account, forwardDuration, err, failedAccountIDs, &lastFailoverErr, &switchCount) {
				continue
			}
			return true
		}
		h.recordEmbeddingsSuccess(c, input, account, result, switchCount)
		return true
	}
}

func (h *OpenAIGatewayHandler) handleEmbeddingsSelectionExhausted(
	c *gin.Context,
	input openAIEmbeddingsForwardInput,
	lastFailoverErr *service.UpstreamFailoverError,
	lastSelectionErr error,
) bool {
	if errors.Is(lastSelectionErr, service.ErrOpenAIModelNotFound) || h.gatewayService.IsModelUnavailableBecauseUnsupported(c.Request.Context(), input.currentAPIKey.GroupID, input.runtimeSelectionModel, nil) {
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, input.currentAPIKey)
		h.handleOpenAIModelNotFound(c, input.req.publicRequestModel, false)
		return true
	}
	if excludeSelectedGroup(input.excludedGroupIDs, input.currentAPIKey) {
		releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, input.currentAPIKey)
		return false
	}
	releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, input.currentAPIKey)
	if lastFailoverErr != nil {
		h.handleFailoverExhausted(c, lastFailoverErr, false)
	} else {
		h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", false)
	}
	return true
}
