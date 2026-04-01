<template>
  <AppLayout>
    <TablePageLayout prefer-page-scroll>
      <template #filters>
        <AccountsViewToolbar
          :loading="loading"
          :usage-refreshing="usageRefreshing"
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
          :view-mode="viewMode"
          :group-view-enabled="groupViewEnabled"
          :platform-count-sort-order="platformCountSortOrder"
          :show-limited-controls="!limitedMode"
          :hide-limited-accounts="hideLimitedAccounts"
          :limited-accounts-count="limitedAccountsCount"
          @update:filters="handleFilterUpdate"
          @update:search-query="handleSearchQueryUpdate"
          @update:view-mode="viewMode = $event"
          @update:platform-count-sort-order="handlePlatformCountSortOrderUpdate"
          @change="debouncedReload"
          @refresh="handleManualRefresh"
          @refresh-usage="handleRefreshActualUsage"
          @sync="showSync = true"
          @create="showCreate = true"
          @import-data="showImportData = true"
          @export-data="openExportDataDialog"
          @show-error-passthrough="showErrorPassthrough = true"
          @show-tls-fingerprint-profiles="showTLSFingerprintProfiles = true"
          @sync-pending-list="handleSyncPendingListChanges"
          @set-auto-refresh-enabled="setAutoRefreshEnabled"
          @set-auto-refresh-interval="handleAutoRefreshIntervalChange"
          @toggle-column="toggleColumn"
          @toggle-group-view="groupViewEnabled = !groupViewEnabled"
          @toggle-hide-limited="toggleHideLimitedAccounts"
          @open-limited-page="openLimitedAccountsPage"
        />
      </template>
      <template #table>
        <div class="space-y-2">
          <AccountPlatformTabs
            :model-value="String(params.platform || '')"
            :platform-counts="toolbarSummary.by_platform"
            @update:model-value="handlePlatformTabChange"
          />

          <AccountLimitedSummaryBar
            v-if="limitedMode"
            :summary="toolbarSummary"
            :loading="summaryLoading"
            :error="summaryError"
            :active-reason="activeLimitedReason"
            @select-reason="handleLimitedReasonSelect"
          />

          <AccountStatusSummaryBar
            v-else
            :summary="toolbarSummary"
            :loading="summaryLoading"
            :error="summaryError"
            :active-status="String(params.status || '')"
            :active-runtime-view="String(params.runtime_view || 'all')"
            @select-status="handleSummaryStatusSelect"
            @select-runtime-view="handleRuntimeViewSelect"
          />

          <AccountBulkActionsBar
            :selected-ids="selIds"
            :selected-platforms="selPlatforms"
            @archive="openArchiveSelectedModal"
            @delete="handleBulkDelete"
            @reset-status="handleBulkResetStatus"
            @refresh-token="handleBulkRefreshToken"
            @edit="showBulkEdit = true"
            @clear="clearSelection"
            @select-page="selectPage"
            @toggle-schedulable="handleBulkToggleSchedulable"
          />

          <div ref="accountTableRef">
            <AccountGroupedView
              v-if="groupViewEnabled"
              :accounts="displayAccounts"
              :groups="groups"
              :group-filter="String(params.group || '')"
              :view-mode="viewMode"
              :columns="cols"
              :selected-ids="selIds"
              :loading="loading"
              :toggling-schedulable="togglingSchedulable"
              :today-stats-by-account-id="todayStatsByAccountId"
              :today-stats-loading="todayStatsLoading"
              :today-stats-error="todayStatsError"
              :usage-manual-refresh-token="usageManualRefreshToken"
              :sort-storage-key="ACCOUNT_SORT_STORAGE_KEY"
              :preserve-input-order="true"
              @toggle-selected="toggleSel"
              @toggle-section-selected="handleToggleSectionSelected"
              @show-temp-unsched="handleShowTempUnsched"
              @toggle-schedulable="handleToggleSchedulable"
              @edit="handleEdit"
              @delete="handleDelete"
              @open-menu="handleOpenMenu"
            />

            <AccountCardGrid
              v-else-if="viewMode === 'card'"
              :accounts="displayAccounts"
              :loading="loading"
              :selected-ids="selIds"
              :toggling-schedulable="togglingSchedulable"
              :today-stats-by-account-id="todayStatsByAccountId"
              :today-stats-loading="todayStatsLoading"
              :usage-manual-refresh-token="usageManualRefreshToken"
              @toggle-selected="toggleSel"
              @show-temp-unsched="handleShowTempUnsched"
              @toggle-schedulable="handleToggleSchedulable"
              @edit="handleEdit"
              @delete="handleDelete"
              @open-menu="handleOpenMenu"
            />

            <AccountsViewTable
              v-else
              :columns="cols"
              :accounts="displayAccounts"
              :loading="loading"
              :all-visible-selected="allVisibleSelected"
              :selected-ids="selIds"
              :toggling-schedulable="togglingSchedulable"
              :today-stats-by-account-id="todayStatsByAccountId"
              :today-stats-loading="todayStatsLoading"
              :today-stats-error="todayStatsError"
              :usage-manual-refresh-token="usageManualRefreshToken"
              :sort-storage-key="ACCOUNT_SORT_STORAGE_KEY"
              :preserve-input-order="true"
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

          <Pagination
            v-if="showStandalonePagination && pagination.total > 0"
            :page="pagination.page"
            :total="pagination.total"
            :page-size="pagination.page_size"
            @update:page="handlePageChange"
            @update:page-size="handlePageSizeChange"
          />
        </div>
      </template>
    </TablePageLayout>
    <AccountsViewDialogsHost
      v-model:include-proxy-on-export="includeProxyOnExport"
      :show-create="showCreate"
      :show-archive-selected="showArchiveSelected"
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
      :show-model-diagnostics="showModelDiagnostics"
      :show-error-passthrough="showErrorPassthrough"
      :show-tls-fingerprint-profiles="showTLSFingerprintProfiles"
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
      :diagnostics-account="diagnosticsAccount"
      :diagnostics-result="diagnosticsResult"
      :diagnostics-loading="diagnosticsLoading"
      :schedule-account="scheduleAcc"
      :schedule-model-options="scheduleModelOptions"
      :sync-dialog-open="syncDialogOpen"
      :sync-dialog-models="syncDialogModels"
      :sync-dialog-submitting="syncDialogSubmitting"
      :menu-show="menu.show"
      :menu-account="menu.acc"
      :menu-position="menu.pos"
      @close-create="showCreate = false"
      @created="handleCreated"
      @close-archive-selected="showArchiveSelected = false"
      @archived="handleArchivedAccounts"
      @models-imported="handleImportedModels"
      @close-sync-dialog="closeSyncDialog"
      @submit-sync-dialog="submitSyncDialog"
      @close-edit="showEdit = false"
      @updated="handleAccountUpdated"
      @close-reauth="closeReAuthModal"
      @close-test="closeTestModal"
      @close-stats="closeStatsModal"
      @close-model-diagnostics="closeModelDiagnostics"
      @close-schedule="closeSchedulePanel"
      @close-menu="closeMenu"
      @test="handleTest"
      @stats="handleViewStats"
      @diagnose-models="handleDiagnoseModels"
      @refresh-model-diagnostics="refreshModelDiagnostics"
      @schedule="handleSchedule"
      @reauth="handleReAuth"
      @refresh-token="handleRefresh"
      @set-privacy="handleSetPrivacy"
      @recover-state="handleRecoverState"
      @reset-quota="handleResetQuota"
      @blacklist="handleBlacklistAccount"
      @test-blacklist="handleTestBlacklistAccount"
      @import-models="handleImportModels"
      @close-sync="showSync = false"
      @reload="handleReloadRequested"
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
      @close-tls-fingerprint-profiles="showTLSFingerprintProfiles = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useModelInventoryStore } from '@/stores'
