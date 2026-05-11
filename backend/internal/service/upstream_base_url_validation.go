package service

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
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
	if !cfg.Security.URLAllowlist.Enabled {
		normalized, err := urlvalidator.ValidateURLFormat(trimmed, cfg.Security.URLAllowlist.AllowInsecureHTTP)
		if err != nil {
			return "", fmt.Errorf("invalid base_url: %w", err)
		}
		return normalized, nil
	}
	normalized, err := urlvalidator.ValidateHTTPURL(trimmed, cfg.Security.URLAllowlist.AllowInsecureHTTP, urlvalidator.ValidationOptions{
		AllowedHosts:     cfg.Security.URLAllowlist.UpstreamHosts,
		RequireAllowlist: true,
		AllowPrivate:     cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	return normalized, nil
}
