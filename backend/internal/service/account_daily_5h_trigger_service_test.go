//go:build unit

package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/stretchr/testify/require"
)

type daily5HSettingServiceStub struct {
	settings   *AccountDaily5HTriggerSettings
	candidates []AccountDaily5HTriggerAccountTypeSummary
}

func (s *daily5HSettingServiceStub) GetAccountDaily5HTriggerSettings(context.Context) (*AccountDaily5HTriggerSettings, error) {
	if s.settings == nil {
		return DefaultAccountDaily5HTriggerSettings(), nil
	}
	return NormalizeAccountDaily5HTriggerSettings(s.settings), nil
}

type accountDaily5HRepoStub struct {
	accounts         []Account
	updateExtraCalls []expiryRepoUpdateExtraCall
}

func (r *accountDaily5HRepoStub) Create(context.Context, *Account) error { panic("unexpected") }
func (r *accountDaily5HRepoStub) GetByID(context.Context, int64) (*Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) GetByIDs(context.Context, []int64) ([]*Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ExistsByID(context.Context, int64) (bool, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) GetByCRSAccountID(context.Context, string) (*Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) FindByExtraField(context.Context, string, any) ([]Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListCRSAccountIDs(context.Context) (map[string]int64, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) Update(context.Context, *Account) error { panic("unexpected") }
func (r *accountDaily5HRepoStub) Delete(context.Context, int64) error    { panic("unexpected") }
func (r *accountDaily5HRepoStub) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListWithFilters(_ context.Context, _ pagination.PaginationParams, _, _, _, _ string, _ int64, lifecycle, _ string) ([]Account, *pagination.PaginationResult, error) {
	if NormalizeAccountLifecycleInput(lifecycle) == AccountLifecycleAll {
		return append([]Account(nil), r.accounts...), nil, nil
	}
	filtered := make([]Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if NormalizeAccountLifecycleInput(account.LifecycleState) != NormalizeAccountLifecycleInput(lifecycle) {
			continue
		}
		filtered = append(filtered, account)
	}
	return filtered, nil, nil
}
func (r *accountDaily5HRepoStub) GetStatusSummary(context.Context, AccountStatusSummaryFilters) (*AccountStatusSummary, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListByGroup(context.Context, int64) ([]Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListActive(context.Context) ([]Account, error) { panic("unexpected") }
func (r *accountDaily5HRepoStub) ListByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) UpdateLastUsed(context.Context, int64) error { panic("unexpected") }
func (r *accountDaily5HRepoStub) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) SetError(context.Context, int64, string) error { panic("unexpected") }
func (r *accountDaily5HRepoStub) ClearError(context.Context, int64) error       { panic("unexpected") }
func (r *accountDaily5HRepoStub) SetSchedulable(context.Context, int64, bool) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) BindGroups(context.Context, int64, []int64) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListSchedulable(context.Context) ([]Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) SetRateLimited(context.Context, int64, time.Time) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) SetOverloaded(context.Context, int64, time.Time) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ClearTempUnschedulable(context.Context, int64) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ClearRateLimit(context.Context, int64) error { panic("unexpected") }
func (r *accountDaily5HRepoStub) ClearAntigravityQuotaScopes(context.Context, int64) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ClearModelRateLimits(context.Context, int64) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	r.updateExtraCalls = append(r.updateExtraCalls, expiryRepoUpdateExtraCall{
		id:      id,
		updates: cloneStringAnyMap(updates),
	})
	for index := range r.accounts {
		if r.accounts[index].ID != id {
			continue
		}
		if r.accounts[index].Extra == nil {
			r.accounts[index].Extra = map[string]any{}
		}
		for key, value := range updates {
			r.accounts[index].Extra[key] = value
		}
		break
	}
	return nil
}
func (r *accountDaily5HRepoStub) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) MarkBlacklisted(context.Context, int64, string, string, time.Time, time.Time) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) RestoreBlacklisted(context.Context, int64) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListBlacklistedIDs(context.Context) ([]int64, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ListBlacklistedForPurge(context.Context, time.Time, int) ([]Account, error) {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) IncrementQuotaUsed(context.Context, int64, float64) error {
	panic("unexpected")
}
func (r *accountDaily5HRepoStub) ResetQuotaUsed(context.Context, int64) error { panic("unexpected") }

type accountDaily5HExecutorStub struct {
	result *BackgroundAccountTestResult
	err    error
	calls  []ScheduledTestExecutionInput
}

func (s *accountDaily5HExecutorStub) RunTestBackgroundDetailed(_ context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error) {
	s.calls = append(s.calls, input)
	return s.result, s.err
}

func TestAccountDaily5HTriggerService_RunOnce_AfterSevenExecutesOnlyOncePerLocalDate(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	now := time.Date(2026, 5, 8, 7, 5, 0, 0, time.UTC)
	account := Account{
		ID:          11,
		Name:        "ChatGPT OAuth",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"manual_models": []any{
				map[string]any{"model_id": "gpt-5.4-mini", "provider": PlatformOpenAI},
				map[string]any{"model_id": "gpt-4.1", "provider": PlatformOpenAI},
			},
		},
	}
	repo := &accountDaily5HRepoStub{accounts: []Account{account}}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	executor := &accountDaily5HExecutorStub{result: &BackgroundAccountTestResult{Status: "success"}}
	svc := NewAccountDaily5HTriggerService(repo, executor, settings, nil, time.Minute)
	svc.SetNow(func() time.Time { return now })
	svc.SetLocation(time.UTC)
	settings.SetAccountDaily5HTriggerCandidateProvider(svc)
	_, _ = settings.UpdateAccountDaily5HTriggerSettings(context.Background(), &AccountDaily5HTriggerSettings{
		Enabled:              true,
		SelectedAccountTypes: []string{AccountDaily5HTypeOpenAI},
		OpenAIModel:          AccountDaily5HTriggerModelSettings{Mode: AccountDaily5HModelModeAuto},
	})

	svc.runOnce(context.Background())
	svc.runOnce(context.Background())

	require.Len(t, executor.calls, 1)
	require.Equal(t, accountDaily5HPrompt, executor.calls[0].Prompt)
	require.Equal(t, "gpt-5.4-mini", executor.calls[0].ModelID)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, AccountDaily5HTriggerStatusSuccess, repo.updateExtraCalls[0].updates[accountDaily5HLastStatusKey])
	require.Equal(t, "2026-05-08", repo.updateExtraCalls[0].updates[accountDaily5HLastLocalDateKey])
	require.Equal(t, "Daily 5H trigger succeeded.", repo.updateExtraCalls[0].updates[accountDaily5HLastSummaryKey])
	snapshot := protocolruntime.Snapshot()
	require.EqualValues(t, 1, snapshot.RecoveryProbeResultByReason["daily_5h_trigger"])
	require.EqualValues(t, 1, snapshot.RecoveryProbeResultByStatus[AccountDaily5HTriggerStatusSuccess])
}