import { invalidateModelRegistry } from '@/stores/modelRegistry'
import { adminAPI } from '@/api/admin'
import { useAccountStatusSummary } from '@/composables/useAccountStatusSummary'
import { useAccountActionMenu } from '@/composables/useAccountActionMenu'
import { useAccountViewMode } from '@/composables/useAccountViewMode'
import { useAccountsRuntimeSummary } from '@/composables/useAccountsRuntimeSummary'
import { useAccountsViewLiveSync } from '@/composables/useAccountsViewLiveSync'
import { useAccountsViewListPatching } from '@/composables/useAccountsViewListPatching'
import { useTableLoader } from '@/composables/useTableLoader'
import { useSwipeSelect } from '@/composables/useSwipeSelect'
import { useTableSelection } from '@/composables/useTableSelection'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import AccountCardGrid from '@/components/admin/account/AccountCardGrid.vue'
import AccountBulkActionsBar from '@/components/admin/account/AccountBulkActionsBar.vue'
import AccountGroupedView from '@/components/admin/account/AccountGroupedView.vue'
import AccountLimitedSummaryBar from '@/components/admin/account/AccountLimitedSummaryBar.vue'
import AccountPlatformTabs from '@/components/admin/account/AccountPlatformTabs.vue'
import AccountStatusSummaryBar from '@/components/admin/account/AccountStatusSummaryBar.vue'
import AccountsViewDialogsHost from '@/components/admin/account/AccountsViewDialogsHost.vue'
import AccountsViewTable from '@/components/admin/account/AccountsViewTable.vue'
import AccountsViewToolbar from '@/components/admin/account/AccountsViewToolbar.vue'
import type { SelectOption } from '@/components/common/Select.vue'
import type {
  AccountModelDiagnosticsResponse,
  BlacklistFeedbackPayload
} from '@/api/admin/accounts'
import {
  canAccountFetchUsage,
  invalidateAccountUsagePresentationCache,
  refreshAccountUsagePresentation,
  resolveActualUsageRefreshLoadOptions,
} from '@/composables/useAccountUsagePresentation'
import { useModelImportExposureSync } from '@/composables/useModelImportExposureSync'
import {
  buildAccountModelImportToastPayload,
  resolveAccountModelImportErrorMessage,
  shouldInvalidateModelInventory
} from '@/utils/accountModelImport'
import type { AccountListRequestParams } from '@/utils/accountListSync'
import type {
  Account,
  AccountRateLimitReason,
  AccountPlatform,
  AccountPlatformCountSortOrder,
  AccountRuntimeView,
  AccountType,
  Proxy as AccountProxy,
  AdminGroup,
  ClaudeModel
} from '@/types'

