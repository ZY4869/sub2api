package service

import (
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

func resolveGeminiForwardServiceTiers(requestBody []byte, responseHeaders http.Header, responseBody []byte) (*string, *string) {
	requested := extractGeminiRequestedServiceTierFromBody(requestBody)
	resolved := extractGeminiResolvedServiceTierFromResponse(responseBody, responseHeaders)
	if resolved == nil {
		resolved = requested
	}
	return requested, resolved
}

func extractGeminiResolvedServiceTierFromResponse(responseBody []byte, headers http.Header) *string {
	for _, path := range []string{"service_tier", "serviceTier"} {
		raw := gjson.GetBytes(responseBody, path)
		if !raw.Exists() {
			continue
		}
		normalized := normalizeGeminiRequestedServiceTier(raw.String())
		return &normalized
	}
	if headers == nil {
		return nil
	}
	for _, key := range []string{"x-gemini-service-tier"} {
		if value := strings.TrimSpace(headers.Get(key)); value != "" {
			normalized := normalizeGeminiRequestedServiceTier(value)
			return &normalized
		}
	}
	return nil
}
