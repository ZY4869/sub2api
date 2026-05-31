package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type geminiModelRuntime struct {
	publicModelName       string
	upstreamModelName     string
	bindingSelectionModel string
	publicCatalogEntry    *service.PublishedPublicCatalogEntry
}

func (h *GatewayHandler) prepareGeminiModelRoute(c *gin.Context, apiKey *service.APIKey) (modelName string, action string, ok bool) {
	modelName, action, err := parseGeminiModelAction(strings.TrimPrefix(c.Param("modelAction"), "/"))
	if err != nil {
		messageKey, fallback, mismatchKind := geminiModelActionRouteMismatchDetails(err)
		protocolruntime.RecordRouteMismatch(mismatchKind)
		slog.Warn(
			"gateway_route_mismatch",
			"runtime_platform", selectionPlatformForGeminiRoute(c, apiKey),
			"inbound_endpoint", GetInboundEndpoint(c),
			"reason", mismatchKind,
		)
		googleErrorWithReason(c, http.StatusNotFound, service.GatewayReasonRouteMismatch, messageKey, fallback)
		return "", "", false
	}
	return modelName, action, true
}

func (h *GatewayHandler) ensureGeminiProtocolCapability(c *gin.Context, selectionPlatform string, modelName string, action string) bool {
	decision := service.DecideProtocolCapability(selectionPlatform, GetInboundEndpoint(c), action)
	if decision.Supported {
		return true
	}
	slog.Warn(
		"gateway_unsupported_action",
		"runtime_platform", selectionPlatform,
		"inbound_endpoint", GetInboundEndpoint(c),
		"action", action,
		"model", modelName,
		"reason", decision.Reason,
	)
	switch decision.Reason {
	case service.GatewayReasonUnsupportedAction:
		protocolruntime.RecordUnsupportedAction(decision.Reason)
	default:
		protocolruntime.RecordRouteMismatch(decision.InternalMismatchKind)
	}
	googleErrorFromDecision(c, decision)
	return false
}

func (h *GatewayHandler) resolveGeminiModelRuntime(c *gin.Context, reqLog *zap.Logger, apiKey *service.APIKey, selectionPlatform string, modelName string) (geminiModelRuntime, bool) {
	runtime := geminiModelRuntime{
		publicModelName:   modelName,
		upstreamModelName: modelName,
	}
	if entry, status, resolveErr := h.gatewayService.ResolveAPIKeyPublishedPublicCatalogRuntimeStatus(c.Request.Context(), apiKey, selectionPlatform, modelName); resolveErr != nil {
		reqLog.Warn("gemini.public_catalog_entry_resolve_failed", zap.Error(resolveErr))
	} else if status == service.PublicCatalogResolutionNoMatch || status == service.PublicCatalogResolutionTimeWindowDenied {
		googlePublicCatalogUnavailableResponse(c, status)
		return geminiModelRuntime{}, false
	} else if status == service.PublicCatalogResolutionMatched {
		runtime.publicCatalogEntry = entry
		if sourceModel := strings.TrimSpace(entry.SourceModelID); sourceModel != "" {
			runtime.upstreamModelName = sourceModel
		}
		c.Request = c.Request.WithContext(service.AttachPublishedPublicCatalogEntry(c.Request.Context(), entry))
	}
	selectionModel := h.gatewayService.ResolveAPIKeySelectionModel(c.Request.Context(), apiKey, selectionPlatform, runtime.publicModelName)
	if selectionModel == "" {
		googlePublicCatalogUnavailableResponse(c, service.PublicCatalogResolutionNoMatch)
		return geminiModelRuntime{}, false
	}
	runtime.bindingSelectionModel = selectionModel
	if runtime.publicCatalogEntry != nil {
		runtime.bindingSelectionModel = runtime.publicModelName
	}
	return runtime, true
}

func parseGeminiModelAction(rest string) (model string, action string, err error) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", "", &pathParseError{"missing path"}
	}

	if i := strings.Index(rest, ":"); i > 0 && i < len(rest)-1 {
		return rest[:i], rest[i+1:], nil
	}
	if i := strings.Index(rest, "/"); i > 0 && i < len(rest)-1 {
		return rest[:i], rest[i+1:], nil
	}
	return "", "", &pathParseError{"invalid model action path"}
}

type pathParseError struct{ msg string }

func (e *pathParseError) Error() string { return e.msg }

func geminiModelActionRouteMismatchDetails(err error) (messageKey string, fallback string, mismatchKind string) {
	var parseErr *pathParseError
	if errors.As(err, &parseErr) {
		switch strings.TrimSpace(parseErr.msg) {
		case "missing path":
			return "gateway.gemini.model_action_path_missing", "Gemini model action path is missing", "missing_model_action_path"
		case "invalid model action path":
			return "gateway.gemini.model_action_path_invalid", "Gemini model action path is invalid", "invalid_model_action_path"
		}
	}
	return "gateway.gemini.model_action_path_invalid", "Gemini model action path is invalid", "invalid_model_action_path"
}

func selectionPlatformForGeminiRoute(c *gin.Context, apiKey *service.APIKey) string {
	if forcePlatform, ok := middleware.GetForcePlatformFromContext(c); ok && strings.TrimSpace(forcePlatform) != "" {
		return forcePlatform
	}
	if apiKey != nil && apiKey.Group != nil && strings.TrimSpace(apiKey.Group.Platform) != "" {
		return apiKey.Group.Platform
	}
	return service.PlatformGemini
}
