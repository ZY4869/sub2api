<template>
  <div
    :class="[
      'account-status-visual flex min-w-[220px] max-w-[240px] flex-col justify-center gap-2 whitespace-normal rounded-[1rem] border px-3.5 py-3 select-none',
      whiteSurfaceEnabled ? whiteSurfaceClass : toneStyles.surfaceClass
    ]"
    data-testid="account-status-visual-cell"
  >
    <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
      <div class="flex min-w-0 items-center gap-1.5">
        <span
          :class="[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-xl',
            toneStyles.iconWrapClass
          ]"
        >
          <Icon :name="statusIconName" size="sm" :stroke-width="2.2" />
        </span>
        <span
          :class="[
            'truncate text-[13px] font-extrabold tracking-tight',
            toneStyles.titleClass
          ]"
        >
          {{ statusTitle }}
        </span>
      </div>

      <button
        v-if="isTempUnschedulable"
        type="button"
        :class="statusTagClass"
        :title="t('admin.accounts.status.viewTempUnschedDetails')"
        @click="emit('show-temp-unsched', account)"
      >
        {{ statusTagText }}
      </button>
      <span v-else :class="statusTagClass">
        {{ statusTagText }}
      </span>

      <AccountErrorTooltipButton
        v-if="hasError && account.error_message"
        :message="account.error_message"
        :ariaLabel="t('admin.accounts.status.error')"
        button-class="rounded-full border border-rose-200/80 bg-white px-1.5 py-1 text-rose-500 transition hover:text-rose-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-rose-400/60 dark:border-rose-400/20 dark:bg-rose-500/10 dark:text-rose-200 dark:hover:text-rose-100"
      />
    </div>

    <div
      v-if="countdownResetAt"
      class="flex flex-wrap items-center gap-1.5"
      data-testid="account-status-visual-countdown"
    >
      <span
        :class="[
          'relative flex h-4 w-4 shrink-0 items-center justify-center',
          toneStyles.iconClass
        ]"
      >
        <Icon name="bolt" size="xs" :stroke-width="2.5" class="opacity-25" />
        <Icon
          name="bolt"
          size="xs"
          :stroke-width="2.5"
          class="absolute account-status-visual-pulse"
        />
      </span>
      <AccountSegmentedCountdown :reset-at="countdownResetAt" :tone="visualTone" />
      <span :class="['text-[10px] font-bold opacity-70', toneStyles.helperTextClass]">
        {{ countdownSuffix }}
      </span>
    </div>

    <div
      v-if="helperText"
      :class="[
        'text-[11px] font-bold leading-snug',
        toneStyles.helperTextClass
      ]"
      data-testid="account-status-visual-helper"
    >
      {{ helperText }}
    </div>

    <div
      v-if="visibleLimitBadges.length > 0"
      data-test="account-limit-badges"
      class="flex flex-wrap gap-1.5"
    >
      <AccountStatusLimitBadge
        v-for="item in visibleLimitBadges"
        :key="item.key"
        :tone="item.tone"
        :label="item.label"
        :countdown="item.countdown"
        :tooltip="item.tooltip"
        :model="item.model"
        :model-display-name="item.modelDisplayName"
        visual-variant="glass"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account, AccountVisualStyle } from '@/types'
import AccountErrorTooltipButton from '@/components/account/AccountErrorTooltipButton.vue'
import AccountStatusLimitBadge from '@/components/account/AccountStatusLimitBadge.vue'
import { createAccountStatusPresentation } from '@/components/account/accountStatusPresentation'
import Icon from '@/components/icons/Icon.vue'
import { useRealtimeCountdownNow } from '@/composables/useRealtimeCountdownNow'
import AccountSegmentedCountdown from './AccountSegmentedCountdown.vue'
import {
  resolveAccountGlassToneStyles,
  type AccountGlassTone,
} from './accountVisualGlass'

type StatusIconName = 'exclamationTriangle' | 'clock' | 'checkCircle'

const props = withDefaults(defineProps<{
  account: Account
  visualStyle?: AccountVisualStyle
  whiteSurfaceEnabled?: boolean
}>(), {
  visualStyle: 'airy',
  whiteSurfaceEnabled: false
})

const emit = defineEmits<{
  'show-temp-unsched': [account: Account]
}>()

const { t } = useI18n()
const { nowMs, nowDate } = useRealtimeCountdownNow('accounts')
const accountRef = toRef(props, 'account')
const {
  isRateLimited,
  isOverloaded,
  isTempUnschedulable,
  hasError,
  statusText,
  rateLimitResumeText,
  rateLimitStatusLabel,
  overloadCountdown,
  visibleLimitBadges,
} = createAccountStatusPresentation(accountRef, t, nowMs, nowDate)

