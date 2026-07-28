//go:build unit

package service

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestPanelRateLimitSettingsDefaultsNormalizeAndRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := newMockSettingRepo()
	svc := NewSettingService(repo, &config.Config{})

	defaults, err := svc.GetPanelRateLimitSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, &PanelRateLimitSettings{
		Enabled:     false,
		UserRPM:     240,
		HeavyRPM:    60,
		PublicIPRPM: 300,
		ExemptAdmin: true,
	}, defaults)

	var callbacks atomic.Int32
	svc.SetOnUpdateCallback(func() {
		callbacks.Add(1)
	})

	updated, err := svc.SetPanelRateLimitSettings(ctx, &PanelRateLimitSettings{
		Enabled:     true,
		UserRPM:     -1,
		HeavyRPM:    200000,
		PublicIPRPM: 0,
		ExemptAdmin: false,
	})
	require.NoError(t, err)
	require.Equal(t, &PanelRateLimitSettings{
		Enabled:     true,
		UserRPM:     240,
		HeavyRPM:    maxPanelRateLimitRPM,
		PublicIPRPM: 300,
		ExemptAdmin: false,
	}, updated)
	require.Equal(t, int32(1), callbacks.Load())

	var stored PanelRateLimitSettings
	require.NoError(t, json.Unmarshal([]byte(repo.data[SettingKeyPanelRateLimitSettings]), &stored))
	require.True(t, stored.Enabled)
	require.Equal(t, maxPanelRateLimitRPM, stored.HeavyRPM)
	require.False(t, stored.ExemptAdmin)
}

func TestPanelRateLimitSettingsInvalidJSONReturnsError(t *testing.T) {
	repo := newMockSettingRepo()
	repo.data[SettingKeyPanelRateLimitSettings] = "{"
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPanelRateLimitSettings(context.Background())
	require.Error(t, err)
	require.Nil(t, settings)
}
