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
        <input v-model.trim="search" type="text" class="input mt-3 w-full" placeholder="搜索当前工作集模型" />
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
            <div class="mt-2 flex flex-wrap gap-1 text-[11px]">
              <span class="inline-flex rounded-full bg-sky-100 px-2 py-1 text-sky-700 dark:bg-sky-500/15 dark:text-sky-200">官方 {{ currentDetailMap[detail.model]?.official_items?.length || 0 }}</span>
              <span class="inline-flex rounded-full bg-emerald-100 px-2 py-1 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200">出售 {{ currentDetailMap[detail.model]?.sale_items?.length || 0 }}</span>
            </div>
          </button>
        </div>
      </aside>

      <BillingPriceColumn
        title="官方价格"
        :description="officialDescription"
        :items="currentOfficialItems"
        @update-item="updateItem('official', $event)"
        @remove-item="removeItem('official', $event)"
      >
        <template #actions>
          <button type="button" class="btn btn-secondary btn-sm" @click="addItem('official')">新增价格项</button>
          <button
            v-for="preset in presetActions"
            :key="preset.kind"
            type="button"
            class="btn btn-secondary btn-sm"
            @click="applyPreset(preset.kind)"
          >
            {{ preset.label }}
          </button>
          <button type="button" class="btn btn-primary btn-sm" :disabled="busy || !currentModel" @click="saveLayer('official')">保存官方</button>
        </template>
      </BillingPriceColumn>

      <div class="space-y-4">
        <BillingBulkDiscountPanel
          :discount-ratio="discountRatio"
          :scope="discountScope"
          :selected-count="selectedSaleItemIds.length"
          @copy-official="copyOfficial"
          @apply-all="applyDiscount(false)"
          @apply-selected="applyDiscount(true)"
          @update:discount-ratio="discountRatio = $event"
          @update:scope="discountScope = $event"
        />

        <BillingPriceColumn
          title="出售价格"
          description="可复制官方结构，再按整单或选中项做比例折扣。"
          :items="currentSaleItems"
          :selected-ids="selectedSaleItemIds"
          selectable
          @toggle-select="toggleSaleSelection"
          @update-item="updateItem('sale', $event)"
          @remove-item="removeItem('sale', $event)"
        >
          <template #actions>
            <button type="button" class="btn btn-secondary btn-sm" @click="addItem('sale')">新增价格项</button>
            <button type="button" class="btn btn-primary btn-sm" :disabled="busy || !currentModel" @click="saveLayer('sale')">保存售价</button>
          </template>
        </BillingPriceColumn>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { BillingPriceItem, BillingPricingSheetDetail } from '@/api/admin/billing'
import BillingBulkDiscountPanel from './BillingBulkDiscountPanel.vue'
import BillingPriceColumn from './BillingPriceColumn.vue'
import { newBillingPriceItem } from './pricingOptions'

type Layer = 'official' | 'sale'
type Preset = 'tiered' | 'batch' | 'service_tier' | 'cache' | 'special'

const props = defineProps<{
  show: boolean
  details: BillingPricingSheetDetail[]
  activeModel: string
  busy?: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'update:activeModel', value: string): void
  (e: 'save-layer', payload: { model: string; layer: Layer; items: BillingPriceItem[] }): void
  (e: 'copy-official', payload: { models: string[] }): void
  (e: 'apply-discount', payload: { models: string[]; itemIds?: string[]; discountRatio: number }): void
}>()

const search = ref('')
const discountRatio = ref(0.9)
const discountScope = ref<'current' | 'workset'>('current')
const selectedSaleItemIds = ref<string[]>([])
const currentDetailMap = ref<Record<string, BillingPricingSheetDetail>>({})

watch(
  () => props.details,
  (details) => {
    currentDetailMap.value = Object.fromEntries(details.map((detail) => [detail.model, cloneDetail(detail)]))
  },
  { immediate: true },
)

watch(
  () => props.activeModel,
  () => {
    selectedSaleItemIds.value = []
  },
)

const filteredDetails = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return props.details
  return props.details.filter((detail) =>
    [detail.model, detail.display_name, detail.provider].some((value) => String(value || '').toLowerCase().includes(keyword)),
  )
})

