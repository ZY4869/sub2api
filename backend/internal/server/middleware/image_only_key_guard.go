package middleware

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func isImageOnlyAllowedGatewayRequest(method, path string) bool {
	method = strings.TrimSpace(strings.ToUpper(method))
	path = strings.TrimSpace(path)
	normalizedPath := strings.TrimRight(strings.ToLower(path), "/")
	if normalizedPath == "" {
		normalizedPath = "/"
	}

	// Allow usage query endpoint (no billing enforcement anyway).
	if normalizedPath == "/v1/usage" {
		return true
	}

	// Allow model listing endpoints so users can "pull" available image models.
	if method == http.MethodGet {
		if normalizedPath == "/v1/models" || strings.HasPrefix(normalizedPath, "/v1/models/") {
			return true
		}
		if normalizedPath == "/grok/v1/models" || strings.HasPrefix(normalizedPath, "/grok/v1/models/") {
			return true
		}
	}

	// Disallow Grok responses surface explicitly — image-only keys must use image endpoints.
	if strings.HasPrefix(normalizedPath, "/grok/v1/responses") {
		return false
	}

	inbound := service.NormalizeInboundEndpoint(path)
	switch inbound {
	case service.EndpointImagesGen, service.EndpointImagesEdits, service.EndpointResponses:
		return true
	default:
		return false
	}
}
