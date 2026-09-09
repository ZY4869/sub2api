package service

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const accountDaily5HTriggerJobName = "account_daily_5h_trigger"

type AccountDaily5HTriggerService struct {
	accountRepo       AccountRepository
	accountTestRunner interface {
		RunTestBackgroundDetailed(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error)
	}
	settingService       *SettingService
	modelRegistryService *ModelRegistryService
	leaderGate           PeriodicJobLeaderGate
	interval             time.Duration
	pricingService       *PricingService
	runMu                sync.Mutex
	now                  func() time.Time
	location             *time.Location
	stopCh               chan struct{}
	stopOnce             sync.Once
	wg                   sync.WaitGroup
}

func NewAccountDaily5HTriggerService(
	accountRepo AccountRepository,
	accountTestRunner interface {
		RunTestBackgroundDetailed(ctx context.Context, input ScheduledTestExecutionInput) (*BackgroundAccountTestResult, error)
	},
	settingService *SettingService,
	modelRegistryService *ModelRegistryService,
	interval time.Duration,
) *AccountDaily5HTriggerService {
	if interval <= 0 {
		interval = time.Minute
	}
	return &AccountDaily5HTriggerService{
		accountRepo:          accountRepo,
		accountTestRunner:    accountTestRunner,
		settingService:       settingService,
		modelRegistryService: modelRegistryService,
		interval:             interval,
		now:                  time.Now,
		location:             accountDaily5HLocation(),
		stopCh:               make(chan struct{}),
	}
}

func (s *AccountDaily5HTriggerService) SetLeaderGate(gate PeriodicJobLeaderGate) {
	if s == nil {
		return
	}
	s.leaderGate = gate
}

