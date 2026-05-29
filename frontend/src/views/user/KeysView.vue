<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <SearchInput
            v-model="filterSearch"
            :placeholder="t('keys.searchPlaceholder')"
            class="w-full sm:w-64"
            @search="onFilterChange"
          />
          <Select
            :model-value="filterGroupId"
            class="w-40"
            :options="groupFilterOptions"
            @update:model-value="onGroupFilterChange"
          />
          <Select
            :model-value="filterStatus"
            class="w-40"
            :options="statusFilterOptions"
            @update:model-value="onStatusFilterChange"
          />
        </div>
      </template>

      <template #actions>
        <div class="flex justify-end gap-3">
          <button
            @click="loadApiKeys"
            :disabled="loading"
            class="btn btn-secondary"
            :title="t('common.refresh')"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <button
            @click="showCreateModal = true"
            class="btn btn-primary"
            data-tour="keys-create-btn"
          >
            <Icon name="plus" size="md" class="mr-2" />
            {{ t("keys.createKey") }}
          </button>
        </div>
      </template>

      <template #table>
        <KeysTable
          :columns="columns"
          :api-keys="apiKeys"
          :loading="loading"
          :copied-key-id="copiedKeyId"
          :usage-stats="usageStats"
          :user-group-rates="userGroupRates"
          :is-admin-mode="isAdminMode"
          :hide-ccs-import-button="!!publicSettings?.hide_ccs_import_button"
          :resolve-group="resolveGroup"
          :get-display-bindings="getDisplayBindings"
          :mask-key="maskKey"
          :format-reset-time="formatResetTime"
          @create="showCreateModal = true"
          @copy="copyToClipboard"
          @edit="editKey"
          @delete="confirmDelete"
          @use-key="openUseKeyModal"
          @import-ccswitch="importToCcswitch"
          @toggle-status="toggleKeyStatus"
          @reset-rate-limit="confirmResetRateLimitFromTable"
        />
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

    <KeyFormDialog
      :show="showCreateModal || showEditModal"
      :show-edit-modal="showEditModal"
      :submitting="submitting"
      :form-data="formData"
      :selected-key="selectedKey"
      :groups="groups"
      :group-model-catalog-items="groupModelCatalogItems"
      :group-model-options="groupModelOptions"
      :group-model-options-loading="groupModelOptionsLoading"
      :is-admin-mode="isAdminMode"
      :api-key-model-selection-required="apiKeyModelSelectionRequired"
      :custom-key-error="customKeyError"
      :status-options="statusOptions"
      @update:form-data="updateFormData"
      @close="closeModals"
      @submit="handleSubmit"
      @confirm-reset-quota="confirmResetQuota"
      @confirm-reset-rate-limit="confirmResetRateLimit"
      @set-expiration-days="setExpirationDays"
    />

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('keys.deleteKey')"
      :message="t('keys.deleteConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteDialog = false"
    />

    <ConfirmDialog
      :show="showResetQuotaDialog"
      :title="t('keys.resetQuotaTitle')"
      :message="
        t('keys.resetQuotaConfirmMessage', {
          name: selectedKey?.name,
          used: selectedKey?.quota_used?.toFixed(4),
        })
      "
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetQuotaUsed"
      @cancel="showResetQuotaDialog = false"
    />

    <ConfirmDialog
      :show="showResetRateLimitDialog"
      :title="t('keys.resetRateLimitTitle')"
      :message="t('keys.resetRateLimitConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetRateLimitUsage"
      @cancel="showResetRateLimitDialog = false"
    />

    <UseKeyModal
      :show="showUseKeyModal"
      :api-key="selectedKey?.key || ''"
      :base-url="publicSettings?.api_base_url || ''"
      :platform="selectedKey?.group?.platform || null"
      :allow-messages-dispatch="selectedKey?.group?.allow_messages_dispatch || false"
      @close="closeUseKeyModal"
    />

    <CcsClientSelectDialog
      :show="showCcsClientSelect"
      @select="handleCcsClientSelect"
      @close="closeCcsClientSelect"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useAppStore } from "@/stores/app";
