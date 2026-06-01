<template>
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
        :actual-usage-refresh-summary="actualUsageRefreshSummary"
        :daily-5-h-trigger-enabled="daily5HTriggerSettingsView.settings.enabled"
        :daily-5-h-trigger-busy="daily5HTriggerSettingsLoading || daily5HTriggerSettingsSaving"
        :account-realtime-countdown-enabled="authStore.user?.account_realtime_countdown_enabled !== false"
        :account-visual-preset-override="accountVisualPresetOverride"
        :visual-style="resolvedAccountVisualPreset"
        :account-visual-style-updating="updatingAccountVisualStyle"
        :account-today-stats-windows="accountTodayStatsWindows"
        :account-group-display-mode="accountGroupDisplayMode"
        :account-display-preferences-updating="updatingAccountDisplayPreferences"
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
        @bulk-edit-filtered="openBulkEditFilteredModal"
        @toggle-daily-5h-trigger="handleToggleDaily5HTrigger"
        @open-daily-5h-settings="handleOpenDaily5HTriggerSettings"
        @toggle-account-realtime-countdown="handleToggleAccountRealtimeCountdown"
        @set-account-visual-preset-override="setAccountVisualPresetOverride"
        @save-account-display-preferences="setAccountDisplayPreferences"
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
          @batch-test="handleOpenBatchTestForSelection"
          @delete="handleBulkDelete"
          @reset-status="handleBulkResetStatus"
          @refresh-token="handleBulkRefreshToken"
          @edit="openBulkEditSelectedModal"
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
            :visual-style="resolvedAccountVisualPreset"
            :white-surface-enabled="airyWhiteSurfaceEnabled"
            :account-today-stats-windows="accountTodayStatsWindows"
            :account-group-display-mode="accountGroupDisplayMode"
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
            :visual-style="resolvedAccountVisualPreset"
            :white-surface-enabled="airyWhiteSurfaceEnabled"
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
            :visual-style="resolvedAccountVisualPreset"
            :white-surface-enabled="airyWhiteSurfaceEnabled"
            :account-today-stats-windows="accountTodayStatsWindows"
            :account-group-display-mode="accountGroupDisplayMode"
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
    :edit-loading="editLoading"
    :show-sync="showSync"
    :show-import-data="showImportData"
    :show-export-data-dialog="showExportDataDialog"
    :show-bulk-edit="showBulkEdit"
    :show-temp-unsched="showTempUnsched"
    :show-delete-dialog="showDeleteDialog"
    :show-re-auth="showReAuth"
    :show-test="showTest"
    :show-batch-test="showBatchTest"
    :show-stats="showStats"
    :show-model-diagnostics="showModelDiagnostics"
    :show-error-passthrough="showErrorPassthrough"
    :show-tls-fingerprint-profiles="showTLSFingerprintProfiles"
    :show-schedule-panel="showSchedulePanel"
    :proxies="proxies"
    :groups="groups"
    :bulk-edit-filters="bulkEditFilters"
    :bulk-edit-filters-total="bulkEditFiltersTotal"
    :selected-ids="selIds"
    :selected-platforms="bulkEditSelectedPlatforms"
    :selected-types="bulkEditSelectedTypes"
    :editing-account="edAcc"
    :temp-unsched-account="tempUnschedAcc"
    :deleting-account="deletingAcc"
    :re-auth-account="reAuthAcc"
    :testing-account="testingAcc"
    :batch-test-accounts="batchTestAccounts"
    :batch-test-default-test-mode="batchTestDefaultTestMode"
    :batch-test-default-model-strategy="batchTestDefaultModelStrategy"
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
    @close-edit="handleCloseEdit"
    @updated="handleAccountUpdated"
    @close-reauth="closeReAuthModal"
    @close-test="closeTestModal"
    @close-batch-test="closeBatchTestModal"
    @batch-test-completed="handleBatchTestCompleted"
    @close-stats="closeStatsModal"
    @close-model-diagnostics="closeModelDiagnostics"
    @close-schedule="closeSchedulePanel"
    @close-menu="closeMenu"
    @test="handleTest"
    @quick-test="handleQuickTest"
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
    @close-bulk-edit="closeBulkEditModal"
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
  <AccountDaily5HTriggerSettingsDialog
    :show="showDaily5HTriggerSettings"
    :saving="daily5HTriggerSettingsSaving"
    :settings="daily5HTriggerSettingsView.settings"
    :candidates="daily5HTriggerSettingsView.candidates"
    @close="showDaily5HTriggerSettings = false"
    @save="handleSaveDaily5HTriggerSettings"
  />
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useAppStore } from "@/stores/app";
import { useAuthStore } from "@/stores/auth";
import { useModelInventoryStore } from "@/stores";
import { invalidateModelRegistry } from "@/stores/modelRegistry";
import { adminAPI } from "@/api/admin";
import { useAccountStatusSummary } from "@/composables/useAccountStatusSummary";
import { useAccountActionMenu } from "@/composables/useAccountActionMenu";
import { useAccountViewMode } from "@/composables/useAccountViewMode";
import { useAccountsRuntimeSummary } from "@/composables/useAccountsRuntimeSummary";
import { useAccountsViewLiveSync } from "@/composables/useAccountsViewLiveSync";
import { useAccountsViewListPatching } from "@/composables/useAccountsViewListPatching";
import { useTableLoader } from "@/composables/useTableLoader";
import { useSwipeSelect } from "@/composables/useSwipeSelect";
import { useTableSelection } from "@/composables/useTableSelection";
import { useAccountVisualStylePreference } from "@/composables/useAccountVisualStylePreference";
import { useAccountDisplayPreferences } from "@/composables/useAccountDisplayPreferences";
import TablePageLayout from "@/components/layout/TablePageLayout.vue";
import Pagination from "@/components/common/Pagination.vue";
import AccountCardGrid from "@/components/admin/account/AccountCardGrid.vue";
import AccountBulkActionsBar from "@/components/admin/account/AccountBulkActionsBar.vue";
import AccountGroupedView from "@/components/admin/account/AccountGroupedView.vue";
import AccountLimitedSummaryBar from "@/components/admin/account/AccountLimitedSummaryBar.vue";
import AccountPlatformTabs from "@/components/admin/account/AccountPlatformTabs.vue";
import AccountStatusSummaryBar from "@/components/admin/account/AccountStatusSummaryBar.vue";
import AccountDaily5HTriggerSettingsDialog from "@/components/admin/account/AccountDaily5HTriggerSettingsDialog.vue";
import AccountsViewDialogsHost from "@/components/admin/account/AccountsViewDialogsHost.vue";
import AccountsViewTable from "@/components/admin/account/AccountsViewTable.vue";
import AccountsViewToolbar from "@/components/admin/account/AccountsViewToolbar.vue";
import { useAccountsColumnPreferences } from './accounts/useAccountsColumnPreferences';
import { useAccountsColumns } from './accounts/useAccountsColumns';
import { useAccountsDaily5HTrigger } from './accounts/useAccountsDaily5HTrigger';
import { useAccountsEditActions } from './accounts/useAccountsEditActions';
import { useAccountsBulkActions } from './accounts/useAccountsBulkActions';
import { useAccountsRowActions } from './accounts/useAccountsRowActions';
import { useAccountsDialogState } from './accounts/useAccountsDialogState';
import { useAccountsDataActions } from './accounts/useAccountsDataActions';
import {
  canAccountFetchUsage,
  resolveActualUsageRefreshLoadOptions,
} from "@/composables/useAccountUsagePresentation";
import { useModelImportExposureSync } from "@/composables/useModelImportExposureSync";
import {
  buildAccountModelImportToastPayload,
  resolveAccountModelImportErrorMessage,
  shouldInvalidateModelInventory,
} from "@/utils/accountModelImport";
import { getPlatformOrderIndex } from "@/utils/platformBranding";
import { buildProviderDisplayName } from "@/utils/providerLabels";
import type { AccountListRequestParams } from "@/utils/accountListSync";
import type {
  Account,
  AccountRateLimitReason,
  AccountPlatform,
  AccountRuntimeView,
  Proxy as AccountProxy,
  AdminGroup,
} from "@/types";