const props = withDefaults(defineProps<{
  limitedMode?: boolean
}>(), {
  limitedMode: false
})

const limitedMode = computed(() => props.limitedMode)
const { t } = useI18n()
const router = useRouter()
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
const { viewMode, groupViewEnabled } = useAccountViewMode()
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
const showArchiveSelected = ref(false)
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
const showModelDiagnostics = ref(false)
const showErrorPassthrough = ref(false)
const showTLSFingerprintProfiles = ref(false)
const edAcc = ref<Account | null>(null)
const tempUnschedAcc = ref<Account | null>(null)
const deletingAcc = ref<Account | null>(null)
const reAuthAcc = ref<Account | null>(null)
const testingAcc = ref<Account | null>(null)
const statsAcc = ref<Account | null>(null)
const diagnosticsAccount = ref<Account | null>(null)
const diagnosticsResult = ref<AccountModelDiagnosticsResponse | null>(null)
const diagnosticsLoading = ref(false)
const showSchedulePanel = ref(false)
const scheduleAcc = ref<Account | null>(null)
const scheduleModelOptions = ref<SelectOption[]>([])
const togglingSchedulable = ref<number | null>(null)
const exportingData = ref(false)
const usageManualRefreshToken = ref(0)
const usageRefreshing = ref(false)
const archivedPanelRefreshToken = ref(0)
const { menu, openMenu, closeMenu, syncMenuAccount, clearMenuAccount } = useAccountActionMenu()

// Column settings
const hiddenColumns = reactive<Set<string>>(new Set())
const DEFAULT_HIDDEN_COLUMNS = ['today_stats', 'proxy', 'notes', 'priority', 'rate_multiplier']
const HIDDEN_COLUMNS_KEY = 'account-hidden-columns'

// Sorting settings
const ACCOUNT_SORT_STORAGE_KEY = 'account-table-sort'
const HIDE_LIMITED_ACCOUNTS_STORAGE_KEY = 'account-always-hide-limited-accounts'
const PLATFORM_COUNT_SORT_ORDER_STORAGE_KEY = 'account-platform-count-sort-order'
const ACCOUNT_PLATFORM_DISPLAY_ORDER: AccountPlatform[] = [
  'anthropic',
  'kiro',
  'openai',
  'copilot',
  'grok',
  'protocol_gateway',
  'gemini',
  'antigravity',
  'sora'
]
const ACCOUNT_PLATFORM_ORDER_INDEX = new Map(
  ACCOUNT_PLATFORM_DISPLAY_ORDER.map((platform, index) => [platform, index])
)

