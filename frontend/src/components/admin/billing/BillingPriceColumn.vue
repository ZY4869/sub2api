<template>
  <section
    class="flex min-h-0 flex-col rounded-3xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900/40"
    :data-testid="columnTestId"
  >
    <div class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 px-4 py-4 dark:border-dark-700">
      <div>
        <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ title }}</h3>
        <p v-if="description" class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ description }}</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <slot name="actions" />
      </div>
    </div>

    <div class="flex-1 space-y-4 overflow-y-auto px-4 py-4">
      <article class="rounded-2xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-800">
        <div class="flex items-center justify-between gap-3">
          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">基础区</h4>
        </div>

        <div class="mt-4 space-y-3">
          <BillingPricingCompactFieldRow
            v-for="field in baseFields"
            :key="field.id"
            :field-id="field.id"
            :label="field.label"
            :unit-label="field.unitLabel"
            :value="field.value"
            :selectable="selectable"
            :selected="selectedIds.includes(field.id)"
            @toggle-select="emit('toggle-select', field.id)"
            @update:value="updateRootNumber(field.field, $event)"
          />
        </div>
      </article>

      <article
        v-if="supportsSpecialSection"
        class="rounded-2xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-800"
      >
        <div class="flex items-center justify-between gap-3">
          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">特殊区</h4>
          <Toggle
            :model-value="form.special_enabled"
            :data-testid="'pricing-special-toggle'"
            @update:model-value="toggleSpecialEnabled"
          />
        </div>

        <div v-if="showSpecialFields" class="mt-4 space-y-3">
          <BillingPricingCompactFieldRow
            v-for="field in specialFields"
            :key="field.id"
            :field-id="field.id"
            :label="field.label"
            :unit-label="field.unitLabel"
            :value="field.value"
            :selectable="selectable"
            :selected="selectedIds.includes(field.id)"
            @toggle-select="emit('toggle-select', field.id)"
            @update:value="updateSpecialNumber(field.field, $event)"
          />
        </div>

        <p v-else class="mt-4 text-xs text-gray-500 dark:text-gray-400">打开后才会显示 Batch 和 Gemini 的特殊价格输入项。</p>
      </article>

      <article
        v-if="supportsTierSection"
        class="rounded-2xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-800"
      >
        <div class="flex items-center justify-between gap-3">
          <div>
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">阶梯区</h4>
            <p class="mt-1 text-xs text-gray-600 dark:text-gray-300">单开关、单阈值，同时控制文本输入和文本输出。</p>
          </div>
          <Toggle
            :model-value="form.tiered_enabled"
            :data-testid="'pricing-tier-toggle'"
            @update:model-value="toggleTieredEnabled"
          />
        </div>

        <div v-if="form.tiered_enabled" class="mt-4 space-y-3">
          <div class="rounded-2xl border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-900/60">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <span class="text-sm font-medium text-gray-900 dark:text-white">共享阈值</span>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">同一个 Token 阈值同时作用于输入价和输出价。</p>
              </div>
              <input
                class="input w-full max-w-[220px]"
                type="number"
                step="1"
                :value="form.tier_threshold_tokens ?? ''"
                data-testid="pricing-field-tier_threshold_tokens"
                @input="updateTierThreshold(($event.target as HTMLInputElement).value)"
              />
            </div>
          </div>

          <div
            v-for="field in tierFields"
            :key="field.id"
            class="rounded-2xl border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-900/60"
          >
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <label
                    v-if="selectable"
                    class="inline-flex items-center gap-2 text-xs text-gray-600 dark:text-gray-300"
                    :data-testid="`field-select-${field.id}`"
                  >
                    <input
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-primary-600"
                      :checked="selectedIds.includes(field.id)"
                      @change="emit('toggle-select', field.id)"
                    />
                    选中
                  </label>
                  <span class="text-sm font-medium text-gray-900 dark:text-white">{{ field.label }}</span>
                </div>
                <p v-if="field.hint" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ field.hint }}</p>
              </div>
              <input
                class="input w-full max-w-[220px]"
                type="number"
                step="0.0000001"
                :value="field.value ?? ''"
                :data-testid="`pricing-field-${field.id}`"
                @input="updateRootNumber(field.field, ($event.target as HTMLInputElement).value)"
              />
            </div>
          </div>
        </div>

        <p v-else class="mt-4 text-xs text-gray-500 dark:text-gray-400">打开后才会显示共享阈值和阈值后的价格。</p>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type {
  BillingPricingCapabilities,
  BillingPricingLayerForm,
} from '@/api/admin/billing'
import Toggle from '@/components/common/Toggle.vue'
import BillingPricingCompactFieldRow from './BillingPricingCompactFieldRow.vue'
import {
  pricingFieldUnitLabelForField,
  type BillingPricingFieldId,
  type RootNumberField,
  type SpecialNumberField,
} from './pricingFieldPresentation'
import {
  cloneBillingPricingLayerForm,
  createEmptyBillingPricingSpecial,
  outputPriceLabel,
} from './pricingOptions'

