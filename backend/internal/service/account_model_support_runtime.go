package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"golang.org/x/sync/singleflight"
)

const (
	accountModelSupportCacheTTL           = 10 * time.Minute
	accountModelSupportRegistryVersionTTL = 5 * time.Second
)

type accountModelSupportSet struct {
	explicitRestrictions bool
	allowedVariants      map[string]struct{}
}

type accountModelSupportCacheEntry struct {
	supportSet *accountModelSupportSet
	expiresAt  time.Time
}

type accountModelSupportRegistryVersionEntry struct {
	version   string
	expiresAt time.Time
}

var (
	accountModelSupportCacheMu         sync.RWMutex
	accountModelSupportCache           = map[string]accountModelSupportCacheEntry{}
	accountModelSupportCacheFlight     singleflight.Group
	accountModelSupportRegistryVersion sync.Map
)

func resolveCanonicalRequestModelWithRegistry(ctx context.Context, registry *ModelRegistryService, requestedModel string) string {
	requestedModel = strings.TrimSpace(requestedModel)
	if requestedModel == "" {
		return ""
	}
	if registry != nil {
		if resolved, ok, err := registry.ResolveModel(ctx, requestedModel); err == nil && ok && resolved != "" {
			return resolved
		}
	}
	if resolved, ok := modelregistry.ResolveToCanonicalID(requestedModel); ok {
		return resolved
	}
	return normalizeRegistryID(requestedModel)
}

func resolveUpstreamModelIDWithRegistry(ctx context.Context, registry *ModelRegistryService, account *Account, requestedModel string) string {
	requestedModel = resolveCanonicalRequestModelWithRegistry(ctx, registry, requestedModel)
	if requestedModel == "" || account == nil {
		return requestedModel
	}
	route := registryRouteForAccount(account)
	if registry != nil {
		if resolved, ok, err := registry.ResolveProtocolModel(ctx, requestedModel, route); err == nil && ok && resolved != "" {
			return resolved
		}
	}
	if resolved, ok := modelregistry.ResolveToProtocolID(requestedModel, route); ok {
		return resolved
	}
	return requestedModel
}

func accountConfiguredSourceModelIDs(account *Account, sourceProtocol string) []string {
	if account == nil {
		return nil
	}
	normalizedSourceProtocol := NormalizeGatewayProtocol(sourceProtocol)
	seen := map[string]struct{}{}
	ordered := make([]string, 0)
	appendModel := func(modelID string) {
		normalized := normalizeRegistryID(modelID)
		if normalized == "" {
			return
		}
		if _, exists := seen[normalized]; exists {
			return
		}
		seen[normalized] = struct{}{}
		ordered = append(ordered, normalized)
	}

	for _, model := range AccountManualModelsFromExtra(account.Extra, IsProtocolGatewayAccount(account)) {
		if normalizedSourceProtocol != "" {
			manualProtocol := NormalizeGatewayProtocol(model.SourceProtocol)
			if manualProtocol != "" && manualProtocol != normalizedSourceProtocol {
				continue
			}
		}
		appendModel(model.ModelID)
	}

	if scope, ok := ExtractAccountModelScopeV2(account.Extra); ok && scope != nil {
		for _, models := range scope.SupportedModelsByProvider {
			for _, modelID := range models {
				appendModel(modelID)
			}
		}
		for _, row := range scope.ManualMappingRows {
			appendModel(row.To)
		}
		for _, modelID := range scope.ManualMappings {
			appendModel(modelID)
		}
	}

	for _, modelID := range account.GetModelMapping() {
		appendModel(modelID)
	}

	return ordered
}

func accountHasExplicitModelRestrictions(account *Account) bool {
	if account == nil {
		return false
	}
	if account.Type == AccountTypeBedrock || account.Platform == PlatformAntigravity {
		return true
	}
	if len(account.GetModelMapping()) > 0 {
		return true
	}
	return len(accountConfiguredSourceModelIDs(account, "")) > 0
}

