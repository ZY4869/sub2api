//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

// --- mock: UserRepository ---

type mockUserRepo struct {
	updateBalanceErr error
	updateBalanceFn  func(ctx context.Context, id int64, amount float64) error
	getByIDFn        func(ctx context.Context, id int64) (*User, error)
	updateFn         func(ctx context.Context, user *User) error
	existsByEmailFn  func(ctx context.Context, email string) (bool, error)
}

func (m *mockUserRepo) Create(context.Context, *User) error { return nil }
func (m *mockUserRepo) GetByID(ctx context.Context, id int64) (*User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &User{}, nil
}
func (m *mockUserRepo) GetByEmail(context.Context, string) (*User, error) { return &User{}, nil }
func (m *mockUserRepo) GetFirstAdmin(context.Context) (*User, error)      { return &User{}, nil }
func (m *mockUserRepo) Update(ctx context.Context, user *User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	return nil
}
func (m *mockUserRepo) Delete(context.Context, int64) error { return nil }
func (m *mockUserRepo) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockUserRepo) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockUserRepo) UpdateBalance(ctx context.Context, id int64, amount float64) error {
	if m.updateBalanceFn != nil {
		return m.updateBalanceFn(ctx, id, amount)
	}
	return m.updateBalanceErr
}
func (m *mockUserRepo) DeductBalance(context.Context, int64, float64) error { return nil }
func (m *mockUserRepo) UpdateConcurrency(context.Context, int64, int) error { return nil }
func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailFn != nil {
		return m.existsByEmailFn(ctx, email)
	}
	return false, nil
}
func (m *mockUserRepo) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	return 0, nil
}
func (m *mockUserRepo) AddGroupToAllowedGroups(context.Context, int64, int64) error { return nil }
func (m *mockUserRepo) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	return nil
}
func (m *mockUserRepo) UpdateTotpSecret(context.Context, int64, *string) error { return nil }
func (m *mockUserRepo) EnableTotp(context.Context, int64) error                { return nil }
func (m *mockUserRepo) DisableTotp(context.Context, int64) error               { return nil }

// --- mock: APIKeyAuthCacheInvalidator ---

type mockAuthCacheInvalidator struct {
	invalidatedUserIDs []int64
	mu                 sync.Mutex
}

func (m *mockAuthCacheInvalidator) InvalidateAuthCacheByKey(context.Context, string)    {}
func (m *mockAuthCacheInvalidator) InvalidateAuthCacheByGroupID(context.Context, int64) {}
func (m *mockAuthCacheInvalidator) InvalidateAuthCacheByUserID(_ context.Context, userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invalidatedUserIDs = append(m.invalidatedUserIDs, userID)
}

// --- mock: BillingCache ---

type mockBillingCache struct {
	invalidateErr       error
	invalidateCallCount atomic.Int64
	invalidatedUserIDs  []int64
	mu                  sync.Mutex
}

func (m *mockBillingCache) GetUserBalance(context.Context, int64) (float64, error)  { return 0, nil }
func (m *mockBillingCache) SetUserBalance(context.Context, int64, float64) error    { return nil }
func (m *mockBillingCache) DeductUserBalance(context.Context, int64, float64) error { return nil }
func (m *mockBillingCache) InvalidateUserBalance(_ context.Context, userID int64) error {
	m.invalidateCallCount.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invalidatedUserIDs = append(m.invalidatedUserIDs, userID)
	return m.invalidateErr
}
func (m *mockBillingCache) GetSubscriptionCache(context.Context, int64, int64) (*SubscriptionCacheData, error) {
	return nil, nil
}
func (m *mockBillingCache) SetSubscriptionCache(context.Context, int64, int64, *SubscriptionCacheData) error {
	return nil
}
func (m *mockBillingCache) UpdateSubscriptionUsage(context.Context, int64, int64, float64) error {
	return nil
}
func (m *mockBillingCache) InvalidateSubscriptionCache(context.Context, int64, int64) error {
	return nil
}
func (m *mockBillingCache) GetAPIKeyRateLimit(context.Context, int64) (*APIKeyRateLimitCacheData, error) {
	return nil, nil
}
func (m *mockBillingCache) SetAPIKeyRateLimit(context.Context, int64, *APIKeyRateLimitCacheData) error {
	return nil
}
func (m *mockBillingCache) UpdateAPIKeyRateLimitUsage(context.Context, int64, float64) error {
	return nil
}
func (m *mockBillingCache) InvalidateAPIKeyRateLimit(context.Context, int64) error {
	return nil
}

