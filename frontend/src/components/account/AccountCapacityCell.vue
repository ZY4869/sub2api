<template>
  <div
    v-if="isGlassVariant"
    :class="[
      compact ? 'inline-flex max-w-[176px]' : 'flex flex-col gap-2.5'
    ]"
    data-testid="airy-capacity-cell"
  >
    <span class="sr-only">{{ formattedCapacityPair }}</span>
    <AccountAiryCapacityPrimary
      :used="currentConcurrency"
      :total="totalConcurrency"
      :formatted-current="formattedCurrentConcurrency"
      :formatted-total="formattedTotalConcurrency"
      :tone="capacityTone"
      :white-surface-enabled="whiteSurfaceEnabled"
      :compact="compact"
    />

    <div
      v-if="!compact && hasAirySecondaryMetrics"
      class="grid gap-2 sm:grid-cols-2"
      data-testid="airy-capacity-metrics"
    >
      <AccountAiryCapacityMetricCard
        v-for="metric in airyMetrics"
        :key="metric.key"
        :label="metric.label"
        :value="metric.value"
        :title="metric.title"
        :tone="metric.tone"
        :tag="metric.tag"
        :white-surface-enabled="whiteSurfaceEnabled"
      >
        <template #icon>
          <svg
            v-if="metric.key === 'window-cost'"
            class="h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v12m-3-2.818l.879.659c1.171.879 3.07.879 4.242 0 1.172-.879 1.172-2.303 0-3.182C13.536 12.219 12.768 12 12 12c-.725 0-1.45-.22-2.003-.659-1.106-.879-1.106-2.303 0-3.182s2.9-.879 4.006 0l.415.33M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <svg
            v-else-if="metric.key === 'sessions'"
            class="h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z" />
          </svg>
          <svg
            v-else
            class="h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="1.8"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
          </svg>
        </template>
      </AccountAiryCapacityMetricCard>

      <QuotaBadge
        v-if="showDailyQuota"
        :used="account.quota_daily_used ?? 0"
        :limit="account.quota_daily_limit!"
        kind="daily"
        visual-variant="glass"
        :white-surface-enabled="whiteSurfaceEnabled"
      />
      <QuotaBadge
        v-if="showWeeklyQuota"
        :used="account.quota_weekly_used ?? 0"
        :limit="account.quota_weekly_limit!"
        kind="weekly"
        visual-variant="glass"
        :white-surface-enabled="whiteSurfaceEnabled"
      />
      <QuotaBadge
        v-if="showTotalQuota"
        :used="account.quota_used ?? 0"
        :limit="account.quota_limit!"
        kind="total"
        visual-variant="glass"
        :white-surface-enabled="whiteSurfaceEnabled"
      />
    </div>
  </div>

  <div v-else class="flex flex-col gap-1.5">
    <div class="flex items-center gap-1.5">
      <span
        :class="[
          concurrencyBaseClass,
          concurrencyClass
        ]"
      >
        <span class="relative flex items-center justify-center">
          <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 016 3.75h2.25A2.25 2.25 0 0110.5 6v2.25a2.25 2.25 0 01-2.25 2.25H6a2.25 2.25 0 01-2.25-2.25V6zM3.75 15.75A2.25 2.25 0 016 13.5h2.25a2.25 2.25 0 012.25 2.25V18a2.25 2.25 0 01-2.25 2.25H6A2.25 2.25 0 013.75 18v-2.25zM13.5 6a2.25 2.25 0 012.25-2.25H18A2.25 2.25 0 0120.25 6v2.25A2.25 2.25 0 0118 10.5h-2.25a2.25 2.25 0 01-2.25-2.25V6zM13.5 15.75a2.25 2.25 0 012.25-2.25H18a2.25 2.25 0 012.25 2.25V18A2.25 2.25 0 0118 20.25h-2.25A2.25 2.25 0 0113.5 18v-2.25z" />
          </svg>
          <span
            v-if="showConcurrencyPulse"
            class="absolute -right-[2px] -top-[2px] flex h-1.5 w-1.5"
          >
            <span
              v-if="visualVariant === 'default'"
              class="absolute inline-flex h-full w-full animate-ping rounded-full bg-blue-400 opacity-75"
            ></span>
            <span class="relative inline-flex h-1.5 w-1.5 rounded-full bg-blue-500"></span>
          </span>
        </span>
        <span class="font-mono font-bold">{{ formattedCurrentConcurrency }}</span>
        <span class="text-gray-400 dark:text-gray-500">/</span>
        <span class="font-mono font-bold opacity-70">{{ formattedTotalConcurrency }}</span>
      </span>
    </div>

    <div v-if="showWindowCost" class="flex items-center gap-1">
      <span
        :class="[
          'inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[10px] font-medium',
          windowCostClass
        ]"
        :title="windowCostTooltip"
      >
        <svg class="h-2.5 w-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v12m-3-2.818l.879.659c1.171.879 3.07.879 4.242 0 1.172-.879 1.172-2.303 0-3.182C13.536 12.219 12.768 12 12 12c-.725 0-1.45-.22-2.003-.659-1.106-.879-1.106-2.303 0-3.182s2.9-.879 4.006 0l.415.33M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span class="font-mono">${{ formatCost(currentWindowCost) }}</span>
        <span class="text-gray-400 dark:text-gray-500">/</span>
        <span class="font-mono">${{ formatCost(account.window_cost_limit) }}</span>
      </span>
    </div>

    <div v-if="showSessionLimit" class="flex items-center gap-1">
      <span
        :class="[
          'inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[10px] font-medium',
          sessionLimitClass
        ]"
        :title="sessionLimitTooltip"
      >
        <svg class="h-2.5 w-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M15 19.128a9.38 9.38 0 002.625.372 9.337 9.337 0 004.121-.952 4.125 4.125 0 00-7.533-2.493M15 19.128v-.003c0-1.113-.285-2.16-.786-3.07M15 19.128v.106A12.318 12.318 0 018.624 21c-2.331 0-4.512-.645-6.374-1.766l-.001-.109a6.375 6.375 0 0111.964-3.07M12 6.375a3.375 3.375 0 11-6.75 0 3.375 3.375 0 016.75 0zm8.25 2.25a2.625 2.625 0 11-5.25 0 2.625 2.625 0 015.25 0z" />
        </svg>
        <span class="font-mono">{{ activeSessions }}</span>
        <span class="text-gray-400 dark:text-gray-500">/</span>
        <span class="font-mono">{{ account.max_sessions }}</span>
      </span>
    </div>

    <div v-if="showRpmLimit" class="flex items-center gap-1">
      <span
        :class="[
          'inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[10px] font-medium',
          rpmClass
        ]"
        :title="rpmTooltip"
      >
        <svg class="h-2.5 w-2.5" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
        </svg>
        <span class="font-mono">{{ currentRPM }}</span>
        <span class="text-gray-400 dark:text-gray-500">/</span>
        <span class="font-mono">{{ account.base_rpm }}</span>
        <span class="text-[9px] opacity-60">{{ rpmStrategyTag }}</span>
      </span>
    </div>

    <QuotaBadge v-if="showDailyQuota" :used="account.quota_daily_used ?? 0" :limit="account.quota_daily_limit!" kind="daily" visual-variant="default" />
    <QuotaBadge v-if="showWeeklyQuota" :used="account.quota_weekly_used ?? 0" :limit="account.quota_weekly_limit!" kind="weekly" visual-variant="default" />
    <QuotaBadge v-if="showTotalQuota" :used="account.quota_used ?? 0" :limit="account.quota_limit!" kind="total" visual-variant="default" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account } from '@/types'
