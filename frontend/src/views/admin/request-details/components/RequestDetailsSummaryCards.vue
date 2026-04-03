<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OpsRequestTraceSummary } from '@/api/admin/ops'
import { formatDurationMs, formatPercent } from '../helpers'
import { formatNumber } from '@/utils/format'

const props = defineProps<{
  summary: OpsRequestTraceSummary | null
  loading: boolean
}>()

const { t } = useI18n()

const cards = computed(() => {
  const totals = props.summary?.totals
  const requestCount = totals?.request_count ?? 0
  const successCount = totals?.success_count ?? 0
  const errorCount = totals?.error_count ?? 0
  const streamCount = totals?.stream_count ?? 0
  const toolCount = totals?.tool_count ?? 0
  const thinkingCount = totals?.thinking_count ?? 0
  const rawCount = totals?.raw_available_count ?? 0

  return [
    {
      key: 'requests',
      label: t('admin.requestDetails.summary.requests'),
      value: formatNumber(requestCount),
      hint: t('admin.requestDetails.summary.successErrorHint', {
        success: formatNumber(successCount),
        error: formatNumber(errorCount)
      })
    },
    {
      key: 'latency',
      label: t('admin.requestDetails.summary.latency'),
      value: formatDurationMs(totals?.p95_duration_ms),
      hint: `P50 ${formatDurationMs(totals?.p50_duration_ms)} / P99 ${formatDurationMs(totals?.p99_duration_ms)}`
    },
    {
      key: 'capability',
      label: t('admin.requestDetails.summary.capability'),
      value: formatPercent(streamCount + toolCount + thinkingCount, Math.max(requestCount * 3, 1)),
      hint: t('admin.requestDetails.summary.capabilityHint', {
        stream: formatNumber(streamCount),
        tools: formatNumber(toolCount),
        thinking: formatNumber(thinkingCount)
      })
    },
    {
      key: 'raw',
      label: t('admin.requestDetails.summary.rawCoverage'),
      value: formatPercent(rawCount, requestCount),
      hint: t('admin.requestDetails.summary.rawCoverageHint', {
        raw: formatNumber(rawCount),
        total: formatNumber(requestCount)
      })
    }
  ]
})
</script>

<template>
  <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
    <div
      v-for="card in cards"
      :key="card.key"
      class="rounded-3xl bg-white p-5 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700"
    >
      <div class="text-xs font-semibold uppercase tracking-[0.2em] text-gray-400 dark:text-gray-500">
        {{ card.label }}
      </div>
      <div class="mt-3 text-3xl font-semibold text-gray-900 dark:text-white">
        <span v-if="loading" class="inline-block h-8 w-24 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></span>
        <span v-else>{{ card.value }}</span>
      </div>
      <div class="mt-3 min-h-[2.5rem] text-sm text-gray-500 dark:text-gray-400">
        <span v-if="loading" class="inline-block h-4 w-full animate-pulse rounded bg-gray-100 dark:bg-dark-700"></span>
        <span v-else>{{ card.hint }}</span>
      </div>
    </div>
  </div>
</template>
