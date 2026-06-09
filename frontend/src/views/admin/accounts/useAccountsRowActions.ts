import type { Account, ClaudeModel } from '@/types'
import type { BlacklistFeedbackPayload } from '@/api/admin/accounts'
import { adminAPI } from '@/api/admin'

export function useAccountsRowActions(ctx: any) {
  const {
    appStore,
    buildAccountModelImportToastPayload,
    buildProviderDisplayName,
    batchTestAccounts,
    batchTestDefaultModelStrategy,
    batchTestDefaultTestMode,
    deletingAcc,
    diagnosticsAccount,
    diagnosticsLoading,
    diagnosticsResult,
    enterAutoRefreshSilentWindow,
    handleImportedModels,
    invalidateModelRegistry,
    importingModelsAccountId,
    modelInventoryStore,
    openBatchTestModal,
    patchAccountInList,
    reAuthAcc,
    refreshAccountSummarySafe,
    refreshArchivedPanel,
    refreshListAndArchivedPanel,
    reload,
    scheduleAcc,
    scheduleModelOptions,
    showBatchTest,
    showDeleteDialog,
    showModelDiagnostics,
    showReAuth,
    showSchedulePanel,
    showStats,
    showTempUnsched,
    showTest,
    statsAcc,
    t,
    tempUnschedAcc,
    testingAcc,
    togglingSchedulable,
    updateSchedulableInList,
    resolveAccountModelImportErrorMessage,
    shouldInvalidateModelInventory
  } = ctx

const closeTestModal = async () => {
  showTest.value = false;
  testingAcc.value = null;
  await refreshListAndArchivedPanel();
};
const closeBatchTestModal = async () => {
  showBatchTest.value = false;
  batchTestAccounts.value = [];
  batchTestDefaultTestMode.value = "health_check";
  batchTestDefaultModelStrategy.value = "auto";
  await refreshListAndArchivedPanel();
};
const closeStatsModal = () => {
  showStats.value = false;
  statsAcc.value = null;
};
const closeModelDiagnostics = () => {
  showModelDiagnostics.value = false;
  diagnosticsAccount.value = null;
  diagnosticsResult.value = null;
  diagnosticsLoading.value = false;
};
const closeReAuthModal = () => {
  showReAuth.value = false;
  reAuthAcc.value = null;
};
const handleTest = (a: Account) => {
  testingAcc.value = a;
  showTest.value = true;
};
const handleQuickTest = (a: Account) => {
  openBatchTestModal([a], {
    testMode: "health_check",
    modelStrategy: "auto",
  });
};
const handleBatchTestCompleted = async () => {
  await refreshListAndArchivedPanel();
};
const handleViewStats = (a: Account) => {
  statsAcc.value = a;
  showStats.value = true;
};
const handleDiagnoseModels = async (a: Account) => {
  const sameAccount = diagnosticsAccount.value?.id === a.id;
  diagnosticsAccount.value = a;
  showModelDiagnostics.value = true;
  if (!sameAccount) {
    diagnosticsResult.value = null;
  }
  diagnosticsLoading.value = true;
  try {
    diagnosticsResult.value = await adminAPI.accounts.diagnoseAccountModels(
      a.id,
    );
  } catch (error: any) {
    console.error("Failed to diagnose account model exposure:", error);
    appStore.showError(
      error?.message || t("admin.accounts.modelDiagnostics.failed"),
    );
  } finally {
    diagnosticsLoading.value = false;
  }
};
const refreshModelDiagnostics = async () => {
  if (!diagnosticsAccount.value) {
    return;
  }
  diagnosticsLoading.value = true;
  try {
    diagnosticsResult.value = await adminAPI.accounts.diagnoseAccountModels(
      diagnosticsAccount.value.id,
      { refresh: true },
    );
  } catch (error: any) {
    console.error(
      "Failed to refresh account model exposure diagnostics:",
      error,
    );
    appStore.showError(
      error?.message || t("admin.accounts.modelDiagnostics.failed"),
    );
  } finally {
    diagnosticsLoading.value = false;
  }
};
const scheduleSourceProtocolLabel = (sourceProtocol?: string) => {
  switch (String(sourceProtocol || "").trim()) {
    case "openai":
      return t("admin.accounts.protocolGateway.protocolOptions.openai");
    case "anthropic":
      return t("admin.accounts.protocolGateway.protocolOptions.anthropic");
    case "gemini":
      return t("admin.accounts.protocolGateway.protocolOptions.gemini");
    default:
      return "";
  }
};
const handleSchedule = async (a: Account) => {
  scheduleAcc.value = a;
  scheduleModelOptions.value = [];
  showSchedulePanel.value = true;
  try {
    const models = await adminAPI.accounts.getAvailableModels(a.id);
    scheduleModelOptions.value = models.map((m: ClaudeModel) => {
      const sourceProtocol = String(m.source_protocol || "").trim();
      const protocolLabel = scheduleSourceProtocolLabel(sourceProtocol);
      const displayName = buildProviderDisplayName({
        provider: m.provider,
        providerLabel: m.provider_label,
        displayName: m.display_name,
        fallbackId: m.id,
      });
      return {
        value: `${sourceProtocol || "default"}::${m.id}`,
        label: protocolLabel
          ? `${displayName} · ${protocolLabel}`
          : displayName,
        model_id: m.id,
        source_protocol: sourceProtocol || undefined,
      };
    });
  } catch {
    scheduleModelOptions.value = [];
  }
};
const closeSchedulePanel = () => {
  showSchedulePanel.value = false;
  scheduleAcc.value = null;
  scheduleModelOptions.value = [];
};
const handleReAuth = (a: Account) => {
  reAuthAcc.value = a;
  showReAuth.value = true;
};
const handleRefresh = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.refreshCredentials(a.id);
    patchAccountInList(updated);
    refreshAccountSummarySafe();
    enterAutoRefreshSilentWindow();
    if (
      a.lifecycle_state === "archived" ||
      updated.lifecycle_state === "archived"
    ) {
      refreshArchivedPanel();
    }
  } catch (error) {
    console.error("Failed to refresh credentials:", error);
  }
};
const handleSetPrivacy = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.setPrivacy(a.id);
    patchAccountInList(updated);
    refreshAccountSummarySafe();
    enterAutoRefreshSilentWindow();
    appStore.showSuccess(t("admin.accounts.setPrivacySuccess"));
  } catch (error: any) {
    console.error("Failed to set privacy:", error);
    appStore.showError(error?.message || t("admin.accounts.setPrivacyFailed"));
  }
};
const handleRecoverState = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.recoverState(a.id);
    patchAccountInList(updated);
    refreshAccountSummarySafe();
    enterAutoRefreshSilentWindow();
    if (
      a.lifecycle_state === "archived" ||
      updated.lifecycle_state === "archived"
    ) {
      refreshArchivedPanel();
    }
    appStore.showSuccess(t("admin.accounts.recoverStateSuccess"));
  } catch (error: any) {
    console.error("Failed to recover account state:", error);
    appStore.showError(
      error?.message || t("admin.accounts.recoverStateFailed"),
    );
  }
};
const handleImportModels = async (
  a: Account,
  trigger: "manual" | "create" = "manual",
) => {
  if (importingModelsAccountId.value === a.id) {
    return;
  }
  importingModelsAccountId.value = a.id;
  appStore.showInfo(t("admin.accounts.probingModels"));
  try {
    const result = await adminAPI.accounts.importModels(a.id, { trigger });
    const toastPayload = buildAccountModelImportToastPayload(t, result);
    if (toastPayload.type === "error") {
      appStore.showError(toastPayload.message, toastPayload.options);
    } else if (toastPayload.type === "warning") {
      appStore.showWarning(toastPayload.message, toastPayload.options);
    } else {
      appStore.showSuccess(toastPayload.message, toastPayload.options);
    }
    if (shouldInvalidateModelInventory(result)) {
      invalidateModelRegistry();
      modelInventoryStore.invalidate();
    }
    handleImportedModels(result);
    return result;
  } catch (error: any) {
    console.error("Failed to import models for account:", error);
    appStore.showError(resolveAccountModelImportErrorMessage(t, error));
    return null;
  } finally {
    importingModelsAccountId.value = null;
  }
};

