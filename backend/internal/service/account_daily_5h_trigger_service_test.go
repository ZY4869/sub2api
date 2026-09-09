//go:build unit

package service

import (
	"context"
	"fmt"
	"sync"
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
	mu               sync.Mutex
	accounts         []Account
	updateExtraCalls []expiryRepoUpdateExtraCall
}

func (r *accountDaily5HRepoStub) Create(context.Context, *Account) error { panic("unexpected") }
func (r *accountDaily5HRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, account := range r.accounts {
		if account.ID == id {
			account.Extra = cloneStringAnyMap(account.Extra)
			return &account, nil
		}
	}
	return nil, fmt.Errorf("missing account %d", id)
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
func (r *accountDaily5HRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, _, _, _, _ string, _ int64, lifecycle, _ string) ([]Account, *pagination.PaginationResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	filtered := make([]Account, 0)
	for _, account := range r.accounts {
		if NormalizeAccountLifecycleInput(lifecycle) != AccountLifecycleAll && NormalizeAccountLifecycleInput(account.LifecycleState) != NormalizeAccountLifecycleInput(lifecycle) {
			continue
		}
		account.Extra = cloneStringAnyMap(account.Extra)
		filtered = append(filtered, account)
	}
	total := len(filtered)
	start := min(params.Offset(), total)
	end := min(start+params.Limit(), total)
	return filtered[start:end], &pagination.PaginationResult{Total: int64(total)}, nil
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
	r.mu.Lock()
	defer r.mu.Unlock()
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
	mu     sync.Mutex
	result *BackgroundAccountTestResult
	err    error
	calls  []ScheduledTestExecutionInput
	run    func(context.Context, ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error)
}

func (s *accountDaily5HExecutorStub) RunTestBackgroundDetailed(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error) {
	s.mu.Lock()
	s.calls = append(s.calls, input)
	s.mu.Unlock()
	if s.run != nil {
		return s.run(ctx, input)
	}
	return s.result, s.err
}

type daily5HLeaderStub struct{ calls, allowed int }

func (g *daily5HLeaderStub) RunIfLeader(ctx context.Context, _ string, _ time.Duration, run func(context.Context)) bool {
	g.calls++
	if g.calls > g.allowed {
		return false
	}
	run(ctx)
	return true
}

func TestAccountDaily5HTriggerService_BatchConcurrencyLeaderLossAndStop(t *testing.T) {
	accounts := make([]Account, 8)
	for i := range accounts {
		accounts[i] = daily5HAccount(int64(i+1), PlatformOpenAI, "gpt-5.4-mini")
	}
	svc, repo, executor, _ := daily5HFixture(t, accounts...)
	gate := &daily5HLeaderStub{allowed: 1}
	svc.SetLeaderGate(gate)
	started := make(chan int64, 8)
	release := make(chan struct{})
	executor.run = func(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error) {
		started <- input.AccountID
		select {
		case <-release:
			return &BackgroundAccountTestResult{Status: "success"}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	done := make(chan struct{})
	go func() { svc.runOnce(context.Background()); close(done) }()
	for i := 0; i < 4; i++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("four requests were not dispatched concurrently")
		}
	}
	select {
	case id := <-started:
		t.Fatalf("unexpected fifth concurrent account %d", id)
	default:
	}
	close(release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("batch did not finish")
	}
	require.Len(t, executor.calls, 4)
	for i, account := range repo.accounts {
		require.Equal(t, i < 4, daily5HSucceeded(account.Extra, "2026-05-08"))
	}
	require.Equal(t, 2, gate.calls)
	svc.Stop()
	gate.allowed = 100
	svc.runOnce(context.Background())
	require.Len(t, executor.calls, 4)
}

func daily5HAccount(id int64, platform string, modelIDs ...string) Account {
	entries := make([]any, 0, len(modelIDs))
	for _, model := range modelIDs {
		entries = append(entries, map[string]any{"display_model_id": model, "target_model_id": model, "provider": platform})
	}
	return Account{ID: id, Name: fmt.Sprint(id), Platform: platform, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, LifecycleState: AccountLifecycleNormal,
		Extra: map[string]any{"model_scope_v2": map[string]any{"policy_mode": "whitelist", "entries": entries}}}
}

func daily5HFixture(t *testing.T, accounts ...Account) (*AccountDaily5HTriggerService, *accountDaily5HRepoStub, *accountDaily5HExecutorStub, *time.Time) {
	t.Helper()
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	resetAccountModelProjectionCache()
	t.Cleanup(resetAccountModelProjectionCache)
	now := time.Date(2026, 5, 8, 7, 0, 0, 0, accountDaily5HLocation())
	repo := &accountDaily5HRepoStub{accounts: accounts}
	executor := &accountDaily5HExecutorStub{result: &BackgroundAccountTestResult{Status: "success"}}
	settings := NewSettingService(&settingRepoStub{values: map[string]string{}}, &config.Config{})
	svc := NewAccountDaily5HTriggerService(repo, executor, settings, nil, time.Minute)
	svc.SetNow(func() time.Time { return now })
	_, err := settings.UpdateAccountDaily5HTriggerSettings(context.Background(), &AccountDaily5HTriggerSettings{Enabled: true})
	require.NoError(t, err)
	return svc, repo, executor, &now
}

func TestAccountDaily5HTriggerService_FullPaginationAndCandidates(t *testing.T) {
	for _, count := range []int{0, 100, 101, 250} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			accounts := make([]Account, count)
			for i := range accounts {
				accounts[i] = daily5HAccount(int64(i+1), PlatformOpenAI, "gpt-5.4-mini")
			}
			svc, repo, executor, _ := daily5HFixture(t, accounts...)
			require.Equal(t, count, svc.ListDaily5HTriggerCandidates(context.Background())[0].Count)
			svc.runOnce(context.Background())
			require.Len(t, executor.calls, count)
			for _, account := range repo.accounts {
				require.True(t, daily5HSucceeded(account.Extra, "2026-05-08"), "account %d", account.ID)
			}
			svc.runOnce(context.Background())
			require.Len(t, executor.calls, count)
		})
	}
}

