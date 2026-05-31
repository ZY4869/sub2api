import { keysAPI } from "@/api";
import { buildApiKeyGroupBindingPayload } from "@/components/keys/apiKeyGroupBindings";
import type { ApiKey } from "@/types";
import type { Ref } from "vue";
import { imageCountWeightTiers, type ApiKeyFormData, type ImageCountWeightTier } from "./types";

interface SubmitApiKeyFormContext {
  formData: Ref<ApiKeyFormData>;
  selectedKey: Ref<ApiKey | null>;
  showEditModal: Ref<boolean>;
  submitting: Ref<boolean>;
  isAdminMode: Ref<boolean>;
  apiKeyModelSelectionRequired: Ref<boolean>;
  customKeyError: Ref<string> | { value: string };
  t: (key: string, params?: Record<string, unknown>) => string;
  appStore: {
    showError: (message: string) => void;
    showSuccess: (message: string) => void;
  };
  onboardingStore: {
    isCurrentStep: (selector: string) => boolean;
    nextStep: (delay?: number) => void;
  };
  syncImageOnlyGroupBindings: () => void;
  normalizeImageCountWeights: (
    weights?: Record<string, number> | null,
  ) => Record<ImageCountWeightTier, number>;
  closeModals: () => void;
  loadApiKeys: () => void;
}

export async function submitApiKeyForm(ctx: SubmitApiKeyFormContext) {
  const {
    formData,
    selectedKey,
    showEditModal,
    submitting,
    isAdminMode,
    apiKeyModelSelectionRequired,
    customKeyError,
    t,
    appStore,
    onboardingStore,
    syncImageOnlyGroupBindings,
    normalizeImageCountWeights,
    closeModals,
    loadApiKeys,
  } = ctx;

  if (formData.value.image_only_enabled) {
    syncImageOnlyGroupBindings();
  }
  const groupBindingsPayload = buildApiKeyGroupBindingPayload(
    formData.value.group_bindings,
    isAdminMode.value,
  );

  if (groupBindingsPayload.length === 0) {
    appStore.showError(t("keys.groupRequired"));
    return;
  }
  void apiKeyModelSelectionRequired;

  if (!showEditModal.value && formData.value.use_custom_key) {
    if (!formData.value.custom_key) {
      appStore.showError(t("keys.customKeyRequired"));
      return;
    }
    if (customKeyError.value) {
      appStore.showError(customKeyError.value);
      return;
    }
  }

  const parseIPList = (text: string): string[] =>
    text
      .split("\n")
      .map((ip) => ip.trim())
      .filter((ip) => ip.length > 0);
  const ipWhitelist = formData.value.enable_ip_restriction
    ? parseIPList(formData.value.ip_whitelist)
    : [];
  const ipBlacklist = formData.value.enable_ip_restriction
    ? parseIPList(formData.value.ip_blacklist)
    : [];
  const quota =
    formData.value.quota && formData.value.quota > 0 ? formData.value.quota : 0;

  let expiresInDays: number | undefined;
  let expiresAt: string | null | undefined;
  if (formData.value.enable_expiration && formData.value.expiration_date) {
    if (!showEditModal.value) {
      const expDate = new Date(formData.value.expiration_date);
      const now = new Date();
      const diffDays = Math.ceil(
        (expDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24),
      );
      expiresInDays = diffDays > 0 ? diffDays : 1;
    } else {
      expiresAt = new Date(formData.value.expiration_date).toISOString();
    }
  } else if (showEditModal.value) {
    expiresAt = "";
  }

  const rateLimitData = formData.value.enable_rate_limit
    ? {
        rate_limit_5h:
          formData.value.rate_limit_5h && formData.value.rate_limit_5h > 0
            ? formData.value.rate_limit_5h
            : 0,
        rate_limit_1d:
          formData.value.rate_limit_1d && formData.value.rate_limit_1d > 0
            ? formData.value.rate_limit_1d
            : 0,
        rate_limit_7d:
          formData.value.rate_limit_7d && formData.value.rate_limit_7d > 0
            ? formData.value.rate_limit_7d
            : 0,
      }
    : { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 };

  const imageOnlyEnabled = !!formData.value.image_only_enabled;
  const imageCountBillingEnabled =
    imageOnlyEnabled && !!formData.value.image_count_billing_enabled;
  const parsedImageMaxCount = Number(formData.value.image_max_count ?? 0);
  const imageMaxCount =
    imageCountBillingEnabled && Number.isFinite(parsedImageMaxCount) && parsedImageMaxCount > 0
      ? Math.floor(parsedImageMaxCount)
      : 0;

  if (imageCountBillingEnabled && imageMaxCount <= 0) {
    appStore.showError(t("keys.imageMaxCountRequired"));
    return;
  }
  if (
    imageCountBillingEnabled &&
    !imageCountWeightTiers.every((tier) => {
      const value = Number(formData.value.image_count_weights[tier]);
      return Number.isInteger(value) && value > 0;
    })
  ) {
    appStore.showError(t("keys.imageCountWeightInvalid"));
    return;
  }
  const imageCountWeights = normalizeImageCountWeights(
    formData.value.image_count_weights,
  );

  submitting.value = true;
  try {
    if (showEditModal.value && selectedKey.value) {
      await keysAPI.update(selectedKey.value.id, {
        name: formData.value.name,
        groups: groupBindingsPayload,
        status: formData.value.status,
        ip_whitelist: ipWhitelist,
        ip_blacklist: ipBlacklist,
        quota,
        image_only_enabled: imageOnlyEnabled,
        image_count_billing_enabled: imageCountBillingEnabled,
        image_max_count: imageMaxCount,
        image_count_weights: imageCountWeights,
        expires_at: expiresAt,
        rate_limit_5h: rateLimitData.rate_limit_5h,
        rate_limit_1d: rateLimitData.rate_limit_1d,
        rate_limit_7d: rateLimitData.rate_limit_7d,
      });
      appStore.showSuccess(t("keys.keyUpdatedSuccess"));
    } else {
      const customKey = formData.value.use_custom_key
        ? formData.value.custom_key
        : undefined;
      await keysAPI.createWithPayload({
        name: formData.value.name,
        groups: groupBindingsPayload,
        image_only_enabled: imageOnlyEnabled,
        image_count_billing_enabled: imageCountBillingEnabled,
        image_max_count: imageMaxCount,
        image_count_weights: imageCountWeights,
        ...(customKey ? { custom_key: customKey } : {}),
        ...(ipWhitelist.length ? { ip_whitelist: ipWhitelist } : {}),
        ...(ipBlacklist.length ? { ip_blacklist: ipBlacklist } : {}),
        ...(quota > 0 ? { quota } : {}),
        ...(expiresInDays && expiresInDays > 0
          ? { expires_in_days: expiresInDays }
          : {}),
        ...rateLimitData,
      });
      appStore.showSuccess(t("keys.keyCreatedSuccess"));
      if (onboardingStore.isCurrentStep('[data-tour="key-form-submit"]')) {
        onboardingStore.nextStep(500);
      }
    }
    closeModals();
    loadApiKeys();
  } catch (error: any) {
    const errorMsg = error?.message || t("keys.failedToSave");
    appStore.showError(errorMsg);
  } finally {
    submitting.value = false;
  }
}
