package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	UpstreamRedirectBlockedStatusCode = http.StatusBadGateway
	UpstreamRedirectBlockedCode       = "UPSTREAM_REDIRECT_NOT_ALLOWED"
	UpstreamRedirectBlockedMessage    = "Upstream redirect is not allowed"
	upstreamRedirectBlockedHeader     = "X-Sub2API-Upstream-Redirect-Blocked"
)

func UpstreamRedirectBlockedBody() []byte {
	payload := map[string]any{
		"error": map[string]string{
			"code":    UpstreamRedirectBlockedCode,
			"message": UpstreamRedirectBlockedMessage,
		},
		"message": UpstreamRedirectBlockedMessage,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return []byte(`{"error":{"code":"UPSTREAM_REDIRECT_NOT_ALLOWED","message":"Upstream redirect is not allowed"},"message":"Upstream redirect is not allowed"}`)
	}
	return body
}

func RewriteUpstreamRedirectBlockedResponse(resp *http.Response) *http.Response {
	if resp == nil {
		return nil
	}
	body := UpstreamRedirectBlockedBody()
	rewritten := *resp
	rewritten.StatusCode = UpstreamRedirectBlockedStatusCode
	rewritten.Status = http.StatusText(UpstreamRedirectBlockedStatusCode)
	rewritten.Body = ioNopCloserBytes(body)
	rewritten.ContentLength = int64(len(body))
	rewritten.Header = cloneRedirectBlockedHeader(resp.Header)
	rewritten.Header.Set("Content-Type", "application/json")
	rewritten.Header.Set(upstreamRedirectBlockedHeader, "true")
	return &rewritten
}

func IsUpstreamRedirectBlockedResponse(resp *http.Response, body []byte) bool {
	if resp != nil {
		if strings.EqualFold(strings.TrimSpace(resp.Header.Get(upstreamRedirectBlockedHeader)), "true") {
			return true
		}
	}
	code := strings.TrimSpace(ExtractUpstreamErrorCode(body))
	return code == UpstreamRedirectBlockedCode
}

func UpstreamRedirectBlockedApplicationError() error {
	return infraerrors.New(
		UpstreamRedirectBlockedStatusCode,
		UpstreamRedirectBlockedCode,
		UpstreamRedirectBlockedMessage,
	)
}

func cloneRedirectBlockedHeader(src http.Header) http.Header {
	if src == nil {
		return make(http.Header)
	}
	header := src.Clone()
	header.Del("Location")
	header.Del("Refresh")
	return header
}

func ioNopCloserBytes(body []byte) ioReadCloser {
	return ioReadCloser{Reader: bytes.NewReader(body)}
}

type ioReadCloser struct {
	*bytes.Reader
}

func (c ioReadCloser) Close() error {
	return nil
}
