//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type mmRepoStub struct {
	getValueFn func(ctx context.Context, key string) (string, error)
	calls      int
}

func (s *mmRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *mmRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	s.calls++
	if s.getValueFn == nil {
		panic("unexpected GetValue call")
	}
	return s.getValueFn(ctx, key)
}

func (s *mmRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *mmRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *mmRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *mmRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *mmRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

type mmUpdateRepoStub struct {
	updates    map[string]string
	getValueFn func(ctx context.Context, key string) (string, error)
}

func (s *mmUpdateRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *mmUpdateRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if s.getValueFn == nil {
		panic("unexpected GetValue call")
	}
	return s.getValueFn(ctx, key)
}

func (s *mmUpdateRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *mmUpdateRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *mmUpdateRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	s.updates = make(map[string]string, len(settings))
	for k, v := range settings {
		s.updates[k] = v
	}
	return nil
}

func (s *mmUpdateRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *mmUpdateRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func resetMaintenanceModeTestCache(t *testing.T) {
	t.Helper()

	maintenanceModeCache.Store((*cachedMaintenanceMode)(nil))
	t.Cleanup(func() {
		maintenanceModeCache.Store((*cachedMaintenanceMode)(nil))
	})
}

func TestIsMaintenanceModeEnabled_ReturnsTrue(t *testing.T) {
	resetMaintenanceModeTestCache(t)

	repo := &mmRepoStub{
		getValueFn: func(ctx context.Context, key string) (string, error) {
			require.Equal(t, SettingKeyMaintenanceModeEnabled, key)
			return "true", nil
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	require.True(t, svc.IsMaintenanceModeEnabled(context.Background()))
	require.Equal(t, 1, repo.calls)
}

func TestIsMaintenanceModeEnabled_ReturnsFalse(t *testing.T) {
	resetMaintenanceModeTestCache(t)

	repo := &mmRepoStub{
		getValueFn: func(ctx context.Context, key string) (string, error) {
			require.Equal(t, SettingKeyMaintenanceModeEnabled, key)
			return "false", nil
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	require.False(t, svc.IsMaintenanceModeEnabled(context.Background()))
	require.Equal(t, 1, repo.calls)
}

func TestIsMaintenanceModeEnabled_ReturnsFalseOnNotFound(t *testing.T) {
	resetMaintenanceModeTestCache(t)

	repo := &mmRepoStub{
		getValueFn: func(ctx context.Context, key string) (string, error) {
			require.Equal(t, SettingKeyMaintenanceModeEnabled, key)
			return "", ErrSettingNotFound
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	require.False(t, svc.IsMaintenanceModeEnabled(context.Background()))
	require.Equal(t, 1, repo.calls)
}

func TestIsMaintenanceModeEnabled_ReturnsFalseOnDBError(t *testing.T) {
	resetMaintenanceModeTestCache(t)

	repo := &mmRepoStub{
		getValueFn: func(ctx context.Context, key string) (string, error) {
			require.Equal(t, SettingKeyMaintenanceModeEnabled, key)
			return "", errors.New("db down")
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	require.False(t, svc.IsMaintenanceModeEnabled(context.Background()))
	require.Equal(t, 1, repo.calls)
}

func TestIsMaintenanceModeEnabled_CachesResult(t *testing.T) {
	resetMaintenanceModeTestCache(t)

	repo := &mmRepoStub{
		getValueFn: func(ctx context.Context, key string) (string, error) {
			require.Equal(t, SettingKeyMaintenanceModeEnabled, key)
			return "true", nil
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	require.True(t, svc.IsMaintenanceModeEnabled(context.Background()))
	require.True(t, svc.IsMaintenanceModeEnabled(context.Background()))
	require.Equal(t, 1, repo.calls)
}

func TestUpdateSettings_InvalidatesMaintenanceModeCache(t *testing.T) {
	resetMaintenanceModeTestCache(t)

	maintenanceModeCache.Store(&cachedMaintenanceMode{
		value:     true,
		expiresAt: time.Now().Add(maintenanceModeCacheTTL).UnixNano(),
	})

	repo := &mmUpdateRepoStub{
		getValueFn: func(ctx context.Context, key string) (string, error) {
			require.Equal(t, SettingKeyMaintenanceModeEnabled, key)
			return "false", nil
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	err := svc.UpdateSettings(context.Background(), &SystemSettings{
		MaintenanceModeEnabled: false,
	})
	require.NoError(t, err)
	require.Equal(t, "false", repo.updates[SettingKeyMaintenanceModeEnabled])
	require.False(t, svc.IsMaintenanceModeEnabled(context.Background()))
}
