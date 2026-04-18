<template>
  <div
    v-if="multiplierEnabled && displayBaseValue != null"
    class="rounded-xl border border-dashed border-primary-200 bg-primary-50/70 px-3 py-2 text-xs text-primary-700 dark:border-primary-500/30 dark:bg-primary-500/10 dark:text-primary-200"
    :data-testid="`pricing-multiplier-inline-${fieldId}`"
  >
    <div class="flex flex-wrap items-center gap-2">
      <template v-if="isItemMode">
        <input
          class="input h-8 min-w-[92px] flex-1 bg-white text-xs dark:bg-dark-900"
          type="text"
          inputmode="decimal"
          autocomplete="off"
          :value="draft"
          :disabled="disabled"
          :data-testid="`pricing-item-multiplier-${fieldId}`"
          @input="handleInput"
          @blur="handleBlur"
        />
      </template>
      <template v-else>
        <span class="rounded-full bg-white/80 px-2 py-1 font-medium text-primary-700 dark:bg-dark-900/70 dark:text-primary-200">
          {{ formattedMultiplier }}x
        </span>
      </template>
      <span class="font-medium">{{ formattedBase }} × {{ formattedMultiplier }} = {{ formattedEffective }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  formatBillingPricingEditableNumber,
  parseBillingPricingDecimalInput,
} from './pricingCurrency'

const props = withDefaults(defineProps<{
  fieldId: string
  multiplierEnabled: boolean
  multiplierMode?: string
  sharedMultiplier?: number
  itemMultiplier?: number
  displayBaseValue?: number
  displayEffectiveValue?: number
  disabled?: boolean
}>(), {
  multiplierMode: 'shared',
  sharedMultiplier: undefined,
  itemMultiplier: undefined,
  displayBaseValue: undefined,
  displayEffectiveValue: undefined,
  disabled: false,
})

const emit = defineEmits<{
  (e: 'update:itemMultiplier', value: number | undefined): void
}>()

const draft = ref('')

const isItemMode = computed(() => props.multiplierMode === 'item')
const currentMultiplier = computed(() => (
  isItemMode.value
    ? (props.itemMultiplier ?? 1)
    : (props.sharedMultiplier ?? 1)
))
const formattedBase = computed(() => formatBillingPricingEditableNumber(props.displayBaseValue))
const formattedMultiplier = computed(() => formatBillingPricingEditableNumber(currentMultiplier.value))
const formattedEffective = computed(() => formatBillingPricingEditableNumber(props.displayEffectiveValue))

watch(
  () => props.itemMultiplier,
  () => {
    draft.value = formatBillingPricingEditableNumber(props.itemMultiplier ?? 1)
  },
  { immediate: true },
)

function handleInput(event: Event) {
  const raw = (event.target as HTMLInputElement).value
  draft.value = raw
  if (!raw.trim()) {
    emit('update:itemMultiplier', undefined)
    return
  }
  const parsed = parseBillingPricingDecimalInput(raw)
  if (parsed != null) {
    emit('update:itemMultiplier', parsed)
  }
}

function handleBlur() {
  if (!draft.value.trim()) {
    draft.value = ''
    emit('update:itemMultiplier', undefined)
    return
  }
  const parsed = parseBillingPricingDecimalInput(draft.value)
  if (parsed == null) {
    draft.value = formatBillingPricingEditableNumber(props.itemMultiplier ?? 1)
    return
  }
  draft.value = formatBillingPricingEditableNumber(parsed)
  emit('update:itemMultiplier', parsed)
}
</script>
