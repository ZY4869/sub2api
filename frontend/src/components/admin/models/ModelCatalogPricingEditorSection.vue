<template>
  <div :class="sectionClass">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ sectionTitle }}</h4>
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ sectionDescription }}</p>
      </div>
      <div class="rounded-lg bg-white/70 px-3 py-2 text-xs text-gray-500 dark:bg-dark-900/60 dark:text-gray-400">
        <div>{{ t('admin.models.units.perMillionTokens') }}</div>
        <div>{{ t('admin.models.units.perImage') }}</div>
        <div>{{ t('admin.models.units.perVideoRequest') }}</div>
      </div>
    </div>

    <section v-for="group in groupedFields" :key="group.key" class="space-y-3">
      <div>
        <h5 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t(group.labelKey) }}</h5>
      </div>
      <div class="grid gap-4 md:grid-cols-2">
        <label v-for="field in group.fields" :key="field.key" class="block">
          <span class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-200">{{ t(field.labelKey) }}</span>
          <input
            :value="formValues[field.key]"
            type="number"
            :min="field.unit === 'threshold' ? 1 : 0"
            :step="field.unit === 'threshold' ? 1 : 0.000001"
            class="input"
            :placeholder="placeholder(field.unit)"
            @input="updateField(field.key, String(($event.target as HTMLInputElement).value))"
          />
          <p
            v-if="field.unit === 'threshold'"
            class="mt-1 text-[11px] text-gray-500 dark:text-gray-400"
          >
            {{ t('admin.models.editor.tierHint', tierDescription) }}
          </p>
          <span v-if="validationErrors[field.key]" class="mt-1 block text-xs text-red-500 dark:text-red-400">
            {{ validationErrors[field.key] }}
          </span>
        </label>
      </div>
    </section>

    <div class="flex flex-wrap justify-end gap-3">
      <button class="btn btn-secondary" :disabled="!activeOverride || saving" @click="emit('reset', detail.model)">
        {{ resetLabel }}
      </button>
      <button class="btn btn-primary" :disabled="!hasChanges || hasValidationErrors || saving" @click="handleSave">
        {{ saving ? t('admin.models.saving') : saveLabel }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  ModelCatalogDetail,
  UpdatePricingOverridePayload
} from '@/api/admin/models'
import {
  MODEL_CATALOG_PRICING_FIELDS,
  MODEL_CATALOG_PRICING_GROUPS,
  parsePricingInput,
  pricingInputValue,
  type ModelCatalogPricingKey,
  type ModelCatalogPricingUnit
} from '@/utils/modelCatalogPricing'
import {
  buildModelCatalogTierDescription,
  MODEL_CATALOG_DEFAULT_THRESHOLD
} from '@/utils/modelCatalogPresentation'

const props = defineProps<{
  detail: ModelCatalogDetail
  layer: 'official' | 'sale'
  saving: boolean
}>()

const emit = defineEmits<{
  (e: 'save', payload: UpdatePricingOverridePayload): void
  (e: 'reset', model: string): void
}>()

const { t } = useI18n()
const formValues = reactive<Record<string, string>>({})
const initialValues = reactive<Record<string, string>>({})
const virtualThresholds = reactive<Record<string, boolean>>({})

const tierDescription = buildModelCatalogTierDescription(MODEL_CATALOG_DEFAULT_THRESHOLD)

const groupedFields = computed(() =>
  MODEL_CATALOG_PRICING_GROUPS.map((group) => ({
    ...group,
    fields: MODEL_CATALOG_PRICING_FIELDS.filter((field) => field.group === group.key)
  }))
)

const activeOverride = computed(() =>
  props.layer === 'official' ? props.detail.official_override_pricing : props.detail.sale_override_pricing
)

const effectivePricing = computed(() =>
  props.layer === 'official' ? props.detail.official_pricing : props.detail.sale_pricing
)

const sectionTitle = computed(() =>
  props.layer === 'official'
    ? t('admin.models.editor.officialTitle')
    : t('admin.models.editor.saleTitle')
)

const sectionDescription = computed(() =>
  props.layer === 'official'
    ? t('admin.models.editor.officialDescription')
    : t('admin.models.editor.saleDescription')
)

const saveLabel = computed(() =>
  props.layer === 'official'
    ? t('admin.models.editor.saveOfficial')
    : t('admin.models.editor.saveSale')
)

const resetLabel = computed(() =>
  props.layer === 'official'
    ? t('admin.models.editor.resetOfficial')
    : t('admin.models.editor.resetSale')
)

const sectionClass = computed(() =>
  props.layer === 'official'
    ? 'space-y-4 rounded-2xl border border-sky-200 bg-sky-50/70 p-4 dark:border-sky-500/20 dark:bg-sky-500/10'
    : 'space-y-4 rounded-2xl border border-emerald-200 bg-emerald-50/70 p-4 dark:border-emerald-500/20 dark:bg-emerald-500/10'
)

watch(
  () => [props.detail, props.layer],
  () => initializeForm(),
  { immediate: true, deep: true }
)

const parsedValues = computed<Partial<Record<ModelCatalogPricingKey, number>>>(() => {
  const parsed: Partial<Record<ModelCatalogPricingKey, number>> = {}
  for (const field of MODEL_CATALOG_PRICING_FIELDS) {
    const value = parsePricingInput(formValues[field.key] ?? '', field.unit)
    if (typeof value === 'number') {
      parsed[field.key] = value
    }
  }
  return parsed
})

