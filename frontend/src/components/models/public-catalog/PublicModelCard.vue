<template>
  <article
    class="group/card relative flex w-full flex-col rounded-[20px] border border-slate-200/80 bg-white p-6 shadow-[0_4px_20px_-4px_rgba(0,0,0,0.03)] transition-all duration-300 hover:-translate-y-1 hover:border-slate-300/80 hover:shadow-[0_12px_30px_-8px_rgba(0,0,0,0.08)] dark:border-dark-700 dark:bg-dark-900 dark:hover:border-dark-600"
    :data-testid="`public-model-card-${item.raw.model}`"
  >
    <PublicModelCardHeader
      :item="item"
      :status-dot-class="cardView.statusDotClass"
      :context-label="cardView.contextLabel"
      :context-class="contextClass"
      :copy-title="copyTitle"
      :detail-label="detailLabel"
      :detail-title="detailTitle"
      @copy="emit('copy', $event)"
      @open-detail="emit('openDetail', $event)"
    />

    <div class="mb-5 flex flex-wrap gap-2">
      <span class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-[11px] font-bold" :class="providerClass">
        <ModelPlatformIcon :platform="item.raw.provider_icon_key || item.raw.provider || ''" size="xs" />
        {{ providerLabel }}
      </span>
      <span class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-[11px] font-bold" :class="cardView.modalityClass">
        <Icon :name="cardView.modalityIcon" size="xs" />
        {{ cardView.modalityLabel }}
      </span>
      <span class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-[11px] font-bold" :class="cardView.statusBadgeClass">
        <Icon :name="healthIcon" size="xs" />
        {{ cardView.statusLabel }}
      </span>
      <span class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-[11px] font-bold" :class="cardView.publishedStatusClass">
        <Icon name="badge" size="xs" />
        {{ cardView.publishedStatusLabel }}
      </span>
      <span class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-[11px] font-bold" :class="cardView.lifecycleClass">
        <Icon name="badge" size="xs" />
        {{ cardView.lifecycleLabel }}
      </span>
      <span
        v-if="cardView.demoLabel"
        class="inline-flex items-center gap-1.5 rounded-md border border-rose-200 bg-rose-50 px-2.5 py-1 text-[11px] font-bold text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-200"
      >
        <Icon name="infoCircle" size="xs" />
        {{ cardView.demoLabel }}
      </span>
    </div>

    <div class="mb-4 rounded-xl border border-slate-100 bg-slate-50/70 px-3 py-2 text-xs text-slate-500 dark:border-dark-700 dark:bg-dark-800/60 dark:text-slate-300">
      <div class="flex flex-wrap items-center gap-2">
        <span class="font-bold text-slate-700 dark:text-slate-100">{{ cardView.healthSourceLabel }}</span>
        <span>{{ cardView.healthReasonLabel }}</span>
        <span v-if="cardView.contextSourceLabel">{{ cardView.contextSourceLabel }}</span>
        <span v-if="cardView.lifecycleSourceLabel">{{ cardView.lifecycleSourceLabel }}</span>
      </div>
      <div v-if="health?.last_checked_at" class="mt-1 font-mono text-[11px] text-slate-400">
        {{ health.last_checked_at }}
      </div>
    </div>

    <div class="mb-5 grid grid-cols-2 gap-3">
      <div class="rounded-xl border border-slate-100 bg-slate-50/80 p-3 dark:border-dark-700 dark:bg-dark-800/70">
        <div class="mb-2 flex items-center justify-between gap-2">
          <span class="text-[10px] font-black uppercase tracking-widest text-slate-400">{{ todayLabel }}</span>
          <PublicModelSuccessBars :rate="cardView.hasHealthMetrics ? health?.success_rate_today : undefined" :label="todayLabel" />
        </div>
        <div class="font-mono text-lg font-black" :class="rateColor(health?.success_rate_today)">
          {{ cardView.hasHealthMetrics ? formatRate(health?.success_rate_today) : '-' }}
        </div>
      </div>
      <div class="rounded-xl border border-slate-100 bg-slate-50/80 p-3 dark:border-dark-700 dark:bg-dark-800/70">
        <div class="mb-2 text-[10px] font-black uppercase tracking-widest text-slate-400">
          {{ latencyLabel }}
        </div>
        <div class="font-mono text-lg font-black text-slate-800 dark:text-white">
          {{ cardView.hasHealthMetrics ? formatLatency(health?.latency_ms) : '-' }}
        </div>
      </div>
    </div>

    <div class="mb-5 flex items-end justify-between gap-3 border-t border-slate-100 pt-4 dark:border-dark-700">
      <div>
        <div class="text-[10px] font-black uppercase tracking-widest text-slate-400">
          {{ weekLabel }}
        </div>
        <div class="mt-1 font-mono text-sm font-black" :class="rateColor(health?.success_rate_7d)">
          {{ cardView.hasHealthMetrics ? formatRate(health?.success_rate_7d) : '-' }}
        </div>
      </div>
      <PublicModelUptimeMatrix
        :days="cardView.hasHealthMetrics ? health?.daily : []"
        :label="matrixLabel"
        :labels="healthLabelsMap"
      />
    </div>

    <div class="mt-auto overflow-hidden rounded-[14px] border border-slate-200/60 bg-[#F8FAFC] p-3.5 transition-colors group-hover/card:bg-[#F4F7FB] dark:border-dark-700 dark:bg-dark-800/70 dark:group-hover/card:bg-dark-800">
      <div class="mb-2.5 flex items-center justify-between px-1">
        <span class="text-[10px] font-black uppercase tracking-widest text-slate-400">
          {{ pricingLabel }}
        </span>
        <span v-if="priceUnitSummary" class="hidden text-[10px] font-bold uppercase tracking-wider text-slate-400 sm:inline-block">
          {{ priceUnitSummary }}
        </span>
      </div>
      <div class="relative z-10 grid grid-cols-[repeat(auto-fit,minmax(8.25rem,1fr))] gap-2.5">
        <PublicModelPriceRow
          v-for="entry in priceEntries"
          :key="entry.id"
          :label="priceEntryLabel(entry.id)"
          :value="formatCatalogPrice(entry, item.raw.currency)"
          :theme="priceTheme(entry)"
          :testid="`public-model-primary-price-${item.raw.model}-${entry.id}`"
        />
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type {
  PublicModelCatalogItem,
  PublicModelCatalogPriceEntry,
  PublicModelCatalogStatusItem,
} from '@/api/meta'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import type { PublicModelCatalogDisplayItem } from '@/utils/publicModelCatalog'
import { priceDisplayUnitSummary } from '@/utils/publicModelCatalog'
import PublicModelCardHeader from './PublicModelCardHeader.vue'
import PublicModelPriceRow from './PublicModelPriceRow.vue'
import PublicModelSuccessBars from './PublicModelSuccessBars.vue'
import PublicModelUptimeMatrix from './PublicModelUptimeMatrix.vue'
import {
  buildPublicModelCardView,
  formatLatency,
  formatRate,
  healthLabels,
  normalizeHealthStatus,
  priceTheme,
  rateColor,
  type Translate,
} from './publicModelCatalogView'

