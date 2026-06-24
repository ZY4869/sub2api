import {
  computed,
  ref,
  toValue,
  watch,
  type MaybeRefOrGetter,
} from "vue";
import { useI18n } from "vue-i18n";
import { useRealtimeCountdownNow } from "@/composables/useRealtimeCountdownNow";
import type {
  Account,
  AccountUsagePresentation,
  AccountUsagePresentationRow,
  AccountUsageRowColor,
  UsageProgress,
} from "@/types";
import { buildOpenAIUsageRefreshKey } from "@/utils/accountUsageRefresh";
import {
  resolveCodexUsageWindow,
  resolveCodexUsageWindowLabel,
  resolveUsageWindowColor,
} from "@/utils/codexUsage";
import {
  formatLocalAbsoluteTime,
  formatLocalTimestamp,
  parseEffectiveResetAt,
} from "@/utils/usageResetTime";
import { resolveUsageWindowLabel } from "@/utils/displayLabels";
import {
  getUsageCacheEntry,
  invalidateAccountUsagePresentationCache,
  resetAccountUsagePresentationCache,
} from "./accountUsagePresentation/cache";
import {
  performUsageLoad,
  refreshAccountUsagePresentation,
  resolveActualUsageRefreshLoadOptions,
} from "./accountUsagePresentation/loader";
import { useGeminiUsagePresentationState } from "./accountUsagePresentation/gemini";
import {
  buildProgressRow,
  buildRows,
  buildUsageRow,
  createEmptyMeta,
  isOpenAIFreeAccount,
  isOpenAIProAccount,
  mergeOpenAIUsageRows,
  resolveOpenAIUsageProgress,
  resolveOpenAIUsageSpecs,
} from "./accountUsagePresentation/rows";
import {
  canAccountFetchUsage,
  getRuntimePlatform,
} from "./accountUsagePresentation/support";
import type {
  LoadUsageOptions,
  OpenAIUsageRowSpec,
  UseAccountUsagePresentationOptions,
} from "./accountUsagePresentation/types";

export {
  canAccountFetchUsage,
  invalidateAccountUsagePresentationCache,
  refreshAccountUsagePresentation,
  resetAccountUsagePresentationCache,
  resolveActualUsageRefreshLoadOptions,
};

const EXPIRED_OPENAI_USAGE_REFRESH_COOLDOWN_MS = 60 * 1000;
const SEVEN_DAYS_SECONDS = 7 * 24 * 60 * 60;

function normalizeOpenAIResetCreditsCount(value: unknown): number | null {
  if (typeof value === "number" && Number.isFinite(value) && value >= 0) {
    return Math.floor(value);
  }
  if (typeof value !== "string") return null;

  const trimmed = value.trim();
  if (!/^\d+(\.\d+)?$/.test(trimmed)) return null;

  const numeric = Number(trimmed);
  return Number.isFinite(numeric) && numeric >= 0
    ? Math.floor(numeric)
    : null;
}

function normalizeOpenAIResetCreditsStatus(value: unknown): string {
  if (typeof value !== "string") return "unknown_or_unsupported";
  const trimmed = value.trim();
  if (
    trimmed === "available" ||
    trimmed === "unknown_or_unsupported" ||
    trimmed === "unsupported"
  ) {
    return trimmed;
  }
  return "unknown_or_unsupported";
}

