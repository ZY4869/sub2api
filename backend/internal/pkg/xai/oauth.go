package xai

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

const (
	OAuthIssuer         = "https://auth.x.ai"
	DefaultAuthorizeURL = OAuthIssuer + "/oauth2/authorize"
	DefaultDeviceURL    = OAuthIssuer + "/oauth2/device/code"
	DefaultTokenURL     = OAuthIssuer + "/oauth2/token"
	DefaultUserInfoURL  = OAuthIssuer + "/oauth2/userinfo"
	DefaultBaseURL      = "https://api.x.ai/v1"
	DefaultCLIBaseURL   = "https://cli-chat-proxy.grok.com/v1"
	DefaultClientID     = "b1a00492-073a-47ea-816f-4c329264a828"
	DefaultScope        = "openid profile email offline_access grok-cli:access api:access"
	DefaultRedirectURI  = "http://127.0.0.1:56121/callback"

	EnvAllowUnsafeURLOverrides = "XAI_ALLOW_UNSAFE_URL_OVERRIDES"
)

var (
	oauthEndpointAllowedHosts = []string{"x.ai", "*.x.ai"}
	baseURLAllowedHosts       = []string{"api.x.ai", "cli-chat-proxy.grok.com"}
)

type RuntimeSanityCheck struct {
	Value     string `json:"value"`
	Valid     bool   `json:"valid"`
	Error     string `json:"error,omitempty"`
	IsDefault bool   `json:"is_default,omitempty"`
}

type RuntimeSanityReport struct {
	BaseURL           RuntimeSanityCheck `json:"base_url"`
	OAuthAuthorizeURL RuntimeSanityCheck `json:"oauth_authorize_url"`
	OAuthTokenURL     RuntimeSanityCheck `json:"oauth_token_url"`
	OAuthDeviceURL    RuntimeSanityCheck `json:"oauth_device_url"`
	OAuthUserInfoURL  RuntimeSanityCheck `json:"oauth_userinfo_url"`
	UnsafeOverrides   bool               `json:"unsafe_url_overrides"`
	PublicPaths       []string           `json:"public_paths"`
}

func ValidateOAuthEndpointURL(raw string) (string, error) {
	if AllowUnsafeURLOverrides() {
		return urlvalidator.ValidateURLFormat(raw, true)
	}
	return urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts:     oauthEndpointAllowedHosts,
		RequireAllowlist: true,
		AllowPrivate:     false,
	})
}

func ValidateTrustedBaseURL(raw string) (string, error) {
	if AllowUnsafeURLOverrides() {
		return urlvalidator.ValidateURLFormat(raw, true)
	}
	normalized, err := urlvalidator.ValidateHTTPSURL(raw, urlvalidator.ValidationOptions{
		AllowedHosts:     baseURLAllowedHosts,
		RequireAllowlist: true,
		AllowPrivate:     false,
	})
	if err != nil {
		return "", err
	}
	return normalizeKnownBaseURLPath(normalized)
}

func NormalizeKnownBaseURLPath(raw string) (string, error) {
	return normalizeKnownBaseURLPath(raw)
}

func normalizeKnownBaseURLPath(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid base URL")
	}
	if parsed.User != nil {
		return "", errors.New("base URL must not include userinfo")
	}
	if parsed.ForceQuery || parsed.RawQuery != "" {
		return "", errors.New("base URL must not include a query")
	}
	if parsed.Fragment != "" {
		return "", errors.New("base URL must not include a fragment")
	}
	path := strings.TrimRight(parsed.Path, "/")
	switch path {
	case "":
		parsed.Path = "/v1"
	case "/v1":
		parsed.Path = path
	default:
		return "", fmt.Errorf("base URL path must be /v1")
	}
	parsed.RawPath = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func BuildAuthorizationURL(authorizeURL, clientID, scope, redirectURI, state, codeChallenge, nonce string) (string, error) {
	authorizeURL = strings.TrimSpace(authorizeURL)
	if authorizeURL == "" {
		authorizeURL = DefaultAuthorizeURL
	}
	authorizeURL, err := ValidateOAuthEndpointURL(authorizeURL)
	if err != nil {
		return "", fmt.Errorf("invalid authorize url: %w", err)
	}
	clientID = firstNonEmpty(clientID, DefaultClientID)
	scope = firstNonEmpty(scope, DefaultScope)
	redirectURI = firstNonEmpty(redirectURI, DefaultRedirectURI)

	parsed, err := url.Parse(authorizeURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid authorize url")
	}
	params := parsed.Query()
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", scope)
	params.Set("state", state)
	if strings.TrimSpace(nonce) != "" {
		params.Set("nonce", strings.TrimSpace(nonce))
	}
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("plan", "generic")
	params.Set("referrer", "sub2api")
	parsed.RawQuery = params.Encode()
	return parsed.String(), nil
}

func GenerateNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func RuntimeSanity(authorizeURL, tokenURL, deviceURL, userInfoURL, baseURL string) RuntimeSanityReport {
	return RuntimeSanityReport{
		BaseURL:           runtimeSanityCheck(firstNonEmpty(baseURL, DefaultCLIBaseURL), ValidateTrustedBaseURL),
		OAuthAuthorizeURL: runtimeSanityCheck(firstNonEmpty(authorizeURL, DefaultAuthorizeURL), ValidateOAuthEndpointURL),
		OAuthTokenURL:     runtimeSanityCheck(firstNonEmpty(tokenURL, DefaultTokenURL), ValidateOAuthEndpointURL),
		OAuthDeviceURL:    runtimeSanityCheck(firstNonEmpty(deviceURL, DefaultDeviceURL), ValidateOAuthEndpointURL),
		OAuthUserInfoURL:  runtimeSanityCheck(firstNonEmpty(userInfoURL, DefaultUserInfoURL), ValidateOAuthEndpointURL),
		UnsafeOverrides:   AllowUnsafeURLOverrides(),
		PublicPaths:       []string{"/grok/v1/responses", "/v1/responses", "/grok/v1/chat/completions"},
	}
}

func runtimeSanityCheck(value string, validate func(string) (string, error)) RuntimeSanityCheck {
	normalized, err := validate(value)
	check := RuntimeSanityCheck{
		Value:     sanitizeRuntimeURLValue(normalized),
		Valid:     err == nil,
		IsDefault: strings.TrimSpace(value) == "",
	}
	if err != nil {
		check.Value = sanitizeRuntimeURLValue(value)
		check.Error = logredact.RedactText(err.Error())
	}
	return check
}

func sanitizeRuntimeURLValue(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return trimmed
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func AllowUnsafeURLOverrides() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvAllowUnsafeURLOverrides))) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
