package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"go.uber.org/zap"
)

func (s *OpenAIGatewayService) codexOAuthUserAgentPolicy(ctx context.Context) CodexOAuthUserAgentPolicy {
	if s == nil || s.settingService == nil {
		return NormalizeCodexOAuthUserAgentPolicy("", "")
	}
	return s.settingService.GetCodexOAuthUserAgentPolicy(ctx)
}

func (s *OpenAIGatewayService) applyCodexOAuthUserAgentPolicy(ctx context.Context, headers http.Header, account *Account) {
	if headers == nil || !isChatGPTOpenAIOAuthAccount(account) {
		return
	}
	policy := s.codexOAuthUserAgentPolicy(ctx)
	before := strings.TrimSpace(headers.Get("user-agent"))
	next := before
	rewritten := false

	switch {
	case policy.Mode == CodexOAuthUAModeCustom && policy.Override != "":
		next = policy.Override
	case policy.Force:
		next = codexCLIUserAgent
	case !openai.IsCodexCLIRequest(before):
		next = codexCLIUserAgent
	}

	if next != "" && next != before {
		headers.Set("user-agent", next)
		rewritten = true
	}
	if rewritten || policy.Mode != CodexOAuthUAModeDefault {
		logger.FromContext(ctx).Info(
			"openai codex oauth user-agent policy applied",
			zap.String("component", "service.openai_gateway"),
			zap.String("mode", policy.Mode),
			zap.Bool("rewritten", rewritten),
			zap.Bool("custom_override", policy.Mode == CodexOAuthUAModeCustom && policy.Override != ""),
		)
	}
}