import { useAuthStore } from "@/stores/auth";
import { useOnboardingStore } from "@/stores/onboarding";
import { useClipboard } from "@/composables/useClipboard";
import { getPersistedPageSize } from "@/composables/usePersistedPageSize";

const { t } = useI18n();
import { keysAPI, authAPI, usageAPI, userGroupsAPI } from "@/api";
import { adminAPI } from "@/api/admin";
import AppLayout from "@/components/layout/AppLayout.vue";
import TablePageLayout from "@/components/layout/TablePageLayout.vue";
import Pagination from "@/components/common/Pagination.vue";
import ConfirmDialog from "@/components/common/ConfirmDialog.vue";
import Select from "@/components/common/Select.vue";
import SearchInput from "@/components/common/SearchInput.vue";
import Icon from "@/components/icons/Icon.vue";
import UseKeyModal from "@/components/keys/UseKeyModal.vue";
import type { PublicModelCatalogItem } from "@/api/meta";
import type {
  ApiKey,
  Group,
  PublicSettings,
  UserGroupModelOption,
} from "@/types";
import {
  bindingToEditableDraft,
  createEmptyEditableBinding,
  getDisplayApiKeyGroups,
  type EditableApiKeyGroupBinding,
} from "@/components/keys/apiKeyGroupBindings";
import type { Column } from "@/components/common/types";
import type { BatchApiKeyUsageStats } from "@/api/usage";
import { buildCcsProviderImportLink } from "@/utils/ccswitchImport";
import KeysTable from "./keys/KeysTable.vue";
import KeyFormDialog from "./keys/KeyFormDialog.vue";
import CcsClientSelectDialog from "./keys/CcsClientSelectDialog.vue";
import { imageCountWeightTiers, type ApiKeyFormData, type ImageCountWeightTier } from "./keys/types";
import { submitApiKeyForm } from "./keys/submit";

// Helper to format date for datetime-local input
const formatDateTimeLocal = (isoDate: string): string => {
  const date = new Date(isoDate);
  const pad = (n: number) => n.toString().padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
};

const appStore = useAppStore();
const authStore = useAuthStore();
const onboardingStore = useOnboardingStore();
const { copyToClipboard: clipboardCopy } = useClipboard();
const isAdminMode = computed(() => authStore.isAdmin);
const apiKeyModelSelectionRequired = computed(
  () =>
    !isAdminMode.value &&
    authStore.user?.api_key_model_binding_mode !== "group_allowed",
);

const columns = computed<Column[]>(() => [
  { key: "name", label: t("common.name"), sortable: true },
  { key: "key", label: t("keys.apiKey"), sortable: false },
  { key: "group", label: t("keys.group"), sortable: false },
  { key: "usage", label: t("keys.usage"), sortable: false },
  { key: "rate_limit", label: t("keys.rateLimitColumn"), sortable: false },
  { key: "expires_at", label: t("keys.expiresAt"), sortable: true },
  { key: "status", label: t("common.status"), sortable: true },
  { key: "last_used_at", label: t("keys.lastUsedAt"), sortable: true },
  { key: "created_at", label: t("keys.created"), sortable: true },
  { key: "actions", label: t("common.actions"), sortable: false },
]);

const apiKeys = ref<ApiKey[]>([]);
const groups = ref<Group[]>([]);
const loading = ref(false);
const submitting = ref(false);
const now = ref(new Date());
let resetTimer: ReturnType<typeof setInterval> | null = null;
const usageStats = ref<Record<string, BatchApiKeyUsageStats>>({});
const userGroupRates = ref<Record<number, number>>({});
const groupModelOptions = ref<Record<number, UserGroupModelOption[]>>({});
const groupModelCatalogItems = ref<Record<number, PublicModelCatalogItem[]>>({});
const groupModelOptionsLoading = ref(false);
const groupMap = computed(
  () => new Map(groups.value.map((group) => [group.id, group] as const)),
);
const defaultImageCountWeights = (): Record<ImageCountWeightTier, number> => ({
  "1K": 1,
  "2K": 1,
  "4K": 2,
});

