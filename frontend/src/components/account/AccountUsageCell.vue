<template>
  <div>
    <div v-if="presentation.meta.antigravityTierLabel" class="mb-1 flex items-center gap-1">
      <span
        :class="[
          'inline-block rounded px-1.5 py-0.5 text-[10px] font-medium',
          presentation.meta.antigravityTierClass,
        ]"
      >
        {{ presentation.meta.antigravityTierLabel }}
      </span>
      <span v-if="presentation.meta.hasIneligibleTiers" class="group relative cursor-help">
        <svg class="h-3.5 w-3.5 text-red-500" fill="currentColor" viewBox="0 0 20 20">
          <path
            fill-rule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
            clip-rule="evenodd"
          />
        </svg>
        <span
          class="pointer-events-none absolute left-0 top-full z-50 mt-1 w-80 whitespace-normal break-words rounded bg-gray-900 px-3 py-2 text-xs leading-relaxed text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 dark:bg-gray-700"
        >
          {{ t('admin.accounts.ineligibleWarning') }}
        </span>
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
      <span class="group relative cursor-help">
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
        <span
          class="pointer-events-none absolute left-0 top-full z-50 mt-1 w-80 whitespace-normal break-words rounded bg-gray-900 px-3 py-2 text-xs leading-relaxed text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 dark:bg-gray-700"
        >
          <div class="mb-1 font-semibold">{{ t('admin.accounts.gemini.quotaPolicy.title') }}</div>
          <div class="mb-2 text-gray-300">{{ t('admin.accounts.gemini.quotaPolicy.note') }}</div>
          <div class="space-y-1">
            <div><strong>{{ presentation.meta.geminiQuotaPolicyChannel }}:</strong></div>
            <div class="pl-2">- {{ presentation.meta.geminiQuotaPolicyLimits }}</div>
            <div class="mt-2">
              <a
                :href="presentation.meta.geminiQuotaPolicyDocsUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="text-blue-400 hover:text-blue-300 underline"
              >
                {{ t('admin.accounts.gemini.quotaPolicy.columns.docs') }} ->
              </a>
            </div>
          </div>
        </span>
      </span>
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
      />
      <p
        v-if="presentation.meta.snapshotUpdatedAtText"
        class="text-[9px] leading-tight text-gray-400 dark:text-gray-500"
        :title="presentation.meta.snapshotUpdatedAtTooltip || undefined"
      >
        {{ t('admin.accounts.usageWindow.snapshotUpdatedAt', { time: presentation.meta.snapshotUpdatedAtText }) }}
      </p>
      <p v-if="presentation.meta.noteText" class="mt-1 text-[9px] italic leading-tight text-gray-400 dark:text-gray-500">
        * {{ presentation.meta.noteText }}
      </p>
    </div>

    <div v-else-if="presentation.state === 'unlimited'" class="text-xs text-gray-400">
      {{ t('admin.accounts.gemini.rateLimit.unlimited') }}
    </div>

    <div v-else class="text-xs text-gray-400">-</div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account, WindowStats } from '@/types'
import { useAccountUsagePresentation } from '@/composables/useAccountUsagePresentation'
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
const { presentation, loadUsage, shouldFetchUsage } = useAccountUsagePresentation(() => props.account)

const skeletonRows = computed(() => {
  return Array.from({ length: presentation.value.meta.loadingRows }, (_, index) => index + 1)
})

watch(
  () => props.manualRefreshToken,
  (nextToken, prevToken) => {
    if (nextToken === prevToken) return
    if (!shouldFetchUsage.value) return

    loadUsage().catch((error) => {
      console.error('Failed to refresh usage after manual refresh:', error)
    })
  }
)
</script>
