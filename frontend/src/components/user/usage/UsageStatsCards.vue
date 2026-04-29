<template>
  <div class="space-y-4">
    <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
      <div class="card p-4">
        <div class="flex items-center gap-3">
          <div class="rounded-lg bg-blue-100 p-2 dark:bg-blue-900/30">
            <Icon
              name="document"
              size="md"
              class="text-blue-600 dark:text-blue-400"
            />
          </div>
          <div>
            <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t("usage.totalRequests") }}
            </p>
            <p class="text-xl font-bold text-gray-900 dark:text-white">
              {{ stats?.total_requests?.toLocaleString() || "0" }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t("usage.inSelectedRange") }}
            </p>
          </div>
        </div>
      </div>

      <div class="card p-4">
        <div class="flex items-center gap-3">
          <div class="rounded-lg bg-amber-100 p-2 dark:bg-amber-900/30">
            <Icon
              name="cube"
              size="md"
              class="text-amber-600 dark:text-amber-400"
            />
          </div>
          <div>
            <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t("usage.totalTokens") }}
            </p>
            <p class="text-xl font-bold text-gray-900 dark:text-white">
              {{ formatTokens(stats?.total_tokens || 0) }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t("usage.in") }}:
              {{ formatTokens(stats?.total_input_tokens || 0) }} /
              {{ t("usage.out") }}:
              {{ formatTokens(stats?.total_output_tokens || 0) }}
            </p>
          </div>
        </div>
      </div>

      <div class="card p-4">
        <div class="flex items-center gap-3">
          <div class="rounded-lg bg-green-100 p-2 dark:bg-green-900/30">
            <Icon
              name="dollar"
              size="md"
              class="text-green-600 dark:text-green-400"
            />
          </div>
          <div class="min-w-0 flex-1">
            <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t("usage.totalCost") }}
            </p>
            <p class="text-xl font-bold text-green-600 dark:text-green-400">
              {{ formatCurrencyBreakdown(stats?.actual_cost_by_currency, stats?.total_actual_cost, 4) }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t("usage.actualCost") }} /
              <span class="line-through">
                {{ formatCurrencyBreakdown(stats?.cost_by_currency, stats?.total_cost, 4) }}
              </span>
              {{ t("usage.standardCost") }}
            </p>
            <p
              v-if="stats?.admin_free_requests"
              class="mt-1 text-[11px] text-emerald-500 dark:text-emerald-300"
            >
              管理员免扣
              {{ stats.admin_free_requests.toLocaleString() }} 次 / ${{
                formatUsageAmount(stats.admin_free_standard_cost, 4)
              }}
              标准成本
            </p>
          </div>
        </div>
      </div>

      <div class="card p-4">
        <div class="flex items-center gap-3">
          <div class="rounded-lg bg-purple-100 p-2 dark:bg-purple-900/30">
            <Icon
              name="clock"
              size="md"
              class="text-purple-600 dark:text-purple-400"
            />
          </div>
          <div>
            <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ t("usage.avgDuration") }}
            </p>
            <p class="text-xl font-bold text-gray-900 dark:text-white">
              {{ formatDuration(stats?.average_duration_ms || 0) }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t("usage.perRequest") }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <div class="card border border-dashed border-primary-200/80 bg-primary-50/40 p-4 dark:border-primary-500/20 dark:bg-primary-500/5">
      <div class="mb-3 flex items-center justify-between gap-3">
        <div>
          <p class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t("usage.todayStats") }}
          </p>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t("usage.todaySoFar") }}
          </p>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-xl bg-white/80 p-4 shadow-sm dark:bg-dark-900/70">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t("usage.todayRequests") }}
          </p>
          <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">
            {{ stats?.today_requests?.toLocaleString() || "0" }}
          </p>
        </div>

        <div class="rounded-xl bg-white/80 p-4 shadow-sm dark:bg-dark-900/70">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t("usage.todayTokens") }}
          </p>
          <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">
            {{ formatTokens(stats?.today_tokens || 0) }}
          </p>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t("usage.in") }}:
            {{ formatTokens(stats?.today_input_tokens || 0) }} /
            {{ t("usage.cacheTokens") }}:
            {{ formatTokens(stats?.today_cache_tokens || 0) }} /
            {{ t("usage.out") }}:
            {{ formatTokens(stats?.today_output_tokens || 0) }}
          </p>
        </div>

        <div class="rounded-xl bg-white/80 p-4 shadow-sm dark:bg-dark-900/70">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t("usage.todayCost") }}
          </p>
          <p class="mt-1 text-xl font-bold text-green-600 dark:text-green-400">
            {{ formatCurrencyBreakdown(stats?.today_actual_cost_by_currency, stats?.today_actual_cost, 4) }}
          </p>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t("usage.actualCost") }} /
            <span class="line-through">
              {{ formatCurrencyBreakdown(stats?.today_cost_by_currency, stats?.today_cost, 4) }}
            </span>
            {{ t("usage.standardCost") }}
          </p>
        </div>

        <div class="rounded-xl bg-white/80 p-4 shadow-sm dark:bg-dark-900/70">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t("usage.todayAvgDuration") }}
          </p>
          <p class="mt-1 text-xl font-bold text-gray-900 dark:text-white">
            {{ formatDuration(stats?.today_average_duration_ms || 0) }}
          </p>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t("usage.perRequest") }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import type { UsageStatsResponse } from "@/types";
import { useTokenDisplayMode } from "@/composables/useTokenDisplayMode";
import { formatUsageAmount } from "@/utils/usageCost";
import Icon from "@/components/icons/Icon.vue";

defineProps<{
  stats: UsageStatsResponse | null;
}>();

const { t } = useI18n();
const { formatTokenDisplay } = useTokenDisplayMode();

const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
};

const formatTokens = (value: number): string => formatTokenDisplay(value);

const formatCurrencyBreakdown = (
  values: Record<string, number> | null | undefined,
  fallbackUSD: number | null | undefined,
  decimals = 4,
) => {
  const entries = Object.entries(values || {})
    .filter(([, value]) => Number.isFinite(value))
    .sort(([left], [right]) => left.localeCompare(right));
  if (entries.length === 0) {
    return `$${formatUsageAmount(fallbackUSD, decimals)}`;
  }
  return entries
    .map(([currency, value]) => {
      const normalized = currency.toUpperCase();
      return `${normalized === "CNY" ? "¥" : "$"}${formatUsageAmount(value, decimals)}`;
    })
    .join(" / ");
};
</script>
