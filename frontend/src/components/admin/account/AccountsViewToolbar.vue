<template>
  <div>
    <div
      :class="[
        'flex flex-col gap-3',
        toolbarShellClass
      ]"
    >
      <AccountTableFilters
        :search-query="searchQuery"
        :filters="filters"
        :groups="groups"
        @update:filters="handleFiltersUpdate"
        @change="emit('change')"
        @update:search-query="handleSearchQueryUpdate"
      />
      <div
        :class="[
          'overflow-x-auto pb-1',
          actionsViewportClass
        ]"
      >
        <AccountTableActions
          :loading="loading"
          @refresh="emit('refresh')"
          @sync="emit('sync')"
          @create="emit('create')"
        >
          <template #after>
            <div
              :class="visualStyleToggleClass"
              data-account-visual-style-toggle="true"
            >
              <span :class="visualStyleLabelClass">
                {{ t("admin.accounts.accountVisualStyle") }}
              </span>
              <button
                type="button"
                class="rounded-full px-3 py-1 text-xs font-semibold transition"
                :class="
                  accountVisualPresetOverride === 'inherit'
                    ? 'bg-slate-900 text-white shadow-sm dark:bg-white dark:text-gray-900'
                    : 'text-gray-600 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-700'
                "
                :disabled="accountVisualStyleUpdating"
                @click="emit('set-account-visual-preset-override', 'inherit')"
              >
                {{ t("admin.accounts.accountVisualStyleInherit") }}
              </button>
              <button
                type="button"
                class="rounded-full px-3 py-1 text-xs font-semibold transition"
                :class="
                  accountVisualPresetOverride === 'classic'
                    ? 'bg-gray-900 text-white shadow-sm dark:bg-white dark:text-gray-900'
                    : 'text-gray-600 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-700'
                "
                :disabled="accountVisualStyleUpdating"
                @click="emit('set-account-visual-preset-override', 'classic')"
              >
                {{ t("admin.accounts.accountVisualStyleClassic") }}
              </button>
              <button
                type="button"
                class="rounded-full px-3 py-1 text-xs font-semibold transition"
                :class="
                  accountVisualPresetOverride === 'airy'
                    ? 'bg-primary-600 text-white shadow-sm'
                    : 'text-gray-600 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-dark-700'
                "
                :disabled="accountVisualStyleUpdating"
                @click="emit('set-account-visual-preset-override', 'airy')"
              >
                {{ t("admin.accounts.accountVisualStyleAiry") }}
              </button>
            </div>

            <div class="relative" ref="displayOptimizationDropdownRef">
              <button
                type="button"
                class="btn btn-secondary px-2 md:px-3"
                data-account-display-optimization-button="true"
                :title="t('admin.accounts.displayOptimization.title')"
                :disabled="accountDisplayPreferencesUpdating"
                @click="toggleDisplayOptimizationDropdown"
              >
                <Icon name="eye" size="sm" />
                <span class="hidden md:inline">
                  {{ t("admin.accounts.displayOptimization.button") }}
                </span>
              </button>
            </div>

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
              :disabled="loading"
              data-one-click-detect-button="true"
              @click="emit('one-click-detect')"
            >
              <Icon name="play" size="sm" />
              <span>{{ t("admin.accounts.batchTest.oneClickDetect") }}</span>
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

            <div
              :class="daily5HToggleClass"
              :title="t('admin.accounts.daily5h.toolbarHint')"
            >
              <div class="hidden md:block">
                <div class="text-xs font-medium text-gray-900 dark:text-white">
                  {{ t("admin.accounts.daily5h.toolbarLabel") }}
                </div>
              </div>
              <button
                type="button"
                class="relative inline-flex h-6 w-11 flex-shrink-0 rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2"
                :class="[daily5HTriggerEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600']"
                role="switch"
                :aria-checked="daily5HTriggerEnabled"
                :disabled="loading || daily5HTriggerBusy"
                data-daily5h-toggle="true"
                @click="emit('toggle-daily-5h-trigger')"
              >
                <span
                  class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
                  :class="[daily5HTriggerEnabled ? 'translate-x-5' : 'translate-x-0']"
                />
              </button>
              <button
                type="button"
                class="btn btn-secondary px-2"
                :disabled="loading || daily5HTriggerBusy"
                :title="t('admin.accounts.daily5h.settingsButtonTitle')"
                data-daily5h-settings="true"
                @click="emit('open-daily-5h-settings')"
              >
                <Icon name="cog" size="sm" />
              </button>
            </div>

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
            </div>

            <button
              type="button"
              class="btn btn-secondary"
              data-account-import-button="true"
              @click="emit('import-data')"
            >
              {{ t("admin.accounts.dataImport") }}
            </button>

            <button
              type="button"
              class="btn btn-secondary"
              data-account-export-button="true"
              @click="emit('export-data')"
            >
              {{
                selectedCount
                  ? t("admin.accounts.dataExportSelected")
                  : t("admin.accounts.dataExport")
              }}
            </button>

            <div class="relative" ref="moreActionsDropdownRef">
              <button
                type="button"
                class="btn btn-secondary px-2 md:px-3"
                :title="t('common.more')"
                data-more-actions-button="true"
                @click="toggleMoreActionsDropdown"
              >
                <Icon name="more" size="sm" class="md:mr-1.5" />
                <span class="hidden md:inline">{{ t("common.more") }}</span>
              </button>
            </div>

            <div class="relative" ref="columnDropdownRef"></div>
          </template>
        </AccountTableActions>
      </div>
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

  <Teleport to="body">
    <div v-if="showAutoRefreshDropdown">
      <div class="fixed inset-0 z-40" @click="closeFloatingMenus"></div>
      <div
        class="fixed z-50 w-56 rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
        :style="floatingAutoRefreshStyle"
        @click.stop
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
          <div class="my-1 border-t border-gray-100 dark:border-gray-700"></div>
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
  </Teleport>

  <Teleport to="body">
    <div v-if="showDisplayOptimizationDropdown">
      <div class="fixed inset-0 z-40" @click="closeFloatingMenus"></div>
      <div
        class="fixed z-50 w-72 rounded-xl border border-gray-200 bg-white p-3 shadow-lg dark:border-gray-700 dark:bg-gray-800"
        :style="floatingDisplayOptimizationStyle"
        data-account-display-optimization-panel="true"
        @click.stop
      >
        <div class="mb-3 flex items-center justify-between">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t("admin.accounts.displayOptimization.title") }}
          </div>
          <button
            type="button"
            class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-200"
            :aria-label="t('common.close')"
            @click="showDisplayOptimizationDropdown = false"
          >
            <Icon name="x" size="xs" />
          </button>
        </div>

        <div class="space-y-4">
          <div>
            <div class="mb-2 text-xs font-semibold text-gray-500 dark:text-gray-400">
              {{ t("admin.accounts.displayOptimization.todayStats") }}
            </div>
            <div class="grid grid-cols-4 gap-1.5">
              <label
                v-for="windowKey in accountTodayStatsWindowOptions"
                :key="windowKey"
                class="flex cursor-pointer items-center justify-center gap-1.5 rounded-lg border px-2 py-2 text-xs font-semibold transition"
                :class="
                  draftTodayStatsWindows.includes(windowKey)
                    ? 'border-primary-200 bg-primary-50 text-primary-700 dark:border-primary-400/30 dark:bg-primary-500/10 dark:text-primary-200'
                    : 'border-gray-200 bg-gray-50 text-gray-600 hover:bg-gray-100 dark:border-gray-700 dark:bg-gray-900/40 dark:text-gray-300 dark:hover:bg-gray-700'
                "
              >
                <input
                  type="checkbox"
                  class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                  :checked="draftTodayStatsWindows.includes(windowKey)"
                  :disabled="isLastSelectedTodayStatsWindow(windowKey)"
                  @change="toggleDraftTodayStatsWindow(windowKey)"
                />
                <span>{{ t(`admin.accounts.displayOptimization.windows.${windowKey}`) }}</span>
              </label>
            </div>
          </div>

          <div>
            <div class="mb-2 text-xs font-semibold text-gray-500 dark:text-gray-400">
              {{ t("admin.accounts.displayOptimization.todayStatsCycleMode") }}
            </div>
            <div class="inline-flex w-full rounded-lg border border-gray-200 bg-gray-50 p-1 dark:border-gray-700 dark:bg-gray-900/40">
              <button
                v-for="mode in accountTodayStatsCycleModeOptions"
                :key="mode"
                type="button"
                class="flex-1 rounded-md px-3 py-1.5 text-xs font-semibold transition"
                :class="
                  draftTodayStatsCycleMode === mode
                    ? 'bg-white text-gray-900 shadow-sm dark:bg-gray-700 dark:text-white'
                    : 'text-gray-500 hover:text-gray-700 dark:text-gray-300 dark:hover:text-white'
                "
                @click="draftTodayStatsCycleMode = mode"
              >
                {{ t(`admin.accounts.displayOptimization.cycleModes.${mode}`) }}
              </button>
            </div>
          </div>

          <div>
            <div class="mb-2 text-xs font-semibold text-gray-500 dark:text-gray-400">
              {{ t("admin.accounts.displayOptimization.groupDisplay") }}
            </div>
            <div class="inline-flex w-full rounded-lg border border-gray-200 bg-gray-50 p-1 dark:border-gray-700 dark:bg-gray-900/40">
              <button
                v-for="mode in accountGroupDisplayModeOptions"
                :key="mode"
                type="button"
                class="flex-1 rounded-md px-3 py-1.5 text-xs font-semibold transition"
                :class="
                  draftGroupDisplayMode === mode
                    ? 'bg-white text-gray-900 shadow-sm dark:bg-gray-700 dark:text-white'
                    : 'text-gray-500 hover:text-gray-700 dark:text-gray-300 dark:hover:text-white'
                "
                @click="draftGroupDisplayMode = mode"
              >
                {{ t(`admin.accounts.displayOptimization.groupModes.${mode}`) }}
              </button>
            </div>
          </div>

          <div>
            <div class="mb-2 text-xs font-semibold text-gray-500 dark:text-gray-400">
              {{ t("admin.accounts.displayOptimization.statusDisplay") }}
            </div>
            <div class="inline-flex w-full rounded-lg border border-gray-200 bg-gray-50 p-1 dark:border-gray-700 dark:bg-gray-900/40">
              <button
                v-for="mode in accountStatusDisplayModeOptions"
                :key="mode"
                type="button"
                class="flex-1 rounded-md px-3 py-1.5 text-xs font-semibold transition"
                :class="
                  draftStatusDisplayMode === mode
                    ? 'bg-white text-gray-900 shadow-sm dark:bg-gray-700 dark:text-white'
                    : 'text-gray-500 hover:text-gray-700 dark:text-gray-300 dark:hover:text-white'
                "
                @click="draftStatusDisplayMode = mode"
              >
                {{ t(`admin.accounts.displayOptimization.statusModes.${mode}`) }}
              </button>
            </div>
          </div>

          <button
            type="button"
            class="btn btn-primary w-full justify-center"
            data-account-display-optimization-save="true"
            :disabled="accountDisplayPreferencesUpdating"
            @click="saveDisplayOptimization"
          >
            {{ t("admin.accounts.displayOptimization.save") }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>

  <Teleport to="body">
    <div v-if="showMoreActionsDropdown">
      <div class="fixed inset-0 z-40" @click="closeFloatingMenus"></div>
      <div
        class="fixed z-50 w-56 rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
        :style="floatingMoreActionsStyle"
        @click.stop
      >
        <div class="p-2">
          <button
            type="button"
            class="flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
            data-account-realtime-toggle="true"
            @click="handleToggleAccountRealtimeCountdown"
          >
            <span>{{ t("admin.accounts.accountRealtimeCountdown") }}</span>
            <span
              class="relative inline-flex h-5 w-9 flex-shrink-0 rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out"
              :class="[
                accountRealtimeCountdownEnabled
                  ? 'bg-primary-600'
                  : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
                :class="[accountRealtimeCountdownEnabled ? 'translate-x-4' : 'translate-x-0']"
              />
            </span>
          </button>
          <button
            type="button"
            class="flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
            data-platform-sort-button="true"
            :title="platformSortToggleTitle"
            @click="
              emit(
                'update:platform-count-sort-order',
                nextPlatformCountSortOrder,
              );
              showMoreActionsDropdown = false;
            "
          >
            <span>{{ platformSortLabel }}</span>
          </button>
          <button
            v-if="showLimitedControls"
            type="button"
            class="flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
            @click="
              emit('toggle-hide-limited');
              showMoreActionsDropdown = false;
            "
          >
            <span>
              {{
                hideLimitedAccounts
                  ? t("admin.accounts.limited.hideToggleOn")
                  : t("admin.accounts.limited.hideToggleOff")
              }}
            </span>
          </button>
          <button
            type="button"
            class="flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
            :title="t('admin.errorPassthrough.title')"
            @click="handleMoreAction('show-error-passthrough')"
          >
            <span>{{ t("admin.errorPassthrough.title") }}</span>
          </button>
          <button
            type="button"
            class="flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
            :title="t('admin.tlsFingerprintProfiles.title')"
            @click="handleMoreAction('show-tls-fingerprint-profiles')"
          >
            <span>{{ t("admin.tlsFingerprintProfiles.title") }}</span>
          </button>
          <button
            type="button"
            class="flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
            :title="t('admin.users.columnSettings')"
            @click="toggleColumnDropdownFromMore"
          >
            <span>{{ t("admin.users.columnSettings") }}</span>
            <Icon name="menu" size="sm" />
          </button>
          <button
            type="button"
            class="flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
            @click="handleMoreAction('sync')"
          >
            <span>{{ t("admin.accounts.syncFromCrs") }}</span>
          </button>
          <button
            v-if="!selectedCount"
            type="button"
            class="flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
            @click="handleMoreAction('bulk-edit-filtered')"
          >
            <span>{{ t("admin.accounts.bulkEdit.editCurrentCategory") }}</span>
            <span class="text-xs text-gray-400 dark:text-gray-500">
              {{ filteredBulkEditTotal }}
            </span>
          </button>
          <label
            v-if="!selectedCount"
            class="flex w-full cursor-pointer items-center justify-between gap-3 rounded-md px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
            :class="[filteredBulkEditExcludeGroupedDisabled ? 'cursor-not-allowed opacity-60' : '']"
            :title="filteredBulkEditExcludeGroupedDisabled ? t('admin.accounts.bulkEdit.excludeGroupedSpecificGroupDisabled') : ''"
          >
            <span>{{ t("admin.accounts.bulkEdit.excludeGrouped") }}</span>
            <input
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:cursor-not-allowed"
              :checked="filteredBulkEditExcludeGrouped"
              :disabled="filteredBulkEditExcludeGroupedDisabled"
              data-account-filtered-bulk-edit-exclude-grouped="true"
              @change="handleFilteredBulkEditExcludeGroupedChange"
            />
          </label>
        </div>
      </div>
    </div>
  </Teleport>

  <Teleport to="body">
    <div v-if="showColumnDropdown">
      <div class="fixed inset-0 z-40" @click="closeFloatingMenus"></div>
      <div
        class="fixed z-50 w-48 rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
        :style="floatingColumnStyle"
        @click.stop
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
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import type {
  AdminGroup,
  AccountGroupDisplayMode,
  AccountStatusDisplayMode,
  AccountTodayStatsCycleMode,
  AccountTodayStatsWindow,
  VisualPreset,
  VisualPresetPreference,
  AccountPlatformCountSortOrder,
  AccountViewMode,
} from "@/types";
import Icon from "@/components/icons/Icon.vue";
import AccountViewModeToggle from "./AccountViewModeToggle.vue";
import AccountTableActions from "./AccountTableActions.vue";
import AccountTableFilters from "./AccountTableFilters.vue";
import { resolveToolbarDropdownPosition } from "@/utils/toolbarDropdownPosition";

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
  daily5HTriggerEnabled?: boolean;
  daily5HTriggerBusy?: boolean;
  accountRealtimeCountdownEnabled?: boolean;
  accountVisualPresetOverride?: VisualPresetPreference;
  visualStyle?: VisualPreset;
  accountVisualStyleUpdating?: boolean;
  accountTodayStatsWindows?: AccountTodayStatsWindow[];
  accountTodayStatsCycleMode?: AccountTodayStatsCycleMode;
  accountGroupDisplayMode?: AccountGroupDisplayMode;
  accountStatusDisplayMode?: AccountStatusDisplayMode;
  accountDisplayPreferencesUpdating?: boolean;
  filteredBulkEditTotal?: number;
  filteredBulkEditExcludeGrouped?: boolean;
  filteredBulkEditExcludeGroupedDisabled?: boolean;
}>();

