<template>
  <div class="flex flex-col items-end">
    <div class="flex h-[26px] items-end gap-1">
      <div
        v-for="(day, index) in matrixDays"
        :key="`${day.date || 'pending'}-${index}`"
        class="group/col relative flex h-full cursor-default items-end pb-0.5"
      >
        <span
          class="w-2 rounded-[2px] transition-all duration-300"
          :class="statusBarClass(day.status)"
          :style="{ height: statusHeight(day.status) }"
        ></span>
        <span
          class="absolute bottom-full mb-1.5 hidden min-w-max flex-col rounded-lg border border-slate-100 bg-white p-2.5 text-xs shadow-[0_4px_20px_rgba(0,0,0,0.12)] group-hover/col:flex dark:border-dark-700 dark:bg-dark-900"
          :class="index >= 5 ? 'right-0' : 'left-1/2 -translate-x-1/2'"
        >
          <span class="mb-1 text-[11px] font-medium text-slate-500 dark:text-slate-400">
            {{ day.date || '-' }}
          </span>
          <span class="font-bold" :class="statusTextClass(day.status)">
            {{ statusLabel(day.status) }}
          </span>
        </span>
      </div>
    </div>
    <span class="mt-1.5 text-[9px] font-bold uppercase tracking-widest text-slate-400">
      {{ label }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { PublicModelCatalogDailyStatus, PublicModelHealthStatus } from '@/api/meta'

const props = defineProps<{
  days?: PublicModelCatalogDailyStatus[]
  label: string
  labels: Record<PublicModelHealthStatus, string>
}>()

const matrixDays = computed<PublicModelCatalogDailyStatus[]>(() => {
  const days = props.days || []
  if (days.length > 0) {
    return days.slice(-7)
  }
  return Array.from({ length: 7 }, () => ({ date: '', status: 'pending' as const }))
})

function statusLabel(status: PublicModelHealthStatus): string {
  return props.labels[status] || props.labels.pending
}

function statusBarClass(status: PublicModelHealthStatus): string {
  switch (status) {
    case 'healthy':
      return 'bg-emerald-400 hover:bg-emerald-500'
    case 'warning':
      return 'bg-amber-400 hover:bg-amber-500'
    case 'error':
      return 'bg-rose-400 hover:bg-rose-500'
    default:
      return 'bg-slate-300 hover:bg-slate-400 dark:bg-dark-600'
  }
}

function statusTextClass(status: PublicModelHealthStatus): string {
  switch (status) {
    case 'healthy':
      return 'text-emerald-600 dark:text-emerald-300'
    case 'warning':
      return 'text-amber-600 dark:text-amber-300'
    case 'error':
      return 'text-rose-600 dark:text-rose-300'
    default:
      return 'text-slate-500 dark:text-slate-300'
  }
}

function statusHeight(status: PublicModelHealthStatus): string {
  if (status === 'healthy') {
    return '100%'
  }
  if (status === 'warning') {
    return '70%'
  }
  if (status === 'error') {
    return '40%'
  }
  return '55%'
}
</script>
