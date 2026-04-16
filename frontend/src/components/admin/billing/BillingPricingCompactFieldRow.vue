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

      <input
        class="input w-full sm:w-[220px] sm:min-w-[220px]"
        type="number"
        :step="step"
        :value="value ?? ''"
        :data-testid="`pricing-field-${fieldId}`"
        @input="emit('update:value', ($event.target as HTMLInputElement).value)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  fieldId: string
  label: string
  unitLabel: string
  value?: number
  selectable?: boolean
  selected?: boolean
  step?: string
}>(), {
  value: undefined,
  selectable: false,
  selected: false,
  step: '0.0000001',
})

void props

const emit = defineEmits<{
  (e: 'toggle-select'): void
  (e: 'update:value', value: string): void
}>()
</script>
