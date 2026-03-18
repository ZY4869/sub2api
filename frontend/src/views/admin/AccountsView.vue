<template>
  <AppLayout>
    <TablePageLayout prefer-page-scroll>
      <template #filters>
        <AccountsViewToolbar
          :loading="loading"
          :search-query="String(params.search || '')"
          :filters="params"
          :groups="groups"
          :has-pending-list-sync="hasPendingListSync"
          :selected-count="selIds.length"
          :auto-refresh-enabled="autoRefreshEnabled"
          :auto-refresh-countdown="autoRefreshCountdown"
          :auto-refresh-intervals="autoRefreshIntervals"
          :auto-refresh-interval-seconds="autoRefreshIntervalSeconds"
          :toggleable-columns="toggleableColumns"
          @update:filters="handleFilterUpdate"
          @update:search-query="handleSearchQueryUpdate"
          @change="debouncedReload"
          @refresh="handleManualRefresh"
          @sync="showSync = true"
          @create="showCreate = true"
          @import-data="showImportData = true"
          @export-data="openExportDataDialog"
          @show-error-passthrough="showErrorPassthrough = true"
          @sync-pending-list="handleSyncPendingListChanges"
          @set-auto-refresh-enabled="setAutoRefreshEnabled"
          @set-auto-refresh-interval="handleAutoRefreshIntervalChange"
          @toggle-column="toggleColumn"
        />
      </template>
      <template #table>
        <AccountBulkActionsBar :selected-ids="selIds" @delete="handleBulkDelete" @reset-status="handleBulkResetStatus" @refresh-token="handleBulkRefreshToken" @edit="showBulkEdit = true" @clear="clearSelection" @select-page="selectPage" @toggle-schedulable="handleBulkToggleSchedulable" />
        <div ref="accountTableRef">
          <AccountsViewTable
            :columns="cols"
            :accounts="accounts"
            :loading="loading"
            :all-visible-selected="allVisibleSelected"
            :selected-ids="selIds"
            :toggling-schedulable="togglingSchedulable"
            :today-stats-by-account-id="todayStatsByAccountId"
            :today-stats-loading="todayStatsLoading"
            :today-stats-error="todayStatsError"
            :usage-manual-refresh-token="usageManualRefreshToken"
            :sort-storage-key="ACCOUNT_SORT_STORAGE_KEY"
            :pagination="pagination"
            @toggle-select-all-visible="toggleSelectAllVisible"
            @toggle-selected="toggleSel"
            @show-temp-unsched="handleShowTempUnsched"
            @toggle-schedulable="handleToggleSchedulable"
            @edit="handleEdit"
            @delete="handleDelete"
            @open-menu="handleOpenMenu"
            @page-change="handlePageChange"
            @page-size-change="handlePageSizeChange"
          />
        </div>
      </template>
    </TablePageLayout>
    <AccountsViewDialogsHost
      v-model:include-proxy-on-export="includeProxyOnExport"
      :show-create="showCreate"
      :show-edit="showEdit"
      :show-sync="showSync"
      :show-import-data="showImportData"
      :show-export-data-dialog="showExportDataDialog"
      :show-bulk-edit="showBulkEdit"
      :show-temp-unsched="showTempUnsched"
      :show-delete-dialog="showDeleteDialog"
      :show-re-auth="showReAuth"
      :show-test="showTest"
      :show-stats="showStats"
      :show-error-passthrough="showErrorPassthrough"
      :show-schedule-panel="showSchedulePanel"
      :proxies="proxies"
      :groups="groups"
      :selected-ids="selIds"
      :selected-platforms="selPlatforms"
      :selected-types="selTypes"
      :editing-account="edAcc"
      :temp-unsched-account="tempUnschedAcc"
      :deleting-account="deletingAcc"
      :re-auth-account="reAuthAcc"
      :testing-account="testingAcc"
      :stats-account="statsAcc"
      :schedule-account="scheduleAcc"
      :schedule-model-options="scheduleModelOptions"
      :sync-dialog-open="syncDialogOpen"
      :sync-dialog-models="syncDialogModels"
      :sync-dialog-submitting="syncDialogSubmitting"
      :menu-show="menu.show"
      :menu-account="menu.acc"
      :menu-position="menu.pos"
      @close-create="showCreate = false"
      @created="reload"
      @models-imported="handleImportedModels"
      @close-sync-dialog="closeSyncDialog"
      @submit-sync-dialog="submitSyncDialog"
      @close-edit="showEdit = false"
      @updated="handleAccountUpdated"
      @close-reauth="closeReAuthModal"
      @close-test="closeTestModal"
      @close-stats="closeStatsModal"
      @close-schedule="closeSchedulePanel"
      @close-menu="closeMenu"
      @test="handleTest"
      @stats="handleViewStats"
      @schedule="handleSchedule"
      @reauth="handleReAuth"
      @refresh-token="handleRefresh"
      @recover-state="handleRecoverState"
      @reset-quota="handleResetQuota"
      @import-models="handleImportModels"
      @close-sync="showSync = false"
      @reload="reload"
      @close-import-data="showImportData = false"
      @data-imported="handleDataImported"
      @close-bulk-edit="showBulkEdit = false"
      @bulk-updated="handleBulkUpdated"
      @close-temp-unsched="showTempUnsched = false"
      @temp-unsched-reset="handleTempUnschedReset"
      @confirm-delete="confirmDelete"
      @close-delete="showDeleteDialog = false"
      @confirm-export="handleExportData"
      @close-export="showExportDataDialog = false"
      @close-error-passthrough="showErrorPassthrough = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useModelInventoryStore } from '@/stores'
