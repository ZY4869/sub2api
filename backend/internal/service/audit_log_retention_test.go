package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type auditLogCleanupRepoStub struct {
	cutoff time.Time
}

func (r *auditLogCleanupRepoStub) CreateAuditLog(context.Context, *AuditLog) error {
	return nil
}

func (r *auditLogCleanupRepoStub) ListAuditLogs(context.Context, *AuditLogFilter) (*AuditLogList, error) {
	return &AuditLogList{}, nil
}

func (r *auditLogCleanupRepoStub) DeleteAuditLogsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.cutoff = cutoff
	return 3, nil
}

type auditRetentionProviderStub struct {
	days int
}

func (p auditRetentionProviderStub) GetAuditLogRetentionDays(context.Context) int {
	return p.days
}

func TestAuditLogCleanupUsesConfiguredRetentionProvider(t *testing.T) {
	repo := &auditLogCleanupRepoStub{}
	svc := NewAuditLogService(repo)
	svc.SetRetentionProvider(auditRetentionProviderStub{days: 7})

	deleted, cutoff, err := svc.CleanupExpired(context.Background())

	require.NoError(t, err)
	require.Equal(t, int64(3), deleted)
	require.WithinDuration(t, time.Now().UTC().AddDate(0, 0, -7), cutoff, 2*time.Second)
	require.Equal(t, cutoff, repo.cutoff)
}

func TestAuditLogRetentionNormalization(t *testing.T) {
	require.Equal(t, DefaultAuditLogRetentionDays, NormalizeAuditLogRetentionDays(0))
	require.Equal(t, 7, NormalizeAuditLogRetentionDays(7))
	require.Equal(t, MaxAuditLogRetentionDays, NormalizeAuditLogRetentionDays(MaxAuditLogRetentionDays+1))
}

type auditRetentionSettingRepoStub struct {
	values map[string]string
}

func (r *auditRetentionSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (r *auditRetentionSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	value, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (r *auditRetentionSettingRepoStub) Set(_ context.Context, key string, value string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}

func (r *auditRetentionSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (r *auditRetentionSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *auditRetentionSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	out := make(map[string]string, len(r.values))
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *auditRetentionSettingRepoStub) Delete(context.Context, string) error {
	return nil
}

func TestSettingServiceAuditLogRetentionDaysDefaultsAndNormalizes(t *testing.T) {
	ctx := context.Background()
	repo := &auditRetentionSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)

	require.Equal(t, DefaultAuditLogRetentionDays, svc.GetAuditLogRetentionDays(ctx))

	repo.values[SettingKeyAuditLogRetentionDays] = "bad"
	require.Equal(t, DefaultAuditLogRetentionDays, svc.GetAuditLogRetentionDays(ctx))

	repo.values[SettingKeyAuditLogRetentionDays] = "7"
	require.Equal(t, 7, svc.GetAuditLogRetentionDays(ctx))

	repo.values[SettingKeyAuditLogRetentionDays] = strconv.Itoa(MaxAuditLogRetentionDays + 10)
	require.Equal(t, MaxAuditLogRetentionDays, svc.GetAuditLogRetentionDays(ctx))
}

func TestSettingServiceUpdateSettingsWritesAuditLogRetentionDays(t *testing.T) {
	repo := &auditRetentionSettingRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, nil)

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		AuditLogRetentionDays: MaxAuditLogRetentionDays + 10,
	})

	require.NoError(t, err)
	require.Equal(t, strconv.Itoa(MaxAuditLogRetentionDays), repo.values[SettingKeyAuditLogRetentionDays])
}
