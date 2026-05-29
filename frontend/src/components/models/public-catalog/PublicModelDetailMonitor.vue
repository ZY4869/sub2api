<template>
  <div class="mx-auto flex max-w-[900px] flex-col gap-8 pb-10">
    <div class="grid gap-6 md:grid-cols-3">
      <div class="rounded-3xl border border-slate-200/60 bg-white p-6 shadow-[0_2px_12px_rgba(0,0,0,0.02)] dark:border-dark-700 dark:bg-dark-900">
        <div class="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">{{ labels.status }}</div>
        <div class="text-[32px] font-black leading-none tracking-tight" :class="statusClass">
          {{ statusLabel }}
        </div>
        <div class="mt-2 text-xs text-slate-400">{{ lastChecked }}</div>
      </div>
      <div class="rounded-3xl border border-slate-200/60 bg-white p-6 shadow-[0_2px_12px_rgba(0,0,0,0.02)] dark:border-dark-700 dark:bg-dark-900">
        <div class="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">{{ labels.latency }}</div>
        <div class="text-[32px] font-black leading-none tracking-tight text-slate-800 dark:text-white">
          {{ formatLatency(health?.latency_ms) }}
        </div>
      </div>
      <div class="relative overflow-hidden rounded-3xl border border-emerald-200/60 bg-gradient-to-br from-emerald-50 to-teal-50/30 p-6 shadow-[0_2px_12px_rgba(16,185,129,0.04)] dark:border-emerald-500/30 dark:from-emerald-500/10 dark:to-teal-500/10">
        <div class="relative z-10 mb-2 text-xs font-bold uppercase tracking-wider text-emerald-700 dark:text-emerald-200">{{ labels.todaySuccess }}</div>
        <div class="relative z-10 text-[32px] font-black leading-none tracking-tight text-emerald-600 dark:text-emerald-300">
          {{ formatRate(health?.success_rate_today) }}
        </div>
      </div>
    </div>

    <section class="overflow-hidden rounded-3xl border border-slate-200/80 bg-white shadow-[0_4px_20px_rgba(0,0,0,0.03)] dark:border-dark-700 dark:bg-dark-900">
      <div class="flex items-center justify-between border-b border-slate-100/80 bg-slate-50/30 p-6 dark:border-dark-700 dark:bg-dark-800/40">
        <div>
          <h3 class="text-[15px] font-extrabold text-slate-800 dark:text-white">{{ labels.dailyMatrix }}</h3>
          <p class="mt-1 text-[11px] font-medium text-slate-400">{{ labels.dailyMatrixCaption }}</p>
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

    <section class="rounded-3xl border border-slate-200/80 bg-white p-8 shadow-[0_4px_20px_rgba(0,0,0,0.03)] dark:border-dark-700 dark:bg-dark-900">
      <div class="mb-6 flex items-center justify-between">
        <div>
          <h3 class="text-[15px] font-extrabold text-slate-800 dark:text-white">{{ labels.successTrend }}</h3>
          <p class="mt-1 text-[11px] font-medium text-slate-400">{{ labels.successTrendCaption }}</p>
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
  healthBadgeClass,
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

const status = computed(() => normalizeHealthStatus(props.health?.status))
const statusLabel = computed(() => healthStatusLabel(props.t, status.value))
const daily = computed<PublicModelCatalogDailyStatus[]>(() => props.health?.daily || [])
const successTrend = computed(() =>
  (props.health?.trend || [])
    .map((point) => point.success_rate)
    .filter((value): value is number => value != null && Number.isFinite(value)),
)
const lastChecked = computed(() => props.health?.last_checked_at || props.labels.pending)
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
