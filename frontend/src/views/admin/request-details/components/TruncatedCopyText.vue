<script setup lang="ts">
import { computed, useAttrs } from 'vue'
import { useClipboard } from '@/composables/useClipboard'

defineOptions({
  inheritAttrs: false
})

const props = withDefaults(defineProps<{
  value?: string | number | null
  displayText?: string | null
  copyValue?: string | null
  titleText?: string | null
  placeholder?: string
  mono?: boolean
  copySuccessMessage?: string
}>(), {
  placeholder: '-',
  mono: false,
  displayText: '',
  copyValue: '',
  titleText: ''
})

const attrs = useAttrs()
const { copyToClipboard } = useClipboard()

const normalizedValue = computed(() => {
  if (props.value === null || typeof props.value === 'undefined') return ''
  return String(props.value).trim()
})

const normalizedCopyValue = computed(() => {
  const preferred = String(props.copyValue || '').trim()
  return preferred || normalizedValue.value
})

const displayValue = computed(() => {
  const preferred = String(props.displayText || '').trim()
  return preferred || normalizedValue.value || props.placeholder
})

const tooltipText = computed(() => {
  const preferred = String(props.titleText || '').trim()
  return preferred || normalizedCopyValue.value || displayValue.value
})

const hasValue = computed(() => normalizedCopyValue.value.length > 0)

async function handleCopy() {
  if (!hasValue.value) return
  await copyToClipboard(normalizedCopyValue.value, props.copySuccessMessage)
}
</script>

<template>
  <span
    v-if="!hasValue"
    v-bind="attrs"
    :title="tooltipText"
    class="truncate text-gray-400 dark:text-gray-500"
  >
    {{ displayValue }}
  </span>

  <button
    v-else
    v-bind="attrs"
    :title="tooltipText"
    class="truncate text-left transition-colors hover:text-blue-600 dark:hover:text-blue-300"
    :class="{ 'font-mono': mono }"
    type="button"
    @click.stop="handleCopy"
  >
    {{ displayValue }}
  </button>
</template>