const payload = computed<UpdatePricingOverridePayload>(() => {
  const next: UpdatePricingOverridePayload = { model: props.detail.model }
  for (const field of MODEL_CATALOG_PRICING_FIELDS) {
    if (formValues[field.key] === initialValues[field.key]) {
      continue
    }
    const parsed = parsePricingInput(formValues[field.key] ?? '', field.unit)
    if (typeof parsed === 'number') {
      next[field.key] = parsed as never
    }
  }
  appendVirtualThreshold(next, 'input_token_threshold', 'input_cost_per_token_above_threshold', 'input_cost_per_token_priority_above_threshold')
  appendVirtualThreshold(next, 'output_token_threshold', 'output_cost_per_token_above_threshold', 'output_cost_per_token_priority_above_threshold')
  return next
})

const hasChanges = computed(() => Object.keys(payload.value).length > 1)

const validationErrors = computed<Partial<Record<ModelCatalogPricingKey, string>>>(() => {
  const errors: Partial<Record<ModelCatalogPricingKey, string>> = {}
  for (const field of MODEL_CATALOG_PRICING_FIELDS) {
    const rawValue = (formValues[field.key] ?? '').trim()
    if (!rawValue) {
      if (formValues[field.key] !== initialValues[field.key]) {
        errors[field.key] = t('admin.models.editor.validationRequired')
      }
      continue
    }
    if (typeof parsedValues.value[field.key] !== 'number') {
      errors[field.key] = field.unit === 'threshold'
        ? t('admin.models.editor.validationPositiveInteger')
        : t('admin.models.editor.validationNonNegative')
    }
  }
  validateTier(
    errors,
    'input_token_threshold',
    'input_cost_per_token_above_threshold',
    'input_cost_per_token_priority',
    'input_cost_per_token_priority_above_threshold'
  )
  validateTier(
    errors,
    'output_token_threshold',
    'output_cost_per_token_above_threshold',
    'output_cost_per_token_priority',
    'output_cost_per_token_priority_above_threshold'
  )
  return errors
})

const hasValidationErrors = computed(() => Object.keys(validationErrors.value).length > 0)

function initializeForm() {
  for (const field of MODEL_CATALOG_PRICING_FIELDS) {
    const value = activeOverride.value?.[field.key] ?? effectivePricing.value?.[field.key]
    if (field.unit === 'threshold' && typeof value !== 'number') {
      formValues[field.key] = String(MODEL_CATALOG_DEFAULT_THRESHOLD)
      initialValues[field.key] = String(MODEL_CATALOG_DEFAULT_THRESHOLD)
      virtualThresholds[field.key] = true
      continue
    }
    const formatted = pricingInputValue(value, field.unit)
    formValues[field.key] = formatted
    initialValues[field.key] = formatted
    virtualThresholds[field.key] = false
  }
}

function hasSourceValue(key: ModelCatalogPricingKey) {
  return typeof (activeOverride.value?.[key] ?? effectivePricing.value?.[key]) === 'number'
}

function fieldChanged(key: ModelCatalogPricingKey) {
  return (formValues[key] ?? '') !== (initialValues[key] ?? '')
}

function validateTier(
  errors: Partial<Record<ModelCatalogPricingKey, string>>,
  thresholdKey: ModelCatalogPricingKey,
  aboveKey: ModelCatalogPricingKey,
  priorityBaseKey: ModelCatalogPricingKey,
  priorityAboveKey: ModelCatalogPricingKey
) {
  const tierTouched =
    hasSourceValue(thresholdKey) ||
    hasSourceValue(aboveKey) ||
    hasSourceValue(priorityAboveKey) ||
    fieldChanged(thresholdKey) ||
    fieldChanged(aboveKey) ||
    fieldChanged(priorityAboveKey)

  if (!tierTouched) {
    return
  }
  if (typeof parsedValues.value[thresholdKey] !== 'number') {
    errors[thresholdKey] = t('admin.models.editor.validationPositiveInteger')
    return
  }
  if (typeof parsedValues.value[aboveKey] !== 'number') {
    errors[aboveKey] = t('admin.models.editor.validationAboveThresholdRequired')
  }
  if (
    typeof parsedValues.value[priorityBaseKey] === 'number' &&
    typeof parsedValues.value[priorityAboveKey] !== 'number'
  ) {
    errors[priorityAboveKey] = t('admin.models.editor.validationPriorityAboveThresholdRequired')
  }
}

function appendVirtualThreshold(
  next: UpdatePricingOverridePayload,
  thresholdKey: ModelCatalogPricingKey,
  aboveKey: ModelCatalogPricingKey,
  priorityAboveKey: ModelCatalogPricingKey
) {
  if (!virtualThresholds[thresholdKey] || thresholdKey in next) {
    return
  }
  if (!fieldChanged(aboveKey) && !fieldChanged(priorityAboveKey)) {
    return
  }
  const parsed = parsePricingInput(formValues[thresholdKey] ?? '', 'threshold')
  if (typeof parsed === 'number') {
    next[thresholdKey] = parsed as never
  }
}

function placeholder(unit: ModelCatalogPricingUnit) {
  if (unit === 'threshold') {
    return String(MODEL_CATALOG_DEFAULT_THRESHOLD)
  }
  if (unit === 'token') {
    return t('admin.models.units.perMillionTokens')
  }
  if (unit === 'video_request') {
    return t('admin.models.units.perVideoRequest')
  }
  return t('admin.models.units.perImage')
}

function updateField(key: ModelCatalogPricingKey, value: string) {
  formValues[key] = value
}

function handleSave() {
  if (!hasChanges.value || hasValidationErrors.value) {
    return
  }
  emit('save', payload.value)
}
</script>