interface BillingSpecialVisibility {
  forceSectionOpen?: boolean
  forceBatchFields?: boolean
  forceProviderFields?: boolean
}

interface PricingFieldDescriptor<T extends BillingPricingFieldId> {
  id: string
  label: string
  unitLabel: string
  value?: number
  field: T
}

interface TierFieldDescriptor<T extends RootNumberField> {
  id: string
  label: string
  hint?: string
  value?: number
  field: T
}

const props = withDefaults(defineProps<{
  title: string
  description?: string
  form: BillingPricingLayerForm
  inputSupported: boolean
  outputChargeSlot?: string
  supportsPromptCaching?: boolean
  capabilities: BillingPricingCapabilities
  selectedIds?: string[]
  selectable?: boolean
  columnTestId?: string
  specialVisibility?: BillingSpecialVisibility
}>(), {
  description: '',
  outputChargeSlot: 'text_output',
  supportsPromptCaching: false,
  selectedIds: () => [],
  selectable: false,
  columnTestId: undefined,
  specialVisibility: () => ({}),
})

const emit = defineEmits<{
  (e: 'update-form', value: BillingPricingLayerForm): void
  (e: 'toggle-select', id: string): void
}>()

const showCachePricing = computed(() => (
  props.supportsPromptCaching
  || props.form.cache_price != null
  || props.form.special.batch_cache_price != null
))

const supportsBatchPricing = computed(() => (
  props.specialVisibility.forceBatchFields
  || (
    props.specialVisibility.forceSectionOpen
    && props.capabilities.supports_batch_pricing
  )
  || props.capabilities.supports_batch_pricing
  || [
    props.form.special.batch_input_price,
    props.form.special.batch_output_price,
    props.form.special.batch_cache_price,
  ].some((value) => value != null)
))

const supportsProviderSpecial = computed(() => (
  props.specialVisibility.forceProviderFields
  || (
    props.specialVisibility.forceSectionOpen
    && props.capabilities.supports_provider_special
  )
  || props.capabilities.supports_provider_special
  || [
    props.form.special.grounding_search,
    props.form.special.grounding_maps,
    props.form.special.file_search_embedding,
    props.form.special.file_search_retrieval,
  ].some((value) => value != null)
))

const supportsSpecialSection = computed(() => supportsBatchPricing.value || supportsProviderSpecial.value)
const showSpecialFields = computed(() => props.form.special_enabled || props.specialVisibility.forceSectionOpen)
const supportsTierSection = computed(() => (
  (props.capabilities.supports_tiered_pricing && props.outputChargeSlot === 'text_output')
  || props.form.tiered_enabled
  || props.form.tier_threshold_tokens != null
  || props.form.input_price_above_threshold != null
  || props.form.output_price_above_threshold != null
))

const baseFields = computed<PricingFieldDescriptor<RootNumberField>[]>(() => {
  const fields: PricingFieldDescriptor<RootNumberField>[] = []

  if (props.inputSupported) {
    fields.push({
      id: 'input_price',
      label: '输入定价',
      unitLabel: pricingFieldUnitLabelForField('input_price', props.outputChargeSlot),
      value: props.form.input_price,
      field: 'input_price',
    })
  }

  fields.push({
    id: 'output_price',
    label: outputPriceLabel(props.outputChargeSlot),
    unitLabel: pricingFieldUnitLabelForField('output_price', props.outputChargeSlot),
    value: props.form.output_price,
    field: 'output_price',
  })

  if (showCachePricing.value) {
    fields.push({
      id: 'cache_price',
      label: '缓存定价',
      unitLabel: pricingFieldUnitLabelForField('cache_price', props.outputChargeSlot),
      value: props.form.cache_price,
      field: 'cache_price',
    })
  }

  return fields
})

