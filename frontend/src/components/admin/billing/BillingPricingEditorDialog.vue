<template>
  <BaseDialog
    :show="show"
    title="模型定价编辑"
    width="account-wide"
    close-on-click-outside
    @close="emit('close')"
  >
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
            <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ detail.provider || '-' }} / {{ detail.mode || '-' }}</div>
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

      <BillingPriceColumn
        title="官方价格"
        :description="officialDescription"
        :form="currentOfficialForm"
        :input-supported="currentDetail?.input_supported ?? true"
        :output-charge-slot="currentDetail?.output_charge_slot || 'text_output'"
        :supports-prompt-caching="currentDetail?.supports_prompt_caching ?? false"
        :capabilities="currentCapabilities"
        column-test-id="official-column"
        @update-form="updateForm('official', $event)"
      >
        <template #actions>
          <button
            type="button"
            class="btn btn-primary btn-sm"
            data-testid="save-layer-official"
            :disabled="busy || !currentModel"
            @click="saveLayer('official')"
          >
            保存官方价
          </button>
        </template>
      </BillingPriceColumn>

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
          description="保留复制官方价和批量折扣能力，但编辑界面统一收敛到简化表单。"
          :form="currentSaleForm"
          :input-supported="currentDetail?.input_supported ?? true"
          :output-charge-slot="currentDetail?.output_charge_slot || 'text_output'"
          :supports-prompt-caching="currentDetail?.supports_prompt_caching ?? false"
          :capabilities="currentCapabilities"
          :selected-ids="selectedSaleFieldIds"
          selectable
          column-test-id="sale-column"
          @toggle-select="toggleSaleSelection"
          @update-form="updateForm('sale', $event)"
        >
          <template #actions>
            <button
              type="button"
              class="btn btn-primary btn-sm"
              data-testid="save-layer-sale"
              :disabled="busy || !currentModel"
              @click="saveLayer('sale')"
            >
              保存售价
            </button>
          </template>
        </BillingPriceColumn>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { BillingPricingCapabilities, BillingPricingLayerForm, BillingPricingSheetDetail } from '@/api/admin/billing'
import BaseDialog from '@/components/common/BaseDialog.vue'
import BillingBulkDiscountPanel from './BillingBulkDiscountPanel.vue'
import BillingPriceColumn from './BillingPriceColumn.vue'
import {
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
  (e: 'save-layer', payload: { model: string; layer: Layer; form: BillingPricingLayerForm }): void
  (e: 'copy-official', payload: { models: string[] }): void
  (e: 'apply-discount', payload: { models: string[]; itemIds?: string[]; discountRatio: number }): void
}>()

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

const filteredDetails = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return props.details

  return props.details.filter((detail) =>
    [detail.model, detail.display_name, detail.provider].some((value) =>
      String(value || '').toLowerCase().includes(keyword),
    ),
  )
})

const currentModel = computed(() => props.activeModel || props.details[0]?.model || '')
const currentDetail = computed(() => currentDetailMap.value[currentModel.value] || null)
const currentOfficialForm = computed(() => currentDetail.value?.official_form || createEmptyBillingPricingLayerForm())
const currentSaleForm = computed(() => currentDetail.value?.sale_form || createEmptyBillingPricingLayerForm())
const currentCapabilities = computed<BillingPricingCapabilities>(() => currentDetail.value?.capabilities || {
  supports_tiered_pricing: false,
  supports_batch_pricing: false,
  supports_service_tier: false,
  supports_prompt_caching: false,
  supports_provider_special: false,
})

const officialDescription = computed(() => {
  const detail = currentDetail.value
  if (!detail) return '统一编辑基础价、特殊价和阶梯价。'
  return `${detail.display_name || detail.model} / ${detail.provider || '-'} / ${detail.mode || '-'}`
})

function updateForm(layer: Layer, form: BillingPricingLayerForm) {
  if (!currentDetail.value) return

  const next = normalizeBillingPricingSheetDetail(currentDetail.value)
  if (layer === 'official') next.official_form = cloneBillingPricingLayerForm(form)
  else next.sale_form = cloneBillingPricingLayerForm(form)

  currentDetailMap.value = {
    ...currentDetailMap.value,
    [next.model]: next,
  }
}

function saveLayer(layer: Layer) {
  if (!currentModel.value) return

  emit('save-layer', {
    model: currentModel.value,
    layer,
    form: cloneBillingPricingLayerForm(layer === 'official' ? currentOfficialForm.value : currentSaleForm.value),
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
      ? props.details.map((detail) => detail.model)
      : [currentModel.value],
  )
  if (models.length === 0) return
  emit('copy-official', { models })
}

function applyDiscount(selectedOnly: boolean) {
  const models = normalizeModels(
    discountScope.value === 'workset'
      ? props.details.map((detail) => detail.model)
      : [currentModel.value],
  )
  if (models.length === 0) return

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