const props = defineProps<{
  item: PublicModelCatalogDisplayItem
  health?: PublicModelCatalogStatusItem
  providerLabel: string
  detailLabel: string
  detailTitle: string
  copyTitle: string
  todayLabel: string
  weekLabel: string
  latencyLabel: string
  matrixLabel: string
  pricingLabel: string
  t: Translate
  priceEntryLabel: (fieldID: string) => string
  formatCatalogPrice: (entry: PublicModelCatalogPriceEntry, currency: string) => string
}>()

const emit = defineEmits<{
  copy: [item: PublicModelCatalogItem]
  openDetail: [item: PublicModelCatalogItem]
}>()

const cardView = computed(() => buildPublicModelCardView(props.item.raw, props.health, props.t))
const healthLabelsMap = computed(() => healthLabels(props.t))
const priceEntries = computed(() => props.item.primaryPrices.slice(0, 3))
const priceUnitSummary = computed(() => priceDisplayUnitSummary(props.t, priceEntries.value))

const healthIcon = computed(() => {
  switch (normalizeHealthStatus(props.health?.health_status || props.item.raw.health_status)) {
    case 'healthy':
      return 'checkCircle'
    case 'warning':
      return 'exclamationTriangle'
    case 'error':
      return 'xCircle'
    default:
      return 'infoCircle'
  }
})

const providerClass = computed(() => {
  const provider = String(props.item.raw.provider || props.item.raw.provider_icon_key || '').toLowerCase()
  if (provider.includes('openai')) {
    return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200'
  }
  if (provider.includes('anthropic')) {
    return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200'
  }
  if (provider.includes('gemini')) {
    return 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-500/30 dark:bg-blue-500/10 dark:text-blue-200'
  }
  return 'border-slate-200 bg-slate-50 text-slate-700 dark:border-dark-700 dark:bg-dark-800 dark:text-slate-200'
})

const contextClass = computed(() => {
  const tokens = props.item.raw.context_window?.tokens || props.item.raw.context_window_tokens || 0
  if (tokens >= 1_000_000 || tokens >= 512_000) {
    return 'border-amber-300 bg-gradient-to-br from-amber-50 to-orange-100 text-amber-700 dark:border-amber-500/40 dark:from-amber-500/10 dark:to-orange-500/10 dark:text-amber-200'
  }
  if (tokens > 128_000) {
    return 'border-rose-200 bg-gradient-to-br from-rose-50 to-red-100 text-rose-700 dark:border-rose-500/40 dark:from-rose-500/10 dark:to-red-500/10 dark:text-rose-200'
  }
  if (tokens > 64_000) {
    return 'border-blue-200 bg-gradient-to-br from-blue-50 to-blue-100 text-blue-700 dark:border-blue-500/40 dark:from-blue-500/10 dark:to-blue-500/10 dark:text-blue-200'
  }
  if (tokens > 0) {
    return 'border-emerald-200 bg-gradient-to-br from-emerald-50 to-teal-100 text-emerald-700 dark:border-emerald-500/40 dark:from-emerald-500/10 dark:to-teal-500/10 dark:text-emerald-200'
  }
  return 'border-slate-200 bg-gradient-to-br from-slate-50 to-slate-100 text-slate-600 dark:border-dark-700 dark:from-dark-800 dark:to-dark-800 dark:text-slate-300'
})

</script>
