<template>
  <BaseDialog
    :show="show"
    title="模型定价编辑"
    width="account-wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <div class="space-y-4">
      <BillingPricingCurrencyToolbar
        :currency="currentCurrency"
        :usd-to-cny-rate="usdToCnyRate"
        :cny-enabled="cnyEnabled"
        :save-blocked="currencySaveBlocked"
        @update:currency="updateCurrentCurrency"
      />

      <div class="grid gap-4 xl:grid-cols-[280px_minmax(0,1fr)_minmax(0,1fr)]">
        <aside class="rounded-3xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-900/40">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">工作集模型</h3>
          <input
            v-model.trim="search"
            type="text"
            class="input mt-3 w-full"
            placeholder="搜索当前工作集模型"
          />

          <div class="mt-4 space-y-2">
            <button
              v-for="detail in filteredDetails"
              :key="detail.model"
              type="button"
              class="w-full rounded-2xl border px-3 py-3 text-left transition"
              :class="detail.model === currentModel ? 'border-primary-400 bg-primary-50 dark:border-primary-500/40 dark:bg-primary-500/10' : 'border-gray-200 bg-white hover:border-primary-300 dark:border-dark-700 dark:bg-dark-800'"
              @click="emit('update:activeModel', detail.model)"
            >
              <div class="font-medium text-gray-900 dark:text-white">{{ detail.display_name || detail.model }}</div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ detail.model }}</div>
              <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                {{ detail.provider || '-' }} / {{ detail.mode || '-' }} / {{ detail.currency }}
              </div>
              <div v-if="detail.pricing_status !== 'ok'" class="mt-2">
                <span
                  class="inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium"
                  :class="pricingStatusClass(detail.pricing_status)"
                >
                  {{ pricingStatusLabel(detail.pricing_status) }}
                </span>
              </div>
              <div class="mt-3 flex flex-wrap gap-1 text-[11px]">
                <span
                  class="inline-flex rounded-full px-2 py-1"
                  :class="billingLayerHasValues(detail.official_form) ? 'bg-sky-100 text-sky-700 dark:bg-sky-500/15 dark:text-sky-200' : 'bg-gray-200 text-gray-600 dark:bg-dark-700 dark:text-gray-300'"
                >
                  官方 {{ countConfiguredBillingFields(detail.official_form) }} 项
                </span>
                <span
                  class="inline-flex rounded-full px-2 py-1"
                  :class="billingLayerHasValues(detail.sale_form) ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200' : 'bg-gray-200 text-gray-600 dark:bg-dark-700 dark:text-gray-300'"
                >
                  售价 {{ countConfiguredBillingFields(detail.sale_form) }} 项
                </span>
              </div>
            </button>
          </div>
        </aside>

        <div class="space-y-4">
          <div
            v-if="currentDetail?.pricing_status !== 'ok'"
            class="rounded-2xl border px-4 py-3"
            :class="currentDetail?.pricing_status === 'conflict' || currentDetail?.pricing_status === 'missing'
              ? 'border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-200'
              : 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200'"
          >
            <div class="flex flex-wrap items-center gap-2 text-sm font-medium">
              <span
                class="inline-flex rounded-full px-2 py-0.5 text-[11px]"
                :class="pricingStatusClass(currentDetail?.pricing_status)"
              >
                {{ pricingStatusLabel(currentDetail?.pricing_status) }}
              </span>
              <span>当前模型定价审计存在提示</span>
            </div>
            <ul class="mt-2 space-y-1 text-xs">
              <li v-for="warning in currentDetail?.pricing_warnings || []" :key="warning">
                {{ warning }}
              </li>
            </ul>
          </div>

          <BillingPriceColumn
            title="官方价格"
            :description="officialDescription"
            :form="currentOfficialForm"
            :currency="currentCurrency"
            :usd-to-cny-rate="usdToCnyRate"
            :input-supported="currentDetail?.input_supported ?? true"
            :output-charge-slot="currentDetail?.output_charge_slot || 'text_output'"
            :supports-prompt-caching="currentDetail?.supports_prompt_caching ?? false"
            :capabilities="currentCapabilities"
            :disabled="currencySaveBlocked"
            column-test-id="official-column"
            @update-form="updateForm('official', $event)"
          >
            <template #actions>
              <button
                type="button"
                class="btn btn-primary btn-sm"
                data-testid="save-layer-official"
                :disabled="saveDisabled"
                @click="saveLayer('official')"
              >
                保存官方价
              </button>
            </template>
          </BillingPriceColumn>
        </div>

        <div class="space-y-4">
          <BillingBulkDiscountPanel
            :discount-ratio="discountRatio"
            :scope="discountScope"
            :selected-count="selectedSaleFieldIds.length"
            @copy-official="copyOfficial"
            @apply-all="applyDiscount(false)"
            @apply-selected="applyDiscount(true)"
            @update:discount-ratio="discountRatio = $event"
            @update:scope="discountScope = $event"
          />

          <BillingPriceColumn
            title="售价"
            description="保留复制官方价和批量折扣能力，但编辑界面统一收敛到紧凑表单。"
            :form="currentSaleForm"
            :currency="currentCurrency"
            :usd-to-cny-rate="usdToCnyRate"
            :input-supported="currentDetail?.input_supported ?? true"
            :output-charge-slot="currentDetail?.output_charge_slot || 'text_output'"
            :supports-prompt-caching="currentDetail?.supports_prompt_caching ?? false"
            :capabilities="currentCapabilities"
            :special-visibility="saleSpecialVisibility"
            :selected-ids="selectedSaleFieldIds"
            :disabled="currencySaveBlocked"
            selectable
            show-multiplier-controls
            column-test-id="sale-column"
            @toggle-select="toggleSaleSelection"
            @update-form="updateForm('sale', $event)"
          >
            <template #actions>
              <button
                type="button"
                class="btn btn-primary btn-sm"
                data-testid="save-layer-sale"
                :disabled="saveDisabled"
                @click="saveLayer('sale')"
              >
                保存售价
              </button>
            </template>
          </BillingPriceColumn>
        </div>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type {
  BillingPricingCapabilities,
  BillingPricingCurrency,
  BillingPricingLayerForm,
  BillingPricingSheetDetail,
} from '@/api/admin/billing'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useExchangeRateStore } from '@/stores/exchangeRate'
import BillingBulkDiscountPanel from './BillingBulkDiscountPanel.vue'
import BillingPriceColumn from './BillingPriceColumn.vue'
import BillingPricingCurrencyToolbar from './BillingPricingCurrencyToolbar.vue'
import {
  billingLayerHasSpecialValues,
  billingLayerHasValues,
  cloneBillingPricingLayerForm,
  countConfiguredBillingFields,
  createEmptyBillingPricingLayerForm,
  normalizeBillingPricingSheetDetail,
} from './pricingOptions'