import { resolveEffectiveAccountPlatformFromAccount } from '@/utils/accountProtocolGateway'
import AccountAiryCapacityMetricCard from './capacity/AccountAiryCapacityMetricCard.vue'
import AccountAiryCapacityPrimary from './capacity/AccountAiryCapacityPrimary.vue'
import {
  formatCapacityValue,
  formatQuotaCurrency,
  resolveCapacityPadWidth,
  resolveCapacityTone
} from './capacity/presentation'
import QuotaBadge from './QuotaBadge.vue'

const props = defineProps<{
  account: Account
  visualVariant?: 'default' | 'glass'
  whiteSurfaceEnabled?: boolean
  compact?: boolean
}>()

const { t } = useI18n()
const runtimePlatform = computed(() => resolveEffectiveAccountPlatformFromAccount(props.account))
const visualVariant = computed(() => props.visualVariant || 'default')
const isGlassVariant = computed(() => visualVariant.value === 'glass')

// 当前并发数
const currentConcurrency = computed(() => props.account.current_concurrency || 0)
const totalConcurrency = computed(() => props.account.concurrency || 0)
const concurrencyPadWidth = computed(() => resolveCapacityPadWidth(totalConcurrency.value))
const formatConcurrencyValue = (value: number) => formatCapacityValue(value, concurrencyPadWidth.value)
const formattedCurrentConcurrency = computed(() => formatConcurrencyValue(currentConcurrency.value))
const formattedTotalConcurrency = computed(() => formatConcurrencyValue(totalConcurrency.value))
const formattedCapacityPair = computed(
  () => `${formattedCurrentConcurrency.value}/${formattedTotalConcurrency.value}`
)
const capacityTone = computed(() => resolveCapacityTone(currentConcurrency.value, totalConcurrency.value))

