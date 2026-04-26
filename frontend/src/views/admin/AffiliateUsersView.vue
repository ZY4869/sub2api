<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex-1 sm:max-w-80">
            <input
              v-model="lookupQuery"
              type="text"
              class="input"
              :placeholder="t('admin.affiliates.lookupPlaceholder')"
              @keydown.enter.prevent="handleLookup"
            />
          </div>

          <div class="flex flex-wrap items-center gap-3">
            <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
              <input v-model="filters.has_custom_code" type="checkbox" />
              {{ t('admin.affiliates.filters.customCode') }}
            </label>
            <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
              <input v-model="filters.has_custom_rate" type="checkbox" />
              {{ t('admin.affiliates.filters.customRate') }}
            </label>
            <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
              <input v-model="filters.has_inviter" type="checkbox" />
              {{ t('admin.affiliates.filters.hasInviter') }}
            </label>
          </div>

          <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="loading"
              :title="t('common.refresh')"
              @click="loadList()"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button type="button" class="btn btn-secondary" @click="openBatchRateDialog()">
              <Icon name="edit" size="md" class="mr-1" />
              {{ t('admin.affiliates.batchRate') }}
            </button>
          </div>
        </div>

        <div v-if="lookupResults.length" class="mt-3 rounded-xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <div class="mb-2 text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.affiliates.lookupResults') }}
          </div>
          <div class="grid grid-cols-1 gap-2 md:grid-cols-2">
            <button
              v-for="item in lookupResults"
              :key="item.user_id"
              type="button"
              class="flex items-center justify-between rounded-lg border border-gray-100 px-3 py-2 text-left transition-colors hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700"
              @click="openEditDialog(item)"
            >
              <div class="min-w-0">
                <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
                  #{{ item.user_id }} · {{ item.email }}
                </div>
                <div class="truncate text-xs text-gray-500 dark:text-dark-400">
                  {{ item.aff_code }}
                </div>
              </div>
              <Icon name="chevronRight" size="sm" class="text-gray-400" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="rows" :loading="loading">
          <template #cell-aff_code="{ value, row }">
            <div class="flex items-center gap-2">
              <code class="font-mono text-xs text-gray-900 dark:text-gray-100">{{ value }}</code>
              <span
                v-if="row.custom_aff_code"
                class="badge bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
              >
                {{ t('admin.affiliates.custom') }}
              </span>
              <button
                type="button"
                class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                :title="t('common.copy')"
                @click="copyText(String(value))"
              >
                <Icon name="copy" size="sm" />
              </button>
            </div>
          </template>

          <template #cell-custom_rebate_rate_percent="{ value }">
            <span class="text-sm text-gray-700 dark:text-gray-200">
              {{ value === undefined || value === null ? '-' : `${Number(value).toFixed(2)}%` }}
            </span>
          </template>

          <template #cell-rebate_balance="{ value }">
            <span class="text-sm font-medium text-gray-900 dark:text-white">
              {{ formatCurrency(Number(value || 0)) }}
            </span>
          </template>

          <template #cell-rebate_frozen_balance="{ value }">
            <span class="text-sm text-gray-700 dark:text-gray-200">
              {{ formatCurrency(Number(value || 0)) }}
            </span>
          </template>

          <template #cell-lifetime_rebate="{ value }">
            <span class="text-sm text-gray-700 dark:text-gray-200">
              {{ formatCurrency(Number(value || 0)) }}
            </span>
          </template>

          <template #cell-updated_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center space-x-1">
              <button
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-600 dark:hover:text-gray-300"
                :title="t('common.edit')"
                @click="openEditDialog(row)"
              >
                <Icon name="edit" size="sm" />
              </button>
              <button
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                :title="t('admin.affiliates.reset')"
                @click="handleReset(row)"
              >
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="showEditDialog"
      :title="t('admin.affiliates.editTitle')"
      width="normal"
      @close="closeEditDialog"
    >
      <div v-if="editTarget" class="space-y-4">
        <div class="rounded-lg border border-gray-100 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-700/40">
          <div class="text-sm font-medium text-gray-900 dark:text-white">
            #{{ editTarget.user_id }} · {{ editTarget.email }}
          </div>
          <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            {{ t('admin.affiliates.currentCode') }}: <code class="font-mono">{{ editTarget.aff_code }}</code>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.affiliates.fields.affCode') }}</label>
          <input v-model="editForm.aff_code" type="text" class="input font-mono uppercase" />
          <p class="mt-1.5 text-xs text-gray-500 dark:text-dark-400">
            {{ t('admin.affiliates.affCodeHint') }}
          </p>
        </div>

        <div>
          <label class="input-label">{{ t('admin.affiliates.fields.customRate') }}</label>
          <input v-model="editForm.custom_rate" type="number" step="0.01" min="0" max="100" class="input" />
          <p class="mt-1.5 text-xs text-gray-500 dark:text-dark-400">
            {{ t('admin.affiliates.customRateHint') }}
          </p>
        </div>
      </div>

      <template #footer>
        <div class="flex w-full items-center justify-between gap-3">
          <button type="button" class="btn btn-secondary" :disabled="editing" @click="closeEditDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="editing" @click="handleSaveEdit">
            <svg
              v-if="editing"
              class="-ml-1 mr-2 h-4 w-4 animate-spin text-white"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{ t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="showBatchDialog"
      :title="t('admin.affiliates.batchRate')"
      width="normal"
      @close="closeBatchDialog"
    >
      <form id="affiliate-batch-form" class="space-y-4" @submit.prevent="handleBatchRate">
        <div>
          <label class="input-label">{{ t('admin.affiliates.batchUserIds') }}</label>
          <textarea v-model="batchForm.user_ids" class="input min-h-[120px] font-mono text-xs" />
          <p class="mt-1.5 text-xs text-gray-500 dark:text-dark-400">
            {{ t('admin.affiliates.batchUserIdsHint') }}
          </p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.affiliates.fields.customRate') }}</label>
          <input v-model="batchForm.custom_rate" type="number" step="0.01" min="0" max="100" class="input" />
        </div>
      </form>

      <template #footer>
        <div class="flex w-full items-center justify-between gap-3">
          <button type="button" class="btn btn-secondary" :disabled="batching" @click="closeBatchDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="affiliate-batch-form" class="btn btn-primary" :disabled="batching">
            <svg
              v-if="batching"
              class="-ml-1 mr-2 h-4 w-4 animate-spin text-white"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{ t('admin.affiliates.apply') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { adminAPI } from '@/api/admin'
import type { AffiliateAdminUser } from '@/api/admin/affiliates'
import type { Column } from '@/components/common/types'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { useAppStore } from '@/stores'
import { buildAuthErrorMessage } from '@/utils/authError'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard: copyText } = useClipboard()

const loading = ref(false)
const rows = ref<AffiliateAdminUser[]>([])
const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

const filters = reactive({
  has_custom_code: false,
  has_custom_rate: false,
  has_inviter: false
})

const lookupQuery = ref('')
const lookupResults = ref<AffiliateAdminUser[]>([])

const columns = computed<Column[]>(() => [
  { key: 'user_id', label: t('admin.affiliates.fields.userId') },
  { key: 'email', label: t('admin.affiliates.fields.email') },
  { key: 'aff_code', label: t('admin.affiliates.fields.affCode') },
  { key: 'custom_rebate_rate_percent', label: t('admin.affiliates.fields.customRate') },
  { key: 'inviter_user_id', label: t('admin.affiliates.fields.inviter') },
  { key: 'invitee_count', label: t('admin.affiliates.fields.invitees') },
  { key: 'rebate_balance', label: t('admin.affiliates.fields.available') },
  { key: 'rebate_frozen_balance', label: t('admin.affiliates.fields.frozen') },
  { key: 'lifetime_rebate', label: t('admin.affiliates.fields.lifetime') },
  { key: 'updated_at', label: t('admin.affiliates.fields.updatedAt') },
  { key: 'actions', label: t('common.actions') }
])

async function loadList() {
  loading.value = true
  try {
    const data = await adminAPI.affiliates.listAffiliateUsers({
      page: pagination.page,
      page_size: pagination.page_size,
      ...(filters.has_custom_code ? { has_custom_code: true } : {}),
      ...(filters.has_custom_rate ? { has_custom_rate: true } : {}),
      ...(filters.has_inviter ? { has_inviter: true } : {})
    })
    rows.value = data.items || []
    pagination.total = data.total || 0
  } catch (err: any) {
    appStore.showError(buildAuthErrorMessage(err, { fallback: t('common.unknownError') }))
  } finally {
    loading.value = false
  }
}

async function handleLookup() {
  const q = lookupQuery.value.trim()
  if (!q) {
    lookupResults.value = []
    return
  }
  try {
    lookupResults.value = await adminAPI.affiliates.lookupAffiliateUsers(q, 20)
  } catch (err: any) {
    appStore.showError(buildAuthErrorMessage(err, { fallback: t('common.unknownError') }))
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  loadList()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  loadList()
}

watch(
  () => ({ ...filters }),
  () => {
    pagination.page = 1
    loadList()
  }
)

const showEditDialog = ref(false)
const editing = ref(false)
const editTarget = ref<AffiliateAdminUser | null>(null)
const editForm = reactive({
  aff_code: '',
  custom_rate: '' as string | number
})

function openEditDialog(target: AffiliateAdminUser) {
  editTarget.value = target
  editForm.aff_code = target.aff_code || ''
  editForm.custom_rate =
    target.custom_rebate_rate_percent === undefined || target.custom_rebate_rate_percent === null
      ? ''
      : Number(target.custom_rebate_rate_percent).toFixed(2)
  showEditDialog.value = true
}

function closeEditDialog() {
  showEditDialog.value = false
  editTarget.value = null
}

function normalizeAffiliateCode(input: string) {
  return input.trim().toUpperCase().replace(/[-\s]/g, '')
}

async function handleSaveEdit() {
  if (!editTarget.value) return

  const original = editTarget.value
  const payload: Record<string, any> = {}

  const newCode = normalizeAffiliateCode(String(editForm.aff_code || ''))
  const oldCode = normalizeAffiliateCode(String(original.aff_code || ''))
  if (newCode !== oldCode) {
    payload.aff_code = String(editForm.aff_code || '')
  }

  const rawRate = String(editForm.custom_rate ?? '').trim()
  const oldRate = original.custom_rebate_rate_percent
  if (rawRate === '') {
    if (oldRate !== undefined && oldRate !== null) {
      payload.custom_rebate_rate_percent = null
    }
  } else {
    const v = Number(rawRate)
    if (!Number.isFinite(v)) {
      appStore.showError(t('admin.affiliates.invalidRate'))
      return
    }
    const clamped = Math.max(0, Math.min(100, v))
    if (oldRate === undefined || oldRate === null || Math.abs(clamped - oldRate) > 1e-9) {
      payload.custom_rebate_rate_percent = clamped
    }
  }

  if (Object.keys(payload).length === 0) {
    appStore.showSuccess(t('common.noChanges'))
    closeEditDialog()
    return
  }

  editing.value = true
  try {
    await adminAPI.affiliates.updateAffiliateUserCustom(original.user_id, payload)
    appStore.showSuccess(t('common.saved'))
    closeEditDialog()
    await loadList()
    // Refresh lookup results as well, so modal reopen reflects latest.
    lookupResults.value = lookupResults.value.map((item) =>
      item.user_id === original.user_id ? { ...item, ...payload } : item
    )
  } catch (err: any) {
    appStore.showError(buildAuthErrorMessage(err, { fallback: t('common.unknownError') }))
  } finally {
    editing.value = false
  }
}

async function handleReset(row: AffiliateAdminUser) {
  try {
    await adminAPI.affiliates.resetAffiliateUserCustom(row.user_id)
    appStore.showSuccess(t('admin.affiliates.resetSuccess'))
    await loadList()
  } catch (err: any) {
    appStore.showError(buildAuthErrorMessage(err, { fallback: t('common.unknownError') }))
  }
}

const showBatchDialog = ref(false)
const batching = ref(false)
const batchForm = reactive({
  user_ids: '',
  custom_rate: '' as string | number
})

function openBatchRateDialog() {
  showBatchDialog.value = true
}

function closeBatchDialog() {
  showBatchDialog.value = false
  batchForm.user_ids = ''
  batchForm.custom_rate = ''
}

function parseUserIDs(input: string): number[] {
  const tokens = input
    .split(/[\s,;\n\r\t]+/g)
    .map((t) => t.trim())
    .filter(Boolean)
  const ids = tokens
    .map((t) => Number(t))
    .filter((n) => Number.isFinite(n) && Number.isInteger(n) && n > 0)
  return Array.from(new Set(ids))
}

async function handleBatchRate() {
  const ids = parseUserIDs(batchForm.user_ids)
  if (!ids.length) {
    appStore.showError(t('admin.affiliates.invalidUserIds'))
    return
  }
  const v = Number(String(batchForm.custom_rate ?? '').trim())
  if (!Number.isFinite(v)) {
    appStore.showError(t('admin.affiliates.invalidRate'))
    return
  }
  const rate = Math.max(0, Math.min(100, v))

  batching.value = true
  try {
    const res = await adminAPI.affiliates.batchUpdateAffiliateUserRates(ids, rate)
    appStore.showSuccess(t('admin.affiliates.batchSuccess', { n: res.updated }))
    closeBatchDialog()
    await loadList()
  } catch (err: any) {
    appStore.showError(buildAuthErrorMessage(err, { fallback: t('common.unknownError') }))
  } finally {
    batching.value = false
  }
}

onMounted(() => {
  loadList()
})
</script>
