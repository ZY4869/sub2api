<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OpsRequestTraceSummaryPoint } from '@/api/admin/ops'
import { formatHistoryLabel } from '@/views/admin/ops/utils/opsFormatters'
import { formatDurationMs } from '../helpers'

const props = defineProps<{
  points: OpsRequestTraceSummaryPoint[]
  loading: boolean
  timeRange: string
}>()

const { t } = useI18n()
const chartWidth = 640
const chartHeight = 240
const padding = 24

const labels = computed(() => props.points.map((point) => formatHistoryLabel(point.bucket_start, props.timeRange)))
const maxRequests = computed(() => Math.max(...props.points.map((point) => point.request_count || 0), 1))
const maxLatency = computed(() => Math.max(...props.points.map((point) => point.p95_duration_ms || 0), 1))

function buildPolyline(values: number[], maxValue: number): string {
  if (!values.length) return ''
  const innerWidth = chartWidth - padding * 2
  const innerHeight = chartHeight - padding * 2
  return values.map((value, index) => {
    const x = padding + (innerWidth * index) / Math.max(values.length - 1, 1)
    const y = chartHeight - padding - (Math.max(value, 0) / Math.max(maxValue, 1)) * innerHeight
    return `${x},${y}`
  }).join(' ')
}

const requestPolyline = computed(() => buildPolyline(props.points.map((point) => point.request_count || 0), maxRequests.value))
const errorPolyline = computed(() => buildPolyline(props.points.map((point) => point.error_count || 0), maxRequests.value))
const latencyPolyline = computed(() => buildPolyline(props.points.map((point) => point.p95_duration_ms || 0), maxLatency.value))
</script>

<template>
  <div class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700">
    <div class="mb-4 flex items-center justify-between">
      <div>
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.requestDetails.charts.trendTitle') }}
        </h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.requestDetails.charts.trendDescription') }}
        </p>
      </div>
      <div class="flex flex-wrap gap-3 text-xs text-gray-500 dark:text-gray-400">
        <span class="inline-flex items-center gap-2"><span class="h-2 w-2 rounded-full bg-blue-600"></span>{{ t('admin.requestDetails.charts.requests') }}</span>
        <span class="inline-flex items-center gap-2"><span class="h-2 w-2 rounded-full bg-red-600"></span>{{ t('admin.requestDetails.charts.errors') }}</span>
        <span class="inline-flex items-center gap-2"><span class="h-2 w-2 rounded-full bg-amber-500"></span>{{ t('admin.requestDetails.charts.p95Latency') }}</span>
      </div>
    </div>

    <div v-if="loading" class="flex h-[320px] items-center justify-center text-sm text-gray-400">
      {{ t('common.loading') }}
    </div>
    <div v-else-if="!points.length" class="flex h-[320px] items-center justify-center text-sm text-gray-400">
      {{ t('common.noData') }}
    </div>
    <div v-else class="space-y-4">
      <svg :viewBox="`0 0 ${chartWidth} ${chartHeight}`" class="h-[280px] w-full">
        <line
          v-for="index in 5"
          :key="index"
          :x1="padding"
          :x2="chartWidth - padding"
          :y1="padding + ((chartHeight - padding * 2) * (index - 1)) / 4"
          :y2="padding + ((chartHeight - padding * 2) * (index - 1)) / 4"
          stroke="currentColor"
          class="text-gray-200 dark:text-dark-700"
          stroke-dasharray="4 4"
        />
        <polyline fill="none" stroke="#2563eb" stroke-width="3" :points="requestPolyline" />
        <polyline fill="none" stroke="#dc2626" stroke-width="3" :points="errorPolyline" />
        <polyline fill="none" stroke="#f59e0b" stroke-width="3" :points="latencyPolyline" />
      </svg>

      <div class="grid grid-cols-2 gap-3 text-xs text-gray-500 dark:text-gray-400 md:grid-cols-4">
        <div
          v-for="(point, index) in points.slice(-4)"
          :key="`${point.bucket_start}-${index}`"
          class="rounded-2xl bg-gray-50 px-3 py-2 dark:bg-dark-900"
        >
          <div class="font-medium text-gray-700 dark:text-gray-200">
            {{ labels[points.length - Math.min(4, points.length) + index] }}
          </div>
          <div class="mt-1">{{ t('admin.requestDetails.charts.requests') }} {{ point.request_count }}</div>
          <div>{{ t('admin.requestDetails.charts.errors') }} {{ point.error_count }}</div>
          <div>{{ formatDurationMs(point.p95_duration_ms) }}</div>
        </div>
      </div>
    </div>
  </div>
</template>
