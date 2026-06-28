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
      class="flex min-w-0 flex-1 flex-nowrap items-center gap-x-1.5 overflow-visible text-gray-700 dark:text-gray-300"
      :title="formatResetValue(row.resetsAt, row.remainingSeconds, nowDate)?.tooltip || undefined"
    >
      <Icon
        v-if="formatResetValue(row.resetsAt, row.remainingSeconds, nowDate)?.showCountdown"
        name="clock"
        size="xs"
        class="shrink-0 text-gray-400 dark:text-gray-500"
      />
      <span
        v-if="formatResetValue(row.resetsAt, row.remainingSeconds, nowDate)?.showCountdown"
        class="shrink-0 whitespace-nowrap rounded-full bg-gray-100 px-1.5 py-0.5 font-medium leading-none text-gray-700 dark:bg-gray-700 dark:text-gray-200"
      >
        {{ formatResetValue(row.resetsAt, row.remainingSeconds, nowDate)?.countdown }}
      </span>
      <span
        v-if="formatResetValue(row.resetsAt, row.remainingSeconds, nowDate)?.showCountdown"
        class="shrink-0 text-gray-400 dark:text-gray-500"
      >·</span>
      <span class="min-w-0 shrink-0 whitespace-nowrap font-semibold leading-snug text-gray-700 dark:text-gray-200">
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
const ONE_DAY_MS = 24 * 60 * 60 * 1000

const pad = (value: number): string => String(value).padStart(2, '0')

const formatLocalTime = (date: Date): string =>
  `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`

const formatAbsoluteDateTime = (date: Date, nowDate: Date): string => {
  const timePart = formatLocalTime(date)
  const datePart = `${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
  if (date.getFullYear() === nowDate.getFullYear()) {
    return `${datePart} ${timePart}`
  }
  return `${date.getFullYear()}-${datePart} ${timePart}`
}

const formatCompactShortCountdown = (date: Date, nowDate: Date): string => {
  const totalMinutes = Math.max(
    0,
    Math.ceil((date.getTime() - nowDate.getTime()) / 60000),
  )
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  return `${pad(hours)}H:${pad(minutes)}M`
}

const formatCompactLongCountdown = (date: Date, nowDate: Date): string => {
  const totalHours = Math.max(
    0,
    Math.ceil((date.getTime() - nowDate.getTime()) / (60 * 60 * 1000)),
  )
  const days = Math.floor(totalHours / 24)
  const hours = totalHours % 24
  return `${pad(days)}D:${pad(hours)}H`
}

function formatResetValue(
  resetsAt: string | null,
  remainingSeconds: number | null | undefined,
  nowDate: Date,
): {
  showCountdown: boolean
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

  const diffMs = effectiveResetAt.getTime() - nowDate.getTime()
  const showCountdown = diffMs > 0
  const showShortCountdown = diffMs > 0 && diffMs < ONE_DAY_MS

  return {
    showCountdown,
    countdown: showShortCountdown
      ? formatCompactShortCountdown(effectiveResetAt, nowDate)
      : diffMs > 0
        ? formatCompactLongCountdown(effectiveResetAt, nowDate)
      : formatResetCountdown(
        effectiveResetAt,
        nowDate,
        t('admin.accounts.usageWindow.now'),
      ),
    absolute: showShortCountdown
      ? formatLocalTime(effectiveResetAt)
      : formatAbsoluteDateTime(effectiveResetAt, nowDate),
    tooltip: formatLocalTimestamp(effectiveResetAt),
  }
}

function resetLabelClass(label: string): string {
  return resolveUsageWindowCapsuleClass(label)
}
</script>
