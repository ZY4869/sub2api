<template>
  <div class="space-y-3">
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
      <div class="card p-3">
        <div class="flex items-center gap-2.5">
          <div class="rounded-md bg-blue-100 p-1.5 dark:bg-blue-900/30">
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
            <p class="text-lg font-bold leading-tight text-gray-900 dark:text-white">
              {{ stats?.total_requests?.toLocaleString() || "0" }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t("usage.inSelectedRange") }}
            </p>
          </div>
        </div>
      </div>

      <div class="card p-3">
        <div class="flex items-center gap-2.5">
          <div class="rounded-md bg-amber-100 p-1.5 dark:bg-amber-900/30">
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
            <p class="text-lg font-bold leading-tight text-gray-900 dark:text-white">
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

      <CacheStatsCard
        :cache-hit-rate="stats?.cache_hit_rate"
        :cache-creation-tokens="stats?.total_cache_creation_tokens"
        :cache-read-tokens="stats?.total_cache_read_tokens"
        :stats-card-style="statsCardStyle"
      />

      <div class="card p-3">
        <div class="flex items-center gap-2.5">
          <div class="rounded-md bg-green-100 p-1.5 dark:bg-green-900/30">
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
            <p class="text-lg font-bold leading-tight text-green-600 dark:text-green-400">
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

      <div class="card p-3">
        <div class="flex items-center gap-2.5">
          <div class="rounded-md bg-purple-100 p-1.5 dark:bg-purple-900/30">
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
            <p class="text-lg font-bold leading-tight text-gray-900 dark:text-white">
              {{ formatDuration(stats?.average_duration_ms || 0) }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ t("usage.perRequest") }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <div class="card border border-dashed border-primary-200/80 bg-primary-50/40 p-3 dark:border-primary-500/20 dark:bg-primary-500/5">
      <div class="mb-2 flex items-center justify-between gap-3">
        <div>
          <p class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t("usage.todayStats") }}
          </p>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t("usage.todaySoFar") }}
          </p>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-4 xl:grid-cols-6">
        <div class="rounded-lg bg-white/80 p-3 shadow-sm dark:bg-dark-900/70">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t("usage.todayRequests") }}
          </p>
          <p class="mt-1 text-lg font-bold leading-tight text-gray-900 dark:text-white">
            {{ stats?.today_requests?.toLocaleString() || "0" }}
          </p>
        </div>

        <div class="rounded-lg bg-white/80 p-3 shadow-sm dark:bg-dark-900/70 md:col-span-2 xl:col-span-3">
          <div class="mb-2 flex items-start justify-between gap-3">
            <div>
              <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t("usage.todayTokens") }}
              </p>
              <p class="mt-1 text-lg font-bold leading-tight text-gray-900 dark:text-white">
                {{ formatTokens(stats?.today_tokens || 0) }}
              </p>
            </div>
            <span class="rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
              {{ formatPercent(stats?.today_cache_hit_rate || 0) }}
            </span>
          </div>
          <div class="grid grid-cols-2 gap-1.5 lg:grid-cols-5">
            <div
              v-for="item in todayTokenItems"
              :key="item.key"
              class="rounded-md border border-gray-100 bg-gray-50/80 px-2.5 py-1.5 dark:border-dark-700 dark:bg-dark-800/70"
            >
              <p class="text-[11px] text-gray-500 dark:text-gray-400">{{ item.label }}</p>
              <p class="mt-1 text-sm font-semibold" :class="item.className">{{ item.value }}</p>
            </div>
          </div>
        </div>

        <div class="rounded-lg bg-white/80 p-3 shadow-sm dark:bg-dark-900/70">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t("usage.todayCost") }}
          </p>
          <p class="mt-1 text-lg font-bold leading-tight text-green-600 dark:text-green-400">
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

        <div class="rounded-lg bg-white/80 p-3 shadow-sm dark:bg-dark-900/70">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t("usage.todayAvgDuration") }}
          </p>
          <p class="mt-1 text-lg font-bold leading-tight text-gray-900 dark:text-white">
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
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { UsageStatsResponse } from "@/types";
import { useTokenDisplayMode } from "@/composables/useTokenDisplayMode";
import { formatUsageAmount } from "@/utils/usageCost";
import Icon from "@/components/icons/Icon.vue";
import CacheStatsCard from "@/components/usage/CacheStatsCard.vue";

const props = withDefaults(defineProps<{
  stats: UsageStatsResponse | null;
  statsCardStyle?: "balanced" | "accent";
}>(), {
  statsCardStyle: "balanced",
});

const { t } = useI18n();
const { formatTokenDisplay } = useTokenDisplayMode();

const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
};

const formatTokens = (value: number): string => formatTokenDisplay(value);

const formatPercent = (value: number): string => {
  const normalized = value <= 1 ? value * 100 : value;
  return `${normalized.toFixed(1)}%`;
};

const todayTokenItems = computed(() => [
  {
    key: "input",
    label: t("usage.inputTokens"),
    value: formatTokens(props.stats?.today_input_tokens || 0),
    className: "text-emerald-600 dark:text-emerald-400",
  },
  {
    key: "cache_write",
    label: t("usage.cacheCreationTokens"),
    value: formatTokens(props.stats?.today_cache_creation_tokens || 0),
    className: "text-amber-600 dark:text-amber-400",
  },
  {
    key: "cache_read",
    label: t("usage.cacheReadTokens"),
    value: formatTokens(props.stats?.today_cache_read_tokens || 0),
    className: "text-sky-600 dark:text-sky-400",
  },
  {
    key: "output",
    label: t("usage.outputTokens"),
    value: formatTokens(props.stats?.today_output_tokens || 0),
    className: "text-violet-600 dark:text-violet-400",
  },
  {
    key: "hit_rate",
    label: t("usage.cacheHitRate"),
    value: formatPercent(props.stats?.today_cache_hit_rate || 0),
    className: "text-teal-600 dark:text-teal-400",
  },
]);

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
