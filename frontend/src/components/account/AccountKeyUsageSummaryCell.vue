<template>
  <div class="min-w-[520px] max-w-full space-y-1.5 text-[11px] leading-none" data-testid="account-key-usage-summary-cell">
    <template v-if="loading && !stats">
      <div class="flex flex-wrap items-center gap-2" data-testid="account-key-usage-today-row">
        <span
          v-for="index in 5"
          :key="index"
          class="h-6 w-20 animate-pulse rounded-full bg-slate-100 dark:bg-slate-800"
        />
      </div>
      <div class="flex flex-wrap items-center gap-2" data-testid="account-key-usage-quota-row">
        <span class="h-6 w-24 animate-pulse rounded-full bg-slate-100 dark:bg-slate-800" />
      </div>
    </template>

    <span v-else-if="error && !stats" class="text-xs text-rose-500">
      {{ error }}
    </span>

    <template v-else>
      <div class="flex flex-wrap items-center gap-2" data-testid="account-key-usage-today-row">
        <span
          v-for="item in todayItems"
          :key="item.key"
          :class="[
            'inline-flex min-w-0 items-center gap-1 rounded-full border px-2 py-1 font-semibold',
            item.className
          ]"
          :title="item.title"
          :data-testid="`account-key-usage-${item.key}`"
        >
          <Icon :name="item.icon" size="xs" :stroke-width="2" class="shrink-0" />
          <span class="shrink-0 text-slate-500 dark:text-slate-300">{{ item.label }}</span>
          <span class="min-w-0 truncate text-left tabular-nums">{{ item.value }}</span>
        </span>
      </div>

      <div class="flex flex-wrap items-center gap-2" data-testid="account-key-usage-quota-row">
        <span
          v-if="quotaItems.length === 0"
          class="inline-flex items-center gap-1 rounded-full border border-emerald-200 bg-emerald-50 px-2 py-1 font-bold text-emerald-700 dark:border-emerald-400/25 dark:bg-emerald-400/10 dark:text-emerald-100"
          :title="t('admin.accounts.keyUsage.unlimited')"
          data-testid="account-key-usage-unlimited"
        >
          <Icon name="checkCircle" size="xs" :stroke-width="2" />
          {{ t('admin.accounts.keyUsage.unlimited') }}
        </span>

        <span
          v-for="quota in quotaItems"
          :key="quota.key"
          class="inline-flex min-w-0 items-center gap-1.5 rounded-full border border-slate-200 bg-slate-50 px-2 py-1 font-semibold text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-100"
          :title="quota.title"
          :data-testid="`account-key-quota-${quota.key}`"
        >
          <span
            :class="[
              'rounded-full border px-1.5 py-0.5 text-[9px] font-black leading-none',
              resolveUsageWindowCapsuleClass(quota.label)
            ]"
          >
            {{ quota.label }}
          </span>
          <span class="tabular-nums">{{ quota.value }}</span>
          <span v-if="quota.resetText" class="text-[10px] font-medium text-slate-400 dark:text-slate-500">
            {{ quota.resetText }}
          </span>
        </span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account, WindowStats } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import { useTokenDisplayMode } from '@/composables/useTokenDisplayMode'
import { useRealtimeCountdownNow } from '@/composables/useRealtimeCountdownNow'
import { formatDiscountPercent, formatUsdAmount } from '@/utils/accountBillingDisplay'
import { resolveUsageWindowCapsuleClass } from '@/utils/accountUsageWindowDisplay'
import { formatNumber } from '@/utils/format'
import {
  formatLocalAbsoluteTime,
  formatResetCountdown,
  parseEffectiveResetAt,
} from '@/utils/usageResetTime'

type IconName = 'chart' | 'database' | 'dollar' | 'calculator' | 'gift'

const props = withDefaults(
  defineProps<{
    account: Account
    stats?: WindowStats | null
    loading?: boolean
    error?: string | null
  }>(),
  {
    stats: null,
    loading: false,
    error: null,
  },
)

const { t } = useI18n()
const { formatTokenDisplay } = useTokenDisplayMode()
const { nowDate } = useRealtimeCountdownNow('accounts')

const currentStats = computed<WindowStats>(() => props.stats ?? {
  requests: 0,
  tokens: 0,
  cost: 0,
  standard_cost: 0,
  user_cost: 0,
})

