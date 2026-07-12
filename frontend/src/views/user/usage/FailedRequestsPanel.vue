<template>
  <div class="card mb-4 overflow-hidden" data-testid="failed-requests-panel">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-6 py-4 dark:border-dark-800">
      <div>
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t("usage.failedRequests.title") }}
        </h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t("usage.failedRequests.description") }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <button
          type="button"
          class="btn btn-ghost btn-sm"
          :aria-label="collapsed ? t('common.expand') : t('common.collapse')"
          :aria-expanded="!collapsed"
          data-testid="failed-requests-collapse-toggle"
          @click="collapsed = !collapsed"
        >
          <Icon :name="collapsed ? 'chevronRight' : 'chevronDown'" size="sm" />
          <span class="sr-only">
            {{ collapsed ? t("common.expand") : t("common.collapse") }}
          </span>
        </button>
        <UsageColumnSettingsMenu
          :hidden-columns="hiddenColumns"
          :columns="columns"
          :always-visible-columns="alwaysVisibleColumns"
          @toggle-column="$emit('toggle-column', $event)"
        />
        <button
          type="button"
          class="btn btn-ghost btn-sm"
          :disabled="loading"
          :aria-label="t('usage.failedRequests.refresh')"
          @click="$emit('refresh')"
        >
          <Icon name="refresh" size="sm" />
          <span class="sr-only">{{ t("usage.failedRequests.refresh") }}</span>
        </button>
      </div>
    </div>

    <template v-if="!collapsed">
      <div class="grid grid-cols-1 gap-2 border-b border-gray-100 px-6 py-3 dark:border-dark-800 md:grid-cols-4">
        <input
          :value="filters.q"
          type="text"
          class="input text-sm"
          :placeholder="t('usage.failedRequests.searchPlaceholder')"
          @input="updateFilter('q', ($event.target as HTMLInputElement).value)"
        />
        <Select
          :model-value="filters.category"
          :options="categoryOptions"
          @update:model-value="updateFilter('category', String($event ?? ''))"
        />
        <Select
          :model-value="filters.statusCode"
          :options="statusOptions"
          @update:model-value="updateFilter('statusCode', $event == null ? null : Number($event))"
        />
      </div>

      <div v-if="loading" class="px-6 py-6 text-sm text-gray-500 dark:text-gray-400">
        {{ t("common.loading") }}
      </div>
      <div v-else-if="error" class="px-6 py-6 text-sm text-rose-600 dark:text-rose-300">
        {{ t("usage.failedRequests.failedToLoad") }}
      </div>
      <div v-else-if="rows.length === 0" class="px-6 py-6 text-sm text-gray-500 dark:text-gray-400">
        {{ t("usage.failedRequests.empty") }}
      </div>
      <FailedRequestsTable
        v-else
        :rows="rows"
        :hidden-columns="hiddenColumns"
        :sort-by="sortBy"
        :sort-order="sortOrder"
        @sort="(sortBy, sortOrder) => emit('sort', sortBy, sortOrder)"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import Select from "@/components/common/Select.vue";
import Icon from "@/components/icons/Icon.vue";
import UsageColumnSettingsMenu from "@/components/usage/UsageColumnSettingsMenu.vue";
import type { UserFailedRequest } from "@/api/usage";
import type { Column } from "@/components/common/types";
import { COMMON_ERROR_STATUS_CODES } from "@/utils/errorBadges";
import FailedRequestsTable from "./FailedRequestsTable.vue";

type FailedRequestFilters = {
  q: string;
  category: string;
  statusCode: number | null;
};

const props = defineProps<{
  rows: UserFailedRequest[];
  loading: boolean;
  error: boolean;
  filters: FailedRequestFilters;
  hiddenColumns: Set<string>;
  columns: Column[];
  alwaysVisibleColumns: string[];
  sortBy: string;
  sortOrder: "asc" | "desc";
}>();

const emit = defineEmits<{
  (e: "refresh"): void;
  (e: "filter-change", value: FailedRequestFilters): void;
  (e: "toggle-column", key: string): void;
  (e: "sort", sortBy: string, sortOrder: "asc" | "desc"): void;
}>();

const { t } = useI18n();
const collapsed = ref(false);

const categoryOptions = computed(() => {
  const codes = ["auth", "rate_limit", "quota", "invalid_request", "service_unavailable", "upstream", "internal", "cyber"];
  return [
    { value: "", label: t("usage.errors.allCategories") },
    ...codes.map((code) => ({ value: code, label: t(`usage.errors.categories.${code}`) })),
  ];
});

const statusOptions = computed(() => [
  { value: null, label: t("usage.errors.allStatuses") },
  ...COMMON_ERROR_STATUS_CODES.map((code) => ({ value: code, label: String(code) })),
]);

function updateFilter<K extends keyof FailedRequestFilters>(key: K, value: FailedRequestFilters[K]) {
  emit("filter-change", { ...props.filters, [key]: value });
}
</script>
