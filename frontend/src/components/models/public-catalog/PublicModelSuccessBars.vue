<template>
  <div class="flex h-3 items-end gap-[2px]" :aria-label="ariaLabel" role="img">
    <span
      v-for="bar in bars"
      :key="bar.index"
      class="w-[3px] rounded-[1px] transition-colors"
      :class="bar.className"
      :style="{ height: bar.height }"
    ></span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  rate?: number | null
  label: string
}>()

const percent = computed(() => {
  if (props.rate == null || !Number.isFinite(props.rate)) {
    return null
  }
  return Math.max(0, Math.min(100, props.rate * 100))
})

const filledCount = computed(() =>
  percent.value == null ? 0 : Math.round(percent.value / 10),
)

const bars = computed(() =>
  Array.from({ length: 10 }, (_, index) => {
    const filled = index < filledCount.value
    return {
      index,
      height: `${40 + index * 6}%`,
      className: filled ? filledClass(percent.value || 0) : 'bg-slate-200 dark:bg-dark-700',
    }
  }),
)

const ariaLabel = computed(() => {
  if (percent.value == null) {
    return `${props.label}: -`
  }
  return `${props.label}: ${percent.value.toFixed(1)}%`
})

function filledClass(value: number): string {
  if (value >= 98) {
    return 'bg-emerald-500'
  }
  if (value >= 90) {
    return 'bg-amber-400'
  }
  return 'bg-rose-500'
}
</script>
