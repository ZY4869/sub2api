<template>
  <div
    :class="[
      'account-status-visual flex min-w-0 max-w-full flex-col justify-center whitespace-normal rounded-[1rem] border select-none',
      compact ? 'gap-1.5 px-3 py-3' : 'gap-2.5 px-4 py-3.5',
      whiteSurfaceEnabled ? whiteSurfaceClass : toneStyles.surfaceClass
    ]"
    data-testid="account-status-visual-cell"
  >
    <div class="flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-1">
      <div class="flex min-w-0 items-center gap-1.5">
        <span
          :class="[
            'flex shrink-0 items-center justify-center rounded-xl',
            compact ? 'h-7 w-7' : 'h-8 w-8',
            toneStyles.iconWrapClass
          ]"
        >
          <Icon :name="statusIconName" size="sm" :stroke-width="2.2" />
        </span>
        <span
          :class="[
            'min-w-0 truncate text-[13px] font-extrabold tracking-tight',
            toneStyles.titleClass
          ]"
          :title="statusTitle"
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
        <span class="min-w-0 truncate">{{ statusTagText }}</span>
      </button>
      <span v-else :class="statusTagClass" :title="statusTagText">
        <span class="min-w-0 truncate">{{ statusTagText }}</span>
      </span>

      <AccountErrorTooltipButton
        v-if="issueDetailText"
        :message="issueDetailText"
        :ariaLabel="t('admin.accounts.status.viewIssueDetails')"
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
      v-if="visibleHelperText"
      :class="[
        'break-words text-[11px] font-bold leading-5',
        toneStyles.helperTextClass
      ]"
      data-testid="account-status-visual-helper"
    >
      {{ visibleHelperText }}
    </div>

    <div
      v-if="visibleLimitBadges.length > 0"
      data-test="account-limit-badges"
      :class="limitBadgeLayoutClass"
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
import { resolveAccountAiryStatus } from './accountAiryStatus'
import type { AiryStatusKind } from './accountAiryStatusTypes'
import {
  resolveAccountGlassToneStyles,
  type AccountGlassTone,
} from './accountVisualGlass'

const props = withDefaults(defineProps<{
  account: Account
  visualStyle?: AccountVisualStyle
  whiteSurfaceEnabled?: boolean
  compact?: boolean
}>(), {
  visualStyle: 'airy',
  whiteSurfaceEnabled: false,
  compact: false
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
  overloadCountdown,
  visibleLimitBadges,
  limitBadgeLayoutClass,
} = createAccountStatusPresentation(accountRef, t, nowMs, nowDate)

const airyStatus = computed(() => resolveAccountAiryStatus(props.account, {
  nowMs: nowMs.value,
  isRateLimited: isRateLimited.value,
  isOverloaded: isOverloaded.value,
  isTempUnschedulable: isTempUnschedulable.value,
  hasError: hasError.value,
  activeLimitBadgeCount: visibleLimitBadges.value.length
}))

const visualTone = computed<AccountGlassTone>(() => airyStatus.value.tone)

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
    case 'purple':
    case 'teal':
      return 'border-sky-200/80 bg-white dark:border-sky-400/20 dark:bg-slate-900'
    case 'emerald':
      return 'border-emerald-200/80 bg-white dark:border-emerald-400/20 dark:bg-slate-900'
    default:
      return 'border-slate-200/85 bg-white dark:border-slate-700/80 dark:bg-slate-900'
  }
})

const statusIconName = computed(() => airyStatus.value.iconName)

const statusTitle = computed(() => {
  return t(airyStatus.value.titleKey)
})

const issueDetailStatusKinds = new Set<AiryStatusKind>([
  'banned',
  'locked',
  'maintenance',
  'offline',
  'overdue',
  'degraded',
  'captcha',
  'syncing',
  'error',
])

const firstText = (...values: Array<unknown>) => {
  for (const value of values) {
    const text = String(value || '').trim()
    if (text) return text
  }
  return ''
}

const issueDetailText = computed(() => {
  if (!issueDetailStatusKinds.has(airyStatus.value.kind)) return ''
  if (airyStatus.value.helperKey) return t(airyStatus.value.helperKey)
  return firstText(
    airyStatus.value.helper,
  )
})

const statusTagText = computed(() => {
  if (airyStatus.value.tagFallback) return airyStatus.value.tagFallback
  if (airyStatus.value.kind === 'tempUnschedulable') return statusText.value
  return t(airyStatus.value.tagKey)
})

const statusTagClass = computed(() => [
  'inline-flex max-w-[82px] shrink-0 items-center rounded-full border px-2 py-1 text-[10px] font-bold tracking-tight transition',
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
  if (airyStatus.value.helperKey) return t(airyStatus.value.helperKey)
  return airyStatus.value.helper || ''
})

const visibleHelperText = computed(() => {
  if (issueDetailText.value) return ''
  return helperText.value
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
