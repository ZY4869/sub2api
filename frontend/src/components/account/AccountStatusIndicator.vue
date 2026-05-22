<template>
  <div
    v-if="visualVariant === 'glass'"
    class="account-status-glass flex min-w-0 items-start gap-3 rounded-[1.1rem] border px-3.5 py-3"
    :class="whiteSurfaceEnabled ? glassWhiteSurfaceClass : glassToneStyles.surfaceClass"
  >
    <span
      :class="[
        'mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-xl',
        glassToneStyles.iconWrapClass,
      ]"
    >
      <Icon :name="glassStatusIconName" size="sm" :stroke-width="2.1" />
    </span>

    <div class="min-w-0 flex-1">
      <div class="flex flex-wrap items-center gap-2">
        <span
          :class="[
            'truncate text-sm font-black tracking-tight',
            glassToneStyles.titleClass,
          ]"
        >
          {{ glassStatusTitle }}
        </span>

        <button
          v-if="isTempUnschedulable"
          type="button"
          :class="[
            'inline-flex items-center rounded-full border px-2 py-1 text-[10px] font-semibold tracking-tight transition',
            glassToneStyles.statusBadgeClass,
          ]"
          :title="t('admin.accounts.status.viewTempUnschedDetails')"
          @click="handleTempUnschedClick"
        >
          {{ statusText }}
        </button>
        <span
          v-else
          :class="[
            'inline-flex items-center rounded-full border px-2 py-1 text-[10px] font-semibold tracking-tight',
            glassToneStyles.statusBadgeClass,
          ]"
        >
          {{ glassStatusBadgeText }}
        </span>

        <AccountErrorTooltipButton
          v-if="hasError && account.error_message"
          :message="account.error_message"
          :ariaLabel="t('admin.accounts.status.error')"
          button-class="rounded-full border border-rose-200/80 bg-white px-1.5 py-1 text-rose-500 transition hover:text-rose-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-rose-400/60 dark:border-rose-400/20 dark:bg-rose-500/10 dark:text-rose-200 dark:hover:text-rose-100"
        />
      </div>

      <div
        v-if="visibleLimitBadges.length > 0"
        data-test="account-limit-badges"
        class="mt-2 flex flex-wrap gap-1.5"
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

      <div
        v-if="glassCountdownResetAt"
        class="mt-2.5 flex flex-wrap items-center gap-2"
      >
        <div class="flex items-center gap-1.5">
          <span
            :class="[
              'flex h-5 w-5 items-center justify-center rounded-full border border-white/50 bg-white dark:border-white/10 dark:bg-white/5',
              glassToneStyles.iconClass,
            ]"
          >
            <Icon name="bolt" size="xs" :stroke-width="2.2" />
          </span>
          <AccountSegmentedCountdown
            :reset-at="glassCountdownResetAt"
            :tone="glassTone"
          />
        </div>
        <span
          v-if="glassCountdownSuffix"
          :class="['text-[11px] font-semibold', glassToneStyles.helperTextClass]"
        >
          {{ glassCountdownSuffix }}
        </span>
      </div>

      <div
        v-if="glassHelperText"
        :class="[
          glassCountdownResetAt ? 'mt-2' : 'mt-2.5',
          'text-[11px] font-semibold',
          glassToneStyles.helperTextClass,
        ]"
      >
        {{ glassHelperText }}
      </div>
    </div>
  </div>

  <div v-else class="flex flex-wrap items-center gap-2">
    <div v-if="isRateLimited" class="flex flex-col items-center gap-1">
      <span class="badge text-xs badge-warning">{{ rateLimitStatusLabel }}</span>
      <span class="text-[11px] text-gray-400 dark:text-gray-500">{{ rateLimitResumeText }}</span>
    </div>

    <div v-else-if="isOverloaded" class="flex flex-col items-center gap-1">
      <span class="badge text-xs badge-danger">{{ t('admin.accounts.status.overloaded') }}</span>
      <span class="text-[11px] text-gray-400 dark:text-gray-500">{{ overloadCountdown }}</span>
    </div>

    <template v-else>
      <button
        v-if="isTempUnschedulable"
        type="button"
        :class="['badge text-xs', statusClass, 'cursor-pointer']"
        :title="t('admin.accounts.status.viewTempUnschedDetails')"
        @click="handleTempUnschedClick"
      >
        {{ statusText }}
      </button>
      <span v-else :class="['badge text-xs', statusClass]">
        {{ statusText }}
      </span>
    </template>

    <AccountErrorTooltipButton
      v-if="hasError && account.error_message"
      :message="account.error_message"
      :ariaLabel="t('admin.accounts.status.error')"
    />

    <div
      v-if="visibleLimitBadges.length > 0"
      data-test="account-limit-badges"
      :class="limitBadgeLayoutClass"
    >
      <div
        v-for="item in visibleLimitBadges"
        :key="item.key"
        class="break-inside-avoid"
      >
        <AccountStatusLimitBadge
          :tone="item.tone"
          :label="item.label"
          :countdown="item.countdown"
          :tooltip="item.tooltip"
          :model="item.model"
          :model-display-name="item.modelDisplayName"
        />
      </div>
    </div>

    <div v-if="isOverloaded">
      <AccountStatusLimitBadge
        tone="red"
        label="529"
        :tooltip="t('admin.accounts.status.overloadedUntil', { time: formatTime(account.overload_until) })"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import AccountStatusLimitBadge from '@/components/account/AccountStatusLimitBadge.vue'
