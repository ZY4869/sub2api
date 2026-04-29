//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type multiGroupRoutingRepoStub struct {
	values map[string]string
}

func (s *multiGroupRoutingRepoStub) Get(context.Context, string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *multiGroupRoutingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (s *multiGroupRoutingRepoStub) Set(context.Context, string, string) error {
	panic("unexpected Set call")
}

func (s *multiGroupRoutingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *multiGroupRoutingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *multiGroupRoutingRepoStub) GetAll(context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *multiGroupRoutingRepoStub) Delete(context.Context, string) error {
	panic("unexpected Delete call")
}

func TestSettingService_IsMultiGroupRoutingEnabled_DefaultsTrue(t *testing.T) {
	svc := NewSettingService(&multiGroupRoutingRepoStub{values: map[string]string{}}, &config.Config{})

	require.True(t, svc.IsMultiGroupRoutingEnabled(context.Background()))
}

func TestSettingService_IsMultiGroupRoutingEnabled_RespectsExplicitFalse(t *testing.T) {
	svc := NewSettingService(&multiGroupRoutingRepoStub{
		values: map[string]string{SettingKeyMultiGroupRoutingEnabled: "false"},
	}, &config.Config{})

	require.False(t, svc.IsMultiGroupRoutingEnabled(context.Background()))
}

func TestSettingService_InitializeDefaultSettings_SetsMultiGroupRoutingEnabledTrue(t *testing.T) {
	repo := &multiGroupRoutingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{})

	require.NoError(t, svc.InitializeDefaultSettings(context.Background()))
	require.Equal(t, "true", repo.values[SettingKeyMultiGroupRoutingEnabled])
}