type ActualUsageRefreshSummary = {
  total: number;
  live: number;
  fallback: number;
};

const props = withDefaults(
  defineProps<{
    limitedMode?: boolean;
  }>(),
  {
    limitedMode: false,
  },
);

const limitedMode = computed(() => props.limitedMode);
const { t } = useI18n();
const router = useRouter();
const appStore = useAppStore();
const authStore = useAuthStore();
const {
  accountVisualPresetOverride,
  resolvedAccountVisualPreset,
  updatingAccountVisualStyle,
  setAccountVisualPresetOverride,
} = useAccountVisualStylePreference();
const {
  accountTodayStatsWindows,
  accountGroupDisplayMode,
  updatingAccountDisplayPreferences,
  setAccountDisplayPreferences,
} = useAccountDisplayPreferences();
const airyWhiteSurfaceEnabled = computed(
  () => resolvedAccountVisualPreset.value === "airy" && appStore.accountAiryWhiteSurfaceEnabled,
);
const modelInventoryStore = useModelInventoryStore();
const {
  syncDialogOpen,
  syncDialogModels,
  syncDialogSubmitting,
  handleImportedModels,
  closeSyncDialog,
  submitSyncDialog,
} = useModelImportExposureSync({ t, appStore, modelInventoryStore });

const proxies = ref<AccountProxy[]>([]);
const groups = ref<AdminGroup[]>([]);
const accountTableRef = ref<HTMLElement | null>(null);
const importingModelsAccountId = ref<number | null>(null);
const { viewMode, groupViewEnabled } = useAccountViewMode();
const { menu, openMenu, closeMenu, syncMenuAccount, clearMenuAccount } =
  useAccountActionMenu();

