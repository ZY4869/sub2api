package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) forwardGeminiStrictV1BetaPassthrough(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	input, ok := resolveGeminiStrictV1BetaPassthroughInput(c)
	if !ok {
		return
	}
	h.forwardGeminiPassthrough(c, input)
}

func resolveGeminiStrictV1BetaPassthroughInput(c *gin.Context) (service.GeminiPublicPassthroughInput, bool) {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return service.GeminiPublicPassthroughInput{}, false
	}

	path := strings.TrimSpace(c.Request.URL.Path)
	method := strings.ToUpper(strings.TrimSpace(c.Request.Method))
	segments := splitGeminiStrictPublicPath(path)
	if len(segments) < 2 || !strings.EqualFold(segments[0], "v1beta") {
		rejectGeminiStrictV1BetaUnsupported(c)
		return service.GeminiPublicPassthroughInput{}, false
	}

	var (
		input service.GeminiPublicPassthroughInput
		ok    bool
	)
	switch strings.ToLower(strings.TrimSpace(segments[1])) {
	case "corpora":
		input, ok = resolveGeminiStrictCorporaPassthroughInput(method, segments)
	case "dynamic":
		input, ok = resolveGeminiStrictDynamicPassthroughInput(method, segments)
	case "generatedfiles":
		input, ok = resolveGeminiStrictGeneratedFilesPassthroughInput(method, segments)
	case "models":
		input, ok = resolveGeminiStrictModelOperationsPassthroughInput(method, segments)
	case "tunedmodels":
		input, ok = resolveGeminiStrictTunedModelsPassthroughInput(method, segments)
	}
	if !ok {
		rejectGeminiStrictV1BetaUnsupported(c)
		return service.GeminiPublicPassthroughInput{}, false
	}
	return input, true
}

func resolveGeminiStrictCorporaPassthroughInput(method string, segments []string) (service.GeminiPublicPassthroughInput, bool) {
	switch len(segments) {
	case 2:
		if method == http.MethodGet || method == http.MethodPost {
			return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiCorpus}, true
		}
	case 3:
		if method == http.MethodGet || method == http.MethodDelete {
			return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiCorpus}, true
		}
	case 4:
		switch strings.ToLower(strings.TrimSpace(segments[3])) {
		case "permissions":
			if method == http.MethodGet || method == http.MethodPost {
				return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiCorpusPermission}, true
			}
		}
	case 5:
		switch strings.ToLower(strings.TrimSpace(segments[3])) {
		case "operations":
			if method == http.MethodGet {
				return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiCorpusOperation}, true
			}
		case "permissions":
			if method == http.MethodGet || method == http.MethodPatch || method == http.MethodDelete {
				return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiCorpusPermission}, true
			}
		}
	}
	return service.GeminiPublicPassthroughInput{}, false
}

