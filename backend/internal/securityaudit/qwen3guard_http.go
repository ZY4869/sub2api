package securityaudit

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func NormalizeBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", infraerrors.BadRequest("prompt_audit_invalid_base_url", "endpoint base_url is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", infraerrors.BadRequest("prompt_audit_invalid_base_url", "endpoint base_url must be absolute")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", infraerrors.BadRequest("prompt_audit_invalid_base_url", "endpoint base_url must use http or https")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", infraerrors.BadRequest("prompt_audit_unsafe_base_url", "endpoint base_url must not include credentials, query, or fragment")
	}
	if strings.TrimSpace(parsed.Hostname()) == "" {
		return "", infraerrors.BadRequest("prompt_audit_invalid_base_url", "endpoint base_url must include host")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if strings.EqualFold(parsed.Path, "/v1") {
		parsed.Path = ""
	}
	parsed.RawPath = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func ChatCompletionsURL(baseURL string) (string, error) {
	base, err := NormalizeBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	return base + "/v1/chat/completions", nil
}

func ModelsURL(baseURL string) (string, error) {
	base, err := NormalizeBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	return base + "/v1/models", nil
}

func NewSecureHTTPClient(endpoint ActiveEndpoint) (*http.Client, error) {
	if _, err := NormalizeBaseURL(endpoint.BaseURL); err != nil {
		return nil, err
	}
	timeout := time.Duration(endpoint.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = time.Duration(DefaultTimeoutMS) * time.Millisecond
	}
	return &http.Client{Transport: secureTransport(timeout), Timeout: timeout}, nil
}

func secureTransport(timeout time.Duration) *http.Transport {
	dialer := &net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}
	return &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          64,
		MaxIdleConnsPerHost:   16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	}
}

func networkGuardError(err error) *GuardError {
	timeout := errors.Is(err, context.DeadlineExceeded)
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		timeout = true
	}
	return &GuardError{Code: ErrorCodeUnavailable, Retryable: true, Timeout: timeout, Cause: err}
}
