import { adminAPI } from "@/api/admin";
import type { Account } from "@/types";
import { createAsyncTaskLimiter } from "@/utils/asyncTaskLimiter";
import { getUsageCacheEntry } from "./cache";
import {
  canAccountFetchUsage,
  getRuntimePlatform,
  getUsageLoadErrorMessage,
  prefersPassiveUsageSnapshot,
  shouldFallbackToActiveAnthropicUsage,
  supportsActiveAnthropicUsage,
  supportsActiveOpenAIUsage,
} from "./support";
import type {
  LoadUsageOptions,
  RefreshUsageOptions,
  RefreshUsageResult,
} from "./types";

const AUTO_USAGE_LOAD_CONCURRENCY = 3;
const autoUsageLoadLimiter = createAsyncTaskLimiter(
  AUTO_USAGE_LOAD_CONCURRENCY,
);

export async function performUsageLoad(
  account: Account,
  options: LoadUsageOptions = {},
): Promise<void> {
  const entry = getUsageCacheEntry(account.id);

  if (entry.request) {
    await entry.request;
    if (!options.force) {
      return;
    }
  }

  entry.loading = true;
  entry.error = null;

  const source =
    options.source ??
    (prefersPassiveUsageSnapshot(account) ? "passive" : undefined);

  const executeRequest = async () => {
    const data = await adminAPI.accounts.getUsage(account.id, {
      force: options.force,
      source,
    });

    let resolvedUsageInfo = data;

    if (shouldFallbackToActiveAnthropicUsage(account, source, data)) {
      try {
        resolvedUsageInfo = await adminAPI.accounts.getUsage(account.id, {
          force: options.force,
          source: "active",
        });
      } catch (fallbackError) {
        console.error(
          "Failed to supplement anthropic passive usage with active usage:",
          fallbackError,
        );
      }
    }

    entry.usageInfo = resolvedUsageInfo;
    entry.preferOpenAIFetchedUsage = Boolean(
      options.force &&
      getRuntimePlatform(account) === "openai" &&
      account.type === "oauth",
    );
  };

  const requestRunner = options.queue
    ? autoUsageLoadLimiter.run(executeRequest)
    : executeRequest();

  const request = requestRunner
    .catch((error) => {
      entry.error = getUsageLoadErrorMessage();
      entry.preferOpenAIFetchedUsage = false;
      console.error("Failed to load usage:", error);
      throw error;
    })
    .finally(() => {
      entry.loading = false;
      entry.request = null;
    });

  entry.request = request;
  await request;
}

export function resolveActualUsageRefreshLoadOptions(
  account: Account,
): LoadUsageOptions {
  if (
    supportsActiveAnthropicUsage(account) ||
    supportsActiveOpenAIUsage(account)
  ) {
    return { source: "active" };
  }

  return {};
}

export async function refreshAccountUsagePresentation(
  accounts: Account[],
  options: RefreshUsageOptions = {},
): Promise<RefreshUsageResult> {
  const refreshableAccounts = accounts.filter(canAccountFetchUsage);
  if (refreshableAccounts.length === 0) {
    return {
      total: 0,
      success: 0,
      activeSuccess: 0,
      fallbackSuccess: 0,
      failed: 0,
    };
  }

  const concurrency = Math.max(
    1,
    Math.min(options.concurrency ?? 4, refreshableAccounts.length),
  );
  let index = 0;
  let success = 0;
  let activeSuccess = 0;
  let fallbackSuccess = 0;
  let failed = 0;

  const worker = async () => {
    while (index < refreshableAccounts.length) {
      const current = refreshableAccounts[index];
      index += 1;

      try {
        const accountOptions = options.resolveLoadOptions?.(current);
        const resolvedSource = accountOptions?.source ?? options.source;

        await performUsageLoad(current, {
          force: accountOptions?.force ?? options.force,
          source: resolvedSource,
        });
        success += 1;
        if (resolvedSource === "active") {
          activeSuccess += 1;
        } else {
          fallbackSuccess += 1;
        }
      } catch {
        failed += 1;
      }
    }
  };

  await Promise.all(Array.from({ length: concurrency }, () => worker()));

  return {
    total: refreshableAccounts.length,
    success,
    activeSuccess,
    fallbackSuccess,
    failed,
  };
}
