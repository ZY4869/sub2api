<template>
  <div
    data-test="account-segmented-countdown"
    class="flex max-w-full flex-wrap items-center gap-1"
    :aria-label="countdownLabel"
    :title="fullCountdownTitle"
  >
    <span
      v-if="prefix"
      :class="prefixClass"
      data-test="account-segmented-countdown-prefix"
    >
      {{ prefix }}
    </span>
    <template v-for="segment in visibleSegments" :key="segment.unit">
      <div
        :class="[blockClass, unitClass(segment.unit)]"
        :data-unit="segment.unit"
      >
        {{ segment.text }}
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRealtimeCountdownNow } from '@/composables/useRealtimeCountdownNow'
import {
  resolveAccountGlassToneStyles,
  type AccountGlassTone,
} from './accountVisualGlass'

const props = defineProps<{
  resetAt: string
  tone: AccountGlassTone
  prefix?: string
}>()

const { nowMs } = useRealtimeCountdownNow('accounts')

const diffMs = computed(() => {
  const resetAtMs = new Date(props.resetAt).getTime()
  if (!Number.isFinite(resetAtMs)) return 0
  return Math.max(0, resetAtMs - nowMs.value)
})

const totalSeconds = computed(() => Math.floor(diffMs.value / 1000))
const days = computed(() => Math.floor(totalSeconds.value / 86400))
const hours = computed(() =>
  Math.floor((totalSeconds.value % 86400) / 3600)
)
const minutes = computed(() =>
  Math.floor((totalSeconds.value % 3600) / 60)
)
const seconds = computed(() =>
  totalSeconds.value % 60
)

const toneStyles = computed(() => resolveAccountGlassToneStyles(props.tone))
const blockClass = computed(() => [
  'flex min-w-[32px] items-center justify-center rounded-full border px-1.5 py-[1.5px] font-mono text-[10px] font-black leading-none tracking-normal ring-1'
])
const prefixClass = computed(() => [
  'inline-flex min-w-[32px] items-center justify-center rounded-full border px-1.5 py-[1.5px] text-[10px] font-black leading-none tracking-normal',
  toneStyles.value.timerBlockClass
])

type CountdownUnit = 'D' | 'H' | 'M' | 'S'

const pad = (value: number): string => String(value).padStart(2, '0')

const visibleSegments = computed(() => {
  if (days.value > 0) {
    return [
      { unit: 'D' as CountdownUnit, text: `${pad(days.value)}D` },
      { unit: 'H' as CountdownUnit, text: `${pad(hours.value)}H` },
    ]
  }
  if (hours.value > 0) {
    return [
      { unit: 'H' as CountdownUnit, text: `${pad(hours.value)}H` },
      { unit: 'M' as CountdownUnit, text: `${pad(minutes.value)}M` },
    ]
  }
  return [
    { unit: 'M' as CountdownUnit, text: `${pad(minutes.value)}M` },
    { unit: 'S' as CountdownUnit, text: `${pad(seconds.value)}S` },
  ]
})

const countdownLabel = computed(() =>
  [props.prefix, ...visibleSegments.value.map((segment) => segment.text)]
    .filter(Boolean)
    .join(' ')
)

const fullCountdownTitle = computed(() => {
  const dayPart = days.value > 0 ? `${pad(days.value)}D ` : ''
  return `${props.prefix ? `${props.prefix} ` : ''}${dayPart}${pad(hours.value)}H ${pad(minutes.value)}M ${pad(seconds.value)}S`
})

function unitClass(unit: CountdownUnit): string {
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
</script>
