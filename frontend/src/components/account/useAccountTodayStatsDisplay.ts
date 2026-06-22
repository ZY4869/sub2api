import { computed, type ComputedRef } from 'vue'
import type { AccountTodayStatsWindow, WindowStats } from '@/types'
import { formatNumber } from '@/utils/format'
import { formatDiscountPercent, formatUsdAmount } from '@/utils/accountBillingDisplay'

type Translate = (key: string) => string
type FormatTokenDisplay = (value: number) => string
type IconName = 'clock' | 'calendar' | 'chart' | 'database'

export type TodayStatColumn = {
  key: string
  label: string
  iconName: IconName
  requests: string
  cost: string
  standardCost: string
  savedCost: string
  savedPercent: string
  hasSavings: boolean
  title: string
  costClass: string
}

export function useAccountTodayStatsDisplay(options: {
  dayStats: ComputedRef<WindowStats>
  weeklyStats: ComputedRef<WindowStats>
  monthlyStats: ComputedRef<WindowStats>
  totalStats: ComputedRef<WindowStats>
  visibleWindows: ComputedRef<AccountTodayStatsWindow[]>
  formatTokenDisplay: FormatTokenDisplay
  t: Translate
}) {
  const createStatColumn = (
    key: string,
    label: string,
    iconName: IconName,
    stats: WindowStats,
    costClass: string,
  ): TodayStatColumn => {
    const requests = formatNumber(stats.requests || 0)
    const costValue = stats.cost || 0
    const standardCostValue = typeof stats.standard_cost === 'number' ? stats.standard_cost : costValue
    const savedValue = Math.max(0, standardCostValue - costValue)
    const savedPercentValue = standardCostValue > 0 ? (1 - costValue / standardCostValue) * 100 : 0
    const cost = formatUsdAmount(costValue)
    const standardCost = formatUsdAmount(standardCostValue)
    const savedCost = formatUsdAmount(savedValue)
    const savedPercent = formatDiscountPercent(savedPercentValue)
    const hasSavings = savedValue > 0.000001
    const costTitle = hasSavings
      ? `${options.t('admin.accounts.keyUsage.discountedCost')}: ${cost} · ${options.t('admin.accounts.keyUsage.standardCost')}: ${standardCost} · ${options.t('admin.accounts.keyUsage.saved')}: ${savedCost} (${savedPercent})`
      : `${options.t('usage.accountBilled')}: ${cost}`
    return {
      key,
      label,
      iconName,
      requests,
      cost,
      standardCost,
      savedCost,
      savedPercent,
      hasSavings,
      title: `${label} ${options.t('admin.accounts.stats.requests')}: ${requests} · ${costTitle}`,
      costClass,
    }
  }

  const statColumnConfig = computed<Record<AccountTodayStatsWindow, TodayStatColumn>>(() => ({
    today: createStatColumn(
      'today',
      options.t('dates.today'),
      'clock',
      options.dayStats.value,
      'text-slate-700 dark:text-slate-200',
    ),
    weekly: createStatColumn(
      'weekly',
      options.t('admin.accounts.status.window7d'),
      'calendar',
      options.weeklyStats.value,
      'text-blue-600 dark:text-blue-300',
    ),
    monthly: createStatColumn(
      'monthly',
      options.t('admin.accounts.stats.monthlyUsage'),
      'chart',
      options.monthlyStats.value,
      'text-emerald-600 dark:text-emerald-300',
    ),
    total: createStatColumn(
      'total',
      options.t('common.total'),
      'database',
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
    `${buildTokenBreakdownTitle(options.dayStats.value, options.formatTokenDisplay)} · ${latencyLabel.value}`
  )
  const tokenLabel = computed(() =>
    buildTokenBreakdownLabel(options.dayStats.value, options.formatTokenDisplay)
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

function buildTokenBreakdownLabel(stats: WindowStats, formatTokenDisplay: FormatTokenDisplay): string {
  const cacheTokens = stats.cache_tokens ?? ((stats.cache_creation_tokens || 0) + (stats.cache_read_tokens || 0))
  if (!stats.input_tokens && !stats.output_tokens && !cacheTokens) {
    return formatTokenDisplay(stats.tokens || 0)
  }
  return [
    `I ${formatTokenDisplay(stats.input_tokens || 0)}`,
    `O ${formatTokenDisplay(stats.output_tokens || 0)}`,
    `C ${formatTokenDisplay(cacheTokens || 0)}`,
  ].join(' ')
}

function buildTokenBreakdownTitle(stats: WindowStats, formatTokenDisplay: FormatTokenDisplay): string {
  const cacheWrite = stats.cache_creation_tokens || 0
  const cacheRead = stats.cache_read_tokens || 0
  const cacheTokens = stats.cache_tokens ?? (cacheWrite + cacheRead)
  if (!stats.input_tokens && !stats.output_tokens && !cacheTokens) {
    return `${String(stats.tokens || 0)} tokens`
  }
  const hitRate = typeof stats.cache_hit_rate === 'number'
    ? ` · hit ${formatCacheHitRate(stats.cache_hit_rate)}`
    : ''
  return [
    `input ${formatTokenDisplay(stats.input_tokens || 0)}`,
    `output ${formatTokenDisplay(stats.output_tokens || 0)}`,
    `cache write ${formatTokenDisplay(cacheWrite)}`,
    `cache read ${formatTokenDisplay(cacheRead)}`,
  ].join(' · ') + hitRate
}

function formatCacheHitRate(value: number): string {
  const normalized = value <= 1 ? value * 100 : value
  return `${normalized.toFixed(1)}%`
}