const emit = defineEmits<{
  "update:filters": [value: Record<string, unknown>];
  "update:searchQuery": [value: string];
  change: [];
  refresh: [];
  "refresh-usage": [];
  "bulk-edit-filtered": [];
  "one-click-detect": [];
  "update:filtered-bulk-edit-exclude-grouped": [value: boolean];
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
  "toggle-daily-5h-trigger": [];
  toggleDaily5hTrigger: [];
  "open-daily-5h-settings": [];
  openDaily5hSettings: [];
  "toggle-account-realtime-countdown": [];
  "set-account-visual-preset-override": [value: VisualPresetPreference];
  "save-account-display-preferences": [value: {
    todayStatsWindows: AccountTodayStatsWindow[];
    todayStatsCycleMode: AccountTodayStatsCycleMode;
    groupDisplayMode: AccountGroupDisplayMode;
    statusDisplayMode: AccountStatusDisplayMode;
  }];
}>();

const { t } = useI18n();

const showAutoRefreshDropdown = ref(false);
const showDisplayOptimizationDropdown = ref(false);
const showColumnDropdown = ref(false);
const showMoreActionsDropdown = ref(false);
const autoRefreshDropdownRef = ref<HTMLElement | null>(null);
const displayOptimizationDropdownRef = ref<HTMLElement | null>(null);
const columnDropdownRef = ref<HTMLElement | null>(null);
const moreActionsDropdownRef = ref<HTMLElement | null>(null);
const AUTO_REFRESH_PANEL_WIDTH = 224;
const AUTO_REFRESH_PANEL_HEIGHT = 240;
const DISPLAY_OPTIMIZATION_PANEL_WIDTH = 288;
const DISPLAY_OPTIMIZATION_PANEL_HEIGHT = 432;
const MORE_ACTIONS_PANEL_WIDTH = 224;
const MORE_ACTIONS_PANEL_HEIGHT = 408;
const COLUMN_PANEL_WIDTH = 192;
const COLUMN_PANEL_HEIGHT = 360;
const accountTodayStatsWindowOptions: AccountTodayStatsWindow[] = [
  "today",
  "weekly",
  "monthly",
  "total",
];
const accountTodayStatsCycleModeOptions: AccountTodayStatsCycleMode[] = [
  "calendar",
  "fixed",
];
const accountGroupDisplayModeOptions: AccountGroupDisplayMode[] = [
  "full",
  "icon",
];
const accountStatusDisplayModeOptions: AccountStatusDisplayMode[] = [
  "simple",
  "detailed",
];
const normalizeTodayStatsWindows = (
  values?: AccountTodayStatsWindow[],
): AccountTodayStatsWindow[] => {
  const selected = new Set(values || []);
  const normalized = accountTodayStatsWindowOptions.filter((value) =>
    selected.has(value),
  );
  return normalized.length > 0 ? normalized : [...accountTodayStatsWindowOptions];
};
const draftTodayStatsWindows = ref<AccountTodayStatsWindow[]>(
  normalizeTodayStatsWindows(props.accountTodayStatsWindows),
);
const draftTodayStatsCycleMode = ref<AccountTodayStatsCycleMode>(
  props.accountTodayStatsCycleMode === "fixed" ? "fixed" : "calendar",
);
const draftGroupDisplayMode = ref<AccountGroupDisplayMode>(
  props.accountGroupDisplayMode === "icon" ? "icon" : "full",
);
const draftStatusDisplayMode = ref<AccountStatusDisplayMode>(
  props.accountStatusDisplayMode === "simple" ? "simple" : "detailed",
);

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
const toolbarShellClass = computed(() =>
  props.visualStyle === "airy"
    ? "rounded-[1.75rem] border border-slate-200/75 bg-[linear-gradient(135deg,rgba(255,255,255,0.97),rgba(248,250,252,0.92))] p-4 dark:border-slate-700/80 dark:bg-[linear-gradient(135deg,rgba(30,41,59,0.82),rgba(15,23,42,0.72))]"
    : "rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800",
);
const actionsViewportClass = computed(() =>
  props.visualStyle === "airy" ? "-mx-1 px-1" : "",
);
const visualStyleToggleClass = computed(() =>
  props.visualStyle === "airy"
    ? "inline-flex items-center gap-1 rounded-full border border-slate-200/80 bg-slate-50/95 p-1 dark:border-slate-700/80 dark:bg-slate-800/80"
    : "inline-flex items-center gap-1 rounded-full border border-gray-200 bg-gray-50 p-1 dark:border-dark-600 dark:bg-dark-900/60",
);
const visualStyleLabelClass = computed(() =>
  props.visualStyle === "airy"
    ? "px-2 text-xs font-medium text-slate-600 dark:text-slate-200"
    : "px-2 text-xs font-medium text-gray-500 dark:text-gray-300",
);
const daily5HToggleClass = computed(() =>
  props.visualStyle === "airy"
    ? "flex items-center gap-2 rounded-2xl border border-slate-200/80 bg-slate-50/95 px-3 py-2 dark:border-slate-700/80 dark:bg-slate-800/80"
    : "flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-3 py-2 dark:border-dark-600 dark:bg-dark-800",
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
  showDisplayOptimizationDropdown.value = false;
  showColumnDropdown.value = false;
  showMoreActionsDropdown.value = false;
};