const loadHideLimitedPreference = () => {
  if (typeof window === 'undefined') {
    return true
  }
  try {
    const saved = localStorage.getItem(HIDE_LIMITED_ACCOUNTS_STORAGE_KEY)
    return saved === 'true'
  } catch (error) {
    console.error('Failed to load limited accounts visibility:', error)
    return false
  }
}

const saveHideLimitedPreference = (value: boolean) => {
  if (typeof window === 'undefined') {
    return
  }
  try {
    localStorage.setItem(HIDE_LIMITED_ACCOUNTS_STORAGE_KEY, String(value))
  } catch (error) {
    console.error('Failed to save limited accounts visibility:', error)
  }
}

const loadPlatformCountSortOrderPreference = (): AccountPlatformCountSortOrder => {
  if (typeof window === 'undefined') {
    return 'count_asc'
  }
  try {
    const saved = localStorage.getItem(PLATFORM_COUNT_SORT_ORDER_STORAGE_KEY)
    return saved === 'count_desc' ? 'count_desc' : 'count_asc'
  } catch (error) {
    console.error('Failed to load platform count sort order:', error)
    return 'count_asc'
  }
}

const savePlatformCountSortOrderPreference = (value: AccountPlatformCountSortOrder) => {
  if (typeof window === 'undefined') {
    return
  }
  try {
    localStorage.setItem(PLATFORM_COUNT_SORT_ORDER_STORAGE_KEY, value)
  } catch (error) {
    console.error('Failed to save platform count sort order:', error)
  }
}

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
  initialParams: {
    platform: '',
    type: '',
    status: '',
    group: '',
    privacy_mode: '',
    search: '',
    lifecycle: 'normal',
    limited_view: limitedMode.value ? 'limited_only' : (loadHideLimitedPreference() ? 'normal_only' : 'all'),
    limited_reason: '',
    runtime_view: 'all'
  }
})

const hideLimitedAccounts = computed(() => !limitedMode.value && String(params.limited_view || '') === 'normal_only')
const platformCountSortOrder = ref<AccountPlatformCountSortOrder>(loadPlatformCountSortOrderPreference())

const handleFilterUpdate = (newFilters: Record<string, unknown>) => {
  Object.assign(params, newFilters)
}

const handlePlatformCountSortOrderUpdate = (value: AccountPlatformCountSortOrder) => {
  platformCountSortOrder.value = value
  savePlatformCountSortOrderPreference(value)
}

const summaryParams = computed<AccountListRequestParams>(() => ({
  platform: String(params.platform || ''),
  type: String(params.type || ''),
  group: String(params.group || ''),
  privacy_mode: String(params.privacy_mode || ''),
  search: String(params.search || ''),
  lifecycle: String(params.lifecycle || ''),
  limited_view: limitedMode.value ? 'limited_only' : 'all',
  limited_reason: limitedMode.value ? '' : String(params.limited_reason || '')
}))

const {
  summary: accountSummaryState,
  loading: summaryLoading,
  error: summaryError,
  refresh: refreshAccountSummary
} = useAccountStatusSummary(summaryParams)

const accountSummary = computed(() => accountSummaryState.value)
const activeLimitedReason = computed<AccountRateLimitReason | ''>(() => {
  const value = String(params.limited_reason || '')
  return value === 'rate_429' || value === 'usage_5h' || value === 'usage_7d' ? value : ''
})

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
    showArchiveSelected.value ||
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
    showModelDiagnostics.value ||
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
  if (diagnosticsAccount.value?.id === nextAccount.id) diagnosticsAccount.value = nextAccount
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
  refreshAccountsIncrementally,
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
  syncAccountRefs,
  onListChanged: refreshAccountSummary
})

