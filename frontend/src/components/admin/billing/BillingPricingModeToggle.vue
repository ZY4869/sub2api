<template>
  <div class="inline-flex rounded-2xl border border-gray-200 bg-white p-1 shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <button
      v-for="item in options"
      :key="item.value"
      type="button"
      class="rounded-xl px-4 py-2 text-sm font-medium transition-colors"
      :class="modelValue === item.value ? 'bg-primary-600 text-white' : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
      @click="emit('update:modelValue', item.value)"
    >
      {{ item.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: 'list' | 'grid'
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: 'list' | 'grid'): void
}>()

const options = [
  { value: 'list', label: '列表模式' },
  { value: 'grid', label: '九宫格模式' },
] as const
</script>
