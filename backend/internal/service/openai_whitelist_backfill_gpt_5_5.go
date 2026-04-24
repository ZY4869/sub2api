package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	openAIGPT55LegacyBaseWhitelist = []string{"gpt-5.2", "gpt-5.4", "gpt-5.4-mini"}
	openAIGPT55LegacyProWhitelist  = []string{"gpt-5.2", "gpt-5.4", "gpt-5.4-mini", "gpt-5.3-codex-spark"}
	openAIGPT55PaidWhitelist       = []string{"gpt-5.2", "gpt-5.4", "gpt-5.4-mini", "gpt-5.5"}
	openAIGPT55ProWhitelist        = []string{"gpt-5.2", "gpt-5.4", "gpt-5.4-mini", "gpt-5.5", "gpt-5.3-codex-spark"}
)

type OpenAIGPT55WhitelistBackfillRepository interface {
	ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, lifecycle string, privacyMode string) ([]Account, *pagination.PaginationResult, error)
	UpdateExtra(ctx context.Context, id int64, updates map[string]any) error
}

type OpenAIGPT55WhitelistBackfillResult struct {
	Scanned int `json:"scanned"`
	Updated int `json:"updated"`
}

func BackfillOpenAIGPT55DefaultWhitelists(ctx context.Context, repo OpenAIGPT55WhitelistBackfillRepository, pageSize int) (*OpenAIGPT55WhitelistBackfillResult, error) {
	if repo == nil {
		return &OpenAIGPT55WhitelistBackfillResult{}, nil
	}
	if pageSize <= 0 {
		pageSize = 100
	}

	result := &OpenAIGPT55WhitelistBackfillResult{}
	for page := 1; ; page++ {
		accounts, paginationResult, err := repo.ListWithFilters(
			ctx,
			pagination.PaginationParams{Page: page, PageSize: pageSize},
			PlatformOpenAI,
			"",
			"",
			"",
			0,
			AccountLifecycleAll,
			"",
		)
		if err != nil {
			return nil, err
		}
		if len(accounts) == 0 {
			return result, nil
		}

		for i := range accounts {
			account := &accounts[i]
			result.Scanned++

			updates := buildOpenAIGPT55WhitelistBackfillUpdates(account)
			if len(updates) == 0 {
				continue
			}
			if err := repo.UpdateExtra(ctx, account.ID, updates); err != nil {
				return nil, err
			}
			result.Updated++
		}

		if paginationResult == nil || paginationResult.Pages <= page {
			return result, nil
		}
	}
}

func buildOpenAIGPT55WhitelistBackfillUpdates(account *Account) map[string]any {
	if account == nil || account.Platform != PlatformOpenAI {
		return nil
	}
	if normalizeOpenAIPlanType(account.GetCredential("plan_type")) == "free" {
		return nil
	}

	scope, ok := ExtractAccountModelScopeV2(account.Extra)
	if !ok || scope == nil {
		return nil
	}
	if scope.PolicyMode != AccountModelPolicyModeWhitelist {
		return nil
	}

	currentIDs, ok := extractDirectWhitelistModelIDs(scope)
	if !ok || len(currentIDs) == 0 {
		return nil
	}

	switch {
	case stringSetEqual(currentIDs, openAIGPT55LegacyBaseWhitelist):
		return map[string]any{
			"model_scope_v2": buildOpenAIWhitelistScopeMap(openAIGPT55PaidWhitelist),
		}
	case stringSetEqual(currentIDs, openAIGPT55LegacyProWhitelist):
		return map[string]any{
			"model_scope_v2": buildOpenAIWhitelistScopeMap(openAIGPT55ProWhitelist),
		}
	default:
		return nil
	}
}

func extractDirectWhitelistModelIDs(scope *AccountModelScopeV2) ([]string, bool) {
	if scope == nil {
		return nil, false
	}
	if len(scope.Entries) == 0 {
		return nil, false
	}

	set := map[string]struct{}{}
	ids := make([]string, 0, len(scope.Entries))
	for _, entry := range scope.Entries {
		displayModelID := normalizeRegistryID(entry.DisplayModelID)
		targetModelID := normalizeRegistryID(firstNonEmptyString(entry.TargetModelID, entry.DisplayModelID))
		if displayModelID == "" || targetModelID == "" {
			continue
		}
		if displayModelID != targetModelID {
			return nil, false
		}
		if _, exists := set[displayModelID]; exists {
			continue
		}
		set[displayModelID] = struct{}{}
		ids = append(ids, displayModelID)
	}
	sort.Strings(ids)
	return ids, true
}

func buildOpenAIWhitelistScopeMap(modelIDs []string) map[string]any {
	entries := make([]AccountModelScopeEntry, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		entries = append(entries, AccountModelScopeEntry{
			DisplayModelID: modelID,
			TargetModelID:  modelID,
			Provider:       PlatformOpenAI,
			SourceProtocol: PlatformOpenAI,
			VisibilityMode: AccountModelVisibilityModeDirect,
		})
	}
	scope := &AccountModelScopeV2{
		PolicyMode: AccountModelPolicyModeWhitelist,
		Entries:    entries,
	}
	return scope.ToMap()
}

func stringSetEqual(left []string, right []string) bool {
	leftNormalized := normalizeStringSetForComparison(left)
	rightNormalized := normalizeStringSetForComparison(right)
	if len(leftNormalized) != len(rightNormalized) {
		return false
	}
	for i := range leftNormalized {
		if leftNormalized[i] != rightNormalized[i] {
			return false
		}
	}
	return true
}

func normalizeStringSetForComparison(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" {
			continue
		}
		if _, ok := set[value]; ok {
			continue
		}
		set[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}

type OpenAIGPT55WhitelistBackfillService struct {
	settingRepo SettingRepository
	accountRepo AccountRepository

	cancel context.CancelFunc

	stopOnce sync.Once
	wg       sync.WaitGroup
}

func NewOpenAIGPT55WhitelistBackfillService(settingRepo SettingRepository, accountRepo AccountRepository) *OpenAIGPT55WhitelistBackfillService {
	return &OpenAIGPT55WhitelistBackfillService{
		settingRepo: settingRepo,
		accountRepo: accountRepo,
	}
}

func (s *OpenAIGPT55WhitelistBackfillService) Start() {
	if s == nil || s.settingRepo == nil || s.accountRepo == nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 5*time.Minute)
		defer timeoutCancel()

		marker, err := s.settingRepo.GetValue(timeoutCtx, SettingKeyOpenAIBackfillGPT55Done)
		if err == nil && strings.EqualFold(strings.TrimSpace(marker), "true") {
			return
		}

		start := time.Now()
		result, err := BackfillOpenAIGPT55DefaultWhitelists(timeoutCtx, s.accountRepo, 200)
		if err != nil {
			slog.Warn("openai_whitelist_backfill_gpt_5_5_failed", "error", err)
			return
		}
		if err := s.settingRepo.Set(timeoutCtx, SettingKeyOpenAIBackfillGPT55Done, "true"); err != nil {
			slog.Warn("openai_whitelist_backfill_gpt_5_5_marker_write_failed", "error", err)
			return
		}
		slog.Info(
			"openai_whitelist_backfill_gpt_5_5_completed",
			"scanned", result.Scanned,
			"updated", result.Updated,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}()
}

func (s *OpenAIGPT55WhitelistBackfillService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
	})
	s.wg.Wait()
}
