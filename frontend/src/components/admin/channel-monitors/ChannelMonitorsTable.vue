<template>
  <DataTable :columns="columns" :data="items" :loading="loading" row-key="id">
    <template #cell-name="{ row }">
      <div class="min-w-0">
        <div class="truncate font-medium text-gray-900 dark:text-white">{{ row.name }}</div>
        <div v-if="row.template_id != null" class="mt-1 flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
          <span class="whitespace-nowrap">{{ t('admin.channelMonitors.fields.template') }}:</span>
          <span class="truncate font-mono">{{ templateLabel(row.template_id) }}</span>
        </div>
      </div>
    </template>

    <template #cell-provider="{ row }">
      <div class="flex items-center gap-2">
        <ModelPlatformIcon :platform="row.provider" size="sm" />
        <span class="text-sm text-gray-700 dark:text-gray-200">{{ row.provider }}</span>
      </div>
    </template>

    <template #cell-endpoint="{ value }">
      <span class="block max-w-[340px] truncate font-mono text-xs text-gray-600 dark:text-gray-300" :title="String(value || '')">
        {{ value || '-' }}
      </span>
    </template>

    <template #cell-enabled="{ row }">
      <Toggle
        :modelValue="row.enabled"
        @update:modelValue="(v) => $emit('toggleEnabled', row, v)"
      />
    </template>

    <template #cell-interval_seconds="{ value }">
      <span class="text-sm text-gray-700 dark:text-gray-200">{{ value }}s</span>
    </template>

    <template #cell-primary_model_id="{ row }">
      <div class="flex min-w-0 items-center gap-2">
        <ModelIcon
          :model="row.primary_model_id"
          :provider="row.provider"
          :display-name="row.primary_model_id"
          size="14px"
        />
        <span class="min-w-0 truncate font-mono text-xs text-gray-700 dark:text-gray-200">{{ row.primary_model_id }}</span>
      </div>
    </template>

    <template #cell-next_run_at="{ value }">
      <span class="text-sm text-gray-600 dark:text-gray-300">{{ formatDateTime(value) }}</span>
    </template>

    <template #cell-api_key_status="{ row }">
      <span
        class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium"
        :class="apiKeyBadgeClass(row)"
      >
        {{ apiKeyBadgeText(row) }}
      </span>
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
          @click="$emit('run', row)"
        >
          <Icon name="play" size="sm" />
          <span class="text-xs">{{ t('admin.channelMonitors.actions.run') }}</span>
        </button>
        <button
          type="button"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
          @click="$emit('history', row)"
        >
          <Icon name="clock" size="sm" />
          <span class="text-xs">{{ t('admin.channelMonitors.actions.history') }}</span>
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
        :title="t('admin.channelMonitors.empty.monitorsTitle')"
        :description="t('admin.channelMonitors.empty.monitorsDescription')"
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
import Toggle from '@/components/common/Toggle.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import type { AdminChannelMonitor, AdminChannelMonitorTemplate } from '@/api/admin/channelMonitors'

const { t } = useI18n()

const props = defineProps<{
  items: AdminChannelMonitor[]
  templates: AdminChannelMonitorTemplate[]
  loading: boolean
}>()

defineEmits<{
  (e: 'edit', monitor: AdminChannelMonitor): void
  (e: 'delete', monitor: AdminChannelMonitor): void
  (e: 'run', monitor: AdminChannelMonitor): void
  (e: 'history', monitor: AdminChannelMonitor): void
  (e: 'toggleEnabled', monitor: AdminChannelMonitor, enabled: boolean): void
}>()

const templateMap = computed(() => new Map(props.templates.map(tpl => [tpl.id, tpl])))

const columns = computed(() => [
  { key: 'name', label: t('admin.channelMonitors.fields.name') },
  { key: 'provider', label: t('admin.channelMonitors.fields.provider') },
  { key: 'endpoint', label: t('admin.channelMonitors.fields.endpoint') },
  { key: 'enabled', label: t('admin.channelMonitors.fields.enabled') },
  { key: 'interval_seconds', label: t('admin.channelMonitors.fields.intervalSeconds') },
  { key: 'primary_model_id', label: t('admin.channelMonitors.fields.primaryModel') },
  { key: 'next_run_at', label: t('admin.channelMonitors.fields.nextRunAt') },
  { key: 'api_key_status', label: t('admin.channelMonitors.fields.apiKey') },
  { key: 'actions', label: t('common.actions') }
])

function templateLabel(id: number): string {
  const tpl = templateMap.value.get(id)
  return tpl ? `${tpl.name}` : String(id)
}

function formatDateTime(value?: string): string {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
  return d.toLocaleString()
}

function apiKeyBadgeText(m: AdminChannelMonitor): string {
  if (m.api_key_decrypt_failed) return t('admin.channelMonitors.status.decryptFailed')
  if (m.api_key_configured) return t('admin.channelMonitors.status.configured')
  return t('admin.channelMonitors.status.missing')
}

function apiKeyBadgeClass(m: AdminChannelMonitor): string {
  if (m.api_key_decrypt_failed) return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
  if (m.api_key_configured) return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
  return 'bg-gray-100 text-gray-700 dark:bg-gray-900/30 dark:text-gray-400'
}
</script>

