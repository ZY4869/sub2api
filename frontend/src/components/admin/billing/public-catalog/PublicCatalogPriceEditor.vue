<template>
  <div class="group/price rounded-xl border border-slate-100 bg-slate-50/80 p-3 dark:border-dark-700 dark:bg-dark-900/40">
    <div class="grid gap-3 sm:grid-cols-2">
      <div>
        <div class="mb-2 flex items-center justify-between gap-2">
          <div class="text-[11px] font-medium text-slate-400">
            {{ t('admin.billing.publicCatalog.price.official') }}
          </div>
        </div>
        <div v-if="officialConfiguredCount === 0" class="mb-2 text-xs text-slate-400 dark:text-slate-500">
          {{ t('admin.billing.publicCatalog.price.noOfficial') }}
        </div>
        <PublicCatalogPriceEntries
          v-if="officialEntries.length > 0"
          :entries="officialEntries"
          :editable="false"
          accent="official"
          :testid-prefix="`${testidPrefix}-official`"
          :price-label="priceLabel"
          :format-price="formatPrice"
          :format-unit="formatUnit"
          :format-input-value="formatInputValue"
          :unpriced-label="t('admin.billing.publicCatalog.price.supportedUnpriced')"
        />
      </div>

      <div>
        <div class="mb-2 flex items-center justify-between gap-2">
          <div class="flex min-w-0 items-center gap-1.5 text-[11px] font-medium text-slate-400">
            <span>{{ t('admin.billing.publicCatalog.price.sale') }}</span>
            <span
              v-if="markup"
              class="rounded-sm bg-emerald-100 px-1 py-0.5 text-[10px] font-semibold leading-none text-emerald-600 dark:bg-emerald-500/15 dark:text-emerald-200"
            >
              {{ t('admin.billing.publicCatalog.price.markup') }}
            </span>
          </div>
          <div
            v-if="editable"
            class="flex shrink-0 items-center overflow-hidden rounded border border-slate-200 bg-white opacity-100 shadow-sm transition sm:opacity-0 sm:group-hover/price:opacity-100 dark:border-dark-600 dark:bg-dark-800"
          >
            <input
              v-model="ratioInput"
              type="number"
              min="0"
              step="1"
              :aria-label="t('admin.billing.publicCatalog.price.localRatio')"
              class="w-11 border-0 bg-transparent px-1 py-0.5 text-center text-[10px] font-bold text-emerald-700 outline-none dark:text-emerald-200"
              :data-testid="`${testidPrefix}-ratio`"
              @keyup.enter="applyRatio"
            />
            <button
              type="button"
              class="border-l border-slate-100 px-1.5 py-0.5 text-[10px] font-semibold text-slate-500 transition hover:bg-emerald-50 hover:text-emerald-700 dark:border-dark-700 dark:hover:bg-emerald-500/10 dark:hover:text-emerald-200"
              :data-testid="`${testidPrefix}-apply-ratio`"
              :aria-label="t('admin.billing.publicCatalog.price.applyLocalRatio')"
              @click="applyRatio"
            >
              %
            </button>
          </div>
        </div>

        <div v-if="saleEntries.length === 0" class="text-xs text-slate-400 dark:text-slate-500">
          {{ t('admin.billing.publicCatalog.price.noSale') }}
        </div>
        <PublicCatalogPriceEntries
          v-else
          :entries="saleEntries"
          :editable="editable"
          accent="sale"
          :testid-prefix="testidPrefix"
          :price-label="priceLabel"
          :format-price="formatPrice"
          :format-unit="formatUnit"
          :format-input-value="formatInputValue"
          :unpriced-label="t('admin.billing.publicCatalog.price.supportedUnpriced')"
          @update-entry="updateEntry"
        />
      </div>
    </div>

    <div
      v-if="editable"
      class="mt-3 rounded-lg border border-emerald-100 bg-white p-3 dark:border-emerald-500/20 dark:bg-dark-800"
    >
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div class="text-xs font-semibold text-slate-700 dark:text-slate-200">
            {{ t('admin.billing.publicCatalog.imageFixed.title') }}
          </div>
          <div class="mt-0.5 text-[11px] text-slate-500 dark:text-slate-400">
            {{ t('admin.billing.publicCatalog.imageFixed.hint') }}
          </div>
        </div>
        <label class="flex items-center gap-2 text-xs font-medium text-slate-600 dark:text-slate-300">
          <input
            type="checkbox"
            class="h-4 w-4 rounded border-slate-300 text-emerald-600 focus:ring-emerald-500"
            :checked="fixedPricingValue.enabled"
            :data-testid="`${testidPrefix}-image-fixed-enabled`"
            @change="updateFixedEnabled(($event.target as HTMLInputElement).checked)"
          />
          {{ t('admin.billing.publicCatalog.imageFixed.enabled') }}
        </label>
      </div>

      <div v-if="fixedPricingValue.enabled" class="mt-3 space-y-3">
        <label class="flex items-center justify-between gap-3 text-xs font-medium text-slate-600 dark:text-slate-300">
          <span>{{ t('admin.billing.publicCatalog.imageFixed.alwaysFixed') }}</span>
          <input
            type="checkbox"
            class="h-4 w-4 rounded border-slate-300 text-emerald-600 focus:ring-emerald-500"
            :checked="fixedPricingValue.always_fixed"
            :data-testid="`${testidPrefix}-image-fixed-always`"
            @change="updateFixedAlways(($event.target as HTMLInputElement).checked)"
          />
        </label>
        <div class="grid gap-2 sm:grid-cols-3">
          <label
            v-for="resolution in imageResolutions"
            :key="resolution"
            class="space-y-1 text-xs font-medium text-slate-600 dark:text-slate-300"
          >
            <span>{{ t(`admin.billing.publicCatalog.imageFixed.${resolution}`) }}</span>
            <input
              :value="formatFixedPriceInput(fixedPricingValue.prices[resolution])"
              type="number"
              min="0"
              step="0.000001"
              class="input h-8 font-mono text-xs"
              :data-testid="`${testidPrefix}-image-fixed-${resolution}`"
              @input="updateFixedPrice(resolution, ($event.target as HTMLInputElement).value)"
            />
          </label>
        </div>
        <p
          v-if="fixedPricingError"
          class="text-[11px] font-medium text-rose-600 dark:text-rose-300"
        >
          {{ fixedPricingError }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  PublicModelCatalogPriceDisplay,
  PublicModelCatalogPriceEntry,
  PublicModelImageFixedPricing,
  PublicModelImageResolution,
} from '@/api/meta'
import { formatCatalogPrice } from '@/utils/publicModelCatalog'
import { clonePriceDisplay, scalePriceDisplay } from './publicCatalogPricing'
import PublicCatalogPriceEntries from './PublicCatalogPriceEntries.vue'

