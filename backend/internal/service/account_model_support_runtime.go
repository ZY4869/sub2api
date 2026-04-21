package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
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
	policySource      string
	hasPolicy         bool
	allowedDisplayIDs map[string]struct{}
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

func accountHasExplicitModelRestrictions(account *Account) bool {
	if account == nil {
		return false
	}
	if account.Type == AccountTypeBedrock || account.Platform == PlatformAntigravity {
		return true
	}
	if scope, ok := ExtractAccountModelScopeV2(account.Extra); ok && scope != nil {
		return true
	}
	if len(account.GetModelMapping()) > 0 {
		return true
	}
	return false
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
		allowedDisplayIDs: map[string]struct{}{},
	}
	if account == nil {
		return supportSet
	}

	projection := BuildAccountModelProjection(ctx, account, registry)
	if projection == nil {
		return supportSet
	}

	supportSet.policySource = strings.TrimSpace(projection.Source)
	// Default library projections are availability hints only; they must not turn
	// unrestricted accounts into implicit allowlists during scheduling checks.
	supportSet.hasPolicy = projection.Explicit
	if len(projection.Entries) == 0 {
		return supportSet
	}
	for _, entry := range projection.Entries {
		for _, candidate := range accountModelRequestedDisplayCandidates(entry.DisplayModelID) {
			supportSet.allowedDisplayIDs[candidate] = struct{}{}
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

	if projection := BuildAccountModelProjection(ctx, account, registry); projection != nil {
		writeAccountModelSupportHashValue(hash, "policy_mode:"+strings.TrimSpace(projection.PolicyMode))
		writeAccountModelSupportHashValue(hash, "policy_source:"+strings.TrimSpace(projection.Source))
		if projection.Explicit {
			writeAccountModelSupportHashValue(hash, "policy_explicit:true")
		} else {
			writeAccountModelSupportHashValue(hash, "policy_explicit:false")
		}
		for _, entry := range projection.Entries {
			writeAccountModelSupportHashValue(hash, "display:"+normalizeRegistryID(entry.DisplayModelID))
			writeAccountModelSupportHashValue(hash, "target:"+normalizeRegistryID(entry.TargetModelID))
			writeAccountModelSupportHashValue(hash, "route:"+normalizeRegistryID(entry.RouteModelID))
			writeAccountModelSupportHashValue(hash, "source_protocol:"+NormalizeGatewayProtocol(entry.SourceProtocol))
			writeAccountModelSupportHashValue(hash, "visibility:"+strings.TrimSpace(entry.VisibilityMode))
		}
		return hex.EncodeToString(hash.Sum(nil))
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

	supportSet := getCachedAccountModelSupportSet(ctx, registry, account)
	if supportSet == nil || !supportSet.hasPolicy {
		return true
	}
	for _, candidate := range accountModelRequestedDisplayCandidates(requestedModel) {
		if _, ok := supportSet.allowedDisplayIDs[candidate]; ok {
			return true
		}
	}
	return false
}

func accountModelRequestedDisplayCandidates(modelID string) []string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil
	}

	ordered := make([]string, 0, 3)
	seen := map[string]struct{}{}
	appendCandidate := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		ordered = append(ordered, value)
	}

	appendCandidate(modelID)
	appendCandidate(normalizeRegistryID(modelID))
	appendCandidate(NormalizeModelCatalogModelID(modelID))
	return ordered
}
