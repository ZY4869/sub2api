<template>
  <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
    <div class="flex items-center justify-between gap-4">
      <div>
        <label class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.groups.profitControl.title') }}
        </label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.groups.profitControl.description') }}
        </p>
      </div>
      <button
        type="button"
        class="relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors"
        :class="form.profit_control_enabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'"
        @click="toggleProfitControlEnabled"
      >
        <span
          class="inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform"
          :class="form.profit_control_enabled ? 'translate-x-6' : 'translate-x-1'"
        />
      </button>
    </div>

    <div v-if="form.profit_control_enabled" class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
      <div>
        <label class="input-label">{{ t('admin.groups.profitControl.minMargin') }}</label>
        <input
          :value="form.profit_min_margin"
          type="number"
          min="0"
          max="100"
          step="0.01"
          class="input"
          placeholder="0"
          @input="updateNumberField('profit_min_margin', $event)"
        />
        <p class="input-hint">{{ t('admin.groups.profitControl.minMarginHint') }}</p>
      </div>
      <div>
        <label class="input-label">{{ t('admin.groups.profitControl.safetyBuffer') }}</label>
        <input
          :value="form.profit_safety_buffer"
          type="number"
          min="0"
          max="100"
          step="0.01"
          class="input"
          placeholder="0"
          @input="updateNumberField('profit_safety_buffer', $event)"
        />
        <p class="input-hint">{{ t('admin.groups.profitControl.safetyBufferHint') }}</p>
      </div>
      <div class="md:col-span-2 rounded-md bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:bg-dark-800 dark:text-gray-300">
        {{ t('admin.groups.profitControl.preview', { value: previewMultiplier }) }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type ProfitControlForm = {
  rate_multiplier: number | string
  profit_control_enabled: boolean
  profit_min_margin: number | string
  profit_safety_buffer: number | string
}

const props = defineProps<{
  form: ProfitControlForm
  t: (key: string, params?: Record<string, unknown>) => string
}>()

const emit = defineEmits<{
  'update:form': [form: ProfitControlForm]
}>()

const updateForm = (patch: Partial<ProfitControlForm>) => {
  emit('update:form', { ...props.form, ...patch })
}

const toggleProfitControlEnabled = () => {
  updateForm({ profit_control_enabled: !props.form.profit_control_enabled })
}

const inputValue = (event: Event) => (event.target as HTMLInputElement).value

const updateNumberField = (
  field: 'profit_min_margin' | 'profit_safety_buffer',
  event: Event,
) => {
  const value = inputValue(event)
  const parsed = Number.parseFloat(value)
  updateForm({ [field]: Number.isNaN(parsed) ? value : parsed })
}

const previewMultiplier = computed(() => {
  const rate = Number(props.form.rate_multiplier)
  const minMargin = Number(props.form.profit_min_margin)
  const safetyBuffer = Number(props.form.profit_safety_buffer)
  if (!Number.isFinite(rate)) return '0.000'
  const multiplier = rate * (1 + Math.max(0, minMargin || 0) / 100 + Math.max(0, safetyBuffer || 0) / 100)
  return multiplier.toFixed(3)
})
</script>