const currentModel = computed(() => props.activeModel || props.details[0]?.model || '')
const currentDetail = computed(() => currentDetailMap.value[currentModel.value] || null)
const currentOfficialItems = computed(() => currentDetail.value?.official_items || [])
const currentSaleItems = computed(() => currentDetail.value?.sale_items || [])
const presetActions = computed(() => {
  const capabilities = currentDetail.value?.capabilities
  if (!capabilities) {
    return [
      { kind: 'tiered' as const, label: '启用阶梯' },
      { kind: 'batch' as const, label: '启用 Batch' },
      { kind: 'service_tier' as const, label: '启用层级' },
      { kind: 'cache' as const, label: '启用缓存' },
      { kind: 'special' as const, label: 'Provider Special' },
    ]
  }

  const actions: Array<{ kind: Preset; label: string }> = []
  if (capabilities.supports_tiered_pricing) actions.push({ kind: 'tiered', label: '启用阶梯' })
  if (capabilities.supports_batch_pricing) actions.push({ kind: 'batch', label: '启用 Batch' })
  if (capabilities.supports_service_tier) actions.push({ kind: 'service_tier', label: '启用层级' })
  if (capabilities.supports_prompt_caching) actions.push({ kind: 'cache', label: '启用缓存' })
  if (capabilities.supports_provider_special) actions.push({ kind: 'special', label: 'Provider Special' })
  return actions
})
const officialDescription = computed(() => {
  const detail = currentDetail.value
  if (!detail) return '可开启阶梯、Batch、服务层级、缓存与 Provider Special。'
  return `${detail.display_name || detail.model} · ${detail.provider || '-'} · ${detail.mode || '-'}`
})

function cloneDetail(detail: BillingPricingSheetDetail): BillingPricingSheetDetail {
  return JSON.parse(JSON.stringify(detail))
}

function replaceLayerItems(layer: Layer, items: BillingPriceItem[]) {
  if (!currentDetail.value) return
  const next = cloneDetail(currentDetail.value)
  if (layer === 'official') next.official_items = items
  else next.sale_items = items
  currentDetailMap.value = { ...currentDetailMap.value, [next.model]: next }
}

function updateItem(layer: Layer, item: BillingPriceItem) {
  const items = (layer === 'official' ? currentOfficialItems.value : currentSaleItems.value).map((entry) =>
    entry.id === item.id ? { ...item, layer } : entry,
  )
  replaceLayerItems(layer, items)
}

function addItem(layer: Layer) {
  const items = [...(layer === 'official' ? currentOfficialItems.value : currentSaleItems.value), newBillingPriceItem(layer)]
  replaceLayerItems(layer, items)
}

function removeItem(layer: Layer, id: string) {
  const items = (layer === 'official' ? currentOfficialItems.value : currentSaleItems.value).filter((item) => item.id !== id)
  replaceLayerItems(layer, items)
  selectedSaleItemIds.value = selectedSaleItemIds.value.filter((itemId) => itemId !== id)
}

function toggleSaleSelection(id: string) {
  selectedSaleItemIds.value = selectedSaleItemIds.value.includes(id)
    ? selectedSaleItemIds.value.filter((itemId) => itemId !== id)
    : [...selectedSaleItemIds.value, id]
}

