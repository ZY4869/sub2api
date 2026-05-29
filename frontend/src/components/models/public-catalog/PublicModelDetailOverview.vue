<template>
  <div class="grid grid-cols-1 gap-8 pb-10 lg:grid-cols-12">
    <div class="flex flex-col gap-8 lg:col-span-8">
      <section>
        <h3 class="mb-4 flex items-center gap-2 text-sm font-extrabold uppercase tracking-widest text-slate-800 dark:text-white">
          {{ labels.telemetry }}
        </h3>
        <div class="grid gap-4 md:grid-cols-3">
          <div class="rounded-2xl border border-slate-200/60 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <div class="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">
              {{ labels.latency }}
            </div>
            <div class="text-3xl font-black tracking-tight text-slate-800 dark:text-white">
              {{ formatLatency(health?.latency_ms) }}
            </div>
          </div>
          <div class="rounded-2xl border border-slate-200/60 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <div class="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">
              {{ labels.weekSuccess }}
            </div>
            <div class="text-3xl font-black tracking-tight" :class="rateColor(health?.success_rate_7d)">
              {{ formatRate(health?.success_rate_7d) }}
            </div>
          </div>
          <div class="relative overflow-hidden rounded-2xl border border-emerald-200/60 bg-gradient-to-br from-emerald-50 to-teal-50/50 p-5 shadow-sm dark:border-emerald-500/30 dark:from-emerald-500/10 dark:to-teal-500/10">
            <div class="relative z-10 mb-2 text-xs font-bold uppercase tracking-wider text-emerald-700 dark:text-emerald-200">
              {{ labels.todaySuccess }}
            </div>
            <div class="relative z-10 text-3xl font-black tracking-tight text-emerald-600 dark:text-emerald-300">
              {{ formatRate(health?.success_rate_today) }}
            </div>
          </div>
        </div>
      </section>

      <section>
        <h3 class="mb-4 flex items-center gap-2 text-sm font-extrabold uppercase tracking-widest text-slate-800 dark:text-white">
          {{ labels.pricing }}
        </h3>
        <div class="overflow-hidden rounded-3xl border border-slate-200/60 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="border-b border-slate-100 p-6 pb-8 dark:border-dark-700">
            <div class="mb-6 flex items-center justify-between">
              <span class="text-[15px] font-bold text-slate-800 dark:text-white">{{ labels.basePrice }}</span>
              <span class="rounded bg-slate-50 px-2 py-1 text-xs font-bold uppercase tracking-wider text-slate-400 dark:bg-dark-800">
                {{ item.currency || 'USD' }} / 1M Tokens
              </span>
            </div>
            <div class="grid gap-4 sm:grid-cols-3">
              <PublicModelPriceRow
                v-for="entry in prices"
                :key="entry.id"
                :label="priceEntryLabel(entry.id)"
                :value="formatCatalogPrice(entry, item.currency)"
                :theme="priceTheme(entry)"
                :testid="`detail-primary-price-${entry.id}`"
              />
            </div>
          </div>
          <div class="bg-slate-50/50 p-6 dark:bg-dark-800/50">
            <div class="mb-4 flex items-center justify-between">
              <div class="text-sm font-bold text-slate-700 dark:text-slate-200">{{ labels.routePolicy }}</div>
              <div class="rounded-md border border-indigo-100 bg-indigo-50 px-2.5 py-1 text-[11px] font-bold uppercase text-indigo-600 dark:border-indigo-500/30 dark:bg-indigo-500/10 dark:text-indigo-200">
                {{ multiplierLabel }}
              </div>
            </div>
            <div class="rounded-2xl border border-slate-200/80 bg-white p-4 dark:border-dark-700 dark:bg-dark-900">
              <div class="flex flex-wrap items-center gap-2 text-[13px] font-medium text-slate-600 dark:text-slate-300">
                <span class="h-2.5 w-2.5 rounded-full bg-blue-500 shadow-sm"></span>
                <span>{{ protocolSummary }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>

    <div class="flex flex-col gap-6 lg:col-span-4">
      <section class="rounded-3xl border border-slate-200/60 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <h4 class="mb-5 text-[11px] font-black uppercase tracking-widest text-slate-400">
          {{ labels.modalities }}
        </h4>
        <div class="flex flex-wrap gap-2">
          <span
            v-for="modality in normalizedModalities"
            :key="modality"
            class="rounded-lg border border-sky-200 bg-sky-50 px-3 py-1.5 text-xs font-bold text-sky-700 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200"
          >
            {{ modality }}
          </span>
          <span v-if="normalizedModalities.length === 0" class="text-sm text-slate-400">-</span>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200/60 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <h4 class="mb-5 text-[11px] font-black uppercase tracking-widest text-slate-400">
          {{ labels.capabilities }}
        </h4>
        <div class="flex flex-wrap gap-2">
          <span
            v-for="capability in normalizedCapabilities"
            :key="capability"
            class="rounded-md border border-orange-200 bg-orange-50 px-2.5 py-1.5 text-[11px] font-bold text-orange-600 dark:border-orange-500/30 dark:bg-orange-500/10 dark:text-orange-200"
          >
            {{ capability }}
          </span>
          <span v-if="normalizedCapabilities.length === 0" class="text-sm text-slate-400">-</span>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200/60 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <h4 class="mb-6 text-[11px] font-black uppercase tracking-widest text-slate-400">
          {{ labels.specs }}
        </h4>
        <div class="space-y-5">
          <div>
            <div class="text-[11px] font-bold uppercase tracking-wider text-slate-400">{{ labels.context }}</div>
            <div class="mt-1 text-lg font-black tracking-tight text-indigo-600 dark:text-indigo-300">
              {{ formatContextWindow(item.context_window_tokens) }}
            </div>
          </div>
          <div>
            <div class="text-[11px] font-bold uppercase tracking-wider text-slate-400">{{ labels.rateLimits }}</div>
            <div class="mt-2 flex flex-wrap gap-3 font-mono text-sm font-black text-slate-800 dark:text-white">
              <span>RPM {{ formatLimit(health?.rate_limit?.rpm) }}</span>
              <span>TPM {{ formatLimit(health?.rate_limit?.tpm) }}</span>
              <span>RPD {{ formatLimit(health?.rate_limit?.rpd) }}</span>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type {
  PublicModelCatalogItem,
  PublicModelCatalogPriceEntry,
  PublicModelCatalogStatusItem,
} from '@/api/meta'
import PublicModelPriceRow from './PublicModelPriceRow.vue'
import {
  formatContextWindow,
  formatLatency,
  formatLimit,
  formatRate,
  priceTheme,
  rateColor,
} from './publicModelCatalogView'

const props = defineProps<{
  item: PublicModelCatalogItem
  health?: PublicModelCatalogStatusItem
  prices: PublicModelCatalogPriceEntry[]
  multiplierLabel: string
  protocolSummary: string
  labels: Record<string, string>
  priceEntryLabel: (fieldID: string) => string
  formatCatalogPrice: (entry: PublicModelCatalogPriceEntry, currency: string) => string
}>()

const normalizedModalities = computed(() => (props.item.modalities || []).map(formatTokenLabel))
const normalizedCapabilities = computed(() => (props.item.capabilities || []).map(formatTokenLabel))

function formatTokenLabel(value: string): string {
  return String(value || '')
    .trim()
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (char) => char.toUpperCase())
}
</script>
