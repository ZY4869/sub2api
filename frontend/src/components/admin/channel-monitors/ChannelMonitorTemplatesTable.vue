<template>
  <DataTable :columns="columns" :data="items" :loading="loading" row-key="id">
    <template #cell-name="{ row }">
      <div class="min-w-0">
        <div class="truncate font-medium text-gray-900 dark:text-white">{{ row.name }}</div>
        <div v-if="row.description" class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
          {{ row.description }}
        </div>
      </div>
    </template>

    <template #cell-provider="{ row }">
      <div class="flex items-center gap-2">
        <ModelPlatformIcon :platform="row.provider" size="sm" />
        <span class="text-sm text-gray-700 dark:text-gray-200">{{ row.provider }}</span>
      </div>
    </template>

    <template #cell-body_override_mode="{ value }">
      <span class="font-mono text-xs text-gray-700 dark:text-gray-200">{{ value || '-' }}</span>
    </template>

    <template #cell-updated_at="{ value }">
      <span class="text-sm text-gray-600 dark:text-gray-300">{{ formatDateTime(value) }}</span>
    </template>

    <template #cell-actions="{ row }">
      <div class="flex items-center gap-1">
        <button
          type="button"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
          @click="$emit('edit', row)"
        >
          <Icon name="edit" size="sm" />
          <span class="text-xs">{{ t('common.edit') }}</span>
        </button>
        <button
          type="button"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
          @click="$emit('apply', row)"
        >
          <Icon name="check" size="sm" />
          <span class="text-xs">{{ t('admin.channelMonitors.actions.apply') }}</span>
        </button>
        <button
          type="button"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
          @click="$emit('associated', row)"
        >
          <Icon name="link" size="sm" />
          <span class="text-xs">{{ t('admin.channelMonitors.actions.associated') }}</span>
        </button>
        <button
          type="button"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
          @click="$emit('delete', row)"
        >
          <Icon name="trash" size="sm" />
          <span class="text-xs">{{ t('common.delete') }}</span>
        </button>
      </div>
    </template>

    <template #empty>
      <EmptyState
        :title="t('admin.channelMonitors.empty.templatesTitle')"
        :description="t('admin.channelMonitors.empty.templatesDescription')"
      />
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import type { AdminChannelMonitorTemplate } from '@/api/admin/channelMonitors'

const { t } = useI18n()

defineProps<{
  items: AdminChannelMonitorTemplate[]
  loading: boolean
}>()

defineEmits<{
  (e: 'edit', tpl: AdminChannelMonitorTemplate): void
  (e: 'delete', tpl: AdminChannelMonitorTemplate): void
  (e: 'apply', tpl: AdminChannelMonitorTemplate): void
  (e: 'associated', tpl: AdminChannelMonitorTemplate): void
}>()

const columns = computed(() => [
  { key: 'name', label: t('admin.channelMonitors.templateFields.name') },
  { key: 'provider', label: t('admin.channelMonitors.templateFields.provider') },
  { key: 'body_override_mode', label: t('admin.channelMonitors.templateFields.bodyOverrideMode') },
  { key: 'updated_at', label: t('admin.channelMonitors.templateFields.updatedAt') },
  { key: 'actions', label: t('common.actions') }
])

function formatDateTime(value?: string): string {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
  return d.toLocaleString()
}
</script>

