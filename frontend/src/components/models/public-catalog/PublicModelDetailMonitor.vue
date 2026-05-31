<template>
  <div class="mx-auto flex max-w-[900px] flex-col gap-8 pb-10">
    <div class="grid gap-6 md:grid-cols-3">
      <div class="rounded-3xl border border-slate-200/60 bg-white p-6 shadow-[0_2px_12px_rgba(0,0,0,0.02)] dark:border-dark-700 dark:bg-dark-900">
        <div class="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">{{ labels.status }}</div>
        <div class="text-[32px] font-black leading-none tracking-tight" :class="statusClass">
          {{ statusLabel }}
        </div>
        <div class="mt-2 text-xs text-slate-400">{{ lastChecked }}</div>
        <div class="mt-3 inline-flex rounded-md border px-2 py-1 text-[11px] font-bold" :class="sourceClass">
          {{ sourceLabel }}
        </div>
      </div>
      <div class="rounded-3xl border border-slate-200/60 bg-white p-6 shadow-[0_2px_12px_rgba(0,0,0,0.02)] dark:border-dark-700 dark:bg-dark-900">
        <div class="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">{{ labels.latency }}</div>
        <div class="text-[32px] font-black leading-none tracking-tight text-slate-800 dark:text-white">
          {{ hasMetrics ? formatLatency(health?.latency_ms) : '-' }}
        </div>
      </div>
      <div class="relative overflow-hidden rounded-3xl border border-emerald-200/60 bg-gradient-to-br from-emerald-50 to-teal-50/30 p-6 shadow-[0_2px_12px_rgba(16,185,129,0.04)] dark:border-emerald-500/30 dark:from-emerald-500/10 dark:to-teal-500/10">
        <div class="relative z-10 mb-2 text-xs font-bold uppercase tracking-wider text-emerald-700 dark:text-emerald-200">{{ labels.todaySuccess }}</div>
        <div class="relative z-10 text-[32px] font-black leading-none tracking-tight text-emerald-600 dark:text-emerald-300">
          {{ hasMetrics ? formatRate(health?.success_rate_today) : '-' }}
        </div>
      </div>
    </div>

    <div
      v-if="!hasMetrics"
      class="rounded-3xl border border-dashed border-slate-300 bg-white/80 px-6 py-5 text-sm text-slate-500 dark:border-dark-700 dark:bg-dark-900/70 dark:text-slate-300"
    >
      {{ reasonLabel }}
    </div>

    <section v-if="hasMetrics" class="overflow-hidden rounded-3xl border border-slate-200/80 bg-white shadow-[0_4px_20px_rgba(0,0,0,0.03)] dark:border-dark-700 dark:bg-dark-900">
      <div class="flex items-center justify-between border-b border-slate-100/80 bg-slate-50/30 p-6 dark:border-dark-700 dark:bg-dark-800/40">
        <div>
          <h3 class="text-[15px] font-extrabold text-slate-800 dark:text-white">{{ labels.dailyMatrix }}</h3>
          <p class="mt-1 text-[11px] font-medium text-slate-400">{{ dailyMatrixCaption }}</p>
        </div>
      </div>
      <div class="divide-y divide-slate-50 dark:divide-dark-700">
        <div
          v-for="day in daily"
          :key="day.date"
          class="grid grid-cols-12 items-center gap-4 px-6 py-4 transition-colors hover:bg-slate-50/50 dark:hover:bg-dark-800/50"
        >
          <div class="col-span-4 text-[13px] font-bold text-slate-700 dark:text-slate-200">{{ day.date || '-' }}</div>
          <div class="col-span-3">
            <span class="rounded-md border px-2 py-1 text-[11px] font-bold" :class="badgeClass(day.status)">
              {{ labelForStatus(day.status) }}
            </span>
          </div>
          <div class="col-span-3 flex items-center gap-3">
            <PublicModelSuccessBars :rate="day.success_rate" :label="labels.successRate" />
            <span class="font-mono text-xs font-bold" :class="rateColor(day.success_rate)">
              {{ formatRate(day.success_rate) }}
            </span>
          </div>
          <div class="col-span-2 text-right font-mono text-xs font-medium text-slate-500">
            {{ formatLatency(day.latency_ms) }}
          </div>
        </div>
      </div>
    </section>

    <section v-if="hasMetrics" class="rounded-3xl border border-slate-200/80 bg-white p-8 shadow-[0_4px_20px_rgba(0,0,0,0.03)] dark:border-dark-700 dark:bg-dark-900">
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