const syncDisplayOptimizationDraft = () => {
  draftTodayStatsWindows.value = normalizeTodayStatsWindows(
    props.accountTodayStatsWindows,
  );
  draftGroupDisplayMode.value =
    props.accountGroupDisplayMode === "icon" ? "icon" : "full";
  draftTodayStatsCycleMode.value =
    props.accountTodayStatsCycleMode === "fixed" ? "fixed" : "calendar";
  draftStatusDisplayMode.value =
    props.accountStatusDisplayMode === "simple" ? "simple" : "detailed";
};

const toggleDisplayOptimizationDropdown = () => {
  syncDisplayOptimizationDraft();
  showDisplayOptimizationDropdown.value = !showDisplayOptimizationDropdown.value;
  showAutoRefreshDropdown.value = false;
  showColumnDropdown.value = false;
  showMoreActionsDropdown.value = false;
};

const toggleColumnDropdownFromMore = () => {
  showColumnDropdown.value = !showColumnDropdown.value;
  showAutoRefreshDropdown.value = false;
  showDisplayOptimizationDropdown.value = false;
  showMoreActionsDropdown.value = false;
};

const toggleMoreActionsDropdown = () => {
  showMoreActionsDropdown.value = !showMoreActionsDropdown.value;
  showAutoRefreshDropdown.value = false;
  showDisplayOptimizationDropdown.value = false;
  showColumnDropdown.value = false;
};