const resolveGroup = (
  groupId: number | null | undefined,
): Group | undefined => {
  if (!groupId) return undefined;
  return groupMap.value.get(groupId);
};

const getDisplayBindings = (key: ApiKey) =>
  getDisplayApiKeyGroups(key, resolveGroup);

const pagination = ref({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0,
});

// Filter state
const filterSearch = ref("");
const filterStatus = ref("");
const filterGroupId = ref<string | number>("");

const showCreateModal = ref(false);
const showEditModal = ref(false);
const showDeleteDialog = ref(false);
const showResetQuotaDialog = ref(false);
const showResetRateLimitDialog = ref(false);
const showUseKeyModal = ref(false);
const showCcsClientSelect = ref(false);
const pendingCcsRow = ref<ApiKey | null>(null);
const selectedKey = ref<ApiKey | null>(null);
const copiedKeyId = ref<number | null>(null);
const publicSettings = ref<PublicSettings | null>(null);
let abortController: AbortController | null = null;

const formData = ref<ApiKeyFormData>({
  name: "",
  group_bindings: [
    createEmptyEditableBinding(),
  ] as EditableApiKeyGroupBinding[],
  status: "active" as "active" | "inactive",
  use_custom_key: false,
  custom_key: "",
  enable_ip_restriction: false,
  ip_whitelist: "",
  ip_blacklist: "",
  // Quota settings (empty = unlimited)
  enable_quota: false,
  quota: null as number | null,
  // Image-only key settings
  image_only_enabled: false,
  image_count_billing_enabled: false,
  image_max_count: null as number | null,
  image_count_weights: defaultImageCountWeights(),
  // Rate limit settings
  enable_rate_limit: false,
  rate_limit_5h: null as number | null,
  rate_limit_1d: null as number | null,
  rate_limit_7d: null as number | null,
  enable_expiration: false,
  expiration_preset: "30" as "7" | "30" | "90" | "custom",
  expiration_date: "",
});

// 自定义Key验证
const customKeyError = computed(() => {
  if (!formData.value.use_custom_key || !formData.value.custom_key) {
    return "";
  }
  const key = formData.value.custom_key;
  if (key.length < 16) {
    return t("keys.customKeyTooShort");
  }
  // 检查字符：只允许字母、数字、下划线、连字符
  if (!/^[a-zA-Z0-9_-]+$/.test(key)) {
    return t("keys.customKeyInvalidChars");
  }
  return "";
});

const updateFormData = (value: ApiKeyFormData) => {
  formData.value = value;
};

watch(
  () => formData.value.image_only_enabled,
  (enabled) => {
    if (enabled) {
      syncImageOnlyGroupBindings();
      return;
    }
    formData.value.image_count_billing_enabled = false;
    formData.value.image_max_count = null;
  },
);

watch(
  () => formData.value.image_count_billing_enabled,
  (enabled) => {
    if (enabled) return;
    formData.value.image_max_count = null;
  },
);

const statusOptions = computed(() => [
  { value: "active", label: t("common.active") },
  { value: "inactive", label: t("common.inactive") },
]);

// Filter dropdown options
const groupFilterOptions = computed(() => [
  { value: "", label: t("keys.allGroups") },
  { value: 0, label: t("keys.noGroup") },
  ...groups.value.map((g) => ({ value: g.id, label: g.name })),
]);

const statusFilterOptions = computed(() => [
  { value: "", label: t("keys.allStatus") },
  { value: "active", label: t("keys.status.active") },
  { value: "inactive", label: t("keys.status.inactive") },
  { value: "quota_exhausted", label: t("keys.status.quota_exhausted") },
  { value: "expired", label: t("keys.status.expired") },
]);

const onFilterChange = () => {
  pagination.value.page = 1;
  loadApiKeys();
};

