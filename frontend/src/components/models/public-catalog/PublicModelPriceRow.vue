<template>
  <div
    class="flex min-w-0 cursor-default flex-col rounded-xl border bg-white px-3.5 py-3 shadow-[0_1px_2px_rgba(0,0,0,0.01)] transition-all dark:bg-dark-900"
    :class="themeClass"
    :data-testid="testid"
  >
    <span
      class="max-w-full truncate whitespace-nowrap text-[11px] font-semibold leading-none text-slate-600 dark:text-slate-300"
      :title="label"
    >
      {{ label }}
    </span>
    <div class="mt-2 flex min-w-0 flex-wrap items-baseline gap-x-1.5 gap-y-1 font-mono leading-none">
      <span class="whitespace-nowrap text-[15px] font-black" :class="valueClass">
        {{ priceParts.amount }}
      </span>
      <span
        v-if="priceParts.unit"
        class="whitespace-nowrap text-[10px] font-bold uppercase text-slate-500 dark:text-slate-400"
      >
        {{ priceParts.unit }}
      </span>
    </div>
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

const priceParts = computed(() => {
  const text = props.value.trim()
  const match = text.match(/^([$¥]\S+)\s+(.+)$/)
  return {
    amount: match?.[1] || text,
    unit: match?.[2] || '',
  }
})

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
