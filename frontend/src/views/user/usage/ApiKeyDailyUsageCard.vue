<template>
  <div
    v-if="selectedApiKeyID"
    class="card mb-4 overflow-hidden"
    data-testid="api-key-daily-usage-card"
  >
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-6 py-4 dark:border-dark-800">
      <div>
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t("usage.apiKeyDailyUsage") }}
        </h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ selectedApiKeyName }}
        </p>
      </div>
      <span class="text-xs text-gray-500 dark:text-gray-400">
        {{ apiKeyDailyRangeLabel }}
      </span>
    </div>
    <div v-if="loading" class="px-6 py-6 text-sm text-gray-500 dark:text-gray-400">
      {{ t("common.loading") }}
    </div>
    <div v-else-if="rows.length === 0" class="px-6 py-6 text-sm text-gray-500 dark:text-gray-400">
      {{ t("usage.apiKeyDailyEmpty") }}
    </div>
    <div v-else class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 text-xs uppercase tracking-wider text-gray-500 dark:bg-dark-950 dark:text-gray-400">
          <tr>
            <th class="px-4 py-3 text-left">{{ t("usage.date") }}</th>
            <th class="px-4 py-3 text-right">{{ t("usage.requests") }}</th>
            <th class="px-4 py-3 text-right">{{ t("usage.inputTokens") }}</th>
            <th class="px-4 py-3 text-right">{{ t("usage.outputTokens") }}</th>
            <th class="px-4 py-3 text-right">{{ t("usage.cacheTokens") }}</th>
            <th class="px-4 py-3 text-right">{{ t("usage.cost") }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-dark-800">
          <tr v-for="row in rows" :key="row.date" data-testid="api-key-daily-usage-row">
            <td class="px-4 py-3 text-gray-900 dark:text-white">{{ row.date }}</td>
            <td class="px-4 py-3 text-right tabular-nums text-gray-700 dark:text-gray-200">
              {{ row.requests.toLocaleString() }}
            </td>
            <td class="px-4 py-3 text-right tabular-nums text-gray-700 dark:text-gray-200">
              {{ formatTokens(row.input_tokens) }}
            </td>
            <td class="px-4 py-3 text-right tabular-nums text-gray-700 dark:text-gray-200">
              {{ formatTokens(row.output_tokens) }}
            </td>
            <td class="px-4 py-3 text-right tabular-nums text-gray-700 dark:text-gray-200">
              {{ formatTokens((row.cache_creation_tokens || 0) + (row.cache_read_tokens || 0)) }}
            </td>
            <td class="px-4 py-3 text-right tabular-nums font-medium text-green-600 dark:text-green-400">
              {{ formatCurrencyBreakdown(undefined, row.actual_cost ?? row.cost) }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import type { TrendDataPoint } from "@/types";

defineProps<{
  selectedApiKeyID: number | null;
  selectedApiKeyName: string;
  apiKeyDailyRangeLabel: string;
  rows: TrendDataPoint[];
  loading: boolean;
  formatTokens: (value: number) => string;
  formatCurrencyBreakdown: (
    values: Record<string, number> | null | undefined,
    fallbackUSD: number | null | undefined,
    decimals?: number,
  ) => string;
}>();

const { t } = useI18n();
</script>
