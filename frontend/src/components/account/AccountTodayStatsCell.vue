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
      <div
        v-if="isAiryVariant"
        class="divide-y divide-slate-100 overflow-hidden rounded-xl border border-slate-200/80 bg-white/80 shadow-[0_2px_8px_rgba(15,23,42,0.03)] dark:divide-slate-700/50 dark:border-slate-700/70 dark:bg-slate-900/50"
        data-testid="account-today-stats-airy-panel"
      >
        <div
          v-for="item in statColumns"
          :key="item.key"
          class="flex min-w-0 items-center gap-1.5 px-2 py-1"
          :title="item.title"
          data-testid="account-today-stats-row"
        >
          <span class="w-6 shrink-0 truncate font-sans text-[9px] font-semibold text-slate-400 dark:text-slate-500">
            {{ item.label }}
          </span>
          <span class="min-w-0 flex-1 truncate text-[11px] font-bold text-slate-800 dark:text-slate-100">
            {{ item.requests }}
          </span>
          <span class="shrink-0 truncate text-[10px] font-bold" :class="item.costClass">
            {{ item.cost }}
          </span>
        </div>

        <div
          v-if="showTodayQualityFooter"
          class="flex min-w-0 items-center justify-between gap-2 px-2 py-1 font-sans leading-tight"
          data-testid="account-today-stats-footer"
        >
          <span
            class="flex min-w-0 flex-1 items-center gap-1 text-slate-400 dark:text-slate-500"
            :title="footerStatsTitle"
          >
            <span class="min-w-0 truncate">{{ tokenLabel }}</span>
            <span class="shrink-0 text-slate-300 dark:text-slate-600">|</span>
            <span class="shrink-0">{{ latencyLabel }}</span>
          </span>
          <div class="flex shrink-0 items-center gap-1 font-bold" :title="successTitle">
            <AccountTodayStatsQualityIcon :class-name="qualityIconClass" />
            <span :class="qualityTextClass">{{ successRateLabel }}</span>
          </div>
        </div>
      </div>

      <template v-else>
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
          data-testid="account-today-stats-footer"
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
            <AccountTodayStatsQualityIcon :class-name="qualityIconClass" />
            <span :class="['min-w-0 truncate', qualityTextClass]">{{ successRateLabel }}</span>
          </div>
        </div>
      </template>
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
import AccountTodayStatsQualityIcon from './AccountTodayStatsQualityIcon.vue'
import { useAccountTodayStatsDisplay } from './useAccountTodayStatsDisplay'

const props = withDefaults(
  defineProps<{
    stats?: AccountTodayStats | null
    loading?: boolean
    error?: string | null
    visibleWindows?: AccountTodayStatsWindow[]
    visualVariant?: 'default' | 'airy'
  }>(),
  {
    stats: null,
    loading: false,
    error: null,
    visualVariant: 'default'
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
const monthlyStats = computed(() => props.stats?.monthly ?? emptyWindowStats())
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
  monthlyStats,
  totalStats,
  visibleWindows,
  formatTokenDisplay,
  t,
})
const isAiryVariant = computed(() => props.visualVariant === 'airy')
const rootClass = computed(() => [
  'flex select-none flex-col font-mono text-[11px] leading-none text-slate-700 dark:text-slate-200',
  isAiryVariant.value ? '' : 'gap-1.5',
  isAiryVariant.value
    ? 'w-[136px] min-w-[104px] max-w-[152px]'
    : statColumns.value.length === 1
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
