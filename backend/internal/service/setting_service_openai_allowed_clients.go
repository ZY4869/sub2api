package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

func (s *SettingService) IsOpenAIClaudeCodeCodexPluginAllowed(ctx context.Context) bool {
	if s == nil || s.settingRepo == nil {
		return false
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyOpenAIAllowClaudeCodeCodexPlugin)
	if err != nil {
		return false
	}
	return value == "true"
}

func (s *SettingService) GetOpenAIAllowedCodexClients(ctx context.Context) []string {
	if !s.IsOpenAIClaudeCodeCodexPluginAllowed(ctx) {
		return nil
	}
	return []string{openai.AllowedClientClaudeCode}
}