// 是否为 Anthropic OAuth/SetupToken 账号
const isAnthropicOAuthOrSetupToken = computed(() => {
  return (
    runtimePlatform.value === 'anthropic' &&
    (props.account.type === 'oauth' || props.account.type === 'setup-token')
  )
})

// 是否显示窗口费用限制
const showWindowCost = computed(() => {
  return (
    isAnthropicOAuthOrSetupToken.value &&
    props.account.window_cost_limit !== undefined &&
    props.account.window_cost_limit !== null &&
    props.account.window_cost_limit > 0
  )
})

// 当前窗口费用
const currentWindowCost = computed(() => props.account.current_window_cost ?? 0)

// 是否显示会话限制
const showSessionLimit = computed(() => {
  return (
    isAnthropicOAuthOrSetupToken.value &&
    props.account.max_sessions !== undefined &&
    props.account.max_sessions !== null &&
    props.account.max_sessions > 0
  )
})

// 当前活跃会话数
const activeSessions = computed(() => props.account.active_sessions ?? 0)

const concurrencyBaseClass = computed(() => {
  if (visualVariant.value === 'glass') {
    return 'inline-flex items-center gap-1.5 rounded-md border px-2 py-[3px] text-xs font-medium transition-colors duration-200'
  }
  return 'inline-flex items-center gap-1.5 rounded-md border px-2 py-[3px] text-xs font-medium shadow-sm backdrop-blur-sm transition-all duration-300'
})

// 并发状态样式
const concurrencyClass = computed(() => {
  const current = currentConcurrency.value
  const max = totalConcurrency.value

  if (current >= max) {
    return visualVariant.value === 'glass'
      ? 'border-rose-200/70 bg-rose-50 text-rose-600 dark:border-rose-400/20 dark:bg-rose-500/10 dark:text-rose-200'
      : 'border-rose-200/60 bg-rose-50 text-rose-600 shadow-[0_0_6px_rgba(225,29,72,0.1)] dark:bg-red-900/30 dark:text-red-400'
  }
  if (current > 0) {
    return visualVariant.value === 'glass'
      ? 'border-blue-200/70 bg-blue-50 text-blue-600 dark:border-sky-400/20 dark:bg-sky-500/10 dark:text-sky-200'
      : 'border-blue-200/60 bg-blue-50 text-blue-600 dark:bg-yellow-900/30 dark:text-yellow-400'
  }
  return visualVariant.value === 'glass'
    ? 'border-slate-200/75 bg-slate-50 text-slate-500 dark:border-slate-700/80 dark:bg-slate-800/70 dark:text-slate-300'
    : 'border-transparent bg-slate-100/50 text-slate-500 dark:bg-gray-800 dark:text-gray-400'
})

