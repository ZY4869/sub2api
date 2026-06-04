<template>
  <div>
    <div v-if="props.loading && !props.stats" class="w-[120px] space-y-1">
      <div class="h-3 w-36 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
      <div class="h-3 w-32 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
      <div class="h-3 w-40 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
    </div>

    <div v-else-if="props.error && !props.stats" class="text-xs text-red-500">
      {{ props.error }}
    </div>

    <div
      v-else-if="props.stats"
      :class="rootClass"
      data-testid="account-today-stats-cell"
    >
      <div class="grid gap-1" :class="statsGridClass">
        <div
          v-for="item in statColumns"
          :key="item.key"
          class="min-w-0 rounded-lg border border-slate-200/80 bg-white/80 px-2 py-1.5 shadow-[0_4px_12px_rgba(15,23,42,0.04)] dark:border-slate-700/70 dark:bg-slate-900/50"
          :title="item.title"
        >
          <div class="truncate font-sans text-[9px] font-semibold text-slate-400 dark:text-slate-500">
            {{ item.label }}
          </div>
          <div class="mt-1 truncate text-xs font-bold text-slate-800 dark:text-slate-100">
            {{ item.requests }}
          </div>
          <div class="mt-0.5 truncate text-[10px] font-bold" :class="item.costClass">
            {{ item.cost }}
          </div>
        </div>
      </div>

      <div
        v-if="showTodayQualityFooter"
        class="mt-0.5 flex min-w-0 flex-col gap-1 border-t border-slate-100 pt-1.5 dark:border-slate-700/70"
      >
        <span
          class="flex min-w-0 flex-wrap items-center gap-x-1 gap-y-0.5 font-sans leading-tight text-slate-400 dark:text-slate-500"
          :title="footerStatsTitle"
        >
          <span class="truncate">{{ tokenLabel }}</span>
          <span class="shrink-0 text-slate-300 dark:text-slate-600">|</span>
          <span class="shrink-0">{{ latencyLabel }}</span>
        </span>
        <div class="flex min-w-0 items-center gap-1 font-bold leading-tight" :title="successTitle">
          <span class="relative flex h-3 w-3 items-center justify-center">
            <svg
              :class="['absolute h-3 w-3 opacity-20', qualityIconClass]"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="3.2"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M3 12h3l2-5 4 10 3-7 2 2h4" />
            </svg>
            <svg
              :class="['absolute h-3 w-3 account-today-stats-ecg', qualityIconClass]"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="3.2"
            >
              <path stroke-linecap="round" stroke-linejoin="round" d="M3 12h3l2-5 4 10 3-7 2 2h4" />
            </svg>
          </span>
          <span :class="['min-w-0 truncate', qualityTextClass]">{{ successRateLabel }}</span>
        </div>
      </div>
    </div>

    <div v-else class="text-xs text-gray-400">-</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTokenDisplayMode } from '@/composables/useTokenDisplayMode'
import type { AccountTodayStats, AccountTodayStatsWindow, WindowStats } from '@/types'
import { normalizeAccountTodayStatsWindows } from '@/utils/accountDisplayPreferences'
import { useAccountTodayStatsDisplay } from './useAccountTodayStatsDisplay'

const props = withDefaults(
  defineProps<{
    stats?: AccountTodayStats | null
    loading?: boolean
    error?: string | null
    visibleWindows?: AccountTodayStatsWindow[]
  }>(),
  {
    stats: null,
    loading: false,
    error: null
  }
)

const { t } = useI18n()
const { formatTokenDisplay } = useTokenDisplayMode()

const emptyWindowStats = (): WindowStats => ({
  requests: 0,
  tokens: 0,
  cost: 0,
  standard_cost: 0,
  user_cost: 0,
  success_rate: 100,
  average_duration_ms: 0
})

const dayStats = computed(() => props.stats ?? emptyWindowStats())
const weeklyStats = computed(() => props.stats?.weekly ?? emptyWindowStats())
const totalStats = computed(() => props.stats?.total ?? emptyWindowStats())
const visibleWindows = computed(() =>
  normalizeAccountTodayStatsWindows(props.visibleWindows),
)

const {
  statColumns,
  successRateLabel,
  isLowSuccess,
  latencyLabel,
  footerStatsTitle,
  tokenLabel,
} = useAccountTodayStatsDisplay({
  dayStats,
  weeklyStats,
  totalStats,
  visibleWindows,
  formatTokenDisplay,
  t,
})
const rootClass = computed(() => [
  'flex select-none flex-col gap-1.5 font-mono text-[11px] leading-none text-slate-700 dark:text-slate-200',
  statColumns.value.length === 1
    ? 'w-[104px] min-w-[104px] max-w-[120px]'
    : 'w-[120px] min-w-[120px] max-w-[132px]',
])
const statsGridClass = computed(() => 'grid-cols-1')
const showTodayQualityFooter = computed(() =>
  visibleWindows.value.includes('today'),
)

const successTitle = computed(() => `${t('admin.accounts.status.active')}: ${successRateLabel.value}`)
const qualityTextClass = computed(() =>
  isLowSuccess.value ? 'text-rose-600 dark:text-rose-300' : 'text-emerald-600 dark:text-emerald-300'
)
const qualityIconClass = computed(() =>
  isLowSuccess.value ? 'text-rose-500 dark:text-rose-300' : 'text-emerald-500 dark:text-emerald-300'
)
</script>

<style scoped>
.account-today-stats-ecg path {
  stroke-dasharray: 15 100;
  animation: account-today-stats-sweep 2s linear infinite;
}

@keyframes account-today-stats-sweep {
  0% {
    stroke-dashoffset: 100;
    opacity: 0;
  }
  15% {
    opacity: 1;
  }
  85% {
    opacity: 1;
  }
  100% {
    stroke-dashoffset: -20;
    opacity: 0;
  }
}
</style>
