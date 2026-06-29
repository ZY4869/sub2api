<template>
  <div :class="[detailedReset ? 'space-y-0.5' : '', visualVariant === 'glass' ? 'usage-progress-glass' : '']">
    <div class="flex min-w-0 items-center gap-1">
      <span
        :title="label"
        :class="[
          'shrink-0 truncate rounded text-center text-[10px] font-medium',
          labelWidthClass,
          labelClass,
        ]"
      >
        {{ label }}
      </span>

      <div
        class="relative shrink-0"
        data-testid="usage-progress-trigger"
        :tabindex="windowStats ? 0 : undefined"
        @mouseenter="showStatsTooltip"
        @mouseleave="hideStatsTooltip"
        @focusin="showStatsTooltip"
        @focusout="hideStatsTooltip"
      >
        <div
          :class="trackClass"
        >
          <div
            data-testid="usage-progress-fill"
            :class="['h-full transition-[width] duration-300', barClass]"
            :style="{ width: barWidth }"
          ></div>
        </div>
        <div
          v-if="windowStats && statsTooltipVisible"
          data-testid="usage-progress-tooltip"
          :class="tooltipClass"
        >
          <div class="flex items-center gap-1.5">
            <span :class="tooltipChipClass"
              >{{ formatRequests }} req</span
            >
            <span
              :class="tooltipChipClass"
              :title="rawTokenCount || undefined"
            >
              {{ formatTokens }}
            </span>
            <span :class="tooltipChipClass"
              >A ${{ formatAccountCost }}</span
            >
            <span
              v-if="windowStats?.user_cost != null"
              :class="tooltipChipClass"
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
        :class="resetTextClass"
      >
        {{ compactResetText }}
      </span>
    </div>

    <div
      v-if="detailedReset"
      :class="[
        detailPaddingClass,
        detailedResetClass,
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
import { useRealtimeCountdownNow } from "@/composables/useRealtimeCountdownNow";
import { useTokenDisplayMode } from "@/composables/useTokenDisplayMode";
import type { AccountUsageDisplayMode, WindowStats } from "@/types";
import {
  formatLocalAbsoluteTime,
  formatLocalTimestamp,
  formatResetCountdown,
  parseEffectiveResetAt,
} from "@/utils/usageResetTime";
import { resolveUsageWindowCapsuleClass } from "@/utils/accountUsageWindowDisplay";

const props = withDefaults(
  defineProps<{
    label: string;
    utilization: number;
    resetsAt?: string | null;
    remainingSeconds?: number | null;
    remainingAnchorMs?: number | null;
    color: "indigo" | "emerald" | "purple" | "amber" | "orange" | "green";
    windowStats?: WindowStats | null;
    detailedReset?: boolean;
    inlineReset?: boolean;
    displayMode?: AccountUsageDisplayMode;
    visualVariant?: "default" | "glass";
  }>(),
  {
    displayMode: "used",
    visualVariant: "default",
  },
);

const { t } = useI18n();
const { nowDate } = useRealtimeCountdownNow("accounts");
const { formatTokenDisplay } = useTokenDisplayMode();
const statsTooltipVisible = ref(false);
const fallbackRemainingAnchorMs = Date.now();

const labelClass = computed(() => {
  const capsuleClass = resolveUsageWindowCapsuleClass(props.label);
  if (!capsuleClass.includes("slate")) return capsuleClass;
  const glassColors = {
    indigo:
      "border border-indigo-200/70 bg-indigo-50 text-indigo-700 dark:border-indigo-400/20 dark:bg-indigo-400/10 dark:text-indigo-100",
    emerald:
      "border border-emerald-200/70 bg-emerald-50 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-400/10 dark:text-emerald-100",
    purple:
      "border border-violet-200/70 bg-violet-50 text-violet-700 dark:border-violet-400/20 dark:bg-violet-400/10 dark:text-violet-100",
    amber:
      "border border-amber-200/70 bg-amber-50 text-amber-700 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-100",
    orange:
      "border border-orange-200/70 bg-orange-50 text-orange-700 dark:border-orange-400/20 dark:bg-orange-400/10 dark:text-orange-100",
    green:
      "border border-green-200/70 bg-green-50 text-green-700 dark:border-green-400/20 dark:bg-green-400/10 dark:text-green-100",
  };
  const colors = {
    indigo:
      "bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300",
    emerald:
      "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300",
    purple:
      "bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300",
    amber:
      "bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300",
    orange:
      "bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-300",
    green:
      "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300",
  };
  return props.visualVariant === "glass" ? glassColors[props.color] : colors[props.color];
});

const labelWidthClass = computed(() => {
  return props.detailedReset
    ? "min-w-[56px] max-w-[88px] px-1.5 py-0.5"
    : "min-w-[32px] max-w-[64px] px-1 py-0";
});

const detailPaddingClass = computed(() => {
  return props.detailedReset ? "pl-[61px]" : "pl-[37px]";
});