func TestAccountDaily5HTriggerService_RunOnce_BeforeSevenDoesNotExecute(t *testing.T) {
	now := time.Date(2026, 5, 8, 6, 59, 0, 0, time.UTC)
	account := Account{
		ID:          12,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
	}
	repo := &accountDaily5HRepoStub{accounts: []Account{account}}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	executor := &accountDaily5HExecutorStub{result: &BackgroundAccountTestResult{Status: "success"}}
	svc := NewAccountDaily5HTriggerService(repo, executor, settings, nil, time.Minute)
	svc.SetNow(func() time.Time { return now })
	svc.SetLocation(time.UTC)
	_, _ = settings.UpdateAccountDaily5HTriggerSettings(context.Background(), &AccountDaily5HTriggerSettings{
		Enabled:              true,
		SelectedAccountTypes: []string{AccountDaily5HTypeOpenAI},
	})

	svc.runOnce(context.Background())

	require.Empty(t, executor.calls)
	require.Empty(t, repo.updateExtraCalls)
}

func TestAccountDaily5HTriggerService_RunOnce_SkipsCNHolidayWithoutAccountWrites(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	now := time.Date(2026, 10, 2, 7, 5, 0, 0, time.UTC)
	account := Account{
		ID:          41,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
	}
	repo := &accountDaily5HRepoStub{accounts: []Account{account}}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	executor := &accountDaily5HExecutorStub{result: &BackgroundAccountTestResult{Status: "success"}}
	svc := NewAccountDaily5HTriggerService(repo, executor, settings, nil, time.Minute)
	svc.SetNow(func() time.Time { return now })
	svc.SetLocation(time.UTC)
	_, _ = settings.UpdateAccountDaily5HTriggerSettings(context.Background(), &AccountDaily5HTriggerSettings{
		Enabled:                   true,
		SelectedAccountTypes:      []string{AccountDaily5HTypeOpenAI},
		SkipCNHolidaysAndWeekends: true,
		OpenAIModel:               AccountDaily5HTriggerModelSettings{Mode: AccountDaily5HModelModeAuto},
	})

	svc.runOnce(context.Background())

	require.Empty(t, executor.calls)
	require.Empty(t, repo.updateExtraCalls)
	snapshot := protocolruntime.Snapshot()
	require.EqualValues(t, 1, snapshot.RecoveryProbeResultByReason["daily_5h_trigger"])
	require.EqualValues(t, 1, snapshot.RecoveryProbeResultByStatus[AccountDaily5HTriggerStatusSkipped])
}

