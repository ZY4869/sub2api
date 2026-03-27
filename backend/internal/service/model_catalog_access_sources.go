package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const modelCatalogAccessSourcePageSize = 200

func (s *ModelCatalogService) populateCatalogAccessSources(ctx context.Context, records map[string]*modelCatalogRecord) {
	if len(records) == 0 || s.adminService == nil {
		return
	}

	accounts, err := s.listAllActiveAccounts(ctx)
	if err != nil {
		logger.FromContext(ctx).Warn("model catalog: failed to load access sources", zap.Error(err))
		return
	}

	gateway := &GatewayService{}
	for _, record := range records {
		record.accessSources = collectModelCatalogAccessSources(ctx, gateway, record, accounts)
	}
}

func (s *ModelCatalogService) listAllActiveAccounts(ctx context.Context) ([]Account, error) {
	accounts := make([]Account, 0)
	page := 1

	for {
		items, total, err := s.adminService.ListAccounts(ctx, page, modelCatalogAccessSourcePageSize, "", "", StatusActive, "", 0, AccountLifecycleAll)
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		accounts = append(accounts, items...)
		if int64(len(accounts)) >= total {
			break
		}
		page++
	}

	return accounts, nil
}

func collectModelCatalogAccessSources(ctx context.Context, gateway *GatewayService, record *modelCatalogRecord, accounts []Account) []string {
	if record == nil || len(accounts) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, 2)
	for index := range accounts {
		account := &accounts[index]
		if !modelCatalogAccountCanServeRecord(ctx, gateway, record, account) {
			continue
		}
		source := modelCatalogAccessSourceForAccount(account)
		if source == "" {
			continue
		}
		seen[source] = struct{}{}
	}

	if len(seen) == 0 {
		return nil
	}

	ordered := make([]string, 0, len(seen))
	for _, source := range []string{ModelCatalogAccessSourceLogin, ModelCatalogAccessSourceKey} {
		if _, ok := seen[source]; ok {
			ordered = append(ordered, source)
		}
	}
	if len(ordered) == len(seen) {
		return ordered
	}

	extras := make([]string, 0, len(seen)-len(ordered))
	for source := range seen {
		if source == ModelCatalogAccessSourceLogin || source == ModelCatalogAccessSourceKey {
			continue
		}
		extras = append(extras, source)
	}
	sort.Strings(extras)
	return append(ordered, extras...)
}

func modelCatalogAccountCanServeRecord(ctx context.Context, gateway *GatewayService, record *modelCatalogRecord, account *Account) bool {
	if gateway == nil || record == nil || account == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(account.Status), StatusActive) {
		return false
	}
	for _, platform := range modelCatalogRequestPlatforms(record) {
		if modelCatalogAccountCanServePlatform(ctx, gateway, record, account, platform) {
			return true
		}
	}
	return false
}

func modelCatalogRequestPlatforms(record *modelCatalogRecord) []string {
	if record == nil {
		return nil
	}
	if len(record.defaultPlatforms) > 0 {
		return compactStrings(record.defaultPlatforms)
	}
	if modelCatalogLooksLikeSora(record) {
		return []string{PlatformSora}
	}
	provider := strings.TrimSpace(record.provider)
	if provider == "" {
		provider = inferModelProvider(record.model)
	}
	switch provider {
	case PlatformAnthropic, PlatformOpenAI, PlatformGemini, PlatformAntigravity, PlatformSora, PlatformGrok:
		return []string{provider}
	case "xai":
		return []string{PlatformGrok}
	default:
		return nil
	}
}

func modelCatalogAccountCanServePlatform(ctx context.Context, gateway *GatewayService, record *modelCatalogRecord, account *Account, platform string) bool {
	if gateway == nil || record == nil || account == nil {
		return false
	}
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return false
	}
	if !gateway.isAccountAllowedForPlatform(account, platform, modelCatalogUsesMixedScheduling(platform)) {
		return false
	}
	for _, candidate := range modelCatalogSupportCandidates(record) {
		if gateway.isModelSupportedByAccountWithContext(ctx, account, candidate) {
			return true
		}
	}
	return false
}

func modelCatalogUsesMixedScheduling(platform string) bool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case PlatformAnthropic, PlatformGemini:
		return true
	default:
		return false
	}
}

func modelCatalogSupportsPlatform(record *modelCatalogRecord, platform string) bool {
	if record == nil {
		return false
	}
	for _, current := range record.defaultPlatforms {
		if strings.EqualFold(strings.TrimSpace(current), platform) {
			return true
		}
	}
	return false
}

func modelCatalogLooksLikeSora(record *modelCatalogRecord) bool {
	if record == nil {
		return false
	}
	if modelCatalogSupportsPlatform(record, PlatformSora) {
		return true
	}
	if len(buildSoraModelAliases(record.model)) > 0 {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(record.mode)) {
	case "video", "prompt_enhance":
		return true
	default:
		return false
	}
}

func modelCatalogSupportCandidates(record *modelCatalogRecord) []string {
	if record == nil {
		return nil
	}

	seen := map[string]struct{}{}
	candidates := make([]string, 0, 7)
	appendCandidate := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		candidates = append(candidates, value)
	}

	appendCandidate(record.model)
	appendCandidate(strings.ReplaceAll(record.model, ".", "-"))
	appendCandidate(NormalizeModelCatalogModelID(record.model))
	appendCandidate(record.canonicalModelID)
	appendCandidate(record.pricingLookupModelID)
	appendCandidate(NormalizeModelCatalogModelID(record.canonicalModelID))
	appendCandidate(NormalizeModelCatalogModelID(record.pricingLookupModelID))
	return candidates
}

func modelCatalogAccessSourceForAccount(account *Account) string {
	if account == nil {
		return ""
	}
	switch account.Type {
	case AccountTypeOAuth, AccountTypeSetupToken, AccountTypeSSO:
		return ModelCatalogAccessSourceLogin
	case AccountTypeAPIKey, AccountTypeUpstream:
		return ModelCatalogAccessSourceKey
	default:
		return ""
	}
}
