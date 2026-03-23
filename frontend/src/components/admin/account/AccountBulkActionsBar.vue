<template>
  <div
    v-if="selectedIds.length > 0"
    class="mb-4 flex items-center justify-between rounded-lg bg-primary-50 p-3 dark:bg-primary-900/20"
  >
    <div class="flex flex-wrap items-center gap-2">
      <span class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkActions.selected', { count: selectedIds.length }) }}
      </span>
      <button
        type="button"
        class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
        @click="emit('select-page')"
      >
        {{ t('admin.accounts.bulkActions.selectCurrentPage') }}
      </button>
      <span class="text-gray-300 dark:text-primary-800">|</span>
      <button
        type="button"
        class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
        @click="emit('clear')"
      >
        {{ t('admin.accounts.bulkActions.clear') }}
      </button>
    </div>

    <div class="flex flex-wrap gap-2">
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        :disabled="!canArchive"
        :title="archiveButtonTitle"
        @click="emit('archive')"
      >
        {{ t('admin.accounts.bulkActions.archive') }}
      </button>
      <button type="button" class="btn btn-danger btn-sm" @click="emit('delete')">
        {{ t('admin.accounts.bulkActions.delete') }}
      </button>
      <button type="button" class="btn btn-secondary btn-sm" @click="emit('reset-status')">
        {{ t('admin.accounts.bulkActions.resetStatus') }}
      </button>
      <button type="button" class="btn btn-secondary btn-sm" @click="emit('refresh-token')">
        {{ t('admin.accounts.bulkActions.refreshToken') }}
      </button>
      <button type="button" class="btn btn-success btn-sm" @click="emit('toggle-schedulable', true)">
        {{ t('admin.accounts.bulkActions.enableScheduling') }}
      </button>
      <button type="button" class="btn btn-warning btn-sm" @click="emit('toggle-schedulable', false)">
        {{ t('admin.accounts.bulkActions.disableScheduling') }}
      </button>
      <button type="button" class="btn btn-primary btn-sm" @click="emit('edit')">
        {{ t('admin.accounts.bulkActions.edit') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountPlatform } from '@/types'

const props = defineProps<{
  selectedIds: number[]
  selectedPlatforms: AccountPlatform[]
}>()

const emit = defineEmits<{
  archive: []
  delete: []
  edit: []
  clear: []
  'select-page': []
  'toggle-schedulable': [value: boolean]
  'reset-status': []
  'refresh-token': []
}>()

const { t } = useI18n()

const canArchive = computed(() => props.selectedPlatforms.length === 1)

const archiveButtonTitle = computed(() =>
  canArchive.value ? '' : t('admin.accounts.bulkActions.archiveMixedPlatformDisabled')
)
</script>