const isLiveSyncBlocked = computed(() => loading.value || isAnyModalOpen.value || isActionMenuOpen.value)
const pendingRuntimeListRefresh = ref(false)
const runtimeSummaryParams = computed<AccountListRequestParams>(() => ({
  platform: String(params.platform || ''),
  type: String(params.type || ''),
  group: String(params.group || ''),
  privacy_mode: String(params.privacy_mode || ''),
  search: String(params.search || ''),
  lifecycle: String(params.lifecycle || ''),
  limited_view: limitedMode.value ? 'limited_only' : 'all',
  limited_reason: limitedMode.value ? '' : String(params.limited_reason || ''),
  runtime_view: String(params.runtime_view || 'all')
}))
const triggerRuntimeInUseRefresh = async () => {
  if (limitedMode.value || String(params.runtime_view || 'all') !== 'in_use_only') {
    pendingRuntimeListRefresh.value = false
    return
  }
  if (isLiveSyncBlocked.value) {
    pendingRuntimeListRefresh.value = true
    return
  }
  pendingRuntimeListRefresh.value = false
  await refreshAccountsIncrementally()
}
const {
  summary: runtimeSummaryState,
  refresh: refreshRuntimeSummary
} = useAccountsRuntimeSummary(runtimeSummaryParams, {
  enabled: computed(() => !limitedMode.value),
  onSummaryChanged: async () => {
    await triggerRuntimeInUseRefresh()
  }
})
const toolbarSummary = computed(() => ({
  ...accountSummary.value,
  in_use: limitedMode.value ? 0 : runtimeSummaryState.value.in_use,
  remaining_available: limitedMode.value
    ? 0
    : Math.max(
        (accountSummary.value.remaining_available + accountSummary.value.in_use) - runtimeSummaryState.value.in_use,
        0
      )
}))
const displayAccounts = computed<Account[]>(() => {
  const pagePlatformCounts = accounts.value.reduce<Partial<Record<AccountPlatform, number>>>((acc, account) => {
    acc[account.platform] = (acc[account.platform] ?? 0) + 1
    return acc
  }, {})

  return accounts.value
    .map((account, index) => ({
      account,
      index,
      count: pagePlatformCounts[account.platform] ?? 0,
      platformRank: ACCOUNT_PLATFORM_ORDER_INDEX.get(account.platform) ?? Number.MAX_SAFE_INTEGER
    }))
    .sort((left, right) => {
      if (left.count !== right.count) {
        return platformCountSortOrder.value === 'count_desc'
          ? right.count - left.count
          : left.count - right.count
      }

      if (left.platformRank !== right.platformRank) {
        return left.platformRank - right.platformRank
      }

      return left.index - right.index
    })
    .map((item) => item.account)
})
const limitedAccountsCount = computed(() => accountSummary.value.limited_breakdown.total)

watch(isLiveSyncBlocked, (blocked, wasBlocked) => {
  if (wasBlocked && !blocked && pendingRuntimeListRefresh.value) {
    triggerRuntimeInUseRefresh().catch((error) => {
      console.error('Failed to refresh in-use accounts after page became idle:', error)
    })
  }
})

const handleManualRefresh = async () => {
  await load()
  await refreshRuntimeSummary(true)
  refreshArchivedPanel()
  usageManualRefreshToken.value += 1
}