func TestAccountDaily5HTriggerService_TimeSettingBoundaryRestartAndDate(t *testing.T) {
	svc, repo, executor, now := daily5HFixture(t, daily5HAccount(1, PlatformOpenAI, "gpt-5.4-mini"))
	settings, _ := svc.settingService.GetAccountDaily5HTriggerSettings(context.Background())
	require.Equal(t, "07:00", settings.TriggerTime)
	settings.TriggerTime = "08:35"
	_, err := svc.settingService.UpdateAccountDaily5HTriggerSettings(context.Background(), settings)
	require.NoError(t, err)
	*now = time.Date(2026, 5, 8, 0, 34, 59, 0, time.UTC)
	svc.runOnce(context.Background())
	require.Empty(t, executor.calls)
	*now = now.Add(time.Second)
	svc.runOnce(context.Background())
	require.Len(t, executor.calls, 1)
	settings.TriggerTime = "07:00"
	_, err = svc.settingService.UpdateAccountDaily5HTriggerSettings(context.Background(), settings)
	require.NoError(t, err)
	restarted := NewAccountDaily5HTriggerService(repo, executor, svc.settingService, nil, time.Minute)
	restarted.SetNow(func() time.Time { return *now })
	restarted.runOnce(context.Background())
	require.Len(t, executor.calls, 1)
	*now = time.Date(2026, 5, 9, 0, 40, 0, 0, time.UTC)
	restarted.runOnce(context.Background())
	require.Len(t, executor.calls, 2)
	require.Equal(t, "2026-05-09", AccountDaily5HLastLocalDate(repo.accounts[0].Extra))
}

func TestAccountDaily5HTriggerSettings_ValidationAndLegacyDefault(t *testing.T) {
	svc, _, _, _ := daily5HFixture(t)
	for _, value := range []string{"7:00", "24:00", "07:60", "07:00:00", " 07:00", "bad"} {
		_, err := svc.settingService.UpdateAccountDaily5HTriggerSettings(context.Background(), &AccountDaily5HTriggerSettings{TriggerTime: value})
		require.Error(t, err, value)
	}
	for _, value := range []string{"00:00", "23:59", ""} {
		settings, err := svc.settingService.UpdateAccountDaily5HTriggerSettings(context.Background(), &AccountDaily5HTriggerSettings{TriggerTime: value})
		require.NoError(t, err)
		if value == "" {
			require.Equal(t, "07:00", settings.TriggerTime)
		}
	}
}