func TestAccountDaily5HTriggerService_RunOnce_SkippedNonWorkdayDoesNotBlockNextWorkday(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	current := time.Date(2026, 10, 3, 7, 5, 0, 0, time.UTC) // Saturday during CN National Day holiday.
	account := Account{
		ID:          42,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"manual_models": []any{
				map[string]any{"model_id": "gpt-5.4-mini", "provider": PlatformOpenAI},
			},
		},
	}
	repo := &accountDaily5HRepoStub{accounts: []Account{account}}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	executor := &accountDaily5HExecutorStub{result: &BackgroundAccountTestResult{Status: "success"}}
	svc := NewAccountDaily5HTriggerService(repo, executor, settings, nil, time.Minute)
	svc.SetNow(func() time.Time { return current })
	svc.SetLocation(time.UTC)
	_, _ = settings.UpdateAccountDaily5HTriggerSettings(context.Background(), &AccountDaily5HTriggerSettings{
		Enabled:                   true,
		SelectedAccountTypes:      []string{AccountDaily5HTypeOpenAI},
		SkipCNHolidaysAndWeekends: true,
		OpenAIModel:               AccountDaily5HTriggerModelSettings{Mode: AccountDaily5HModelModeAuto},
	})

	svc.runOnce(context.Background())
	require.Empty(t, executor.calls)
	require.Empty(t, repo.updateExtraCalls)
	require.Empty(t, AccountDaily5HLastLocalDate(repo.accounts[0].Extra))

	current = time.Date(2026, 10, 8, 7, 5, 0, 0, time.UTC)
	svc.runOnce(context.Background())

	require.Len(t, executor.calls, 1)
	require.Equal(t, "gpt-5.4-mini", executor.calls[0].ModelID)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, "2026-10-08", repo.updateExtraCalls[0].updates[accountDaily5HLastLocalDateKey])
	require.Equal(t, AccountDaily5HTriggerStatusSuccess, repo.updateExtraCalls[0].updates[accountDaily5HLastStatusKey])
}

