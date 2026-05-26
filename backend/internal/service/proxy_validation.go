package service

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/hostexceptions"
)

const proxyInvalidHostCode = "PROXY_INVALID_HOST"

func ValidateProxyEndpointWithConfig(cfg *config.Config, protocol, host string, port int) error {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	host = strings.TrimSpace(host)
	if protocol != "http" && protocol != "https" && protocol != "socks5" && protocol != "socks5h" {
		return infraerrors.BadRequest(proxyInvalidHostCode, "unsupported proxy protocol")
	}
	if port <= 0 || port > 65535 {
		return infraerrors.BadRequest(proxyInvalidHostCode, "invalid proxy port")
	}
	if host == "" {
		return infraerrors.BadRequest(proxyInvalidHostCode, "proxy host is required")
	}
	if strings.Contains(host, "://") {
		return infraerrors.BadRequest(proxyInvalidHostCode, "proxy host must not be a URL")
	}
	if strings.ContainsAny(host, "/?#@") {
		return infraerrors.BadRequest(proxyInvalidHostCode, "proxy host contains invalid characters")
	}

	normalizedHost, err := normalizeProxyHost(host, port)
	if err != nil {
		return infraerrors.BadRequest(proxyInvalidHostCode, err.Error())
	}
	allowPrivate := cfg != nil && cfg.Security.URLAllowlist.AllowPrivateHosts
	if !allowPrivate {
		if err := hostexceptions.ValidateResolvedHost(context.Background(), cfg, hostexceptions.ScopeProxyHost, protocol, normalizedHost, port); err != nil {
			return infraerrors.BadRequest(proxyInvalidHostCode, "proxy host is not allowed by outbound security policy")
		}
	}
	return nil
}

func ValidateProxyURLForOutbound(proxyURL string, allowPrivateHosts bool) error {
	return ValidateProxyURLForOutboundWithConfig(proxyURL, nil, allowPrivateHosts)
}

func ValidateProxyURLForOutboundWithConfig(proxyURL string, cfg *config.Config, allowPrivateHosts bool) error {
	trimmed := strings.TrimSpace(proxyURL)
	if trimmed == "" {
		return nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Hostname() == "" {
		return infraerrors.BadRequest(proxyInvalidHostCode, "invalid proxy URL")
	}
	port := 0
	if parsed.Port() != "" {
		port, _ = strconv.Atoi(parsed.Port())
	} else {
		port = hostexceptions.DefaultPortForScheme(parsed.Scheme)
	}
	effectiveCfg := cfg
	if effectiveCfg == nil {
		effectiveCfg = &config.Config{}
	} else {
		copyCfg := *cfg
		copyURLAllowlist := cfg.Security.URLAllowlist
		copyCfg.Security.URLAllowlist = copyURLAllowlist
		effectiveCfg = &copyCfg
	}
	effectiveCfg.Security.URLAllowlist.AllowPrivateHosts = allowPrivateHosts
	return ValidateProxyEndpointWithConfig(effectiveCfg, parsed.Scheme, parsed.Hostname(), port)
}

func normalizeProxyHost(host string, port int) (string, error) {
	if parsed, err := url.Parse("//" + host); err == nil && parsed.Hostname() != "" {
		if parsed.Port() != "" {
			return "", fmt.Errorf("proxy port must be provided separately")
		}
		host = parsed.Hostname()
	}
	if strings.Contains(host, ":") && net.ParseIP(host) == nil {
		if _, err := net.ResolveIPAddr("ip", host); err != nil {
			return "", fmt.Errorf("invalid proxy host")
		}
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))
	splitHost, _, err := net.SplitHostPort(address)
	if err != nil || strings.TrimSpace(splitHost) == "" {
		return "", fmt.Errorf("invalid proxy host")
	}
	return strings.TrimSpace(splitHost), nil
}
