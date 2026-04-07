<template>
  <AppLayout>
    <TablePageLayout prefer-page-scroll>
      <template #filters>
        <div class="space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="flex flex-wrap items-center gap-3">
              <SearchInput
                :model-value="String(params.search || '')"
                :placeholder="t('admin.accounts.blacklist.searchPlaceholder')"
                class="w-full sm:w-64"
                @update:model-value="handleSearchUpdate"
                @search="reload"
              />
              <Select :model-value="params.platform" class="w-40" :options="platformOptions" @update:model-value="handlePlatformUpdate" @change="reload" />
              <Select :model-value="params.group" class="w-48" :options="groupOptions" @update:model-value="handleGroupUpdate" @change="reload" />
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <button type="button" class="btn btn-primary" :disabled="selectedIds.length === 0 || submitting" @click="handleBatchRetest">
                {{ t('admin.accounts.blacklist.batchRetest', { count: selectedIds.length }) }}
              </button>
              <button type="button" class="btn btn-danger" :disabled="selectedIds.length === 0 || submitting" @click="handleBatchDelete">
                {{ t('admin.accounts.blacklist.batchDelete', { count: selectedIds.length }) }}
              </button>
              <button type="button" class="btn btn-danger" :disabled="totalBlacklistedCount === 0 || submitting" @click="handleDeleteAllBlacklisted">
                {{ t('admin.accounts.blacklist.deleteAll', { count: totalBlacklistedCount }) }}
              </button>
            </div>
          </div>
          <div class="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-gray-500 dark:text-gray-400">
            <span>{{ t('admin.accounts.blacklist.totalCountLabel') }} {{ totalBlacklistedCount }}</span>
            <span>{{ t('admin.accounts.blacklist.currentResultLabel') }} {{ pagination.total }}</span>
          </div>
        </div>
      </template>

      <template #table>
        <div v-if="selectedIds.length > 0" class="mb-4 flex items-center justify-between rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:bg-rose-900/20 dark:text-rose-200">
          <span>{{ t('admin.accounts.blacklist.selected', { count: selectedIds.length }) }}</span>
          <button type="button" class="btn btn-secondary btn-sm" @click="clearSelection">
            {{ t('common.clear') }}
          </button>
        </div>

        <DataTable :columns="columns" :data="accounts" :loading="loading" row-key="id">
          <template #header-select>
            <input type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" :checked="allVisibleSelected" @change="toggleVisible(($event.target as HTMLInputElement).checked)" />
          </template>

          <template #cell-select="{ row }">
            <input type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" :checked="selectedIds.includes(row.id)" @change="toggle(row.id)" />
          </template>

          <template #cell-name="{ row }">
            <div class="flex flex-col">
              <span class="font-medium text-gray-900 dark:text-white">{{ row.name }}</span>
              <span class="max-w-md truncate text-xs text-gray-500 dark:text-gray-400" :title="row.lifecycle_reason_message || row.error_message || ''">
                {{ row.lifecycle_reason_message || row.error_message || '-' }}
              </span>
            </div>
          </template>

          <template #cell-platform="{ row }">
            <PlatformTypeBadge
              :platform="row.platform"
              :gateway-protocol="row.gateway_protocol"
              :type="row.type"
              :plan-type="row.credentials?.plan_type"
            />
          </template>

          <template #cell-groups="{ row }">
            <AccountGroupsCell :groups="row.groups" :max-display="3" />
          </template>

          <template #cell-blacklisted_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) || '-' }}</span>
          </template>

          <template #cell-blacklist_purge_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) || '-' }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-2">
              <button type="button" class="btn btn-secondary btn-sm" :disabled="submitting" @click="handleSingleRetest(row.id)">
                {{ t('admin.accounts.blacklist.retestSingle') }}
              </button>
              <button type="button" class="btn btn-danger btn-sm" :disabled="submitting" @click="handleDelete(row.id, row.name)">
                {{ t('admin.accounts.blacklist.deleteNow') }}
              </button>
            </div>
          </template>
        </DataTable>

        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>
    <BlacklistRetestModal
      :show="showRetestModal"
      :accounts="retestTargets"
      :submitting="submitting"
      @close="handleRetestModalClose"
      @confirm="handleRetestConfirm"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { BlacklistRetestRequestPayload } from '@/api/admin/accounts'
