<template>
  <div class="space-y-2">
    <div class="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
      <div class="card flex items-center gap-2 p-2">
        <div class="rounded-md bg-blue-100 p-1 text-blue-600 dark:bg-blue-900/30">
          <Icon name="document" size="sm" />
        </div>
        <div>
          <p class="text-xs font-medium text-gray-500">{{ t('usage.totalRequests') }}</p>
          <p class="text-base font-bold leading-tight">{{ stats?.total_requests?.toLocaleString() || '0' }}</p>
          <p class="text-xs text-gray-400">{{ t('usage.inSelectedRange') }}</p>
        </div>
      </div>
      <div class="card flex items-center gap-2 p-2">
        <div class="rounded-md bg-amber-100 p-1 text-amber-600 dark:bg-amber-900/30">
          <Icon name="cube" size="sm" />
        </div>
        <div>
          <p class="text-xs font-medium text-gray-500">{{ t('usage.totalTokens') }}</p>
          <p class="text-base font-bold leading-tight">{{ formatTokens(stats?.total_tokens || 0) }}</p>
          <p class="text-xs text-gray-500">
            {{ t('usage.in') }}: {{ formatTokens(stats?.total_input_tokens || 0) }} /
            {{ t('usage.out') }}: {{ formatTokens(stats?.total_output_tokens || 0) }}
          </p>
        </div>
      </div>
      <CacheStatsCard
        :cache-hit-rate="stats?.cache_hit_rate"
        :cache-creation-tokens="stats?.total_cache_creation_tokens"
        :cache-read-tokens="stats?.total_cache_read_tokens"
        :stats-card-style="statsCardStyle"
      />
      <div class="card flex items-center gap-2 p-2">
        <div class="rounded-md bg-green-100 p-1 text-green-600 dark:bg-green-900/30">
          <Icon name="dollar" size="sm" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-xs font-medium text-gray-500">{{ t('usage.totalCost') }}</p>
          <p class="text-base font-bold leading-tight text-green-600">
            {{ formatPrimaryCost(stats) }}
          </p>
          <p v-if="stats?.total_account_cost != null" class="text-xs text-gray-400">
            {{ t('usage.userBilled') }}:
            <span class="text-gray-300">{{ formatCurrencyBreakdown(stats?.actual_cost_by_currency, stats?.total_actual_cost || 0) }}</span>
            · {{ t('usage.standardCost') }}:
            <span class="text-gray-300">{{ formatCurrencyBreakdown(stats?.cost_by_currency, stats?.total_cost || 0) }}</span>
          </p>
          <p v-else class="text-xs text-gray-400">
            {{ t('usage.actualCost') }}:
            <span>{{ formatCurrencyBreakdown(stats?.actual_cost_by_currency, stats?.total_actual_cost || 0) }}</span>
            · {{ t('usage.standardCost') }}:
            <span class="line-through">{{ formatCurrencyBreakdown(stats?.cost_by_currency, stats?.total_cost || 0) }}</span>
          </p>
          <p v-if="stats?.admin_free_requests" class="mt-1 text-[11px] text-emerald-500 dark:text-emerald-300">
            管理员免扣 {{ stats.admin_free_requests.toLocaleString() }} 次 / ${{ (stats.admin_free_standard_cost || 0).toFixed(4) }} 标准成本
          </p>
        </div>
      </div>
      <div class="card flex items-center gap-2 p-2">
        <div class="rounded-md bg-purple-100 p-1 text-purple-600 dark:bg-purple-900/30">
          <Icon name="clock" size="sm" />
        </div>
        <div>
          <p class="text-xs font-medium text-gray-500">{{ t('usage.avgDuration') }}</p>
          <p class="text-base font-bold leading-tight">{{ formatDuration(stats?.average_duration_ms || 0) }}</p>
        </div>
      </div>
    </div>

    <div class="card border border-dashed border-primary-200/80 bg-primary-50/40 p-2 dark:border-primary-500/20 dark:bg-primary-500/5">
      <div class="mb-1.5">
        <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.usage.todayStats') }}</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('usage.todaySoFar') }}</p>
      </div>
      <div class="grid grid-cols-1 gap-2 md:grid-cols-4 xl:grid-cols-6">
        <div class="rounded-md bg-white/80 p-2 shadow-sm dark:bg-dark-900/70">
          <p class="text-xs font-medium text-gray-500">{{ t('admin.usage.todayRequests') }}</p>
          <p class="mt-0.5 text-base font-bold leading-tight">{{ stats?.today_requests?.toLocaleString() || '0' }}</p>
        </div>
        <div class="rounded-md bg-white/80 p-2 shadow-sm dark:bg-dark-900/70 md:col-span-2 xl:col-span-3">
          <div class="mb-1.5 flex items-start justify-between gap-2">
            <div>
              <p class="text-xs font-medium text-gray-500">{{ t('admin.usage.todayTokens') }}</p>
              <p class="mt-0.5 text-base font-bold leading-tight">{{ formatTokens(stats?.today_tokens || 0) }}</p>
            </div>
            <span class="rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
              {{ formatPercent(stats?.today_cache_hit_rate || 0) }}
            </span>
          </div>
          <div class="grid grid-cols-2 gap-1.5 lg:grid-cols-5">
            <div
              v-for="item in todayTokenItems"
              :key="item.key"
              class="rounded-md border border-gray-100 bg-gray-50/80 px-2 py-1 dark:border-dark-700 dark:bg-dark-800/70"
            >
              <p class="text-[11px] text-gray-500 dark:text-gray-400">{{ item.label }}</p>
              <p class="mt-1 text-sm font-semibold" :class="item.className">{{ item.value }}</p>
            </div>
          </div>
        </div>
        <div class="rounded-md bg-white/80 p-2 shadow-sm dark:bg-dark-900/70">
          <p class="text-xs font-medium text-gray-500">{{ t('admin.usage.todayCost') }}</p>
          <p class="mt-0.5 text-base font-bold leading-tight text-green-600">
            {{ formatCurrencyBreakdown(stats?.today_actual_cost_by_currency, stats?.today_actual_cost || 0) }}
          </p>
          <p class="text-xs text-gray-400">
            {{ t('usage.actualCost') }}:
            <span>{{ formatCurrencyBreakdown(stats?.today_actual_cost_by_currency, stats?.today_actual_cost || 0) }}</span>
            · {{ t('usage.standardCost') }}:
            <span class="line-through">{{ formatCurrencyBreakdown(stats?.today_cost_by_currency, stats?.today_cost || 0) }}</span>
          </p>
        </div>
        <div class="rounded-md bg-white/80 p-2 shadow-sm dark:bg-dark-900/70">
          <p class="text-xs font-medium text-gray-500">{{ t('usage.todayAvgDuration') }}</p>
          <p class="mt-0.5 text-base font-bold leading-tight">{{ formatDuration(stats?.today_average_duration_ms || 0) }}</p>
        </div>
      </div>
    </div>

    <div
      v-if="platformBreakdown.length > 0"
      class="card p-4"
      data-testid="admin-usage-platform-breakdown"
    >
      <div class="mb-3 flex items-center justify-between gap-3">
        <div>
          <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('usage.platformBreakdown') }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('usage.platformBreakdownHint') }}</p>
        </div>
      </div>
      <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
        <div
          v-for="item in platformBreakdown"
          :key="item.platform"
          class="rounded-lg border border-gray-200 bg-white/80 p-3 dark:border-dark-700 dark:bg-dark-900/70"
        >
          <div class="mb-2 flex items-center gap-2">
            <PlatformIcon
              v-if="item.platform !== 'unknown'"
              :platform="item.platform"
              size="sm"
              class="shrink-0"
            />
            <span class="truncate text-sm font-semibold text-gray-900 dark:text-white">
              {{ formatPlatformLabel(item.platform) }}
            </span>
          </div>
          <div class="grid grid-cols-2 gap-2 text-xs text-gray-500 dark:text-gray-400">
            <span>{{ t('usage.totalRequests') }}</span>
            <span class="text-right text-gray-800 dark:text-gray-100">{{ item.requests.toLocaleString() }}</span>
            <span>{{ t('usage.totalTokens') }}</span>
            <span class="text-right text-gray-800 dark:text-gray-100">{{ formatTokens(item.total_tokens || 0) }}</span>
            <span>{{ t('usage.actualCost') }}</span>
            <span class="text-right text-gray-800 dark:text-gray-100">
              {{ formatCurrencyBreakdown(item.actual_cost_by_currency, item.actual_cost || 0) }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminUsageStatsResponse } from '@/api/admin/usage'
