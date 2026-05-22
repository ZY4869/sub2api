package antigravity

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type captureTokenRoundTripper struct {
	t *testing.T
}

func (rt captureTokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.t.Helper()
	if req.Header.Get("User-Agent") != GetUserAgent() {
		rt.t.Fatalf("User-Agent = %q, want %q", req.Header.Get("User-Agent"), GetUserAgent())
	}
	if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		rt.t.Fatalf("Content-Type = %q", req.Header.Get("Content-Type"))
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"access_token":"access","expires_in":3600,"token_type":"Bearer"}`)),
		Request:    req,
	}, nil
}

func TestClientTokenRequestsUseConfiguredUserAgent(t *testing.T) {
	oldSecret := defaultClientSecret
	oldOverride := userAgentVersionOverride
	defaultClientSecret = "test-secret"
	SetUserAgentVersionOverride(func() string { return "9.8.7-test" })
	t.Cleanup(func() {
		defaultClientSecret = oldSecret
		userAgentVersionOverride = oldOverride
	})

	client := &Client{httpClient: &http.Client{Transport: captureTokenRoundTripper{t: t}}}

	if _, err := client.ExchangeCode(context.Background(), "code", "verifier"); err != nil {
		t.Fatalf("ExchangeCode returned error: %v", err)
	}
	if _, err := client.RefreshToken(context.Background(), "refresh-token"); err != nil {
		t.Fatalf("RefreshToken returned error: %v", err)
	}
}