const handleSyncPendingListChanges = async () => {
  await syncPendingListChanges()
  await refreshRuntimeSummary(true)
  refreshArchivedPanel()
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

const refreshArchivedPanel = () => {
  archivedPanelRefreshToken.value += 1
}

const refreshAccountSummarySafe = () => {
  refreshAccountSummary().catch((error) => {
    console.error('Failed to refresh account summary:', error)
  })
  refreshRuntimeSummary(true).catch((error) => {
    console.error('Failed to refresh account runtime summary:', error)
  })
}

const handleEdit = (a: Account) => { edAcc.value = a; showEdit.value = true }
const handleOpenMenu = ({ account, event }: { account: Account; event: MouseEvent }) => {
  openMenu({ account, event })
}
const toggleSelectAllVisible = (checked: boolean) => {
  toggleVisible(checked)
}
const handleToggleSectionSelected = ({ ids, checked }: { ids: number[]; checked: boolean }) => {
  ids.forEach((id) => {
    if (checked) {
      select(id)
      return
    }
    deselect(id)
  })
}
const applyBoardSelection = (next: {
  platform?: string
  status?: string
  runtimeView?: string
}) => {
  if (!limitedMode.value) {
    params.limited_view = hideLimitedAccounts.value ? 'normal_only' : 'all'
    params.limited_reason = ''
  }
  if (typeof next.platform !== 'undefined') {
    params.platform = next.platform
  }
  if (typeof next.status !== 'undefined') {
    params.status = next.status
  }
  if (typeof next.runtimeView !== 'undefined') {
    params.runtime_view = next.runtimeView
  }
  debouncedReload()
}

const handlePlatformTabChange = (value: string) => {
  applyBoardSelection({
    platform: value
  })
}
const handleSummaryStatusSelect = (status: string) => {
  const nextStatus = String(params.status || '') === status ? '' : status
  applyBoardSelection({
    status: nextStatus,
    runtimeView: limitedMode.value ? String(params.runtime_view || 'all') : 'all'
  })
}
const handleRuntimeViewSelect = (runtimeView: AccountRuntimeView | string) => {
  if (limitedMode.value) {
    return
  }
  const nextRuntimeView = String(params.runtime_view || 'all') === runtimeView ? 'all' : runtimeView
  applyBoardSelection({
    status: '',
    runtimeView: nextRuntimeView
  })
}
const handleLimitedReasonSelect = (reason: AccountRateLimitReason | '') => {
  params.limited_reason = activeLimitedReason.value === reason ? '' : reason
  debouncedReload()
}
const toggleHideLimitedAccounts = () => {
  const nextHidden = !hideLimitedAccounts.value
  params.limited_view = nextHidden ? 'normal_only' : 'all'
  if (nextHidden && String(params.status || '') === 'rate_limited') {
    params.status = ''
  }
  saveHideLimitedPreference(nextHidden)
  debouncedReload()
}
const openLimitedAccountsPage = () => {
  router.push({ path: '/admin/accounts/limited' }).catch((error) => {
    console.error('Failed to open limited accounts page:', error)
  })
}
const showStandalonePagination = computed(() => groupViewEnabled.value || viewMode.value === 'card')
const handleBulkDelete = async () => { if(!confirm(t('common.confirm'))) return; try { await Promise.all(selIds.value.map(id => adminAPI.accounts.delete(id))); clearSelection(); reload() } catch (error) { console.error('Failed to bulk delete accounts:', error) } }
const openArchiveSelectedModal = () => {
  if (selIds.value.length === 0) {
    return
  }
  if (selPlatforms.value.length !== 1) {
    appStore.showWarning(t('admin.accounts.bulkActions.archiveMixedPlatformDisabled'))
    return
  }
  showArchiveSelected.value = true
}
const handleRefreshActualUsage = async () => {
  if (usageRefreshing.value) return

  const visibleAccounts = accounts.value.filter(canAccountFetchUsage)
  if (visibleAccounts.length === 0) {
    appStore.showWarning(t('admin.accounts.refreshActualUsageNoAccounts'))
    return
  }

  usageRefreshing.value = true
  invalidateAccountUsagePresentationCache(visibleAccounts.map((account) => account.id))

  try {
    const result = await refreshAccountUsagePresentation(visibleAccounts, {
      force: true,
      concurrency: 4,
      resolveLoadOptions: resolveActualUsageRefreshLoadOptions,
    })

    if (result.failed > 0 && result.success > 0) {
      appStore.showWarning(
        t('admin.accounts.refreshActualUsagePartial', {
          success: result.success,
          failed: result.failed
        })
      )
      return
    }

    if (result.failed > 0) {
      appStore.showError(
        t('admin.accounts.refreshActualUsageFailedCount', {
          failed: result.failed
        })
      )
      return
    }

    appStore.showSuccess(
      t('admin.accounts.refreshActualUsageSuccess', {
        count: result.success
      })
    )
  } catch (error: any) {
    console.error('Failed to refresh actual account usage:', error)
    appStore.showError(error?.message || t('admin.accounts.refreshActualUsageFailed'))
  } finally {
    usageRefreshing.value = false
  }
}
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

const refreshGroups = async () => {
  try {
    groups.value = await adminAPI.groups.getAll()
  } catch (error) {
    console.error('Failed to refresh groups:', error)
  }
}

const refreshListAndArchivedPanel = async () => {
  refreshArchivedPanel()
  await reload()
}

const handleReloadRequested = async () => {
  await refreshListAndArchivedPanel()
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
    refreshAccountSummarySafe()
  } catch (error) {
    console.error('Failed to bulk toggle schedulable:', error)
    appStore.showError(t('common.error'))
  }
}
const handleBulkUpdated = () => { showBulkEdit.value = false; clearSelection(); reload() }
const handleDataImported = async () => {
  showImportData.value = false
  await refreshGroups()
  await refreshListAndArchivedPanel()
}
const handleCreated = async () => {
  showCreate.value = false
  await reload()
}
const handleArchivedAccounts = async () => {
  showArchiveSelected.value = false
  clearSelection()
  await refreshGroups()
  await refreshListAndArchivedPanel()
}
const handleAccountUpdated = (updatedAccount: Account) => {
  const editedArchived = edAcc.value?.id === updatedAccount.id && edAcc.value.lifecycle_state === 'archived'
  patchAccountInList(updatedAccount)
  refreshAccountSummarySafe()
  enterAutoRefreshSilentWindow()
  if (editedArchived || (updatedAccount.lifecycle_state && updatedAccount.lifecycle_state !== 'normal')) {
    refreshArchivedPanel()
  }
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
const closeTestModal = async () => {
  showTest.value = false
  testingAcc.value = null
  await refreshListAndArchivedPanel()
}
const closeStatsModal = () => { showStats.value = false; statsAcc.value = null }
const closeModelDiagnostics = () => {
  showModelDiagnostics.value = false
  diagnosticsAccount.value = null
  diagnosticsResult.value = null
  diagnosticsLoading.value = false
}
const closeReAuthModal = () => { showReAuth.value = false; reAuthAcc.value = null }
const handleTest = (a: Account) => { testingAcc.value = a; showTest.value = true }
const handleViewStats = (a: Account) => { statsAcc.value = a; showStats.value = true }
const handleDiagnoseModels = async (a: Account) => {
  const sameAccount = diagnosticsAccount.value?.id === a.id
  diagnosticsAccount.value = a
  showModelDiagnostics.value = true
  if (!sameAccount) {
    diagnosticsResult.value = null
  }
  diagnosticsLoading.value = true
  try {
    diagnosticsResult.value = await adminAPI.accounts.diagnoseAccountModels(a.id)
  } catch (error: any) {
    console.error('Failed to diagnose account model exposure:', error)
    appStore.showError(error?.message || t('admin.accounts.modelDiagnostics.failed'))
  } finally {
    diagnosticsLoading.value = false
  }
}
const refreshModelDiagnostics = async () => {
  if (!diagnosticsAccount.value) {
    return
  }
  diagnosticsLoading.value = true
  try {
    diagnosticsResult.value = await adminAPI.accounts.diagnoseAccountModels(
      diagnosticsAccount.value.id,
      { refresh: true }
    )
  } catch (error: any) {
    console.error('Failed to refresh account model exposure diagnostics:', error)
    appStore.showError(error?.message || t('admin.accounts.modelDiagnostics.failed'))
  } finally {
    diagnosticsLoading.value = false
  }
}
const scheduleSourceProtocolLabel = (sourceProtocol?: string) => {
  switch (String(sourceProtocol || '').trim()) {
    case 'openai':
      return t('admin.accounts.protocolGateway.protocolOptions.openai')
    case 'anthropic':
      return t('admin.accounts.protocolGateway.protocolOptions.anthropic')
    case 'gemini':
      return t('admin.accounts.protocolGateway.protocolOptions.gemini')
    default:
      return ''
  }
}
const handleSchedule = async (a: Account) => {
  scheduleAcc.value = a
  scheduleModelOptions.value = []
  showSchedulePanel.value = true
  try {
    const models = await adminAPI.accounts.getAvailableModels(a.id)
    scheduleModelOptions.value = models.map((m: ClaudeModel) => {
      const sourceProtocol = String(m.source_protocol || '').trim()
      const protocolLabel = scheduleSourceProtocolLabel(sourceProtocol)
      return {
        value: `${sourceProtocol || 'default'}::${m.id}`,
        label: protocolLabel ? `${m.display_name || m.id} · ${protocolLabel}` : (m.display_name || m.id),
        model_id: m.id,
        source_protocol: sourceProtocol || undefined
      }
    })
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
    refreshAccountSummarySafe()
    enterAutoRefreshSilentWindow()
    if (a.lifecycle_state === 'archived' || updated.lifecycle_state === 'archived') {
      refreshArchivedPanel()
    }
  } catch (error) {
    console.error('Failed to refresh credentials:', error)
  }
}
const handleSetPrivacy = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.setPrivacy(a.id)
    patchAccountInList(updated)
    refreshAccountSummarySafe()
    enterAutoRefreshSilentWindow()
    appStore.showSuccess(t('admin.accounts.setPrivacySuccess'))
  } catch (error: any) {
    console.error('Failed to set privacy:', error)
    appStore.showError(error?.message || t('admin.accounts.setPrivacyFailed'))
  }
}
const handleRecoverState = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.recoverState(a.id)
    patchAccountInList(updated)
    refreshAccountSummarySafe()
    enterAutoRefreshSilentWindow()
    if (a.lifecycle_state === 'archived' || updated.lifecycle_state === 'archived') {
      refreshArchivedPanel()
    }
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
    refreshAccountSummarySafe()
    enterAutoRefreshSilentWindow()
    if (a.lifecycle_state === 'archived' || updated.lifecycle_state === 'archived') {
      refreshArchivedPanel()
    }
    appStore.showSuccess(t('common.success'))
  } catch (error) {
    console.error('Failed to reset quota:', error)
  }
}
const handleBlacklistAccount = async (a: Account) => {
  if (!window.confirm(t('admin.accounts.blacklist.addConfirm', { name: a.name }))) {
    return
  }
  try {
    const updated = await adminAPI.accounts.blacklist(a.id)
    patchAccountInList(updated)
    refreshAccountSummarySafe()
    enterAutoRefreshSilentWindow()
    appStore.showSuccess(t('admin.accounts.blacklist.addSuccess'))
  } catch (error: any) {
    console.error('Failed to blacklist account:', error)
    appStore.showError(error?.message || t('admin.accounts.blacklist.addFailed'))
  }
}
const handleTestBlacklistAccount = async (payload: {
  account: Account
  source: 'test_modal'
  feedback?: BlacklistFeedbackPayload
}) => {
  try {
    const updated = await adminAPI.accounts.blacklist(payload.account.id, {
      source: payload.source,
      feedback: payload.feedback
    })
    patchAccountInList(updated)
    refreshAccountSummarySafe()
    enterAutoRefreshSilentWindow()
    appStore.showSuccess(t('admin.accounts.blacklist.addSuccess'))
    await closeTestModal()
  } catch (error: any) {
    console.error('Failed to blacklist account from test modal:', error)
    appStore.showError(error?.message || t('admin.accounts.blacklist.addFailed'))
  }
}
const handleDelete = (a: Account) => { deletingAcc.value = a; showDeleteDialog.value = true }
const confirmDelete = async () => {
  if (!deletingAcc.value) return
  const deletingArchived = deletingAcc.value.lifecycle_state === 'archived'
  try {
    await adminAPI.accounts.delete(deletingAcc.value.id)
    showDeleteDialog.value = false
    deletingAcc.value = null
    if (deletingArchived) {
      await refreshListAndArchivedPanel()
      return
    }
    await reload()
  } catch (error) {
    console.error('Failed to delete account:', error)
  }
}
const handleToggleSchedulable = async (a: Account) => {
  const nextSchedulable = !a.schedulable
  togglingSchedulable.value = a.id
  try {
    const updated = await adminAPI.accounts.setSchedulable(a.id, nextSchedulable)
    updateSchedulableInList([a.id], updated?.schedulable ?? nextSchedulable)
    refreshAccountSummarySafe()
    enterAutoRefreshSilentWindow()
    if (a.lifecycle_state === 'archived' || updated.lifecycle_state === 'archived') {
      refreshArchivedPanel()
    }
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
  refreshAccountSummarySafe()
  enterAutoRefreshSilentWindow()
  if (updated.lifecycle_state === 'archived') {
    refreshArchivedPanel()
  }
}

const loadRuntimeOptions = async () => {
  try {
    const [p, g] = await Promise.all([adminAPI.proxies.getAll(), adminAPI.groups.getAll()])
    proxies.value = p
    groups.value = g
  } catch (error) {
    console.error('Failed to load proxies/groups:', error)
  }
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
  await loadRuntimeOptions()
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
