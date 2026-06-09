<template>
  <div
    v-if="total > 0"
    class="mb-4 flex flex-col gap-3 rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800 md:flex-row md:items-center md:justify-between"
    data-account-filtered-bulk-edit-bar="true"
  >
    <div class="min-w-0">
      <div class="text-sm font-medium text-gray-900 dark:text-white">
        {{ t('admin.accounts.bulkEdit.currentCategoryTargets', { count: total }) }}
      </div>
      <label
        class="mt-2 inline-flex items-center gap-2 text-xs text-gray-600 dark:text-dark-300"
        :title="excludeGroupedDisabled ? t('admin.accounts.bulkEdit.excludeGroupedSpecificGroupDisabled') : ''"
      >
        <input
          type="checkbox"
          class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-60"
          :checked="excludeGrouped"
          :disabled="excludeGroupedDisabled"
          data-account-filtered-bulk-edit-exclude-grouped="true"
          @change="handleExcludeGroupedChange"
        />
        <span>{{ t('admin.accounts.bulkEdit.excludeGrouped') }}</span>
      </label>
    </div>

    <button
      type="button"
      class="btn btn-secondary justify-center md:justify-start"
      :disabled="loading"
      data-account-filtered-bulk-edit-button="true"
      @click="emit('edit')"
    >
      <Icon name="edit" size="sm" />
      <span>{{ t('admin.accounts.bulkEdit.editCurrentCategory') }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  total: number
  loading: boolean
  excludeGrouped: boolean
  excludeGroupedDisabled: boolean
}>()

const emit = defineEmits<{
  'update:excludeGrouped': [value: boolean]
  edit: []
}>()

const { t } = useI18n()

const handleExcludeGroupedChange = (event: Event) => {
  emit('update:excludeGrouped', (event.target as HTMLInputElement).checked)
}
</script>