// --- 测试 ---

func TestUpdateBalance_Success(t *testing.T) {
	repo := &mockUserRepo{}
	cache := &mockBillingCache{}
	svc := NewUserService(repo, nil, cache)

	err := svc.UpdateBalance(context.Background(), 42, 100.0)
	require.NoError(t, err)

	// 等待异步 goroutine 完成
	require.Eventually(t, func() bool {
		return cache.invalidateCallCount.Load() == 1
	}, 2*time.Second, 10*time.Millisecond, "应异步调用 InvalidateUserBalance")

	cache.mu.Lock()
	defer cache.mu.Unlock()
	require.Equal(t, []int64{42}, cache.invalidatedUserIDs, "应对 userID=42 失效缓存")
}

func TestUpdateBalance_NilBillingCache_NoPanic(t *testing.T) {
	repo := &mockUserRepo{}
	svc := NewUserService(repo, nil, nil) // billingCache = nil

	err := svc.UpdateBalance(context.Background(), 1, 50.0)
	require.NoError(t, err, "billingCache 为 nil 时不应 panic")
}

func TestUpdateBalance_CacheFailure_DoesNotAffectReturn(t *testing.T) {
	repo := &mockUserRepo{}
	cache := &mockBillingCache{invalidateErr: errors.New("redis connection refused")}
	svc := NewUserService(repo, nil, cache)

	err := svc.UpdateBalance(context.Background(), 99, 200.0)
	require.NoError(t, err, "缓存失效失败不应影响主流程返回值")

	// 等待异步 goroutine 完成（即使失败也应调用）
	require.Eventually(t, func() bool {
		return cache.invalidateCallCount.Load() == 1
	}, 2*time.Second, 10*time.Millisecond, "即使失败也应调用 InvalidateUserBalance")
}

func TestUpdateBalance_RepoError_ReturnsError(t *testing.T) {
	repo := &mockUserRepo{updateBalanceErr: errors.New("database error")}
	cache := &mockBillingCache{}
	svc := NewUserService(repo, nil, cache)

	err := svc.UpdateBalance(context.Background(), 1, 100.0)
	require.Error(t, err, "repo 失败时应返回错误")
	require.Contains(t, err.Error(), "update balance")

	// repo 失败时不应触发缓存失效
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, int64(0), cache.invalidateCallCount.Load(),
		"repo 失败时不应调用 InvalidateUserBalance")
}

func TestUpdateBalance_WithAuthCacheInvalidator(t *testing.T) {
	repo := &mockUserRepo{}
	auth := &mockAuthCacheInvalidator{}
	cache := &mockBillingCache{}
	svc := NewUserService(repo, auth, cache)

	err := svc.UpdateBalance(context.Background(), 77, 300.0)
	require.NoError(t, err)

	// 验证 auth cache 同步失效
	auth.mu.Lock()
	require.Equal(t, []int64{77}, auth.invalidatedUserIDs)
	auth.mu.Unlock()

	// 验证 billing cache 异步失效
	require.Eventually(t, func() bool {
		return cache.invalidateCallCount.Load() == 1
	}, 2*time.Second, 10*time.Millisecond)
}

func TestNewUserService_FieldsAssignment(t *testing.T) {
	repo := &mockUserRepo{}
	auth := &mockAuthCacheInvalidator{}
	cache := &mockBillingCache{}

	svc := NewUserService(repo, auth, cache)
	require.NotNil(t, svc)
	require.Equal(t, repo, svc.userRepo)
	require.Equal(t, auth, svc.authCacheInvalidator)
	require.Equal(t, cache, svc.billingCache)
}

func TestUpdateProfile_UsageModelDisplayMode_DefaultFallback(t *testing.T) {
	user := &User{
		ID:                    1,
		Email:                 "alice@example.com",
		Username:              "alice",
		Concurrency:           5,
		Status:                StatusActive,
		UsageModelDisplayMode: "",
	}
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			clone := *user
			return &clone, nil
		},
		updateFn: func(_ context.Context, updated *User) error {
			user.UsageModelDisplayMode = updated.UsageModelDisplayMode
			return nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	updated, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		Username: ptrString("alice-2"),
	})
	require.NoError(t, err)
	require.Equal(t, UsageModelDisplayModeModelOnly, updated.EffectiveUsageModelDisplayMode())
	require.Equal(t, UsageModelDisplayModeModelOnly, NormalizeUserUsageModelDisplayMode(user.UsageModelDisplayMode))
}

