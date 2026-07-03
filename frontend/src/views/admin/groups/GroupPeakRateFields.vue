<template>
  <div
    v-if="form.subscription_type === 'subscription'"
    class="border-t border-gray-200 pt-4 dark:border-dark-600"
  >
    <div class="flex items-center justify-between gap-4">
      <div>
        <label class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.groups.peakRate.title') }}
        </label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.groups.peakRate.description') }}
        </p>
      </div>
      <button
        type="button"
        class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
        :class="form.peak_rate_enabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600'"
        @click="togglePeakRateEnabled"
      >
        <span
          class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
          :class="form.peak_rate_enabled ? 'translate-x-6' : 'translate-x-1'"
        />
      </button>
    </div>

    <div v-if="form.peak_rate_enabled" class="mt-4 grid gap-3 sm:grid-cols-3">
      <div>
        <label class="input-label">{{ t('admin.groups.peakRate.start') }}</label>
        <input
          :value="form.peak_start"
          type="time"
          class="input"
          step="60"
          required
          @input="updateTextField('peak_start', $event)"
        />
      </div>
      <div>
        <label class="input-label">{{ t('admin.groups.peakRate.end') }}</label>
        <input
          :value="form.peak_end"
          type="time"
          class="input"
          step="60"
          required
          @input="updateTextField('peak_end', $event)"
        />
      </div>
      <div>
        <label class="input-label">{{ t('admin.groups.peakRate.multiplier') }}</label>
        <input
          :value="form.peak_rate_multiplier"
          type="number"
          class="input"
          min="0"
          step="0.001"
          required
          @input="updateMultiplierField"
        />
      </div>
    </div>

    <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
      {{
        timezone
          ? t('admin.groups.peakRate.serverTime', { timezone, offset: utcOffset || '-' })
          : t('admin.groups.peakRate.multiplierHint')
      }}
    </p>
  </div>
</template>

<script setup lang="ts">
type PeakRateForm = {
  subscription_type: string
  peak_rate_enabled: boolean
  peak_start: string
  peak_end: string
  peak_rate_multiplier: number | string
}

const props = defineProps<{
  t: (key: string, values?: Record<string, unknown>) => string
  form: PeakRateForm
  timezone?: string
  utcOffset?: string
}>()

const emit = defineEmits<{
  'update:form': [form: PeakRateForm]
}>()

const updateForm = (patch: Partial<PeakRateForm>) => {
  emit('update:form', { ...props.form, ...patch })
}

const togglePeakRateEnabled = () => {
  updateForm({ peak_rate_enabled: !props.form.peak_rate_enabled })
}

const inputValue = (event: Event) => (event.target as HTMLInputElement).value

const updateTextField = (field: 'peak_start' | 'peak_end', event: Event) => {
  updateForm({ [field]: inputValue(event).trim() })
}

const updateMultiplierField = (event: Event) => {
  const value = inputValue(event)
  const parsed = Number.parseFloat(value)
  updateForm({ peak_rate_multiplier: Number.isNaN(parsed) ? value : parsed })
}
</script>
