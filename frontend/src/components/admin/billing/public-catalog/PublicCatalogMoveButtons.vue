<template>
  <div class="mt-3 flex flex-wrap justify-end gap-2">
    <button
      type="button"
      class="inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-2 py-1 text-xs font-medium text-slate-500 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40 dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300 dark:hover:bg-dark-700"
      :disabled="moveUpDisabled"
      @click="emit('move-up')"
    >
      <Icon name="arrowUp" size="xs" />
      {{ t('admin.billing.publicCatalog.card.moveUp') }}
    </button>
    <button
      type="button"
      class="inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-2 py-1 text-xs font-medium text-slate-500 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40 dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300 dark:hover:bg-dark-700"
      :disabled="moveDownDisabled"
      @click="emit('move-down')"
    >
      <Icon name="arrowDown" size="xs" />
      {{ t('admin.billing.publicCatalog.card.moveDown') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  moveUpDisabled: boolean
  moveDownDisabled: boolean
}>()

const emit = defineEmits<{
  (e: 'move-up'): void
  (e: 'move-down'): void
}>()

const { t } = useI18n()
</script>
