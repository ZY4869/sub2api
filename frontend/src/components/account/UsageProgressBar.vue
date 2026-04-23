<template>
  <div :class="detailedReset ? 'space-y-0.5' : ''">
    <div class="flex min-w-0 items-center gap-1">
      <span
        :class="[
          'shrink-0 rounded text-center text-[10px] font-medium',
          labelWidthClass,
          labelClass,
        ]"
      >
        {{ label }}
      </span>

      <div
        class="relative"
        data-testid="usage-progress-trigger"
        :tabindex="windowStats ? 0 : undefined"
        @mouseenter="showStatsTooltip"
        @mouseleave="hideStatsTooltip"
        @focusin="showStatsTooltip"
        @focusout="hideStatsTooltip"
      >
        <div
          class="h-1.5 w-8 shrink-0 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"
        >
          <div
            data-testid="usage-progress-fill"
            :class="['h-full transition-all duration-300', barClass]"
            :style="{ width: barWidth }"
          ></div>
        </div>
        <div
          v-if="windowStats && statsTooltipVisible"
          data-testid="usage-progress-tooltip"
          class="absolute left-1/2 top-full z-20 mt-2 min-w-max -translate-x-1/2 rounded-lg border border-gray-200 bg-white px-2.5 py-2 text-[10px] text-gray-700 shadow-lg dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200"
        >
          <div class="flex items-center gap-1.5">
            <span class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-700"
              >{{ formatRequests }} req</span
            >
            <span
              class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-700"
              :title="rawTokenCount || undefined"
            >
              {{ formatTokens }}
            </span>
            <span class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-700"
              >A ${{ formatAccountCost }}</span
            >
            <span
              v-if="windowStats?.user_cost != null"
              class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-700"
            >
              U ${{ formatUserCost }}
            </span>
          </div>
        </div>
      </div>

      <span
        :class="[
          'w-[32px] shrink-0 text-right text-[10px] font-medium',
          textClass,
        ]"
      >
        {{ displayPercent }}
      </span>

      <span
        v-if="inlineReset && !detailedReset && effectiveResetAt"
        :class="remainingTextClass"
      >
        {{ t("admin.accounts.usageWindow.remainingLabel") }}
        {{ resetCountdownText }}
      </span>

      <span
        v-else-if="!detailedReset && effectiveResetAt"
        class="shrink-0 text-[10px] text-gray-400"
      >
        {{ compactResetText }}
      </span>
    </div>

    <div
      v-if="detailedReset"
      :class="[
        detailPaddingClass,
        'flex items-center gap-1 text-[10px] text-gray-400',
      ]"
      :title="resetTooltip || undefined"
    >
      <span
        >{{ t("admin.accounts.usageWindow.remainingLabel") }}
        {{ resetCountdownText }}</span
      >
      <span
        >{{ t("admin.accounts.usageWindow.resetAtLabel") }}
        {{ resetAbsoluteText }}</span
      >
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useUiNow } from "@/composables/useUiNow";
import { useTokenDisplayMode } from "@/composables/useTokenDisplayMode";
import type { AccountUsageDisplayMode, WindowStats } from "@/types";
import {
  formatLocalAbsoluteTime,
  formatLocalTimestamp,
  formatResetCountdown,
  parseEffectiveResetAt,
} from "@/utils/usageResetTime";

const props = withDefaults(
  defineProps<{
    label: string;
    utilization: number;
    resetsAt?: string | null;
    remainingSeconds?: number | null;
    color: "indigo" | "emerald" | "purple" | "amber";
    windowStats?: WindowStats | null;
    detailedReset?: boolean;
    inlineReset?: boolean;
    displayMode?: AccountUsageDisplayMode;
  }>(),
  {
    displayMode: "used",
  },
);

const { t } = useI18n();
const { nowDate } = useUiNow();
const { formatTokenDisplay } = useTokenDisplayMode();
const statsTooltipVisible = ref(false);

const labelClass = computed(() => {
  const colors = {
    indigo:
      "bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300",
    emerald:
      "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300",
    purple:
      "bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300",
    amber:
      "bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300",
  };
  return colors[props.color];
});

const labelWidthClass = computed(() => {
  return props.detailedReset
    ? "min-w-[56px] px-1.5 py-0.5"
    : "w-[32px] px-1 py-0";
});

const detailPaddingClass = computed(() => {
  return props.detailedReset ? "pl-[61px]" : "pl-[37px]";
});

const barClass = computed(() => {
  if (props.utilization >= 100) return "bg-red-500";
  if (props.utilization >= 80) return "bg-amber-500";
  return "bg-green-500";
});

const textClass = computed(() => {
  if (props.utilization >= 100) return "text-red-600 dark:text-red-400";
  if (props.utilization >= 80) return "text-amber-600 dark:text-amber-400";
  return "text-gray-600 dark:text-gray-400";
});

const remainingTextClass = computed(() => {
  return "shrink-0 text-[10px] font-semibold text-amber-700 dark:text-amber-300";
});

const resolvedBarPercent = computed(() => {
  const usedPercent = Math.max(0, Math.min(props.utilization, 100));
  return props.displayMode === "remaining" ? 100 - usedPercent : usedPercent;
});

const barWidth = computed(() => `${resolvedBarPercent.value}%`);

const displayPercent = computed(() => {
  const percent = Math.round(props.utilization);
  return percent > 999 ? ">999%" : `${percent}%`;
});

const effectiveResetAt = computed(() =>
  parseEffectiveResetAt(props.resetsAt ?? null, props.remainingSeconds ?? null),
);

const compactResetText = computed(() => {
  if (!effectiveResetAt.value) return "-";
  return formatResetCountdown(
    effectiveResetAt.value,
    nowDate.value,
    t("admin.accounts.usageWindow.now"),
  );
});

const resetCountdownText = computed(() => {
  if (!effectiveResetAt.value) return "-";
  return formatResetCountdown(
    effectiveResetAt.value,
    nowDate.value,
    t("admin.accounts.usageWindow.now"),
  );
});

const resetAbsoluteText = computed(() => {
  if (!effectiveResetAt.value) return "-";
  return formatLocalAbsoluteTime(effectiveResetAt.value, nowDate.value, {
    today: t("dates.today"),
    tomorrow: t("dates.tomorrow"),
  });
});

const resetTooltip = computed(() => {
  if (!effectiveResetAt.value) return "";
  return formatLocalTimestamp(effectiveResetAt.value);
});

const formatRequests = computed(() => {
  if (!props.windowStats) return "";
  const requests = props.windowStats.requests;
  if (requests >= 1000000) return `${(requests / 1000000).toFixed(1)}M`;
  if (requests >= 1000) return `${(requests / 1000).toFixed(1)}K`;
  return requests.toString();
});

const formatTokens = computed(() => {
  if (!props.windowStats) return "";
  return formatTokenDisplay(props.windowStats.tokens);
});

const rawTokenCount = computed(() => {
  return props.windowStats ? props.windowStats.tokens.toLocaleString() : "";
});

const formatAccountCost = computed(() => {
  if (!props.windowStats) return "0.00";
  return props.windowStats.cost.toFixed(2);
});

const formatUserCost = computed(() => {
  if (!props.windowStats || props.windowStats.user_cost == null) return "0.00";
  return props.windowStats.user_cost.toFixed(2);
});

const showStatsTooltip = () => {
  if (!props.windowStats) return;
  statsTooltipVisible.value = true;
};

const hideStatsTooltip = () => {
  statsTooltipVisible.value = false;
};
</script>