type Layer = 'official' | 'sale'

const props = defineProps<{
  show: boolean
  details: BillingPricingSheetDetail[]
  activeModel: string
  busy?: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'update:activeModel', value: string): void
  (e: 'save-layer', payload: {
    model: string
    layer: Layer
    form: BillingPricingLayerForm
    currency: BillingPricingCurrency
  }): void
  (e: 'copy-official', payload: { models: string[] }): void
  (e: 'apply-discount', payload: {
    models: string[]
    itemIds?: string[]
    discountRatio: number
  }): void
}>()

const exchangeRateStore = useExchangeRateStore()

const search = ref('')
const discountRatio = ref(0.9)
const discountScope = ref<'current' | 'workset'>('current')
const selectedSaleFieldIds = ref<string[]>([])
const currentDetailMap = ref<Record<string, BillingPricingSheetDetail>>({})

watch(
  () => props.details,
  (details) => {
    currentDetailMap.value = Object.fromEntries(
      details.map((detail) => [detail.model, normalizeBillingPricingSheetDetail(detail)]),
    )
  },
  { immediate: true },
)

watch(
  () => props.activeModel,
  () => {
    selectedSaleFieldIds.value = []
  },
)

watch(
  () => props.show,
  (show) => {
    if (show) {
      exchangeRateStore.fetchExchangeRate()
    }
  },
  { immediate: true },
)

const detailList = computed(() => props.details.map((detail) => (
  currentDetailMap.value[detail.model] || normalizeBillingPricingSheetDetail(detail)
)))

const filteredDetails = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) {
    return detailList.value
  }

  return detailList.value.filter((detail) =>
    [detail.model, detail.display_name, detail.provider].some((value) =>
      String(value || '').toLowerCase().includes(keyword),
    ),
  )
})

const currentModel = computed(() => props.activeModel || detailList.value[0]?.model || '')
const currentDetail = computed(() => currentDetailMap.value[currentModel.value] || null)
const currentCurrency = computed<BillingPricingCurrency>(() => currentDetail.value?.currency || 'USD')
const usdToCnyRate = computed(() => exchangeRateStore.exchangeRate?.rate ?? null)
const cnyEnabled = computed(() => typeof usdToCnyRate.value === 'number' && usdToCnyRate.value > 0)
const currencySaveBlocked = computed(() => currentCurrency.value === 'CNY' && !cnyEnabled.value)
const saveDisabled = computed(() => props.busy || !currentModel.value || currencySaveBlocked.value)

