import type { MaybeRefOrGetter } from "vue";
import type {
  Account,
  AccountUsageInfo,
  AccountUsagePresentationRow,
  AccountUsageRowColor,
  WindowStats,
} from "@/types";

export interface UsageCacheEntry {
  loading: boolean;
  error: string | null;
  usageInfo: AccountUsageInfo | null;
  preferOpenAIFetchedUsage: boolean;
  request: Promise<void> | null;
}

export interface UsageRowOptions {
  windowStats?: WindowStats | null;
  remainingSeconds?: number | null;
  inlineRemaining?: boolean;
  detailedReset?: boolean;
}

export interface LoadUsageOptions {
  force?: boolean;
  source?: "passive" | "active";
  queue?: boolean;
}

export interface RefreshUsageOptions extends LoadUsageOptions {
  concurrency?: number;
  resolveLoadOptions?: (account: Account) => LoadUsageOptions | undefined;
}

export interface RefreshUsageResult {
  total: number;
  success: number;
  activeSuccess: number;
  fallbackSuccess: number;
  failed: number;
}

export interface UseAccountUsagePresentationOptions {
  autoLoadEnabled?: MaybeRefOrGetter<boolean>;
}

export type OpenAIUsageScope = "normal" | "spark";

export type OpenAIUsageRowSpec = {
  key: AccountUsagePresentationRow["key"];
  scope: OpenAIUsageScope;
  window: "5h" | "7d";
  color: AccountUsageRowColor;
};
