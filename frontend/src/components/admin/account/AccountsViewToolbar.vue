<template>
  <div>
    <div class="flex flex-wrap-reverse items-start justify-between gap-3">
      <AccountTableFilters
        :search-query="searchQuery"
        :filters="filters"
        :groups="groups"
        @update:filters="handleFiltersUpdate"
        @change="emit('change')"
        @update:search-query="handleSearchQueryUpdate"
      />
      <AccountTableActions
        :loading="loading"
        @refresh="emit('refresh')"
        @sync="emit('sync')"
        @create="emit('create')"
      >
        <template #after>
          <AccountViewModeToggle
            :model-value="viewMode"
            @update:model-value="emit('update:view-mode', $event)"
          />

          <button
            type="button"
            class="btn btn-secondary"
            @click="emit('toggle-group-view')"
          >
            {{
              groupViewEnabled
                ? t("admin.accounts.groupView.disable")
                : t("admin.accounts.groupView.enable")
            }}
          </button>

          <button
            type="button"
            class="btn btn-secondary"
            data-platform-sort-button="true"
            :title="platformSortToggleTitle"
            @click="
              emit(
                'update:platform-count-sort-order',
                nextPlatformCountSortOrder,
              )
            "
          >
            {{ platformSortLabel }}
          </button>

          <button
            v-if="showLimitedControls"
            type="button"
            class="btn btn-secondary"
            @click="emit('toggle-hide-limited')"
          >
            {{
              hideLimitedAccounts
                ? t("admin.accounts.limited.hideToggleOn")
                : t("admin.accounts.limited.hideToggleOff")
            }}
          </button>

          <button
            v-if="showLimitedControls"
            type="button"
            class="btn btn-secondary"
            @click="emit('open-limited-page')"
          >
            {{
              t("admin.accounts.limited.entry", { count: limitedAccountsCount })
            }}
          </button>

          <button
            type="button"
            class="btn btn-secondary"
            data-actual-usage-button="true"
            :title="t('admin.accounts.refreshActualUsageTitle')"
            :aria-label="t('admin.accounts.refreshActualUsageTitle')"
            :disabled="loading || usageRefreshing"
            @click="emit('refresh-usage')"
          >
            <Icon
              name="refresh"
              size="md"
              :class="[usageRefreshing ? 'animate-spin' : '']"
            />
            <span class="hidden md:inline">
              {{
                usageRefreshing
                  ? t("admin.accounts.refreshingActualUsage")
                  : t("admin.accounts.refreshActualUsage")
              }}
            </span>
          </button>

          <div class="relative" ref="autoRefreshDropdownRef">
            <button
              type="button"
              class="btn btn-secondary px-2 md:px-3"
              :title="t('admin.accounts.autoRefresh')"
              @click="toggleAutoRefreshDropdown"
            >
              <Icon
                name="refresh"
                size="sm"
                :class="[autoRefreshEnabled ? 'animate-spin' : '']"
              />
              <span class="hidden md:inline">
                {{
                  autoRefreshEnabled
                    ? t("admin.accounts.autoRefreshCountdown", {
                        seconds: autoRefreshCountdown,
                      })
                    : t("admin.accounts.autoRefresh")
                }}
              </span>
            </button>
            <div
              v-if="showAutoRefreshDropdown"
              class="absolute right-0 z-50 mt-2 w-56 origin-top-right rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
            >
              <div class="p-2">
                <button
                  type="button"
                  class="flex w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
                  @click="emit('set-auto-refresh-enabled', !autoRefreshEnabled)"
                >
                  <span>{{ t("admin.accounts.enableAutoRefresh") }}</span>
                  <Icon
                    v-if="autoRefreshEnabled"
                    name="check"
                    size="sm"
                    class="text-primary-500"
                  />
                </button>
                <div
                  class="my-1 border-t border-gray-100 dark:border-gray-700"
                ></div>
                <button
                  v-for="sec in autoRefreshIntervals"
                  :key="sec"
                  type="button"
                  class="flex w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
                  @click="emit('set-auto-refresh-interval', sec)"
                >
                  <span>{{ autoRefreshIntervalLabel(sec) }}</span>
                  <Icon
                    v-if="autoRefreshIntervalSeconds === sec"
                    name="check"
                    size="sm"
                    class="text-primary-500"
                  />
                </button>
              </div>
            </div>
          </div>

          <button
            type="button"
            class="btn btn-secondary"
            :title="t('admin.errorPassthrough.title')"
            @click="emit('show-error-passthrough')"
          >
            <Icon name="shield" size="md" class="mr-1.5" />
            <span class="hidden md:inline">{{
              t("admin.errorPassthrough.title")
            }}</span>
          </button>

          <button
            type="button"
            class="btn btn-secondary"
            :title="t('admin.tlsFingerprintProfiles.title')"
            @click="emit('show-tls-fingerprint-profiles')"
          >
            <span class="hidden md:inline">{{
              t("admin.tlsFingerprintProfiles.title")
            }}</span>
            <span class="md:hidden">{{
              t("admin.tlsFingerprintProfiles.shortTitle")
            }}</span>
          </button>

          <div class="relative" ref="columnDropdownRef">
            <button
              type="button"
              class="btn btn-secondary px-2 md:px-3"
              :title="t('admin.users.columnSettings')"
              @click="toggleColumnDropdown"
            >
              <svg
                class="h-4 w-4 md:mr-1.5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M9 4.5v15m6-15v15m-10.875 0h15.75c.621 0 1.125-.504 1.125-1.125V5.625c0-.621-.504-1.125-1.125-1.125H4.125C3.504 4.5 3 5.004 3 5.625v12.75c0 .621.504 1.125 1.125 1.125z"
                />
              </svg>
              <span class="hidden md:inline">{{
                t("admin.users.columnSettings")
              }}</span>
            </button>
            <div
              v-if="showColumnDropdown"
              class="absolute right-0 z-50 mt-2 w-48 origin-top-right rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
            >
              <div class="max-h-80 overflow-y-auto p-2">
                <button
                  v-for="col in toggleableColumns"
                  :key="col.key"
                  type="button"
                  class="flex w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
                  @click="emit('toggle-column', col.key)"
                >
                  <span>{{ col.label }}</span>
                  <Icon
                    v-if="col.visible"
                    name="check"
                    size="sm"
                    class="text-primary-500"
                  />
                </button>
              </div>
            </div>
          </div>
        </template>

        <template #beforeCreate>
          <button
            type="button"
            class="btn btn-secondary"
            @click="emit('import-data')"
          >
            {{ t("admin.accounts.dataImport") }}
          </button>
          <button
            type="button"
            class="btn btn-secondary"
            @click="emit('export-data')"
          >
            {{
              selectedCount
                ? t("admin.accounts.dataExportSelected")
                : t("admin.accounts.dataExport")
            }}
          </button>
        </template>
      </AccountTableActions>
    </div>

    <div
      v-if="hasPendingListSync"
      class="mt-2 flex items-center justify-between rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800 dark:border-amber-700/40 dark:bg-amber-900/20 dark:text-amber-200"
    >
      <span>{{ t("admin.accounts.listPendingSyncHint") }}</span>
      <button
        type="button"
        class="btn btn-secondary px-2 py-1 text-xs"
        @click="emit('sync-pending-list')"
      >
        {{ t("admin.accounts.listPendingSyncAction") }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import type {
  AdminGroup,
  AccountPlatformCountSortOrder,
  AccountViewMode,
} from "@/types";
import Icon from "@/components/icons/Icon.vue";
import AccountViewModeToggle from "./AccountViewModeToggle.vue";
import AccountTableActions from "./AccountTableActions.vue";
import AccountTableFilters from "./AccountTableFilters.vue";

interface ToggleableColumn {
  key: string;
  label: string;
  visible: boolean;
}

interface ActualUsageRefreshSummary {
  total: number;
  live: number;
  fallback: number;
}

const props = defineProps<{
  loading: boolean;
  usageRefreshing: boolean;
  searchQuery: string;
  filters: Record<string, unknown>;
  groups: AdminGroup[];
  hasPendingListSync: boolean;
  selectedCount: number;
  autoRefreshEnabled: boolean;
  autoRefreshCountdown: number;
  autoRefreshIntervals: readonly number[];
  autoRefreshIntervalSeconds: number;
  toggleableColumns: ToggleableColumn[];
  viewMode: AccountViewMode;
  groupViewEnabled: boolean;
  platformCountSortOrder: AccountPlatformCountSortOrder;
  showLimitedControls?: boolean;
  hideLimitedAccounts?: boolean;
  limitedAccountsCount?: number;
  actualUsageRefreshSummary: ActualUsageRefreshSummary;
}>();

const emit = defineEmits<{
  "update:filters": [value: Record<string, unknown>];
  "update:searchQuery": [value: string];
  change: [];
  refresh: [];
  "refresh-usage": [];
  sync: [];
  create: [];
  "import-data": [];
  "export-data": [];
  "show-error-passthrough": [];
  "show-tls-fingerprint-profiles": [];
  "sync-pending-list": [];
  "set-auto-refresh-enabled": [value: boolean];
  "set-auto-refresh-interval": [value: number];
  "toggle-column": [key: string];
  "update:view-mode": [value: AccountViewMode];
  "update:platform-count-sort-order": [value: AccountPlatformCountSortOrder];
  "toggle-group-view": [];
  "toggle-hide-limited": [];
  "open-limited-page": [];
}>();

const { t } = useI18n();

const showAutoRefreshDropdown = ref(false);
const showColumnDropdown = ref(false);
const autoRefreshDropdownRef = ref<HTMLElement | null>(null);
const columnDropdownRef = ref<HTMLElement | null>(null);

const nextPlatformCountSortOrder = computed<AccountPlatformCountSortOrder>(
  () =>
    props.platformCountSortOrder === "count_desc" ? "count_asc" : "count_desc",
);
const platformSortLabel = computed(() =>
  props.platformCountSortOrder === "count_desc"
    ? t("admin.accounts.platformSort.countDesc")
    : t("admin.accounts.platformSort.countAsc"),
);
const platformSortToggleTitle = computed(() =>
  nextPlatformCountSortOrder.value === "count_desc"
    ? t("admin.accounts.platformSort.toggleDesc")
    : t("admin.accounts.platformSort.toggleAsc"),
);

const handleFiltersUpdate = (value: Record<string, unknown>) => {
  emit("update:filters", value);
};

const handleSearchQueryUpdate = (value: string) => {
  emit("update:searchQuery", value);
};

const autoRefreshIntervalLabel = (sec: number) => {
  if (sec === 5) return t("admin.accounts.refreshInterval5s");
  if (sec === 10) return t("admin.accounts.refreshInterval10s");
  if (sec === 15) return t("admin.accounts.refreshInterval15s");
  if (sec === 30) return t("admin.accounts.refreshInterval30s");
  return `${sec}s`;
};

const toggleAutoRefreshDropdown = () => {
  showAutoRefreshDropdown.value = !showAutoRefreshDropdown.value;
  showColumnDropdown.value = false;
};

const toggleColumnDropdown = () => {
  showColumnDropdown.value = !showColumnDropdown.value;
  showAutoRefreshDropdown.value = false;
};

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as Node | null;
  if (
    columnDropdownRef.value &&
    target &&
    !columnDropdownRef.value.contains(target)
  ) {
    showColumnDropdown.value = false;
  }
  if (
    autoRefreshDropdownRef.value &&
    target &&
    !autoRefreshDropdownRef.value.contains(target)
  ) {
    showAutoRefreshDropdown.value = false;
  }
};

onMounted(() => {
  document.addEventListener("click", handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener("click", handleClickOutside);
});
</script>
