<template>
  <div
    v-for="row in rows"
    :key="row.key"
    class="grid min-w-0 max-w-full grid-cols-[minmax(48px,auto)_minmax(0,1fr)] items-start gap-x-2 gap-y-1 text-[10px] tabular-nums"
  >
    <span
      :title="row.label"
      data-testid="account-usage-reset-window-label"
      :class="[
        'min-w-[48px] max-w-[92px] shrink-0 rounded-full border px-2 py-1 text-center font-black leading-none ring-1',
        resetLabelClass(row.label)
      ]"
    >
      {{ row.label }}
    </span>

    <span
      v-if="formatRowResetValue(row, nowDate)"
      class="flex min-w-0 flex-1 flex-nowrap items-center gap-x-1.5 overflow-visible text-gray-700 dark:text-gray-300"
      :title="formatRowResetValue(row, nowDate)?.tooltip || undefined"
    >
      <Icon
        v-if="formatRowResetValue(row, nowDate)?.segments.length"
        name="clock"
        size="xs"
        :class="['shrink-0', resetIconClass(row.label)]"
      />
      <span
        v-if="formatRowResetValue(row, nowDate)?.segments.length"
        data-testid="account-usage-reset-countdown"
        :aria-label="formatRowResetValue(row, nowDate)?.countdownText"
        class="flex shrink-0 items-center gap-1 whitespace-nowrap"
      >
        <span
          v-for="segment in formatRowResetValue(row, nowDate)?.segments"
          :key="segment.unit"
          data-testid="account-usage-reset-countdown-segment"
          :class="[
            'rounded-full px-1.5 py-0.5 font-bold leading-none ring-1',
            resetCountdownSegmentClass(segment.unit)
          ]"
        >
          {{ segment.text }}
        </span>
      </span>
      <span
        v-if="formatRowResetValue(row, nowDate)?.segments.length"
        class="shrink-0 text-gray-400 dark:text-gray-500"
      >·</span>
      <span class="min-w-0 shrink-0 whitespace-nowrap font-semibold leading-snug text-gray-700 dark:text-gray-200">
        {{ formatRowResetValue(row, nowDate)?.absolute || '-' }}
      </span>
    </span>

    <span v-else class="text-gray-400 dark:text-gray-500">-</span>
  </div>
</template>

<script setup lang="ts">
import type { AccountUsageResetRow } from '@/types'
import {
  formatLocalTimestamp,
  parseEffectiveResetAt,
} from '@/utils/usageResetTime'
import {
  resolveUsageResetIconClass,
  resolveUsageResetWindowLabelClass,
} from '@/utils/accountUsageWindowDisplay'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  rows: AccountUsageResetRow[]
  nowDate: Date
}>()

const ONE_DAY_MS = 24 * 60 * 60 * 1000
const fallbackRemainingAnchorMs = Date.now()

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

type CountdownUnit = 'D' | 'H' | 'M' | 'S'

interface CountdownSegment {
  text: string
  unit: CountdownUnit
}

const makeCountdownSegment = (value: number, unit: CountdownUnit): CountdownSegment => ({
  text: `${pad(value)}${unit}`,
  unit,
})

const formatCompactShortCountdown = (date: Date, nowDate: Date): CountdownSegment[] => {
  const totalMinutes = Math.max(
    0,
    Math.ceil((date.getTime() - nowDate.getTime()) / 60000),
  )
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  return [makeCountdownSegment(hours, 'H'), makeCountdownSegment(minutes, 'M')]
}

const formatCompactLongCountdown = (date: Date, nowDate: Date): CountdownSegment[] => {
  const totalHours = Math.max(
    0,
    Math.ceil((date.getTime() - nowDate.getTime()) / (60 * 60 * 1000)),
  )
  const days = Math.floor(totalHours / 24)
  const hours = totalHours % 24
  return [makeCountdownSegment(days, 'D'), makeCountdownSegment(hours, 'H')]
}

function formatResetValue(
  resetsAt: string | null,
  remainingSeconds: number | null | undefined,
  remainingAnchorMs: number | null | undefined,
  nowDate: Date,
): {
  segments: CountdownSegment[]
  countdownText: string
  absolute: string
  tooltip: string
} | null {
  const anchorDate =
    typeof remainingAnchorMs === 'number' && Number.isFinite(remainingAnchorMs)
      ? new Date(remainingAnchorMs)
      : new Date(fallbackRemainingAnchorMs)
  const effectiveResetAt = parseEffectiveResetAt(
    resetsAt,
    remainingSeconds ?? null,
    anchorDate,
  )
  if (!effectiveResetAt) return null

  const hasAbsoluteResetAt = typeof resetsAt === 'string' && resetsAt.trim() !== ''
  const diffMs = effectiveResetAt.getTime() - nowDate.getTime()
  const showShortCountdown = diffMs > 0 && diffMs < ONE_DAY_MS
  const segments = showShortCountdown
    ? formatCompactShortCountdown(effectiveResetAt, nowDate)
    : diffMs > 0
      ? formatCompactLongCountdown(effectiveResetAt, nowDate)
      : []

  return {
    segments,
    countdownText: segments.map((segment) => segment.text).join(' '),
    absolute: hasAbsoluteResetAt
      ? showShortCountdown
        ? formatLocalTime(effectiveResetAt)
        : formatAbsoluteDateTime(effectiveResetAt, nowDate)
      : '',
    tooltip: hasAbsoluteResetAt ? formatLocalTimestamp(effectiveResetAt) : '',
  }
}

function formatRowResetValue(row: AccountUsageResetRow, nowDate: Date) {
  return formatResetValue(
    row.resetsAt,
    row.remainingSeconds,
    row.remainingAnchorMs,
    nowDate,
  )
}

function resetLabelClass(label: string): string {
  return resolveUsageResetWindowLabelClass(label)
}

function resetCountdownSegmentClass(unit: CountdownUnit): string {
  switch (unit) {
    case 'D':
      return 'bg-emerald-100 text-emerald-800 ring-emerald-200 dark:bg-emerald-400/20 dark:text-emerald-50 dark:ring-emerald-300/20'
    case 'H':
      return 'bg-indigo-100 text-indigo-800 ring-indigo-200 dark:bg-indigo-400/20 dark:text-indigo-50 dark:ring-indigo-300/20'
    case 'M':
      return 'bg-sky-100 text-sky-800 ring-sky-200 dark:bg-sky-400/20 dark:text-sky-50 dark:ring-sky-300/20'
    case 'S':
      return 'bg-rose-100 text-rose-800 ring-rose-200 dark:bg-rose-400/20 dark:text-rose-50 dark:ring-rose-300/20'
  }
}

function resetIconClass(label: string): string {
  return resolveUsageResetIconClass(label)
}
</script>
