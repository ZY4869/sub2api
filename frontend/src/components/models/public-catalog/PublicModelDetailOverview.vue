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
              {{ hasMetrics ? formatLatency(health?.latency_ms) : '-' }}
            </div>
            <div class="mt-2 text-xs text-slate-400">{{ healthSourceText }}</div>
          </div>
          <div class="rounded-2xl border border-slate-200/60 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <div class="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">
              {{ labels.weekSuccess }}
            </div>
            <div class="text-3xl font-black tracking-tight" :class="rateColor(health?.success_rate_7d)">
              {{ hasMetrics ? formatRate(health?.success_rate_7d) : '-' }}
            </div>
            <div class="mt-2 text-xs text-slate-400">{{ healthReasonText }}</div>
          </div>
          <div class="relative overflow-hidden rounded-2xl border border-emerald-200/60 bg-gradient-to-br from-emerald-50 to-teal-50/50 p-5 shadow-sm dark:border-emerald-500/30 dark:from-emerald-500/10 dark:to-teal-500/10">
            <div class="relative z-10 mb-2 text-xs font-bold uppercase tracking-wider text-emerald-700 dark:text-emerald-200">
              {{ labels.todaySuccess }}
            </div>
            <div class="relative z-10 text-3xl font-black tracking-tight text-emerald-600 dark:text-emerald-300">
              {{ hasMetrics ? formatRate(health?.success_rate_today) : '-' }}
            </div>
          </div>
        </div>
      </section>

      <section>
        <h3 class="mb-4 flex items-center gap-2 text-sm font-extrabold uppercase tracking-widest text-slate-800 dark:text-white">
          {{ labels.publishStatus }}
        </h3>
        <div class="grid gap-4 sm:grid-cols-2">
          <div class="rounded-2xl border border-slate-200/60 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <div class="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">
              {{ labels.publishAvailability }}
            </div>
            <div class="text-lg font-black tracking-tight text-slate-800 dark:text-white">
              {{ publishStatusText }}
            </div>
          </div>
          <div class="rounded-2xl border border-slate-200/60 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
            <div class="mb-2 text-xs font-bold uppercase tracking-wider text-slate-500">
              {{ labels.realtimeSource }}
            </div>
            <div class="text-lg font-black tracking-tight text-slate-800 dark:text-white">
              {{ healthSourceText }}
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
              <div class="flex flex-wrap items-center gap-2">
                <span class="text-[15px] font-bold text-slate-800 dark:text-white">{{ labels.salePrice }}</span>
                <span
                  v-if="discountActive"
                  class="inline-flex items-center rounded-md border border-rose-200 bg-rose-50 px-2 py-0.5 text-xs font-bold text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-200"
                >
                  {{ discountLabel }}
                </span>
              </div>
              <span v-if="priceUnitSummary" class="rounded bg-slate-50 px-2 py-1 text-xs font-bold uppercase tracking-wider text-slate-400 dark:bg-dark-800">
                {{ item.currency || 'USD' }} {{ priceUnitSummary }}
              </span>
            </div>
            <div class="grid gap-4 sm:grid-cols-3">
              <PublicModelPriceRow
                v-for="entry in prices"
                :key="entry.id"
                :label="priceEntryLabel(entry.id)"
                :value="formatCatalogPrice(entry, item.currency)"
                :original-value="originalPriceValue(entry.id)"
                :theme="priceTheme(entry)"
                :testid="`detail-primary-price-${entry.id}`"
              />
            </div>
          </div>
          <div class="border-b border-slate-100 p-6 dark:border-dark-700">
            <div class="mb-4 text-sm font-bold text-slate-700 dark:text-slate-200">{{ labels.officialReferencePrice }}</div>
            <div v-if="officialPrices.length === 0" class="rounded-2xl border border-dashed border-slate-200 bg-slate-50/70 p-4 text-sm text-slate-500 dark:border-dark-700 dark:bg-dark-800/50 dark:text-slate-400">
              {{ labels.officialReferenceMissing }}
            </div>
            <div v-else class="grid gap-4 sm:grid-cols-3">
              <PublicModelPriceRow
                v-for="entry in officialPrices"
                :key="`official-${entry.id}`"
                :label="priceEntryLabel(entry.id)"
                :value="formatCatalogPrice(entry, item.currency)"
                theme="blue"
                :testid="`detail-official-price-${entry.id}`"
              />
            </div>
          </div>
          <div class="bg-slate-50/50 p-6 dark:bg-dark-800/50">
            <div class="mb-4 flex items-center justify-between">
              <div class="text-sm font-bold text-slate-700 dark:text-slate-200">{{ labels.multiplierRules }}</div>
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
              {{ formatContextWindow(item.context_window?.tokens || item.context_window_tokens) }}
            </div>
            <div class="mt-1 text-xs text-slate-400">{{ contextSourceText }}</div>
          </div>
          <div>
            <div class="text-[11px] font-bold uppercase tracking-wider text-slate-400">{{ labels.lifecycleSource }}</div>
            <div class="mt-1 text-sm font-bold text-slate-700 dark:text-slate-200">
              {{ lifecycleSourceText }}
            </div>
          </div>
        </div>
      </section>

      <section class="rounded-3xl border border-slate-200/60 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <h4 class="mb-5 text-[11px] font-black uppercase tracking-widest text-slate-400">
          {{ labels.capabilityMatrix }}
        </h4>
        <div class="space-y-3">
          <div
            v-for="entry in capabilityRows"
            :key="`${entry.capability}-${entry.protocol}-${entry.endpoint}`"
            class="rounded-xl border border-slate-100 bg-slate-50/70 p-3 dark:border-dark-700 dark:bg-dark-800/60"
          >
            <div class="flex items-center justify-between gap-3">
              <span class="text-sm font-bold text-slate-700 dark:text-slate-100">{{ formatTokenLabel(entry.capability) }}</span>
              <span class="rounded-md px-2 py-1 text-[11px] font-bold" :class="supportClass(entry.support)">
                {{ supportText(entry.support) }}
              </span>
            </div>
            <div class="mt-2 text-xs text-slate-400">
              {{ entry.protocol || '-' }} · {{ entry.endpoint || '-' }} · {{ sourceText(entry.source, entry.verified) }}
            </div>
          </div>
          <span v-if="capabilityRows.length === 0" class="text-sm text-slate-400">-</span>
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
  formatRate,
  hasHealthMetrics,
  healthReasonLabel,
  healthSourceLabel,
  lifecycleSourceLabel,
  publishedStatusLabel,
  priceTheme,
  rateColor,
  sourceLabel,
  supportLabel,
} from './publicModelCatalogView'
import { priceDisplayUnitSummary } from '@/utils/publicModelCatalog'

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
const hasMetrics = computed(() => hasHealthMetrics(props.health))
const officialPrices = computed(() => [
  ...(props.item.official_price_display?.primary || []),
  ...(props.item.official_price_display?.secondary || []),
].filter((entry) => entry.configured !== false && !entry.supported_unpriced))
const priceUnitSummary = computed(() => priceDisplayUnitSummary(labelsLookup, props.prices))
const discountActive = computed(() => !!props.item.discount_status?.active)
const discountLabel = computed(() => {
  const percent = props.item.discount_status?.reduction_percent || 0
  return percent > 0 ? props.labels.discountBadge.replace('{percent}', formatDiscountPercent(percent)) : ''
})
const originalPriceEntries = computed(() => {
  const display = props.item.original_sale_price_display || props.item.original_price_display
  const entries = [...(display?.primary || []), ...(display?.secondary || [])]
  return new Map(entries.map((entry) => [entry.id, entry]))
})
const healthSourceText = computed(() => healthSourceLabel(labelsLookup, props.health?.health_source))
const healthReasonText = computed(() => healthReasonLabel(labelsLookup, props.health?.status_reason))
const publishStatusText = computed(() => publishedStatusLabel((key) => labelsLookup(key), props.item))
const contextSourceText = computed(() => sourceLabel(labelsLookup, props.item.context_window?.source, props.item.context_window?.verified))
const lifecycleSourceText = computed(() => lifecycleSourceLabel(labelsLookup, props.item.lifecycle))
const capabilityRows = computed(() => (props.item.capability_matrix || []).slice(0, 8))