func TestUpdateProfile_UsageModelDisplayMode_Success(t *testing.T) {
	user := &User{
		ID:                    1,
		Email:                 "alice@example.com",
		Username:              "alice",
		Concurrency:           5,
		Status:                StatusActive,
		UsageModelDisplayMode: UsageModelDisplayModeModelOnly,
	}
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			clone := *user
			return &clone, nil
		},
		updateFn: func(_ context.Context, updated *User) error {
			user.Username = updated.Username
			user.UsageModelDisplayMode = updated.UsageModelDisplayMode
			return nil
		},
		existsByEmailFn: func(context.Context, string) (bool, error) { return false, nil },
	}
	svc := NewUserService(repo, nil, nil)

	updated, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		Username:              ptrString("alice-2"),
		UsageModelDisplayMode: ptrString(UsageModelDisplayModeDisplayAndModel),
	})
	require.NoError(t, err)
	require.Equal(t, "alice-2", updated.Username)
	require.Equal(t, UsageModelDisplayModeDisplayAndModel, updated.UsageModelDisplayMode)
	require.Equal(t, UsageModelDisplayModeDisplayAndModel, user.UsageModelDisplayMode)
}

func TestUpdateProfile_UsageModelDisplayMode_Invalid(t *testing.T) {
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			return &User{
				ID:                    1,
				Email:                 "alice@example.com",
				Username:              "alice",
				UsageModelDisplayMode: UsageModelDisplayModeModelOnly,
			}, nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	_, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		UsageModelDisplayMode: ptrString("bad-mode"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "usage_model_display_mode")
}

func TestUpdateProfile_RealtimeCountdownPreferences_PartialUpdate(t *testing.T) {
	user := &User{
		ID:                              1,
		Email:                           "alice@example.com",
		Username:                        "alice",
		GlobalRealtimeCountdownEnabled:  false,
		AccountRealtimeCountdownEnabled: true,
	}
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			clone := *user
			return &clone, nil
		},
		updateFn: func(_ context.Context, updated *User) error {
			user.GlobalRealtimeCountdownEnabled = updated.GlobalRealtimeCountdownEnabled
			user.AccountRealtimeCountdownEnabled = updated.AccountRealtimeCountdownEnabled
			return nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	updated, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		GlobalRealtimeCountdownEnabled: ptrBool(true),
	})
	require.NoError(t, err)
	require.True(t, updated.GlobalRealtimeCountdownEnabled)
	require.True(t, user.GlobalRealtimeCountdownEnabled)
	require.True(t, updated.AccountRealtimeCountdownEnabled)
	require.True(t, user.AccountRealtimeCountdownEnabled)
}

func TestUpdateProfile_VisualPreset_DefaultFallback(t *testing.T) {
	user := &User{
		ID:                          1,
		Email:                       "alice@example.com",
		Username:                    "alice",
		VisualPresetPreference:      "",
		AccountVisualPresetOverride: "",
	}
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			clone := *user
			return &clone, nil
		},
		updateFn: func(_ context.Context, updated *User) error {
			user.VisualPresetPreference = updated.VisualPresetPreference
			user.AccountVisualPresetOverride = updated.AccountVisualPresetOverride
			return nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	updated, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		Username: ptrString("alice-2"),
	})
	require.NoError(t, err)
	require.Equal(t, VisualPresetClassic, updated.EffectiveVisualPreset(VisualPresetClassic))
	require.Equal(t, VisualPresetPreferenceInherit, NormalizeVisualPresetPreference(user.VisualPresetPreference))
	require.Equal(t, VisualPresetPreferenceInherit, NormalizeVisualPresetPreference(user.AccountVisualPresetOverride))
}