const onGroupFilterChange = (value: string | number | boolean | null) => {
  filterGroupId.value = value as string | number;
  onFilterChange();
};

const onStatusFilterChange = (value: string | number | boolean | null) => {
  filterStatus.value = value as string;
  onFilterChange();
};

const maskKey = (key: string): string => {
  if (key.length <= 12) return key;
  return `${key.slice(0, 8)}...${key.slice(-4)}`;
};

const copyToClipboard = async (text: string, keyId: number) => {
  const success = await clipboardCopy(text, t("keys.copied"));
  if (success) {
    copiedKeyId.value = keyId;
    setTimeout(() => {
      copiedKeyId.value = null;
    }, 800);
  }
};

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== "object") return false;
  const { name, code } = error as { name?: string; code?: string };
  return name === "AbortError" || code === "ERR_CANCELED";
};

const loadApiKeys = async () => {
  abortController?.abort();
  const controller = new AbortController();
  abortController = controller;
  const { signal } = controller;
  loading.value = true;
  try {
    // Build filters
    const filters: {
      search?: string;
      status?: string;
      group_id?: number | string;
    } = {};
    if (filterSearch.value) filters.search = filterSearch.value;
    if (filterStatus.value) filters.status = filterStatus.value;
    if (filterGroupId.value !== "") filters.group_id = filterGroupId.value;

    const response = await keysAPI.list(
      pagination.value.page,
      pagination.value.page_size,
      filters,
      {
        signal,
      },
    );
    if (signal.aborted) return;
    apiKeys.value = response.items;
    pagination.value.total = response.total;
    pagination.value.pages = response.pages;

    // Load usage stats for all API keys in the list
    if (response.items.length > 0) {
      const keyIds = response.items.map((k) => k.id);
      try {
        const usageResponse = await usageAPI.getDashboardApiKeysUsage(keyIds, {
          signal,
        });
        if (signal.aborted) return;
        usageStats.value = usageResponse.stats;
      } catch (e) {
        if (!isAbortError(e)) {
          console.error("Failed to load usage stats:", e);
        }
      }
    }
  } catch (error) {
    if (isAbortError(error)) {
      return;
    }
    appStore.showError(t("keys.failedToLoad"));
  } finally {
    if (abortController === controller) {
      loading.value = false;
    }
  }
};

const loadGroups = async () => {
  try {
    groups.value = isAdminMode.value
      ? await adminAPI.groups.getAll()
      : await userGroupsAPI.getAvailable();
  } catch (error) {
    console.error("Failed to load groups:", error);
  }
};

const loadGroupModelOptions = async () => {
  if (isAdminMode.value) {
    groupModelOptions.value = {};
    groupModelCatalogItems.value = {};
    return;
  }
  groupModelOptionsLoading.value = true;
  try {
    const response = await userGroupsAPI.getModelOptions();
    groupModelOptions.value = Object.fromEntries(
      response.map((group) => [group.group_id, group.models]),
    );
    syncImageOnlyGroupBindings();
  } catch (error) {
    groupModelOptions.value = {};
    console.error("Failed to load group model options:", error);
  } finally {
    groupModelOptionsLoading.value = false;
  }
};

const loadGroupModelCatalog = async (groupId: number) => {
  if (isAdminMode.value || !groupId || groupModelCatalogItems.value[groupId]) {
    return;
  }
  try {
    const snapshot = await userGroupsAPI.getModelCatalog(groupId);
    groupModelCatalogItems.value = {
      ...groupModelCatalogItems.value,
      [groupId]: snapshot.items || [],
    };
    syncImageOnlyGroupBindings();
  } catch (error) {
    groupModelCatalogItems.value = {
      ...groupModelCatalogItems.value,
      [groupId]: [],
    };
    console.error("Failed to load group model catalog:", error);
  }
};

function normalizeImageCountWeights(
  weights?: Record<string, number> | null,
): Record<ImageCountWeightTier, number> {
  const normalized = defaultImageCountWeights();
  imageCountWeightTiers.forEach((tier) => {
    const value = Number(weights?.[tier] ?? normalized[tier]);
    if (Number.isFinite(value) && value > 0) {
      normalized[tier] = Math.floor(value);
    }
  });
  return normalized;
}