func TestAccountDaily5HTriggerService_IncludePausedAccountsHonorsWindowFiltering(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	now := time.Date(2026, 5, 8, 7, 10, 0, 0, time.UTC)
	blockedUntil := now.Add(15 * time.Minute)
	pausedAccount := Account{
		ID:          13,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusDisabled,
		Schedulable: false,
		Extra: map[string]any{
			"manual_models": []any{
				map[string]any{"model_id": "gpt-5.4-mini", "provider": PlatformOpenAI},
			},
		},
	}
	windowBlocked := pausedAccount
	windowBlocked.ID = 14
	windowBlocked.TempUnschedulableUntil = &blockedUntil
	windowBlocked.Extra = cloneStringAnyMap(pausedAccount.Extra)
	repo := &accountDaily5HRepoStub{accounts: []Account{pausedAccount, windowBlocked}}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	executor := &accountDaily5HExecutorStub{result: &BackgroundAccountTestResult{Status: "success"}}
	svc := NewAccountDaily5HTriggerService(repo, executor, settings, nil, time.Minute)
	svc.SetNow(func() time.Time { return now })
	svc.SetLocation(time.UTC)
	_, _ = settings.UpdateAccountDaily5HTriggerSettings(context.Background(), &AccountDaily5HTriggerSettings{
		Enabled:               true,
		SelectedAccountTypes:  []string{AccountDaily5HTypeOpenAI},
		IncludePausedAccounts: true,
	})

	svc.runOnce(context.Background())

	require.Len(t, executor.calls, 1)
	require.Equal(t, int64(13), executor.calls[0].AccountID)
	require.Len(t, repo.updateExtraCalls, 2)
	require.Equal(t, AccountDaily5HTriggerStatusSuccess, repo.updateExtraCalls[0].updates[accountDaily5HLastStatusKey])
	require.Equal(t, AccountDaily5HTriggerStatusSkipped, repo.updateExtraCalls[1].updates[accountDaily5HLastStatusKey])
	require.Equal(t, "Account is temporarily unschedulable and is skipped for the daily 5H trigger.", repo.updateExtraCalls[1].updates[accountDaily5HLastSummaryKey])
	snapshot := protocolruntime.Snapshot()
	require.EqualValues(t, 2, snapshot.RecoveryProbeResultByReason["daily_5h_trigger"])
	require.EqualValues(t, 1, snapshot.RecoveryProbeResultByStatus[AccountDaily5HTriggerStatusSuccess])
	require.EqualValues(t, 1, snapshot.RecoveryProbeResultByStatus[AccountDaily5HTriggerStatusSkipped])
}

func TestAccountDaily5HTriggerService_RunOnce_IgnoreFreeAccountsSkipsOnlyOpenAIFree(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	now := time.Date(2026, 5, 8, 7, 12, 0, 0, time.UTC)
	buildOpenAIAccount := func(id int64, plan string) Account {
		return Account{
			ID:             id,
			Platform:       PlatformOpenAI,
			Type:           AccountTypeOAuth,
			Status:         StatusActive,
			Schedulable:    true,
			LifecycleState: AccountLifecycleNormal,
			Credentials:    map[string]any{"plan_type": plan},
			Extra: map[string]any{
				"manual_models": []any{
					map[string]any{"model_id": "gpt-5.4-mini", "provider": PlatformOpenAI},
				},
			},
		}
	}
	repo := &accountDaily5HRepoStub{accounts: []Account{
		buildOpenAIAccount(31, "free"),
		buildOpenAIAccount(32, "plus"),
		buildOpenAIAccount(33, "pro"),
	}}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	executor := &accountDaily5HExecutorStub{result: &BackgroundAccountTestResult{Status: "success"}}
	svc := NewAccountDaily5HTriggerService(repo, executor, settings, nil, time.Minute)
	svc.SetNow(func() time.Time { return now })
	svc.SetLocation(time.UTC)
	_, _ = settings.UpdateAccountDaily5HTriggerSettings(context.Background(), &AccountDaily5HTriggerSettings{
		Enabled:              true,
		SelectedAccountTypes: []string{AccountDaily5HTypeOpenAI},
		IgnoreFreeAccounts:   true,
	})

	svc.runOnce(context.Background())

	require.Len(t, executor.calls, 2)
	require.Equal(t, int64(32), executor.calls[0].AccountID)
	require.Equal(t, int64(33), executor.calls[1].AccountID)
	require.Len(t, repo.updateExtraCalls, 3)
	require.Equal(t, int64(31), repo.updateExtraCalls[0].id)
	require.Equal(t, AccountDaily5HTriggerStatusSkipped, repo.updateExtraCalls[0].updates[accountDaily5HLastStatusKey])
	require.Equal(t, "OpenAI Free account is excluded from the daily 5H trigger.", repo.updateExtraCalls[0].updates[accountDaily5HLastSummaryKey])
	require.Equal(t, AccountDaily5HTriggerStatusSuccess, repo.updateExtraCalls[1].updates[accountDaily5HLastStatusKey])
	require.Equal(t, AccountDaily5HTriggerStatusSuccess, repo.updateExtraCalls[2].updates[accountDaily5HLastStatusKey])
	snapshot := protocolruntime.Snapshot()
	require.EqualValues(t, 3, snapshot.RecoveryProbeResultByReason["daily_5h_trigger"])
	require.EqualValues(t, 2, snapshot.RecoveryProbeResultByStatus[AccountDaily5HTriggerStatusSuccess])
	require.EqualValues(t, 1, snapshot.RecoveryProbeResultByStatus[AccountDaily5HTriggerStatusSkipped])
}