const barClass = computed(() => {
  if (props.visualVariant === "glass") {
    if (props.utilization >= 100) {
      return "bg-gradient-to-r from-rose-500 to-rose-600";
    }
    if (props.utilization >= 80) {
      return "bg-gradient-to-r from-orange-400 to-orange-500";
    }
    if (props.utilization >= 50) {
      return "bg-gradient-to-r from-amber-300 to-amber-400";
    }
    if (props.utilization >= 25) {
      return "bg-gradient-to-r from-emerald-300 to-emerald-400";
    }
    return "bg-gradient-to-r from-emerald-200 to-emerald-300";
  }
  if (props.utilization >= 100) return "bg-red-500";
  if (props.utilization >= 80) return "bg-amber-500";
  return "bg-green-500";
});

const textClass = computed(() => {
  if (props.visualVariant === "glass") {
    if (props.utilization >= 100) return "text-rose-800 dark:text-rose-100";
    if (props.utilization >= 80) return "text-orange-700 dark:text-orange-100";
    if (props.utilization >= 50) return "text-amber-700 dark:text-amber-100";
    return "text-slate-600 dark:text-slate-200";
  }
  if (props.utilization >= 100) return "text-red-600 dark:text-red-400";
  if (props.utilization >= 80) return "text-amber-600 dark:text-amber-400";
  return "text-gray-600 dark:text-gray-400";
});

const remainingTextClass = computed(() => {
  if (props.visualVariant === "glass") {
    return "shrink-0 text-[10px] font-semibold text-amber-700 dark:text-amber-100";
  }
  return "shrink-0 text-[10px] font-semibold text-amber-700 dark:text-amber-300";
});

const trackClass = computed(() => {
  if (props.visualVariant === "glass") {
    return "h-1.5 w-16 shrink-0 overflow-hidden rounded-full border border-slate-200/75 bg-slate-100 dark:border-slate-700/80 dark:bg-slate-800/70";
  }
  return "h-1.5 w-8 shrink-0 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700";
});

const tooltipClass = computed(() => {
  if (props.visualVariant === "glass") {
    return "absolute left-1/2 top-full z-20 mt-2 min-w-max -translate-x-1/2 rounded-2xl border border-slate-200/80 bg-white px-2.5 py-2 text-[10px] text-slate-700 ring-1 ring-slate-200/60 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:ring-slate-700/70";
  }
  return "absolute left-1/2 top-full z-20 mt-2 min-w-max -translate-x-1/2 rounded-lg border border-gray-200 bg-white px-2.5 py-2 text-[10px] text-gray-700 shadow-lg dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200";
});

const tooltipChipClass = computed(() => {
  if (props.visualVariant === "glass") {
    return "rounded-full border border-slate-200/75 bg-slate-50 px-1.5 py-0.5 dark:border-slate-700/80 dark:bg-slate-800/70";
  }
  return "rounded bg-gray-100 px-1.5 py-0.5 dark:bg-gray-700";
});

const resetTextClass = computed(() => {
  if (props.visualVariant === "glass") {
    return "shrink-0 text-[10px] font-medium text-slate-500 dark:text-slate-300";
  }
  return "shrink-0 text-[10px] text-gray-400";
});

const detailedResetClass = computed(() => {
  if (props.visualVariant === "glass") {
    return "flex items-center gap-1 text-[10px] font-medium text-slate-500 dark:text-slate-300";
  }
  return "flex items-center gap-1 text-[10px] text-gray-400";
});

const resolvedBarPercent = computed(() => {
  const usedPercent = Math.max(0, Math.min(props.utilization, 100));
  return props.displayMode === "remaining" ? 100 - usedPercent : usedPercent;
});

const barWidth = computed(() => `${resolvedBarPercent.value}%`);

const displayPercent = computed(() => {
  if (props.displayMode === "remaining") {
    return `${Math.round(resolvedBarPercent.value)}%`;
  }
  const percent = Math.round(props.utilization);
  return percent > 999 ? ">999%" : `${percent}%`;
});

const hasAbsoluteResetAt = computed(
  () => typeof props.resetsAt === "string" && props.resetsAt.trim() !== "",
);

const effectiveResetAt = computed(() => {
  const anchorMs = props.remainingAnchorMs;
  const baseTime =
    typeof anchorMs === "number" && Number.isFinite(anchorMs)
      ? new Date(anchorMs)
      : new Date(fallbackRemainingAnchorMs);
  return parseEffectiveResetAt(
    props.resetsAt ?? null,
    props.remainingSeconds ?? null,
    baseTime,
  );
});

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
  if (!effectiveResetAt.value || !hasAbsoluteResetAt.value) return "-";
  return formatLocalAbsoluteTime(effectiveResetAt.value, nowDate.value, {
    today: t("dates.today"),
    tomorrow: t("dates.tomorrow"),
  });
});

const resetTooltip = computed(() => {
  if (!effectiveResetAt.value || !hasAbsoluteResetAt.value) return "";
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