func resolveGeminiStrictFileSearchPassthroughInput(c *gin.Context) (service.GeminiPublicPassthroughInput, bool) {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return service.GeminiPublicPassthroughInput{}, false
	}

	method := strings.ToUpper(strings.TrimSpace(c.Request.Method))
	segments := splitGeminiStrictPublicPath(c.Request.URL.Path)
	if len(segments) == 0 {
		return service.GeminiPublicPassthroughInput{}, false
	}

	if len(segments) == 4 &&
		strings.EqualFold(segments[0], "upload") &&
		strings.EqualFold(segments[1], "v1beta") &&
		strings.EqualFold(segments[2], "fileSearchStores") &&
		method == http.MethodPost {
		_, action, ok := parseGeminiStrictPublicActionSegment(segments[3])
		if ok && strings.EqualFold(action, "uploadToFileSearchStore") {
			return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiUploadOperation}, true
		}
		return service.GeminiPublicPassthroughInput{}, false
	}

	if len(segments) < 2 || !strings.EqualFold(segments[0], "v1beta") || !strings.EqualFold(segments[1], "fileSearchStores") {
		return service.GeminiPublicPassthroughInput{}, false
	}

	switch len(segments) {
	case 2:
		if method == http.MethodGet || method == http.MethodPost {
			return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiFileSearchStore}, true
		}
	case 3:
		if _, action, ok := parseGeminiStrictPublicActionSegment(segments[2]); ok {
			switch {
			case method == http.MethodPost && strings.EqualFold(action, service.ProtocolCapabilityActionImportFile):
				return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiFileSearchStore}, true
			case method == http.MethodPost && strings.EqualFold(action, "uploadToFileSearchStore"):
				return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiUploadOperation}, true
			default:
				return service.GeminiPublicPassthroughInput{}, false
			}
		}
		if method == http.MethodGet || method == http.MethodDelete {
			return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiFileSearchStore}, true
		}
	case 4:
		switch strings.ToLower(strings.TrimSpace(segments[3])) {
		case "documents":
			if method == http.MethodGet {
				return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiDocument}, true
			}
		}
	case 5:
		switch strings.ToLower(strings.TrimSpace(segments[3])) {
		case "documents":
			if method == http.MethodGet || method == http.MethodDelete {
				return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiDocument}, true
			}
		case "operations":
			if method == http.MethodGet {
				return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiOperation}, true
			}
		}
	case 6:
		if strings.EqualFold(segments[3], "upload") && strings.EqualFold(segments[4], "operations") && method == http.MethodGet {
			return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiUploadOperation}, true
		}
	}

	return service.GeminiPublicPassthroughInput{}, false
}

func resolveGeminiStrictDynamicPassthroughInput(method string, segments []string) (service.GeminiPublicPassthroughInput, bool) {
	if len(segments) != 3 || method != http.MethodPost {
		return service.GeminiPublicPassthroughInput{}, false
	}
	dynamicName, action, ok := parseGeminiStrictPublicActionSegment(segments[2])
	if !ok {
		return service.GeminiPublicPassthroughInput{}, false
	}
	switch action {
	case service.ProtocolCapabilityActionGenerateContent, service.ProtocolCapabilityActionStreamGenerateContent:
		return service.GeminiPublicPassthroughInput{
			RequestedModel: strings.TrimSpace(dynamicName),
			ResourceKind:   service.UpstreamResourceKindGeminiDynamic,
		}, true
	default:
		return service.GeminiPublicPassthroughInput{}, false
	}
}

func resolveGeminiStrictGeneratedFilesPassthroughInput(method string, segments []string) (service.GeminiPublicPassthroughInput, bool) {
	switch len(segments) {
	case 2:
		if method == http.MethodGet {
			return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiGeneratedFile}, true
		}
	case 5:
		if method == http.MethodGet && strings.EqualFold(segments[3], "operations") {
			return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiGeneratedFileOperation}, true
		}
	}
	return service.GeminiPublicPassthroughInput{}, false
}

func resolveGeminiStrictModelOperationsPassthroughInput(method string, segments []string) (service.GeminiPublicPassthroughInput, bool) {
	if method != http.MethodGet || len(segments) < 4 || !strings.EqualFold(segments[3], "operations") {
		return service.GeminiPublicPassthroughInput{}, false
	}
	if len(segments) == 4 || len(segments) == 5 {
		return service.GeminiPublicPassthroughInput{
			RequestedModel: strings.TrimSpace(segments[2]),
			ResourceKind:   service.UpstreamResourceKindGeminiModelOperation,
		}, true
	}
	return service.GeminiPublicPassthroughInput{}, false
}