const {
  hiddenColumns,
  ACCOUNT_SORT_STORAGE_KEY,
  hideLimitedAccounts,
  platformCountSortOrder,
  handlePlatformCountSortOrderUpdate,
  saveHideLimitedPreference,
  loadHideLimitedPreference,
  toggleColumn,
} = useAccountsColumnPreferences({
  getLimitedView: () => !limitedMode.value && String(params.limited_view || ""),
  refreshTodayStats: () => refreshTodayStats(),
});

const {
  items: accounts,
  loading,
  params,
  pagination,
  load: baseLoad,
  reload: baseReload,
  debouncedReload: baseDebouncedReload,
  handlePageChange: baseHandlePageChange,
  handlePageSizeChange: baseHandlePageSizeChange,
} = useTableLoader<Account, AccountListRequestParams>({
  fetchFn: adminAPI.accounts.list,
  initialParams: {
    platform: "",
    type: "",
    status: "",
    group: "",
    privacy_mode: "",
    search: "",
    lifecycle: "normal",
    limited_view: limitedMode.value
      ? "limited_only"
      : loadHideLimitedPreference()
        ? "normal_only"
        : "all",
    limited_reason: "",
    runtime_view: "all",
  },
});

const normalizeAccountFilters = (nextFilters: Record<string, unknown>) => {
  const normalized = { ...nextFilters };
  if (
    !limitedMode.value &&
    typeof normalized.status !== "undefined" &&
    String(normalized.status || "")
  ) {
    normalized.runtime_view = "all";
  }
  return normalized;
};

const resetVisibleTableState = () => {
  clearSelection();
  closeMenu();
  pagination.page = 1;
};

const handleFilterUpdate = (newFilters: Record<string, unknown>) => {
  Object.assign(params, normalizeAccountFilters(newFilters));
  resetVisibleTableState();
};

const summaryParams = computed<AccountListRequestParams>(() => ({
  platform: String(params.platform || ""),
  type: String(params.type || ""),
  group: String(params.group || ""),
  privacy_mode: String(params.privacy_mode || ""),
  search: String(params.search || ""),
  lifecycle: String(params.lifecycle || ""),
  limited_view: limitedMode.value ? "limited_only" : "all",
  limited_reason: limitedMode.value ? "" : String(params.limited_reason || ""),
}));

const {
  summary: accountSummaryState,
  loading: summaryLoading,
  error: summaryError,
  refresh: refreshAccountSummary,
} = useAccountStatusSummary(summaryParams);

