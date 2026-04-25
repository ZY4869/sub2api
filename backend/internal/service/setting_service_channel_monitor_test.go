//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSettingService_GetChannelMonitorRuntime_ClampsInterval(t *testing.T) {
	ctx := context.Background()
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyChannelMonitorEnabled:                "true",
			SettingKeyChannelMonitorDefaultIntervalSeconds: "5",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	runtime, err := svc.GetChannelMonitorRuntime(ctx)
	require.NoError(t, err)
	require.True(t, runtime.Enabled)
	require.Equal(t, 15, runtime.DefaultIntervalSeconds)

	repo.values[SettingKeyChannelMonitorDefaultIntervalSeconds] = "3601"
	runtime, err = svc.GetChannelMonitorRuntime(ctx)
	require.NoError(t, err)
	require.Equal(t, 3600, runtime.DefaultIntervalSeconds)

	repo.values[SettingKeyChannelMonitorDefaultIntervalSeconds] = "60"
	runtime, err = svc.GetChannelMonitorRuntime(ctx)
	require.NoError(t, err)
	require.Equal(t, 60, runtime.DefaultIntervalSeconds)
}
