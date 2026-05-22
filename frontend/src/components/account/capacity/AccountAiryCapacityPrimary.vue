<template>
  <div
    :class="wrapperClass"
    :title="title"
    data-testid="airy-capacity-primary"
  >
    <div class="flex items-end gap-[3px]">
      <div
        v-for="cell in cells"
        :key="cell.index"
        :style="cell.style"
        :class="cell.className"
        data-testid="airy-capacity-bar"
      />
    </div>

    <div class="flex items-baseline font-mono text-[11px] font-bold tracking-tight leading-none">
      <span>{{ formattedCurrent }}</span>
      <span class="mx-[1px] text-[10px] opacity-40">/</span>
      <span class="opacity-80">{{ formattedTotal }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CapacityTone } from './presentation'
import { resolveDisplayedCapacityCells } from './presentation'

const props = defineProps<{
  used: number
  total: number
  formattedCurrent: string
  formattedTotal: string
  tone: CapacityTone
  whiteSurfaceEnabled?: boolean
  compact?: boolean
}>()

const themeByTone: Record<
  CapacityTone,
  { chrome: string; surface: string; bar: string; effect: string }
> = {
  idle: {
    chrome: 'border-slate-200/85 text-slate-600 dark:border-slate-700/80 dark:text-slate-200',
    surface: 'bg-white/78 dark:bg-slate-900/78',
    bar: 'bg-slate-200/90 dark:bg-slate-700/75',
    effect: ''
  },
  active: {
    chrome: 'border-emerald-200/90 text-emerald-700 dark:border-emerald-400/25 dark:text-emerald-200',
    surface: 'bg-emerald-50/96 dark:bg-emerald-500/10',
    bar: 'bg-emerald-500 dark:bg-emerald-400',
    effect: 'account-capacity-breathe'
  },
  warning: {
    chrome: 'border-amber-200/90 text-amber-700 dark:border-amber-400/25 dark:text-amber-100',
    surface: 'bg-amber-50/96 dark:bg-amber-500/10',
    bar: 'bg-amber-400 dark:bg-amber-300',
    effect: 'account-capacity-breathe'
  },
  high: {
    chrome: 'border-orange-200/90 text-orange-700 dark:border-orange-400/25 dark:text-orange-100',
    surface: 'bg-orange-50/96 dark:bg-orange-500/10',
    bar: 'bg-orange-500 dark:bg-orange-400',
    effect: 'account-capacity-breathe'
  },
  full: {
    chrome: 'border-rose-200/90 text-rose-700 dark:border-rose-400/25 dark:text-rose-100',
    surface: 'bg-rose-50/96 dark:bg-rose-500/10',
    bar: 'bg-rose-500 dark:bg-rose-400',
    effect: 'account-capacity-urgent'
  }
}

const resolvedTheme = computed(() => themeByTone[props.tone])
const displayedCells = computed(() => resolveDisplayedCapacityCells(props.used, props.total))

const wrapperClass = computed(() => {
  const surfaceClass = props.whiteSurfaceEnabled
    ? 'bg-white dark:bg-slate-900'
    : resolvedTheme.value.surface

  return [
    props.compact
      ? 'inline-flex w-[116px] items-center justify-center gap-2 rounded-[0.95rem] border px-2 py-[5px] transition-colors duration-300'
      : 'inline-flex items-center gap-2 rounded-[0.95rem] border px-2.5 py-[5px] shadow-[0_10px_24px_rgba(148,163,184,0.10)] transition-colors duration-300',
    resolvedTheme.value.chrome,
    surfaceClass
  ]
})

const title = computed(() => `当前占用并发/最大上限: ${props.used}/${props.total}`)

const cells = computed(() => {
  return Array.from({ length: displayedCells.value.displayedTotal }, (_, index) => {
    const filled = index < displayedCells.value.displayedUsed
    return {
      index,
      style: filled ? { animationDelay: `${index * 0.15}s` } : undefined,
      className: [
        'h-3.5 w-[4px] rounded-[1px] transition-all duration-500',
        filled
          ? `${resolvedTheme.value.bar} ${resolvedTheme.value.effect}`.trim()
          : 'bg-slate-200/80 dark:bg-slate-700/70'
      ]
    }
  })
})
</script>
