package service

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	grokUpstreamUserAgent = "sub2api-grok/1.0"
	grokCLIVersion        = "0.2.93"
)

func grokEffectiveAPIBaseURL(account *Account) string {
	if account != nil && account.IsGrokOAuth() {
		return defaultGrokCLIBaseURL
	}
	if account != nil {
		return account.GetBaseURL()
	}
	return ""
}

func grokJoinVersionedEndpoint(baseURL string, endpoint string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	endpoint = strings.TrimSpace(endpoint)
	if baseURL == "" {
		return "", fmt.Errorf("grok base url is required")
	}
	if endpoint == "" {
		return "", fmt.Errorf("grok endpoint is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid grok base url")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid grok base url")
	}

	endpointPath := "/" + strings.TrimLeft(endpoint, "/")
	basePath := strings.TrimRight(parsed.EscapedPath(), "/")
	if basePath == "" {
		parsed.Path = endpointPath
		parsed.RawPath = ""
		return parsed.String(), nil
	}

	if strings.HasPrefix(endpointPath, "/v1/") || endpointPath == "/v1" {
		if basePath == "/v1" || strings.HasSuffix(basePath, "/v1") {
			endpointPath = strings.TrimPrefix(endpointPath, "/v1")
			if endpointPath == "" {
				endpointPath = "/"
			}
		}
	}
	parsed.Path = strings.TrimRight(basePath, "/") + endpointPath
	parsed.RawPath = ""
	return parsed.String(), nil
}

func applyGrokCLIHeaders(headers httpHeaderSetter) {
	if headers == nil {
		return
	}
	headers.Set("User-Agent", grokUpstreamUserAgent)
	headers.Set("X-Grok-Client-Version", grokCLIVersion)
}

type httpHeaderSetter interface {
	Set(key, value string)
}

type grokUpstreamRequestMetadata struct {
	RouteMode         string
	EffectiveHost     string
	EffectiveEndpoint string
}

func newGrokUpstreamRequestMetadata(account *Account, rawURL string, endpoint string) grokUpstreamRequestMetadata {
	meta := grokUpstreamRequestMetadata{
		RouteMode:         grokRouteModeForAccount(account),
		EffectiveEndpoint: strings.TrimSpace(endpoint),
	}
	if parsed, err := url.Parse(strings.TrimSpace(rawURL)); err == nil {
		meta.EffectiveHost = parsed.Host
		if path := strings.TrimSpace(parsed.EscapedPath()); path != "" {
			meta.EffectiveEndpoint = path
		}
	}
	return meta
}

func newGrokSSOReverseRequestMetadata(endpoint string) grokUpstreamRequestMetadata {
	meta := grokUpstreamRequestMetadata{
		RouteMode:         GrokRouteModeSSOReverse,
		EffectiveHost:     "grok.com",
		EffectiveEndpoint: strings.TrimSpace(endpoint),
	}
	if meta.EffectiveEndpoint == "" {
		meta.EffectiveEndpoint = "/"
	}
	if parsed, err := url.Parse(defaultGrokReverseBaseURL); err == nil && strings.TrimSpace(parsed.Host) != "" {
		meta.EffectiveHost = parsed.Host
	}
	return meta
}

func applyGrokForwardMetadata(result *GrokGatewayForwardResult, meta grokUpstreamRequestMetadata) *GrokGatewayForwardResult {
	if result == nil {
		return nil
	}
	if strings.TrimSpace(result.RouteMode) == "" {
		result.RouteMode = meta.RouteMode
	}
	if strings.TrimSpace(result.EffectiveHost) == "" {
		result.EffectiveHost = meta.EffectiveHost
	}
	if strings.TrimSpace(result.EffectiveEndpoint) == "" {
		result.EffectiveEndpoint = meta.EffectiveEndpoint
	}
	return result
}

func applyGrokSSOReverseForwardMetadata(result *GrokGatewayForwardResult, endpoint string) *GrokGatewayForwardResult {
	if result == nil {
		return nil
	}
	if strings.TrimSpace(result.Endpoint) == "" {
		result.Endpoint = strings.TrimSpace(endpoint)
	}
	return applyGrokForwardMetadata(result, newGrokSSOReverseRequestMetadata(endpoint))
}

func grokRouteModeForAccount(account *Account) string {
	if account == nil {
		return GrokRouteModeAPIKey
	}
	switch {
	case account.IsGrokSSO():
		return GrokRouteModeSSOReverse
	case account.IsGrokOAuth():
		return GrokRouteModeOAuthBuild
	default:
		return GrokRouteModeAPIKey
	}
}
