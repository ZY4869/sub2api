<template>
  <div ref="rootRef">
    <div v-if="presentation.meta.antigravityTierLabel" class="mb-1 flex items-center gap-1">
      <span
        :class="[
          'inline-block rounded px-1.5 py-0.5 text-[10px] font-medium',
          presentation.meta.antigravityTierClass,
        ]"
      >
        {{ presentation.meta.antigravityTierLabel }}
      </span>
      <button
        v-if="presentation.meta.hasIneligibleTiers"
        type="button"
        class="inline-flex cursor-help items-center text-red-500"
        @mouseenter="handleTooltipEnter('ineligible', $event)"
        @mouseleave="scheduleTooltipHide"
        @focusin="handleTooltipEnter('ineligible', $event)"
        @focusout="handleTriggerFocusOut"
        @keydown.esc.prevent="hideTooltip"
      >
        <svg class="h-3.5 w-3.5 text-red-500" fill="currentColor" viewBox="0 0 20 20">
          <path
            fill-rule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
            clip-rule="evenodd"
          />
        </svg>
      </button>
    </div>

    <div v-if="presentation.meta.protocolGatewayBadgeLabel" class="mb-1 flex items-center gap-1">
      <span
        :class="[
          'inline-block rounded px-1.5 py-0.5 text-[10px] font-medium',
          presentation.meta.protocolGatewayBadgeClass,
        ]"
      >
        {{ presentation.meta.protocolGatewayBadgeLabel }}
      </span>
    </div>

    <div v-if="presentation.meta.geminiAuthTypeLabel" class="mb-1 flex items-center gap-1">
      <span
        :class="[
          'inline-block rounded px-1.5 py-0.5 text-[10px] font-medium',
          presentation.meta.geminiTierClass,
        ]"
      >
        {{ presentation.meta.geminiAuthTypeLabel }}
      </span>
      <button
        type="button"
        class="inline-flex cursor-help items-center"
        @mouseenter="handleTooltipEnter('gemini-policy', $event)"
        @mouseleave="scheduleTooltipHide"
        @focusin="handleTooltipEnter('gemini-policy', $event)"
        @focusout="handleTriggerFocusOut"
        @keydown.esc.prevent="hideTooltip"
      >
        <svg
          class="h-3.5 w-3.5 text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path
            fill-rule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-3a1 1 0 00-.867.5 1 1 0 11-1.731-1A3 3 0 0113 8a3.001 3.001 0 01-2 2.83V11a1 1 0 11-2 0v-1a1 1 0 011-1 1 1 0 100-2zm0 8a1 1 0 100-2 1 1 0 000 2z"
            clip-rule="evenodd"
          />
        </svg>
      </button>
    </div>

    <div v-if="presentation.state === 'loading'" class="space-y-1.5">
      <div v-for="index in skeletonRows" :key="index" class="flex items-center gap-1">
        <div class="h-3 w-[32px] animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
        <div class="h-1.5 w-8 animate-pulse rounded-full bg-gray-200 dark:bg-gray-700"></div>
        <div class="h-3 w-[32px] animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
      </div>
    </div>

    <div v-else-if="presentation.state === 'error'" class="text-xs text-red-500">
      {{ presentation.error }}
    </div>

    <div v-else-if="presentation.state === 'bars'" class="space-y-1">
      <UsageProgressBar
        v-for="row in presentation.windowRows"
        :key="row.key"
        :label="row.label"
        :utilization="row.utilization"
        :resets-at="row.resetsAt"
        :remaining-seconds="row.remainingSeconds"
        :window-stats="row.windowStats"
        :color="row.color"
        :inline-reset="row.inlineRemaining"
        :display-mode="accountUsageDisplayMode"
      />
      <p
        v-if="presentation.meta.snapshotUpdatedAtText"
        class="text-[9px] leading-tight text-gray-400 dark:text-gray-500"
        :title="presentation.meta.snapshotUpdatedAtTooltip || undefined"
      >
        {{ t('admin.accounts.usageWindow.snapshotUpdatedAt', { time: presentation.meta.snapshotUpdatedAtText }) }}
      </p>
      <div v-if="presentation.meta.sampledBadgeLabel" class="mt-1">
        <button
          type="button"
          class="inline-flex cursor-help items-center rounded-full border border-amber-300/80 bg-amber-50 px-1.5 py-0.5 text-[9px] font-semibold tracking-wide text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200"
          @mouseenter="handleTooltipEnter('sampled', $event)"
          @mouseleave="scheduleTooltipHide"
          @focusin="handleTooltipEnter('sampled', $event)"
          @focusout="handleTriggerFocusOut"
          @keydown.esc.prevent="hideTooltip"
        >
          {{ presentation.meta.sampledBadgeLabel }}
        </button>
      </div>
      <p v-if="presentation.meta.noteText" class="mt-1 text-[9px] italic leading-tight text-gray-400 dark:text-gray-500">
        * {{ presentation.meta.noteText }}
      </p>
    </div>

    <div v-else-if="presentation.state === 'unlimited'" class="text-xs text-gray-400">
      {{ t('admin.accounts.gemini.rateLimit.unlimited') }}
    </div>

    <div v-else class="text-xs text-gray-400">-</div>
  </div>

  <Teleport to="body">
    <div
      v-if="tooltipVisible && activeTooltipKey"
      ref="tooltipRef"
      class="fixed z-[99999] max-w-[min(20rem,calc(100vw-1.5rem))] rounded-lg bg-gray-900 px-3 py-2 text-xs leading-relaxed text-white shadow-xl ring-1 ring-white/10 dark:bg-gray-800"
      :style="tooltipStyle"
      @mouseenter="cancelTooltipHide"
      @mouseleave="scheduleTooltipHide"
      @focusin="cancelTooltipHide"
      @focusout="handleTooltipFocusOut"
    >
      <div
        :class="[
          'absolute left-1/2 h-2 w-2 -translate-x-1/2 rotate-45 bg-gray-900 dark:bg-gray-800',
          tooltipPlacement === 'top' ? '-bottom-1' : '-top-1',
        ]"
      ></div>

      <template v-if="activeTooltipKey === 'ineligible'">
        {{ t('admin.accounts.ineligibleWarning') }}
      </template>

      <template v-else-if="activeTooltipKey === 'sampled'">
        {{ presentation.meta.sampledBadgeTooltip }}
      </template>

      <template v-else-if="activeTooltipKey === 'gemini-policy'">
        <div class="space-y-2 whitespace-normal break-words">
          <div class="font-semibold">{{ t('admin.accounts.gemini.quotaPolicy.title') }}</div>
          <div class="text-gray-200 dark:text-gray-300">{{ t('admin.accounts.gemini.quotaPolicy.note') }}</div>
          <div class="space-y-1">
            <div class="font-medium">{{ presentation.meta.geminiQuotaPolicyChannel }}</div>
            <div>{{ presentation.meta.geminiQuotaPolicyLimits }}</div>
          </div>
          <a
            v-if="presentation.meta.geminiQuotaPolicyDocsUrl"
            :href="presentation.meta.geminiQuotaPolicyDocsUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center text-blue-300 underline underline-offset-2 hover:text-blue-200"
          >
            {{ t('admin.accounts.gemini.quotaPolicy.columns.docs') }}
          </a>
        </div>
      </template>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account, WindowStats } from '@/types'