type PriceSection = 'primary' | 'secondary'

const props = withDefaults(defineProps<{
  official?: PublicModelCatalogPriceDisplay | null
  sale?: PublicModelCatalogPriceDisplay | null
  imageFixedPricing?: PublicModelImageFixedPricing | null
  currency?: string
  editable?: boolean
  testidPrefix?: string
}>(), {
  official: null,
  sale: null,
  imageFixedPricing: null,
  currency: 'USD',
  editable: false,
  testidPrefix: 'public-catalog-price',
})

const emit = defineEmits<{
  (e: 'update:sale', value: PublicModelCatalogPriceDisplay): void
  (e: 'update:imageFixedPricing', value: PublicModelImageFixedPricing): void
}>()

const { t, te } = useI18n()
const ratioInput = ref('120')
const imageResolutions: PublicModelImageResolution[] = ['1K', '2K', '4K']

const officialDisplay = computed(() => clonePriceDisplay(props.official || undefined))
const saleDisplay = computed(() => {
  const explicit = clonePriceDisplay(props.sale || undefined)
  return hasPriceEntries(explicit) ? explicit : clonePriceDisplay(officialDisplay.value)
})

const officialEntries = computed(() => flattenPriceDisplay(officialDisplay.value))
const saleEntries = computed(() => flattenPriceDisplay(saleDisplay.value))
const officialConfiguredCount = computed(() =>
  officialEntries.value.filter(({ entry }) => entry.configured !== false && !entry.supported_unpriced).length,
)
const markup = computed(() => saleEntries.value.some((saleEntry) => {
  const officialEntry = officialEntries.value.find((entry) => entry.entry.id === saleEntry.entry.id)
  return officialEntry ? saleEntry.entry.value > officialEntry.entry.value : false
}))
const fixedPricingValue = computed(() => normalizeFixedPricing(props.imageFixedPricing))
const fixedPricingError = computed(() => {
  if (!fixedPricingValue.value.enabled || !fixedPricingValue.value.always_fixed) return ''
  return imageResolutions.every((resolution) => Number(fixedPricingValue.value.prices[resolution] || 0) > 0)
    ? ''
    : t('admin.billing.publicCatalog.imageFixed.alwaysFixedError')
})

