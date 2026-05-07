<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Column } from '@/components/common/types'
import type { AdminUsageLog, UsageModelDisplayMode } from '@/types'
import { adminUsageAPI } from '@/api/admin/usage'
import { formatDateTime } from '@/utils/format'
import { useTokenDisplayMode } from '@/composables/useTokenDisplayMode'
import { useUsageModelDisplayModePreference } from '@/composables/useUsageModelDisplayModePreference'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Pagination from '@/components/common/Pagination.vue'
import UsageModelCell from '@/components/common/UsageModelCell.vue'
import UsageModelDisplayModeToggle from '@/components/common/UsageModelDisplayModeToggle.vue'
import UsageRequestPreviewModal from '@/components/user/usage/UsageRequestPreviewModal.vue'

const { t } = useI18n()
const { formatTokenDisplay } = useTokenDisplayMode()
const {
  usageModelDisplayMode,
  updatingUsageModelDisplayMode,
  setUsageModelDisplayMode,
} = useUsageModelDisplayModePreference()

defineProps<{
  items: AdminUsageLog[]
  total: number
  page: number
  pageSize: number
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:page', value: number): void
  (e: 'update:pageSize', value: number): void
}>()

const previewTarget = ref<{ id: number; request_id?: string | null } | null>(null)
const previewOpen = ref(false)

const columns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.requestDetails.subject.ledger.columns.createdAt') },
  { key: 'request_id', label: t('admin.requestDetails.subject.ledger.columns.requestId') },
  { key: 'api_key_id', label: t('admin.requestDetails.subject.ledger.columns.apiKeyId') },
  { key: 'account_id', label: t('admin.requestDetails.subject.ledger.columns.accountId') },
  { key: 'group_id', label: t('admin.requestDetails.subject.ledger.columns.groupId') },
  { key: 'models', label: t('admin.requestDetails.subject.ledger.columns.models') },
  { key: 'status', label: t('admin.requestDetails.subject.ledger.columns.status') },
  { key: 'total_tokens', label: t('admin.requestDetails.subject.ledger.columns.totalTokens') },
  { key: 'total_cost', label: t('admin.requestDetails.subject.ledger.columns.totalStandardCost') },
  { key: 'actual_cost', label: t('admin.requestDetails.subject.ledger.columns.totalUserCost') },
  { key: 'duration_ms', label: t('admin.requestDetails.subject.ledger.columns.durationMs') },
  { key: 'preview_available', label: t('admin.requestDetails.subject.ledger.columns.previewAvailable') },
  { key: 'actions', label: t('admin.requestDetails.subject.ledger.columns.actions') },
])

function openPreview(row: AdminUsageLog) {
  previewTarget.value = {
    id: row.id,
    request_id: row.request_id,
  }
  previewOpen.value = true
}

function closePreview() {
  previewOpen.value = false
  previewTarget.value = null
}

function formatCurrency(value?: number | null): string {
  return Number(value || 0).toFixed(4)
}

function statusBadgeClass(status: AdminUsageLog['status']): string {
  return status === 'failed'
    ? 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300'
    : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
}

async function handleUsageModelDisplayModeChange(mode: UsageModelDisplayMode) {
  await setUsageModelDisplayMode(mode)
}
</script>

<template>
  <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <div class="mb-4 flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
      <div>
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.requestDetails.subject.ledger.title') }}
        </h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.requestDetails.subject.ledger.description') }}
        </p>
      </div>
      <UsageModelDisplayModeToggle
        class="md:hidden"
        :model-value="usageModelDisplayMode"
        :disabled="updatingUsageModelDisplayMode"
        @update:modelValue="handleUsageModelDisplayModeChange"
      />
    </div>

    <DataTable :columns="columns" :data="items" :loading="loading">
      <template #header-models="{ column }">
        <div class="flex items-center justify-between gap-3">
          <span>{{ column.label }}</span>
          <div class="hidden md:block" @click.stop>
            <UsageModelDisplayModeToggle
              :model-value="usageModelDisplayMode"
              :disabled="updatingUsageModelDisplayMode"
              :show-label="false"
              compact
              @update:modelValue="handleUsageModelDisplayModeChange"
            />
          </div>
        </div>
      </template>

      <template #cell-created_at="{ value }">
        <span class="text-sm text-gray-600 dark:text-gray-300">
          {{ formatDateTime(value) }}
        </span>
      </template>

      <template #cell-request_id="{ row }">
        <div class="max-w-[220px] truncate font-mono text-xs text-gray-700 dark:text-gray-200" :title="row.request_id">
          {{ row.request_id || '-' }}
        </div>
      </template>

      <template #cell-api_key_id="{ value }">
        <span class="font-mono text-xs text-gray-700 dark:text-gray-200">{{ value || '-' }}</span>
      </template>

      <template #cell-account_id="{ value }">
        <span class="font-mono text-xs text-gray-700 dark:text-gray-200">{{ value || '-' }}</span>
      </template>

      <template #cell-group_id="{ value }">
        <span class="font-mono text-xs text-gray-700 dark:text-gray-200">{{ value || '-' }}</span>
      </template>

      <template #cell-models="{ row }">
        <UsageModelCell :row="row" :mode="usageModelDisplayMode" />
      </template>

      <template #cell-status="{ row }">
        <span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium" :class="statusBadgeClass(row.status)">
          {{ row.status }}
        </span>
      </template>

      <template #cell-total_tokens="{ value }">
        <span class="text-sm text-gray-700 dark:text-gray-200">{{ formatTokenDisplay(value || 0) }}</span>
      </template>

      <template #cell-total_cost="{ value }">
        <span class="font-mono text-xs text-gray-700 dark:text-gray-200">${{ formatCurrency(value) }}</span>
      </template>

      <template #cell-actual_cost="{ value }">
        <span class="font-mono text-xs text-emerald-700 dark:text-emerald-300">${{ formatCurrency(value) }}</span>
      </template>

      <template #cell-duration_ms="{ value }">
        <span class="text-sm text-gray-700 dark:text-gray-200">{{ value || 0 }} ms</span>
      </template>

      <template #cell-preview_available="{ value }">
        <span
          class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
          :class="value ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'"
        >
          {{ value ? t('common.yes') : t('common.no') }}
        </span>
      </template>

      <template #cell-actions="{ row }">
        <button class="btn btn-secondary btn-sm" type="button" @click="openPreview(row)">
          {{ t('usage.requestPreview.action') }}
        </button>
      </template>

      <template #empty>
        <EmptyState :message="t('admin.requestDetails.subject.ledger.empty')" />
      </template>
    </DataTable>

    <Pagination
      v-if="total > 0"
      class="mt-4"
      :page="page"
      :total="total"
      :page-size="pageSize"
      @update:page="emit('update:page', $event)"
      @update:pageSize="emit('update:pageSize', $event)"
    />

    <UsageRequestPreviewModal
      :show="previewOpen"
      :usage-log="previewTarget"
      :preview-loader="adminUsageAPI.getRequestPreview"
      @close="closePreview"
    />
  </section>
</template>
