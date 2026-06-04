import { computed, type ComputedRef } from 'vue'
import type { AccountTodayStatsWindow, WindowStats } from '@/types'
import { formatCurrency, formatNumber } from '@/utils/format'

type Translate = (key: string) => string
type FormatTokenDisplay = (value: number) => string

export type TodayStatColumn = {
  key: string
  label: string
  requests: string
  cost: string
  title: string
  costClass: string
}

export function useAccountTodayStatsDisplay(options: {
  dayStats: ComputedRef<WindowStats>
  weeklyStats: ComputedRef<WindowStats>
  totalStats: ComputedRef<WindowStats>
  visibleWindows: ComputedRef<AccountTodayStatsWindow[]>
  formatTokenDisplay: FormatTokenDisplay
  t: Translate
}) {
  const createStatColumn = (
    key: string,
    label: string,
    stats: WindowStats,
    costClass: string,
  ): TodayStatColumn => {
    const requests = formatNumber(stats.requests || 0)
    const cost = formatCurrency(stats.cost || 0)
    return {
      key,
      label,
      requests,
      cost,
      title: `${label} ${options.t('admin.accounts.stats.requests')}: ${requests} · ${options.t('usage.accountBilled')}: ${cost}`,
      costClass,
    }
  }

  const statColumnConfig = computed<Record<AccountTodayStatsWindow, TodayStatColumn>>(() => ({
    today: createStatColumn(
      'today',
      options.t('dates.today'),
      options.dayStats.value,
      'text-slate-700 dark:text-slate-200',
    ),
    weekly: createStatColumn(
      'weekly',
      options.t('admin.accounts.status.window7d'),
      options.weeklyStats.value,
      'text-blue-600 dark:text-blue-300',
    ),
    total: createStatColumn(
      'total',
      options.t('common.total'),
      options.totalStats.value,
      'text-indigo-600 dark:text-indigo-300',
    ),
  }))
  const statColumns = computed<TodayStatColumn[]>(() =>
    options.visibleWindows.value.map((key) => statColumnConfig.value[key])
  )
  const successRate = computed(() => {
    const value = options.dayStats.value.success_rate
    if (typeof value !== 'number' || Number.isNaN(value)) return 100
    return Math.max(0, Math.min(value, 100))
  })
  const successRateLabel = computed(() =>
    `${successRate.value.toFixed(successRate.value % 1 === 0 ? 0 : 1)}%`
  )
  const isLowSuccess = computed(() => successRate.value < 95)
  const latencyLabel = computed(() => {
    const value = options.dayStats.value.average_duration_ms
    if (typeof value !== 'number' || value <= 0 || Number.isNaN(value)) return '0ms'
    if (value >= 1000) return `${(value / 1000).toFixed(value >= 10000 ? 0 : 1)}s`
    return `${Math.round(value)}ms`
  })
  const footerStatsTitle = computed(() =>
    `${String(options.dayStats.value.tokens || 0)} tokens · ${latencyLabel.value}`
  )
  const tokenLabel = computed(() =>
    options.formatTokenDisplay(options.dayStats.value.tokens || 0)
  )

  return {
    statColumns,
    successRateLabel,
    isLowSuccess,
    latencyLabel,
    footerStatsTitle,
    tokenLabel,
  }
}
