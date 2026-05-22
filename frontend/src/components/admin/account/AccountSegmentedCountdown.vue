<template>
  <div data-test="account-segmented-countdown" class="flex items-center gap-[3px]">
    <template v-if="days > 0">
      <div :class="blockClass">{{ days }}</div>
      <span :class="accentClass">d</span>
    </template>
    <div :class="blockClass">{{ hours }}</div>
    <span :class="accentClass">:</span>
    <div :class="blockClass">{{ minutes }}</div>
    <span :class="accentClass">:</span>
    <div :class="blockClass">{{ seconds }}</div>
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
  String(Math.floor((totalSeconds.value % 86400) / 3600)).padStart(2, '0')
)
const minutes = computed(() =>
  String(Math.floor((totalSeconds.value % 3600) / 60)).padStart(2, '0')
)
const seconds = computed(() =>
  String(totalSeconds.value % 60).padStart(2, '0')
)

const toneStyles = computed(() => resolveAccountGlassToneStyles(props.tone))
const blockClass = computed(() => [
  'flex min-w-[24px] items-center justify-center rounded-md border px-1 py-[1.5px] font-mono text-[12px] font-black tracking-tight',
  toneStyles.value.timerBlockClass
])
const accentClass = computed(() => [
  'text-[11px] font-black leading-none',
  toneStyles.value.timerAccentClass
])
</script>
