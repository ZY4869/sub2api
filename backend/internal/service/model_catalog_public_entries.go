package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var publicCatalogSlugInvalidChars = regexp.MustCompile(`[^a-z0-9]+`)

func publicModelCatalogEntryID(accountID int64, protocol string, sourceModelID string) string {
	identity := strings.Join([]string{
		strconv.FormatInt(accountID, 10),
		strings.TrimSpace(strings.ToLower(protocol)),
		NormalizeModelCatalogModelID(sourceModelID),
	}, ":")
	sum := sha256.Sum256([]byte("acct:" + identity))
	return "acct_" + hex.EncodeToString(sum[:])[:16]
}

func publicModelCatalogSourceAlias(account *Account, group *Group, protocol string) string {
	if account != nil {
		if alias := strings.TrimSpace(account.GetExtraString("public_model_source_alias")); alias != "" {
			return alias
		}
		if account.ID > 0 {
			return publicModelCatalogDefaultSourceAlias(account.ID, protocol)
		}
	}
	if group != nil && strings.TrimSpace(group.Name) != "" {
		return publicModelCatalogDefaultSourceAlias(group.ID, protocol)
	}
	if protocol != "" {
		return publicModelCatalogDefaultSourceAlias(0, protocol)
	}
	return "default"
}

func publicModelCatalogDefaultSourceAlias(id int64, protocol string) string {
	identity := strings.TrimSpace(strings.ToLower(protocol))
	if id > 0 {
		identity = strings.Join([]string{identity, strconv.FormatInt(id, 10)}, ":")
	}
	if identity == "" {
		identity = "source"
	}
	sum := sha256.Sum256([]byte("source:" + identity))
	return "source-" + hex.EncodeToString(sum[:])[:8]
}

func publicModelCatalogSourceAliasSlug(alias string, accountID int64) string {
	normalized := strings.TrimSpace(strings.ToLower(alias))
	normalized = publicCatalogSlugInvalidChars.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" && accountID > 0 {
		normalized = "acct-" + strconv.FormatInt(accountID, 10)
	}
	if normalized == "" {
		return "source"
	}
	if len(normalized) > 32 {
		normalized = strings.Trim(normalized[:32], "-")
	}
	if normalized == "" {
		return "source"
	}
	return normalized
}

func defaultPublicModelCatalogPublicID(baseModel string, alias string, duplicate bool, accountID int64) string {
	base := NormalizeModelCatalogModelID(baseModel)
	if base == "" {
		base = strings.TrimSpace(baseModel)
	}
	if base == "" {
		return ""
	}
	if !duplicate {
		return base
	}
	return base + "@" + publicModelCatalogSourceAliasSlug(alias, accountID)
}