const handleToggleAccountRealtimeCountdown = () => {
  showMoreActionsDropdown.value = false;
  emit("toggle-account-realtime-countdown");
};

const closeFloatingMenus = () => {
  showAutoRefreshDropdown.value = false;
  showDisplayOptimizationDropdown.value = false;
  showMoreActionsDropdown.value = false;
  showColumnDropdown.value = false;
};

const isLastSelectedTodayStatsWindow = (windowKey: AccountTodayStatsWindow) =>
  draftTodayStatsWindows.value.includes(windowKey) &&
  draftTodayStatsWindows.value.length <= 1;

const toggleDraftTodayStatsWindow = (windowKey: AccountTodayStatsWindow) => {
  if (draftTodayStatsWindows.value.includes(windowKey)) {
    if (draftTodayStatsWindows.value.length <= 1) {
      return;
    }
    draftTodayStatsWindows.value = draftTodayStatsWindows.value.filter(
      (value) => value !== windowKey,
    );
    return;
  }
  const selected = new Set([...draftTodayStatsWindows.value, windowKey]);
  draftTodayStatsWindows.value = accountTodayStatsWindowOptions.filter((value) =>
    selected.has(value),
  );
};

const handleFilteredBulkEditExcludeGroupedChange = (event: Event) => {
  emit(
    "update:filtered-bulk-edit-exclude-grouped",
    (event.target as HTMLInputElement).checked,
  );
};