const accountSummary = computed(() => accountSummaryState.value);
const activeLimitedReason = computed<AccountRateLimitReason | "">(() => {
  const value = String(params.limited_reason || "");
  return value === "rate_429" ||
    value === "usage_5h" ||
    value === "usage_7d" ||
    value === "usage_7d_all"
    ? value
    : "";
});

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
  selectVisible: selectPage,
} = useTableSelection<Account>({
  rows: accounts,
  getId: (account) => account.id,
});

const {
  showCreate, showArchiveSelected, showEdit, editLoading, showSync,
  showImportData, showExportDataDialog, includeProxyOnExport, showBulkEdit,
  bulkEditFilters, bulkEditFiltersTotal, showTempUnsched, showDeleteDialog,
  showReAuth, showTest, showBatchTest, showStats, showModelDiagnostics,
  showErrorPassthrough, showTLSFingerprintProfiles, showDaily5HTriggerSettings,
  edAcc, tempUnschedAcc, deletingAcc, reAuthAcc, testingAcc,
  batchTestAccounts, batchTestDefaultTestMode, batchTestDefaultModelStrategy,
  statsAcc, diagnosticsAccount, diagnosticsResult, diagnosticsLoading,
  showSchedulePanel, scheduleAcc, scheduleModelOptions, togglingSchedulable,
  activeEditRequestToken, activeEditAbortController,
  exportingData, usageManualRefreshToken, usageRefreshing,
  daily5HTriggerSettingsLoading, daily5HTriggerSettingsSaving,
  archivedPanelRefreshToken, selPlatforms, bulkEditSelectedPlatforms,
  bulkEditSelectedTypes,
} = useAccountsDialogState(accounts, isSelected);

const handleSearchQueryUpdate = (value: string) => {
  params.search = value;
  debouncedReload();
};

useSwipeSelect(accountTableRef, {
  isSelected,
  select,
  deselect,
});

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
    showBatchTest.value ||
    showStats.value ||
    showModelDiagnostics.value ||
    showSchedulePanel.value ||
    showErrorPassthrough.value ||
    showDaily5HTriggerSettings.value
  );
});
const isActionMenuOpen = computed(() => menu.show);
const syncAccountRefs = (nextAccount: Account) => {
  if (edAcc.value?.id === nextAccount.id) edAcc.value = nextAccount;
  if (reAuthAcc.value?.id === nextAccount.id) reAuthAcc.value = nextAccount;
  if (tempUnschedAcc.value?.id === nextAccount.id)
    tempUnschedAcc.value = nextAccount;
  if (deletingAcc.value?.id === nextAccount.id) deletingAcc.value = nextAccount;
  if (testingAcc.value?.id === nextAccount.id) testingAcc.value = nextAccount;
  if (statsAcc.value?.id === nextAccount.id) statsAcc.value = nextAccount;
  if (diagnosticsAccount.value?.id === nextAccount.id)
    diagnosticsAccount.value = nextAccount;
  if (scheduleAcc.value?.id === nextAccount.id) scheduleAcc.value = nextAccount;
  if (
    batchTestAccounts.value.some((account) => account.id === nextAccount.id)
  ) {
    batchTestAccounts.value = batchTestAccounts.value.map((account) =>
      account.id === nextAccount.id ? nextAccount : account,
    );
  }
  syncMenuAccount(nextAccount);
};

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
  resumeAutoRefresh,
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
  onListChanged: refreshAccountSummary,
});

const isLiveSyncBlocked = computed(
  () => loading.value || isAnyModalOpen.value || isActionMenuOpen.value,
);
const {
  daily5HTriggerSettingsView,
  handleToggleDaily5HTrigger,
  handleSaveDaily5HTriggerSettings,
  handleOpenDaily5HTriggerSettings,
  handleToggleAccountRealtimeCountdown,
  loadDaily5HTriggerSettings,
  applyDaily5HTriggerSettingsView,
} = useAccountsDaily5HTrigger({
  appStore, authStore, t, showDaily5HTriggerSettings,
  daily5HTriggerSettingsLoading, daily5HTriggerSettingsSaving,
});