func (s *ModelCatalogService) buildPublicModelCatalogAccountEntryItems(
	ctx context.Context,
	records map[string]*modelCatalogRecord,
	pricingSnapshot *BillingPricingCatalogSnapshot,
	rules []BillingRule,
) ([]PublicModelCatalogItem, bool, error) {
	if s == nil || s.gatewayService == nil || s.gatewayService.groupRepo == nil || s.gatewayService.accountRepo == nil {
		return nil, false, nil
	}
	groups, err := s.gatewayService.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, true, err
	}

	type accountModelCandidate struct {
		group          *Group
		account        Account
		protocol       string
		sourceAlias    string
		availableModel AvailableTestModel
		item           PublicModelCatalogItem
	}
	candidates := make([]accountModelCandidate, 0)
	baseCounts := map[string]int{}
	seenEntryIDs := map[string]struct{}{}

	for i := range groups {
		group := groups[i]
		if !group.IsActive() || strings.TrimSpace(group.Platform) == "" {
			continue
		}
		queryPlatforms := QueryPlatformsForGroupPlatform(group.Platform, false)
		accounts, err := s.gatewayService.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, group.ID, queryPlatforms)
		if err != nil {
			return nil, true, err
		}
		for j := range accounts {
			account := accounts[j]
			if !account.IsSchedulable() {
				continue
			}
			for _, protocol := range groupProjectionPlatformsForAccount(group.Platform, &account) {
				accountForProjection := ResolveProtocolGatewayInboundAccount(&account, protocol)
				if accountForProjection == nil {
					continue
				}
				models := BuildAvailableTestModels(ctx, accountForProjection, s.modelRegistryService)
				models = filterAvailableTestModelsForPublishedCatalogAccount(ctx, s.gatewayService, group.ID, protocol, accountForProjection, models)
				for _, model := range models {
					sourceModelID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(model.TargetModelID, model.CanonicalID, model.ID))
					if sourceModelID == "" {
						continue
					}
					entryID := publicModelCatalogEntryID(account.ID, protocol, sourceModelID)
					if _, exists := seenEntryIDs[entryID]; exists {
						continue
					}
					item, ok := buildPublicModelCatalogItemForAccountModel(accountForProjection, model, records, pricingSnapshot, rules)
					if !ok {
						continue
					}
					seenEntryIDs[entryID] = struct{}{}
					sourceAlias := publicModelCatalogSourceAlias(accountForProjection, &group, protocol)
					item.EntryID = entryID
					item.SourceAccountID = account.ID
					item.SourceAccountName = strings.TrimSpace(account.Name)
					item.SourceProtocol = firstNonEmptyTrimmed(model.SourceProtocol, protocol)
					item.SourceAlias = sourceAlias
					item.SourceModelID = sourceModelID
					item.BaseModel = sourceModelID
					item.SourceIDs = uniqueTrimmedStringsPreserveCase(append(item.SourceIDs, sourceModelID, model.ID))
					item.RequestProtocols = uniqueTrimmedStringsPreserveCase(append(item.RequestProtocols, protocol))
					candidates = append(candidates, accountModelCandidate{
						group:          &group,
						account:        account,
						protocol:       protocol,
						sourceAlias:    sourceAlias,
						availableModel: model,
						item:           item,
					})
					baseCounts[sourceModelID]++
				}
			}
		}
	}

	items := make([]PublicModelCatalogItem, 0, len(candidates))
	usedPublicIDs := map[string]int{}
	for _, candidate := range candidates {
		item := candidate.item
		baseModel := firstNonEmptyTrimmed(item.BaseModel, item.SourceModelID, item.Model)
		item.PublicModelID = defaultPublicModelCatalogPublicID(baseModel, item.SourceAlias, baseCounts[baseModel] > 1, item.SourceAccountID)
		if item.PublicModelID == "" {
			continue
		}
		if used := usedPublicIDs[item.PublicModelID]; used > 0 {
			item.PublicModelID = item.PublicModelID + "-" + strconv.Itoa(used+1)
		}
		usedPublicIDs[item.PublicModelID]++
		item.Model = item.PublicModelID
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		leftName := strings.ToLower(strings.TrimSpace(firstNonEmptyTrimmed(items[i].DisplayName, items[i].BaseModel, items[i].Model)))
		rightName := strings.ToLower(strings.TrimSpace(firstNonEmptyTrimmed(items[j].DisplayName, items[j].BaseModel, items[j].Model)))
		if leftName != rightName {
			return leftName < rightName
		}
		if items[i].SourceAccountID != items[j].SourceAccountID {
			return items[i].SourceAccountID < items[j].SourceAccountID
		}
		return items[i].EntryID < items[j].EntryID
	})
	return items, true, nil
}

