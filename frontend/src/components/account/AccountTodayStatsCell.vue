<template>
  <div>
    <div v-if="props.loading && !props.stats" class="w-[170px] space-y-1">
      <div class="h-3 w-36 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
      <div class="h-3 w-32 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
      <div class="h-3 w-40 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
    </div>

    <div v-else-if="props.error && !props.stats" class="text-xs text-red-500">
      {{ props.error }}
    </div>

    <div
      v-else-if="props.stats"
      class="flex min-w-[160px] max-w-[170px] select-none flex-col gap-1 font-mono text-[11px] leading-none text-slate-700 dark:text-slate-200"
      data-testid="account-today-stats-cell"
    >
      <div class="flex items-center justify-between gap-1">
        <span class="font-sans text-slate-400 dark:text-slate-500">{{ t('admin.accounts.stats.requests') }}:</span>
        <span class="font-bold">
          {{ formatNumber(dayStats.requests) }}
          <span class="mx-0.5 font-normal text-slate-300 dark:text-slate-600">/</span>
          {{ formatNumber(weeklyStats.requests) }}
          <span class="mx-0.5 font-normal text-slate-300 dark:text-slate-600">/</span>
          <span class="text-blue-600 dark:text-blue-300">{{ formatNumber(totalStats.requests) }}</span>
        </span>
      </div>

      <div class="flex items-center justify-between gap-1">
        <span class="font-sans text-slate-400 dark:text-slate-500">{{ t('usage.accountBilled') }}:</span>
        <span class="font-bold">
          {{ formatCurrency(dayStats.cost) }}
          <span class="mx-0.5 font-normal text-slate-300 dark:text-slate-600">/</span>
          {{ formatCurrency(weeklyStats.cost) }}
          <span class="mx-0.5 font-normal text-slate-300 dark:text-slate-600">/</span>
          <span class="text-indigo-600 dark:text-indigo-300">{{ formatCurrency(totalStats.cost) }}</span>
        </span>
      </div>

      <div class="mt-0.5 flex items-center justify-between gap-1 border-t border-slate-100 pt-1.5 dark:border-slate-700/70">
        <span
          class="font-sans text-slate-400 dark:text-slate-500"
          :title="String(dayStats.tokens || 0)"
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