const showConcurrencyPulse = computed(() => {
  const current = currentConcurrency.value
  const max = totalConcurrency.value
  return current > 0 && current < max
})

// 窗口费用状态样式
const windowCostClass = computed(() => {
  if (!showWindowCost.value) return ''

  const current = currentWindowCost.value
  const limit = props.account.window_cost_limit || 0
  const reserve = props.account.window_cost_sticky_reserve || 10

  if (current >= limit + reserve) {
    return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
  }
  if (current >= limit) {
    return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
  }
  if (current >= limit * 0.8) {
    return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
  }
  return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
})

// 窗口费用提示文字
const windowCostTooltip = computed(() => {
  if (!showWindowCost.value) return ''

  const current = currentWindowCost.value
  const limit = props.account.window_cost_limit || 0
  const reserve = props.account.window_cost_sticky_reserve || 10

  if (current >= limit + reserve) {
    return t('admin.accounts.capacity.windowCost.blocked')
  }
  if (current >= limit) {
    return t('admin.accounts.capacity.windowCost.stickyOnly')
  }
  return t('admin.accounts.capacity.windowCost.normal')
})

// 会话限制状态样式
const sessionLimitClass = computed(() => {
  if (!showSessionLimit.value) return ''

  const current = activeSessions.value
  const max = props.account.max_sessions || 0

  if (current >= max) {
    return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
  }
  if (current >= max * 0.8) {
    return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
  }
  return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
})

// 会话限制提示文字
const sessionLimitTooltip = computed(() => {
  if (!showSessionLimit.value) return ''

  const current = activeSessions.value
  const max = props.account.max_sessions || 0
  const idle = props.account.session_idle_timeout_minutes || 5

  if (current >= max) {
    return t('admin.accounts.capacity.sessions.full', { idle })
  }
  return t('admin.accounts.capacity.sessions.normal', { idle })
})

// 是否显示 RPM 限制
const showRpmLimit = computed(() => {
  return (
    isAnthropicOAuthOrSetupToken.value &&
    props.account.base_rpm !== undefined &&
    props.account.base_rpm !== null &&
    props.account.base_rpm > 0
  )
})

// 当前 RPM 计数
const currentRPM = computed(() => props.account.current_rpm ?? 0)

// RPM 策略
const rpmStrategy = computed(() => props.account.rpm_strategy || 'tiered')

// RPM 策略标签
const rpmStrategyTag = computed(() => {
  return rpmStrategy.value === 'sticky_exempt' ? '[S]' : '[T]'
})

// RPM buffer 计算（与后端一致：base <= 0 时 buffer 为 0）
const rpmBuffer = computed(() => {
  const base = props.account.base_rpm || 0
  return props.account.rpm_sticky_buffer ?? (base > 0 ? Math.max(1, Math.floor(base / 5)) : 0)
})

// RPM 状态样式
const rpmClass = computed(() => {
  if (!showRpmLimit.value) return ''

  const current = currentRPM.value
  const base = props.account.base_rpm ?? 0
  const buffer = rpmBuffer.value

  if (rpmStrategy.value === 'tiered') {
    if (current >= base + buffer) {
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    }
    if (current >= base) {
      return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
    }
  } else {
    if (current >= base) {
      return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
    }
  }
  if (current >= base * 0.8) {
    return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
  }
  return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
})

// RPM 提示文字（增强版：显示策略、区域、缓冲区）
const rpmTooltip = computed(() => {
  if (!showRpmLimit.value) return ''

  const current = currentRPM.value
  const base = props.account.base_rpm ?? 0
  const buffer = rpmBuffer.value

  if (rpmStrategy.value === 'tiered') {
    if (current >= base + buffer) {
      return t('admin.accounts.capacity.rpm.tieredBlocked', { buffer })
    }
    if (current >= base) {
      return t('admin.accounts.capacity.rpm.tieredStickyOnly', { buffer })
    }
    if (current >= base * 0.8) {
      return t('admin.accounts.capacity.rpm.tieredWarning')
    }
    return t('admin.accounts.capacity.rpm.tieredNormal')
  } else {
    if (current >= base) {
      return t('admin.accounts.capacity.rpm.stickyExemptOver')
    }
    if (current >= base * 0.8) {
      return t('admin.accounts.capacity.rpm.stickyExemptWarning')
    }
    return t('admin.accounts.capacity.rpm.stickyExemptNormal')
  }
})