const currentOfficialForm = computed(() => (
  currentDetail.value?.official_form || createEmptyBillingPricingLayerForm()
))
const currentSaleForm = computed(() => (
  currentDetail.value?.sale_form || createEmptyBillingPricingLayerForm()
))
const currentCapabilities = computed<BillingPricingCapabilities>(() => (
  currentDetail.value?.capabilities || {
    supports_tiered_pricing: false,
    supports_batch_pricing: false,
    supports_service_tier: false,
    supports_prompt_caching: false,
    supports_provider_special: false,
  }
))

const saleSpecialVisibility = computed(() => {
  const officialForm = currentOfficialForm.value
  const capabilities = currentCapabilities.value
  const forceSectionOpen = officialForm.special_enabled || billingLayerHasSpecialValues(officialForm)

  return {
    forceSectionOpen,
    forceBatchFields: officialForm.special_enabled
      ? capabilities.supports_batch_pricing
      : [
        officialForm.special.batch_input_price,
        officialForm.special.batch_output_price,
        officialForm.special.batch_cache_price,
      ].some((value) => value != null),
    forceProviderFields: officialForm.special_enabled
      ? capabilities.supports_provider_special
      : [
        officialForm.special.grounding_search,
        officialForm.special.grounding_maps,
        officialForm.special.file_search_embedding,
        officialForm.special.file_search_retrieval,
      ].some((value) => value != null),
  }
})

const officialDescription = computed(() => {
  const detail = currentDetail.value
  if (!detail) {
    return '统一编辑基础价、特殊价和阶梯价。'
  }

  return `${detail.display_name || detail.model} / ${detail.provider || '-'} / ${detail.mode || '-'} / ${detail.currency}`
})

function pricingStatusLabel(status?: BillingPricingSheetDetail['pricing_status']): string {
  switch (status) {
    case 'conflict':
      return '冲突'
    case 'missing':
      return '缺价'
    case 'fallback':
      return '回退'
    default:
      return '正常'
  }
}

function pricingStatusClass(status?: BillingPricingSheetDetail['pricing_status']): string {
  switch (status) {
    case 'conflict':
    case 'missing':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-200'
    case 'fallback':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-200'
    default:
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200'
  }
}

function updateForm(layer: Layer, form: BillingPricingLayerForm) {
  if (!currentDetail.value) {
    return
  }

  const next = normalizeBillingPricingSheetDetail(currentDetail.value)
  if (layer === 'official') {
    next.official_form = cloneBillingPricingLayerForm(form)
  } else {
    next.sale_form = cloneBillingPricingLayerForm(form)
  }

  currentDetailMap.value = {
    ...currentDetailMap.value,
    [next.model]: next,
  }
}

function updateCurrentCurrency(currency: BillingPricingCurrency) {
  if (!currentDetail.value) {
    return
  }

  currentDetailMap.value = {
    ...currentDetailMap.value,
    [currentDetail.value.model]: {
      ...currentDetail.value,
      currency,
    },
  }
}

function saveLayer(layer: Layer) {
  if (!currentModel.value) {
    return
  }

  emit('save-layer', {
    model: currentModel.value,
    layer,
    form: cloneBillingPricingLayerForm(
      layer === 'official' ? currentOfficialForm.value : currentSaleForm.value,
    ),
    currency: currentCurrency.value,
  })
}

function toggleSaleSelection(id: string) {
  selectedSaleFieldIds.value = selectedSaleFieldIds.value.includes(id)
    ? selectedSaleFieldIds.value.filter((itemId) => itemId !== id)
    : [...selectedSaleFieldIds.value, id]
}

function copyOfficial() {
  const models = normalizeModels(
    discountScope.value === 'workset'
      ? detailList.value.map((detail) => detail.model)
      : [currentModel.value],
  )
  if (models.length === 0) {
    return
  }

  emit('copy-official', { models })
}

function applyDiscount(selectedOnly: boolean) {
  const models = normalizeModels(
    discountScope.value === 'workset'
      ? detailList.value.map((detail) => detail.model)
      : [currentModel.value],
  )
  if (models.length === 0) {
    return
  }

  emit('apply-discount', {
    models,
    itemIds: selectedOnly ? [...selectedSaleFieldIds.value] : undefined,
    discountRatio: discountRatio.value,
  })
}

function normalizeModels(models: string[]): string[] {
  return Array.from(new Set(models.map((model) => model.trim()).filter(Boolean)))
}
</script>