import { invalidateModelRegistry } from '@/stores/modelRegistry'
import { adminAPI } from '@/api/admin'
import { useAccountActionMenu } from '@/composables/useAccountActionMenu'
import { useAccountsViewLiveSync } from '@/composables/useAccountsViewLiveSync'
import { useAccountsViewListPatching } from '@/composables/useAccountsViewListPatching'
import { useTableLoader } from '@/composables/useTableLoader'
import { useSwipeSelect } from '@/composables/useSwipeSelect'
import { useTableSelection } from '@/composables/useTableSelection'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import AccountBulkActionsBar from '@/components/admin/account/AccountBulkActionsBar.vue'
import AccountsViewDialogsHost from '@/components/admin/account/AccountsViewDialogsHost.vue'
import AccountsViewTable from '@/components/admin/account/AccountsViewTable.vue'
import AccountsViewToolbar from '@/components/admin/account/AccountsViewToolbar.vue'
import type { SelectOption } from '@/components/common/Select.vue'
import { useModelImportExposureSync } from '@/composables/useModelImportExposureSync'
import {
  buildAccountModelImportToastPayload,
  resolveAccountModelImportErrorMessage,
  shouldInvalidateModelInventory
} from '@/utils/accountModelImport'
import type { AccountListRequestParams } from '@/utils/accountListSync'
import type { Account, AccountPlatform, AccountType, Proxy as AccountProxy, AdminGroup, ClaudeModel } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const modelInventoryStore = useModelInventoryStore()
const {
  syncDialogOpen,
  syncDialogModels,
  syncDialogSubmitting,
  handleImportedModels,
  closeSyncDialog,
  submitSyncDialog
} = useModelImportExposureSync({ t, appStore, modelInventoryStore })

const proxies = ref<AccountProxy[]>([])
const groups = ref<AdminGroup[]>([])
const accountTableRef = ref<HTMLElement | null>(null)
const importingModelsAccountId = ref<number | null>(null)
const selPlatforms = computed<AccountPlatform[]>(() => {
  const platforms = new Set(
    accounts.value
      .filter(a => isSelected(a.id))
      .map(a => a.platform)
  )
  return [...platforms]
})
const selTypes = computed<AccountType[]>(() => {
  const types = new Set(
    accounts.value
      .filter(a => isSelected(a.id))
      .map(a => a.type)
  )
  return [...types]
})
const showCreate = ref(false)
const showEdit = ref(false)
const showSync = ref(false)
const showImportData = ref(false)
const showExportDataDialog = ref(false)
const includeProxyOnExport = ref(true)
const showBulkEdit = ref(false)
const showTempUnsched = ref(false)
const showDeleteDialog = ref(false)
const showReAuth = ref(false)
const showTest = ref(false)
const showStats = ref(false)
const showErrorPassthrough = ref(false)
const edAcc = ref<Account | null>(null)
const tempUnschedAcc = ref<Account | null>(null)
const deletingAcc = ref<Account | null>(null)
const reAuthAcc = ref<Account | null>(null)
const testingAcc = ref<Account | null>(null)
const statsAcc = ref<Account | null>(null)
const showSchedulePanel = ref(false)
const scheduleAcc = ref<Account | null>(null)
const scheduleModelOptions = ref<SelectOption[]>([])
const togglingSchedulable = ref<number | null>(null)
const exportingData = ref(false)
const usageManualRefreshToken = ref(0)
const { menu, openMenu, closeMenu, syncMenuAccount, clearMenuAccount } = useAccountActionMenu()