import { useTableLoader } from '@/composables/useTableLoader'
import { useTableSelection } from '@/composables/useTableSelection'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import type { Column } from '@/components/common/types'
import type { Account, AdminGroup } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import AccountGroupsCell from '@/components/account/AccountGroupsCell.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import BlacklistRetestModal from '@/components/admin/account/BlacklistRetestModal.vue'

const { t } = useI18n()
const appStore = useAppStore()
const groups = ref<AdminGroup[]>([])
const submitting = ref(false)
const totalBlacklistedCount = ref(0)
const showRetestModal = ref(false)
const retestTargets = ref<Account[]>([])

const {
  items: accounts,
  loading,
  params,
  pagination,
  reload,
  handlePageChange,
  handlePageSizeChange
} = useTableLoader<Account, { platform: string; group: string; search: string; lifecycle: string }>({
  fetchFn: adminAPI.accounts.list,
  initialParams: { platform: '', group: '', search: '', lifecycle: 'blacklisted' }
})

const { selectedIds, allVisibleSelected, clear: clearSelection, toggle, toggleVisible } = useTableSelection<Account>({
  rows: accounts,
  getId: (account) => account.id
})

const columns = computed<Column[]>(() => [
  { key: 'select', label: '' },
  { key: 'name', label: t('admin.accounts.columns.name'), sortable: true },
  { key: 'platform', label: t('admin.accounts.columns.platformType') },
  { key: 'groups', label: t('admin.accounts.columns.groups') },
  { key: 'blacklisted_at', label: t('admin.accounts.blacklist.blacklistedAt'), sortable: true },
  { key: 'blacklist_purge_at', label: t('admin.accounts.blacklist.purgeAt'), sortable: true },
  { key: 'actions', label: t('admin.accounts.columns.actions') }
])

const platformOptions = computed(() => [
  { value: '', label: t('admin.accounts.allPlatforms') },
  { value: 'anthropic', label: t('admin.accounts.platforms.anthropic') },
  { value: 'kiro', label: t('admin.accounts.platforms.kiro') },
  { value: 'openai', label: t('admin.accounts.platforms.openai') },
  { value: 'copilot', label: t('admin.accounts.platforms.copilot') },
  { value: 'grok', label: t('admin.accounts.platforms.grok') },
  { value: 'protocol_gateway', label: t('admin.accounts.platforms.protocol_gateway') },
  { value: 'gemini', label: t('admin.accounts.platforms.gemini') },
  { value: 'antigravity', label: t('admin.accounts.platforms.antigravity') }
])

const groupOptions = computed(() => [
  { value: '', label: t('admin.accounts.allGroups') },
  { value: 'ungrouped', label: t('admin.accounts.ungroupedGroup') },
  ...groups.value.map((group) => ({ value: String(group.id), label: group.name }))
])

const handleSearchUpdate = (value: string) => {
  params.search = value
}

const handlePlatformUpdate = (value: string | number | boolean | null) => {
  params.platform = String(value || '')
}

const handleGroupUpdate = (value: string | number | boolean | null) => {
  params.group = String(value || '')
}

const refreshTotalBlacklistedCount = async () => {
  const response = await adminAPI.accounts.list(1, 1, { lifecycle: 'blacklisted' })
  totalBlacklistedCount.value = response.total || 0
}

const refreshBlacklistView = async () => {
  clearSelection()
  await Promise.all([reload(), refreshTotalBlacklistedCount()])
}

const summarizeRetest = (results: Awaited<ReturnType<typeof adminAPI.accounts.retestBlacklistedAccounts>>['results']) => {
  const restored = results.filter((item) => item.restored).length
  const failed = results.length - restored
  if (restored > 0 && failed === 0) {
    appStore.showSuccess(t('admin.accounts.blacklist.retestSuccess', { count: restored }))
    return
  }
  if (restored > 0) {
    appStore.showWarning(t('admin.accounts.blacklist.retestPartial', { restored, failed }))
    return
  }
  appStore.showError(t('admin.accounts.blacklist.retestFailed'))
}

const closeRetestModal = () => {
  showRetestModal.value = false
  retestTargets.value = []
}