func resolveGeminiStrictTunedModelsPassthroughInput(method string, segments []string) (service.GeminiPublicPassthroughInput, bool) {
	switch len(segments) {
	case 2:
		if method == http.MethodGet || method == http.MethodPost {
			return service.GeminiPublicPassthroughInput{ResourceKind: service.UpstreamResourceKindGeminiTunedModel}, true
		}
	case 3:
		if tunedModel, action, ok := parseGeminiStrictPublicActionSegment(segments[2]); ok {
			if method != http.MethodPost {
				return service.GeminiPublicPassthroughInput{}, false
			}
			input := service.GeminiPublicPassthroughInput{
				RequestedModel: strings.TrimSpace(tunedModel),
				ResourceKind:   service.UpstreamResourceKindGeminiTunedModel,
			}
			switch action {
			case service.ProtocolCapabilityActionGenerateContent, service.ProtocolCapabilityActionStreamGenerateContent, service.ProtocolCapabilityActionTransferOwnership:
				return input, true
			case service.ProtocolCapabilityActionBatchGenerateContent, service.ProtocolCapabilityActionGeminiAsyncEmbedding:
				input.ResourceKind = service.UpstreamResourceKindGeminiTunedModelOperation
				return input, true
			default:
				return service.GeminiPublicPassthroughInput{}, false
			}
		}
		if method == http.MethodGet || method == http.MethodPatch || method == http.MethodDelete {
			return service.GeminiPublicPassthroughInput{
				RequestedModel: strings.TrimSpace(segments[2]),
				ResourceKind:   service.UpstreamResourceKindGeminiTunedModel,
			}, true
		}
	case 4:
		input := service.GeminiPublicPassthroughInput{RequestedModel: strings.TrimSpace(segments[2])}
		switch strings.ToLower(strings.TrimSpace(segments[3])) {
		case "permissions":
			if method == http.MethodGet || method == http.MethodPost {
				input.ResourceKind = service.UpstreamResourceKindGeminiTunedModelPermission
				return input, true
			}
		case "operations":
			if method == http.MethodGet {
				input.ResourceKind = service.UpstreamResourceKindGeminiTunedModelOperation
				return input, true
			}
		}
	case 5:
		input := service.GeminiPublicPassthroughInput{RequestedModel: strings.TrimSpace(segments[2])}
		switch strings.ToLower(strings.TrimSpace(segments[3])) {
		case "permissions":
			if method == http.MethodGet || method == http.MethodPatch || method == http.MethodDelete {
				input.ResourceKind = service.UpstreamResourceKindGeminiTunedModelPermission
				return input, true
			}
		case "operations":
			if method == http.MethodGet {
				input.ResourceKind = service.UpstreamResourceKindGeminiTunedModelOperation
				return input, true
			}
		}
	}
	return service.GeminiPublicPassthroughInput{}, false
}

func splitGeminiStrictPublicPath(path string) []string {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func parseGeminiStrictPublicActionSegment(segment string) (string, string, bool) {
	name, action, ok := strings.Cut(strings.TrimSpace(segment), ":")
	if !ok {
		return "", "", false
	}
	name = strings.TrimSpace(name)
	action = strings.TrimSpace(action)
	if name == "" || action == "" {
		return "", "", false
	}
	return name, action, true
}

func rejectGeminiStrictV1BetaUnsupported(c *gin.Context) {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return
	}
	applyGeminiPublicPathMetadata(c, "")
	protocolruntime.RecordUnsupportedAction(service.GatewayReasonUnsupportedAction)
	slog.Warn(
		"gateway_unsupported_action",
		"runtime_platform", service.PlatformGemini,
		"inbound_endpoint", GetInboundEndpoint(c),
		"method", strings.TrimSpace(c.Request.Method),
		"path", strings.TrimSpace(c.Request.URL.Path),
		"reason", "unsupported_action",
	)
	googleErrorWithReason(
		c,
		http.StatusBadRequest,
		service.GatewayReasonUnsupportedAction,
		"gateway.public_endpoint.unsupported_action",
		"%s does not support this action on the current route",
		firstNonEmptyHandlerString(strings.TrimSpace(c.Request.URL.Path), GetInboundEndpoint(c)),
	)
}
