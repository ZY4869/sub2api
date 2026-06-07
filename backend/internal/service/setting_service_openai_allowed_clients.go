package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

func NormalizeOpenAIAllowedCodexClients(values []string, legacyAllowClaudeCode bool) []string {
	normalized := openai.NormalizeAllowedClientIDs(values)
	if len(normalized) > 0 {
		return normalized
	}
	if legacyAllowClaudeCode {
		return []string{openai.AllowedClientClaudeCode}
	}
	return nil
}

func ParseOpenAIAllowedCodexClients(raw string, legacyAllowClaudeCode bool) []string {
	rawValues := parseJSONSettingStringSlice(raw)
	return NormalizeOpenAIAllowedCodexClients(rawValues, legacyAllowClaudeCode)
}

func MarshalOpenAIAllowedCodexClients(values []string) (string, error) {
	normalized := openai.NormalizeAllowedClientIDs(values)
	if normalized == nil {
		normalized = []string{}
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("marshal openai allowed codex clients: %w", err)
	}
	return string(data), nil
}

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
	if s == nil || s.settingRepo == nil {
		return nil
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyOpenAIAllowClaudeCodeCodexPlugin)
	if err != nil {
		return nil
	}
	if value != "true" {
		return nil
	}
	rawClients, err := s.settingRepo.GetValue(ctx, SettingKeyOpenAIAllowedCodexClients)
	if err != nil {
		rawClients = ""
	}
	return ParseOpenAIAllowedCodexClients(rawClients, true)
}
