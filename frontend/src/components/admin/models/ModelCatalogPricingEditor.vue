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

    <div class="grid gap-4 md:grid-cols-2">
      <label v-for="field in MODEL_CATALOG_PRICING_FIELDS" :key="field.key" class="block">
        <span class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-200">{{ t(field.labelKey) }}</span>
        <input
          :value="formValues[field.key]"
          type="number"
          min="0"
          step="0.000001"
          class="input"
          :placeholder="field.unit === 'token' ? t('admin.models.units.perMillionTokens') : t('admin.models.units.perImage')"
          @input="updateField(field.key, String(($event.target as HTMLInputElement).value))"
        />
        <span v-if="validationErrors[field.key]" class="mt-1 block text-xs text-red-500 dark:text-red-400">
          {{ validationErrors[field.key] }}
        </span>
      </label>
    </div>

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
  parsePricingInput,
  pricingInputValue,
  type ModelCatalogPricingKey
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

const hasChanges = computed(() =>
  MODEL_CATALOG_PRICING_FIELDS.some((field) => formValues[field.key] !== initialValues[field.key])
)

const validationErrors = computed<Partial<Record<ModelCatalogPricingKey, string>>>(() => {
  const errors: Partial<Record<ModelCatalogPricingKey, string>> = {}
  for (const field of MODEL_CATALOG_PRICING_FIELDS) {
    if (formValues[field.key] === initialValues[field.key]) {
      continue
    }
    const rawValue = formValues[field.key].trim()
    if (rawValue === '') {
      errors[field.key] = t('admin.models.editor.validationRequired')
      continue
    }
    const parsed = Number(rawValue)
    if (!Number.isFinite(parsed) || parsed < 0) {
      errors[field.key] = t('admin.models.editor.validationNonNegative')
    }
  }
  return errors
})

const hasValidationErrors = computed(() => Object.keys(validationErrors.value).length > 0)

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
      payload[field.key] = parsed as NonNullable<UpdatePricingOverridePayload[typeof field.key]>
    }
  }

  emit('save', payload)
}
</script>