function applyPreset(kind: Preset) {
  const baseItems = ensureBaseItems(currentOfficialItems.value)
  switch (kind) {
    case 'tiered':
      replaceLayerItems('official', baseItems.map((item) => (
        item.charge_slot === 'text_input' || item.charge_slot === 'text_output'
          ? { ...item, mode: 'tiered', threshold_tokens: item.threshold_tokens ?? 200000, price_above_threshold: item.price_above_threshold ?? item.price }
          : item
      )))
      break
    case 'batch':
      replaceLayerItems('official', mergePresetItems(baseItems, baseItems
          .filter((item) => item.charge_slot === 'text_input' || item.charge_slot === 'text_output' || item.charge_slot === 'cache_create' || item.charge_slot === 'cache_read')
          .map((item) => newBillingPriceItem('official', {
            charge_slot: item.charge_slot,
            unit: item.unit,
            mode: 'batch',
            batch_mode: 'batch',
            price: Number((item.price * 0.5).toFixed(8)),
            formula_source: item.id,
            formula_multiplier: 0.5,
          })),
      ))
      break
    case 'service_tier':
      replaceLayerItems('official', mergePresetItems(baseItems, baseItems
          .filter((item) => ['text_input', 'text_output', 'cache_read', 'image_output'].includes(item.charge_slot))
          .flatMap((item) => [
            newBillingPriceItem('official', { charge_slot: item.charge_slot, unit: item.unit, mode: 'service_tier', service_tier: 'priority', price: Number((item.price * 2).toFixed(8)) }),
            newBillingPriceItem('official', { charge_slot: item.charge_slot, unit: item.unit, mode: 'service_tier', service_tier: 'flex', price: Number((item.price * 0.5).toFixed(8)) }),
          ]),
      ))
      break
    case 'cache':
      replaceLayerItems('official', mergePresetItems(baseItems, [
        newBillingPriceItem('official', { charge_slot: 'cache_create' }),
        newBillingPriceItem('official', { charge_slot: 'cache_read' }),
        newBillingPriceItem('official', { charge_slot: 'cache_storage_token_hour' }),
      ]))
      break
    case 'special':
      replaceLayerItems('official', mergePresetItems(baseItems, [
        newBillingPriceItem('official', { charge_slot: 'grounding_search_request', mode: 'provider_special' }),
        newBillingPriceItem('official', { charge_slot: 'grounding_maps_request', mode: 'provider_special' }),
        newBillingPriceItem('official', { charge_slot: 'file_search_embedding_token', mode: 'provider_special' }),
      ]))
      break
  }
}

function saveLayer(layer: Layer) {
  if (!currentModel.value) return
  emit('save-layer', {
    model: currentModel.value,
    layer,
    items: cloneItems(layer === 'official' ? currentOfficialItems.value : currentSaleItems.value, layer),
  })
}

function copyOfficial() {
  const models = normalizeModels(discountScope.value === 'workset' ? props.details.map((detail) => detail.model) : [currentModel.value])
  if (models.length === 0) return
  emit('copy-official', { models })
}

function applyDiscount(selectedOnly: boolean) {
  const models = normalizeModels(discountScope.value === 'workset' ? props.details.map((detail) => detail.model) : [currentModel.value])
  if (models.length === 0) return
  emit('apply-discount', {
    models,
    itemIds: selectedOnly ? selectedSaleItemIds.value : undefined,
    discountRatio: discountRatio.value,
  })
}

function cloneItems(items: BillingPriceItem[], layer: Layer): BillingPriceItem[] {
  return items.map((item) => ({ ...item, layer }))
}

function ensureBaseItems(items: BillingPriceItem[]): BillingPriceItem[] {
  if (items.length > 0) return [...items]
  return [
    newBillingPriceItem('official', { charge_slot: 'text_input' }),
    newBillingPriceItem('official', { charge_slot: 'text_output' }),
  ]
}

function mergePresetItems(current: BillingPriceItem[], additions: BillingPriceItem[]): BillingPriceItem[] {
  const merged = new Map<string, BillingPriceItem>()
  for (const item of current) {
    const key = [
      item.charge_slot,
      item.mode,
      item.service_tier || '',
      item.batch_mode || '',
      item.surface || '',
      item.operation_type || '',
      item.input_modality || '',
      item.output_modality || '',
      item.cache_phase || '',
      item.grounding_kind || '',
      item.context_window || '',
    ].join('|')
    merged.set(key, item)
  }
  for (const item of additions) {
    const key = [
      item.charge_slot,
      item.mode,
      item.service_tier || '',
      item.batch_mode || '',
      item.surface || '',
      item.operation_type || '',
      item.input_modality || '',
      item.output_modality || '',
      item.cache_phase || '',
      item.grounding_kind || '',
      item.context_window || '',
    ].join('|')
    merged.set(key, item)
  }
  return [...merged.values()]
}

function normalizeModels(models: string[]): string[] {
  return Array.from(new Set(models.map((model) => model.trim()).filter(Boolean)))
}
</script>
