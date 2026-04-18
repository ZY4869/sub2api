<template>
  <div class="mx-auto max-w-7xl space-y-6">
    <section class="overflow-hidden rounded-[2rem] border border-slate-200 bg-[radial-gradient(circle_at_top_left,_rgba(14,116,144,0.12),_transparent_35%),linear-gradient(135deg,_rgba(255,255,255,0.98),_rgba(240,249,255,0.92))] p-6 shadow-sm dark:border-dark-700 dark:bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.12),_transparent_35%),linear-gradient(135deg,_rgba(15,23,42,0.96),_rgba(17,24,39,0.92))] md:p-8">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="max-w-3xl">
          <p class="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">
            {{ t('ui.modelCatalog.eyebrow') }}
          </p>
          <h1 class="mt-3 text-3xl font-semibold tracking-tight text-slate-950 dark:text-white">
            {{ t('nav.modelsCatalog') }}
          </h1>
          <p class="mt-3 text-sm leading-7 text-slate-700 dark:text-slate-200">
            {{ t('ui.modelCatalog.description') }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <span class="rounded-full border border-slate-200 bg-white/80 px-4 py-2 text-sm text-slate-700 dark:border-dark-700 dark:bg-dark-900/80 dark:text-slate-200">
            {{ modelCountLabel }}
          </span>
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading"
            data-testid="public-models-refresh"
            @click="loadCatalog(true)"
          >
            {{ loading ? t('ui.modelCatalog.refreshing') : t('ui.modelCatalog.refresh') }}
          </button>
        </div>
      </div>
    </section>

    <div
      v-if="errorMessage"
      class="rounded-3xl border border-rose-200 bg-rose-50 px-6 py-4 text-sm text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-200"
    >
      {{ errorMessage }}
    </div>

    <div class="grid gap-6 xl:grid-cols-[280px_minmax(0,1fr)]">
      <aside class="space-y-4 rounded-3xl border border-slate-200 bg-white/90 p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900/80">
        <section class="space-y-3">
          <div class="text-sm font-semibold text-slate-900 dark:text-white">
            {{ t('ui.modelCatalog.filters.provider') }}
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              class="rounded-full px-3 py-1.5 text-sm transition"
              :class="selectedProvider === '' ? activeFilterClass : inactiveFilterClass"
              data-testid="models-filter-provider-all"
              @click="selectedProvider = ''"
            >
              {{ t('ui.modelCatalog.filters.all') }}
            </button>
            <button
              v-for="provider in providerOptions"
              :key="provider.id"
              type="button"
              class="rounded-full px-3 py-1.5 text-sm transition"
              :class="selectedProvider === provider.id ? activeFilterClass : inactiveFilterClass"
              :data-testid="`models-filter-provider-${provider.id}`"
              @click="selectedProvider = provider.id"
            >
              {{ provider.label }}
            </button>
          </div>
        </section>

        <section class="space-y-3">
          <div class="text-sm font-semibold text-slate-900 dark:text-white">
            {{ t('ui.modelCatalog.filters.protocol') }}
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              class="rounded-full px-3 py-1.5 text-sm transition"
              :class="selectedProtocol === '' ? activeFilterClass : inactiveFilterClass"
              data-testid="models-filter-protocol-all"
              @click="selectedProtocol = ''"
            >
              {{ t('ui.modelCatalog.filters.all') }}
            </button>
            <button
              v-for="protocol in protocolOptions"
              :key="protocol"
              type="button"
              class="rounded-full px-3 py-1.5 text-sm transition"
              :class="selectedProtocol === protocol ? activeFilterClass : inactiveFilterClass"
              :data-testid="`models-filter-protocol-${protocol}`"
              @click="selectedProtocol = protocol"
            >
              {{ protocol }}
            </button>
          </div>
        </section>

        <section class="space-y-3">
          <div class="text-sm font-semibold text-slate-900 dark:text-white">
            {{ t('ui.modelCatalog.filters.multiplier') }}
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              class="rounded-full px-3 py-1.5 text-sm transition"
              :class="selectedMultiplier === '' ? activeFilterClass : inactiveFilterClass"
              data-testid="models-filter-multiplier-all"
              @click="selectedMultiplier = ''"
            >
              {{ t('ui.modelCatalog.filters.all') }}
            </button>
            <button
              v-for="option in multiplierOptions"
              :key="option.id"
              type="button"
              class="rounded-full px-3 py-1.5 text-sm transition"
              :class="selectedMultiplier === option.id ? activeFilterClass : inactiveFilterClass"
              :data-testid="`models-filter-multiplier-${option.id}`"
              @click="selectedMultiplier = option.id"
            >
              {{ option.label }}
            </button>
          </div>
        </section>
      </aside>

      <section class="space-y-4">
        <div
          v-for="item in filteredItems"
          :key="item.model"
          class="rounded-3xl border border-slate-200 bg-white/90 p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md dark:border-dark-700 dark:bg-dark-900/80"
        >
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-3">
                <ModelIcon
                  :model="item.model"
                  :provider="item.provider"
                  :display-name="item.display_name"
                  size="22px"
                />
                <div class="min-w-0">
                  <div class="truncate text-lg font-semibold text-slate-950 dark:text-white">
                    {{ item.display_name || item.model }}
                  </div>
                  <div class="truncate text-sm text-slate-500 dark:text-slate-400">
                    {{ item.model }}
                  </div>
                </div>
              </div>

              <div class="mt-4 flex flex-wrap items-center gap-2 text-xs">
                <span class="inline-flex items-center gap-1 rounded-full bg-slate-100 px-2.5 py-1 text-slate-700 dark:bg-dark-800 dark:text-slate-200">
                  <ModelPlatformIcon :platform="item.provider_icon_key || item.provider || ''" size="sm" />
                  {{ item.provider || '-' }}
                </span>
                <span
                  v-for="protocol in item.request_protocols || []"
                  :key="protocol"
                  class="inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-sky-700 dark:bg-sky-500/15 dark:text-sky-200"
                >
                  {{ protocol }}
                </span>
                <span class="inline-flex rounded-full bg-emerald-100 px-2.5 py-1 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200">
                  {{ multiplierSummaryLabel(item.multiplier_summary) }}
                </span>
              </div>
            </div>

            <div class="rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-700 dark:bg-dark-800 dark:text-slate-200">
              <div class="text-xs uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">{{ item.currency }}</div>
              <div class="mt-2 space-y-1.5">
                <div
                  v-for="entry in item.price_display.primary"
                  :key="entry.id"
                  class="flex items-center justify-between gap-3"
                >
                  <span>{{ priceEntryLabel(entry.id) }}</span>
                  <span class="font-semibold">{{ formatCatalogPrice(entry, item.currency) }}</span>
                </div>
              </div>
            </div>
          </div>

          <div
            v-if="item.price_display.secondary?.length"
            class="mt-4 flex flex-wrap gap-2 text-xs text-slate-500 dark:text-slate-400"
          >
            <span
              v-for="entry in item.price_display.secondary"
              :key="entry.id"
              class="rounded-full border border-slate-200 px-2.5 py-1 dark:border-dark-700"
            >
              {{ priceEntryLabel(entry.id) }}: {{ formatCatalogPrice(entry, item.currency) }}
            </span>
          </div>
        </div>

        <div
          v-if="!loading && filteredItems.length === 0"
          class="rounded-3xl border border-dashed border-slate-300 bg-white/80 px-6 py-12 text-center text-sm text-slate-500 dark:border-dark-700 dark:bg-dark-900/70 dark:text-slate-400"
        >
          {{ t('ui.modelCatalog.empty') }}
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  PublicModelCatalogMultiplierSummary,
  PublicModelCatalogPriceEntry,
  PublicModelCatalogSnapshot,
} from '@/api/meta'
import { getModelCatalog, getUSDCNYExchangeRate } from '@/api/meta'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'