const isPaused = computed(() => !props.account.schedulable || props.account.status === 'inactive')
const isBlacklisted = computed(() => props.account.lifecycle_state === 'blacklisted')

const visualTone = computed<AccountGlassTone>(() => {
  if (hasError.value || isBlacklisted.value || isOverloaded.value) return 'red'
  if (isRateLimited.value) {
    switch (props.account.rate_limit_reason) {
      case 'usage_5h':
      case 'rate_429':
        return 'orange'
      case 'usage_7d':
        return 'indigo'
      case 'usage_7d_all':
        return 'amber'
      default:
        return 'amber'
    }
  }
  if (isTempUnschedulable.value) return 'sky'
  if (isPaused.value) return 'slate'
  return 'emerald'
})

const toneStyles = computed(() => resolveAccountGlassToneStyles(visualTone.value))
const whiteSurfaceClass = computed(() => {
  switch (visualTone.value) {
    case 'red':
      return 'border-rose-200/80 bg-white dark:border-rose-400/20 dark:bg-slate-900'
    case 'orange':
    case 'amber':
      return 'border-amber-200/80 bg-white dark:border-amber-400/20 dark:bg-slate-900'
    case 'indigo':
    case 'sky':
      return 'border-sky-200/80 bg-white dark:border-sky-400/20 dark:bg-slate-900'
    case 'emerald':
      return 'border-emerald-200/80 bg-white dark:border-emerald-400/20 dark:bg-slate-900'
    default:
      return 'border-slate-200/85 bg-white dark:border-slate-700/80 dark:bg-slate-900'
  }
})

const statusIconName = computed<StatusIconName>(() => {
  if (hasError.value || isBlacklisted.value || isOverloaded.value) return 'exclamationTriangle'
  if (isRateLimited.value || isTempUnschedulable.value || isPaused.value) return 'clock'
  return 'checkCircle'
})

const statusTitle = computed(() => {
  if (isBlacklisted.value) return t('admin.accounts.lifecycle.blacklisted')
  if (hasError.value) return t('admin.accounts.status.error')
  if (isOverloaded.value) return t('admin.accounts.status.overloaded')
  if (isRateLimited.value) {
    switch (props.account.rate_limit_reason) {
      case 'usage_5h':
        return t('admin.accounts.status.visualUsage5hTitle')
      case 'usage_7d':
      case 'usage_7d_all':
        return t('admin.accounts.status.visualUsage7dTitle')
      default:
        return rateLimitStatusLabel.value
    }
  }
  if (isTempUnschedulable.value) return t('admin.accounts.status.tempUnschedulable')
  if (isPaused.value) return t('admin.accounts.status.paused')
  return t('admin.accounts.status.active')
})

const statusTagText = computed(() => {
  if (isBlacklisted.value) return t('admin.accounts.lifecycle.blacklisted')
  if (hasError.value) return t('admin.accounts.status.error')
  if (isOverloaded.value) return '529'
  if (isRateLimited.value) {
    switch (props.account.rate_limit_reason) {
      case 'usage_5h':
        return t('admin.accounts.status.visualUsage5hTag')
      case 'usage_7d':
        return t('admin.accounts.status.visualUsage7dTag')
      case 'usage_7d_all':
        return t('admin.accounts.status.visualUsage7dAllTag')
      default:
        return '429'
    }
  }
  if (isTempUnschedulable.value) return statusText.value
  if (isPaused.value) return t('admin.accounts.status.paused')
  return t('admin.accounts.status.active')
})

const statusTagClass = computed(() => [
  'inline-flex shrink-0 items-center rounded-full border px-2 py-1 text-[10px] font-bold tracking-tight transition',
  toneStyles.value.statusBadgeClass
])

const countdownResetAt = computed(() => {
  if (isRateLimited.value && props.account.rate_limit_reset_at) return props.account.rate_limit_reset_at
  if (isOverloaded.value && props.account.overload_until) return props.account.overload_until
  return null
})

const countdownSuffix = computed(() => {
  if (isRateLimited.value) return t('admin.accounts.status.visualAfterResume')
  if (isOverloaded.value) return t('admin.accounts.status.visualAfterRelease')
  return ''
})

const helperText = computed(() => {
  if (isRateLimited.value) return rateLimitResumeText.value
  if (isOverloaded.value) return overloadCountdown.value || ''
  return ''
})
</script>

<style scoped>
.account-status-visual-pulse :deep(path) {
  stroke-dasharray: 15 100;
  animation: account-status-visual-sweep 2s linear infinite;
  stroke-linecap: round;
  stroke-linejoin: round;
}

@keyframes account-status-visual-sweep {
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
