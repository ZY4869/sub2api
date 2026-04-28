package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type channelMonitorServiceRepoStub struct {
	ChannelMonitorRepository
	createErr error
	created   *ChannelMonitor
}

func (s *channelMonitorServiceRepoStub) Create(_ context.Context, monitor *ChannelMonitor) (*ChannelMonitor, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	clone := *monitor
	clone.ID = 42
	clone.CreatedAt = time.Now().UTC()
	clone.UpdatedAt = clone.CreatedAt
	s.created = &clone
	return &clone, nil
}

type channelMonitorEncryptorStub struct {
	encrypted string
}

func (s channelMonitorEncryptorStub) Encrypt(plaintext string) (string, error) {
	return s.encrypted + plaintext, nil
}

func (s channelMonitorEncryptorStub) Decrypt(ciphertext string) (string, error) {
	return ciphertext, nil
}

func TestChannelMonitorService_Create_DisabledAllowsEmptyAPIKey(t *testing.T) {
	repo := &channelMonitorServiceRepoStub{}
	svc := newChannelMonitorServiceForCreateTest(repo)

	created, err := svc.Create(context.Background(), validChannelMonitorForCreate(false), nil)
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Nil(t, created.APIKeyEncrypted)
	require.Equal(t, 60, created.IntervalSeconds)
	require.False(t, created.Enabled)
	require.Nil(t, created.NextRunAt)
	require.NotNil(t, repo.created)
}

func TestChannelMonitorService_Create_EnabledRequiresAPIKey(t *testing.T) {
	svc := newChannelMonitorServiceForCreateTest(&channelMonitorServiceRepoStub{})

	_, err := svc.Create(context.Background(), validChannelMonitorForCreate(true), nil)
	require.ErrorIs(t, err, ErrChannelMonitorAPIKeyRequired)

	blank := "   "
	_, err = svc.Create(context.Background(), validChannelMonitorForCreate(true), &blank)
	require.ErrorIs(t, err, ErrChannelMonitorAPIKeyRequired)
}

func TestChannelMonitorService_Create_EncryptsAPIKeyAndSchedulesEnabledMonitor(t *testing.T) {
	repo := &channelMonitorServiceRepoStub{}
	svc := newChannelMonitorServiceForCreateTest(repo)
	key := "sk-secret"

	created, err := svc.Create(context.Background(), validChannelMonitorForCreate(true), &key)
	require.NoError(t, err)
	require.NotNil(t, created.APIKeyEncrypted)
	require.Equal(t, "encrypted:sk-secret", *created.APIKeyEncrypted)
	require.NotNil(t, created.NextRunAt)
	require.NotContains(t, created.Endpoint, "sk-secret")
}

func TestChannelMonitorService_Create_ReturnsRepositoryError(t *testing.T) {
	svc := newChannelMonitorServiceForCreateTest(&channelMonitorServiceRepoStub{
		createErr: errors.New("db down"),
	})
	key := "sk-secret"

	_, err := svc.Create(context.Background(), validChannelMonitorForCreate(true), &key)
	require.ErrorContains(t, err, "db down")
}

func newChannelMonitorServiceForCreateTest(repo ChannelMonitorRepository) *ChannelMonitorService {
	settingRepo := &modelCatalogSettingRepoStub{values: map[string]string{
		SettingKeyChannelMonitorDefaultIntervalSeconds: "60",
	}}
	return NewChannelMonitorService(
		repo,
		nil,
		nil,
		NewSettingService(settingRepo, &config.Config{}),
		channelMonitorEncryptorStub{encrypted: "encrypted:"},
		&config.Config{},
	)
}

func validChannelMonitorForCreate(enabled bool) *ChannelMonitor {
	return &ChannelMonitor{
		Name:             "OpenAI health",
		Provider:         ChannelMonitorProviderOpenAI,
		Endpoint:         "https://api.openai.example/v1/responses",
		Enabled:          enabled,
		PrimaryModelID:   "gpt-5.4",
		BodyOverrideMode: ChannelMonitorBodyOverrideModeOff,
	}
}