const saveDisplayOptimization = () => {
  emit("save-account-display-preferences", {
    todayStatsWindows: normalizeTodayStatsWindows(
      draftTodayStatsWindows.value,
    ),
    todayStatsCycleMode: draftTodayStatsCycleMode.value,
    groupDisplayMode: draftGroupDisplayMode.value,
    statusDisplayMode: draftStatusDisplayMode.value,
  });
  showDisplayOptimizationDropdown.value = false;
};

const floatingAutoRefreshStyle = computed(() => {
  const position = resolveToolbarDropdownPosition({
    target: autoRefreshDropdownRef.value,
    panelWidth: AUTO_REFRESH_PANEL_WIDTH,
    panelHeight: AUTO_REFRESH_PANEL_HEIGHT,
  });
  return {
    top: `${position.top}px`,
    left: `${position.left}px`,
  };
});

const floatingDisplayOptimizationStyle = computed(() => {
  const position = resolveToolbarDropdownPosition({
    target: displayOptimizationDropdownRef.value,
    panelWidth: DISPLAY_OPTIMIZATION_PANEL_WIDTH,
    panelHeight: DISPLAY_OPTIMIZATION_PANEL_HEIGHT,
  });
  return {
    top: `${position.top}px`,
    left: `${position.left}px`,
  };
});

