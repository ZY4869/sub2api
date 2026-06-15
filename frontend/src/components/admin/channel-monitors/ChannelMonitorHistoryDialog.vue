<template>
  <BaseDialog
    :show="show"
    :title="title"
    width="extra-wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <div class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="min-w-0">
          <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
            {{ monitor?.name || '-' }}
          </div>
          <div class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
            {{ monitorTarget }}
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <div class="flex items-center gap-2">
            <span class="text-sm text-gray-600 dark:text-gray-300">{{ t('admin.channelMonitors.history.limit') }}</span>
            <input v-model.number="limit" type="number" min="1" max="200" class="input w-24" />
          </div>

          <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadHistories">
            {{ t('common.refresh') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="loading || !monitor" @click="runNow">
            <Icon v-if="running" name="refresh" size="md" class="mr-2 animate-spin" />
            {{ t('admin.channelMonitors.actions.run') }}
          </button>
        </div>
      </div>

      <DataTable :columns="columns" :data="histories" :loading="loading" row-key="id">
        <template #cell-status="{ value }">
          <span class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium" :class="statusClass(value)">
            {{ statusLabel(value) }}
          </span>
        </template>

        <template #cell-account="{ row }">
          <span class="text-xs text-gray-600 dark:text-gray-300">
            {{ row.account_name_snapshot || row.account_id || '-' }}
          </span>
        </template>

        <template #cell-created_at="{ value }">
          <span class="text-sm text-gray-600 dark:text-gray-300">{{ formatDateTime(value) }}</span>
        </template>

        <template #cell-latency_ms="{ value }">
          <span class="font-mono text-xs text-gray-700 dark:text-gray-200">{{ value }}ms</span>
        </template>

        <template #cell-message="{ row }">
          <span
            class="block max-w-[520px] truncate text-xs text-gray-600 dark:text-gray-300"
            :title="row.error_message || row.response_text || ''"
          >
            {{ row.error_message || row.response_text || '-' }}
          </span>
        </template>

        <template #empty>
          <EmptyState
            :title="t('admin.channelMonitors.history.emptyTitle')"
            :description="t('admin.channelMonitors.history.emptyDescription')"
          />
        </template>
      </DataTable>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { adminAPI } from '@/api/admin'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import type { AdminChannelMonitor, AdminChannelMonitorHistory } from '@/api/admin/channelMonitors'
import {
  getChannelMonitorStatusClass,
  getChannelMonitorStatusLabel
} from '@/utils/channelMonitorPresentation'

const { t } = useI18n()
const appStore = useAppStore()

const props = defineProps<{
  show: boolean
  monitor: AdminChannelMonitor | null
  initialHistories: AdminChannelMonitorHistory[] | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const title = computed(() => t('admin.channelMonitors.history.title'))
const monitorTarget = computed(() => {
  if (!props.monitor) return '-'
  if (props.monitor.probe_mode === 'account_pool') {
    const count = Array.isArray(props.monitor.account_ids) ? props.monitor.account_ids.length : 0
    return t('admin.channelMonitors.fields.accountPoolTarget', { count })
  }
  return props.monitor.endpoint || '-'
})

const histories = ref<AdminChannelMonitorHistory[]>([])
const loading = ref(false)
const running = ref(false)
const limit = ref(50)

const columns = computed(() => [
  { key: 'created_at', label: t('admin.channelMonitors.history.fields.createdAt') },
  { key: 'account', label: t('admin.channelMonitors.history.fields.account') },
  { key: 'model_id', label: t('admin.channelMonitors.history.fields.model') },
  { key: 'status', label: t('admin.channelMonitors.history.fields.status') },
  { key: 'http_status', label: t('admin.channelMonitors.history.fields.httpStatus') },
  { key: 'latency_ms', label: t('admin.channelMonitors.history.fields.latency') },
  { key: 'message', label: t('admin.channelMonitors.history.fields.message') }
])

function formatDateTime(value?: string): string {
  if (!value) return '-'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return String(value)
  return d.toLocaleString()
}

function statusClass(status?: string): string {
  return getChannelMonitorStatusClass(status)
}

function statusLabel(status?: string): string {
  return getChannelMonitorStatusLabel(status)
}

async function loadHistories() {
  if (!props.monitor) return
  loading.value = true
  try {
    histories.value = await adminAPI.channelMonitors.listMonitorHistories(props.monitor.id, limit.value)
  } catch (err) {
    appStore.showError(t('admin.channelMonitors.messages.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function runNow() {
  if (!props.monitor) return
  running.value = true
  try {
    histories.value = await adminAPI.channelMonitors.runMonitor(props.monitor.id)
    appStore.showSuccess(t('admin.channelMonitors.messages.ran'))
  } catch (err) {
    appStore.showError(t('admin.channelMonitors.messages.runFailed'))
  } finally {
    running.value = false
  }
}

watch(
  () => [props.show, props.monitor?.id, props.initialHistories] as const,
  ([show, _id, initial]) => {
    if (!show) return
    if (initial && initial.length > 0) {
      histories.value = initial
      return
    }
    loadHistories()
  },
  { immediate: true }
)
</script>