func (s *AccountDaily5HTriggerService) Start() {
	if s == nil || s.accountRepo == nil || s.accountTestRunner == nil || s.settingService == nil {
		return
	}
	s.settingService.SetAccountDaily5HTriggerCandidateProvider(s)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.runLeaderOnce(context.Background())
		for {
			select {
			case <-ticker.C:
				s.runLeaderOnce(context.Background())
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *AccountDaily5HTriggerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *AccountDaily5HTriggerService) runLeaderOnce(ctx context.Context) bool {
	if s == nil {
		return false
	}
	if s.leaderGate == nil {
		s.runOnce(ctx)
		return true
	}
	return s.leaderGate.RunIfLeader(ctx, accountDaily5HTriggerJobName, periodicJobLeaderTTL(s.interval), s.runOnce)
}

func (s *AccountDaily5HTriggerService) ListDaily5HTriggerCandidates(ctx context.Context) []AccountDaily5HTriggerAccountTypeSummary {
	if s == nil || s.accountRepo == nil {
		return []AccountDaily5HTriggerAccountTypeSummary{}
	}
	accounts, err := s.listManagedAccounts(ctx)
	if err != nil {
		return []AccountDaily5HTriggerAccountTypeSummary{}
	}
	return s.buildCandidates(ctx, accounts)
}

func (s *AccountDaily5HTriggerService) shouldRunForAccount(settings *AccountDaily5HTriggerSettings, account *Account, now time.Time) (bool, string, string) {
	if account == nil || settings == nil {
		return false, AccountDaily5HSkipReasonLifecycleExcluded, "Account is not available for the daily 5H trigger."
	}
	if !IsManagedRuntimeAccount(account) {
		return false, AccountDaily5HSkipReasonLifecycleExcluded, "Account is outside the managed runtime lifecycle and is skipped."
	}
	if !containsAccountDaily5HType(settings.SelectedAccountTypes, accountDaily5HAccountType(account)) {
		return false, AccountDaily5HSkipReasonAccountType, "Account type is not selected for the daily 5H trigger."
	}
	if settings.IgnoreFreeAccounts && isAccountDaily5HOpenAIFree(account) {
		return false, AccountDaily5HSkipReasonFreeExcluded, "OpenAI Free account is excluded from the daily 5H trigger."
	}
	if account.RateLimitResetAt != nil && now.Before(account.RateLimitResetAt.UTC()) {
		return false, AccountDaily5HSkipReasonRateLimited, "Account is still rate-limited and is skipped for the daily 5H trigger."
	}
	if account.TempUnschedulableUntil != nil && now.Before(account.TempUnschedulableUntil.UTC()) {
		return false, AccountDaily5HSkipReasonTempUnsched, "Account is temporarily unschedulable and is skipped for the daily 5H trigger."
	}
	if account.OverloadUntil != nil && now.Before(account.OverloadUntil.UTC()) {
		return false, AccountDaily5HSkipReasonOverloaded, "Account is overloaded and is skipped for the daily 5H trigger."
	}
	if account.SessionWindowEnd != nil && now.Before(account.SessionWindowEnd.UTC()) {
		return false, AccountDaily5HSkipReasonSessionWindow, "Account is inside an active session window and is skipped for the daily 5H trigger."
	}
	if !settings.IncludePausedAccounts && (!account.Schedulable || NormalizeAdminAccountStatusInput(account.Status) != StatusActive) {
		return false, AccountDaily5HSkipReasonPausedExcluded, "Paused accounts are excluded from the daily 5H trigger."
	}
	return true, "", ""
}

func isAccountDaily5HOpenAIFree(account *Account) bool {
	return accountDaily5HAccountType(account) == AccountDaily5HTypeOpenAI &&
		normalizeOpenAIPlanType(account.GetCredential("plan_type")) == "free"
}

func (s *AccountDaily5HTriggerService) SetNow(now func() time.Time) {
	if s == nil || now == nil {
		return
	}
	s.now = now
}

func (s *AccountDaily5HTriggerService) SetLocation(location *time.Location) {
	if s == nil || location == nil {
		return
	}
	s.location = location
}

func (s *AccountDaily5HTriggerService) buildCandidates(ctx context.Context, accounts []Account) []AccountDaily5HTriggerAccountTypeSummary {
	type bucket struct {
		count  int
		models map[string]*AccountDaily5HTriggerModelOption
	}
	buckets := map[string]*bucket{
		AccountDaily5HTypeOpenAI:    {models: map[string]*AccountDaily5HTriggerModelOption{}},
		AccountDaily5HTypeAnthropic: {models: map[string]*AccountDaily5HTriggerModelOption{}},
		AccountDaily5HTypeGemini:    {models: map[string]*AccountDaily5HTriggerModelOption{}},
	}
	for index := range accounts {
		account := accounts[index]
		if !IsManagedRuntimeAccount(&account) {
			continue
		}
		typeKey := accountDaily5HAccountType(&account)
		if typeKey == "" {
			continue
		}
		current := buckets[typeKey]
		current.count++
		for _, model := range accountDaily5HTextModels(BuildAvailableTestModels(ctx, &account, s.modelRegistryService)) {
			item, ok := current.models[model.ID]
			if !ok {
				current.models[model.ID] = &AccountDaily5HTriggerModelOption{
					ModelID:       model.ID,
					DisplayName:   firstNonEmptyTestModelLabel(model.DisplayName, model.ID),
					Provider:      model.Provider,
					ProviderLabel: model.ProviderLabel,
					AccountCount:  1,
				}
				continue
			}
			item.AccountCount++
		}
	}
	out := make([]AccountDaily5HTriggerAccountTypeSummary, 0, len(buckets))
	for _, typeKey := range []string{AccountDaily5HTypeOpenAI, AccountDaily5HTypeAnthropic, AccountDaily5HTypeGemini} {
		current := buckets[typeKey]
		models := make([]AccountDaily5HTriggerModelOption, 0, len(current.models))
		for _, item := range current.models {
			models = append(models, *item)
		}
		sort.SliceStable(models, func(i, j int) bool {
			return strings.ToLower(models[i].ModelID) < strings.ToLower(models[j].ModelID)
		})
		out = append(out, AccountDaily5HTriggerAccountTypeSummary{
			AccountType: typeKey,
			Count:       current.count,
			Models:      models,
		})
	}
	return out
}

func (s *AccountDaily5HTriggerService) selectModelForAccount(ctx context.Context, settings *AccountDaily5HTriggerSettings, account *Account) (string, string, string) {
	models := accountDaily5HTextModels(BuildAvailableTestModels(ctx, account, s.modelRegistryService))
	if len(models) == 0 {
		return "", AccountDaily5HSkipReasonNoFamilyModel, "No callable text model is visible to this account."
	}
	config := accountDaily5HModelSettingsForAccount(settings, account)
	if config.Mode == AccountDaily5HModelModeFixed {
		for _, model := range models {
			if strings.EqualFold(model.ID, config.FixedModelID) {
				return model.ID, "", ""
			}
		}
		return "", AccountDaily5HSkipReasonFixedModelHidden, "The configured fixed model is no longer visible to this account."
	}
	return s.pickDaily5HModel(accountDaily5HAccountType(account), models), "", ""
}

func accountDaily5HModelSettingsForAccount(settings *AccountDaily5HTriggerSettings, account *Account) AccountDaily5HTriggerModelSettings {
	if settings == nil || account == nil {
		return AccountDaily5HTriggerModelSettings{Mode: AccountDaily5HModelModeAuto}
	}
	switch accountDaily5HAccountType(account) {
	case AccountDaily5HTypeOpenAI:
		return settings.OpenAIModel
	case AccountDaily5HTypeAnthropic:
		return settings.AnthropicModel
	case AccountDaily5HTypeGemini:
		return settings.GeminiModel
	default:
		return AccountDaily5HTriggerModelSettings{Mode: AccountDaily5HModelModeAuto}
	}
}

func containsAccountDaily5HType(items []string, expected string) bool {
	expected = strings.TrimSpace(strings.ToLower(expected))
	for _, item := range items {
		if strings.TrimSpace(strings.ToLower(item)) == expected {
			return true
		}
	}
	return false
}

func (s *AccountDaily5HTriggerService) listManagedAccounts(ctx context.Context) ([]Account, error) {
	if s == nil || s.accountRepo == nil {
		return []Account{}, nil
	}
	accounts := make([]Account, 0)
	seen := make(map[int64]bool)
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		params := pagination.PaginationParams{Page: page, PageSize: 100}
		batch, result, err := s.accountRepo.ListWithFilters(ctx, params, "", "", "", "", 0, AccountLifecycleNormal, "")
		if err != nil {
			return nil, err
		}
		for _, account := range batch {
			if !seen[account.ID] {
				accounts = append(accounts, account)
				seen[account.ID] = true
			}
		}
		if len(batch) < params.PageSize || (result != nil && int64(page*params.PageSize) >= result.Total) {
			break
		}
	}
	return accounts, nil
}

func runErrString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func resultErrorMessage(result *BackgroundAccountTestResult) string {
	if result == nil {
		return ""
	}
	return result.ErrorMessage
}
