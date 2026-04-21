package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const accountModelProjectionCacheTTL = 10 * time.Minute

type accountModelProjectionCacheEntry struct {
	projection *AccountModelProjection
	expiresAt  time.Time
}

var (
	accountModelProjectionCacheMu     sync.RWMutex
	accountModelProjectionCache       = map[string]accountModelProjectionCacheEntry{}
	accountModelProjectionCacheFlight singleflight.Group
)

func getCachedAccountModelProjection(ctx context.Context, account *Account, registry *ModelRegistryService) *AccountModelProjection {
	if account == nil {
		return nil
	}
	cacheKey := buildAccountModelProjectionCacheKey(ctx, account, registry)
	if cacheKey == "" {
		return buildAccountModelProjectionUncached(ctx, account, registry)
	}

	now := time.Now()
	accountModelProjectionCacheMu.RLock()
	cached, ok := accountModelProjectionCache[cacheKey]
	accountModelProjectionCacheMu.RUnlock()
	if ok && now.Before(cached.expiresAt) && cached.projection != nil {
		return cloneAccountModelProjection(cached.projection)
	}

	result, _, _ := accountModelProjectionCacheFlight.Do(cacheKey, func() (any, error) {
		refreshedNow := time.Now()
		accountModelProjectionCacheMu.RLock()
		cached, ok := accountModelProjectionCache[cacheKey]
		accountModelProjectionCacheMu.RUnlock()
		if ok && refreshedNow.Before(cached.expiresAt) && cached.projection != nil {
			return cloneAccountModelProjection(cached.projection), nil
		}

		projection := buildAccountModelProjectionUncached(ctx, account, registry)
		accountModelProjectionCacheMu.Lock()
		accountModelProjectionCache[cacheKey] = accountModelProjectionCacheEntry{
			projection: cloneAccountModelProjection(projection),
			expiresAt:  refreshedNow.Add(accountModelProjectionCacheTTL),
		}
		accountModelProjectionCacheMu.Unlock()
		return cloneAccountModelProjection(projection), nil
	})
	if projection, ok := result.(*AccountModelProjection); ok {
		return projection
	}
	return buildAccountModelProjectionUncached(ctx, account, registry)
}

func buildAccountModelProjectionCacheKey(ctx context.Context, account *Account, registry *ModelRegistryService) string {
	if account == nil {
		return ""
	}

	payload := map[string]any{
		"platform":         CanonicalizePlatformValue(account.Platform),
		"type":             account.Type,
		"effective_route":  registryRouteForAccount(account),
		"registry_version": resolveAccountModelSupportRegistryVersion(ctx, registry),
	}
	if account.Credentials != nil {
		payload["model_mapping"] = account.Credentials["model_mapping"]
	}
	if account.Extra != nil {
		payload["model_scope_v2"] = account.Extra["model_scope_v2"]
		payload[accountModelProbeSnapshotExtraKey] = account.Extra[accountModelProbeSnapshotExtraKey]
		payload[gatewayExtraProtocolKey] = account.Extra[gatewayExtraProtocolKey]
		payload[gatewayExtraAcceptedProtocolsKey] = account.Extra[gatewayExtraAcceptedProtocolsKey]
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func cloneAccountModelProjection(projection *AccountModelProjection) *AccountModelProjection {
	if projection == nil {
		return nil
	}
	cloned := &AccountModelProjection{
		PolicyMode: projection.PolicyMode,
		Explicit:   projection.Explicit,
		Source:     projection.Source,
	}
	if len(projection.Entries) > 0 {
		cloned.Entries = append([]AccountModelProjectionEntry(nil), projection.Entries...)
	}
	return cloned
}

func resetAccountModelProjectionCache() {
	accountModelProjectionCacheMu.Lock()
	accountModelProjectionCache = map[string]accountModelProjectionCacheEntry{}
	accountModelProjectionCacheMu.Unlock()
}
