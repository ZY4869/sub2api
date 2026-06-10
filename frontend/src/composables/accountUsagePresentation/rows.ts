import type {
  Account,
  AccountUsageInfo,
  AccountUsagePresentationMeta,
  AccountUsagePresentationRow,
  AccountUsageRowColor,
  UsageProgress,
} from "@/types";
import { resolveCodexUsageWindowLabel } from "@/utils/codexUsage";
import { parseEffectiveResetAt } from "@/utils/usageResetTime";
import type { OpenAIUsageRowSpec, UsageRowOptions } from "./types";

type Translate = (key: string) => string;

const OPENAI_USAGE_ROW_SPECS: OpenAIUsageRowSpec[] = [
  { key: "openai-5h", scope: "normal", window: "5h", color: "indigo" },
  { key: "openai-7d", scope: "normal", window: "7d", color: "orange" },
  { key: "openai-spark-5h", scope: "spark", window: "5h", color: "indigo" },
  { key: "openai-spark-7d", scope: "spark", window: "7d", color: "orange" },
];

export function buildUsageRow(
  key: string,
  label: string,
  utilization: number,
  resetsAt: string | null,
  color: AccountUsageRowColor,
  options: UsageRowOptions = {},
): AccountUsagePresentationRow {
  const normalizedResetAt =
    typeof resetsAt === "string" && resetsAt.trim() !== "" ? resetsAt : null;
  const effectiveResetAt =
    normalizedResetAt === null && (options.remainingSeconds ?? 0) > 0
      ? parseEffectiveResetAt(null, options.remainingSeconds ?? null)
      : null;

  return {
    key,
    label,
    utilization,
    resetsAt:
      normalizedResetAt ??
      (effectiveResetAt ? effectiveResetAt.toISOString() : null),
    remainingSeconds: options.remainingSeconds ?? null,
    windowStats: options.windowStats ?? null,
    color,
    inlineRemaining: options.inlineRemaining ?? false,
    detailedReset: options.detailedReset ?? false,
  };
}

export function buildProgressRow(
  key: string,
  label: string,
  progress: UsageProgress | null | undefined,
  color: AccountUsageRowColor,
  options: UsageRowOptions = {},
): AccountUsagePresentationRow | null {
  if (!progress) return null;

  return buildUsageRow(
    key,
    label,
    progress.utilization,
    progress.resets_at,
    color,
    {
      windowStats: progress.window_stats,
      remainingSeconds: progress.remaining_seconds,
      inlineRemaining: options.inlineRemaining,
      detailedReset: options.detailedReset,
    },
  );
}

export function buildRows(
  ...rows: Array<AccountUsagePresentationRow | null>
): AccountUsagePresentationRow[] {
  return rows.filter((row): row is AccountUsagePresentationRow => row !== null);
}

function findRowByKey(
  rows: AccountUsagePresentationRow[],
  key: AccountUsagePresentationRow["key"],
): AccountUsagePresentationRow | null {
  return rows.find((row) => row.key === key) ?? null;
}

export function isOpenAIProAccount(account: Account): boolean {
  return (
    String(account.credentials?.plan_type || "")
      .trim()
      .toLowerCase() === "pro"
  );
}

export function isOpenAIFreeAccount(account: Account): boolean {
  return (
    String(account.credentials?.plan_type || "")
      .trim()
      .toLowerCase() === "free"
  );
}

export function resolveOpenAIUsageSpecs(
  includeSpark: boolean,
  freeOnly7d = false,
): OpenAIUsageRowSpec[] {
  const allowedRows = includeSpark
    ? OPENAI_USAGE_ROW_SPECS
    : OPENAI_USAGE_ROW_SPECS.filter((spec) => spec.scope === "normal");
  return freeOnly7d
    ? allowedRows.filter((spec) => spec.scope === "normal" && spec.window === "7d")
    : allowedRows;
}

export function resolveOpenAIUsageLabel(
  spec: OpenAIUsageRowSpec,
  _t: Translate,
): string {
  if (spec.scope === "spark") {
    return `Spark ${resolveCodexUsageWindowLabel(null, spec.window)}`;
  }
  return resolveCodexUsageWindowLabel(null, spec.window);
}

export function resolveOpenAIUsageProgress(
  usage: AccountUsageInfo | null | undefined,
  spec: OpenAIUsageRowSpec,
): UsageProgress | null | undefined {
  if (spec.scope === "spark") {
    return spec.window === "5h"
      ? usage?.spark_five_hour
      : usage?.spark_seven_day;
  }
  return spec.window === "5h" ? usage?.five_hour : usage?.seven_day;
}

export function mergeOpenAIUsageRows(
  primaryRows: AccountUsagePresentationRow[],
  fallbackRows: AccountUsagePresentationRow[],
  includeSpark: boolean,
  freeOnly7d: boolean,
  fillMissingRows: boolean,
  t: Translate,
): AccountUsagePresentationRow[] {
  const specs = resolveOpenAIUsageSpecs(includeSpark, freeOnly7d);
  const hasResolvedRows = specs.some((spec) => {
    return (
      findRowByKey(primaryRows, spec.key) !== null ||
      findRowByKey(fallbackRows, spec.key) !== null
    );
  });

  return buildRows(
    ...specs.map(
      (spec) =>
        findRowByKey(primaryRows, spec.key) ??
        findRowByKey(fallbackRows, spec.key) ??
        (fillMissingRows && hasResolvedRows
          ? buildUsageRow(
              spec.key,
              resolveOpenAIUsageLabel(spec, t),
              0,
              null,
              spec.color,
              {
                inlineRemaining: true,
                detailedReset: true,
              },
            )
          : null),
    ),
  );
}

export function createEmptyMeta(
  loadingRows: number,
): AccountUsagePresentationMeta {
  return { loadingRows };
}