func TestUpdateProfile_VisualPreset_Success(t *testing.T) {
	user := &User{
		ID:                          1,
		Email:                       "alice@example.com",
		Username:                    "alice",
		VisualPresetPreference:      VisualPresetPreferenceInherit,
		AccountVisualPresetOverride: VisualPresetPreferenceInherit,
	}
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			clone := *user
			return &clone, nil
		},
		updateFn: func(_ context.Context, updated *User) error {
			user.Username = updated.Username
			user.VisualPresetPreference = updated.VisualPresetPreference
			user.AccountVisualPresetOverride = updated.AccountVisualPresetOverride
			return nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	updated, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		Username:                    ptrString("alice-2"),
		VisualPresetPreference:      ptrString(VisualPresetAiry),
		AccountVisualPresetOverride: ptrString(VisualPresetClassic),
	})
	require.NoError(t, err)
	require.Equal(t, "alice-2", updated.Username)
	require.Equal(t, VisualPresetAiry, updated.VisualPresetPreference)
	require.Equal(t, VisualPresetClassic, updated.AccountVisualPresetOverride)
	require.Equal(t, VisualPresetAiry, user.VisualPresetPreference)
	require.Equal(t, VisualPresetClassic, user.AccountVisualPresetOverride)
	require.Equal(t, VisualPresetClassic, updated.EffectiveVisualPreset(VisualPresetClassic))
}

func TestUpdateProfile_VisualPresetPreference_Invalid(t *testing.T) {
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			return &User{
				ID:                          1,
				Email:                       "alice@example.com",
				Username:                    "alice",
				VisualPresetPreference:      VisualPresetPreferenceInherit,
				AccountVisualPresetOverride: VisualPresetPreferenceInherit,
			}, nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	_, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		VisualPresetPreference: ptrString("bad-style"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "visual_preset_preference")
}

func TestUpdateProfile_AccountVisualPresetOverride_Invalid(t *testing.T) {
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			return &User{
				ID:                          1,
				Email:                       "alice@example.com",
				Username:                    "alice",
				VisualPresetPreference:      VisualPresetPreferenceInherit,
				AccountVisualPresetOverride: VisualPresetPreferenceInherit,
			}, nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	_, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		AccountVisualPresetOverride: ptrString("bad-style"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "account_visual_preset_override")
}

func TestUpdateProfile_VisualPreset_PartialUpdateDoesNotOverrideOtherPreferences(t *testing.T) {
	user := &User{
		ID:                              1,
		Email:                           "alice@example.com",
		Username:                        "alice",
		UsageModelDisplayMode:           UsageModelDisplayModeDisplayOnly,
		GlobalRealtimeCountdownEnabled:  true,
		AccountRealtimeCountdownEnabled: false,
		VisualPresetPreference:          VisualPresetAiry,
		AccountVisualPresetOverride:     VisualPresetPreferenceInherit,
	}
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			clone := *user
			return &clone, nil
		},
		updateFn: func(_ context.Context, updated *User) error {
			user.UsageModelDisplayMode = updated.UsageModelDisplayMode
			user.GlobalRealtimeCountdownEnabled = updated.GlobalRealtimeCountdownEnabled
			user.AccountRealtimeCountdownEnabled = updated.AccountRealtimeCountdownEnabled
			user.VisualPresetPreference = updated.VisualPresetPreference
			user.AccountVisualPresetOverride = updated.AccountVisualPresetOverride
			return nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	updated, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		AccountVisualPresetOverride: ptrString(VisualPresetClassic),
	})
	require.NoError(t, err)
	require.Equal(t, UsageModelDisplayModeDisplayOnly, updated.UsageModelDisplayMode)
	require.True(t, updated.GlobalRealtimeCountdownEnabled)
	require.False(t, updated.AccountRealtimeCountdownEnabled)
	require.Equal(t, VisualPresetAiry, updated.VisualPresetPreference)
	require.Equal(t, VisualPresetClassic, updated.AccountVisualPresetOverride)
	require.Equal(t, VisualPresetClassic, updated.EffectiveVisualPreset(VisualPresetClassic))
}

func TestUpdateProfile_AccountDisplayPreferences_DefaultFallback(t *testing.T) {
	user := &User{
		ID:                       1,
		Email:                    "alice@example.com",
		Username:                 "alice",
		AccountTodayStatsWindows: nil,
		AccountGroupDisplayMode:  "",
	}
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			clone := *user
			return &clone, nil
		},
		updateFn: func(_ context.Context, updated *User) error {
			user.AccountTodayStatsWindows = updated.AccountTodayStatsWindows
			user.AccountGroupDisplayMode = updated.AccountGroupDisplayMode
			return nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	updated, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		Username: ptrString("alice-2"),
	})
	require.NoError(t, err)
	require.Equal(t, DefaultAccountTodayStatsWindows(), NormalizeAccountTodayStatsWindows(updated.AccountTodayStatsWindows))
	require.Equal(t, AccountGroupDisplayModeFull, NormalizeAccountGroupDisplayMode(updated.AccountGroupDisplayMode))
}

