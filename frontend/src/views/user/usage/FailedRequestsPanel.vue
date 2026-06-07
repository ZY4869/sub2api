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

    <div v-if="loading" class="px-6 py-6 text-sm text-gray-500 dark:text-gray-400">
      {{ t("common.loading") }}
    </div>
    <div v-else-if="error" class="px-6 py-6 text-sm text-rose-600 dark:text-rose-300">
      {{ t("usage.failedRequests.failedToLoad") }}
    </div>
    <div v-else-if="rows.length === 0" class="px-6 py-6 text-sm text-gray-500 dark:text-gray-400">
      {{ t("usage.failedRequests.empty") }}
    </div>
    <div v-else class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 text-xs uppercase tracking-wider text-gray-500 dark:bg-dark-950 dark:text-gray-400">
          <tr>
            <th class="px-4 py-3 text-left">{{ t("usage.time") }}</th>
            <th class="px-4 py-3 text-left">{{ t("usage.model") }}</th>
            <th class="px-4 py-3 text-left">{{ t("usage.status") }}</th>
            <th class="px-4 py-3 text-left">{{ t("usage.failedRequests.phase") }}</th>
            <th class="px-4 py-3 text-left">{{ t("usage.endpoint") }}</th>
            <th class="px-4 py-3 text-left">{{ t("usage.errorMessage") }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-dark-800">
          <tr
            v-for="row in rows"
            :key="row.id"
            data-testid="failed-request-row"
          >
            <td class="whitespace-nowrap px-4 py-3 text-gray-700 dark:text-gray-200">
              {{ formatDateTime(row.created_at) }}
            </td>
            <td class="px-4 py-3 text-gray-900 dark:text-white">
              <div class="max-w-48 truncate font-medium" :title="row.requested_model || row.model || '-'">
                {{ row.requested_model || row.model || "-" }}
              </div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ row.platform || t("usage.unknown") }}
              </div>
            </td>
            <td class="px-4 py-3">
              <span class="inline-flex rounded-full bg-rose-100 px-2 py-0.5 text-xs font-medium text-rose-700 dark:bg-rose-500/15 dark:text-rose-300">
                {{ formatStatus(row.status_code) }}
              </span>
            </td>
            <td class="px-4 py-3 text-gray-700 dark:text-gray-200">
              <div>{{ row.phase || "-" }}</div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ row.error_source || row.error_owner || "-" }}
              </div>
            </td>
            <td class="px-4 py-3 text-gray-700 dark:text-gray-200">
              <div class="max-w-52 truncate" :title="row.inbound_endpoint || row.request_path || '-'">
                {{ row.inbound_endpoint || row.request_path || "-" }}
              </div>
              <div
                v-if="row.upstream_endpoint"
                class="mt-1 max-w-52 truncate text-xs text-gray-500 dark:text-gray-400"
                :title="row.upstream_endpoint"
              >
                {{ row.upstream_endpoint }}
              </div>
            </td>
            <td class="px-4 py-3 text-gray-700 dark:text-gray-200">
              <div class="max-w-96 whitespace-normal break-words">
                {{ row.message || "-" }}
              </div>
              <div v-if="row.request_id" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ row.request_id }}
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import type { UserFailedRequest } from "@/api/usage";
import Icon from "@/components/icons/Icon.vue";
import { formatDateTime } from "@/utils/format";

defineProps<{
  rows: UserFailedRequest[];
  loading: boolean;
  error: boolean;
}>();

defineEmits<{
  (e: "refresh"): void;
}>();

const { t } = useI18n();

const formatStatus = (statusCode: number): string => {
  if (!Number.isFinite(statusCode) || statusCode <= 0) {
    return t("usage.failedRequests.failed");
  }
  return String(statusCode);
};
</script>
