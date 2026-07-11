package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"go.uber.org/zap"
)

func enforceCodexIdentityHeaders(ctx context.Context, headers http.Header, account *Account) {
	if headers == nil || !isChatGPTOpenAIOAuthAccount(account) {
		return
	}
	userAgent := strings.TrimSpace(headers.Get("user-agent"))
	if userAgent == "" {
		userAgent = codexCLIUserAgent
		headers.Set("user-agent", userAgent)
	}
	originator := openai.SanitizeCodexOriginator(headers.Get("originator"))
	if paired := openai.CodexOriginatorForUserAgent(userAgent); paired != "" {
		originator = paired
	}
	if originator == "" {
		originator = "codex_cli_rs"
	}
	headers.Set("originator", originator)

	version := strings.TrimSpace(headers.Get("version"))
	if version == "" || CompareVersions(version, codexCLIVersion) < 0 {
		headers.Set("version", codexCLIVersion)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	logger.FromContext(ctx).Debug(
		"openai codex identity headers normalized",
		zap.String("component", "service.openai_gateway"),
		zap.String("originator", originator),
		zap.String("version", headers.Get("version")),
	)
}
