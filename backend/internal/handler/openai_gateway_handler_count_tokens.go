package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// CountTokens handles Anthropic /v1/messages/count_tokens for OpenAI-compatible groups.
func (h *OpenAIGatewayHandler) CountTokens(c *gin.Context) {
	requestStart := time.Now()
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.anthropicErrorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	if apiKey.Group != nil {
		applyOpenAIPlatformContext(c, apiKey.Group.Platform)
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.anthropicErrorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.count_tokens",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.anthropicErrorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}
	if !gjson.ValidBytes(body) {
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	reqModel := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if reqModel == "" {
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	reqLog = reqLog.With(zap.String("model", reqModel))
	setOpsRequestContext(c, reqModel, false, body)
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())

	publicRequestModel := service.NormalizeOpenAICompatRequestedModel(reqModel)
	runtimeSelectionModel := publicRequestModel
	var publicCatalogEntry *service.PublishedPublicCatalogEntry
	if entry, status, resolveErr := h.gatewayService.ResolveAPIKeyPublishedPublicCatalogRuntimeStatus(c.Request.Context(), apiKey, service.OpenAIPlatformFromContext(c.Request.Context()), reqModel); resolveErr != nil {
		reqLog.Warn("openai_count_tokens.public_catalog_entry_resolve_failed", zap.Error(resolveErr))
	} else if status == service.PublicCatalogResolutionNoMatch || status == service.PublicCatalogResolutionTimeWindowDenied {
		h.anthropicPublicCatalogUnavailableResponse(c, status)
		return
	} else if status == service.PublicCatalogResolutionMatched {
		publicCatalogEntry = entry
		runtimeSelectionModel = service.NormalizeOpenAICompatRequestedModel(firstNonEmptyHandlerString(entry.SourceModelID, runtimeSelectionModel))
		c.Request = c.Request.WithContext(service.AttachPublishedPublicCatalogEntry(c.Request.Context(), entry))
	}

	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	currentAPIKey, _, err := resolveSelectedOpenAIAPIKey(
		c,
		h.settingService,
		h.gatewayService,
		h.billingCacheService,
		apiKey,
		subscription,
		publicRequestModel,
		openAICompatiblePlatforms,
		nil,
	)
	if err != nil {
		reqLog.Info("openai_count_tokens.group_selection_failed", zap.Error(err))
		status, code, message := groupSelectionErrorDetails(err)
		h.anthropicErrorResponse(c, status, code, message)
		return
	}
	defer releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
	if currentAPIKey.Group != nil {
		applyOpenAIPlatformContext(c, currentAPIKey.Group.Platform)
		if !currentAPIKey.Group.AllowMessagesDispatch {
			h.anthropicErrorResponse(c, http.StatusForbidden, "permission_error", "This group does not allow /v1/messages dispatch")
			return
		}
	}
	channelSelectionModel, channelState, err := bindGatewayChannelState(c, h.gatewayService, currentAPIKey.Group, publicRequestModel)
	if err != nil {
		if errors.Is(err, service.ErrChannelModelNotAllowed) {
			h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Requested model is not allowed by the bound channel")
			return
		}
		if errors.Is(err, service.ErrModelHardRemoved) {
			h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Requested model is no longer available")
			return
		}
		h.anthropicErrorResponse(c, http.StatusInternalServerError, "api_error", "Failed to resolve channel routing")
		return
	}
	if publicCatalogEntry == nil {
		runtimeSelectionModel = channelSelectionModel
	}
	routingStart := time.Now()
	selection, _, err := h.gatewayService.SelectAccountWithScheduler(
		c.Request.Context(),
		currentAPIKey.GroupID,
		"",
		"",
		runtimeSelectionModel,
		nil,
		service.OpenAIUpstreamTransportAny,
	)
	if err != nil {
		reqLog.Warn("openai_count_tokens.account_select_failed", zap.Error(err))
		if errors.Is(err, service.ErrOpenAIModelNotFound) {
			h.anthropicErrorResponse(c, http.StatusNotFound, "invalid_request_error", "The requested model does not exist or is not available")
			return
		}
		h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable")
		return
	}
	if selection == nil || selection.Account == nil {
		h.anthropicErrorResponse(c, http.StatusServiceUnavailable, "api_error", "No available accounts")
		return
	}
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
	account := selection.Account
	ctx := reattachGatewayChannelState(c.Request.Context(), channelState)
	c.Request = c.Request.WithContext(ctx)
	setOpsSelectedAccountDetails(c, account)
	setOpsEndpointContext(c, account.GetMappedModel(runtimeSelectionModel), service.RequestTypeSync)
	service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
	_, err = h.gatewayService.ForwardAnthropicCountTokensCompat(c.Request.Context(), c, account, body, "")
	if err != nil {
		reqLog.Warn("openai_count_tokens.forward_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		return
	}
	reqLog.Debug("openai_count_tokens.request_completed", zap.Int64("account_id", account.ID))
}