const handleResetQuota = async (a: Account) => {
  try {
    const updated = await adminAPI.accounts.resetAccountQuota(a.id);
    patchAccountInList(updated);
    refreshAccountSummarySafe();
    enterAutoRefreshSilentWindow();
    if (
      a.lifecycle_state === "archived" ||
      updated.lifecycle_state === "archived"
    ) {
      refreshArchivedPanel();
    }
    appStore.showSuccess(t("common.success"));
  } catch (error) {
    console.error("Failed to reset quota:", error);
  }
};
const handleRestoreOriginalProxy = async (a: Account) => {
  try {
    const result = await adminAPI.accounts.restoreOriginalProxy(a.id);
    appStore.showSuccess(
      t("admin.accounts.restoreOriginalProxySuccess", {
        name: result.restored_proxy_name,
      }),
    );
    await refreshListAndArchivedPanel();
    refreshAccountSummarySafe();
    enterAutoRefreshSilentWindow();
  } catch (error: any) {
    console.error("Failed to restore original proxy:", error);
    appStore.showError(
      error?.message || t("admin.accounts.restoreOriginalProxyFailed"),
    );
  }
};
const handleBlacklistAccount = async (a: Account) => {
  if (
    !window.confirm(t("admin.accounts.blacklist.addConfirm", { name: a.name }))
  ) {
    return;
  }
  try {
    const updated = await adminAPI.accounts.blacklist(a.id);
    patchAccountInList(updated);
    refreshAccountSummarySafe();
    enterAutoRefreshSilentWindow();
    appStore.showSuccess(t("admin.accounts.blacklist.addSuccess"));
  } catch (error: any) {
    console.error("Failed to blacklist account:", error);
    appStore.showError(
      error?.message || t("admin.accounts.blacklist.addFailed"),
    );
  }
};
const handleTestBlacklistAccount = async (payload: {
  account: Account;
  source: "test_modal";
  feedback?: BlacklistFeedbackPayload;
}) => {
  try {
    const updated = await adminAPI.accounts.blacklist(payload.account.id, {
      source: payload.source,
      feedback: payload.feedback,
    });
    patchAccountInList(updated);
    refreshAccountSummarySafe();
    enterAutoRefreshSilentWindow();
    appStore.showSuccess(t("admin.accounts.blacklist.addSuccess"));
    await closeTestModal();
  } catch (error: any) {
    console.error("Failed to blacklist account from test modal:", error);
    appStore.showError(
      error?.message || t("admin.accounts.blacklist.addFailed"),
    );
  }
};
const handleDelete = (a: Account) => {
  deletingAcc.value = a;
  showDeleteDialog.value = true;
};
const confirmDelete = async () => {
  if (!deletingAcc.value) return;
  const deletingArchived = deletingAcc.value.lifecycle_state === "archived";
  try {
    await adminAPI.accounts.delete(deletingAcc.value.id);
    showDeleteDialog.value = false;
    deletingAcc.value = null;
    if (deletingArchived) {
      await refreshListAndArchivedPanel();
      return;
    }
    await reload();
  } catch (error) {
    console.error("Failed to delete account:", error);
  }
};
const handleToggleSchedulable = async (a: Account) => {
  const nextSchedulable = !a.schedulable;
  togglingSchedulable.value = a.id;
  try {
    const updated = await adminAPI.accounts.setSchedulable(
      a.id,
      nextSchedulable,
    );
    updateSchedulableInList([a.id], updated?.schedulable ?? nextSchedulable);
    refreshAccountSummarySafe();
    enterAutoRefreshSilentWindow();
    if (
      a.lifecycle_state === "archived" ||
      updated.lifecycle_state === "archived"
    ) {
      refreshArchivedPanel();
    }
  } catch (error) {
    console.error("Failed to toggle schedulable:", error);
    appStore.showError(t("admin.accounts.failedToToggleSchedulable"));
  } finally {
    togglingSchedulable.value = null;
  }
};
const handleShowTempUnsched = (a: Account) => {
  tempUnschedAcc.value = a;
  showTempUnsched.value = true;
};
const handleTempUnschedReset = async (updated: Account) => {
  showTempUnsched.value = false;
  tempUnschedAcc.value = null;
  patchAccountInList(updated);
  refreshAccountSummarySafe();
  enterAutoRefreshSilentWindow();
  if (updated.lifecycle_state === "archived") {
    refreshArchivedPanel();
  }
};


  return {
    closeTestModal,
    closeBatchTestModal,
    closeStatsModal,
    closeModelDiagnostics,
    closeReAuthModal,
    handleTest,
    handleQuickTest,
    handleBatchTestCompleted,
    handleViewStats,
    handleDiagnoseModels,
    refreshModelDiagnostics,
    scheduleSourceProtocolLabel,
    handleSchedule,
    closeSchedulePanel,
    handleReAuth,
    handleRefresh,
    handleSetPrivacy,
    handleRecoverState,
    handleImportModels,
    handleResetQuota,
    handleRestoreOriginalProxy,
    handleBlacklistAccount,
    handleTestBlacklistAccount,
    handleDelete,
    confirmDelete,
    handleToggleSchedulable,
    handleShowTempUnsched,
    handleTempUnschedReset
  }
}
