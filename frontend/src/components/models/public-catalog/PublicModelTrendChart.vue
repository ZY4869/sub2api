<template>
  <div class="relative w-full" :style="{ height: `${height}px` }">
    <svg viewBox="0 0 1000 100" preserveAspectRatio="none" class="h-full w-full overflow-visible">
      <defs>
        <linearGradient :id="gradientId" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" :stop-color="stroke" stop-opacity="0.2" />
          <stop offset="100%" :stop-color="stroke" stop-opacity="0" />
        </linearGradient>
      </defs>
      <line x1="0" y1="0" x2="1000" y2="0" stroke="#f1f5f9" stroke-width="1" stroke-dasharray="5,5" />
      <line x1="0" y1="50" x2="1000" y2="50" stroke="#f1f5f9" stroke-width="1" stroke-dasharray="5,5" />
      <line x1="0" y1="100" x2="1000" y2="100" stroke="#e2e8f0" stroke-width="1" />
      <polygon v-if="points" :points="areaPoints" :fill="`url(#${gradientId})`" />
      <polyline v-if="points" :points="points" fill="none" :stroke="stroke" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
    <div v-if="!points" class="absolute inset-0 flex items-center justify-center text-sm text-slate-400">
      {{ emptyLabel }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  values: number[]
  stroke: string
  emptyLabel: string
  height?: number
  percent?: boolean
}>(), {
  height: 160,
  percent: false,
})

const gradientId = `public-model-trend-${Math.random().toString(36).slice(2, 10)}`

const normalized = computed(() =>
  props.values.filter((value) => Number.isFinite(value)),
)

const points = computed(() => {
  if (normalized.value.length < 2) {
    return ''
  }
  const maxValue = Math.max(...normalized.value)
  const minValue = props.percent ? Math.min(0.9, Math.min(...normalized.value) - 0.05) : 0
  const max = maxValue === minValue ? maxValue + 1 : maxValue * 1.1
  const range = max - minValue || 1
  return normalized.value
    .map((value, index) => {
      const x = (index / (normalized.value.length - 1)) * 1000
      const y = 100 - ((value - minValue) / range) * 100
      return `${x},${Math.max(0, Math.min(100, y))}`
    })
    .join(' ')
})

const areaPoints = computed(() => (points.value ? `0,100 ${points.value} 1000,100` : ''))
</script>