// Column settings
const hiddenColumns = reactive<Set<string>>(new Set())
const DEFAULT_HIDDEN_COLUMNS = ['today_stats', 'proxy', 'notes', 'priority', 'rate_multiplier']
const HIDDEN_COLUMNS_KEY = 'account-hidden-columns'

// Sorting settings
const ACCOUNT_SORT_STORAGE_KEY = 'account-table-sort'

const loadSavedColumns = () => {
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (saved) {
      const parsed = JSON.parse(saved) as string[]
      parsed.forEach(key => {
        hiddenColumns.add(key)
      })
    } else {
      DEFAULT_HIDDEN_COLUMNS.forEach(key => {
        hiddenColumns.add(key)
      })
    }
  } catch (e) {
    console.error('Failed to load saved columns:', e)
    DEFAULT_HIDDEN_COLUMNS.forEach(key => {
      hiddenColumns.add(key)
    })
  }
}

const saveColumnsToStorage = () => {
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
  } catch (e) {
    console.error('Failed to save columns:', e)
  }
}

if (typeof window !== 'undefined') {
  loadSavedColumns()
}

const {
  items: accounts,
  loading,
  params,
  pagination,
  load: baseLoad,
  reload: baseReload,
  debouncedReload: baseDebouncedReload,
  handlePageChange: baseHandlePageChange,
  handlePageSizeChange: baseHandlePageSizeChange
} = useTableLoader<Account, AccountListRequestParams>({
  fetchFn: adminAPI.accounts.list,
  initialParams: { platform: '', type: '', status: '', group: '', search: '' }
})

const handleFilterUpdate = (newFilters: Record<string, unknown>) => {
  Object.assign(params, newFilters)
}

const {
  selectedIds: selIds,
  allVisibleSelected,
  isSelected,
  setSelectedIds,
  select,
  deselect,
  toggle: toggleSel,
  clear: clearSelection,
  removeMany: removeSelectedAccounts,
  toggleVisible,
  selectVisible: selectPage
} = useTableSelection<Account>({
  rows: accounts,
  getId: (account) => account.id
})

const handleSearchQueryUpdate = (value: string) => {
  params.search = value
  debouncedReload()
}

useSwipeSelect(accountTableRef, {
  isSelected,
  select,
  deselect
})

const isAnyModalOpen = computed(() => {
  return (
    showCreate.value ||
    showEdit.value ||
    showSync.value ||
    showImportData.value ||
    showExportDataDialog.value ||
    showBulkEdit.value ||
    showTempUnsched.value ||
    showDeleteDialog.value ||
    showReAuth.value ||
    showTest.value ||
    showStats.value ||
    showSchedulePanel.value ||
    showErrorPassthrough.value
  )
})
const isActionMenuOpen = computed(() => menu.show)
const syncAccountRefs = (nextAccount: Account) => {
  if (edAcc.value?.id === nextAccount.id) edAcc.value = nextAccount
  if (reAuthAcc.value?.id === nextAccount.id) reAuthAcc.value = nextAccount
  if (tempUnschedAcc.value?.id === nextAccount.id) tempUnschedAcc.value = nextAccount
  if (deletingAcc.value?.id === nextAccount.id) deletingAcc.value = nextAccount
  syncMenuAccount(nextAccount)
}

