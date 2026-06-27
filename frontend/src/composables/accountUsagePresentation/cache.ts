import { reactive } from "vue";
import type { UsageCacheEntry } from "./types";

const usageCache = new Map<number, UsageCacheEntry>();

function createUsageCacheEntry(): UsageCacheEntry {
  return reactive({
    loading: false,
    error: null,
    usageInfo: null,
    loadedAtMs: null,
    preferOpenAIFetchedUsage: false,
    request: null,
  }) as UsageCacheEntry;
}

export function getUsageCacheEntry(accountID: number): UsageCacheEntry {
  const cached = usageCache.get(accountID);
  if (cached) return cached;

  const created = createUsageCacheEntry();
  usageCache.set(accountID, created);
  return created;
}

export function resetUsageCacheEntry(entry: UsageCacheEntry): void {
  entry.loading = false;
  entry.error = null;
  entry.usageInfo = null;
  entry.loadedAtMs = null;
  entry.preferOpenAIFetchedUsage = false;
  entry.request = null;
}

export function resetAccountUsagePresentationCache(): void {
  usageCache.clear();
}

export function invalidateAccountUsagePresentationCache(
  accountIDs: number[],
): void {
  const uniqueIDs = [
    ...new Set(
      accountIDs.filter(
        (accountID) => Number.isFinite(accountID) && accountID > 0,
      ),
    ),
  ];
  uniqueIDs.forEach((accountID) => {
    const entry = usageCache.get(accountID);
    if (entry) {
      resetUsageCacheEntry(entry);
    }
  });
}