const specialFields = computed<PricingFieldDescriptor<SpecialNumberField>[]>(() => {
  if (!showSpecialFields.value) return []

  const fields: PricingFieldDescriptor<SpecialNumberField>[] = []

  if (supportsBatchPricing.value) {
    if (props.inputSupported) {
      fields.push({
        id: 'batch_input_price',
        label: 'Batch 输入定价',
        unitLabel: pricingFieldUnitLabelForField('batch_input_price', props.outputChargeSlot),
        value: props.form.special.batch_input_price,
        field: 'batch_input_price',
      })
    }

    fields.push({
      id: 'batch_output_price',
      label: `Batch ${outputPriceLabel(props.outputChargeSlot)}`,
      unitLabel: pricingFieldUnitLabelForField('batch_output_price', props.outputChargeSlot),
      value: props.form.special.batch_output_price,
      field: 'batch_output_price',
    })

    if (showCachePricing.value) {
      fields.push({
        id: 'batch_cache_price',
        label: 'Batch 缓存定价',
        unitLabel: pricingFieldUnitLabelForField('batch_cache_price', props.outputChargeSlot),
        value: props.form.special.batch_cache_price,
        field: 'batch_cache_price',
      })
    }
  }

  if (supportsProviderSpecial.value) {
    fields.push(
      {
        id: 'grounding_search',
        label: 'Grounding Search',
        unitLabel: pricingFieldUnitLabelForField('grounding_search', props.outputChargeSlot),
        value: props.form.special.grounding_search,
        field: 'grounding_search',
      },
      {
        id: 'grounding_maps',
        label: 'Grounding Maps',
        unitLabel: pricingFieldUnitLabelForField('grounding_maps', props.outputChargeSlot),
        value: props.form.special.grounding_maps,
        field: 'grounding_maps',
      },
      {
        id: 'file_search_embedding',
        label: 'File Search Embedding',
        unitLabel: pricingFieldUnitLabelForField('file_search_embedding', props.outputChargeSlot),
        value: props.form.special.file_search_embedding,
        field: 'file_search_embedding',
      },
      {
        id: 'file_search_retrieval',
        label: 'File Search Retrieval',
        unitLabel: pricingFieldUnitLabelForField('file_search_retrieval', props.outputChargeSlot),
        value: props.form.special.file_search_retrieval,
        field: 'file_search_retrieval',
      },
    )
  }

  return fields
})

const tierFields = computed<TierFieldDescriptor<'input_price_above_threshold' | 'output_price_above_threshold'>[]>(() => {
  if (!props.form.tiered_enabled) return []

  const fields: TierFieldDescriptor<'input_price_above_threshold' | 'output_price_above_threshold'>[] = []

  if (props.inputSupported) {
    fields.push({
      id: 'input_price_above_threshold',
      label: '输入阈值后定价',
      hint: '超过共享阈值后的输入单价。',
      value: props.form.input_price_above_threshold,
      field: 'input_price_above_threshold',
    })
  }

  if (props.outputChargeSlot === 'text_output') {
    fields.push({
      id: 'output_price_above_threshold',
      label: '输出阈值后定价',
      hint: '超过共享阈值后的输出单价。',
      value: props.form.output_price_above_threshold,
      field: 'output_price_above_threshold',
    })
  }

  return fields
})

function emitForm(next: BillingPricingLayerForm) {
  emit('update-form', cloneBillingPricingLayerForm(next))
}

function normalizeOptionalNumber(raw: string, integer = false): number | undefined {
  const normalized = raw.trim()
  if (!normalized) return undefined

  const parsed = integer ? Number.parseInt(normalized, 10) : Number(normalized)
  return Number.isFinite(parsed) ? parsed : undefined
}

function updateRootNumber(field: RootNumberField, raw: string) {
  const next = cloneBillingPricingLayerForm(props.form)
  next[field] = normalizeOptionalNumber(raw) as BillingPricingLayerForm[RootNumberField]
  emitForm(next)
}

function updateSpecialNumber(field: SpecialNumberField, raw: string) {
  const next = cloneBillingPricingLayerForm(props.form)
  next.special = {
    ...next.special,
    [field]: normalizeOptionalNumber(raw),
  }
  next.special_enabled = true
  emitForm(next)
}

function updateTierThreshold(raw: string) {
  const next = cloneBillingPricingLayerForm(props.form)
  next.tier_threshold_tokens = normalizeOptionalNumber(raw, true)
  next.tiered_enabled = true
  emitForm(next)
}

function toggleSpecialEnabled(value: boolean) {
  const next = cloneBillingPricingLayerForm(props.form)
  next.special_enabled = value
  if (!value) {
    next.special = createEmptyBillingPricingSpecial()
  }
  emitForm(next)
}

function toggleTieredEnabled(value: boolean) {
  const next = cloneBillingPricingLayerForm(props.form)
  next.tiered_enabled = value
  if (!value) {
    next.tier_threshold_tokens = undefined
    next.input_price_above_threshold = undefined
    next.output_price_above_threshold = undefined
  }
  emitForm(next)
}
</script>
