<template>
  <div class="flex flex-wrap items-center gap-2">
    <span
      v-if="showLabel"
      class="text-xs font-medium text-gray-500 dark:text-gray-400"
    >
      {{ labelText }}:
    </span>
    <div
      class="inline-flex border border-gray-200 bg-white p-0.5 shadow-sm dark:border-dark-600 dark:bg-dark-800"
      :class="compact ? 'rounded-md' : 'rounded-lg'"
    >
      <button
        v-for="option in options"
        :key="option.value"
        type="button"
        class="font-medium transition-colors"
        :class="[
          compact ? 'rounded px-2 py-0.5 text-[11px]' : 'rounded-md px-2.5 py-1 text-xs',
          mode === option.value
            ? 'bg-primary-500 text-white'
            : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'
        ]"
        :disabled="disabled"
        @click="$emit('update:modelValue', option.value)"
      >
        {{ option.label }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UsageModelDisplayMode } from '@/types'
import { normalizeUsageModelDisplayMode } from '@/utils/usageModelPresentation'

const props = defineProps<{
  modelValue?: UsageModelDisplayMode | null
  disabled?: boolean
  showLabel?: boolean
  compact?: boolean
  labelText?: string
}>()

defineEmits<{
  (e: 'update:modelValue', value: UsageModelDisplayMode): void
}>()

const { t } = useI18n()

const mode = computed(() => normalizeUsageModelDisplayMode(props.modelValue))
const showLabel = computed(() => props.showLabel !== false)
const compact = computed(() => props.compact === true)
const labelText = computed(() => props.labelText || t('usage.modelDisplay'))
const options = computed(() => [
  { value: 'model_only' as const, label: t('usage.modelDisplayModelOnly') },
  { value: 'display_only' as const, label: t('usage.modelDisplayDisplayOnly') },
  { value: 'display_and_model' as const, label: t('usage.modelDisplayBoth') },
])
</script>