const pendingRuntimeListRefresh = ref(false);
const runtimeSummaryParams = computed<AccountListRequestParams>(() => ({
  platform: String(params.platform || ""),
  type: String(params.type || ""),
  group: String(params.group || ""),
  privacy_mode: String(params.privacy_mode || ""),
  search: String(params.search || ""),
  lifecycle: String(params.lifecycle || ""),
  limited_view: limitedMode.value ? "limited_only" : "all",
  limited_reason: limitedMode.value ? "" : String(params.limited_reason || ""),
  runtime_view: String(params.runtime_view || "all"),
}));
const triggerRuntimeInUseRefresh = async () => {
  if (
    limitedMode.value ||
    String(params.runtime_view || "all") !== "in_use_only"
  ) {
    pendingRuntimeListRefresh.value = false;
    return;
  }
  if (isLiveSyncBlocked.value) {
    pendingRuntimeListRefresh.value = true;
    return;
  }
  pendingRuntimeListRefresh.value = false;
  await refreshAccountsIncrementally();
};
const { summary: runtimeSummaryState, refresh: refreshRuntimeSummary } =
  useAccountsRuntimeSummary(runtimeSummaryParams, {
    enabled: computed(() => !limitedMode.value),
    onSummaryChanged: async () => {
      await triggerRuntimeInUseRefresh();
    },
  });
const toolbarSummary = computed(() => ({
  ...accountSummary.value,
  in_use: limitedMode.value ? 0 : runtimeSummaryState.value.in_use,
  remaining_available: limitedMode.value
    ? 0
    : Math.max(
        accountSummary.value.remaining_available +
          accountSummary.value.in_use -
          runtimeSummaryState.value.in_use,
        0,
      ),
}));
const displayAccounts = computed<Account[]>(() => {
  const pagePlatformCounts = accounts.value.reduce<
    Partial<Record<AccountPlatform, number>>
  >((acc, account) => {
    acc[account.platform] = (acc[account.platform] ?? 0) + 1;
    return acc;
  }, {});

  return accounts.value
    .map((account, index) => ({
      account,
      index,
      count: pagePlatformCounts[account.platform] ?? 0,
      platformRank: getPlatformOrderIndex(account.platform),
    }))
    .sort((left, right) => {
      if (left.count !== right.count) {
        return platformCountSortOrder.value === "count_desc"
          ? right.count - left.count
          : left.count - right.count;
      }

      if (left.platformRank !== right.platformRank) {
        return left.platformRank - right.platformRank;
      }

      return left.index - right.index;
    })
    .map((item) => item.account);
});
const actualUsageRefreshSummary = computed<ActualUsageRefreshSummary>(() => {
  const refreshableAccounts = accounts.value.filter(canAccountFetchUsage);
  const live = refreshableAccounts.reduce((count, account) => {
    return resolveActualUsageRefreshLoadOptions(account).source === "active"
      ? count + 1
      : count;
  }, 0);

  return {
    total: refreshableAccounts.length,
    live,
    fallback: Math.max(refreshableAccounts.length - live, 0),
  };
});
const limitedAccountsCount = computed(
  () => accountSummary.value.limited_breakdown.total,
);

watch(isLiveSyncBlocked, (blocked, wasBlocked) => {
  if (wasBlocked && !blocked && pendingRuntimeListRefresh.value) {
    triggerRuntimeInUseRefresh().catch((error) => {
      console.error(
        "Failed to refresh in-use accounts after page became idle:",
        error,
      );
    });
  }
});

const handleManualRefresh = async () => {
  await load();
  await refreshRuntimeSummary(true);
  refreshArchivedPanel();
  usageManualRefreshToken.value += 1;
};

const handleSyncPendingListChanges = async () => {
  await syncPendingListChanges();
  await refreshRuntimeSummary(true);
  refreshArchivedPanel();
  usageManualRefreshToken.value += 1;
};

const { toggleableColumns, cols } = useAccountsColumns({
  t, authStore, hiddenColumns, resolvedAccountVisualPreset, accountGroupDisplayMode,
});
const { patchAccountInList } = useAccountsViewListPatching({
  accounts,
  params,
  pagination,
  hasPendingListSync,
  removeSelectedAccounts,
  syncAccountRefs,
  clearRemovedAccount: clearMenuAccount,
});

const refreshArchivedPanel = () => {
  archivedPanelRefreshToken.value += 1;
};

