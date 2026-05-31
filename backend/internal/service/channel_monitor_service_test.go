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
	enabled   []*ChannelMonitor
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

func (s *channelMonitorServiceRepoStub) ListEnabled(context.Context) ([]*ChannelMonitor, error) {
	return s.enabled, nil
}

type channelMonitorHistoryRepoStub struct {
	ChannelMonitorHistoryRepository
	latest []*ChannelMonitorHistory
}

func (s *channelMonitorHistoryRepoStub) ListLatestByMonitorIDs(context.Context, []int64) ([]*ChannelMonitorHistory, error) {
	return s.latest, nil
}

type channelMonitorRollupRepoStub struct {
	ChannelMonitorRollupRepository
	daily []*ChannelMonitorDailyRollup
}

func (s *channelMonitorRollupRepoStub) ListDailyByMonitorIDs(context.Context, []int64, time.Time) ([]*ChannelMonitorDailyRollup, error) {
	return s.daily, nil
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

func TestChannelMonitorService_PublicModelCatalogHealthAggregatesAndSanitizes(t *testing.T) {
	now := time.Now().UTC()
	repo := &channelMonitorServiceRepoStub{
		enabled: []*ChannelMonitor{
			{ID: 10, Enabled: true, PrimaryModelID: "gpt-5.4"},
			{ID: 11, Enabled: true, PrimaryModelID: "gpt-5.4"},
		},
	}
	historyRepo := &channelMonitorHistoryRepoStub{
		latest: []*ChannelMonitorHistory{
			{MonitorID: 10, ModelID: "gpt-5.4", Status: ChannelMonitorStatusSuccess, LatencyMs: 320, CreatedAt: now.Add(-time.Minute)},
			{MonitorID: 11, ModelID: "gpt-5.4-source", Status: ChannelMonitorStatusDegraded, LatencyMs: 640, CreatedAt: now},
		},
	}
	rollupRepo := &channelMonitorRollupRepoStub{
		daily: []*ChannelMonitorDailyRollup{
			{MonitorID: 10, ModelID: "gpt-5.4", Day: now, TotalChecks: 10, AvailableChecks: 10, TotalLatencyMs: 3200},
			{MonitorID: 11, ModelID: "gpt-5.4-source", Day: now, TotalChecks: 10, AvailableChecks: 8, DegradedChecks: 2, TotalLatencyMs: 5120},
		},
	}
	svc := NewChannelMonitorService(repo, historyRepo, rollupRepo, nil, nil, &config.Config{})

	statuses, err := svc.PublicModelCatalogHealth(context.Background(), []PublicModelCatalogItem{{
		Model:     "gpt-5.4-public",
		SourceIDs: []string{"gpt-5.4", "gpt-5.4-source"},
	}})

	require.NoError(t, err)
	status := statuses["gpt-5.4-public"]
	require.Equal(t, PublicModelHealthStatusWarning, status.Status)
	require.Equal(t, "gpt-5.4-public", status.Model)
	require.NotNil(t, status.SuccessRateToday)
	require.InDelta(t, 0.9, *status.SuccessRateToday, 0.0001)
	require.NotNil(t, status.LatencyMs)
	require.Equal(t, int64(640), *status.LatencyMs)
	require.NotEmpty(t, status.Daily)
	require.NotContains(t, mustModelCatalogJSON(t, status), "monitor_id")
	require.NotContains(t, mustModelCatalogJSON(t, status), "gpt-5.4-source")
}

func TestChannelMonitorService_PublicModelCatalogHealthPendingWithoutMonitorData(t *testing.T) {
	svc := NewChannelMonitorService(&channelMonitorServiceRepoStub{}, &channelMonitorHistoryRepoStub{}, &channelMonitorRollupRepoStub{}, nil, nil, &config.Config{})

	statuses, err := svc.PublicModelCatalogHealth(context.Background(), []PublicModelCatalogItem{{Model: "gpt-5.4"}})

	require.NoError(t, err)
	require.Equal(t, PublicModelHealthStatusPending, statuses["gpt-5.4"].Status)
	require.Empty(t, statuses["gpt-5.4"].Daily)
}

func TestChannelMonitorService_PublicModelCatalogHealthStaleHistoryHidesMetrics(t *testing.T) {
	stale := time.Now().UTC().Add(-(publicModelCatalogProbeHistoryTTL + time.Minute))
	repo := &channelMonitorServiceRepoStub{
		enabled: []*ChannelMonitor{{ID: 10, Enabled: true, PrimaryModelID: "gpt-5.4"}},
	}
	historyRepo := &channelMonitorHistoryRepoStub{
		latest: []*ChannelMonitorHistory{
			{MonitorID: 10, ModelID: "gpt-5.4", Status: ChannelMonitorStatusSuccess, LatencyMs: 320, CreatedAt: stale},
		},
	}
	rollupRepo := &channelMonitorRollupRepoStub{
		daily: []*ChannelMonitorDailyRollup{
			{MonitorID: 10, ModelID: "gpt-5.4", Day: stale, TotalChecks: 10, AvailableChecks: 10, TotalLatencyMs: 3200},
		},
	}
	svc := NewChannelMonitorService(repo, historyRepo, rollupRepo, nil, nil, &config.Config{})

	statuses, err := svc.PublicModelCatalogHealth(context.Background(), []PublicModelCatalogItem{{Model: "gpt-5.4"}})

	require.NoError(t, err)
	status := statuses["gpt-5.4"]
	require.Equal(t, PublicModelHealthStatusPending, status.Status)
	require.Equal(t, PublicModelHealthSourceNone, status.HealthSource)
	require.Equal(t, PublicModelHealthReasonStaleHistory, status.StatusReason)
	require.NotEmpty(t, status.LastCheckedAt)
	require.Nil(t, status.SuccessRateToday)
	require.Nil(t, status.SuccessRate7d)
	require.Nil(t, status.LatencyMs)
	require.Empty(t, status.Daily)
	require.Empty(t, status.Trend)
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
