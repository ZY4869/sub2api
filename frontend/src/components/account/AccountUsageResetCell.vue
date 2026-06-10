<template>
  <div v-if="presentation.state === 'loading'" class="space-y-1.5">
    <div v-for="index in presentation.meta.loadingRows" :key="index" class="flex items-center gap-2">
      <div class="h-3 w-[32px] animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
      <div class="h-3 w-20 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
    </div>
  </div>

  <div v-else-if="presentation.state === 'error'" class="text-xs text-red-500">
    {{ presentation.error }}
  </div>

  <div v-else-if="presentation.resetRows.length > 0" class="max-w-full space-y-1.5 overflow-visible">
    <div
      v-for="row in presentation.resetRows"
      :key="row.key"
      class="grid min-w-0 max-w-full grid-cols-[minmax(42px,auto)_1fr] items-start gap-x-2 gap-y-1 text-[10px] tabular-nums"
    >
      <span
        :title="row.label"
        data-testid="account-usage-reset-window-label"
        :class="[
          'min-w-[42px] max-w-[82px] shrink-0 rounded-full border px-1.5 py-1 text-center font-bold leading-none',
          resetLabelClass(row.label)
        ]"
      >
        {{ row.label }}
      </span>

      <span
        v-if="formatResetValue(row.resetsAt, row.remainingSeconds)"
        class="flex min-w-0 flex-1 flex-wrap items-center gap-x-1.5 gap-y-1 overflow-visible text-gray-700 dark:text-gray-300"
        :title="formatResetValue(row.resetsAt, row.remainingSeconds)?.tooltip || undefined"
      >
        <Icon
          name="clock"
          size="xs"
          class="shrink-0 text-gray-400 dark:text-gray-500"
        />
        <span
          class="shrink-0 whitespace-nowrap rounded-full bg-gray-100 px-1.5 py-0.5 font-medium leading-none text-gray-700 dark:bg-gray-700 dark:text-gray-200"
        >
          {{ formatResetValue(row.resetsAt, row.remainingSeconds)?.countdown }}
        </span>
        <span class="shrink-0 text-gray-400 dark:text-gray-500">·</span>
        <span class="min-w-[112px] flex-1 whitespace-normal break-words font-semibold leading-snug text-gray-700 dark:text-gray-200">
          {{ formatResetValue(row.resetsAt, row.remainingSeconds)?.absolute }}
        </span>
      </span>

      <span v-else class="text-gray-400 dark:text-gray-500">-</span>
    </div>
  </div>

  <div v-else class="text-xs text-gray-400">-</div>
</template>

<script setup lang="ts">
import { useAccountUsagePresentation } from '@/composables/useAccountUsagePresentation'
import { useRealtimeCountdownNow } from '@/composables/useRealtimeCountdownNow'
import {
  formatLocalAbsoluteTime,
  formatLocalTimestamp,
  formatResetCountdown,
  parseEffectiveResetAt,
} from '@/utils/usageResetTime'
import type { Account } from '@/types'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  account: Account
}>()

const { t } = useI18n()
const { nowDate } = useRealtimeCountdownNow('accounts')
const { presentation } = useAccountUsagePresentation(() => props.account)

function formatResetValue(
  resetsAt: string | null,
  remainingSeconds?: number | null,
): {
  countdown: string
  absolute: string
  tooltip: string
} | null {
  const effectiveResetAt = parseEffectiveResetAt(
    resetsAt,
    remainingSeconds ?? null,
    nowDate.value,
  )
  if (!effectiveResetAt) return null

  return {
    countdown: formatResetCountdown(
      effectiveResetAt,
      nowDate.value,
      t('admin.accounts.usageWindow.now'),
    ),
    absolute: formatLocalAbsoluteTime(effectiveResetAt, nowDate.value, {
      today: t('dates.today'),
      tomorrow: t('dates.tomorrow'),
    }),
    tooltip: formatLocalTimestamp(effectiveResetAt),
  }
}

function resetLabelClass(label: string): string {
  const normalized = label.trim().toUpperCase()
  if (normalized.includes('5H')) {
    return 'border-indigo-200 bg-indigo-50 text-indigo-700 dark:border-indigo-400/25 dark:bg-indigo-400/10 dark:text-indigo-100'
  }
  if (normalized.includes('30D')) {
    return 'border-green-200 bg-green-50 text-green-700 dark:border-green-400/25 dark:bg-green-400/10 dark:text-green-100'
  }
  if (normalized.includes('7D')) {
    return 'border-orange-200 bg-orange-50 text-orange-700 dark:border-orange-400/25 dark:bg-orange-400/10 dark:text-orange-100'
  }
  return 'border-slate-200 bg-slate-50 text-slate-600 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200'
}
</script>
