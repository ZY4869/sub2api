import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { adminAPI } from "@/api";
import { useAppStore } from "@/stores";
import type { Account, AccountUsagePresentation } from "@/types";
import {
  invalidateAccountUsagePresentationCache,
  refreshAccountUsagePresentation,
} from "@/composables/useAccountUsagePresentation";
import { getRuntimePlatform } from "@/composables/accountUsagePresentation/support";

function formatOpenAIQuotaResetRemaining(value: number | null): string {
  if (value === null) return "--";
  const normalized = Number.isFinite(Number(value))
    ? Math.max(0, Math.floor(Number(value)))
    : null;
  return normalized === null ? "--" : String(normalized).padStart(2, "0");
}

function resolveRefreshResetCreditsErrorMessage(error: any, fallback: string) {
  const responseData = error?.response?.data ?? {};
  return (
    responseData.detail ||
    responseData.message ||
    error?.message ||
    fallback
  );
}

function resolveResetQuotaErrorMessage(error: any, t: (key: string) => string) {
  const responseData = error?.response?.data ?? {};
  const reason = String(
    responseData.reason ||
      responseData.error ||
      responseData.code ||
      responseData.error_code ||
      "",
  );

  if (reason === "OPENAI_RESET_CREDITS_NO_CREDIT") {
    return t("admin.accounts.usageWindow.resetQuotaNoCredit");
  }
  if (reason === "OPENAI_RESET_CREDITS_NOTHING_TO_RESET") {
    return t("admin.accounts.usageWindow.resetQuotaNothingToReset");
  }

  return (
    responseData.detail ||
    responseData.message ||
    error?.message ||
    t("admin.accounts.usageWindow.resetQuotaFailed")
  );
}

export function useOpenAIResetCreditsControls(
  account: () => Account,
  presentation: () => AccountUsagePresentation,
) {
  const { t } = useI18n();
  const getAppStore = () => useAppStore();
  const resetting = ref(false);
  const refreshingResetCredits = ref(false);

  const canResetOpenAIQuota = computed(() => {
    const currentAccount = account();
    return (
      getRuntimePlatform(currentAccount) === "openai" &&
      currentAccount.type === "oauth"
    );
  });

  const openAIQuotaResetRemaining = computed(() => {
    return presentation().meta.openAIResetCreditsAvailableCount ?? null;
  });

  const resetCreditsUnsupported = computed(() => {
    return presentation().meta.openAIResetCreditsStatus === "unsupported";
  });

  const resetCreditsUnknown = computed(() => {
    return (
      !presentation().meta.openAIResetCreditsKnown &&
      !resetCreditsUnsupported.value
    );
  });

  const resetCreditsZero = computed(() => openAIQuotaResetRemaining.value === 0);

  const resetButtonDisabled = computed(() => {
    return (
      resetting.value ||
      refreshingResetCredits.value ||
      resetCreditsUnsupported.value
    );
  });

  const resetCreditsStatusLabel = computed(() => {
    if (resetCreditsUnsupported.value) {
      return (
        presentation().meta.openAIResetCreditsUnsupportedReason ||
        t("admin.accounts.usageWindow.resetQuotaUnsupported")
      );
    }
    return t("admin.accounts.usageWindow.resetQuotaRemaining", {
      count: formatOpenAIQuotaResetRemaining(openAIQuotaResetRemaining.value),
    });
  });

  async function resetOpenAIQuota() {
    const currentAccount = account();
    if (!canResetOpenAIQuota.value || resetButtonDisabled.value) return;
    if (!window.confirm(t("admin.accounts.usageWindow.resetQuotaConfirm"))) {
      return;
    }

    resetting.value = true;
    try {
      await adminAPI.accounts.resetAccountQuota(currentAccount.id);
      invalidateAccountUsagePresentationCache([currentAccount.id]);
      await refreshAccountUsagePresentation([currentAccount], {
        force: true,
        source: "active",
      });
      const appStore = getAppStore();
      appStore.showSuccess(t("admin.accounts.usageWindow.resetQuotaSuccess"));
    } catch (error: any) {
      invalidateAccountUsagePresentationCache([currentAccount.id]);
      await refreshAccountUsagePresentation([currentAccount], {
        force: true,
        source: "active",
      }).catch(() => {});
      const appStore = getAppStore();
      appStore.showError(resolveResetQuotaErrorMessage(error, t));
    } finally {
      resetting.value = false;
    }
  }

  async function refreshOpenAIResetCredits() {
    const currentAccount = account();
    if (
      !canResetOpenAIQuota.value ||
      refreshingResetCredits.value ||
      resetting.value
    ) {
      return;
    }

    refreshingResetCredits.value = true;
    try {
      invalidateAccountUsagePresentationCache([currentAccount.id]);
      const result = await refreshAccountUsagePresentation([currentAccount], {
        force: true,
        source: "active",
      });
      if (result.failed > 0) {
        const appStore = getAppStore();
        appStore.showError(
          t("admin.accounts.usageWindow.refreshResetCreditsFailed"),
        );
        return;
      }
      const appStore = getAppStore();
      appStore.showSuccess(
        t("admin.accounts.usageWindow.refreshResetCreditsSuccess"),
      );
    } catch (error: any) {
      const appStore = getAppStore();
      appStore.showError(
        resolveRefreshResetCreditsErrorMessage(
          error,
          t("admin.accounts.usageWindow.refreshResetCreditsFailed"),
        ),
      );
    } finally {
      refreshingResetCredits.value = false;
    }
  }

  return {
    canResetOpenAIQuota,
    resetCreditsStatusLabel,
    resetCreditsUnsupported,
    resetCreditsUnknown,
    resetCreditsZero,
    resetting,
    refreshingResetCredits,
    resetButtonDisabled,
    resetOpenAIQuota,
    refreshOpenAIResetCredits,
  };
}