const refreshAccountSummarySafe = () => {
  refreshAccountSummary().catch((error) => {
    console.error("Failed to refresh account summary:", error);
  });
  refreshRuntimeSummary(true).catch((error) => {
    console.error("Failed to refresh account runtime summary:", error);
  });
};

const { handleCloseEdit, handleEdit } = useAccountsEditActions({
  adminAPI, appStore, t, edAcc, showEdit, editLoading, activeEditRequestToken,
  activeEditAbortController,
});
const handleOpenMenu = ({
  account,
  event,
}: {
  account: Account;
  event: MouseEvent;
}) => {
  openMenu({ account, event });
};
const toggleSelectAllVisible = (checked: boolean) => {
  toggleVisible(checked);
};
const handleToggleSectionSelected = ({
  ids,
  checked,
}: {
  ids: number[];
  checked: boolean;
}) => {
  ids.forEach((id) => {
    if (checked) {
      select(id);
      return;
    }
    deselect(id);
  });
};
const applyBoardSelection = (next: {
  platform?: string;
  status?: string;
  runtimeView?: string;
}) => {
  resetVisibleTableState();
  if (!limitedMode.value) {
    params.limited_view = hideLimitedAccounts.value ? "normal_only" : "all";
    params.limited_reason = "";
  }
  if (typeof next.platform !== "undefined") {
    params.platform = next.platform;
  }
  if (typeof next.status !== "undefined") {
    params.status = next.status;
  }
  if (typeof next.runtimeView !== "undefined") {
    params.runtime_view = next.runtimeView;
  }
  debouncedReload();
};

const handlePlatformTabChange = (value: string) => {
  applyBoardSelection({
    platform: value,
  });
};
const handleSummaryStatusSelect = (status: string) => {
  const nextStatus = String(params.status || "") === status ? "" : status;
  applyBoardSelection({
    status: nextStatus,
    runtimeView: limitedMode.value
      ? String(params.runtime_view || "all")
      : "all",
  });
};
const handleRuntimeViewSelect = (runtimeView: AccountRuntimeView | string) => {
  if (limitedMode.value) {
    return;
  }
  const nextRuntimeView =
    String(params.runtime_view || "all") === runtimeView ? "all" : runtimeView;
  applyBoardSelection({
    status: "",
    runtimeView: nextRuntimeView,
  });
};
const handleLimitedReasonSelect = (reason: AccountRateLimitReason | "") => {
  resetVisibleTableState();
  params.limited_reason = activeLimitedReason.value === reason ? "" : reason;
  debouncedReload();
};
const toggleHideLimitedAccounts = () => {
  resetVisibleTableState();
  const nextHidden = !hideLimitedAccounts.value;
  params.limited_view = nextHidden ? "normal_only" : "all";
  if (nextHidden && String(params.status || "") === "rate_limited") {
    params.status = "";
  }
  saveHideLimitedPreference(nextHidden);
  debouncedReload();
};
const openLimitedAccountsPage = () => {
  router.push({ path: "/admin/accounts/limited" }).catch((error) => {
    console.error("Failed to open limited accounts page:", error);
  });
};
const showStandalonePagination = computed(
  () => groupViewEnabled.value || viewMode.value === "card",
);
const {
  handleBulkDelete, openArchiveSelectedModal, openBulkEditSelectedModal, openBulkEditFilteredModal, closeBulkEditModal,
  openBatchTestModal, handleOpenBatchTestForSelection, handleRefreshActualUsage, handleBulkResetStatus,
  handleBulkRefreshToken, updateSchedulableInList, handleBulkToggleSchedulable,
} = useAccountsBulkActions({
  t, appStore, adminAPI, accounts, params, pagination, selIds, selPlatforms,
  bulkEditFilters, bulkEditFiltersTotal, showBulkEdit, showArchiveSelected,
  showBatchTest, batchTestAccounts, batchTestDefaultTestMode, batchTestDefaultModelStrategy,
  usageRefreshing, clearSelection, reload, load, setSelectedIds, refreshAccountSummarySafe,
  usageManualRefreshToken, canAccountFetchUsage, resolveActualUsageRefreshLoadOptions,
});

const refreshListAndArchivedPanel = async () => {
  refreshArchivedPanel();
  await reload();
};

