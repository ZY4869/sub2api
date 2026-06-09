import type { Account, ToastDetailItem } from '@/types'
import type { BulkUpdateAccountsFilters } from '@/api/admin/accounts'
import {
  canAccountFetchUsage,
  invalidateAccountUsagePresentationCache,
  refreshAccountUsagePresentation,
  resolveActualUsageRefreshLoadOptions,
} from '@/composables/useAccountUsagePresentation'
import { adminAPI } from '@/api/admin'

interface OpenBulkEditFilteredModalOptions {
  excludeGrouped?: boolean
}

export function useAccountsBulkActions(ctx: any) {
  const {
    accounts,
    appStore,
    batchTestAccounts,
    batchTestDefaultModelStrategy,
    batchTestDefaultTestMode,
    bulkEditFilters,
    bulkEditFiltersTotal,
    clearSelection,
    load,
    params,
    reload,
    selIds,
    selPlatforms,
    showArchiveSelected,
    showBatchTest,
    showBulkEdit,
    setSelectedIds,
    t,
    refreshAccountSummarySafe,
    usageRefreshing
  } = ctx

const handleBulkDelete = async () => {
  if (!confirm(t("common.confirm"))) return;
  try {
    await Promise.all(selIds.value.map((id: number) => adminAPI.accounts.delete(id)));
    clearSelection();
    reload();
  } catch (error) {
    console.error("Failed to bulk delete accounts:", error);
  }
};
const openArchiveSelectedModal = () => {
  if (selIds.value.length === 0) {
    return;
  }
  if (selPlatforms.value.length !== 1) {
    appStore.showWarning(
      t("admin.accounts.bulkActions.archiveMixedPlatformDisabled"),
    );
    return;
  }
  showArchiveSelected.value = true;
};

const toOptionalString = (value: unknown) => {
  const v = String(value || "").trim();
  return v ? v : undefined;
};

const buildBulkEditFiltersFromParams = (): BulkUpdateAccountsFilters => ({
  platform: toOptionalString(params.platform),
  type: toOptionalString(params.type),
  status: toOptionalString(params.status),
  group: toOptionalString(params.group),
  search: toOptionalString(params.search),
  lifecycle: toOptionalString(params.lifecycle),
  privacy_mode: toOptionalString(params.privacy_mode),
  limited_view: toOptionalString(params.limited_view),
  limited_reason: toOptionalString(params.limited_reason),
  runtime_view: toOptionalString(params.runtime_view),
});

const openBulkEditSelectedModal = () => {
  bulkEditFilters.value = null;
  bulkEditFiltersTotal.value = null;
  showBulkEdit.value = true;
};

const resolveFilteredBulkEditTotal = async (filters: BulkUpdateAccountsFilters) => {
  const response = await adminAPI.accounts.list(1, 1, filters)
  return Number(response.total || 0)
}

const openBulkEditFilteredModal = async (
  options: OpenBulkEditFilteredModalOptions = {},
) => {
  const filters = buildBulkEditFiltersFromParams();
  if (options.excludeGrouped) {
    if (filters.group && filters.group !== "ungrouped") {
      appStore.showWarning(t("admin.accounts.bulkEdit.excludeGroupedSpecificGroupDisabled"));
    } else {
      filters.group = "ungrouped";
    }
  }

  let targetTotal = 0;
  try {
    targetTotal = await resolveFilteredBulkEditTotal(filters);
  } catch (error) {
    console.error("Failed to resolve filtered bulk edit targets:", error);
    appStore.showError(t("admin.accounts.bulkEdit.resolveTargetsFailed"));
    return;
  }

  if (targetTotal <= 0) {
    appStore.showWarning(t("admin.accounts.bulkEdit.noFilteredTargets"));
    return;
  }
  bulkEditFilters.value = filters;
  bulkEditFiltersTotal.value = targetTotal;
  showBulkEdit.value = true;
};

const closeBulkEditModal = () => {
  showBulkEdit.value = false;
  bulkEditFilters.value = null;
  bulkEditFiltersTotal.value = null;
};
const resolveAccountsByID = (accountIDs: number[]) => {
  if (accountIDs.length === 0) {
    return [] as Account[];
  }
  const accountByID = new Map(
    accounts.value.map((account: Account) => [account.id, account]),
  );
  return accountIDs
    .map((accountID) => accountByID.get(accountID))
    .filter((account): account is Account => Boolean(account));
};
const openBatchTestModal = (
  targetAccounts: Account[],
  options: {
    testMode?: "real_forward" | "health_check";
    modelStrategy?: "auto" | "specified";
  } = {},
) => {
  if (targetAccounts.length === 0) {
    appStore.showWarning(t("admin.accounts.batchTest.noTargets"));
    return;
  }
  batchTestAccounts.value = targetAccounts;
  batchTestDefaultTestMode.value = options.testMode ?? "health_check";
  batchTestDefaultModelStrategy.value = options.modelStrategy ?? "auto";
  showBatchTest.value = true;
};
const handleOpenBatchTestForSelection = () => {
  openBatchTestModal(resolveAccountsByID(selIds.value), {
    testMode: "health_check",
    modelStrategy: "auto",
  });
};
const handleRefreshActualUsage = async () => {
  if (usageRefreshing.value) return;

  const visibleAccounts = accounts.value.filter(canAccountFetchUsage);
  if (visibleAccounts.length === 0) {
    appStore.showWarning(t("admin.accounts.refreshActualUsageNoAccounts"));
    return;
  }

  usageRefreshing.value = true;
  invalidateAccountUsagePresentationCache(
    visibleAccounts.map((account: Account) => account.id),
  );

  try {
    const result = await refreshAccountUsagePresentation(visibleAccounts, {
      force: true,
      concurrency: 4,
      resolveLoadOptions: resolveActualUsageRefreshLoadOptions,
    });
    const toastDetails: ToastDetailItem[] = [];
    if (result.activeSuccess > 0) {
      toastDetails.push({
        text: t("admin.accounts.refreshActualUsageDetailActive", {
          count: result.activeSuccess,
        }),
        tone: "success",
      });
    }
    if (result.fallbackSuccess > 0) {
      toastDetails.push({
        text: t("admin.accounts.refreshActualUsageDetailFallback", {
          count: result.fallbackSuccess,
        }),
        tone: "warning",
      });
    }
    if (result.failed > 0) {
      toastDetails.push({
        text: t("admin.accounts.refreshActualUsageDetailFailed", {
          count: result.failed,
        }),
        tone: "error",
      });
    }

    if (result.failed > 0 && result.success > 0) {
      appStore.showWarning(
        t("admin.accounts.refreshActualUsagePartial", {
          success: result.success,
          failed: result.failed,
          live: result.activeSuccess,
          fallback: result.fallbackSuccess,
        }),
        { details: toastDetails },
      );
      return;
    }

    if (result.failed > 0) {
      appStore.showError(
        t("admin.accounts.refreshActualUsageFailedCount", {
          failed: result.failed,
        }),
        { details: toastDetails },
      );
      return;
    }

    appStore.showSuccess(
      t("admin.accounts.refreshActualUsageSuccess", {
        count: result.success,
        live: result.activeSuccess,
        fallback: result.fallbackSuccess,
      }),
      { details: toastDetails },
    );
  } catch (error: any) {
    console.error("Failed to refresh actual account usage:", error);
    appStore.showError(
      error?.message || t("admin.accounts.refreshActualUsageFailed"),
    );
  } finally {
    usageRefreshing.value = false;
  }
};
const handleBulkResetStatus = async () => {
  if (!confirm(t("common.confirm"))) return;
  try {
    const result = await adminAPI.accounts.batchClearError(selIds.value);
    if (result.failed > 0) {
      appStore.showError(
        t("admin.accounts.bulkActions.partialSuccess", {
          success: result.success,
          failed: result.failed,
        }),
      );
    } else {
      appStore.showSuccess(
        t("admin.accounts.bulkActions.resetStatusSuccess", {
          count: result.success,
        }),
      );
      clearSelection();
    }
    reload();
  } catch (error) {
    console.error("Failed to bulk reset status:", error);
    appStore.showError(String(error));
  }
};
const handleBulkRefreshToken = async () => {
  if (!confirm(t("common.confirm"))) return;
  try {
    const result = await adminAPI.accounts.batchRefresh(selIds.value);
    if (result.failed > 0) {
      appStore.showError(
        t("admin.accounts.bulkActions.partialSuccess", {
          success: result.success,
          failed: result.failed,
        }),
      );
    } else {
      appStore.showSuccess(
        t("admin.accounts.bulkActions.refreshTokenSuccess", {
          count: result.success,
        }),
      );
      clearSelection();
    }
    reload();
  } catch (error) {
    console.error("Failed to bulk refresh token:", error);
    appStore.showError(String(error));
  }
};
const updateSchedulableInList = (
  accountIds: number[],
  schedulable: boolean,
) => {
  if (accountIds.length === 0) return;
  const idSet = new Set(accountIds);
  accounts.value = accounts.value.map((account: Account) =>
    idSet.has(account.id) ? { ...account, schedulable } : account,
  );
};
const normalizeBulkSchedulableResult = (
  result: {
    success?: number;
    failed?: number;
    success_ids?: number[];
    failed_ids?: number[];
    results?: Array<{ account_id: number; success: boolean }>;
  },
  accountIds: number[],
) => {
  const responseSuccessIds = Array.isArray(result.success_ids)
    ? result.success_ids
    : [];
  const responseFailedIds = Array.isArray(result.failed_ids)
    ? result.failed_ids
    : [];
  if (responseSuccessIds.length > 0 || responseFailedIds.length > 0) {
    return {
      successIds: responseSuccessIds,
      failedIds: responseFailedIds,
      successCount:
        typeof result.success === "number"
          ? result.success
          : responseSuccessIds.length,
      failedCount:
        typeof result.failed === "number"
          ? result.failed
          : responseFailedIds.length,
      hasIds: true,
      hasCounts: true,
    };
  }

  const results = Array.isArray(result.results) ? result.results : [];
  if (results.length > 0) {
    const successIds = results
      .filter((item) => item.success)
      .map((item) => item.account_id);
    const failedIds = results
      .filter((item) => !item.success)
      .map((item) => item.account_id);
    return {
      successIds,
      failedIds,
      successCount:
        typeof result.success === "number" ? result.success : successIds.length,
      failedCount:
        typeof result.failed === "number" ? result.failed : failedIds.length,
      hasIds: true,
      hasCounts: true,
    };
  }

  const hasExplicitCounts =
    typeof result.success === "number" || typeof result.failed === "number";
  const successCount = typeof result.success === "number" ? result.success : 0;
  const failedCount = typeof result.failed === "number" ? result.failed : 0;
  if (
    hasExplicitCounts &&
    failedCount === 0 &&
    successCount === accountIds.length &&
    accountIds.length > 0
  ) {
    return {
      successIds: accountIds,
      failedIds: [],
      successCount,
      failedCount,
      hasIds: true,
      hasCounts: true,
    };
  }

  return {
    successIds: [],
    failedIds: [],
    successCount,
    failedCount,
    hasIds: false,
    hasCounts: hasExplicitCounts,
  };
};

const handleBulkToggleSchedulable = async (schedulable: boolean) => {
  const accountIds = [...selIds.value];
  try {
    const result = await adminAPI.accounts.bulkUpdate(accountIds, {
      schedulable,
    });
    const {
      successIds,
      failedIds,
      successCount,
      failedCount,
      hasIds,
      hasCounts,
    } = normalizeBulkSchedulableResult(result, accountIds);
    if (!hasIds && !hasCounts) {
      appStore.showError(t("admin.accounts.bulkSchedulableResultUnknown"));
      setSelectedIds(accountIds);
      load().catch((error: unknown) => {
        console.error("Failed to refresh accounts:", error);
      });
      return;
    }
    if (successIds.length > 0) {
      updateSchedulableInList(successIds, schedulable);
    }
    if (successCount > 0 && failedCount === 0) {
      const message = schedulable
        ? t("admin.accounts.bulkSchedulableEnabled", { count: successCount })
        : t("admin.accounts.bulkSchedulableDisabled", { count: successCount });
      appStore.showSuccess(message);
    }
    if (failedCount > 0) {
      const message =
        hasCounts || hasIds
          ? t("admin.accounts.bulkSchedulablePartial", {
              success: successCount,
              failed: failedCount,
            })
          : t("admin.accounts.bulkSchedulableResultUnknown");
      appStore.showError(message);
      setSelectedIds(failedIds.length > 0 ? failedIds : accountIds);
    } else {
      if (hasIds) clearSelection();
      else setSelectedIds(accountIds);
    }
    refreshAccountSummarySafe();
  } catch (error) {
    console.error("Failed to bulk toggle schedulable:", error);
    appStore.showError(t("common.error"));
  }
};


  return {
    handleBulkDelete,
    openArchiveSelectedModal,
    toOptionalString,
    buildBulkEditFiltersFromParams,
    openBulkEditSelectedModal,
    openBulkEditFilteredModal,
    closeBulkEditModal,
    resolveAccountsByID,
    openBatchTestModal,
    handleOpenBatchTestForSelection,
    handleRefreshActualUsage,
    handleBulkResetStatus,
    handleBulkRefreshToken,
    updateSchedulableInList,
    normalizeBulkSchedulableResult,
    handleBulkToggleSchedulable
  }
}