func TestAccountDaily5HTriggerService_SelectModelForAccount_FixedModelMustStayVisible(t *testing.T) {
	account := &Account{
		ID:       15,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"manual_models": []any{
				map[string]any{"model_id": "gpt-5.4-mini", "provider": PlatformOpenAI},
			},
		},
	}
	settings := &AccountDaily5HTriggerSettings{
		Enabled:              true,
		SelectedAccountTypes: []string{AccountDaily5HTypeOpenAI},
		OpenAIModel: AccountDaily5HTriggerModelSettings{
			Mode:         AccountDaily5HModelModeFixed,
			FixedModelID: "gpt-5.4-mini",
		},
	}
	svc := NewAccountDaily5HTriggerService(nil, nil, nil, nil, time.Minute)

	modelID, skipReason, skipSummary := svc.selectModelForAccount(context.Background(), settings, account)
	require.Equal(t, "gpt-5.4-mini", modelID)
	require.Empty(t, skipReason)
	require.Empty(t, skipSummary)

	settings.OpenAIModel.FixedModelID = "gpt-5.5-mini"
	modelID, skipReason, skipSummary = svc.selectModelForAccount(context.Background(), settings, account)
	require.Equal(t, "", modelID)
	require.Equal(t, AccountDaily5HSkipReasonFixedModelHidden, skipReason)
	require.Equal(t, "The configured fixed model is no longer visible to this account.", skipSummary)
}