const {
  autoRefreshIntervals,
  autoRefreshEnabled,
  autoRefreshIntervalSeconds,
  autoRefreshCountdown,
  hasPendingListSync,
  todayStatsByAccountId,
  todayStatsLoading,
  todayStatsError,
  load,
  reload,
  debouncedReload,
  handlePageChange,
  handlePageSizeChange,
  refreshTodayStats,
  syncPendingListChanges,
  setAutoRefreshEnabled,
  handleAutoRefreshIntervalChange,
  enterAutoRefreshSilentWindow,
  pauseAutoRefresh,
  resumeAutoRefresh
} = useAccountsViewLiveSync({
  accounts,
  loading,
  params,
  pagination,
  hiddenColumns,
  baseLoad,
  baseReload,
  baseDebouncedReload,
  baseHandlePageChange,
  baseHandlePageSizeChange,
  isAnyModalOpen,
  isActionMenuOpen,
  syncAccountRefs
})

const handleManualRefresh = async () => {
  await load()
  usageManualRefreshToken.value += 1
}

const handleSyncPendingListChanges = async () => {
  await syncPendingListChanges()
  usageManualRefreshToken.value += 1
}

const toggleColumn = (key: string) => {
  const wasHidden = hiddenColumns.has(key)
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key)
  } else {
    hiddenColumns.add(key)
  }
  saveColumnsToStorage()
  if ((key === 'today_stats' || key === 'usage') && wasHidden) {
    refreshTodayStats().catch((error) => {
      console.error('Failed to load account today stats after showing column:', error)
    })
  }
}

// All available columns
const allColumns = computed(() => {
  const c = [
    { key: 'select', label: '', sortable: false },
    { key: 'name', label: t('admin.accounts.columns.name'), sortable: true },
    { key: 'platform_type', label: t('admin.accounts.columns.platformType'), sortable: false },
    { key: 'capacity', label: t('admin.accounts.columns.capacity'), sortable: false },
    { key: 'status', label: t('admin.accounts.columns.status'), sortable: true },
    { key: 'schedulable', label: t('admin.accounts.columns.schedulable'), sortable: true },
    { key: 'today_stats', label: t('admin.accounts.columns.todayStats'), sortable: false }
  ]
  if (!authStore.isSimpleMode) {
    c.push({ key: 'groups', label: t('admin.accounts.columns.groups'), sortable: false })
  }
  c.push(
    { key: 'usage', label: t('admin.accounts.columns.usageWindows'), sortable: false },
    { key: 'usage_reset_dates', label: t('admin.accounts.columns.usageResetDates'), sortable: false },
    { key: 'proxy', label: t('admin.accounts.columns.proxy'), sortable: false },
    { key: 'priority', label: t('admin.accounts.columns.priority'), sortable: true },
    { key: 'rate_multiplier', label: t('admin.accounts.columns.billingRateMultiplier'), sortable: true },
    { key: 'last_used_at', label: t('admin.accounts.columns.lastUsed'), sortable: true },
    { key: 'expires_at', label: t('admin.accounts.columns.expiresAt'), sortable: true },
    { key: 'notes', label: t('admin.accounts.columns.notes'), sortable: false },
    { key: 'actions', label: t('admin.accounts.columns.actions'), sortable: false }
  )
  return c
})

// Columns that can be toggled (exclude select, name, and actions)
const toggleableColumns = computed(() =>
  allColumns.value
    .filter(col => col.key !== 'select' && col.key !== 'name' && col.key !== 'actions')
    .map(col => ({
      key: col.key,
      label: col.label,
      visible: !hiddenColumns.has(col.key)
    }))
)

// Filtered columns based on visibility
const cols = computed(() =>
  allColumns.value.filter(col =>
    col.key === 'select' || col.key === 'name' || col.key === 'actions' || !hiddenColumns.has(col.key)
  )
)

const { patchAccountInList } = useAccountsViewListPatching({
  accounts,
  params,
  pagination,
  hasPendingListSync,
  removeSelectedAccounts,
  syncAccountRefs,
  clearRemovedAccount: clearMenuAccount
})