const { t } = useI18n()

const activeFilterClass = 'bg-primary-600 text-white'
const inactiveFilterClass = 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-dark-800 dark:text-slate-300 dark:hover:bg-dark-700'
const PROTOCOL_ORDER = ['openai', 'anthropic', 'gemini', 'grok', 'antigravity', 'vertex-batch']

const loading = ref(false)
const errorMessage = ref('')
const etag = ref<string | null>(null)
const catalog = ref<PublicModelCatalogSnapshot | null>(null)
const usdToCnyRate = ref<number | null>(null)

const selectedProvider = ref('')
const selectedProtocol = ref('')
const selectedMultiplier = ref('')

const modelCountLabel = computed(() => (
  t('ui.modelCatalog.modelCount', { count: catalog.value?.items.length || 0 })
))

const providerOptions = computed(() => {
  const seen = new Map<string, string>()
  for (const item of catalog.value?.items || []) {
    const key = String(item.provider || '').trim()
    if (key && !seen.has(key)) {
      seen.set(key, key)
    }
  }
  return Array.from(seen.entries())
    .map(([id, label]) => ({ id, label }))
    .sort((left, right) => left.label.localeCompare(right.label))
})

const protocolOptions = computed(() => {
  const seen = new Set<string>()
  for (const item of catalog.value?.items || []) {
    for (const protocol of item.request_protocols || []) {
      if (protocol) {
        seen.add(protocol)
      }
    }
  }
  return Array.from(seen).sort((left, right) => PROTOCOL_ORDER.indexOf(left) - PROTOCOL_ORDER.indexOf(right))
})