function imageModelIdsForBinding(binding: EditableApiKeyGroupBinding): string[] {
  const groupID = Number(binding.group_id) || 0;
  if (groupID <= 0) {
    return [];
  }
  const catalogByModel = new Map(
    (groupModelCatalogItems.value[groupID] || []).map((item) => [item.model, item]),
  );
  return (groupModelOptions.value[groupID] || [])
    .filter((model) => {
      const catalogItem = catalogByModel.get(model.public_id);
      if (catalogItem?.mode === "image") {
        return true;
      }
      const protocols = catalogItem?.request_protocols || model.request_protocols || [];
      return protocols.some((protocol) => String(protocol).toLowerCase().includes("image"));
    })
    .map((model) => model.public_id);
}

function syncImageOnlyGroupBindings() {
  if (!formData.value.image_only_enabled || isAdminMode.value) {
    return;
  }
  formData.value.group_bindings = formData.value.group_bindings.map((binding) => {
    const imageModels = imageModelIdsForBinding(binding);
    if (imageModels.length === 0) {
      return binding;
    }
    if (
      binding.model_selection_dirty &&
      arraysEqual(binding.selected_models, imageModels)
    ) {
      return binding;
    }
    return {
      ...binding,
      selected_models: imageModels,
      model_patterns_text: imageModels.join("\n"),
      model_selection_dirty: true,
    };
  });
}

function arraysEqual(left: string[], right: string[]): boolean {
  return left.length === right.length && left.every((item, index) => item === right[index]);
}

const loadUserGroupRates = async () => {
  try {
    userGroupRates.value = await userGroupsAPI.getUserGroupRates();
  } catch (error) {
    console.error("Failed to load user group rates:", error);
  }
};

const loadPublicSettings = async () => {
  try {
    publicSettings.value = await authAPI.getPublicSettings();
  } catch (error) {
    console.error("Failed to load public settings:", error);
  }
};

const openUseKeyModal = (key: ApiKey) => {
  selectedKey.value = key;
  showUseKeyModal.value = true;
};

const closeUseKeyModal = () => {
  showUseKeyModal.value = false;
  selectedKey.value = null;
};

const handlePageChange = (page: number) => {
  pagination.value.page = page;
  loadApiKeys();
};

const handlePageSizeChange = (pageSize: number) => {
  pagination.value.page_size = pageSize;
  pagination.value.page = 1;
  loadApiKeys();
};

const editKey = (key: ApiKey) => {
  selectedKey.value = key;
  const hasIPRestriction =
    key.ip_whitelist?.length > 0 || key.ip_blacklist?.length > 0;
  const hasExpiration = !!key.expires_at;
  const bindings = getDisplayBindings(key);
  formData.value = {
    name: key.name,
    group_bindings: bindings.length
      ? bindings.map(bindingToEditableDraft)
      : [createEmptyEditableBinding()],
    status:
      key.status === "quota_exhausted" || key.status === "expired"
        ? "inactive"
        : key.status,
    use_custom_key: false,
    custom_key: "",
    enable_ip_restriction: hasIPRestriction,
    ip_whitelist: (key.ip_whitelist || []).join("\n"),
    ip_blacklist: (key.ip_blacklist || []).join("\n"),
    enable_quota: key.quota > 0,
    quota: key.quota > 0 ? key.quota : null,
    image_only_enabled: !!key.image_only_enabled,
    image_count_billing_enabled:
      !!key.image_only_enabled && !!key.image_count_billing_enabled,
    image_max_count:
      !!key.image_only_enabled &&
      !!key.image_count_billing_enabled &&
      (key.image_max_count || 0) > 0
        ? key.image_max_count
        : null,
    image_count_weights: normalizeImageCountWeights(key.image_count_weights),
    enable_rate_limit:
      key.rate_limit_5h > 0 || key.rate_limit_1d > 0 || key.rate_limit_7d > 0,
    rate_limit_5h: key.rate_limit_5h || null,
    rate_limit_1d: key.rate_limit_1d || null,
    rate_limit_7d: key.rate_limit_7d || null,
    enable_expiration: hasExpiration,
    expiration_preset: "custom",
    expiration_date: key.expires_at ? formatDateTimeLocal(key.expires_at) : "",
  };
  showEditModal.value = true;
};

