<template>
  <div class="overflow-x-auto">
    <table class="w-full text-sm">
      <thead class="bg-gray-50 text-xs uppercase tracking-wider text-gray-500 dark:bg-dark-950 dark:text-gray-400">
        <tr>
          <th v-if="isColumnVisible('created_at')" class="px-4 py-3 text-left">
            <button type="button" class="font-semibold uppercase" @click="emitSort('created_at')">
              {{ t("usage.time") }}{{ sortMark("created_at") }}
            </button>
          </th>
          <th v-if="isColumnVisible('key_name')" class="px-4 py-3 text-left">{{ t("usage.errors.keyName") }}</th>
          <th v-if="isColumnVisible('model')" class="px-4 py-3 text-left">
            <button type="button" class="font-semibold uppercase" @click="emitSort('model')">
              {{ t("usage.model") }}{{ sortMark("model") }}
            </button>
          </th>
          <th v-if="isColumnVisible('status')" class="px-4 py-3 text-left">
            <button type="button" class="font-semibold uppercase" @click="emitSort('status')">
              {{ t("usage.status") }}{{ sortMark("status") }}
            </button>
          </th>
          <th v-if="isColumnVisible('category')" class="px-4 py-3 text-left">{{ t("usage.errors.category") }}</th>
          <th v-if="isColumnVisible('endpoint')" class="px-4 py-3 text-left">{{ t("usage.endpoint") }}</th>
          <th v-if="isColumnVisible('client_ip')" class="px-4 py-3 text-left">{{ t("usage.failedRequests.clientIP") }}</th>
          <th v-if="isColumnVisible('group_name')" class="px-4 py-3 text-left">{{ t("usage.callGroup") }}</th>
          <th v-if="isColumnVisible('message')" class="px-4 py-3 text-left">{{ t("usage.errorMessage") }}</th>
          <th v-if="isColumnVisible('user_agent')" class="px-4 py-3 text-left">{{ t("usage.userAgent") }}</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-100 dark:divide-dark-800">
        <tr v-for="row in rows" :key="row.id" data-testid="failed-request-row">
          <td v-if="isColumnVisible('created_at')" class="whitespace-nowrap px-4 py-3 text-gray-700 dark:text-gray-200">
            {{ formatDateTime(row.created_at) }}
          </td>
          <td v-if="isColumnVisible('key_name')" class="px-4 py-3 text-gray-700 dark:text-gray-200">
            <span>{{ row.key_name || (row.api_key_id ? `#${row.api_key_id}` : "-") }}</span>
            <span
              v-if="row.key_deleted"
              class="ml-1 inline-flex rounded bg-rose-100 px-1 py-px text-[10px] font-medium text-rose-700 dark:bg-rose-500/15 dark:text-rose-300"
            >
              {{ t("usage.errors.keyDeleted") }}
            </span>
          </td>
          <td v-if="isColumnVisible('model')" class="px-4 py-3 text-gray-900 dark:text-white">
            <div class="max-w-48 truncate font-medium" :title="row.requested_model || row.model || '-'">
              {{ row.requested_model || row.model || "-" }}
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ row.platform || t("usage.unknown") }}
            </div>
          </td>
          <td v-if="isColumnVisible('status')" class="px-4 py-3">
            <span class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium" :class="statusCodeBadgeClass(row.status_code)">
              {{ formatStatus(row.status_code) }}
            </span>
          </td>
          <td v-if="isColumnVisible('category')" class="px-4 py-3 text-gray-700 dark:text-gray-200">
            {{ t("usage.errors.categories." + (row.category || mapErrorCategory(row.phase, row.type))) }}
          </td>
          <td v-if="isColumnVisible('endpoint')" class="px-4 py-3 text-gray-700 dark:text-gray-200">
            <div class="max-w-52 truncate" :title="row.inbound_endpoint || row.request_path || '-'">
              {{ row.inbound_endpoint || row.request_path || "-" }}
            </div>
            <div v-if="row.upstream_endpoint" class="mt-1 max-w-52 truncate text-xs text-gray-500 dark:text-gray-400" :title="row.upstream_endpoint">
              {{ row.upstream_endpoint }}
            </div>
          </td>
          <td v-if="isColumnVisible('client_ip')" class="px-4 py-3 font-mono text-xs text-gray-600 dark:text-gray-400">
            {{ row.client_ip || "-" }}
          </td>
          <td v-if="isColumnVisible('group_name')" class="px-4 py-3 text-gray-700 dark:text-gray-200">
            {{ row.group_name || "-" }}
          </td>
          <td v-if="isColumnVisible('message')" class="px-4 py-3 text-gray-700 dark:text-gray-200">
            <div class="max-w-96 whitespace-normal break-words">{{ row.message || "-" }}</div>
            <div v-if="row.request_id" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ row.request_id }}
            </div>
          </td>
          <td v-if="isColumnVisible('user_agent')" class="px-4 py-3 text-gray-700 dark:text-gray-200">
            <span class="block max-w-72 truncate" :title="row.user_agent || ''">
              {{ row.user_agent || "-" }}
            </span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import type { UserFailedRequest } from "@/api/usage";
import { formatDateTime } from "@/utils/format";
import { mapErrorSortKey, statusCodeBadgeClass } from "@/utils/errorBadges";
import { mapErrorCategory } from "@/utils/errorCategory";

const props = defineProps<{
  rows: UserFailedRequest[];
  hiddenColumns: Set<string>;
  sortBy: string;
  sortOrder: "asc" | "desc";
}>();

const emit = defineEmits<{
  (e: "sort", sortBy: string, sortOrder: "asc" | "desc"): void;
}>();

const { t } = useI18n();

function isColumnVisible(key: string): boolean {
  return !props.hiddenColumns.has(key);
}

function emitSort(key: string) {
  const mapped = mapErrorSortKey(key);
  const nextOrder = props.sortBy === mapped && props.sortOrder === "asc" ? "desc" : "asc";
  emit("sort", mapped, nextOrder);
}

function sortMark(key: string): string {
  if (props.sortBy !== mapErrorSortKey(key)) return "";
  return props.sortOrder === "asc" ? " ↑" : " ↓";
}

const formatStatus = (statusCode: number): string => {
  if (!Number.isFinite(statusCode) || statusCode <= 0) {
    return t("usage.failedRequests.failed");
  }
  return String(statusCode);
};
</script>
