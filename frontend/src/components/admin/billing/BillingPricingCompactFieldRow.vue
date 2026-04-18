<template>
  <div class="rounded-2xl border border-gray-200 bg-white px-3 py-3 dark:border-dark-700 dark:bg-dark-900/60">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex min-w-0 flex-wrap items-center gap-2 sm:flex-1">
        <label
          v-if="selectable"
          class="inline-flex shrink-0 items-center gap-2 text-xs text-gray-600 dark:text-gray-300"
          :data-testid="`field-select-${fieldId}`"
        >
          <input
            type="checkbox"
            class="h-4 w-4 rounded border-gray-300 text-primary-600"
            :checked="selected"
            :disabled="disabled"
            @change="emit('toggle-select')"
          />
          选中
        </label>
        <span class="min-w-0 text-sm font-medium text-gray-900 dark:text-white">{{ label }}</span>
        <span
          class="inline-flex shrink-0 rounded-full bg-gray-100 px-2 py-1 text-[11px] font-medium text-gray-500 dark:bg-dark-700 dark:text-gray-300"
          :data-testid="`pricing-field-unit-${fieldId}`"
        >
          {{ unitLabel }}
        </span>
      </div>

      <div class="flex w-full flex-col gap-2 sm:w-[220px] sm:min-w-[220px]">
        <input
          class="input w-full"
          type="text"
          inputmode="decimal"
          autocomplete="off"
          :value="draft"
          :disabled="disabled"
          :data-testid="`pricing-field-${fieldId}`"
          @focus="focused = true"
          @input="handleInput"
          @blur="handleBlur"
          @keydown.enter="($event.target as HTMLInputElement).blur()"
        />
        <slot name="detail" />
      </div>
    </div>

    <p
      v-if="secondaryText"
      class="mt-2 text-xs text-gray-500 dark:text-gray-400"
      :data-testid="`pricing-field-secondary-${fieldId}`"
    >
      {{ secondaryText }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import {
  formatBillingPricingEditableNumber,
  parseBillingPricingDecimalInput,
} from './pricingCurrency'

const props = withDefaults(defineProps<{
  fieldId: string
  label: string
  unitLabel: string
  value?: number
  secondaryText?: string
  selectable?: boolean
  selected?: boolean
  disabled?: boolean
}>(), {
  value: undefined,
  secondaryText: '',
  selectable: false,
  selected: false,
  disabled: false,
})

const emit = defineEmits<{
  (e: 'toggle-select'): void
  (e: 'update:value', value: string): void
}>()

const draft = ref('')
const focused = ref(false)

watch(
  () => props.value,
  () => {
    if (!focused.value) {
      draft.value = formatBillingPricingEditableNumber(props.value)
    }
  },
  { immediate: true },
)

function handleInput(event: Event) {
  const raw = (event.target as HTMLInputElement).value
  draft.value = raw

  if (!raw.trim()) {
    emit('update:value', '')
    return
  }

  if (parseBillingPricingDecimalInput(raw) != null) {
    emit('update:value', raw)
  }
}

function handleBlur() {
  focused.value = false

  if (!draft.value.trim()) {
    draft.value = ''
    emit('update:value', '')
    return
  }

  const parsed = parseBillingPricingDecimalInput(draft.value)
  if (parsed == null) {
    draft.value = formatBillingPricingEditableNumber(props.value)
    return
  }

  const normalized = formatBillingPricingEditableNumber(parsed)
  draft.value = normalized
  emit('update:value', normalized)
}
</script>