const toggleKeyStatus = async (key: ApiKey) => {
  const newStatus = key.status === "active" ? "inactive" : "active";
  try {
    await keysAPI.toggleStatus(key.id, newStatus);
    appStore.showSuccess(
      newStatus === "active"
        ? t("keys.keyEnabledSuccess")
        : t("keys.keyDisabledSuccess"),
    );
    loadApiKeys();
  } catch (error) {
    appStore.showError(t("keys.failedToUpdateStatus"));
  }
};

const confirmDelete = (key: ApiKey) => {
  selectedKey.value = key;
  showDeleteDialog.value = true;
};

const handleSubmit = async () => {
  await submitApiKeyForm({
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
  });
};

/**
 * 处理删除 API Key 的操作
 * 优化：错误处理改进，优先显示后端返回的具体错误消息（如权限不足等），
 * 若后端未返回消息则显示默认的国际化文本
 */
const handleDelete = async () => {
  if (!selectedKey.value) return;

  try {
    await keysAPI.delete(selectedKey.value.id);
    appStore.showSuccess(t("keys.keyDeletedSuccess"));
    showDeleteDialog.value = false;
    loadApiKeys();
  } catch (error: any) {
    // 优先使用后端返回的错误消息，提供更具体的错误信息给用户
    const errorMsg = error?.message || t("keys.failedToDelete");
    appStore.showError(errorMsg);
  }
};

const closeModals = () => {
  showCreateModal.value = false;
  showEditModal.value = false;
  selectedKey.value = null;
  groupModelCatalogItems.value = {};
  formData.value = {
    name: "",
    group_bindings: [createEmptyEditableBinding()],
    status: "active",
    use_custom_key: false,
    custom_key: "",
    enable_ip_restriction: false,
    ip_whitelist: "",
    ip_blacklist: "",
    enable_quota: false,
    quota: null,
    image_only_enabled: false,
    image_count_billing_enabled: false,
    image_max_count: null,
    image_count_weights: defaultImageCountWeights(),
    enable_rate_limit: false,
    rate_limit_5h: null,
    rate_limit_1d: null,
    rate_limit_7d: null,
    enable_expiration: false,
    expiration_preset: "30",
    expiration_date: "",
  };
};

watch(
  () => [
    isAdminMode.value,
    showCreateModal.value || showEditModal.value,
    formData.value.group_bindings.map((binding) => binding.group_id).join(","),
  ] as const,
  ([adminMode, isDialogOpen]) => {
    if (adminMode || !isDialogOpen) {
      return;
    }
    const groupIDs = Array.from(
      new Set(
        formData.value.group_bindings
          .map((binding) => Number(binding.group_id) || 0)
          .filter((groupID) => groupID > 0),
      ),
    );
    groupIDs.forEach((groupID) => {
      void loadGroupModelCatalog(groupID);
    });
    syncImageOnlyGroupBindings();
  },
  { immediate: true },
);

// Show reset quota confirmation dialog
const confirmResetQuota = () => {
  showResetQuotaDialog.value = true;
};

// Set expiration date based on quick select days
const setExpirationDays = (days: number) => {
  formData.value.expiration_preset = days.toString() as "7" | "30" | "90";
  const expDate = new Date();
  expDate.setDate(expDate.getDate() + days);
  formData.value.expiration_date = formatDateTimeLocal(expDate.toISOString());
};

