<template>
  <div class="space-y-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div>
        <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.models.editor.title') }}</h4>
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.models.editor.description') }}</p>
      </div>
      <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
        <span>{{ t('admin.models.units.perMillionTokens') }}</span>
        <span>/</span>
        <span>{{ t('admin.models.units.perImage') }}</span>
      </div>
    </div>

    <section v-for="group in groupedFields" :key="group.key" class="space-y-3">
      <div>
        <h5 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t(group.labelKey) }}</h5>
      </div>
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
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
          <span v-if="validationErrors[field.key]" class="mt-1 block text-xs text-red-500 dark:text-red-400">
            {{ validationErrors[field.key] }}
          </span>
        </label>
      </div>
    </section>

    <div class="flex flex-wrap justify-end gap-3">
      <button class="btn btn-secondary" :disabled="!detail.override_pricing || saving" @click="emit('reset', detail.model)">
        {{ t('admin.models.resetOverride') }}
      </button>
      <button class="btn btn-primary" :disabled="!hasChanges || hasValidationErrors || saving" @click="handleSave">
        {{ saving ? t('admin.models.saving') : t('admin.models.saveOverride') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelCatalogDetail, UpdatePricingOverridePayload } from '@/api/admin/models'
import {
  MODEL_CATALOG_PRICING_FIELDS,
  MODEL_CATALOG_PRICING_GROUPS,
  parsePricingInput,
  pricingInputValue,
  type ModelCatalogPricingKey,
  type ModelCatalogPricingUnit
} from '@/utils/modelCatalogPricing'

const props = defineProps<{
  detail: ModelCatalogDetail
  saving: boolean
}>()

const emit = defineEmits<{
  (e: 'save', payload: UpdatePricingOverridePayload): void
  (e: 'reset', model: string): void
}>()

const { t } = useI18n()
const formValues = reactive<Record<string, string>>({})
const initialValues = reactive<Record<string, string>>({})

const groupedFields = computed(() =>
  MODEL_CATALOG_PRICING_GROUPS.map((group) => ({
    ...group,
    fields: MODEL_CATALOG_PRICING_FIELDS.filter((field) => field.group === group.key)
  }))
)

watch(
  () => props.detail,
  (detail) => {
    for (const field of MODEL_CATALOG_PRICING_FIELDS) {
      const value = detail.override_pricing?.[field.key] ?? detail.effective_pricing?.[field.key]
      const formatted = pricingInputValue(value, field.unit)
      formValues[field.key] = formatted
      initialValues[field.key] = formatted
    }
  },
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

const hasChanges = computed(() =>
  MODEL_CATALOG_PRICING_FIELDS.some((field) => formValues[field.key] !== initialValues[field.key])
)

const validationErrors = computed<Partial<Record<ModelCatalogPricingKey, string>>>(() => {
  const errors: Partial<Record<ModelCatalogPricingKey, string>> = {}
  for (const field of MODEL_CATALOG_PRICING_FIELDS) {
    const rawValue = (formValues[field.key] ?? '').trim()
    if (!rawValue) {
      if (initialValues[field.key] !== '' && rawValue !== initialValues[field.key]) {
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

  validateTier(errors, 'input_token_threshold', 'input_cost_per_token_above_threshold', 'input_cost_per_token_priority', 'input_cost_per_token_priority_above_threshold')
  validateTier(errors, 'output_token_threshold', 'output_cost_per_token_above_threshold', 'output_cost_per_token_priority', 'output_cost_per_token_priority_above_threshold')
  return errors
})

const hasValidationErrors = computed(() => Object.keys(validationErrors.value).length > 0)

function validateTier(
  errors: Partial<Record<ModelCatalogPricingKey, string>>,
  thresholdKey: ModelCatalogPricingKey,
  aboveKey: ModelCatalogPricingKey,
  priorityBaseKey: ModelCatalogPricingKey,
  priorityAboveKey: ModelCatalogPricingKey
) {
  if (typeof parsedValues.value[thresholdKey] !== 'number') {
    return
  }
  if (typeof parsedValues.value[aboveKey] !== 'number') {
    errors[aboveKey] = t('admin.models.editor.validationAboveThresholdRequired')
  }
  if (typeof parsedValues.value[priorityBaseKey] === 'number' && typeof parsedValues.value[priorityAboveKey] !== 'number') {
    errors[priorityAboveKey] = t('admin.models.editor.validationPriorityAboveThresholdRequired')
  }
}

function placeholder(unit: ModelCatalogPricingUnit) {
  if (unit === 'threshold') {
    return t('admin.models.units.tokens')
  }
  return unit === 'token' ? t('admin.models.units.perMillionTokens') : t('admin.models.units.perImage')
}

function updateField(key: ModelCatalogPricingKey, value: string) {
  formValues[key] = value
}

function handleSave() {
  if (hasValidationErrors.value) {
    return
  }

  const payload: UpdatePricingOverridePayload = { model: props.detail.model }
  for (const field of MODEL_CATALOG_PRICING_FIELDS) {
    if (formValues[field.key] === initialValues[field.key]) {
      continue
    }
    const parsed = parsePricingInput(formValues[field.key], field.unit)
    if (typeof parsed === 'number') {
      payload[field.key] = parsed as never
    }
  }
  emit('save', payload)
}
</script>