const handleEdit = (a: Account) => { edAcc.value = a; showEdit.value = true }
const handleOpenMenu = ({ account, event }: { account: Account; event: MouseEvent }) => {
  openMenu({ account, event })
}
const toggleSelectAllVisible = (checked: boolean) => {
  toggleVisible(checked)
}
const handleBulkDelete = async () => { if(!confirm(t('common.confirm'))) return; try { await Promise.all(selIds.value.map(id => adminAPI.accounts.delete(id))); clearSelection(); reload() } catch (error) { console.error('Failed to bulk delete accounts:', error) } }
const handleBulkResetStatus = async () => {
  if (!confirm(t('common.confirm'))) return
  try {
    const result = await adminAPI.accounts.batchClearError(selIds.value)
    if (result.failed > 0) {
      appStore.showError(t('admin.accounts.bulkActions.partialSuccess', { success: result.success, failed: result.failed }))
    } else {
      appStore.showSuccess(t('admin.accounts.bulkActions.resetStatusSuccess', { count: result.success }))
      clearSelection()
    }
    reload()
  } catch (error) {
    console.error('Failed to bulk reset status:', error)
    appStore.showError(String(error))
  }
}
const handleBulkRefreshToken = async () => {
  if (!confirm(t('common.confirm'))) return
  try {
    const result = await adminAPI.accounts.batchRefresh(selIds.value)
    if (result.failed > 0) {
      appStore.showError(t('admin.accounts.bulkActions.partialSuccess', { success: result.success, failed: result.failed }))
    } else {
      appStore.showSuccess(t('admin.accounts.bulkActions.refreshTokenSuccess', { count: result.success }))
      clearSelection()
    }
    reload()
  } catch (error) {
    console.error('Failed to bulk refresh token:', error)
    appStore.showError(String(error))
  }
}
const updateSchedulableInList = (accountIds: number[], schedulable: boolean) => {
  if (accountIds.length === 0) return
  const idSet = new Set(accountIds)
  accounts.value = accounts.value.map((account) => (idSet.has(account.id) ? { ...account, schedulable } : account))
}
const normalizeBulkSchedulableResult = (
  result: {
    success?: number
    failed?: number
    success_ids?: number[]
    failed_ids?: number[]
    results?: Array<{ account_id: number; success: boolean }>
  },
  accountIds: number[]
) => {
  const responseSuccessIds = Array.isArray(result.success_ids) ? result.success_ids : []
  const responseFailedIds = Array.isArray(result.failed_ids) ? result.failed_ids : []
  if (responseSuccessIds.length > 0 || responseFailedIds.length > 0) {
    return {
      successIds: responseSuccessIds,
      failedIds: responseFailedIds,
      successCount: typeof result.success === 'number' ? result.success : responseSuccessIds.length,
      failedCount: typeof result.failed === 'number' ? result.failed : responseFailedIds.length,
      hasIds: true,
      hasCounts: true
    }
  }

  const results = Array.isArray(result.results) ? result.results : []
  if (results.length > 0) {
    const successIds = results.filter(item => item.success).map(item => item.account_id)
    const failedIds = results.filter(item => !item.success).map(item => item.account_id)
    return {
      successIds,
      failedIds,
      successCount: typeof result.success === 'number' ? result.success : successIds.length,
      failedCount: typeof result.failed === 'number' ? result.failed : failedIds.length,
      hasIds: true,
      hasCounts: true
    }
  }

  const hasExplicitCounts = typeof result.success === 'number' || typeof result.failed === 'number'
  const successCount = typeof result.success === 'number' ? result.success : 0
  const failedCount = typeof result.failed === 'number' ? result.failed : 0
  if (hasExplicitCounts && failedCount === 0 && successCount === accountIds.length && accountIds.length > 0) {
    return {
      successIds: accountIds,
      failedIds: [],
      successCount,
      failedCount,
      hasIds: true,
      hasCounts: true
    }
  }

  return {
    successIds: [],
    failedIds: [],
    successCount,
    failedCount,
    hasIds: false,
    hasCounts: hasExplicitCounts
  }
}
const handleBulkToggleSchedulable = async (schedulable: boolean) => {
  const accountIds = [...selIds.value]
  try {
    const result = await adminAPI.accounts.bulkUpdate(accountIds, { schedulable })
    const { successIds, failedIds, successCount, failedCount, hasIds, hasCounts } = normalizeBulkSchedulableResult(result, accountIds)
    if (!hasIds && !hasCounts) {
      appStore.showError(t('admin.accounts.bulkSchedulableResultUnknown'))
      setSelectedIds(accountIds)
      load().catch((error) => {
        console.error('Failed to refresh accounts:', error)
      })
      return
    }
    if (successIds.length > 0) {
      updateSchedulableInList(successIds, schedulable)
    }
    if (successCount > 0 && failedCount === 0) {
      const message = schedulable
        ? t('admin.accounts.bulkSchedulableEnabled', { count: successCount })
        : t('admin.accounts.bulkSchedulableDisabled', { count: successCount })
      appStore.showSuccess(message)
    }
    if (failedCount > 0) {
      const message = hasCounts || hasIds
        ? t('admin.accounts.bulkSchedulablePartial', { success: successCount, failed: failedCount })
        : t('admin.accounts.bulkSchedulableResultUnknown')
      appStore.showError(message)
      setSelectedIds(failedIds.length > 0 ? failedIds : accountIds)
    } else {
      if (hasIds) clearSelection()
      else setSelectedIds(accountIds)
    }
  } catch (error) {
    console.error('Failed to bulk toggle schedulable:', error)
    appStore.showError(t('common.error'))
  }
}
const handleBulkUpdated = () => { showBulkEdit.value = false; clearSelection(); reload() }
const handleDataImported = () => { showImportData.value = false; reload() }
const handleAccountUpdated = (updatedAccount: Account) => {
  patchAccountInList(updatedAccount)
  enterAutoRefreshSilentWindow()
}
const formatExportTimestamp = () => {
  const now = new Date()
  const pad2 = (value: number) => String(value).padStart(2, '0')
  return `${now.getFullYear()}${pad2(now.getMonth() + 1)}${pad2(now.getDate())}${pad2(now.getHours())}${pad2(now.getMinutes())}${pad2(now.getSeconds())}`
}
const openExportDataDialog = () => {
  includeProxyOnExport.value = true
  showExportDataDialog.value = true
}
const handleExportData = async () => {
  if (exportingData.value) return
  exportingData.value = true
  try {
    const dataPayload = await adminAPI.accounts.exportData(
      selIds.value.length > 0
        ? { ids: selIds.value, includeProxies: includeProxyOnExport.value }
        : {
            includeProxies: includeProxyOnExport.value,
            filters: {
              platform: params.platform,
              type: params.type,
              status: params.status,
              search: params.search
            }
          }
    )
    const timestamp = formatExportTimestamp()
    const filename = `sub2api-account-${timestamp}.json`
    const blob = new Blob([JSON.stringify(dataPayload, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    link.click()
    URL.revokeObjectURL(url)
    appStore.showSuccess(t('admin.accounts.dataExported'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.dataExportFailed'))
  } finally {
    exportingData.value = false
    showExportDataDialog.value = false
  }
}
const closeTestModal = () => { showTest.value = false; testingAcc.value = null }
const closeStatsModal = () => { showStats.value = false; statsAcc.value = null }
const closeReAuthModal = () => { showReAuth.value = false; reAuthAcc.value = null }
const handleTest = (a: Account) => { testingAcc.value = a; showTest.value = true }
const handleViewStats = (a: Account) => { statsAcc.value = a; showStats.value = true }
const handleSchedule = async (a: Account) => {
  scheduleAcc.value = a
  scheduleModelOptions.value = []
  showSchedulePanel.value = true
  try {
    const models = await adminAPI.accounts.getAvailableModels(a.id)
    scheduleModelOptions.value = models.map((m: ClaudeModel) => ({ value: m.id, label: m.display_name || m.id }))
  } catch {
    scheduleModelOptions.value = []
  }
}
const closeSchedulePanel = () => { showSchedulePanel.value = false; scheduleAcc.value = null; scheduleModelOptions.value = [] }
const handleReAuth = (a: Account) => { reAuthAcc.value = a; showReAuth.value = true }
const handleRefresh = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.refreshCredentials(a.id)
    patchAccountInList(updated)
    enterAutoRefreshSilentWindow()
  } catch (error) {
    console.error('Failed to refresh credentials:', error)
  }
}
const handleRecoverState = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.recoverState(a.id)
    patchAccountInList(updated)
    enterAutoRefreshSilentWindow()
    appStore.showSuccess(t('admin.accounts.recoverStateSuccess'))
  } catch (error: any) {
    console.error('Failed to recover account state:', error)
    appStore.showError(error?.message || t('admin.accounts.recoverStateFailed'))
  }
}
const handleImportModels = async (a: Account, trigger: 'manual' | 'create' = 'manual') => {
  if (importingModelsAccountId.value === a.id) {
    return
  }
  importingModelsAccountId.value = a.id
  appStore.showInfo(t('admin.accounts.probingModels'))
  try {
    const result = await adminAPI.accounts.importModels(a.id, { trigger })
    const toastPayload = buildAccountModelImportToastPayload(t, result)
    if (toastPayload.type === 'error') {
      appStore.showError(toastPayload.message, toastPayload.options)
    } else if (toastPayload.type === 'warning') {
      appStore.showWarning(toastPayload.message, toastPayload.options)
    } else {
      appStore.showSuccess(toastPayload.message, toastPayload.options)
    }
    if (shouldInvalidateModelInventory(result)) {
      invalidateModelRegistry()
      modelInventoryStore.invalidate()
    }
    handleImportedModels(result)
    return result
  } catch (error: any) {
    console.error('Failed to import models for account:', error)
    appStore.showError(resolveAccountModelImportErrorMessage(t, error))
    return null
  } finally {
    importingModelsAccountId.value = null
  }
}

const handleResetQuota = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.resetAccountQuota(a.id)
    patchAccountInList(updated)
    enterAutoRefreshSilentWindow()
    appStore.showSuccess(t('common.success'))
  } catch (error) {
    console.error('Failed to reset quota:', error)
  }
}
const handleDelete = (a: Account) => { deletingAcc.value = a; showDeleteDialog.value = true }
const confirmDelete = async () => { if(!deletingAcc.value) return; try { await adminAPI.accounts.delete(deletingAcc.value.id); showDeleteDialog.value = false; deletingAcc.value = null; reload() } catch (error) { console.error('Failed to delete account:', error) } }
const handleToggleSchedulable = async (a: Account) => {
  const nextSchedulable = !a.schedulable
  togglingSchedulable.value = a.id
  try {
    const updated = await adminAPI.accounts.setSchedulable(a.id, nextSchedulable)
    updateSchedulableInList([a.id], updated?.schedulable ?? nextSchedulable)
    enterAutoRefreshSilentWindow()
  } catch (error) {
    console.error('Failed to toggle schedulable:', error)
    appStore.showError(t('admin.accounts.failedToToggleSchedulable'))
  } finally {
    togglingSchedulable.value = null
  }
}
const handleShowTempUnsched = (a: Account) => { tempUnschedAcc.value = a; showTempUnsched.value = true }
const handleTempUnschedReset = async (updated: Account) => {
  showTempUnsched.value = false
  tempUnschedAcc.value = null
  patchAccountInList(updated)
  enterAutoRefreshSilentWindow()
}

// 婊氬姩鏃跺叧闂搷浣滆彍鍗曪紙涓嶅叧闂垪璁剧疆涓嬫媺鑿滃崟锛?
const handleScroll = () => {
  menu.show = false
}

// 鐐瑰嚮澶栭儴鍏抽棴鍒楄缃笅鎷夎彍鍗?

onMounted(async () => {
  load().catch((error) => {
    console.error('Failed to load accounts:', error)
  })
  try {
    const [p, g] = await Promise.all([adminAPI.proxies.getAll(), adminAPI.groups.getAll()])
    proxies.value = p
    groups.value = g
  } catch (error) {
    console.error('Failed to load proxies/groups:', error)
  }
  window.addEventListener('scroll', handleScroll, true)

  if (autoRefreshEnabled.value) {
    autoRefreshCountdown.value = autoRefreshIntervalSeconds.value
    resumeAutoRefresh()
  } else {
    pauseAutoRefresh()
  }
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll, true)
})
</script>
