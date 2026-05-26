package service

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/hostexceptions"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

const accountInvalidBaseURLCode = "ACCOUNT_INVALID_BASE_URL"

func validateAccountUpstreamBaseURL(cfg *config.Config, raw string) (string, error) {
	normalized, err := validateUpstreamBaseURLWithConfig(cfg, raw)
	if err != nil {
		return "", infraerrors.BadRequest(accountInvalidBaseURLCode, err.Error())
	}
	return normalized, nil
}

func validateUpstreamBaseURLWithConfig(cfg *config.Config, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("base_url is required")
	}
	if cfg == nil {
		normalized, err := urlvalidator.ValidateURLFormat(trimmed, false)
		if err != nil {
			return "", fmt.Errorf("invalid base_url: %w", err)
		}
		return normalized, nil
	}
	allowPrivate, privateExceptionMatched := upstreamBaseURLPrivatePolicy(cfg, trimmed)
	if !cfg.Security.URLAllowlist.Enabled {
		normalized, err := urlvalidator.ValidateHTTPURL(trimmed, cfg.Security.URLAllowlist.AllowInsecureHTTP, urlvalidator.ValidationOptions{
			AllowPrivate: allowPrivate,
		})
		if err != nil {
			return "", fmt.Errorf("invalid base_url: %w", err)
		}
		return normalized, nil
	}
	if privateExceptionMatched {
		normalized, err := urlvalidator.ValidateHTTPURL(trimmed, cfg.Security.URLAllowlist.AllowInsecureHTTP, urlvalidator.ValidationOptions{
			AllowPrivate: true,
		})
		if err != nil {
			return "", fmt.Errorf("invalid base_url: %w", err)
		}
		return normalized, nil
	}
	normalized, err := urlvalidator.ValidateHTTPURL(trimmed, cfg.Security.URLAllowlist.AllowInsecureHTTP, urlvalidator.ValidationOptions{
		AllowedHosts:     cfg.Security.URLAllowlist.UpstreamHosts,
		RequireAllowlist: true,
		AllowPrivate:     allowPrivate,
	})
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	return normalized, nil
}

func upstreamBaseURLPrivatePolicy(cfg *config.Config, raw string) (bool, bool) {
	if cfg == nil || cfg.Security.URLAllowlist.AllowPrivateHosts {
		return cfg != nil && cfg.Security.URLAllowlist.AllowPrivateHosts, false
	}
	if len(cfg.Security.URLAllowlist.PrivateHostExceptions) == 0 {
		return false, false
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false, false
	}
	port := hostexceptions.PortForURL(parsed)
	match, ok, err := hostexceptions.ValidateResolvedHostException(context.Background(), cfg, hostexceptions.ScopeUpstreamBaseURL, parsed.Scheme, parsed.Hostname(), port)
	if err == nil && ok {
		hostexceptions.LogMatch(context.Background(), match)
		return true, true
	}
	return false, false
}
