//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type settingAllRepoStub struct {
	values map[string]string
}

func (s *settingAllRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingAllRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	panic("unexpected GetValue call")
}

func (s *settingAllRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingAllRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *settingAllRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingAllRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for k, v := range s.values {
		out[k] = v
	}
	return out, nil
}

func (s *settingAllRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestSettingService_GetAllSettings_MasksTelegramToken(t *testing.T) {
	repo := &settingAllRepoStub{
		values: map[string]string{
			SettingKeyTelegramBotToken: "123456:ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			SettingKeyTelegramChatID:   "-100123456",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetAllSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, "-100123456", settings.TelegramChatID)
	require.True(t, settings.TelegramBotTokenConfigured)
	require.Equal(t, "123456...WXYZ", settings.TelegramBotTokenMasked)
}

func TestSettingService_UpdateSettings_PreservesTelegramTokenWhenEmpty(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		TelegramChatID: "-100123456",
	})
	require.NoError(t, err)
	require.Equal(t, "-100123456", repo.updates[SettingKeyTelegramChatID])
	_, exists := repo.updates[SettingKeyTelegramBotToken]
	require.False(t, exists)
}

func TestSettingService_UpdateSettings_StoresTelegramTokenWhenProvided(t *testing.T) {
	repo := &settingUpdateRepoStub{}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		TelegramChatID:   "-100123456",
		TelegramBotToken: " 123456:ABCDEF-token ",
	})
	require.NoError(t, err)
	require.Equal(t, "-100123456", repo.updates[SettingKeyTelegramChatID])
	require.Equal(t, "123456:ABCDEF-token", repo.updates[SettingKeyTelegramBotToken])
}