func collectModelSupportVariants(ctx context.Context, registry *ModelRegistryService, route string, modelID string) map[string]struct{} {
	set := map[string]struct{}{}
	add := func(value string) {
		normalized := normalizeRegistryID(value)
		if normalized == "" {
			return
		}
		set[normalized] = struct{}{}
	}

	add(modelID)

	if registry != nil {
		if resolution, err := registry.ExplainResolution(ctx, modelID); err == nil && resolution != nil {
			add(resolution.CanonicalID)
			add(resolution.EffectiveID)
			add(resolution.PricingID)
			add(resolution.Entry.ID)
			for _, alias := range resolution.Entry.Aliases {
				add(alias)
			}
			for _, protocolID := range resolution.Entry.ProtocolIDs {
				add(protocolID)
			}
			if resolution.ReplacementEntry != nil {
				add(resolution.ReplacementEntry.ID)
				for _, alias := range resolution.ReplacementEntry.Aliases {
					add(alias)
				}
				for _, protocolID := range resolution.ReplacementEntry.ProtocolIDs {
					add(protocolID)
				}
			}
		}
		if resolved, ok, err := registry.ResolveModel(ctx, modelID); err == nil && ok {
			add(resolved)
		}
		if resolved, ok, err := registry.ResolveProtocolModel(ctx, modelID, route); err == nil && ok {
			add(resolved)
		}
		return set
	}

	if resolution, ok := modelregistry.ExplainSeedResolution(modelID); ok && resolution != nil {
		add(resolution.CanonicalID)
		add(resolution.EffectiveID)
		add(resolution.PricingID)
		add(resolution.Entry.ID)
		for _, alias := range resolution.Entry.Aliases {
			add(alias)
		}
		for _, protocolID := range resolution.Entry.ProtocolIDs {
			add(protocolID)
		}
		if resolution.ReplacementEntry != nil {
			add(resolution.ReplacementEntry.ID)
			for _, alias := range resolution.ReplacementEntry.Aliases {
				add(alias)
			}
			for _, protocolID := range resolution.ReplacementEntry.ProtocolIDs {
				add(protocolID)
			}
		}
	}
	if resolved, ok := modelregistry.ResolveToCanonicalID(modelID); ok {
		add(resolved)
	}
	if resolved, ok := modelregistry.ResolveToProtocolID(modelID, route); ok {
		add(resolved)
	}
	return set
}

func collectRequestedModelSupportVariants(ctx context.Context, registry *ModelRegistryService, account *Account, requestedModel string) map[string]struct{} {
	set := map[string]struct{}{}
	addAll := func(values map[string]struct{}) {
		for value := range values {
			set[value] = struct{}{}
		}
	}

	route := registryRouteForAccount(account)
	addAll(collectModelSupportVariants(ctx, registry, route, requestedModel))

	if canonical := resolveCanonicalRequestModelWithRegistry(ctx, registry, requestedModel); canonical != "" {
		addAll(collectModelSupportVariants(ctx, registry, route, canonical))
	}
	if upstream := resolveUpstreamModelIDWithRegistry(ctx, registry, account, requestedModel); upstream != "" {
		addAll(collectModelSupportVariants(ctx, registry, route, upstream))
	}
	return set
}

func getCachedAccountModelSupportSet(ctx context.Context, registry *ModelRegistryService, account *Account) *accountModelSupportSet {
	if account == nil {
		return &accountModelSupportSet{}
	}
	cacheKey := buildAccountModelSupportCacheKey(ctx, registry, account)
	if cacheKey == "" {
		return buildAccountModelSupportSet(ctx, registry, account)
	}
	now := time.Now()
	accountModelSupportCacheMu.RLock()
	cached, ok := accountModelSupportCache[cacheKey]
	accountModelSupportCacheMu.RUnlock()
	if ok && now.Before(cached.expiresAt) && cached.supportSet != nil {
		return cached.supportSet
	}

	result, _, _ := accountModelSupportCacheFlight.Do(cacheKey, func() (any, error) {
		refreshedNow := time.Now()
		accountModelSupportCacheMu.RLock()
		cached, ok := accountModelSupportCache[cacheKey]
		accountModelSupportCacheMu.RUnlock()
		if ok && refreshedNow.Before(cached.expiresAt) && cached.supportSet != nil {
			return cached.supportSet, nil
		}
		supportSet := buildAccountModelSupportSet(ctx, registry, account)
		accountModelSupportCacheMu.Lock()
		accountModelSupportCache[cacheKey] = accountModelSupportCacheEntry{
			supportSet: supportSet,
			expiresAt:  refreshedNow.Add(accountModelSupportCacheTTL),
		}
		accountModelSupportCacheMu.Unlock()
		return supportSet, nil
	})
	if supportSet, ok := result.(*accountModelSupportSet); ok && supportSet != nil {
		return supportSet
	}
	return buildAccountModelSupportSet(ctx, registry, account)
}

