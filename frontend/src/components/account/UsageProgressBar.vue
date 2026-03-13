<template>
  <div :class="detailedReset ? 'space-y-0.5' : ''">
    <div class="flex items-center gap-1">
      <span
        :class="['w-[32px] shrink-0 rounded px-1 text-center text-[10px] font-medium', labelClass]"
      >
        {{ label }}
      </span>

      <div class="h-1.5 w-8 shrink-0 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
        <div
          :class="['h-full transition-all duration-300', barClass]"
          :style="{ width: barWidth }"
        ></div>
      </div>

      <span :class="['w-[32px] shrink-0 text-right text-[10px] font-medium', textClass]">
        {{ displayPercent }}
      </span>

      <span v-if="!detailedReset && effectiveResetAt" class="shrink-0 text-[10px] text-gray-400">
        {{ compactResetText }}
      </span>
    </div>

    <div
      v-if="detailedReset"
      class="pl-[37px] text-[10px] text-gray-400"
      :title="resetTooltip || undefined"
    >
      {{ t('admin.accounts.usageWindow.remainingLabel') }} {{ resetCountdownText }}
      ·
      {{ t('admin.accounts.usageWindow.resetAtLabel') }} {{ resetAbsoluteText }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { WindowStats } from '@/types'
import {
  formatLocalAbsoluteTime,
  formatLocalTimestamp,
  formatResetCountdown,
  parseEffectiveResetAt
} from '@/utils/usageResetTime'

const props = defineProps<{
  label: string
  utilization: number
  resetsAt?: string | null
  remainingSeconds?: number | null
  color: 'indigo' | 'emerald' | 'purple' | 'amber'
  windowStats?: WindowStats | null
  detailedReset?: boolean
}>()

const { t } = useI18n()

const labelClass = computed(() => {
  const colors = {
    indigo: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300',
    emerald: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300',
    purple: 'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300',
    amber: 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
  }
  return colors[props.color]
})

const barClass = computed(() => {
  if (props.utilization >= 100) return 'bg-red-500'
  if (props.utilization >= 80) return 'bg-amber-500'
  return 'bg-green-500'
})

const textClass = computed(() => {
  if (props.utilization >= 100) return 'text-red-600 dark:text-red-400'
  if (props.utilization >= 80) return 'text-amber-600 dark:text-amber-400'
  return 'text-gray-600 dark:text-gray-400'
})

const barWidth = computed(() => `${Math.min(props.utilization, 100)}%`)

const displayPercent = computed(() => {
  const percent = Math.round(props.utilization)
  return percent > 999 ? '>999%' : `${percent}%`
})

const effectiveResetAt = computed(() =>
  parseEffectiveResetAt(props.resetsAt ?? null, props.remainingSeconds ?? null)
)

const compactResetText = computed(() => {
  if (!effectiveResetAt.value) return '-'
  return formatResetCountdown(effectiveResetAt.value, new Date(), t('admin.accounts.usageWindow.now'))
})

const resetCountdownText = computed(() => {
  if (!effectiveResetAt.value) return '-'
  return formatResetCountdown(effectiveResetAt.value, new Date(), t('admin.accounts.usageWindow.now'))
})

const resetAbsoluteText = computed(() => {
  if (!effectiveResetAt.value) return '-'
  return formatLocalAbsoluteTime(effectiveResetAt.value, new Date(), {
    today: t('dates.today'),
    tomorrow: t('dates.tomorrow')
  })
})

const resetTooltip = computed(() => {
  if (!effectiveResetAt.value) return ''
  return formatLocalTimestamp(effectiveResetAt.value)
})
</script>