import { useAccountUsagePresentation } from '@/composables/useAccountUsagePresentation'
import { useAccountUsageDisplayMode } from '@/composables/useAccountUsageDisplayMode'
import { useFloatingTooltip } from '@/composables/useFloatingTooltip'
import { useViewportAutoLoadGate } from '@/composables/useViewportAutoLoadGate'
import UsageProgressBar from './UsageProgressBar.vue'

const props = withDefaults(
  defineProps<{
    account: Account
    todayStats?: WindowStats | null
    todayStatsLoading?: boolean
    manualRefreshToken?: number
  }>(),
  {
    todayStats: null,
    todayStatsLoading: false,
    manualRefreshToken: 0
  }
)

const { t } = useI18n()
const { rootRef, autoLoadEnabled } = useViewportAutoLoadGate()
const { accountUsageDisplayMode } = useAccountUsageDisplayMode()
const { presentation, requestAutoLoad, shouldFetchUsage } = useAccountUsagePresentation(() => props.account, {
  autoLoadEnabled,
})
const {
  tooltipVisible,
  tooltipRef,
  tooltipPlacement,
  tooltipStyle,
  showFloatingTooltip,
  hideFloatingTooltip,
} = useFloatingTooltip()
const activeTooltipKey = ref<'ineligible' | 'gemini-policy' | 'sampled' | null>(null)
let hideTooltipTimer: number | null = null

const skeletonRows = computed(() => {
  return Array.from({ length: presentation.value.meta.loadingRows }, (_, index) => index + 1)
})

const cancelTooltipHide = () => {
  if (hideTooltipTimer) {
    window.clearTimeout(hideTooltipTimer)
    hideTooltipTimer = null
  }
}

const hideTooltip = () => {
  cancelTooltipHide()
  activeTooltipKey.value = null
  hideFloatingTooltip()
}

const scheduleTooltipHide = () => {
  cancelTooltipHide()
  hideTooltipTimer = window.setTimeout(() => {
    hideTooltip()
  }, 80)
}

const handleTooltipEnter = async (
  key: 'ineligible' | 'gemini-policy' | 'sampled',
  event: MouseEvent | FocusEvent,
) => {
  const triggerEl = event.currentTarget as HTMLElement | null
  if (!triggerEl) return

  cancelTooltipHide()
  activeTooltipKey.value = key
  await showFloatingTooltip(triggerEl)
}

const handleTriggerFocusOut = (event: FocusEvent) => {
  const nextTarget = event.relatedTarget as Node | null
  if (tooltipRef.value?.contains(nextTarget)) return
  scheduleTooltipHide()
}

const handleTooltipFocusOut = (event: FocusEvent) => {
  const nextTarget = event.relatedTarget as Node | null
  if (tooltipRef.value?.contains(nextTarget)) return
  scheduleTooltipHide()
}

watch(
  () => props.manualRefreshToken,
  (nextToken, prevToken) => {
    if (nextToken === prevToken) return
    if (!shouldFetchUsage.value) return

    requestAutoLoad()
  }
)

onBeforeUnmount(() => {
  cancelTooltipHide()
})
</script>