const discountedCost = computed(() => currentStats.value.cost || 0)
const standardCost = computed(() => {
  const value = currentStats.value.standard_cost
  return typeof value === 'number' && Number.isFinite(value) ? value : discountedCost.value
})
const savedCost = computed(() => Math.max(0, standardCost.value - discountedCost.value))
const savedPercent = computed(() =>
  standardCost.value > 0 ? Math.max(0, 1 - discountedCost.value / standardCost.value) * 100 : 0
)

const createTodayItem = (
  key: string,
  label: string,
  value: string,
  icon: IconName,
  className: string,
  title = `${label}: ${value}`,
) => ({ key, label, value, icon, className, title })

const todayItems = computed(() => [
  createTodayItem(
    'requests',
    t('admin.accounts.keyUsage.requests'),
    formatNumber(currentStats.value.requests || 0),
    'chart',
    'border-slate-200 bg-white text-slate-800 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100',
  ),
  createTodayItem(
    'tokens',
    t('admin.accounts.keyUsage.tokens'),
    formatTokenDisplay(currentStats.value.tokens || 0),
    'database',
    'border-slate-200 bg-white text-slate-800 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100',
    `${currentStats.value.tokens || 0} tokens`,
  ),
  createTodayItem(
    'discounted-cost',
    t('admin.accounts.keyUsage.discountedCost'),
    formatUsdAmount(discountedCost.value),
    'dollar',
    'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-400/25 dark:bg-emerald-400/10 dark:text-emerald-100',
  ),
  createTodayItem(
    'standard-cost',
    t('admin.accounts.keyUsage.standardCost'),
    formatUsdAmount(standardCost.value),
    'calculator',
    'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-400/25 dark:bg-sky-400/10 dark:text-sky-100',
  ),
  createTodayItem(
    'saved',
    t('admin.accounts.keyUsage.saved'),
    `${formatUsdAmount(savedCost.value)} / ${formatDiscountPercent(savedPercent.value)}`,
    'gift',
    savedCost.value > 0
      ? 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-400/25 dark:bg-amber-400/10 dark:text-amber-100'
      : 'border-slate-200 bg-slate-50 text-slate-500 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-300',
  ),
])

const finiteNumber = (value: unknown): number => (
  typeof value === 'number' && Number.isFinite(value) ? value : 0
)

const formatQuotaValue = (used: number, limit: number) => {
  const remaining = Math.max(0, limit - used)
  return `${formatNumber(remaining)} / ${formatNumber(limit)}`
}

const formatQuotaReset = (resetAt: string | null | undefined) => {
  const effectiveResetAt = parseEffectiveResetAt(resetAt ?? null, null, nowDate.value)
  if (!effectiveResetAt) return ''
  return formatResetCountdown(
    effectiveResetAt,
    nowDate.value,
    t('admin.accounts.usageWindow.now'),
  )
}

const formatQuotaResetTitle = (resetAt: string | null | undefined) => {
  const effectiveResetAt = parseEffectiveResetAt(resetAt ?? null, null, nowDate.value)
  if (!effectiveResetAt) return ''
  return formatLocalAbsoluteTime(effectiveResetAt, nowDate.value, {
    today: t('dates.today'),
    tomorrow: t('dates.tomorrow'),
  })
}

const quotaItems = computed(() => {
  const specs = [
    {
      key: 'daily',
      label: '1D',
      used: finiteNumber(props.account.quota_daily_used),
      limit: finiteNumber(props.account.quota_daily_limit),
      resetAt: props.account.quota_daily_reset_at,
    },
    {
      key: 'weekly',
      label: '7D',
      used: finiteNumber(props.account.quota_weekly_used),
      limit: finiteNumber(props.account.quota_weekly_limit),
      resetAt: props.account.quota_weekly_reset_at,
    },
    {
      key: 'monthly',
      label: '30D',
      used: finiteNumber(props.account.quota_monthly_used),
      limit: finiteNumber(props.account.quota_monthly_limit),
      resetAt: props.account.quota_monthly_reset_at,
    },
    {
      key: 'total',
      label: t('ui.usageWindow.total'),
      used: finiteNumber(props.account.quota_used),
      limit: finiteNumber(props.account.quota_limit),
      resetAt: null,
    },
  ]

  return specs
    .filter((item) => item.limit > 0)
    .map((item) => ({
      key: item.key,
      label: item.label,
      value: formatQuotaValue(item.used, item.limit),
      resetText: formatQuotaReset(item.resetAt),
      title: `${t('admin.accounts.keyUsage.callQuota')} ${item.label}: ${formatQuotaValue(item.used, item.limit)}${item.resetAt ? ` · ${formatQuotaResetTitle(item.resetAt)}` : ''}`,
    }))
})
</script>
