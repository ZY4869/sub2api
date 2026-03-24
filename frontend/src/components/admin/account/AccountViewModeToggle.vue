<template>
  <div class="flex items-center gap-1 rounded-2xl border border-gray-200 bg-white p-1 shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      class="rounded-xl px-4 py-2 text-sm font-medium transition-colors"
      :class="modelValue === option.value ? 'bg-primary-600 text-white shadow-sm' : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white'"
      @click="emit('update:modelValue', option.value)"
    >
      {{ option.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountViewMode } from '@/types'

const props = defineProps<{
  modelValue: AccountViewMode
}>()

const emit = defineEmits<{
  'update:modelValue': [value: AccountViewMode]
}>()

const { t } = useI18n()

const options = computed(() => [
  { value: 'table' as const, label: t('admin.accounts.viewMode.table') },
  { value: 'card' as const, label: t('admin.accounts.viewMode.card') }
])

void props
</script>
