<template>
  <div>
    <div v-if="props.loading && !props.stats" class="w-[212px] space-y-1">
      <div class="h-3 w-36 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
      <div class="h-3 w-32 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
      <div class="h-3 w-40 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
    </div>

    <div v-else-if="props.error && !props.stats" class="text-xs text-red-500">
      {{ props.error }}
    </div>

    <div
      v-else-if="props.stats"
      class="flex min-w-[212px] max-w-[228px] select-none flex-col gap-1.5 font-mono text-[11px] leading-none text-slate-700 dark:text-slate-200"
      data-testid="account-today-stats-cell"
    >
      <div class="grid grid-cols-3 gap-1">
        <div
          v-for="item in statColumns"
          :key="item.key"
          class="min-w-0 rounded-md border border-slate-100 bg-white/60 px-1.5 py-1 dark:border-slate-700/70 dark:bg-slate-900/40"
          :title="item.title"
        >
          <div class="truncate font-sans text-[9px] font-semibold text-slate-400 dark:text-slate-500">
            {{ item.label }}
          </div>
          <div class="mt-0.5 truncate font-bold text-slate-800 dark:text-slate-100">
            {{ item.requests }}
          </div>
          <div class="mt-0.5 truncate text-[10px] font-bold" :class="item.costClass">
            {{ item.cost }}
          </div>
        </div>
      </div>

      <div class="mt-0.5 flex items-center justify-between gap-1 border-t border-slate-100 pt-1.5 dark:border-slate-700/70">
        <span
          class="font-sans text-slate-400 dark:text-slate-500"
          :title="footerStatsTitle"
        >
          {{ formatTokenDisplay(dayStats.tokens || 0) }}
          <span class="text-slate-300 dark:text-slate-600">|</span>
          {{ latencyLabel }}
        </span>
        <div class="flex items-center gap-1 font-bold" :title="successTitle">
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
          <span :class="qualityTextClass">{{ successRateLabel }}</span>
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
import type { AccountTodayStats, WindowStats } from '@/types'
import { formatNumber, formatCurrency } from '@/utils/format'

const props = withDefaults(
  defineProps<{
    stats?: AccountTodayStats | null
    loading?: boolean
    error?: string | null
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

type StatColumn = {
  key: string
  label: string
  requests: string
  cost: string
  title: string
  costClass: string
}

const createStatColumn = (
  key: string,
  label: string,
  stats: WindowStats,
  costClass: string,
): StatColumn => {
  const requests = formatNumber(stats.requests || 0)
  const cost = formatCurrency(stats.cost || 0)
  return {
    key,
    label,
    requests,
    cost,
    title: `${label} ${t('admin.accounts.stats.requests')}: ${requests} · ${t('usage.accountBilled')}: ${cost}`,
    costClass,
  }
}

const statColumns = computed<StatColumn[]>(() => [
  createStatColumn(
    'today',
    t('dates.today'),
    dayStats.value,
    'text-slate-700 dark:text-slate-200',
  ),
  createStatColumn(
    'weekly',
    t('admin.accounts.status.window7d'),
    weeklyStats.value,
    'text-blue-600 dark:text-blue-300',
  ),
  createStatColumn(
    'total',
    t('common.total'),
    totalStats.value,
    'text-indigo-600 dark:text-indigo-300',
  ),
])

const successRate = computed(() => {
  const value = dayStats.value.success_rate
  if (typeof value !== 'number' || Number.isNaN(value)) return 100
  return Math.max(0, Math.min(value, 100))
})

const successRateLabel = computed(() => `${successRate.value.toFixed(successRate.value % 1 === 0 ? 0 : 1)}%`)
const successTitle = computed(() => `${t('admin.accounts.status.active')}: ${successRateLabel.value}`)
const isLowSuccess = computed(() => successRate.value < 95)
const qualityTextClass = computed(() =>
  isLowSuccess.value ? 'text-rose-600 dark:text-rose-300' : 'text-emerald-600 dark:text-emerald-300'
)
const qualityIconClass = computed(() =>
  isLowSuccess.value ? 'text-rose-500 dark:text-rose-300' : 'text-emerald-500 dark:text-emerald-300'
)
const latencyLabel = computed(() => {
  const value = dayStats.value.average_duration_ms
  if (typeof value !== 'number' || value <= 0 || Number.isNaN(value)) return '0ms'
  if (value >= 1000) return `${(value / 1000).toFixed(value >= 10000 ? 0 : 1)}s`
  return `${Math.round(value)}ms`
})
const footerStatsTitle = computed(() => {
  return `${String(dayStats.value.tokens || 0)} tokens · ${latencyLabel.value}`
})
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
