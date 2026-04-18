<template>
  <article class="rounded-2xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-800">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <h4 class="text-sm font-semibold text-gray-900 dark:text-white">倍率定价</h4>
        <p class="mt-1 text-xs text-gray-600 dark:text-gray-300">
          在出售基准价上按倍率动态计算最终售价。
        </p>
      </div>
      <Toggle
        :model-value="enabled"
        :data-testid="'pricing-multiplier-toggle'"
        @update:model-value="emit('update:enabled', $event)"
      />
    </div>

    <div v-if="enabled" class="mt-4 space-y-3">
      <div class="flex flex-wrap gap-2">
        <button
          type="button"
          class="rounded-full px-3 py-1.5 text-xs font-medium transition"
          :class="mode === 'shared' ? activeModeClass : inactiveModeClass"
          data-testid="pricing-multiplier-mode-shared"
          :disabled="disabled"
          @click="emit('update:mode', 'shared')"
        >
          统一倍率
        </button>
        <button
          type="button"
          class="rounded-full px-3 py-1.5 text-xs font-medium transition"
          :class="mode === 'item' ? activeModeClass : inactiveModeClass"
          data-testid="pricing-multiplier-mode-item"
          :disabled="disabled"
          @click="emit('update:mode', 'item')"
        >
          分项倍率
        </button>
      </div>

      <div class="flex flex-wrap items-end gap-3">
        <label class="min-w-[180px] flex-1 text-xs text-gray-600 dark:text-gray-300">
          <span class="mb-1.5 block">统一倍率</span>
          <input
            class="input w-full"
            type="text"
            inputmode="decimal"
            autocomplete="off"
            :value="sharedDraft"
            :disabled="disabled"
            data-testid="pricing-shared-multiplier"
            @input="handleInput"
            @blur="handleBlur"
          />
        </label>
        <button
          type="button"
          class="btn btn-secondary btn-sm"
          data-testid="pricing-apply-shared-multiplier"
          :disabled="disabled || sharedMultiplier == null || fieldCount === 0"
          @click="emit('apply-shared')"
        >
          应用到所有已填写项
        </button>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { BillingPricingMultiplierMode } from '@/api/admin/billing'
import Toggle from '@/components/common/Toggle.vue'
import {
  formatBillingPricingEditableNumber,
  parseBillingPricingDecimalInput,
} from './pricingCurrency'

const props = withDefaults(defineProps<{
  enabled: boolean
  mode: BillingPricingMultiplierMode
  sharedMultiplier?: number
  fieldCount: number
  disabled?: boolean
}>(), {
  sharedMultiplier: undefined,
  disabled: false,
})

const emit = defineEmits<{
  (e: 'update:enabled', value: boolean): void
  (e: 'update:mode', value: BillingPricingMultiplierMode): void
  (e: 'update:sharedMultiplier', value: number | undefined): void
  (e: 'apply-shared'): void
}>()

const activeModeClass = 'bg-primary-600 text-white'
const inactiveModeClass = 'bg-white text-gray-600 hover:bg-gray-100 dark:bg-dark-900 dark:text-gray-300 dark:hover:bg-dark-700'

const sharedDraft = ref('')

watch(
  () => props.sharedMultiplier,
  () => {
    sharedDraft.value = formatBillingPricingEditableNumber(props.sharedMultiplier)
  },
  { immediate: true },
)

function handleInput(event: Event) {
  const raw = (event.target as HTMLInputElement).value
  sharedDraft.value = raw
  if (!raw.trim()) {
    emit('update:sharedMultiplier', undefined)
    return
  }
  const parsed = parseBillingPricingDecimalInput(raw)
  if (parsed != null) {
    emit('update:sharedMultiplier', parsed)
  }
}

function handleBlur() {
  if (!sharedDraft.value.trim()) {
    sharedDraft.value = ''
    emit('update:sharedMultiplier', undefined)
    return
  }
  const parsed = parseBillingPricingDecimalInput(sharedDraft.value)
  if (parsed == null) {
    sharedDraft.value = formatBillingPricingEditableNumber(props.sharedMultiplier)
    return
  }
  sharedDraft.value = formatBillingPricingEditableNumber(parsed)
  emit('update:sharedMultiplier', parsed)
}
</script>