const floatingMoreActionsStyle = computed(() => {
  const position = resolveToolbarDropdownPosition({
    target: moreActionsDropdownRef.value,
    panelWidth: MORE_ACTIONS_PANEL_WIDTH,
    panelHeight: MORE_ACTIONS_PANEL_HEIGHT,
  });
  return {
    top: `${position.top}px`,
    left: `${position.left}px`,
  };
});

const floatingColumnStyle = computed(() => {
  const position = resolveToolbarDropdownPosition({
    target: moreActionsDropdownRef.value || columnDropdownRef.value,
    panelWidth: COLUMN_PANEL_WIDTH,
    panelHeight: COLUMN_PANEL_HEIGHT,
  });
  return {
    top: `${position.top}px`,
    left: `${position.left}px`,
  };
});

const handleMoreAction = (
  action:
    | "show-error-passthrough"
    | "show-tls-fingerprint-profiles"
    | "sync"
    | "bulk-edit-filtered",
) => {
  showMoreActionsDropdown.value = false;
  if (action === "show-error-passthrough") {
    emit("show-error-passthrough");
    return;
  }
  if (action === "show-tls-fingerprint-profiles") {
    emit("show-tls-fingerprint-profiles");
    return;
  }
  if (action === "sync") {
    emit("sync");
    return;
  }
  if (action === "bulk-edit-filtered") {
    emit("bulk-edit-filtered");
  }
};

</script>
