package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type adminComplianceSettingRepo struct {
	values map[string]string
}

func (r *adminComplianceSettingRepo) Get(_ context.Context, key string) (*Setting, error) {
	value, err := r.GetValue(context.Background(), key)
	if err != nil {
		return nil, err
	}
	return &Setting{Key: key, Value: value}, nil
}

func (r *adminComplianceSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	if r.values == nil {
		return "", ErrSettingNotFound
	}
	value, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}

func (r *adminComplianceSettingRepo) Set(_ context.Context, key, value string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}

func (r *adminComplianceSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (r *adminComplianceSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *adminComplianceSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	out := map[string]string{}
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *adminComplianceSettingRepo) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestAdminComplianceStatusDefaultsDisabled(t *testing.T) {
	svc := NewSettingService(&adminComplianceSettingRepo{}, nil)

	status, err := svc.GetAdminComplianceStatus(context.Background(), 101)
	require.NoError(t, err)
	require.False(t, status.Enabled)
	require.False(t, status.Required)
	require.NotEmpty(t, status.DocumentHash)
}

func TestAdminComplianceAcknowledgeClearsRequirement(t *testing.T) {
	repo := &adminComplianceSettingRepo{values: map[string]string{
		SettingKeyAdminComplianceEnabled: "true",
	}}
	svc := NewSettingService(repo, nil)

	before, err := svc.GetAdminComplianceStatus(context.Background(), 101)
	require.NoError(t, err)
	require.True(t, before.Enabled)
	require.True(t, before.Required)

	after, err := svc.AcknowledgeAdminCompliance(context.Background(), 101)
	require.NoError(t, err)
	require.True(t, after.Enabled)
	require.False(t, after.Required)
	require.NotNil(t, after.AcknowledgedAt)

	stored, err := repo.GetValue(context.Background(), adminComplianceAcknowledgementKey(101))
	require.NoError(t, err)
	require.Contains(t, stored, after.DocumentHash)
}