function updateEntry(section: PriceSection, index: number, raw: string) {
  const value = Number(raw)
  if (!Number.isFinite(value) || value < 0) {
    return
  }
  const next = clonePriceDisplay(saleDisplay.value)
  const entries = next[section] || []
  if (!entries[index]) {
    return
  }
  entries[index] = {
    ...entries[index],
    value,
    configured: true,
    supported_unpriced: false,
  }
  next[section] = entries
  emit('update:sale', next)
}

function applyRatio() {
  const ratio = Number(ratioInput.value) / 100
  if (!Number.isFinite(ratio) || ratio < 0) {
    return
  }
  emit('update:sale', scalePriceDisplay(officialDisplay.value, ratio))
}

function updateFixedEnabled(enabled: boolean) {
  emit('update:imageFixedPricing', {
    ...fixedPricingValue.value,
    enabled,
    always_fixed: enabled ? fixedPricingValue.value.always_fixed : false,
  })
}

function updateFixedAlways(alwaysFixed: boolean) {
  emit('update:imageFixedPricing', {
    ...fixedPricingValue.value,
    always_fixed: alwaysFixed,
  })
}

function updateFixedPrice(resolution: PublicModelImageResolution, raw: string) {
  const price = Number(raw)
  emit('update:imageFixedPricing', {
    ...fixedPricingValue.value,
    prices: {
      ...fixedPricingValue.value.prices,
      [resolution]: Number.isFinite(price) && price > 0 ? price : null,
    },
  })
}

function normalizeFixedPricing(value?: PublicModelImageFixedPricing | null): PublicModelImageFixedPricing {
  return {
    enabled: Boolean(value?.enabled),
    always_fixed: Boolean(value?.always_fixed),
    prices: {
      '1K': normalizeFixedPrice(value?.prices?.['1K']),
      '2K': normalizeFixedPrice(value?.prices?.['2K']),
      '4K': normalizeFixedPrice(value?.prices?.['4K']),
    },
  }
}

function normalizeFixedPrice(value: number | null | undefined): number | null {
  const next = Number(value)
  return Number.isFinite(next) && next > 0 ? next : null
}

function formatFixedPriceInput(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return ''
  return String(value)
}

function flattenPriceDisplay(display: PublicModelCatalogPriceDisplay) {
  return (['primary', 'secondary'] as const).flatMap((section) =>
    (display[section] || []).map((entry, index) => ({ section, index, entry })),
  )
}

function hasPriceEntries(display: PublicModelCatalogPriceDisplay): boolean {
  return (display.primary?.length || 0) > 0 || (display.secondary?.length || 0) > 0
}

function priceLabel(id: string): string {
  const key = `admin.billing.publicCatalog.price.labels.${id}`
  return te(key) ? t(key) : id.replace(/_/g, ' ')
}

function formatPrice(entry: PublicModelCatalogPriceEntry): string {
  return formatCatalogPrice(t, entry, props.currency, null)
}

function formatInputValue(value: number): string {
  const digits = Math.abs(value) >= 1 ? 4 : 8
  return value.toFixed(digits).replace(/\.?0+$/, '')
}

function formatUnit(entry: PublicModelCatalogPriceEntry): string {
  const unit = resolveDisplayUnit(entry)
  return t(`admin.billing.publicCatalog.price.units.${unit}`)
}

function resolveDisplayUnit(entry: PublicModelCatalogPriceEntry): 'perMillionTokens' | 'perImage' | 'perRequest' | 'perVideo' {
  switch (entry.display_unit) {
    case 'per_million_tokens':
      return 'perMillionTokens'
    case 'per_image':
      return 'perImage'
    case 'per_video':
      return 'perVideo'
    case 'per_request':
      return 'perRequest'
  }
  switch (entry.unit_kind) {
    case 'token':
      return 'perMillionTokens'
    case 'image':
      return 'perImage'
    case 'video':
      return 'perVideo'
    case 'request':
      return 'perRequest'
  }
  if (entry.unit === 'image') return 'perImage'
  if (entry.unit === 'video_request') return 'perVideo'
  if (String(entry.unit || '').includes('token')) return 'perMillionTokens'
  return 'perRequest'
}
</script>