func TestUpdateProfile_AccountDisplayPreferences_Success(t *testing.T) {
	user := &User{
		ID:                       1,
		Email:                    "alice@example.com",
		Username:                 "alice",
		AccountTodayStatsWindows: DefaultAccountTodayStatsWindows(),
		AccountGroupDisplayMode:  AccountGroupDisplayModeFull,
	}
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			clone := *user
			return &clone, nil
		},
		updateFn: func(_ context.Context, updated *User) error {
			user.AccountTodayStatsWindows = updated.AccountTodayStatsWindows
			user.AccountGroupDisplayMode = updated.AccountGroupDisplayMode
			return nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	updated, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		AccountTodayStatsWindows: []string{AccountTodayStatsWindowToday, AccountTodayStatsWindowTotal},
		AccountGroupDisplayMode:  ptrString(AccountGroupDisplayModeIcon),
	})
	require.NoError(t, err)
	require.Equal(t, []string{AccountTodayStatsWindowToday, AccountTodayStatsWindowTotal}, updated.AccountTodayStatsWindows)
	require.Equal(t, AccountGroupDisplayModeIcon, updated.AccountGroupDisplayMode)
	require.Equal(t, updated.AccountTodayStatsWindows, user.AccountTodayStatsWindows)
	require.Equal(t, updated.AccountGroupDisplayMode, user.AccountGroupDisplayMode)
}

func TestUpdateProfile_AccountTodayStatsWindows_Invalid(t *testing.T) {
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			return &User{
				ID:                       1,
				Email:                    "alice@example.com",
				Username:                 "alice",
				AccountTodayStatsWindows: DefaultAccountTodayStatsWindows(),
				AccountGroupDisplayMode:  AccountGroupDisplayModeFull,
			}, nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	_, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		AccountTodayStatsWindows: []string{AccountTodayStatsWindowToday, "bad"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "account_today_stats_windows")

	_, err = svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		AccountTodayStatsWindows: []string{},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "account_today_stats_windows")
}

func TestUpdateProfile_AccountGroupDisplayMode_Invalid(t *testing.T) {
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			return &User{
				ID:                       1,
				Email:                    "alice@example.com",
				Username:                 "alice",
				AccountTodayStatsWindows: DefaultAccountTodayStatsWindows(),
				AccountGroupDisplayMode:  AccountGroupDisplayModeFull,
			}, nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	_, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		AccountGroupDisplayMode: ptrString("bad"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "account_group_display_mode")
}

func TestUpdateProfile_AccountDisplayPreferences_PartialUpdateDoesNotOverrideOtherPreferences(t *testing.T) {
	user := &User{
		ID:                       1,
		Email:                    "alice@example.com",
		Username:                 "alice",
		AccountTodayStatsWindows: []string{AccountTodayStatsWindowToday, AccountTodayStatsWindowTotal},
		AccountGroupDisplayMode:  AccountGroupDisplayModeIcon,
	}
	repo := &mockUserRepo{
		getByIDFn: func(context.Context, int64) (*User, error) {
			clone := *user
			return &clone, nil
		},
		updateFn: func(_ context.Context, updated *User) error {
			user.AccountTodayStatsWindows = updated.AccountTodayStatsWindows
			user.AccountGroupDisplayMode = updated.AccountGroupDisplayMode
			return nil
		},
	}
	svc := NewUserService(repo, nil, nil)

	updated, err := svc.UpdateProfile(context.Background(), 1, UpdateProfileRequest{
		AccountGroupDisplayMode: ptrString(AccountGroupDisplayModeFull),
	})
	require.NoError(t, err)
	require.Equal(t, []string{AccountTodayStatsWindowToday, AccountTodayStatsWindowTotal}, updated.AccountTodayStatsWindows)
	require.Equal(t, AccountGroupDisplayModeFull, updated.AccountGroupDisplayMode)
}

func ptrString(value string) *string {
	return &value
}

func ptrBool(value bool) *bool {
	return &value
}
