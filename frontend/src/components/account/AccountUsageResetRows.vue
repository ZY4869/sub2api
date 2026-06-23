<template>
  <div
    v-for="row in rows"
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
      v-if="formatResetValue(row.resetsAt, row.remainingSeconds, nowDate)"
      class="flex min-w-0 flex-1 flex-wrap items-center gap-x-1.5 gap-y-1 overflow-visible text-gray-700 dark:text-gray-300"
      :title="formatResetValue(row.resetsAt, row.remainingSeconds, nowDate)?.tooltip || undefined"
    >
      <Icon
        name="clock"
        size="xs"
        class="shrink-0 text-gray-400 dark:text-gray-500"
      />
      <span
        class="shrink-0 whitespace-nowrap rounded-full bg-gray-100 px-1.5 py-0.5 font-medium leading-none text-gray-700 dark:bg-gray-700 dark:text-gray-200"
      >
        {{ formatResetValue(row.resetsAt, row.remainingSeconds, nowDate)?.countdown }}
      </span>
      <span class="shrink-0 text-gray-400 dark:text-gray-500">·</span>
      <span class="min-w-[112px] flex-1 whitespace-normal break-words font-semibold leading-snug text-gray-700 dark:text-gray-200">
        {{ formatResetValue(row.resetsAt, row.remainingSeconds, nowDate)?.absolute }}
      </span>
    </span>

    <span v-else class="text-gray-400 dark:text-gray-500">-</span>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { AccountUsageResetRow } from '@/types'
import {
  formatLocalAbsoluteTime,
  formatLocalTimestamp,
  formatResetCountdown,
  parseEffectiveResetAt,
} from '@/utils/usageResetTime'
import { resolveUsageWindowCapsuleClass } from '@/utils/accountUsageWindowDisplay'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  rows: AccountUsageResetRow[]
  nowDate: Date
}>()

const { t } = useI18n()

function formatResetValue(
  resetsAt: string | null,
  remainingSeconds: number | null | undefined,
  nowDate: Date,
): {
  countdown: string
  absolute: string
  tooltip: string
} | null {
  const effectiveResetAt = parseEffectiveResetAt(
    resetsAt,
    remainingSeconds ?? null,
    nowDate,
  )
  if (!effectiveResetAt) return null

  return {
    countdown: formatResetCountdown(
      effectiveResetAt,
      nowDate,
      t('admin.accounts.usageWindow.now'),
    ),
    absolute: formatLocalAbsoluteTime(effectiveResetAt, nowDate, {
      today: t('dates.today'),
      tomorrow: t('dates.tomorrow'),
    }),
    tooltip: formatLocalTimestamp(effectiveResetAt),
  }
}

function resetLabelClass(label: string): string {
  return resolveUsageWindowCapsuleClass(label)
}
</script>
