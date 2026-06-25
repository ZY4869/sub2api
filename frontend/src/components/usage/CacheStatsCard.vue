<template>
  <div
    class="card p-3"
    :class="styleClass"
    data-testid="usage-cache-stats-card"
  >
    <div class="flex items-center gap-2.5">
      <div class="rounded-md bg-teal-100 p-1.5 text-teal-600 dark:bg-teal-900/30 dark:text-teal-400">
        <Icon
          name="sync"
          size="md"
          :stroke-width="2"
        />
      </div>
      <div class="min-w-0 flex-1">
        <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
          {{ t("usage.cacheHitRate") }}
        </p>
        <div class="mt-0.5 flex items-baseline justify-between gap-3">
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ t("usage.cacheHitRate") }}</span>
          <span class="text-lg font-bold text-teal-600 dark:text-teal-400">
            {{ formatPercent(cacheHitRate) }}
          </span>
        </div>
        <div class="mt-1.5 grid grid-cols-3 gap-1.5 text-[11px] text-gray-500 dark:text-gray-400">
          <div>
            <span class="block">{{ t("usage.cacheWrite") }}</span>
            <span class="font-semibold text-amber-600 dark:text-amber-400">{{ formatTokens(cacheCreationTokens) }}</span>
          </div>
          <div>
            <span class="block">{{ t("usage.cacheRead") }}</span>
            <span class="font-semibold text-sky-600 dark:text-sky-400">{{ formatTokens(cacheReadTokens) }}</span>
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
  statsCardStyle?: "balanced" | "accent";
}>();

const { t } = useI18n();
const { formatTokenDisplay } = useTokenDisplayMode();

const formatTokens = (value: number | null | undefined): string =>
  formatTokenDisplay(value || 0);

const cacheTotalTokens = computed(
  () => (props.cacheCreationTokens || 0) + (props.cacheReadTokens || 0),
);

const styleClass = computed(() =>
  props.statsCardStyle === "accent"
    ? "border-teal-200/80 bg-teal-50/40 dark:border-teal-500/20 dark:bg-teal-500/5"
    : "",
);

const formatPercent = (value: number | null | undefined): string => {
  const numeric = Number.isFinite(Number(value)) ? Number(value) : 0;
  const normalized = numeric <= 1 ? numeric * 100 : numeric;
  return `${normalized.toFixed(1)}%`;
};
</script>