func filterAvailableTestModelsForPublishedCatalogAccount(
	ctx context.Context,
	gateway *GatewayService,
	groupID int64,
	protocol string,
	account *Account,
	models []AvailableTestModel,
) []AvailableTestModel {
	if len(models) == 0 {
		return nil
	}
	entries := make([]APIKeyPublicModelEntry, 0, len(models))
	for _, model := range models {
		publicID := NormalizeModelCatalogModelID(model.ID)
		if publicID == "" {
			continue
		}
		sourceID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(model.TargetModelID, model.CanonicalID, model.ID))
		entries = append(entries, APIKeyPublicModelEntry{
			PublicID:          publicID,
			AliasID:           publicID,
			SourceID:          sourceID,
			DisplayName:       model.DisplayName,
			Platform:          protocol,
			AvailabilityState: firstNonEmptyTrimmed(model.AvailabilityState, AccountModelAvailabilityUnknown),
			StaleState:        firstNonEmptyTrimmed(model.StaleState, AccountModelStaleStateUnverified),
			LifecycleStatus:   normalizePublicModelLifecycleStatus(model.Status, model.DisplayName, model.ID, sourceID),
		})
	}
	if gateway != nil {
		entries = gateway.filterPublicEntriesByActiveChannel(ctx, groupID, protocol, entries)
		entries = filterOpenAIAPIKeyPublicEntriesForRuntimeQuota(account, entries)
	}
	filtered := make([]AvailableTestModel, 0, len(entries))
	confirmedByPublicID := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if publicModelCatalogEntryConfirmedAvailable(entry) {
			confirmedByPublicID[entry.PublicID] = struct{}{}
		}
	}
	for _, model := range models {
		publicID := NormalizeModelCatalogModelID(model.ID)
		if _, ok := confirmedByPublicID[publicID]; ok {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

func publicModelCatalogEntryConfirmedAvailable(entry APIKeyPublicModelEntry) bool {
	return strings.EqualFold(entry.AvailabilityState, AccountModelAvailabilityVerified) &&
		strings.EqualFold(entry.StaleState, AccountModelStaleStateFresh)
}

func buildPublicModelCatalogItemForAccountModel(
	account *Account,
	model AvailableTestModel,
	records map[string]*modelCatalogRecord,
	pricingSnapshot *BillingPricingCatalogSnapshot,
	rules []BillingRule,
) (PublicModelCatalogItem, bool) {
	sourceModelID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(model.TargetModelID, model.CanonicalID, model.ID))
	if sourceModelID == "" {
		return PublicModelCatalogItem{}, false
	}
	projection := PublicModelProjectionEntry{
		PublicID:          NormalizeModelCatalogModelID(model.ID),
		DisplayName:       strings.TrimSpace(model.DisplayName),
		Platform:          firstNonEmptyTrimmed(model.SourceProtocol, RoutingPlatformForAccount(account)),
		AvailabilityState: firstNonEmptyTrimmed(model.AvailabilityState, AccountModelAvailabilityUnknown),
		StaleState:        firstNonEmptyTrimmed(model.StaleState, AccountModelStaleStateUnverified),
		LifecycleStatus:   normalizePublicModelLifecycleStatus(model.Status, model.DisplayName, model.ID, sourceModelID),
		SourceIDs:         []string{sourceModelID},
		AliasIDs:          []string{NormalizeModelCatalogModelID(model.ID)},
	}
	item, ok := buildPublicModelCatalogItemFromProjection(projection, records, pricingSnapshot, rules)
	if !ok {
		return PublicModelCatalogItem{}, false
	}
	item.BaseModel = sourceModelID
	item.SourceModelID = sourceModelID
	item.SourceProtocol = firstNonEmptyTrimmed(model.SourceProtocol, RoutingPlatformForAccount(account))
	if item.DisplayName == "" {
		item.DisplayName = firstNonEmptyTrimmed(model.DisplayName, FormatModelCatalogDisplayName(sourceModelID), sourceModelID)
	}
	if item.Mode == "" {
		item.Mode = firstNonEmptyTrimmed(model.Mode, inferModelMode(sourceModelID, ""))
	}
	if item.Provider == "" {
		item.Provider = NormalizeModelProvider(firstNonEmptyTrimmed(model.Provider, RoutingPlatformForAccount(account)))
		item.ProviderIconKey = item.Provider
	}
	return item, true
}