const multiplierOptions = computed(() => {
  const values = new Set<number>()
  let hasDisabled = false
  let hasMixed = false
  for (const item of catalog.value?.items || []) {
    if (item.multiplier_summary.kind === 'disabled') {
      hasDisabled = true
    }
    if (item.multiplier_summary.kind === 'mixed') {
      hasMixed = true
    }
    if (item.multiplier_summary.kind === 'uniform' && typeof item.multiplier_summary.value === 'number') {
      values.add(item.multiplier_summary.value)
    }
  }

  const options = Array.from(values)
    .sort((left, right) => left - right)
    .map((value) => ({
      id: `uniform:${value}`,
      label: `${formatMultiplier(value)}x`,
    }))

  if (hasDisabled) {
    options.unshift({ id: 'disabled', label: t('ui.modelCatalog.multiplier.disabled') })
  }
  if (hasMixed) {
    options.push({ id: 'mixed', label: t('ui.modelCatalog.multiplier.mixed') })
  }
  return options
})

const filteredItems = computed(() => (
  (catalog.value?.items || []).filter((item) => {
    if (selectedProvider.value && item.provider !== selectedProvider.value) {
      return false
    }
    if (selectedProtocol.value && !(item.request_protocols || []).includes(selectedProtocol.value)) {
      return false
    }
    if (!selectedMultiplier.value) {
      return true
    }
    if (selectedMultiplier.value === 'disabled') {
      return item.multiplier_summary.kind === 'disabled'
    }
    if (selectedMultiplier.value === 'mixed') {
      return item.multiplier_summary.kind === 'mixed'
    }
    if (!selectedMultiplier.value.startsWith('uniform:')) {
      return true
    }
    const target = Number(selectedMultiplier.value.slice('uniform:'.length))
    return item.multiplier_summary.kind === 'uniform' && item.multiplier_summary.value === target
  })
))

onMounted(() => {
  loadCatalog().catch(() => undefined)
})

