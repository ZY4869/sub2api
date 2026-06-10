<template>
  <div
    :class="[
      'account-status-visual inline-flex min-w-0 max-w-full flex-col items-start justify-center whitespace-normal rounded-[1rem] border select-none',
      isSimpleMode ? 'w-full gap-1.5 px-2.5 py-2.5' : 'w-fit',
      compact && !isSimpleMode ? 'gap-1.5 px-3 py-3' : '',
      !compact && !isSimpleMode ? 'gap-2.5 px-4 py-3.5' : '',
      whiteSurfaceEnabled ? whiteSurfaceClass : toneStyles.surfaceClass
    ]"
    data-testid="account-status-visual-cell"
  >
    <div class="flex min-w-0 max-w-full items-center gap-1.5">
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

      <div
        v-if="isSimpleMode && visibleLimitBadges.length > 0"
        class="flex shrink-0 items-center gap-1"
        data-testid="account-status-visual-simple-icons"
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
          visual-variant="icon"
        />
      </div>

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
    </div>

    <div
      v-if="!isSimpleMode && visibleHelperText"
      :class="[
        'break-words text-[11px] font-bold leading-5',
        toneStyles.helperTextClass
      ]"
      data-testid="account-status-visual-helper"
    >
      {{ visibleHelperText }}
    </div>

    <div
      v-if="!isSimpleMode && visibleLimitBadges.length > 0"
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
import type { AccountStatusDisplayMode } from '@/types'
import AccountErrorTooltipButton from '@/components/account/AccountErrorTooltipButton.vue'
import AccountStatusLimitBadge from '@/components/account/AccountStatusLimitBadge.vue'
import { createAccountStatusPresentation } from '@/components/account/accountStatusPresentation'
import Icon from '@/components/icons/Icon.vue'
import { useRealtimeCountdownNow } from '@/composables/useRealtimeCountdownNow'
import AccountSegmentedCountdown from './AccountSegmentedCountdown.vue'
import { resolveAccountAiryStatus } from './accountAiryStatus'
import type { AccountGlassTone } from './accountVisualGlass'
import { useAccountStatusVisualDisplay } from './useAccountStatusVisualDisplay'

const props = withDefaults(defineProps<{
  account: Account
  visualStyle?: AccountVisualStyle
  whiteSurfaceEnabled?: boolean
  compact?: boolean
  displayMode?: AccountStatusDisplayMode
}>(), {
  visualStyle: 'airy',
  whiteSurfaceEnabled: false,
  compact: false,
  displayMode: 'detailed'
})

defineEmits<{
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
  rateLimitResumeText,
  overloadCountdown,
  visibleLimitBadges,
  limitBadgeLayoutClass,
} = createAccountStatusPresentation(accountRef, t, nowMs, nowDate, {
  showGenericRateLimitBadges: false,
})

const airyStatus = computed(() => resolveAccountAiryStatus(props.account, {
  nowMs: nowMs.value,
  isRateLimited: isRateLimited.value,
  isOverloaded: isOverloaded.value,
  isTempUnschedulable: isTempUnschedulable.value,
  hasError: hasError.value,
  activeLimitBadgeCount: visibleLimitBadges.value.length
}))

const visualTone = computed<AccountGlassTone>(() => airyStatus.value.tone)
const isSimpleMode = computed(() => props.displayMode === 'simple')

const statusIconName = computed(() => airyStatus.value.iconName)

const {
  toneStyles,
  whiteSurfaceClass,
  statusTitle,
  issueDetailText,
  countdownResetAt,
  visibleHelperText,
} = useAccountStatusVisualDisplay({
  account: accountRef,
  airyStatus,
  visualTone,
  nowDate,
  isRateLimited,
  isOverloaded,
  rateLimitResumeText,
  overloadCountdown,
  t,
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
