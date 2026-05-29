package repository

import (
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/hostexceptions"
	"github.com/Wei-Shaw/sub2api/internal/pkg/proxyutil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

func buildUpstreamTransportWithTLSFingerprint(settings poolSettings, proxyURL *url.URL, profile *tlsfingerprint.Profile) (*http.Transport, error) {
	baseDialer := &net.Dialer{}
	targetDialer := newUpstreamGuardedDialer(settings, baseDialer, hostexceptions.ScopeUpstreamBaseURL, "")
	proxyDialer := newUpstreamGuardedDialer(settings, baseDialer, hostexceptions.ScopeProxyHost, proxyScheme(proxyURL))
	transport := &http.Transport{
		MaxIdleConns:          settings.maxIdleConns,
		MaxIdleConnsPerHost:   settings.maxIdleConnsPerHost,
		MaxConnsPerHost:       settings.maxConnsPerHost,
		IdleConnTimeout:       settings.idleConnTimeout,
		ResponseHeaderTimeout: settings.responseHeaderTimeout,
		ForceAttemptHTTP2:     false,
	}

	if proxyURL == nil {
		slog.Debug("tls_fingerprint_transport_direct")
		dialer := tlsfingerprint.NewDialer(profile, targetDialer.DialContext)
		transport.DialTLSContext = dialer.DialTLSContext
		return transport, nil
	}

	scheme := strings.ToLower(proxyURL.Scheme)
	switch scheme {
	case "socks5", "socks5h":
		slog.Debug("tls_fingerprint_transport_socks5", "proxy", proxyURL.Host)
		socks5Dialer := tlsfingerprint.NewSOCKS5ProxyDialerWithForwardDialer(profile, proxyURL, proxyDialer)
		transport.DialTLSContext = socks5Dialer.DialTLSContext
	case "http", "https":
		slog.Debug("tls_fingerprint_transport_http_connect", "proxy", proxyURL.Host)
		httpDialer := tlsfingerprint.NewHTTPProxyDialerWithBaseDialer(profile, proxyURL, proxyDialer.DialContext)
		transport.DialTLSContext = httpDialer.DialTLSContext
	default:
		slog.Debug("tls_fingerprint_transport_unknown_scheme_fallback", "scheme", scheme)
		if err := proxyutil.ConfigureTransportProxyWithDialer(transport, proxyURL, proxyDialer); err != nil {
			return nil, err
		}
	}
	return transport, nil
}