const handleBulkUpdated = () => {
  closeBulkEditModal();
  clearSelection();
  reload();
};
const handleAccountUpdated = (updatedAccount: Account) => {
  const editedArchived =
    edAcc.value?.id === updatedAccount.id &&
    edAcc.value.lifecycle_state === "archived";
  patchAccountInList(updatedAccount);
  refreshAccountSummarySafe();
  enterAutoRefreshSilentWindow();
  if (
    editedArchived ||
    (updatedAccount.lifecycle_state &&
      updatedAccount.lifecycle_state !== "normal")
  ) {
    refreshArchivedPanel();
  }
};
const {
  handleReloadRequested, handleDataImported, handleCreated,
  handleArchivedAccounts, openExportDataDialog, handleExportData,
} = useAccountsDataActions({
  adminAPI, appStore, t, groups, reload, refreshArchivedPanel, showImportData,
  showCreate, showArchiveSelected, clearSelection, includeProxyOnExport,
  showExportDataDialog, exportingData, selIds, params,
});
const {
  closeTestModal, closeBatchTestModal, closeStatsModal, closeModelDiagnostics, closeReAuthModal, handleTest, handleQuickTest,
  handleBatchTestCompleted, handleViewStats, handleDiagnoseModels, refreshModelDiagnostics, handleSchedule, closeSchedulePanel,
  handleReAuth, handleRefresh, handleSetPrivacy, handleRecoverState, handleImportModels, handleResetQuota, handleBlacklistAccount,
  handleTestBlacklistAccount, handleDelete, confirmDelete, handleToggleSchedulable, handleShowTempUnsched, handleTempUnschedReset,
} = useAccountsRowActions({
  t, appStore, adminAPI, modelInventoryStore, invalidateModelRegistry, buildProviderDisplayName,
  buildAccountModelImportToastPayload, resolveAccountModelImportErrorMessage, shouldInvalidateModelInventory,
  handleImportedModels, openBatchTestModal, refreshListAndArchivedPanel, patchAccountInList, refreshAccountSummarySafe,
  enterAutoRefreshSilentWindow, updateSchedulableInList, reload, showTest, testingAcc, showBatchTest, batchTestAccounts,
  batchTestDefaultTestMode, batchTestDefaultModelStrategy, showStats, statsAcc, showModelDiagnostics, diagnosticsAccount,
  diagnosticsResult, diagnosticsLoading, showReAuth, reAuthAcc, scheduleAcc, scheduleModelOptions, showSchedulePanel,
  importingModelsAccountId, deletingAcc, showDeleteDialog, togglingSchedulable, tempUnschedAcc, showTempUnsched,
});

const loadRuntimeOptions = async () => {
  const [proxyResult, groupResult, daily5HResult] = await Promise.allSettled([
    adminAPI.proxies.getAll(),
    adminAPI.groups.getAll(),
    adminAPI.accounts.getDaily5HTriggerSettings(),
  ]);
  if (proxyResult.status === "fulfilled") {
    proxies.value = proxyResult.value;
  } else {
    console.error("Failed to load proxies:", proxyResult.reason);
  }
  if (groupResult.status === "fulfilled") {
    groups.value = groupResult.value;
  } else {
    console.error("Failed to load groups:", groupResult.reason);
  }
  if (daily5HResult.status === "fulfilled") {
    applyDaily5HTriggerSettingsView(daily5HResult.value);
  } else {
    console.error(
      "Failed to load daily 5H trigger settings:",
      daily5HResult.reason,
    );
  }
};

// Close the action menu on scroll, but keep the column settings dropdown open.
const handleScroll = () => {
  menu.show = false;
};

// Close the column settings dropdown when clicking outside.

onMounted(async () => {
  load().catch((error) => {
    console.error("Failed to load accounts:", error);
  });
  loadDaily5HTriggerSettings().catch((error) => {
    console.error("Failed to preload daily 5H trigger settings:", error);
  });
  await loadRuntimeOptions();
  window.addEventListener("scroll", handleScroll, true);

  if (autoRefreshEnabled.value) {
    resumeAutoRefresh();
  } else {
    pauseAutoRefresh();
  }
});

onUnmounted(() => {
  window.removeEventListener("scroll", handleScroll, true);
});
</script>