async function loadCatalog(force = false) {
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await getModelCatalog(force ? null : etag.value)
    if (!response.notModified && response.data) {
      catalog.value = response.data
    }
    etag.value = response.etag
    if ((catalog.value?.items || []).some((item) => item.currency === 'CNY')) {
      const rate = await getUSDCNYExchangeRate()
      usdToCnyRate.value = rate.rate
    }
  } catch (error) {
    errorMessage.value = resolveErrorMessage(error, t('ui.modelCatalog.loadFailed'))
  } finally {
    loading.value = false
  }
}

function priceEntryLabel(fieldId: string): string {
  switch (fieldId) {
    case 'input_price':
      return t('ui.modelCatalog.priceFields.input')
    case 'output_price':
      return t('ui.modelCatalog.priceFields.output')
    case 'cache_price':
      return t('ui.modelCatalog.priceFields.cache')
    case 'input_price_above_threshold':
      return t('ui.modelCatalog.priceFields.inputTier')
    case 'output_price_above_threshold':
      return t('ui.modelCatalog.priceFields.outputTier')
    case 'batch_input_price':
      return t('ui.modelCatalog.priceFields.batchInput')
    case 'batch_output_price':
      return t('ui.modelCatalog.priceFields.batchOutput')
    case 'batch_cache_price':
      return t('ui.modelCatalog.priceFields.batchCache')
    case 'grounding_search':
      return t('ui.modelCatalog.priceFields.groundingSearch')
    case 'grounding_maps':
      return t('ui.modelCatalog.priceFields.groundingMaps')
    case 'file_search_embedding':
      return t('ui.modelCatalog.priceFields.embedding')
    case 'file_search_retrieval':
      return t('ui.modelCatalog.priceFields.retrieval')
    default:
      return fieldId
  }
}

function multiplierSummaryLabel(summary: PublicModelCatalogMultiplierSummary): string {
  if (summary.kind === 'disabled') {
    return t('ui.modelCatalog.multiplier.disabled')
  }
  if (summary.kind === 'mixed') {
    return t('ui.modelCatalog.multiplier.mixed')
  }
  return `${formatMultiplier(summary.value || 1)}x`
}

function formatCatalogPrice(entry: PublicModelCatalogPriceEntry, currency: string): string {
  const nextCurrency = currency === 'CNY' ? 'CNY' : 'USD'
  const symbol = nextCurrency === 'CNY' ? '¥' : '$'
  const unit = resolveDisplayUnit(entry.unit)
  const rawValue = convertCurrency(entry.value, nextCurrency)
  const displayValue = unit === 'per_million_tokens' ? rawValue * 1_000_000 : rawValue
  const suffix = unit === 'per_million_tokens'
    ? t('ui.modelCatalog.units.perMillionTokens')
    : unit === 'per_image'
      ? t('ui.modelCatalog.units.perImage')
      : t('ui.modelCatalog.units.perRequest')
  return `${symbol}${formatNumber(displayValue)} ${suffix}`
}

function resolveDisplayUnit(unit?: string): 'per_million_tokens' | 'per_request' | 'per_image' {
  switch (unit) {
    case 'image':
      return 'per_image'
    case 'video_request':
    case 'grounding_search_request':
    case 'grounding_maps_request':
      return 'per_request'
    default:
      if (String(unit || '').includes('token')) {
        return 'per_million_tokens'
      }
      return 'per_request'
  }
}

function convertCurrency(value: number, currency: string): number {
  if (currency !== 'CNY') {
    return value
  }
  if (typeof usdToCnyRate.value === 'number' && usdToCnyRate.value > 0) {
    return value * usdToCnyRate.value
  }
  return value
}

function formatMultiplier(value: number): string {
  return formatNumber(value)
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: value >= 1 ? 4 : 8,
  }).format(value)
}

function resolveErrorMessage(error: unknown, fallback: string): string {
  if (
    typeof error === 'object'
    && error
    && 'message' in error
    && typeof (error as { message?: unknown }).message === 'string'
  ) {
    return String((error as { message: string }).message)
  }
  return fallback
}
</script>
