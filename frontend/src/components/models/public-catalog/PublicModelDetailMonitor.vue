<template>
  <div class="mx-auto flex max-w-[960px] flex-col gap-5 pb-8">
    <div class="grid gap-3 md:grid-cols-3">
      <div class="min-w-0 rounded-lg border border-slate-200/60 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900 2xl:p-5">
        <div class="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">{{ labels.status }}</div>
        <div class="truncate text-2xl font-black leading-none 2xl:text-[32px]" :class="statusClass">
          {{ statusLabel }}
        </div>
        <div class="mt-2 text-xs text-slate-400">{{ lastChecked }}</div>
        <div class="mt-3 inline-flex rounded-md border px-2 py-1 text-[11px] font-bold" :class="sourceClass">
          {{ sourceLabel }}
        </div>
      </div>
      <div class="min-w-0 rounded-lg border border-slate-200/60 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900 2xl:p-5">
        <div class="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">{{ labels.latency }}</div>
        <div class="truncate text-2xl font-black leading-none text-slate-800 dark:text-white 2xl:text-[32px]">
          {{ hasMetrics ? formatLatency(health?.latency_ms) : '-' }}
        </div>
      </div>
      <div class="relative min-w-0 overflow-hidden rounded-lg border border-emerald-200/60 bg-emerald-50 p-4 shadow-sm dark:border-emerald-500/30 dark:bg-emerald-500/10 2xl:p-5">
        <div class="relative z-10 mb-2 text-xs font-bold uppercase tracking-wider text-emerald-700 dark:text-emerald-200">{{ labels.todaySuccess }}</div>
        <div class="relative z-10 truncate text-2xl font-black leading-none text-emerald-600 dark:text-emerald-300 2xl:text-[32px]">
          {{ hasMetrics ? formatRate(health?.success_rate_today) : '-' }}
        </div>
      </div>
    </div>

    <div
      v-if="!hasMetrics"
      class="rounded-lg border border-dashed border-slate-300 bg-white/80 px-5 py-5 text-sm text-slate-500 dark:border-dark-700 dark:bg-dark-900/70 dark:text-slate-300"
    >
      {{ reasonLabel }}
    </div>

    <section v-if="hasMetrics" class="overflow-hidden rounded-lg border border-slate-200/80 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <div class="flex items-center justify-between border-b border-slate-100/80 bg-slate-50/30 p-4 dark:border-dark-700 dark:bg-dark-800/40 2xl:p-5">
        <div>
          <h3 class="text-[15px] font-extrabold text-slate-800 dark:text-white">{{ labels.dailyMatrix }}</h3>
          <p class="mt-1 text-[11px] font-medium text-slate-400">{{ dailyMatrixCaption }}</p>
        </div>
      </div>
      <div class="divide-y divide-slate-50 dark:divide-dark-700">
        <div
          v-for="day in daily"
          :key="day.date"
          class="grid grid-cols-1 gap-2 px-4 py-4 transition-colors hover:bg-slate-50/50 dark:hover:bg-dark-800/50 sm:grid-cols-12 sm:items-center sm:gap-3 2xl:px-5"
        >
          <div class="min-w-0 text-[13px] font-bold text-slate-700 dark:text-slate-200 sm:col-span-4">{{ day.date || '-' }}</div>
          <div class="sm:col-span-3">
            <span class="rounded-md border px-2 py-1 text-[11px] font-bold" :class="badgeClass(day.status)">
              {{ labelForStatus(day.status) }}
            </span>
          </div>
          <div class="flex items-center gap-3 sm:col-span-3">
            <PublicModelSuccessBars :rate="day.success_rate" :label="labels.successRate" />
            <span class="font-mono text-xs font-bold" :class="rateColor(day.success_rate)">
              {{ formatRate(day.success_rate) }}
            </span>
          </div>
          <div class="font-mono text-xs font-medium text-slate-500 sm:col-span-2 sm:text-right">
            {{ formatLatency(day.latency_ms) }}
          </div>
        </div>
      </div>
    </section>

    <section v-if="hasMetrics" class="rounded-lg border border-slate-200/80 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900 2xl:p-6">
      <div class="mb-6 flex items-center justify-between">
        <div>
          <h3 class="text-[15px] font-extrabold text-slate-800 dark:text-white">{{ labels.successTrend }}</h3>
          <p class="mt-1 text-[11px] font-medium text-slate-400">{{ successTrendCaption }}</p>
        </div>
      </div>
      <PublicModelTrendChart
        :values="successTrend"
        stroke="#10b981"
        :empty-label="labels.pending"
        percent
      />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { PublicModelCatalogDailyStatus, PublicModelCatalogStatusItem, PublicModelHealthStatus } from '@/api/meta'
import PublicModelSuccessBars from './PublicModelSuccessBars.vue'
import PublicModelTrendChart from './PublicModelTrendChart.vue'
import {
  formatLatency,
  formatRate,
  hasHealthMetrics,
  healthBadgeClass,
  healthReasonLabel,
  healthSourceLabel,
  healthStatusLabel,
  normalizeHealthStatus,
  rateColor,
  type Translate,
} from './publicModelCatalogView'

const props = defineProps<{
  health?: PublicModelCatalogStatusItem
  labels: Record<string, string>
  t: Translate
}>()

const status = computed(() => normalizeHealthStatus(props.health?.health_status))
const statusLabel = computed(() => healthStatusLabel(props.t, status.value))
const hasMetrics = computed(() => hasHealthMetrics(props.health))
const daily = computed<PublicModelCatalogDailyStatus[]>(() => (hasMetrics.value ? props.health?.daily || [] : []))
const successTrend = computed(() =>
  (hasMetrics.value ? props.health?.trend || [] : [])
    .map((point) => point.success_rate)
    .filter((value): value is number => value != null && Number.isFinite(value)),
)
const lastChecked = computed(() => props.health?.last_checked_at || props.labels.pending)
const sourceLabel = computed(() => healthSourceLabel(props.t, props.health?.health_source))
const reasonLabel = computed(() => healthReasonLabel(props.t, props.health?.status_reason))
const dailyMatrixCaption = computed(() => {
  switch (props.health?.health_source) {
    case 'traffic':
      return props.labels.dailyMatrixCaptionTraffic || props.labels.dailyMatrixCaption
    case 'probe':
      return props.labels.dailyMatrixCaptionProbe || props.labels.dailyMatrixCaption
    default:
      return props.labels.dailyMatrixCaption
  }
})
const successTrendCaption = computed(() => {
  switch (props.health?.health_source) {
    case 'traffic':
      return props.labels.successTrendCaptionTraffic || props.labels.successTrendCaption
    case 'probe':
      return props.labels.successTrendCaptionProbe || props.labels.successTrendCaption
    default:
      return props.labels.successTrendCaption
  }
})
const sourceClass = computed(() => {
  switch (props.health?.health_source) {
    case 'traffic':
      return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200'
    case 'probe':
      return 'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200'
    default:
      return 'border-slate-200 bg-slate-50 text-slate-600 dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300'
  }
})
const statusClass = computed(() => {
  switch (status.value) {
    case 'healthy':
      return 'text-emerald-600 dark:text-emerald-300'
    case 'warning':
      return 'text-amber-600 dark:text-amber-300'
    case 'error':
      return 'text-rose-600 dark:text-rose-300'
    default:
      return 'text-slate-500 dark:text-slate-300'
  }
})

function labelForStatus(value: PublicModelHealthStatus): string {
  return healthStatusLabel(props.t, value)
}

function badgeClass(value: PublicModelHealthStatus): string {
  return healthBadgeClass(value)
}

</script>