// 只要账号配置了额度，就展示额度徽章
const isQuotaEligible = computed(() => true)

const showDailyQuota = computed(() => {
  return isQuotaEligible.value && (props.account.quota_daily_limit ?? 0) > 0
})

const showWeeklyQuota = computed(() => {
  return isQuotaEligible.value && (props.account.quota_weekly_limit ?? 0) > 0
})

const showTotalQuota = computed(() => {
  return isQuotaEligible.value && (props.account.quota_limit ?? 0) > 0
})

const hasAirySecondaryMetrics = computed(() => {
  return airyMetrics.value.length > 0 || showDailyQuota.value || showWeeklyQuota.value || showTotalQuota.value
})

const formatCost = (value: number | null | undefined) => {
  if (value === null || value === undefined) return '0'
  return value.toFixed(2)
}

const resolveMetricTone = (current: number, limit: number, dangerCeiling: number = limit) => {
  if (current >= dangerCeiling) {
    return 'danger'
  }
  if (current >= limit) {
    return 'warning'
  }
  if (current >= limit * 0.8) {
    return 'warning'
  }
  return 'safe'
}

const airyMetrics = computed(() => {
  const metrics: Array<{
    key: string
    label: string
    value: string
    title: string
    tone: 'neutral' | 'safe' | 'warning' | 'danger'
    tag?: string
  }> = []

  if (showWindowCost.value) {
    const limit = props.account.window_cost_limit || 0
    const reserve = props.account.window_cost_sticky_reserve || 10
    metrics.push({
      key: 'window-cost',
      label: t('admin.accounts.capacity.cards.windowCost'),
      value: `$${formatQuotaCurrency(currentWindowCost.value)} / $${formatQuotaCurrency(limit)}`,
      title: windowCostTooltip.value,
      tone: resolveMetricTone(currentWindowCost.value, limit, limit + reserve)
    })
  }

  if (showSessionLimit.value) {
    const max = props.account.max_sessions || 0
    metrics.push({
      key: 'sessions',
      label: t('admin.accounts.capacity.cards.sessions'),
      value: `${activeSessions.value} / ${max}`,
      title: sessionLimitTooltip.value,
      tone: activeSessions.value >= max ? 'danger' : activeSessions.value >= max * 0.8 ? 'warning' : 'safe'
    })
  }

  if (showRpmLimit.value) {
    const base = props.account.base_rpm ?? 0
    const warningLimit = base * 0.8
    const overLimit = rpmStrategy.value === 'tiered' ? base + rpmBuffer.value : base
    metrics.push({
      key: 'rpm',
      label: t('admin.accounts.capacity.cards.rpm'),
      value: `${currentRPM.value} / ${base}`,
      title: rpmTooltip.value,
      tone: currentRPM.value >= overLimit ? 'danger' : currentRPM.value >= warningLimit ? 'warning' : 'safe',
      tag: rpmStrategyTag.value
    })
  }

  return metrics
})
</script>

<style scoped>
:global(.account-capacity-breathe) {
  animation: account-capacity-breathe 1.2s ease-in-out infinite alternate;
}

:global(.account-capacity-urgent) {
  animation: account-capacity-urgent 0.6s ease-in-out infinite alternate;
}

@keyframes account-capacity-breathe {
  0% { opacity: 1; }
  100% { opacity: 0.3; }
}

@keyframes account-capacity-urgent {
  0% { opacity: 1; transform: scaleY(1); }
  100% { opacity: 0.75; transform: scaleY(0.85); }
}
</style>