func buildAccountModelSupportSet(ctx context.Context, registry *ModelRegistryService, account *Account) *accountModelSupportSet {
	supportSet := &accountModelSupportSet{
		explicitRestrictions: accountHasExplicitModelRestrictions(account),
		allowedVariants:      map[string]struct{}{},
	}
	if account == nil || !supportSet.explicitRestrictions {
		return supportSet
	}
	explicitModelIDs := accountConfiguredSourceModelIDs(account, "")
	if len(explicitModelIDs) == 0 {
		return supportSet
	}
	route := registryRouteForAccount(account)
	for _, modelID := range explicitModelIDs {
		for candidate := range collectModelSupportVariants(ctx, registry, route, modelID) {
			supportSet.allowedVariants[candidate] = struct{}{}
		}
	}
	return supportSet
}

func buildAccountModelSupportCacheKey(ctx context.Context, registry *ModelRegistryService, account *Account) string {
	if account == nil {
		return ""
	}
	hash := sha256.New()
	writeAccountModelSupportHashValue(hash, CanonicalizePlatformValue(account.Platform))
	writeAccountModelSupportHashValue(hash, strings.TrimSpace(account.Type))
	writeAccountModelSupportHashValue(hash, registryRouteForAccount(account))
	writeAccountModelSupportHashValue(hash, resolveAccountModelSupportRegistryVersion(ctx, registry))

	mapping := account.GetModelMapping()
	if len(mapping) > 0 {
		keys := make([]string, 0, len(mapping))
		for key := range mapping {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			writeAccountModelSupportHashValue(hash, "map:"+normalizeRegistryID(key)+"=>"+normalizeRegistryID(mapping[key]))
		}
	}

	for _, modelID := range accountConfiguredSourceModelIDs(account, "") {
		writeAccountModelSupportHashValue(hash, "allow:"+normalizeRegistryID(modelID))
	}

	return hex.EncodeToString(hash.Sum(nil))
}

func writeAccountModelSupportHashValue(hash hash.Hash, value string) {
	_, _ = hash.Write([]byte(strings.TrimSpace(value)))
	_, _ = hash.Write([]byte{'\n'})
}

func resolveAccountModelSupportRegistryVersion(ctx context.Context, registry *ModelRegistryService) string {
	if registry == nil || registry.settingRepo == nil {
		return "seed"
	}
	cacheKey := fmt.Sprintf("%p", registry.settingRepo)
	if cached, ok := accountModelSupportRegistryVersion.Load(cacheKey); ok {
		if entry, ok := cached.(accountModelSupportRegistryVersionEntry); ok && time.Now().Before(entry.expiresAt) && entry.version != "" {
			return entry.version
		}
	}

	hash := sha256.New()
	for _, key := range []string{
		SettingKeyModelRegistryEntries,
		SettingKeyModelRegistryHiddenModels,
		SettingKeyModelRegistryTombstones,
		SettingKeyModelRegistryAvailableModels,
	} {
		value, _ := registry.settingRepo.GetValue(ctx, key)
		writeAccountModelSupportHashValue(hash, key)
		writeAccountModelSupportHashValue(hash, value)
	}
	version := hex.EncodeToString(hash.Sum(nil))
	accountModelSupportRegistryVersion.Store(cacheKey, accountModelSupportRegistryVersionEntry{
		version:   version,
		expiresAt: time.Now().Add(accountModelSupportRegistryVersionTTL),
	})
	return version
}

func resetAccountModelSupportRuntimeCaches() {
	accountModelSupportCacheMu.Lock()
	accountModelSupportCache = map[string]accountModelSupportCacheEntry{}
	accountModelSupportCacheMu.Unlock()
	accountModelSupportRegistryVersion = sync.Map{}
}

func isRequestedModelSupportedByAccount(ctx context.Context, registry *ModelRegistryService, account *Account, requestedModel string) bool {
	if account == nil {
		return false
	}
	if strings.TrimSpace(requestedModel) == "" {
		return true
	}
	if account.Type == AccountTypeBedrock {
		_, ok := ResolveBedrockModelID(account, requestedModel)
		return ok
	}
	if account.Platform == PlatformAntigravity {
		return mapAntigravityModel(account, requestedModel) != ""
	}

	requestedVariants := collectRequestedModelSupportVariants(ctx, registry, account, requestedModel)
	if len(requestedVariants) == 0 {
		return account.IsModelSupported(requestedModel)
	}

	if len(account.GetModelMapping()) > 0 {
		for candidate := range requestedVariants {
			if account.IsModelSupported(candidate) {
				return true
			}
		}
	}

	supportSet := getCachedAccountModelSupportSet(ctx, registry, account)
	if supportSet == nil || !supportSet.explicitRestrictions {
		return true
	}
	for candidate := range requestedVariants {
		if _, ok := supportSet.allowedVariants[candidate]; ok {
			return true
		}
	}
	return false
}
