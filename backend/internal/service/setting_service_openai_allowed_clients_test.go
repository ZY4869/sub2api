package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

type openAIAllowedClientsSettingRepoStub struct {
	values map[string]string
}

func (s *openAIAllowedClientsSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	if value, ok := s.values[key]; ok {
		return &Setting{Key: key, Value: value}, nil
	}
	return nil, ErrSettingNotFound
}

func (s *openAIAllowedClientsSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *openAIAllowedClientsSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	return nil
}

func (s *openAIAllowedClientsSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *openAIAllowedClientsSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *openAIAllowedClientsSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	result := make(map[string]string, len(s.values))
	for key, value := range s.values {
		result[key] = value
	}
	return result, nil
}

func (s *openAIAllowedClientsSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestSettingServiceOpenAIAllowedCodexClients_DefaultsAndRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := &openAIAllowedClientsSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{})

	require.False(t, svc.IsOpenAIClaudeCodeCodexPluginAllowed(ctx))
	require.Nil(t, svc.GetOpenAIAllowedCodexClients(ctx))

	err := svc.UpdateSettings(ctx, &SystemSettings{OpenAIAllowClaudeCodeCodexPlugin: true})
	require.NoError(t, err)
	require.Equal(t, "true", repo.values[SettingKeyOpenAIAllowClaudeCodeCodexPlugin])
	require.True(t, svc.IsOpenAIClaudeCodeCodexPluginAllowed(ctx))
	require.Equal(t, []string{openai.AllowedClientClaudeCode}, svc.GetOpenAIAllowedCodexClients(ctx))

	settings, err := svc.GetAllSettings(ctx)
	require.NoError(t, err)
	require.True(t, settings.OpenAIAllowClaudeCodeCodexPlugin)
}

func TestSettingServiceOpenAIAllowedCodexClients_ReadErrorFallsBackClosed(t *testing.T) {
	svc := NewSettingService(openAIAllowedClientsBrokenRepo{}, &config.Config{})

	require.False(t, svc.IsOpenAIClaudeCodeCodexPluginAllowed(context.Background()))
	require.Nil(t, svc.GetOpenAIAllowedCodexClients(context.Background()))
}

type openAIAllowedClientsBrokenRepo struct{ SettingRepository }

func (openAIAllowedClientsBrokenRepo) GetValue(ctx context.Context, key string) (string, error) {
	return "", errors.New("settings unavailable")
}