function labelsLookup(key: string): string {
  const map: Record<string, string> = {
    'ui.modelCatalog.healthSource.traffic': props.labels.healthSourceTraffic,
    'ui.modelCatalog.healthSource.probe': props.labels.healthSourceProbe,
    'ui.modelCatalog.healthSource.none': props.labels.healthSourceNone,
    'ui.modelCatalog.healthReason.trafficRecent': props.labels.healthReasonTrafficRecent,
    'ui.modelCatalog.healthReason.probeRecent': props.labels.healthReasonProbeRecent,
    'ui.modelCatalog.healthReason.monitorDisabled': props.labels.healthReasonMonitorDisabled,
    'ui.modelCatalog.healthReason.noHistory': props.labels.healthReasonNoHistory,
    'ui.modelCatalog.healthReason.staleHistory': props.labels.healthReasonStaleHistory,
    'ui.modelCatalog.healthReason.checking': props.labels.healthReasonChecking,
    'ui.modelCatalog.publishStatus.published': props.labels.publishPublished,
    'ui.modelCatalog.publishStatus.liveFallback': props.labels.publishLiveFallback,
    'ui.modelCatalog.publishStatus.unknown': props.labels.publishUnknown,
    'ui.modelCatalog.units.perMillionTokens': props.labels.perMillionTokens,
    'ui.modelCatalog.units.perImage': props.labels.perImage,
    'ui.modelCatalog.units.perRequest': props.labels.perRequest,
    'ui.modelCatalog.units.perVideo': props.labels.perVideo,
    'ui.modelCatalog.support.supported': props.labels.supported,
    'ui.modelCatalog.support.partial': props.labels.partial,
    'ui.modelCatalog.support.unsupported': props.labels.unsupported,
    'ui.modelCatalog.support.unknown': props.labels.unknown,
    'ui.modelCatalog.source.verified': props.labels.verified,
    'ui.modelCatalog.source.probe': props.labels.verified,
    'ui.modelCatalog.source.declared': props.labels.declared,
    'ui.modelCatalog.source.pricing': props.labels.pricingSource,
    'ui.modelCatalog.source.snapshot': props.labels.snapshotSource,
    'ui.modelCatalog.source.inferred': props.labels.inferred,
    'ui.modelCatalog.source.unknown': props.labels.unknown,
    'ui.modelCatalog.discountBadge': props.labels.discountBadge,
  }
  return map[key] || key
}

function supportText(value?: string): string {
  return supportLabel(labelsLookup, value)
}

function sourceText(source?: string, verified?: boolean): string {
  return sourceLabel(labelsLookup, source, verified)
}

function supportClass(value?: string): string {
  switch (value) {
    case 'supported':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200'
    case 'partial':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-200'
    case 'unsupported':
      return 'bg-rose-50 text-rose-700 dark:bg-rose-500/10 dark:text-rose-200'
    default:
      return 'bg-slate-100 text-slate-500 dark:bg-dark-700 dark:text-slate-300'
  }
}

function formatTokenLabel(value: string): string {
  return String(value || '')
    .trim()
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (char) => char.toUpperCase())
}

function originalPriceValue(entryID: string): string | undefined {
  if (!discountActive.value) return undefined
  const entry = originalPriceEntries.value.get(entryID)
  if (!entry) return undefined
  return props.formatCatalogPrice(entry, props.item.currency)
}

function formatDiscountPercent(value: number): string {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(value)
}
</script>