func TestAccountDaily5HTriggerService_ListCandidates_FiltersByFamilyAndCountsAccounts(t *testing.T) {
	accounts := []Account{
		{
			ID:             16,
			Platform:       PlatformOpenAI,
			Type:           AccountTypeOAuth,
			LifecycleState: AccountLifecycleNormal,
			Extra: map[string]any{
				"manual_models": []any{
					map[string]any{"model_id": "gpt-5.4-mini", "provider": PlatformOpenAI},
					map[string]any{"model_id": "gpt-4.1", "provider": PlatformOpenAI},
				},
			},
		},
		{
			ID:             17,
			Platform:       PlatformAnthropic,
			Type:           AccountTypeOAuth,
			LifecycleState: AccountLifecycleNormal,
			Extra: map[string]any{
				"manual_models": []any{
					map[string]any{"model_id": "claude-3.5-haiku", "provider": PlatformAnthropic},
					map[string]any{"model_id": "claude-sonnet-4-5", "provider": PlatformAnthropic},
				},
			},
		},
		{
			ID:             18,
			Platform:       PlatformGemini,
			Type:           AccountTypeOAuth,
			LifecycleState: AccountLifecycleNormal,
			Extra: map[string]any{
				"manual_models": []any{
					map[string]any{"model_id": "gemini-2.5-flash", "provider": PlatformGemini},
					map[string]any{"model_id": "text-embedding-004", "provider": PlatformGemini},
				},
			},
		},
		{
			ID:             19,
			Platform:       PlatformOpenAI,
			Type:           AccountTypeOAuth,
			LifecycleState: AccountLifecycleArchived,
			Extra: map[string]any{
				"manual_models": []any{
					map[string]any{"model_id": "gpt-5.5-mini", "provider": PlatformOpenAI},
				},
			},
		},
		{
			ID:             20,
			Platform:       PlatformOpenAI,
			Type:           AccountTypeOAuth,
			LifecycleState: AccountLifecycleBlacklisted,
			Extra: map[string]any{
				"manual_models": []any{
					map[string]any{"model_id": "gpt-5.6-mini", "provider": PlatformOpenAI},
				},
			},
		},
	}
	repo := &accountDaily5HRepoStub{accounts: accounts}
	svc := NewAccountDaily5HTriggerService(repo, nil, nil, nil, time.Minute)

	items := svc.ListDaily5HTriggerCandidates(context.Background())

	require.Len(t, items, 3)
	require.Equal(t, 1, items[0].Count)
	require.NotEmpty(t, items[0].Models)
	require.True(t, containsDaily5HModel(items[0].Models, "gpt-5.4-mini"))
	require.True(t, allDaily5HModelsContain(items[0].Models, "mini"))
	require.NotEmpty(t, items[1].Models)
	require.True(t, allDaily5HModelsContain(items[1].Models, "haiku"))
	require.NotEmpty(t, items[2].Models)
	require.True(t, allDaily5HModelsContain(items[2].Models, "gemini"))
}

func TestAccountDaily5HTriggerService_RunOnce_SkipsArchivedAndBlacklistedAccounts(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	now := time.Date(2026, 5, 8, 7, 15, 0, 0, time.UTC)
	accounts := []Account{
		{
			ID:             21,
			Platform:       PlatformOpenAI,
			Type:           AccountTypeOAuth,
			Status:         StatusActive,
			Schedulable:    true,
			LifecycleState: AccountLifecycleArchived,
			Extra: map[string]any{
				"manual_models": []any{
					map[string]any{"model_id": "gpt-5.4-mini", "provider": PlatformOpenAI},
				},
			},
		},
		{
			ID:             22,
			Platform:       PlatformOpenAI,
			Type:           AccountTypeOAuth,
			Status:         StatusActive,
			Schedulable:    true,
			LifecycleState: AccountLifecycleBlacklisted,
			Extra: map[string]any{
				"manual_models": []any{
					map[string]any{"model_id": "gpt-5.4-mini", "provider": PlatformOpenAI},
				},
			},
		},
	}
	repo := &accountDaily5HRepoStub{accounts: accounts}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	executor := &accountDaily5HExecutorStub{result: &BackgroundAccountTestResult{Status: "success"}}
	svc := NewAccountDaily5HTriggerService(repo, executor, settings, nil, time.Minute)
	svc.SetNow(func() time.Time { return now })
	svc.SetLocation(time.UTC)
	_, _ = settings.UpdateAccountDaily5HTriggerSettings(context.Background(), &AccountDaily5HTriggerSettings{
		Enabled:              true,
		SelectedAccountTypes: []string{AccountDaily5HTypeOpenAI},
	})

	svc.runOnce(context.Background())

	require.Empty(t, executor.calls)
	require.Empty(t, repo.updateExtraCalls)
}

