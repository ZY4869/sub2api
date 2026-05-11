package service

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

const (
	defaultBaiduDocumentAIHost = "paddleocr.aistudio-app.com"

	documentAIUploadFallbackMaxBytes           = int64(50 * 1024 * 1024)
	documentAIUpstreamJSONFallbackMaxBytes     = int64(10 * 1024 * 1024)
	documentAIResultDownloadFallbackMaxBytes   = int64(100 * 1024 * 1024)
	documentAIReadLimitExceededMessage         = "document ai payload exceeds size limit"
	documentAIBase64DecodedSizeOverflowMessage = "document ai file_base64 exceeds size limit"
)

func documentAIAllowedHosts(cfg *config.Config) []string {
	if cfg != nil && len(cfg.Security.URLAllowlist.DocumentAIHosts) > 0 {
		return cfg.Security.URLAllowlist.DocumentAIHosts
	}
	return []string{defaultBaiduDocumentAIHost}
}

func validateDocumentAIURLWithConfig(cfg *config.Config, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("url is required")
	}
	if cfg == nil {
		return urlvalidator.ValidateHTTPURL(trimmed, false, urlvalidator.ValidationOptions{
			AllowedHosts:     []string{defaultBaiduDocumentAIHost},
			RequireAllowlist: true,
		})
	}
	if !cfg.Security.URLAllowlist.Enabled {
		return urlvalidator.ValidateURLFormat(trimmed, cfg.Security.URLAllowlist.AllowInsecureHTTP)
	}
	return urlvalidator.ValidateHTTPURL(trimmed, cfg.Security.URLAllowlist.AllowInsecureHTTP, urlvalidator.ValidationOptions{
		AllowedHosts:     documentAIAllowedHosts(cfg),
		RequireAllowlist: true,
		AllowPrivate:     cfg.Security.URLAllowlist.AllowPrivateHosts,
	})
}

func documentAIUploadMaxBytes(cfg *config.Config) int64 {
	if cfg != nil && cfg.Gateway.DocumentAIUploadMaxBytes > 0 {
		return cfg.Gateway.DocumentAIUploadMaxBytes
	}
	return documentAIUploadFallbackMaxBytes
}

func documentAIUpstreamJSONReadMaxBytes(cfg *config.Config) int64 {
	if cfg != nil && cfg.Gateway.DocumentAIUpstreamJSONReadMaxBytes > 0 {
		return cfg.Gateway.DocumentAIUpstreamJSONReadMaxBytes
	}
	return documentAIUpstreamJSONFallbackMaxBytes
}

func documentAIResultReadMaxBytes(cfg *config.Config) int64 {
	if cfg != nil && cfg.Gateway.DocumentAIResultReadMaxBytes > 0 {
		return cfg.Gateway.DocumentAIResultReadMaxBytes
	}
	return documentAIResultDownloadFallbackMaxBytes
}

func readAllLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	if maxBytes <= 0 {
		return nil, errors.New("read limit must be positive")
	}
	limited := io.LimitReader(r, maxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("%s: max=%d", documentAIReadLimitExceededMessage, maxBytes)
	}
	return body, nil
}

func decodedBase64SizeWithinLimit(raw string, maxBytes int64) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || maxBytes <= 0 {
		return true
	}
	decodedLen := base64.StdEncoding.DecodedLen(len(trimmed))
	return int64(decodedLen) <= maxBytes
}