import { useTokenDisplayMode } from '@/composables/useTokenDisplayMode'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import CacheStatsCard from '@/components/usage/CacheStatsCard.vue'
import { getPlatformEnglishName } from '@/utils/platformBranding'

const props = withDefaults(defineProps<{
  stats: AdminUsageStatsResponse | null
  statsCardStyle?: 'balanced' | 'accent'
}>(), {
  statsCardStyle: 'balanced'
})

const { t } = useI18n()
const { formatTokenDisplay } = useTokenDisplayMode()

const platformBreakdown = computed(() => props.stats?.platform_breakdown || [])

const formatDuration = (ms: number) =>
  ms < 1000 ? `${ms.toFixed(0)}ms` : `${(ms / 1000).toFixed(2)}s`

const formatTokens = (value: number) => formatTokenDisplay(value)

const formatPercent = (value: number) => {
  const normalized = value <= 1 ? value * 100 : value
  return `${normalized.toFixed(1)}%`
}

const formatCurrencyAmount = (currency: string, value: number) => {
  const normalized = currency.toUpperCase()
  const prefix = normalized === 'CNY' ? '¥' : '$'
  return `${prefix}${value.toFixed(4)}`
}

const formatCurrencyBreakdown = (values?: Record<string, number>, fallbackUSD = 0) => {
  const entries = Object.entries(values || {})
    .filter(([, value]) => Number.isFinite(value))
    .sort(([left], [right]) => left.localeCompare(right))
  if (entries.length === 0) {
    return formatCurrencyAmount('USD', fallbackUSD)
  }
  return entries.map(([currency, value]) => formatCurrencyAmount(currency, value)).join(' / ')
}

