package service

import (
	"context"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
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
		location:             time.Local,
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

func (s *AccountDaily5HTriggerService) runOnce(ctx context.Context) {
	settings, err := s.settingService.GetAccountDaily5HTriggerSettings(ctx)
	if err != nil || settings == nil || !settings.Enabled {
		return
	}
	now := s.now().In(s.location)
	if now.Hour() < defaultAccountDaily5HTriggerHour {
		return
	}
	localDate := now.Format("2006-01-02")
	accounts, err := s.listManagedAccounts(ctx)
	if err != nil {
		slog.Warn("account_daily_5h_trigger_list_failed", "error", err)
		return
	}
	for index := range accounts {
		account := accounts[index]
		if AccountDaily5HLastLocalDate(account.Extra) == localDate {
			continue
		}
		shouldRun, skipReason, skipSummary := s.shouldRunForAccount(settings, &account, now.UTC())
		requestID := firstNonEmptyString(requestIDFromContext(ctx), "generated:"+generateRequestID())
		if !shouldRun {
			_ = s.accountRepo.UpdateExtra(ctx, account.ID, BuildAccountDaily5HTriggerExtra(localDate, AccountDaily5HTriggerStatusSkipped, "", skipSummary))
			slog.Info(
				"account_daily_5h_trigger_skipped",
				"request_id", requestID,
				"account_id", account.ID,
				"account_type", accountDaily5HAccountType(&account),
				"local_date", localDate,
				"skip_reason", skipReason,
				"summary", skipSummary,
			)
			protocolruntime.RecordRecoveryProbeResult("daily_5h_trigger", AccountDaily5HTriggerStatusSkipped, 0)
			continue
		}
		modelID, modelSkipReason, modelSkipSummary := s.selectModelForAccount(ctx, settings, &account)
		if strings.TrimSpace(modelID) == "" {
			_ = s.accountRepo.UpdateExtra(ctx, account.ID, BuildAccountDaily5HTriggerExtra(localDate, AccountDaily5HTriggerStatusSkipped, "", modelSkipSummary))
			slog.Info(
				"account_daily_5h_trigger_skipped",
				"request_id", requestID,
				"account_id", account.ID,
				"account_type", accountDaily5HAccountType(&account),
				"local_date", localDate,
				"skip_reason", modelSkipReason,
				"summary", modelSkipSummary,
			)
			protocolruntime.RecordRecoveryProbeResult("daily_5h_trigger", AccountDaily5HTriggerStatusSkipped, 0)
			continue
		}
		startedAt := time.Now()
		triggerCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		triggerCtx = EnsureRequestMetadata(triggerCtx)
		SetProbeActionMetadata(triggerCtx, "daily_5h_trigger")
		protocolruntime.RecordRecoveryProbeStarted("daily_5h_trigger")
		requestID = firstNonEmptyString(requestIDFromContext(triggerCtx), requestID)
		slog.Info(
			"account_daily_5h_trigger_started",
			"request_id", requestID,
			"account_id", account.ID,
			"account_type", accountDaily5HAccountType(&account),
			"model_id", modelID,
			"local_date", localDate,
		)
		result, runErr := s.accountTestRunner.RunTestBackgroundDetailed(triggerCtx, ScheduledTestExecutionInput{
			AccountID:     account.ID,
			ModelID:       modelID,
			Prompt:        accountDaily5HPrompt,
			TestMode:      "real_forward",
			OperationType: UsageOperationTypeScheduledTest,
		})
		cancel()
		status := AccountDaily5HTriggerStatusSuccess
		summary := "Daily 5H trigger succeeded."
		if runErr != nil || result == nil || !strings.EqualFold(strings.TrimSpace(result.Status), "success") {
			status = AccountDaily5HTriggerStatusFailed
			summary = firstNonEmptyString(strings.TrimSpace(runErrString(runErr)), strings.TrimSpace(resultErrorMessage(result)), "Daily 5H trigger failed.")
		}
		switch status {
		case AccountDaily5HTriggerStatusSuccess:
			protocolruntime.RecordRecoveryProbeSuccess("daily_5h_trigger")
		default:
			protocolruntime.RecordRecoveryProbeRetry("daily_5h_trigger")
		}
		protocolruntime.RecordRecoveryProbeResult("daily_5h_trigger", status, time.Since(startedAt).Milliseconds())
		if updateErr := s.accountRepo.UpdateExtra(ctx, account.ID, BuildAccountDaily5HTriggerExtra(localDate, status, modelID, summary)); updateErr != nil {
			slog.Warn(
				"account_daily_5h_trigger_update_failed",
				"request_id", requestID,
				"account_id", account.ID,
				"account_type", accountDaily5HAccountType(&account),
				"model_id", modelID,
				"local_date", localDate,
				"error", updateErr,
			)
		}
		eventName := "account_daily_5h_trigger_success"
		if status != AccountDaily5HTriggerStatusSuccess {
			eventName = "account_daily_5h_trigger_failed"
		}
		slog.Info(
			eventName,
			"request_id", requestID,
			"account_id", account.ID,
			"account_type", accountDaily5HAccountType(&account),
			"status", status,
			"model_id", modelID,
			"local_date", localDate,
			"summary", summary,
		)
	}
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
		for _, model := range filterAccountDaily5HFamilyModels(typeKey, BuildAvailableTestModels(ctx, &account, s.modelRegistryService)) {
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
	models := filterAccountDaily5HFamilyModels(accountDaily5HAccountType(account), BuildAvailableTestModels(ctx, account, s.modelRegistryService))
	if len(models) == 0 {
		return "", AccountDaily5HSkipReasonNoFamilyModel, "No visible model in the required family is available for this account."
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
	return pickLatestAvailableTestModelID(models), "", ""
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

func filterAccountDaily5HFamilyModels(typeKey string, models []AvailableTestModel) []AvailableTestModel {
	out := make([]AvailableTestModel, 0, len(models))
	for _, model := range models {
		id := strings.ToLower(strings.TrimSpace(model.ID))
		switch typeKey {
		case AccountDaily5HTypeOpenAI:
			if strings.Contains(id, "mini") {
				out = append(out, model)
			}
		case AccountDaily5HTypeAnthropic:
			if strings.Contains(id, "haiku") {
				out = append(out, model)
			}
		case AccountDaily5HTypeGemini:
			if strings.Contains(id, "gemini") {
				out = append(out, model)
			}
		}
	}
	return out
}

func pickLatestAvailableTestModelID(models []AvailableTestModel) string {
	if len(models) == 0 {
		return ""
	}
	best := ""
	bestMajor := -1
	bestMinor := -1
	for _, model := range models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		major, minor := parseAccountDaily5HModelVersion(id)
		if major > bestMajor || (major == bestMajor && minor > bestMinor) {
			best = id
			bestMajor = major
			bestMinor = minor
		}
	}
	if best != "" {
		return best
	}
	return models[0].ID
}

func parseAccountDaily5HModelVersion(id string) (int, int) {
	major := -1
	minor := -1
	normalized := strings.TrimSpace(strings.ToLower(id))
	for _, token := range strings.FieldsFunc(normalized, func(r rune) bool {
		return (r < '0' || r > '9') && r != '.'
	}) {
		if token == "" {
			continue
		}
		parts := strings.SplitN(token, ".", 3)
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		parsedMajor, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		parsedMinor := 0
		if len(parts) > 1 && parts[1] != "" {
			if value, err := strconv.Atoi(parts[1]); err == nil {
				parsedMinor = value
			}
		}
		major = parsedMajor
		minor = parsedMinor
		break
	}
	return major, minor
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
	params := pagination.PaginationParams{Page: 1, PageSize: 10000}
	accounts, _, err := s.accountRepo.ListWithFilters(ctx, params, "", "", "", "", 0, AccountLifecycleNormal, "")
	if err != nil {
		return nil, err
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