func TestAccountDaily5HTriggerService_RetryPersistenceAndLimit(t *testing.T) {
	svc, repo, executor, now := daily5HFixture(t, daily5HAccount(1, PlatformOpenAI, "gpt-5.4-mini"))
	executor.result = &BackgroundAccountTestResult{Status: "failed", ErrorMessage: "temporary upstream error"}
	svc.runOnce(context.Background())
	require.False(t, daily5HSucceeded(repo.accounts[0].Extra, "2026-05-08"))
	require.Equal(t, 1, daily5HAttempts(repo.accounts[0].Extra, "2026-05-08"))
	*now = now.Add(4 * time.Minute)
	svc.runOnce(context.Background())
	require.Len(t, executor.calls, 1)
	restarted := NewAccountDaily5HTriggerService(repo, executor, svc.settingService, nil, time.Minute)
	restarted.SetNow(func() time.Time { return *now })
	*now = now.Add(time.Minute)
	restarted.runOnce(context.Background())
	require.Len(t, executor.calls, 2)
	*now = now.Add(14 * time.Minute)
	restarted.runOnce(context.Background())
	require.Len(t, executor.calls, 2)
	*now = now.Add(time.Minute)
	restarted.runOnce(context.Background())
	require.Len(t, executor.calls, 3)
	*now = now.Add(time.Hour)
	restarted.runOnce(context.Background())
	require.Len(t, executor.calls, 3)
	require.Nil(t, repo.accounts[0].Extra[accountDaily5HNextRetryKey])
	*now = now.AddDate(0, 0, 1)
	executor.result = &BackgroundAccountTestResult{Status: "success"}
	restarted.runOnce(context.Background())
	require.Len(t, executor.calls, 4)
	require.True(t, daily5HSucceeded(repo.accounts[0].Extra, "2026-05-09"))
}

func TestAccountDaily5HTriggerService_WindowsAndPauseRecoverSameDay(t *testing.T) {
	for _, reason := range []string{"rate", "temp", "overload", "session", "paused"} {
		t.Run(reason, func(t *testing.T) {
			svc, repo, executor, now := daily5HFixture(t, daily5HAccount(1, PlatformOpenAI, "gpt-5.4-mini"))
			until := now.Add(time.Hour)
			switch reason {
			case "rate":
				repo.accounts[0].RateLimitResetAt = &until
			case "temp":
				repo.accounts[0].TempUnschedulableUntil = &until
			case "overload":
				repo.accounts[0].OverloadUntil = &until
			case "session":
				repo.accounts[0].SessionWindowEnd = &until
			case "paused":
				repo.accounts[0].Schedulable = false
			}
			svc.runOnce(context.Background())
			require.Empty(t, executor.calls)
			require.False(t, daily5HSucceeded(repo.accounts[0].Extra, "2026-05-08"))
			require.Equal(t, 0, daily5HAttempts(repo.accounts[0].Extra, "2026-05-08"))
			*now = until
			repo.accounts[0].Schedulable = true
			svc.runOnce(context.Background())
			require.Len(t, executor.calls, 1)
			require.True(t, daily5HSucceeded(repo.accounts[0].Extra, "2026-05-08"))
		})
	}
}

func TestAccountDaily5HTriggerService_AccountExclusions(t *testing.T) {
	for _, scenario := range []string{"archived", "blacklisted", "free", "type", "paused", "include_paused_window"} {
		t.Run(scenario, func(t *testing.T) {
			svc, repo, executor, now := daily5HFixture(t, daily5HAccount(1, PlatformOpenAI, "gpt-5.4-mini"))
			settings, _ := svc.settingService.GetAccountDaily5HTriggerSettings(context.Background())
			switch scenario {
			case "archived":
				repo.accounts[0].LifecycleState = AccountLifecycleArchived
			case "blacklisted":
				repo.accounts[0].LifecycleState = AccountLifecycleBlacklisted
			case "free":
				repo.accounts[0].Credentials = map[string]any{"plan_type": "free"}
				settings.IgnoreFreeAccounts = true
			case "type":
				settings.SelectedAccountTypes = []string{AccountDaily5HTypeAnthropic}
			case "paused":
				repo.accounts[0].Schedulable = false
			case "include_paused_window":
				settings.IncludePausedAccounts = true
				repo.accounts[0].Schedulable = false
				until := now.Add(time.Hour)
				repo.accounts[0].SessionWindowEnd = &until
			}
			_, err := svc.settingService.UpdateAccountDaily5HTriggerSettings(context.Background(), settings)
			require.NoError(t, err)
			svc.runOnce(context.Background())
			require.Empty(t, executor.calls)
			if scenario == "include_paused_window" {
				*now = now.Add(time.Hour)
				svc.runOnce(context.Background())
				require.Len(t, executor.calls, 1)
			}
		})
	}
}

