<template>
  <DataTable :columns="columns" :data="channels" :loading="loading">
    <template #cell-name="{ value }">
      <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
    </template>

    <template #cell-description="{ value }">
      <span class="text-sm text-gray-600 dark:text-gray-400">{{ value || '-' }}</span>
    </template>

    <template #cell-status="{ row }">
      <Toggle
        :modelValue="row.status === 'active'"
        @update:modelValue="emit('toggle-status', row)"
      />
    </template>

    <template #cell-group_count="{ row }">
      <span
        class="inline-flex items-center rounded bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-800 dark:bg-dark-600 dark:text-gray-300"
      >
        {{ (row.group_ids || []).length }}
        {{ t('admin.channels.groupsUnit', 'groups') }}
      </span>
    </template>

    <template #cell-pricing_count="{ row }">
      <span
        class="inline-flex items-center rounded bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-800 dark:bg-dark-600 dark:text-gray-300"
      >
        {{ (row.model_pricing || []).length }}
        {{ t('admin.channels.pricingUnit', 'pricing rules') }}
      </span>
    </template>

    <template #cell-created_at="{ value }">
      <span class="text-sm text-gray-600 dark:text-gray-400">
        {{ formatDate(value) }}
      </span>
    </template>

    <template #cell-actions="{ row }">
      <div class="flex items-center gap-1">
        <button
          @click="emit('edit', row)"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
        >
          <Icon name="edit" size="sm" />
          <span class="text-xs">{{ t('common.edit', 'Edit') }}</span>
        </button>
        <button
          @click="emit('delete', row)"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
        >
          <Icon name="trash" size="sm" />
          <span class="text-xs">{{ t('common.delete', 'Delete') }}</span>
        </button>
      </div>
    </template>

    <template #empty>
      <EmptyState
        :title="t('admin.channels.noChannelsYet', 'No Channels Yet')"
        :description="t('admin.channels.createFirstChannel', 'Create your first channel to manage model pricing')"
        :action-text="t('admin.channels.createChannel', 'Create Channel')"
        @action="emit('create')"
      />
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Channel } from '@/api/admin/channels'
import type { Column } from '@/components/common/types'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  columns: Column[]
  channels: Channel[]
  loading: boolean
}>()

const emit = defineEmits<{
  'toggle-status': [channel: Channel]
  edit: [channel: Channel]
  delete: [channel: Channel]
  create: []
}>()

const { t } = useI18n()

function formatDate(value: string): string {
  if (!value) return '-'
  return new Date(value).toLocaleDateString()
}
</script>