const runRetest = async (payload: BlacklistRetestRequestPayload) => {
  if (payload.account_ids.length === 0 || submitting.value) return
  submitting.value = true
  try {
    const response = await adminAPI.accounts.retestBlacklistedAccounts(payload)
    summarizeRetest(response.results)
    closeRetestModal()
    await refreshBlacklistView()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.blacklist.retestFailed'))
  } finally {
    submitting.value = false
  }
}

const openRetestModal = (accountIds: number[]) => {
  if (accountIds.length === 0 || submitting.value) {
    return
  }
  const accountSet = new Set(accountIds)
  const targets = accounts.value.filter((account) => accountSet.has(account.id))
  if (targets.length === 0) {
    return
  }
  retestTargets.value = targets
  showRetestModal.value = true
}

const handleSingleRetest = (accountId: number) => {
  openRetestModal([accountId])
}

const handleBatchRetest = () => {
  openRetestModal([...selectedIds.value])
}

const handleRetestModalClose = () => {
  if (submitting.value) {
    return
  }
  closeRetestModal()
}

const handleRetestConfirm = async (payload: BlacklistRetestRequestPayload) => {
  await runRetest(payload)
}

const handleDelete = async (accountId: number, accountName: string) => {
  if (!window.confirm(t('admin.accounts.blacklist.deleteConfirm', { name: accountName }))) {
    return
  }
  submitting.value = true
  try {
    await adminAPI.accounts.delete(accountId)
    appStore.showSuccess(t('admin.accounts.blacklist.deleteSuccess'))
    await refreshBlacklistView()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.blacklist.deleteFailed'))
  } finally {
    submitting.value = false
  }
}

const summarizeBatchDelete = (
  result: Awaited<ReturnType<typeof adminAPI.accounts.batchDeleteBlacklistedAccounts>>,
  mode: 'selected' | 'all'
) => {
  const successCount = result.deleted_count
  const failedCount = result.failed_count
  if (failedCount === 0) {
    appStore.showSuccess(
      t(mode === 'all' ? 'admin.accounts.blacklist.deleteAllSuccess' : 'admin.accounts.blacklist.batchDeleteSuccess', {
        count: successCount
      })
    )
    return
  }
  if (successCount > 0) {
    appStore.showWarning(
      t(mode === 'all' ? 'admin.accounts.blacklist.deleteAllPartial' : 'admin.accounts.blacklist.batchDeletePartial', {
        success: successCount,
        failed: failedCount
      })
    )
    return
  }
  appStore.showError(t(mode === 'all' ? 'admin.accounts.blacklist.deleteAllFailed' : 'admin.accounts.blacklist.batchDeleteFailed'))
}

const runBatchDelete = async (
  payload: { ids?: number[]; delete_all?: boolean },
  mode: 'selected' | 'all'
) => {
  if (submitting.value) return
  if (mode === 'selected' && (!payload.ids || payload.ids.length === 0)) return
  if (mode === 'all' && totalBlacklistedCount.value === 0) return
  submitting.value = true
  try {
    const result = await adminAPI.accounts.batchDeleteBlacklistedAccounts(payload)
    summarizeBatchDelete(result, mode)
    await refreshBlacklistView()
  } catch (error: any) {
    appStore.showError(
      error?.message ||
        t(mode === 'all' ? 'admin.accounts.blacklist.deleteAllFailed' : 'admin.accounts.blacklist.batchDeleteFailed')
    )
  } finally {
    submitting.value = false
  }
}

const handleBatchDelete = async () => {
  const accountIds = [...selectedIds.value]
  if (accountIds.length === 0) {
    return
  }
  if (!window.confirm(t('admin.accounts.blacklist.batchDeleteConfirm', { count: accountIds.length }))) {
    return
  }
  await runBatchDelete({ ids: accountIds }, 'selected')
}

const handleDeleteAllBlacklisted = async () => {
  if (totalBlacklistedCount.value === 0) {
    return
  }
  if (!window.confirm(t('admin.accounts.blacklist.deleteAllConfirm', { count: totalBlacklistedCount.value }))) {
    return
  }
  await runBatchDelete({ delete_all: true }, 'all')
}

onMounted(() => {
  refreshBlacklistView().catch((error) => {
    console.error('Failed to load blacklisted accounts:', error)
  })
  adminAPI.groups.getAll()
    .then((allGroups) => {
      groups.value = allGroups
    })
    .catch((error) => {
      console.error('Failed to load groups for blacklisted accounts:', error)
    })
})
</script>
