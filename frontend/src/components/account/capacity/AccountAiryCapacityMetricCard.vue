<template>
  <div
    :class="[
      rootClass,
      toneClass
    ]"
    :title="title"
    data-testid="airy-capacity-metric-card"
  >
    <div class="flex items-center gap-2">
      <div :class="iconClass">
        <slot name="icon" />
      </div>
      <div class="min-w-0">
        <div class="text-[10px] font-semibold uppercase tracking-[0.16em] opacity-65">
          {{ label }}
        </div>
        <div class="mt-1 flex items-end gap-1">
          <span class="truncate font-mono text-[12px] font-bold leading-none">{{ value }}</span>
          <span
            v-if="tag"
            class="rounded-full px-1.5 py-0.5 text-[9px] font-black leading-none tracking-[0.12em] opacity-75"
          >
            {{ tag }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  label: string
  value: string
  title?: string
  tone?: 'neutral' | 'safe' | 'warning' | 'danger'
  tag?: string
  whiteSurfaceEnabled?: boolean
}>(), {
  tone: 'neutral',
  title: '',
  tag: '',
  whiteSurfaceEnabled: false
})

const rootClass = computed(() =>
  props.whiteSurfaceEnabled
    ? 'rounded-[1rem] border border-slate-200/85 bg-white px-2.5 py-2 shadow-[0_8px_20px_rgba(15,23,42,0.05)] dark:border-slate-700/80 dark:bg-slate-900'
    : 'rounded-[1rem] border border-white/80 bg-white/76 px-2.5 py-2 shadow-[0_10px_24px_rgba(148,163,184,0.10)] backdrop-blur-sm dark:border-slate-700/80 dark:bg-slate-900/68'
)

const toneMap = {
  neutral: {
    root: 'text-slate-600 dark:text-slate-200',
    icon: 'border-slate-200/90 bg-slate-100/85 text-slate-500 dark:border-slate-700/80 dark:bg-slate-800/90 dark:text-slate-300'
  },
  safe: {
    root: 'text-emerald-700 dark:text-emerald-200',
    icon: 'border-emerald-200/90 bg-emerald-50 text-emerald-600 dark:border-emerald-400/25 dark:bg-emerald-500/12 dark:text-emerald-300'
  },
  warning: {
    root: 'text-amber-700 dark:text-amber-100',
    icon: 'border-amber-200/90 bg-amber-50 text-amber-600 dark:border-amber-400/25 dark:bg-amber-500/12 dark:text-amber-300'
  },
  danger: {
    root: 'text-rose-700 dark:text-rose-100',
    icon: 'border-rose-200/90 bg-rose-50 text-rose-600 dark:border-rose-400/25 dark:bg-rose-500/12 dark:text-rose-300'
  }
} as const

const toneClass = computed(() => toneMap[props.tone].root)
const iconClass = computed(() => [
  'flex h-8 w-8 shrink-0 items-center justify-center rounded-[0.9rem] border',
  toneMap[props.tone].icon
])
</script>