const formatPrimaryCost = (stats: AdminUsageStatsResponse | null) => {
  if (!stats) return formatCurrencyAmount('USD', 0)
  if (stats.total_account_cost != null) {
    return formatCurrencyAmount('USD', stats.total_account_cost || 0)
  }
  return formatCurrencyBreakdown(stats.actual_cost_by_currency, stats.total_actual_cost || 0)
}

const formatPlatformLabel = (platform: string) =>
  platform === 'unknown' ? t('usage.unknownPlatform') : getPlatformEnglishName(platform)

const todayTokenItems = computed(() => [
  {
    key: 'input',
    label: t('usage.inputTokens'),
    value: formatTokens(props.stats?.today_input_tokens || 0),
    className: 'text-emerald-600 dark:text-emerald-400',
  },
  {
    key: 'cache_write',
    label: t('usage.cacheCreationTokens'),
    value: formatTokens(props.stats?.today_cache_creation_tokens || 0),
    className: 'text-amber-600 dark:text-amber-400',
  },
  {
    key: 'cache_read',
    label: t('usage.cacheReadTokens'),
    value: formatTokens(props.stats?.today_cache_read_tokens || 0),
    className: 'text-sky-600 dark:text-sky-400',
  },
  {
    key: 'output',
    label: t('usage.outputTokens'),
    value: formatTokens(props.stats?.today_output_tokens || 0),
    className: 'text-violet-600 dark:text-violet-400',
  },
  {
    key: 'hit_rate',
    label: t('usage.cacheHitRate'),
    value: formatPercent(props.stats?.today_cache_hit_rate || 0),
    className: 'text-teal-600 dark:text-teal-400',
  },
])
</script>