// Reset quota used for an API key
const resetQuotaUsed = async () => {
  if (!selectedKey.value) return;
  showResetQuotaDialog.value = false;
  try {
    await keysAPI.update(selectedKey.value.id, { reset_quota: true });
    appStore.showSuccess(t("keys.quotaResetSuccess"));
    // Update local state
    if (selectedKey.value) {
      selectedKey.value.quota_used = 0;
    }
  } catch (error: any) {
    const errorMsg =
      error.response?.data?.detail || t("keys.failedToResetQuota");
    appStore.showError(errorMsg);
  }
};

// Show reset rate limit confirmation dialog (from edit modal)
const confirmResetRateLimit = () => {
  showResetRateLimitDialog.value = true;
};

// Show reset rate limit confirmation dialog (from table row)
const confirmResetRateLimitFromTable = (row: ApiKey) => {
  selectedKey.value = row;
  showResetRateLimitDialog.value = true;
};

// Reset rate limit usage for an API key
const resetRateLimitUsage = async () => {
  if (!selectedKey.value) return;
  showResetRateLimitDialog.value = false;
  try {
    await keysAPI.update(selectedKey.value.id, {
      reset_rate_limit_usage: true,
    });
    appStore.showSuccess(t("keys.rateLimitResetSuccess"));
    // Refresh key data
    await loadApiKeys();
    // Update the editing key with fresh data
    const refreshedKey = apiKeys.value.find(
      (k) => k.id === selectedKey.value!.id,
    );
    if (refreshedKey) {
      selectedKey.value = refreshedKey;
    }
  } catch (error: any) {
    const errorMsg =
      error.response?.data?.detail || t("keys.failedToResetRateLimit");
    appStore.showError(errorMsg);
  }
};

const importToCcswitch = (row: ApiKey) => {
  const platform = row.group?.platform || "anthropic";

  // For antigravity platform, show client selection dialog
  if (platform === "antigravity") {
    pendingCcsRow.value = row;
    showCcsClientSelect.value = true;
    return;
  }

  // For other platforms, execute directly
  executeCcsImport(row, platform === "gemini" ? "gemini" : "claude");
};

const executeCcsImport = (row: ApiKey, clientType: "claude" | "gemini") => {
  const baseUrl = publicSettings.value?.api_base_url || window.location.origin;
  const platform = row.group?.platform || "anthropic";
  const providerName =
    (publicSettings.value?.site_name || "sub2api").trim() || "sub2api";
  const deeplink = buildCcsProviderImportLink({
    apiKey: row.key,
    baseUrl,
    clientType,
    platform,
    providerName,
  });

  try {
    window.open(deeplink, "_self");

    // Check if the protocol handler worked by detecting if we're still focused
    setTimeout(() => {
      if (document.hasFocus()) {
        // Still focused means the protocol handler likely failed
        appStore.showError(t("keys.ccSwitchNotInstalled"));
      }
    }, 100);
  } catch (error) {
    appStore.showError(t("keys.ccSwitchNotInstalled"));
  }
};

const handleCcsClientSelect = (clientType: "claude" | "gemini") => {
  if (pendingCcsRow.value) {
    executeCcsImport(pendingCcsRow.value, clientType);
  }
  showCcsClientSelect.value = false;
  pendingCcsRow.value = null;
};

const closeCcsClientSelect = () => {
  showCcsClientSelect.value = false;
  pendingCcsRow.value = null;
};

function formatResetTime(resetAt: string | null): string {
  if (!resetAt) return "";
  const diff = new Date(resetAt).getTime() - now.value.getTime();
  if (diff <= 0) return t("keys.resetNow");
  const days = Math.floor(diff / 86400000);
  const hours = Math.floor((diff % 86400000) / 3600000);
  const mins = Math.floor((diff % 3600000) / 60000);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${mins}m`;
  return `${mins}m`;
}

onMounted(() => {
  loadApiKeys();
  loadGroups();
  loadGroupModelOptions();
  loadUserGroupRates();
  loadPublicSettings();
  resetTimer = setInterval(() => {
    now.value = new Date();
  }, 60000);
});

onUnmounted(() => {
  if (resetTimer) clearInterval(resetTimer);
});
</script>