func TestAccountDaily5HTriggerService_RunOnce_SkipsWhenFixedModelIsNoLongerVisible(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	now := time.Date(2026, 5, 8, 7, 20, 0, 0, time.UTC)
	account := Account{
		ID:             23,
		Platform:       PlatformOpenAI,
		Type:           AccountTypeOAuth,
		Status:         StatusActive,
		Schedulable:    true,
		LifecycleState: AccountLifecycleNormal,
		Extra: map[string]any{
			"manual_models": []any{
				map[string]any{"model_id": "gpt-5.4-mini", "provider": PlatformOpenAI},
			},
		},
	}
	repo := &accountDaily5HRepoStub{accounts: []Account{account}}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	executor := &accountDaily5HExecutorStub{result: &BackgroundAccountTestResult{Status: "success"}}
	svc := NewAccountDaily5HTriggerService(repo, executor, settings, nil, time.Minute)
	svc.SetNow(func() time.Time { return now })
	svc.SetLocation(time.UTC)
	_, _ = settings.UpdateAccountDaily5HTriggerSettings(context.Background(), &AccountDaily5HTriggerSettings{
		Enabled:              true,
		SelectedAccountTypes: []string{AccountDaily5HTypeOpenAI},
		OpenAIModel: AccountDaily5HTriggerModelSettings{
			Mode:         AccountDaily5HModelModeFixed,
			FixedModelID: "gpt-5.5-mini",
		},
	})

	svc.runOnce(context.Background())

	require.Empty(t, executor.calls)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, AccountDaily5HTriggerStatusSkipped, repo.updateExtraCalls[0].updates[accountDaily5HLastStatusKey])
	require.Equal(t, "The configured fixed model is no longer visible to this account.", repo.updateExtraCalls[0].updates[accountDaily5HLastSummaryKey])
}

func TestAccountDaily5HTriggerService_RunOnce_SkipsWhenNoFamilyModelIsAvailable(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	now := time.Date(2026, 5, 8, 7, 25, 0, 0, time.UTC)
	account := Account{
		ID:             24,
		Platform:       PlatformOpenAI,
		Type:           AccountTypeOAuth,
		Status:         StatusActive,
		Schedulable:    true,
		LifecycleState: AccountLifecycleNormal,
		Extra: map[string]any{
			"model_scope_v2": map[string]any{
				"policy_mode": AccountModelPolicyModeWhitelist,
				"entries": []any{
					map[string]any{
						"display_model_id": "friendly-standard",
						"target_model_id":  "gpt-5.4",
						"provider":         PlatformOpenAI,
						"visibility_mode":  AccountModelVisibilityModeAlias,
					},
				},
			},
		},
	}
	repo := &accountDaily5HRepoStub{accounts: []Account{account}}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	executor := &accountDaily5HExecutorStub{result: &BackgroundAccountTestResult{Status: "success"}}
	svc := NewAccountDaily5HTriggerService(repo, executor, settings, nil, time.Minute)
	svc.SetNow(func() time.Time { return now })
	svc.SetLocation(time.UTC)
	_, _ = settings.UpdateAccountDaily5HTriggerSettings(context.Background(), &AccountDaily5HTriggerSettings{
		Enabled:              true,
		SelectedAccountTypes: []string{AccountDaily5HTypeOpenAI},
	})

	svc.runOnce(context.Background())

	require.Empty(t, executor.calls)
	require.Len(t, repo.updateExtraCalls, 1)
	require.Equal(t, AccountDaily5HTriggerStatusSkipped, repo.updateExtraCalls[0].updates[accountDaily5HLastStatusKey])
	require.Equal(t, "No visible model in the required family is available for this account.", repo.updateExtraCalls[0].updates[accountDaily5HLastSummaryKey])
}

func containsDaily5HModel(items []AccountDaily5HTriggerModelOption, expected string) bool {
	for _, item := range items {
		if item.ModelID == expected {
			return true
		}
	}
	return false
}

func allDaily5HModelsContain(items []AccountDaily5HTriggerModelOption, needle string) bool {
	for _, item := range items {
		if !strings.Contains(strings.ToLower(item.ModelID), needle) {
			return false
		}
	}
	return true
}
