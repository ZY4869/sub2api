<template>
  <div
    class="flex flex-1 cursor-default items-center justify-between rounded-lg border bg-white px-3 py-2.5 shadow-[0_1px_2px_rgba(0,0,0,0.01)] transition-all dark:bg-dark-900"
    :class="themeClass"
    :data-testid="testid"
  >
    <span class="text-[11px] font-semibold text-slate-600 dark:text-slate-300">
      {{ label }}
    </span>
    <span class="flex items-baseline gap-1 text-right font-mono text-[13px] tracking-tight">
      <span class="font-black" :class="valueClass">{{ value }}</span>
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  label: string
  value: string
  theme: 'blue' | 'emerald' | 'amber'
  testid?: string
}>()

const themeClass = computed(() => {
  switch (props.theme) {
    case 'emerald':
      return 'border-slate-100 hover:border-emerald-200 dark:border-dark-700 dark:hover:border-emerald-500/50'
    case 'amber':
      return 'border-slate-100 hover:border-amber-200 dark:border-dark-700 dark:hover:border-amber-500/50'
    default:
      return 'border-slate-100 hover:border-blue-200 dark:border-dark-700 dark:hover:border-blue-500/50'
  }
})

const valueClass = computed(() => {
  switch (props.theme) {
    case 'emerald':
      return 'text-emerald-700 dark:text-emerald-300'
    case 'amber':
      return 'text-amber-700 dark:text-amber-300'
    default:
      return 'text-blue-700 dark:text-blue-300'
  }
})
</script>