function normalizeOpenAIResetCreditsReason(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function resolveLongOpenAIUsageFallbackLabel(
  window: "5h" | "7d",
  progress: UsageProgress | null | undefined,
): string {
  if (
    window === "7d" &&
    typeof progress?.remaining_seconds === "number" &&
    Number.isFinite(progress.remaining_seconds) &&
    progress.remaining_seconds > SEVEN_DAYS_SECONDS
  ) {
    return resolveCodexUsageWindowLabel(30 * 24 * 60, window);
  }
  return resolveCodexUsageWindowLabel(null, window);
}

export function useAccountUsagePresentation(
  accountSource: MaybeRefOrGetter<Account>,
  options: UseAccountUsagePresentationOptions = {},
) {
  const { t } = useI18n();
  const { nowMs, nowDate } = useRealtimeCountdownNow("accounts");
  const account = computed(() => toValue(accountSource));
  const cacheEntry = computed(() => getUsageCacheEntry(account.value.id));
  const usageInfo = computed(() => cacheEntry.value.usageInfo);
  const lastExpiredOpenAIUsageRefresh = ref<{
    key: string;
    requestedAt: number;
  } | null>(null);
  const pendingAutoLoadOptions = ref<LoadUsageOptions | null>(null);
  const autoLoadEnabled = computed(() => {
    return options.autoLoadEnabled == null
      ? true
      : Boolean(toValue(options.autoLoadEnabled));
  });
  const shouldShowOpenAISparkUsage = computed(() => {
    if (
      getRuntimePlatform(account.value) !== "openai" ||
      account.value.type !== "oauth"
    )
      return false;
    return isOpenAIProAccount(account.value);
  });
  const shouldShowOnlyOpenAI7dUsage = computed(() => {
    if (
      getRuntimePlatform(account.value) !== "openai" ||
      account.value.type !== "oauth"
    )
      return false;
    return isOpenAIFreeAccount(account.value);
  });

  const loadingRows = computed(() => {
    const runtimePlatform = getRuntimePlatform(account.value);
    if (runtimePlatform === "anthropic") {
      return account.value.type === "oauth" ? 3 : 1;
    }
    if (runtimePlatform === "openai")
      return shouldShowOnlyOpenAI7dUsage.value
        ? 1
        : shouldShowOpenAISparkUsage.value
          ? 4
          : 2;
    return 1;
  });

  const shouldFetchUsage = computed(() => {
    return canAccountFetchUsage(account.value);
  });

  const codex5hWindow = computed(() =>
    resolveCodexUsageWindow(account.value.extra, "5h", nowDate.value),
  );
  const codex7dWindow = computed(() =>
    resolveCodexUsageWindow(account.value.extra, "7d", nowDate.value),
  );
  const codexSpark5hWindow = computed(() =>
    resolveCodexUsageWindow(account.value.extra, "5h", nowDate.value, "spark"),
  );
  const codexSpark7dWindow = computed(() =>
    resolveCodexUsageWindow(account.value.extra, "7d", nowDate.value, "spark"),
  );

  const resolveOpenAIFetchedUsageLabel = (
    spec: OpenAIUsageRowSpec,
    progress: UsageProgress | null | undefined,
  ) => {
    const snapshotWindow = resolveCodexUsageWindow(
      account.value.extra,
      spec.window,
      nowDate.value,
      spec.scope,
    );
    const baseLabel =
      snapshotWindow.windowMinutes != null
        ? snapshotWindow.label
        : resolveLongOpenAIUsageFallbackLabel(spec.window, progress);
    return spec.scope === "spark" ? `Spark ${baseLabel}` : baseLabel;
  };

  const codexRows = computed(() =>
    buildRows(
      shouldShowOnlyOpenAI7dUsage.value ||
        codex5hWindow.value.usedPercent === null
        ? null
        : buildUsageRow(
            "openai-5h",
            codex5hWindow.value.label,
            codex5hWindow.value.usedPercent,
            codex5hWindow.value.resetAt,
            resolveUsageWindowColor(codex5hWindow.value.label),
            {
              inlineRemaining: true,
              detailedReset: true,
            },
          ),
      codex7dWindow.value.usedPercent === null
        ? null
        : buildUsageRow(
            "openai-7d",
            codex7dWindow.value.label,
            codex7dWindow.value.usedPercent,
            codex7dWindow.value.resetAt,
            resolveUsageWindowColor(codex7dWindow.value.label),
            {
              inlineRemaining: true,
              detailedReset: true,
            },
          ),
      !shouldShowOpenAISparkUsage.value ||
        codexSpark5hWindow.value.usedPercent === null
        ? null
        : buildUsageRow(
            "openai-spark-5h",
            `Spark ${codexSpark5hWindow.value.label}`,
            codexSpark5hWindow.value.usedPercent,
            codexSpark5hWindow.value.resetAt,
            resolveUsageWindowColor(codexSpark5hWindow.value.label),
            {
              inlineRemaining: true,
              detailedReset: true,
            },
          ),
      !shouldShowOpenAISparkUsage.value ||
        codexSpark7dWindow.value.usedPercent === null
        ? null
        : buildUsageRow(
            "openai-spark-7d",
            `Spark ${codexSpark7dWindow.value.label}`,
            codexSpark7dWindow.value.usedPercent,
            codexSpark7dWindow.value.resetAt,
            resolveUsageWindowColor(codexSpark7dWindow.value.label),
            {
              inlineRemaining: true,
              detailedReset: true,
            },
          ),
    ),
  );

  const hasCodexUsage = computed(() => codexRows.value.length > 0);
  const expectedOpenAICodexRowCount = computed(() =>
    shouldShowOnlyOpenAI7dUsage.value
      ? 1
      : shouldShowOpenAISparkUsage.value
        ? 4
        : 2,
  );
  const hasCompleteCodexUsage = computed(
    () => codexRows.value.length === expectedOpenAICodexRowCount.value,
  );

  const openAIFetchedRows = computed(() =>
    buildRows(
      ...resolveOpenAIUsageSpecs(
        shouldShowOpenAISparkUsage.value,
        shouldShowOnlyOpenAI7dUsage.value,
      ).map((spec) => {
        const progress = resolveOpenAIUsageProgress(usageInfo.value, spec);
        const label = resolveOpenAIFetchedUsageLabel(spec, progress);
        return (
          buildProgressRow(
            spec.key,
            label,
            progress,
            resolveUsageWindowColor(label),
            { inlineRemaining: true, detailedReset: true },
          )
        );
      }),
    ),
  );

  const hasOpenAIUsageFallback = computed(
    () => openAIFetchedRows.value.length > 0,
  );

  const isActiveOpenAIRateLimited = computed(() => {
    if (
      getRuntimePlatform(account.value) !== "openai" ||
      account.value.type !== "oauth"
    )
      return false;
    if (!account.value.rate_limit_reset_at) return false;

    const resetAt = Date.parse(account.value.rate_limit_reset_at);
    return !Number.isNaN(resetAt) && resetAt > nowMs.value;
  });

  const isOpenAICodexSnapshotStale = computed(() => {
    if (
      getRuntimePlatform(account.value) !== "openai" ||
      account.value.type !== "oauth"
    )
      return false;

    const updatedAtRaw = account.value.extra?.codex_usage_updated_at;
    if (!updatedAtRaw) return true;

    const updatedAt = Date.parse(String(updatedAtRaw));
    if (Number.isNaN(updatedAt)) return true;

    return nowMs.value - updatedAt >= 10 * 60 * 1000;
  });

  const openAICodexEarliestResetAt = computed(() => {
    if (
      getRuntimePlatform(account.value) !== "openai" ||
      account.value.type !== "oauth"
    )
      return null;

    let earliestResetAt: string | null = null;
    let earliestResetMs = Number.POSITIVE_INFINITY;

    for (const row of codexRows.value) {
      const effectiveResetAt = parseEffectiveResetAt(
        row.resetsAt,
        row.remainingSeconds,
      );
      if (!effectiveResetAt) continue;

      const resetMs = effectiveResetAt.getTime();
      if (resetMs < earliestResetMs) {
        earliestResetMs = resetMs;
        earliestResetAt = effectiveResetAt.toISOString();
      }
    }

    return earliestResetAt;
  });

  const hasExpiredOpenAICodexWindow = computed(() => {
    if (!openAICodexEarliestResetAt.value) return false;

    const resetAt = Date.parse(openAICodexEarliestResetAt.value);
    return !Number.isNaN(resetAt) && resetAt <= nowMs.value;
  });

  const preferFetchedOpenAIUsage = computed(() => {
    return (
      (isActiveOpenAIRateLimited.value ||
        isOpenAICodexSnapshotStale.value ||
        hasExpiredOpenAICodexWindow.value) &&
      hasOpenAIUsageFallback.value
    );
  });

  const shouldUseFetchedOpenAIUsage = computed(() => {
    if (!hasOpenAIUsageFallback.value) return false;
    return (
      preferFetchedOpenAIUsage.value ||
      cacheEntry.value.preferOpenAIFetchedUsage
    );
  });

  const openAIResolvedRows = computed(() => {
    if (
      getRuntimePlatform(account.value) !== "openai" ||
      account.value.type !== "oauth"
    )
      return [];

    const fillMissingRows = isOpenAIProAccount(account.value);

    if (shouldUseFetchedOpenAIUsage.value) {
      return mergeOpenAIUsageRows(
        openAIFetchedRows.value,
        codexRows.value,
        shouldShowOpenAISparkUsage.value,
        shouldShowOnlyOpenAI7dUsage.value,
        fillMissingRows,
        t,
      );
    }

    return mergeOpenAIUsageRows(
      codexRows.value,
      openAIFetchedRows.value,
      shouldShowOpenAISparkUsage.value,
      shouldShowOnlyOpenAI7dUsage.value,
      fillMissingRows,
      t,
    );
  });

  const shouldPreferFetchedOpenAIMeta = computed(() => {
    if (
      getRuntimePlatform(account.value) !== "openai" ||
      account.value.type !== "oauth"
    )
      return false;
    return (
      shouldUseFetchedOpenAIUsage.value ||
      (!hasCompleteCodexUsage.value && hasOpenAIUsageFallback.value)
    );
  });

  const shouldAutoLoadUsageOnMount = computed(() => {
    if (
      getRuntimePlatform(account.value) === "openai" &&
      account.value.type === "oauth"
    ) {
      return (
        isActiveOpenAIRateLimited.value ||
        !hasCompleteCodexUsage.value ||
        isOpenAICodexSnapshotStale.value
      );
    }
    return shouldFetchUsage.value;
  });

  const openAIUsageRefreshKey = computed(() =>
    buildOpenAIUsageRefreshKey(account.value),
  );

  const snapshotUpdatedAt = computed(() => {
    if (!hasCodexUsage.value || shouldPreferFetchedOpenAIMeta.value)
      return null;

    const updatedAtRaw = account.value.extra?.codex_usage_updated_at;
    if (typeof updatedAtRaw !== "string" || updatedAtRaw.trim() === "")
      return null;

    const parsed = new Date(updatedAtRaw);
    if (Number.isNaN(parsed.getTime())) return null;

    return parsed;
  });

  const openAISnapshotUpdatedAtText = computed(() => {
    if (!snapshotUpdatedAt.value) return "";
    return formatLocalAbsoluteTime(snapshotUpdatedAt.value, nowDate.value, {
      today: t("dates.today"),
      tomorrow: t("dates.tomorrow"),
    });
  });

  const openAISnapshotUpdatedAtTooltip = computed(() => {
    if (!snapshotUpdatedAt.value) return "";
    return formatLocalTimestamp(snapshotUpdatedAt.value);
  });
  const openAIResetCredits = computed(() => {
    if (
      getRuntimePlatform(account.value) !== "openai" ||
      account.value.type !== "oauth"
    ) {
      return {
        known: false,
        count: null,
        status: "unknown_or_unsupported",
        unsupportedReason: "",
      };
    }

    const usageStatus = normalizeOpenAIResetCreditsStatus(
      usageInfo.value?.openai_reset_credits?.status,
    );
    const usageReason = normalizeOpenAIResetCreditsReason(
      usageInfo.value?.openai_reset_credits?.unsupported_reason,
    );
    const hasUsageResetCredits = usageInfo.value?.openai_reset_credits != null;
    const usageCount = normalizeOpenAIResetCreditsCount(
      usageInfo.value?.openai_reset_credits?.available_count,
    );
    if (usageCount !== null) {
      return {
        known: true,
        count: usageCount,
        status: "available",
        unsupportedReason: "",
      };
    }
    if (usageStatus === "unsupported") {
      return {
        known: false,
        count: null,
        status: usageStatus,
        unsupportedReason: usageReason,
      };
    }
    if (hasUsageResetCredits) {
      return {
        known: false,
        count: null,
        status: usageStatus,
        unsupportedReason: "",
      };
    }

    const extraStatusRaw = account.value.extra?.openai_rate_limit_reset_credits_status;
    const hasExtraStatus = typeof extraStatusRaw === "string" && extraStatusRaw.trim() !== "";
    const extraStatus = normalizeOpenAIResetCreditsStatus(extraStatusRaw);
    const extraReason = normalizeOpenAIResetCreditsReason(
      account.value.extra?.openai_rate_limit_reset_credits_unsupported_reason,
    );
    const extraCount = normalizeOpenAIResetCreditsCount(
      account.value.extra?.openai_rate_limit_reset_credits_available_count,
    );
    if (extraStatus === "unsupported") {
      return {
        known: false,
        count: null,
        status: extraStatus,
        unsupportedReason: extraReason,
      };
    }
    if (hasExtraStatus && extraStatus === "unknown_or_unsupported") {
      return {
        known: false,
        count: null,
        status: extraStatus,
        unsupportedReason: "",
      };
    }
    if (extraCount !== null) {
      return {
        known: true,
        count: extraCount,
        status: "available",
        unsupportedReason: "",
      };
    }

    return {
      known: false,
      count: null,
      status: usageStatus || extraStatus,
      unsupportedReason: "",
    };
  });
  const fetchedSnapshotUpdatedAt = computed(() => {
    const updatedAtRaw = usageInfo.value?.updated_at;
    if (typeof updatedAtRaw !== "string" || updatedAtRaw.trim() === "")
      return null;

    const parsed = new Date(updatedAtRaw);
    if (Number.isNaN(parsed.getTime())) return null;

    return parsed;
  });
  const fetchedSnapshotUpdatedAtText = computed(() => {
    if (!fetchedSnapshotUpdatedAt.value) return "";
    return formatLocalAbsoluteTime(
      fetchedSnapshotUpdatedAt.value,
      nowDate.value,
      {
        today: t("dates.today"),
        tomorrow: t("dates.tomorrow"),
      },
    );
  });
  const fetchedSnapshotUpdatedAtTooltip = computed(() => {
    if (!fetchedSnapshotUpdatedAt.value) return "";
    return formatLocalTimestamp(fetchedSnapshotUpdatedAt.value);
  });

  const getAntigravityUsageFromAPI = (modelNames: string[]) => {
    const quota = usageInfo.value?.antigravity_quota;
    if (!quota) return null;

    let maxUtilization = 0;
    let earliestReset: string | null = null;

    for (const model of modelNames) {
      const modelQuota = quota[model];
      if (!modelQuota) continue;

      if (modelQuota.utilization > maxUtilization) {
        maxUtilization = modelQuota.utilization;
      }

      if (
        modelQuota.reset_time &&
        (!earliestReset || modelQuota.reset_time < earliestReset)
      ) {
        earliestReset = modelQuota.reset_time;
      }
    }

    if (maxUtilization === 0 && earliestReset === null) {
      const hasAnyData = modelNames.some((model) => quota[model]);
      if (!hasAnyData) return null;
    }

    return { utilization: maxUtilization, resetTime: earliestReset };
  };

  const antigravityRows = computed(() =>
    buildRows(
      (() => {
        const usage = getAntigravityUsageFromAPI([
          "gemini-3-pro-low",
          "gemini-3-pro-high",
          "gemini-3-pro-preview",
        ]);
        return usage
          ? buildUsageRow(
              "antigravity-g3p",
              t("admin.accounts.usageWindow.gemini3Pro"),
              usage.utilization,
              usage.resetTime,
              "indigo",
            )
          : null;
      })(),
      (() => {
        const usage = getAntigravityUsageFromAPI(["gemini-3-flash"]);
        return usage
          ? buildUsageRow(
              "antigravity-g3f",
              t("admin.accounts.usageWindow.gemini3Flash"),
              usage.utilization,
              usage.resetTime,
              "emerald",
            )
          : null;
      })(),
      (() => {
        const usage = getAntigravityUsageFromAPI([
          "gemini-2.5-flash-image",
          "gemini-3.1-flash-image",
          "gemini-3-pro-image",
        ]);
        return usage
          ? buildUsageRow(
              "antigravity-image",
              t("admin.accounts.usageWindow.gemini3Image"),
              usage.utilization,
              usage.resetTime,
              "purple",
            )
          : null;
      })(),
      (() => {
        const usage = getAntigravityUsageFromAPI([
          "claude-opus-4.1",
          "claude-sonnet-4.5",
          "claude-haiku-4.5",
          "claude-opus-4-1-20250805",
          "claude-sonnet-4-5-20250929",
          "claude-haiku-4-5-20251001",
          "claude-sonnet-4-5",
          "claude-opus-4-5-thinking",
          "claude-sonnet-4-6",
          "claude-opus-4-6",
          "claude-opus-4-6-thinking",
        ]);
        return usage
          ? buildUsageRow(
              "antigravity-claude",
              t("admin.accounts.usageWindow.claude"),
              usage.utilization,
              usage.resetTime,
              "amber",
            )
          : null;
      })(),
    ),
  );

  const antigravityTier = computed(() => {
    const loadCodeAssist = account.value.extra?.load_code_assist as
      | Record<string, unknown>
      | undefined;
    if (!loadCodeAssist) return null;

    const paidTier = loadCodeAssist.paidTier as
      | Record<string, unknown>
      | undefined;
    if (paidTier && typeof paidTier.id === "string") {
      return paidTier.id;
    }

    const currentTier = loadCodeAssist.currentTier as
      | Record<string, unknown>
      | undefined;
    if (currentTier && typeof currentTier.id === "string") {
      return currentTier.id;
    }

    return null;
  });

  const antigravityTierLabel = computed(() => {
    switch (antigravityTier.value) {
      case "free-tier":
        return t("admin.accounts.tier.free");
      case "g1-pro-tier":
        return t("admin.accounts.tier.pro");
      case "g1-ultra-tier":
        return t("admin.accounts.tier.ultra");
      default:
        return null;
    }
  });

  const antigravityTierClass = computed(() => {
    switch (antigravityTier.value) {
      case "free-tier":
        return "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300";
      case "g1-pro-tier":
        return "bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300";
      case "g1-ultra-tier":
        return "bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300";
      default:
        return "";
    }
  });

  const hasIneligibleTiers = computed(() => {
    const loadCodeAssist = account.value.extra?.load_code_assist as
      | Record<string, unknown>
      | undefined;
    const ineligibleTiers = loadCodeAssist?.ineligibleTiers as
      | unknown[]
      | undefined;
    return Array.isArray(ineligibleTiers) && ineligibleTiers.length > 0;
  });

  const {
    geminiAuthTypeLabel,
    geminiTierClass,
    geminiQuotaPolicyChannel,
    geminiQuotaPolicyLimits,
    geminiQuotaPolicyDocsUrl,
    isProtocolGatewayGeminiAccount,
    protocolGatewayBadgeLabel,
    protocolGatewayBadgeClass,
    geminiRows,
  } = useGeminiUsagePresentationState(account, usageInfo, t);

  const hasAccountQuota = computed(() => {
    return (
      (account.value.quota_daily_limit ?? 0) > 0 ||
      (account.value.quota_weekly_limit ?? 0) > 0 ||
      (account.value.quota_monthly_limit ?? 0) > 0 ||
      (account.value.quota_limit ?? 0) > 0
    );
  });

  const normalizeResetAt = (value: unknown): string | null => {
    if (typeof value !== "string" || value.trim() === "") return null;
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return null;
    return date.toISOString();
  };

  const makeQuotaRow = (
    key: string,
    label: string,
    used: number,
    limit: number,
    color: AccountUsageRowColor,
    options: {
      resetAt?: string | null;
      startKey?: string;
      periodMs?: number;
    } = {},
  ): AccountUsagePresentationRow | null => {
    if (limit <= 0) return null;

    let resetsAt: string | null = normalizeResetAt(options.resetAt);
    if (!resetsAt && options.startKey && options.periodMs) {
      const startStr = account.value.extra?.[options.startKey] as string | undefined;
      if (startStr) {
        const startDate = new Date(startStr);
        if (!Number.isNaN(startDate.getTime())) {
          resetsAt = new Date(startDate.getTime() + options.periodMs).toISOString();
        }
      }
    }

    const utilization = limit > 0 ? (used / limit) * 100 : 0;
    return buildUsageRow(key, label, utilization, resetsAt, color);
  };

  const apiKeyQuotaRows = computed(() =>
    buildRows(
      makeQuotaRow(
        "apikey-daily",
        resolveUsageWindowLabel("1d", t),
        account.value.quota_daily_used ?? 0,
        account.value.quota_daily_limit ?? 0,
        "indigo",
        {
          resetAt: account.value.quota_daily_reset_at,
          startKey: "quota_daily_start",
          periodMs: 24 * 60 * 60 * 1000,
        },
      ),
      makeQuotaRow(
        "apikey-weekly",
        resolveUsageWindowLabel("7d", t),
        account.value.quota_weekly_used ?? 0,
        account.value.quota_weekly_limit ?? 0,
        "orange",
        {
          resetAt: account.value.quota_weekly_reset_at,
          startKey: "quota_weekly_start",
          periodMs: 7 * 24 * 60 * 60 * 1000,
        },
      ),
      makeQuotaRow(
        "apikey-monthly",
        "30D",
        account.value.quota_monthly_used ?? 0,
        account.value.quota_monthly_limit ?? 0,
        "green",
        {
          resetAt: account.value.quota_monthly_reset_at,
          startKey: "quota_monthly_start",
          periodMs: 30 * 24 * 60 * 60 * 1000,
        },
      ),
      makeQuotaRow(
        "apikey-total",
        resolveUsageWindowLabel("total", t),
        account.value.quota_used ?? 0,
        account.value.quota_limit ?? 0,
        "purple",
      ),
    ),
  );

  const anthropicRows = computed(() =>
    buildRows(
      buildProgressRow(
        "anthropic-5h",
        "5h",
        usageInfo.value?.five_hour,
        "indigo",
      ),
      buildProgressRow(
        "anthropic-7d",
        "7d",
        usageInfo.value?.seven_day,
        "orange",
      ),
      buildProgressRow(
        "anthropic-7d-sonnet",
        "7d S",
        usageInfo.value?.seven_day_sonnet,
        "orange",
      ),
    ),
  );

  const openAIRefreshRows = computed(() => {
    if (
      getRuntimePlatform(account.value) !== "openai" ||
      account.value.type !== "oauth"
    )
      return [];
    return openAIResolvedRows.value;
  });

  const openAIEarliestResetAt = computed(() => {
    if (
      getRuntimePlatform(account.value) !== "openai" ||
      account.value.type !== "oauth"
    )
      return null;

    let earliestResetAt: string | null = null;
    let earliestResetMs = Number.POSITIVE_INFINITY;

    for (const row of openAIRefreshRows.value) {
      const effectiveResetAt = parseEffectiveResetAt(
        row.resetsAt,
        row.remainingSeconds,
      );
      if (!effectiveResetAt) continue;

      const resetMs = effectiveResetAt.getTime();
      if (resetMs < earliestResetMs) {
        earliestResetMs = resetMs;
        earliestResetAt = effectiveResetAt.toISOString();
      }
    }

    return earliestResetAt;
  });

  const loadUsage = async (options: LoadUsageOptions = {}) => {
    const currentAccount = account.value;
    if (!shouldFetchUsage.value) return;
    await performUsageLoad(currentAccount, options).catch(() => {
      const entry = getUsageCacheEntry(currentAccount.id);
      if (!entry.error) {
        entry.error = t("common.error");
      }
    });
  };

  const queueAutoLoad = (loadOptions: LoadUsageOptions = {}) => {
    const pending = pendingAutoLoadOptions.value;
    pendingAutoLoadOptions.value = {
      force: loadOptions.force ?? pending?.force,
      source: loadOptions.source ?? pending?.source,
      queue: loadOptions.queue ?? pending?.queue,
    };
  };

  const requestAutoLoad = (loadOptions: LoadUsageOptions = {}) => {
    if (!shouldFetchUsage.value) return;
    if (!autoLoadEnabled.value) {
      queueAutoLoad(loadOptions);
      return;
    }

    loadUsage(loadOptions).catch((error) => {
      console.error("Failed to auto load usage:", error);
    });
  };

  watch(
    () =>
      [
        account.value.id,
        shouldAutoLoadUsageOnMount.value,
        autoLoadEnabled.value,
      ] as const,
    ([, shouldLoad]) => {
      if (!shouldLoad) return;
      if (cacheEntry.value.usageInfo || cacheEntry.value.loading) return;

      requestAutoLoad({ queue: true });
    },
    { immediate: true },
  );

  watch(openAIUsageRefreshKey, (nextKey, prevKey) => {
    if (!prevKey || nextKey === prevKey) return;
    if (
      getRuntimePlatform(account.value) !== "openai" ||
      account.value.type !== "oauth"
    )
      return;
    if (
      !isActiveOpenAIRateLimited.value &&
      hasCompleteCodexUsage.value &&
      !isOpenAICodexSnapshotStale.value
    )
      return;

    requestAutoLoad();
  });

  watch(
    () => [account.value.id, openAIEarliestResetAt.value, nowMs.value] as const,
    ([accountID, earliestResetAt, currentNow]) => {
      if (
        getRuntimePlatform(account.value) !== "openai" ||
        account.value.type !== "oauth"
      )
        return;
      if (!earliestResetAt) return;

      const resetAtMs = Date.parse(earliestResetAt);
      if (Number.isNaN(resetAtMs) || resetAtMs > currentNow) return;

      const refreshKey = `${accountID}:${earliestResetAt}`;
      const lastRefresh = lastExpiredOpenAIUsageRefresh.value;
      if (
        lastRefresh &&
        lastRefresh.key === refreshKey &&
        currentNow - lastRefresh.requestedAt <
          EXPIRED_OPENAI_USAGE_REFRESH_COOLDOWN_MS
      ) {
        return;
      }

      lastExpiredOpenAIUsageRefresh.value = {
        key: refreshKey,
        requestedAt: currentNow,
      };

      requestAutoLoad();
    },
    { immediate: true },
  );

  watch(autoLoadEnabled, (enabled) => {
    if (!enabled || !pendingAutoLoadOptions.value) return;

    const loadOptions = pendingAutoLoadOptions.value;
    pendingAutoLoadOptions.value = null;
    loadUsage(loadOptions).catch((error) => {
      console.error("Failed to flush deferred usage load:", error);
    });
  });

  const presentation = computed<AccountUsagePresentation>(() => {
    const meta = createEmptyMeta(loadingRows.value);
    const currentState = cacheEntry.value;
    let state: AccountUsagePresentation["state"] = "empty";
    let windowRows: AccountUsagePresentationRow[] = [];

    if (
      getRuntimePlatform(account.value) === "openai" &&
      account.value.type === "oauth"
    ) {
      meta.openAIResetCreditsAvailableCount = openAIResetCredits.value.count;
      meta.openAIResetCreditsKnown = openAIResetCredits.value.known;
      meta.openAIResetCreditsStatus = openAIResetCredits.value.status;
      meta.openAIResetCreditsUnsupportedReason = openAIResetCredits.value.unsupportedReason;
      meta.snapshotUpdatedAtText = shouldPreferFetchedOpenAIMeta.value
        ? fetchedSnapshotUpdatedAtText.value ||
          openAISnapshotUpdatedAtText.value ||
          undefined
        : openAISnapshotUpdatedAtText.value ||
          fetchedSnapshotUpdatedAtText.value ||
          undefined;
      meta.snapshotUpdatedAtTooltip = shouldPreferFetchedOpenAIMeta.value
        ? fetchedSnapshotUpdatedAtTooltip.value ||
          openAISnapshotUpdatedAtTooltip.value ||
          undefined
        : openAISnapshotUpdatedAtTooltip.value ||
          fetchedSnapshotUpdatedAtTooltip.value ||
          undefined;

      if (
        shouldUseFetchedOpenAIUsage.value &&
        openAIResolvedRows.value.length > 0
      ) {
        state = "bars";
        windowRows = openAIResolvedRows.value;
      } else if (
        isActiveOpenAIRateLimited.value &&
        currentState.loading &&
        openAIFetchedRows.value.length === 0
      ) {
        state = "loading";
      } else if (openAIResolvedRows.value.length > 0) {
        state = "bars";
        windowRows = openAIResolvedRows.value;
      } else if (currentState.loading) {
        state = "loading";
      } else if (openAIFetchedRows.value.length > 0) {
        state = "bars";
        windowRows = openAIFetchedRows.value;
      }
    } else if (
      getRuntimePlatform(account.value) === "anthropic" &&
      (account.value.type === "oauth" || account.value.type === "setup-token")
    ) {
      if (currentState.loading) {
        state = "loading";
      } else if (currentState.error) {
        state = "error";
      } else if (anthropicRows.value.length > 0) {
        state = "bars";
        windowRows = anthropicRows.value;
        meta.snapshotUpdatedAtText =
          fetchedSnapshotUpdatedAtText.value || undefined;
        meta.snapshotUpdatedAtTooltip =
          fetchedSnapshotUpdatedAtTooltip.value || undefined;
        if (usageInfo.value?.source === "passive") {
          meta.sampledBadgeLabel = t("admin.accounts.usageWindow.sampledBadge");
          meta.sampledBadgeTooltip = t(
            "admin.accounts.usageWindow.passiveSampled",
          );
        }
      }
    } else if (
      getRuntimePlatform(account.value) === "antigravity" &&
      account.value.type === "oauth"
    ) {
      meta.antigravityTierLabel = antigravityTierLabel.value;
      meta.antigravityTierClass = antigravityTierClass.value;
      meta.hasIneligibleTiers = hasIneligibleTiers.value;

      if (currentState.loading) {
        state = "loading";
      } else if (currentState.error) {
        state = "error";
      } else if (antigravityRows.value.length > 0) {
        state = "bars";
        windowRows = antigravityRows.value;
      }
    } else if (isProtocolGatewayGeminiAccount.value) {
      meta.protocolGatewayBadgeLabel = protocolGatewayBadgeLabel.value;
      meta.protocolGatewayBadgeClass = protocolGatewayBadgeClass.value;
      state = "unlimited";
    } else if (getRuntimePlatform(account.value) === "gemini") {
      meta.geminiAuthTypeLabel = geminiAuthTypeLabel.value;
      meta.geminiTierClass = geminiTierClass.value;
      meta.geminiQuotaPolicyChannel = geminiQuotaPolicyChannel.value;
      meta.geminiQuotaPolicyLimits = geminiQuotaPolicyLimits.value;
      meta.geminiQuotaPolicyDocsUrl = geminiQuotaPolicyDocsUrl.value;

      if (currentState.loading) {
        state = "loading";
      } else if (currentState.error) {
        state = "error";
      } else if (geminiRows.value.length > 0) {
        state = "bars";
        windowRows = geminiRows.value;
        meta.noteText =
          t("admin.accounts.gemini.quotaPolicy.simulatedNote") ||
          "Simulated quota";
      } else {
        state = "unlimited";
      }
    } else if (hasAccountQuota.value) {
      if (apiKeyQuotaRows.value.length > 0) {
        state = "bars";
        windowRows = apiKeyQuotaRows.value;
      }
    }

    const resetRows =
      state === "bars"
        ? windowRows
            .filter(
              (row) =>
                parseEffectiveResetAt(row.resetsAt, row.remainingSeconds) !==
                null,
            )
            .map((row) => ({
              key: row.key,
              label: row.label,
              resetsAt: row.resetsAt,
              remainingSeconds: row.remainingSeconds,
            }))
        : [];

    return {
      loading: currentState.loading,
      error: currentState.error,
      state,
      windowRows,
      resetRows,
      meta,
    };
  });

  return {
    presentation,
    loadUsage,
    requestAutoLoad,
    shouldFetchUsage,
  };
}