func TestAccountDaily5HTriggerService_HolidaysSkipWithoutWritesAndNextWorkdayRuns(t *testing.T) {
	svc, repo, executor, now := daily5HFixture(t, daily5HAccount(1, PlatformOpenAI, "gpt-5.4-mini"))
	settings, _ := svc.settingService.GetAccountDaily5HTriggerSettings(context.Background())
	settings.SkipCNHolidaysAndWeekends = true
	_, err := svc.settingService.UpdateAccountDaily5HTriggerSettings(context.Background(), settings)
	require.NoError(t, err)
	*now = time.Date(2026, 10, 1, 8, 0, 0, 0, accountDaily5HLocation())
	svc.runOnce(context.Background())
	require.Empty(t, executor.calls)
	require.Empty(t, repo.updateExtraCalls)
	*now = time.Date(2026, 10, 8, 8, 0, 0, 0, accountDaily5HLocation())
	svc.runOnce(context.Background())
	require.Len(t, executor.calls, 1)
}

func TestAccountDaily5HTriggerService_FixedAndNoTextModelsRecover(t *testing.T) {
	svc, repo, executor, _ := daily5HFixture(t, daily5HAccount(1, PlatformOpenAI, "gpt-image-2"))
	svc.runOnce(context.Background())
	require.Empty(t, executor.calls)
	require.Equal(t, AccountDaily5HSkipReasonNoFamilyModel, repo.accounts[0].Extra[accountDaily5HLastSkipReasonKey])
	repo.accounts[0] = daily5HAccount(1, PlatformOpenAI, "gpt-5.4")
	settings, _ := svc.settingService.GetAccountDaily5HTriggerSettings(context.Background())
	settings.OpenAIModel = AccountDaily5HTriggerModelSettings{Mode: "fixed", FixedModelID: "gpt-5.4-mini"}
	_, err := svc.settingService.UpdateAccountDaily5HTriggerSettings(context.Background(), settings)
	require.NoError(t, err)
	svc.runOnce(context.Background())
	require.Empty(t, executor.calls)
	require.Equal(t, AccountDaily5HSkipReasonFixedModelHidden, repo.accounts[0].Extra[accountDaily5HLastSkipReasonKey])
	settings.OpenAIModel.FixedModelID = "gpt-5.4"
	_, err = svc.settingService.UpdateAccountDaily5HTriggerSettings(context.Background(), settings)
	require.NoError(t, err)
	svc.runOnce(context.Background())
	require.Len(t, executor.calls, 1)
	require.Equal(t, "gpt-5.4", executor.calls[0].ModelID)
}

func TestAccountDaily5HTriggerService_AliasFallbackAndKnownCost(t *testing.T) {
	svc, _, _, _ := daily5HFixture(t)
	svc.pricingService = &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"gpt-5.4-mini": {InputCostPerToken: 1, OutputCostPerToken: 2},
		"gpt-4.1-mini": {InputCostPerToken: 1, OutputCostPerToken: 1},
	}}
	models := []AvailableTestModel{{ID: "unknown-mini", Mode: "chat"}, {ID: "my-mini", TargetModelID: "gpt-4.1-mini", Mode: "chat"}, {ID: "gpt-5.4-mini", Mode: "chat"}}
	require.Equal(t, "my-mini", svc.pickDaily5HModel(AccountDaily5HTypeOpenAI, models))
	for _, tc := range []struct{ platform, model string }{{PlatformOpenAI, "gpt-5.4"}, {PlatformAnthropic, "claude-sonnet-4-6"}, {PlatformGemini, "gemini-2.5-pro"}} {
		account := daily5HAccount(1, tc.platform, tc.model)
		id, reason, _ := svc.selectModelForAccount(context.Background(), DefaultAccountDaily5HTriggerSettings(), &account)
		require.Empty(t, reason)
		require.Equal(t, tc.model, id)
	}
	require.Empty(t, accountDaily5HTextModels([]AvailableTestModel{{ID: "gemini-3.1-flash-tts-preview"}, {ID: "gpt-image-2"}, {ID: "text-embedding-3-small"}}))
}

func TestAccountDaily5HTriggerService_LegacyFailureAndTerminalReauth(t *testing.T) {
	svc, repo, executor, now := daily5HFixture(t, daily5HAccount(1, PlatformOpenAI, "gpt-5.4-mini"))
	repo.accounts[0].Extra[accountDaily5HLastLocalDateKey] = "2026-05-08"
	repo.accounts[0].Extra[accountDaily5HLastStatusKey] = "failed"
	executor.result = &BackgroundAccountTestResult{Status: "failed", NeedsReauth: true}
	svc.runOnce(context.Background())
	require.Len(t, executor.calls, 1)
	*now = now.Add(time.Hour)
	svc.runOnce(context.Background())
	require.Len(t, executor.calls, 1)
	require.Equal(t, true, repo.accounts[0].Extra[accountDaily5HStoppedKey])
}