import AccountErrorTooltipButton from '@/components/account/AccountErrorTooltipButton.vue'
import Icon from '@/components/icons/Icon.vue'
import { useRealtimeCountdownNow } from '@/composables/useRealtimeCountdownNow'
import type { Account } from '@/types'
import AccountSegmentedCountdown from '@/components/admin/account/AccountSegmentedCountdown.vue'
import { formatTime } from '@/utils/format'
import {
  resolveAccountGlassToneStyles,
  type AccountGlassTone,
} from '@/components/admin/account/accountVisualGlass'
import { createAccountStatusPresentation } from './accountStatusPresentation'

const { t } = useI18n()
const { nowMs, nowDate } = useRealtimeCountdownNow('accounts')

const props = withDefaults(defineProps<{
  account: Account
  visualVariant?: 'default' | 'glass'
  whiteSurfaceEnabled?: boolean
}>(), {
  visualVariant: 'default',
  whiteSurfaceEnabled: false
})

const emit = defineEmits<{
  (e: 'show-temp-unsched', account: Account): void
}>()

const accountRef = toRef(props, 'account')
const {
  isRateLimited,
  isOverloaded,
  isTempUnschedulable,
  hasError,
  statusClass,
  statusText,
  rateLimitResumeText,
  rateLimitStatusLabel,
  overloadCountdown,
  visibleLimitBadges,
  limitBadgeLayoutClass,
} = createAccountStatusPresentation(accountRef, t, nowMs, nowDate)

const handleTempUnschedClick = () => {
  if (!isTempUnschedulable.value) return
  emit('show-temp-unsched', props.account)
}

const glassTone = computed<AccountGlassTone>(() => {
  if (hasError.value) return 'red'
  if (isOverloaded.value) return 'red'
  if (isRateLimited.value) {
    switch (props.account.rate_limit_reason) {
      case 'usage_5h':
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
  if (!props.account.schedulable || props.account.status === 'inactive') return 'slate'
  return 'emerald'
})

const glassToneStyles = computed(() => resolveAccountGlassToneStyles(glassTone.value))
const glassWhiteSurfaceClass = computed(() => {
  if (glassTone.value === 'red') {
    return 'border-rose-200/80 bg-white dark:border-rose-400/20 dark:bg-slate-900'
  }
  if (glassTone.value === 'orange' || glassTone.value === 'amber') {
    return 'border-amber-200/80 bg-white dark:border-amber-400/20 dark:bg-slate-900'
  }
  if (glassTone.value === 'indigo' || glassTone.value === 'sky') {
    return 'border-sky-200/80 bg-white dark:border-sky-400/20 dark:bg-slate-900'
  }
  if (glassTone.value === 'emerald') {
    return 'border-emerald-200/80 bg-white dark:border-emerald-400/20 dark:bg-slate-900'
  }
  return 'border-slate-200/85 bg-white dark:border-slate-700/80 dark:bg-slate-900'
})

const glassStatusIconName = computed(() => {
  if (hasError.value) return 'exclamationTriangle'
  if (isOverloaded.value) return 'exclamationTriangle'
  if (isRateLimited.value) return 'clock'
  if (isTempUnschedulable.value) return 'clock'
  if (!props.account.schedulable || props.account.status === 'inactive') return 'clock'
  return 'checkCircle'
})

const glassStatusTitle = computed(() => {
  if (hasError.value) return statusText.value
  if (isOverloaded.value) return t('admin.accounts.status.overloaded')
  if (isRateLimited.value) return rateLimitStatusLabel.value
  return statusText.value
})

const glassStatusBadgeText = computed(() => {
  if (hasError.value) return t('admin.accounts.status.error')
  if (isOverloaded.value) return '529'
  if (isRateLimited.value) return rateLimitStatusLabel.value
  return statusText.value
})

const glassCountdownResetAt = computed(() => {
  if (isRateLimited.value && props.account.rate_limit_reset_at) {
    return props.account.rate_limit_reset_at
  }
  if (isOverloaded.value && props.account.overload_until) {
    return props.account.overload_until
  }
  return null
})

const glassCountdownSuffix = computed(() => {
  return ''
})

const glassHelperText = computed(() => {
  if (isRateLimited.value) return rateLimitResumeText.value
  if (isOverloaded.value) return overloadCountdown.value || ''
  return ''
})
</script>
