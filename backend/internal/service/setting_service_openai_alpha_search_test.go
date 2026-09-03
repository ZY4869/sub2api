package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type alphaSearchSettingRepoStub struct {
	value string
	err   error
}

func (s *alphaSearchSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (s *alphaSearchSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if key != SettingKeyOpenAIAlphaSearchEnabled {
		return "", ErrSettingNotFound
	}
	return s.value, s.err
}

func (s *alphaSearchSettingRepoStub) Set(context.Context, string, string) error { return nil }

func (s *alphaSearchSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, nil
}

func (s *alphaSearchSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}

func (s *alphaSearchSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return nil, nil
}

func (s *alphaSearchSettingRepoStub) Delete(context.Context, string) error { return nil }

func TestSettingServiceOpenAIAlphaSearchEnabled(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		repoErr     error
		wantEnabled bool
		wantErr     bool
	}{
		{name: "enabled", value: "true", wantEnabled: true},
		{name: "explicitly disabled", value: "false", wantEnabled: false},
		{name: "legacy missing key", repoErr: ErrSettingNotFound, wantEnabled: true},
		{name: "repository failure", repoErr: errors.New("database unavailable"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSettingService(&alphaSearchSettingRepoStub{value: tt.value, err: tt.repoErr}, &config.Config{})
			enabled, err := svc.IsOpenAIAlphaSearchEnabled(context.Background())
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantEnabled, enabled)
		})
	}
}
