<template>
  <div class="card p-4" data-testid="usage-cache-stats-card">
    <div class="flex items-center gap-3">
      <div class="rounded-lg bg-teal-100 p-2 dark:bg-teal-900/30">
        <Icon
          name="sync"
          size="md"
          class="text-teal-600 dark:text-teal-400"
          :stroke-width="2"
        />
      </div>
      <div class="min-w-0">
        <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
          {{ t("usage.cacheHitRate") }}
        </p>
        <p class="text-xl font-bold text-teal-600 dark:text-teal-400">
          {{ formatPercent(cacheHitRate) }}
        </p>
        <div class="mt-1 grid grid-cols-3 gap-2 text-[11px] text-gray-500 dark:text-gray-400">
          <div>
            <span class="block">{{ t("usage.cacheWrite") }}</span>
            <span class="font-semibold text-gray-800 dark:text-gray-100">{{ formatTokens(cacheCreationTokens) }}</span>
          </div>
          <div>
            <span class="block">{{ t("usage.cacheRead") }}</span>
            <span class="font-semibold text-gray-800 dark:text-gray-100">{{ formatTokens(cacheReadTokens) }}</span>
          </div>
          <div>
            <span class="block">{{ t("common.total") }}</span>
            <span class="font-semibold text-gray-800 dark:text-gray-100">{{ formatTokens(cacheTotalTokens) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import Icon from "@/components/icons/Icon.vue";
import { useTokenDisplayMode } from "@/composables/useTokenDisplayMode";

const props = defineProps<{
  cacheHitRate?: number | null;
  cacheCreationTokens?: number | null;
  cacheReadTokens?: number | null;
}>();

const { t } = useI18n();
const { formatTokenDisplay } = useTokenDisplayMode();

const formatTokens = (value: number | null | undefined): string =>
  formatTokenDisplay(value || 0);

const cacheTotalTokens = computed(
  () => (props.cacheCreationTokens || 0) + (props.cacheReadTokens || 0),
);

const formatPercent = (value: number | null | undefined): string => {
  const numeric = Number.isFinite(Number(value)) ? Number(value) : 0;
  const normalized = numeric <= 1 ? numeric * 100 : numeric;
  return `${normalized.toFixed(1)}%`;
};
</script>
