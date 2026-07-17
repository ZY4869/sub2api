<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.auditLogs.title') }}</h1>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.auditLogs.description') }}</p>
        </div>
        <button class="btn btn-secondary" :disabled="cleaning" @click="cleanup">
          <Icon name="trash" size="sm" class="mr-2" />
          {{ cleaning ? t('common.saving') : t('admin.auditLogs.cleanupExpired') }}
        </button>
      </div>

      <div class="card p-4">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
          <input v-model="filters.action" class="input" :placeholder="t('admin.auditLogs.filters.action')" @keyup.enter="applyFilters" />
          <input v-model="filters.target_type" class="input" :placeholder="t('admin.auditLogs.filters.targetType')" @keyup.enter="applyFilters" />
          <input v-model="filters.target_id" class="input" :placeholder="t('admin.auditLogs.filters.targetId')" @keyup.enter="applyFilters" />
          <input v-model="filters.request_id" class="input font-mono text-sm" :placeholder="t('admin.auditLogs.filters.requestId')" @keyup.enter="applyFilters" />
          <input v-model.number="filters.actor_user_id" type="number" min="1" class="input" :placeholder="t('admin.auditLogs.filters.actorUserId')" @keyup.enter="applyFilters" />
          <Select v-model="filters.status" :options="statusOptions" @change="applyFilters" />
        </div>
        <div class="mt-3 flex flex-wrap gap-2">
          <button class="btn btn-primary" @click="applyFilters">{{ t('common.search') }}</button>
          <button class="btn btn-secondary" @click="resetFilters">{{ t('common.reset') }}</button>
          <button class="btn btn-secondary" :disabled="loading" @click="loadLogs">{{ t('common.refresh') }}</button>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div v-if="loading" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</div>
        <div v-else-if="logs.length === 0" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('admin.auditLogs.empty') }}</div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800/60">
              <tr>
                <th v-for="column in columns" :key="column" class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t(`admin.auditLogs.columns.${column}`) }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="log in logs" :key="log.id" class="hover:bg-gray-50 dark:hover:bg-dark-800/40">
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">{{ formatDateTime(log.created_at) }}</td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
                  <div>{{ log.actor_user_id || '-' }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ log.actor_role || '-' }}</div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
                  <span class="rounded bg-gray-100 px-2 py-1 font-mono text-xs dark:bg-dark-800">{{ log.action || '-' }}</span>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
                  <div>{{ log.target_type || '-' }}</div>
                  <div class="font-mono text-xs text-gray-500 dark:text-gray-400">{{ log.target_id || '-' }}</div>
                </td>
                <td class="px-4 py-3 text-sm">
                  <span class="rounded-full px-2 py-1 text-xs font-medium" :class="statusClass(log.status)">{{ statusLabel(log.status) }}</span>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
                  <div class="font-mono text-xs">{{ log.request_id || '-' }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ log.client_ip || '-' }}</div>
                </td>
                <td class="max-w-sm px-4 py-3 text-xs text-gray-700 dark:text-gray-200">
                  <pre class="whitespace-pre-wrap break-words rounded bg-gray-50 p-2 dark:bg-dark-800">{{ formatMetadata(log.metadata) }}</pre>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="pagination.total > 0" class="border-t border-gray-200 p-4 dark:border-dark-700">
          <Pagination :page="pagination.page" :total="pagination.total" :page-size="pagination.page_size" @update:page="handlePageChange" @update:pageSize="handlePageSizeChange" />
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { AuditLog } from '@/api/admin/auditLogs'
import type { PaginatedResponse } from '@/types'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const cleaning = ref(false)
const logs = ref<AuditLog[]>([])
const columns = ['createdAt', 'actor', 'action', 'target', 'status', 'request', 'metadata']

const pagination = reactive<PaginatedResponse<AuditLog>>({ items: [], total: 0, page: 1, page_size: 20, pages: 1 })
const filters = reactive({
  actor_user_id: undefined as number | undefined,
  action: '',
  target_type: '',
  target_id: '',
  status: '',
  request_id: ''
})

const statusOptions = computed(() => ['success', 'denied', 'failure'].reduce(
  (options, status) => [...options, { value: status, label: t(`admin.auditLogs.status.${status}`) }],
  [{ value: '', label: t('admin.auditLogs.filters.allStatuses') }]
))

function query() {
  return {
    page: pagination.page,
    page_size: pagination.page_size,
    actor_user_id: filters.actor_user_id || undefined,
    action: filters.action || undefined,
    target_type: filters.target_type || undefined,
    target_id: filters.target_id || undefined,
    status: filters.status || undefined,
    request_id: filters.request_id || undefined
  }
}

function formatDateTime(value: string) {
  return value ? new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : '-'
}

function statusLabel(status: string) {
  return t(`admin.auditLogs.status.${status || 'success'}`)
}

function statusClass(status: string) {
  if (status === 'denied') return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
  if (status === 'failure') return 'bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300'
  return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
}

function formatMetadata(metadata?: Record<string, unknown>) {
  if (!metadata || Object.keys(metadata).length === 0) return '-'
  return JSON.stringify(metadata, null, 2)
}

async function loadLogs() {
  loading.value = true
  try {
    const data = await adminAPI.auditLogs.list(query())
    logs.value = data.items
    Object.assign(pagination, data)
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.auditLogs.loadFailed'))
  } finally {
    loading.value = false
  }
}

function applyFilters() {
  pagination.page = 1; void loadLogs()
}

function resetFilters() {
  filters.actor_user_id = undefined
  filters.action = ''
  filters.target_type = ''
  filters.target_id = ''
  filters.status = ''
  filters.request_id = ''
  applyFilters()
}

function requestStepUpTotp() {
  return window.prompt(t('admin.auditLogs.stepUpTotpPrompt'))?.trim() || ''
}

async function cleanup() {
  if (!window.confirm(t('admin.auditLogs.cleanupConfirm'))) return
  const stepUpTotp = requestStepUpTotp()
  if (!stepUpTotp) {
    appStore.showError(t('admin.auditLogs.stepUpTotpRequired'))
    return
  }
  cleaning.value = true
  try {
    const result = await adminAPI.auditLogs.cleanupExpired({ stepUpTotp })
    appStore.showSuccess(t('admin.auditLogs.cleanupSuccess', { count: result.deleted }))
    await loadLogs()
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || t('admin.auditLogs.cleanupFailed'))
  } finally {
    cleaning.value = false
  }
}

function handlePageChange(page: number) {
  pagination.page = page; void loadLogs()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadLogs()
}

onMounted(() => {
  void loadLogs()
})
</script>
