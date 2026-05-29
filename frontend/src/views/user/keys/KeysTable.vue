<template>
  <DataTable :columns="columns" :data="apiKeys" :loading="loading">
    <template #cell-key="{ value, row }">
      <div class="flex items-center gap-2">
        <code class="code text-xs">{{ maskKey(value) }}</code>
        <button
          @click="$emit('copy', value, row.id)"
          class="rounded-lg p-1 transition-colors hover:bg-gray-100 dark:hover:bg-dark-700"
          :class="
            copiedKeyId === row.id
              ? 'text-green-500'
              : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'
          "
          :title="copiedKeyId === row.id ? t('keys.copied') : t('keys.copyToClipboard')"
        >
          <Icon v-if="copiedKeyId === row.id" name="check" size="sm" :stroke-width="2" />
          <Icon v-else name="clipboard" size="sm" />
        </button>
      </div>
    </template>

    <template #cell-name="{ value, row }">
      <div class="flex items-center gap-1.5">
        <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
        <Icon
          v-if="row.ip_whitelist?.length > 0 || row.ip_blacklist?.length > 0"
          name="shield"
          size="sm"
          class="text-blue-500"
          :title="t('keys.ipRestrictionEnabled')"
        />
      </div>
    </template>

    <template #cell-group="{ row }">
      <div v-if="getDisplayBindings(row).length" class="space-y-1">
        <div class="flex flex-wrap gap-1.5">
          <div
            v-for="binding in getDisplayBindings(row).slice(0, 2)"
            :key="`${row.id}-${binding.group_id}`"
            class="flex items-center gap-1.5"
          >
            <GroupBadge
              :name="resolveGroup(binding.group_id)?.name || binding.group_name || `#${binding.group_id}`"
              :platform="resolveGroup(binding.group_id)?.platform || binding.platform"
              :subscription-type="resolveGroup(binding.group_id)?.subscription_type"
              :rate-multiplier="resolveGroup(binding.group_id)?.rate_multiplier"
              :user-rate-multiplier="resolveGroup(binding.group_id) ? userGroupRates[binding.group_id] : undefined"
            />
            <span class="rounded-full bg-gray-100 px-2 py-0.5 text-[11px] text-gray-500 dark:bg-dark-700 dark:text-gray-300">
              P{{ binding.priority ?? resolveGroup(binding.group_id)?.priority ?? 1 }}
            </span>
          </div>
          <span
            v-if="getDisplayBindings(row).length > 2"
            class="rounded-full bg-gray-100 px-2 py-0.5 text-[11px] text-gray-500 dark:bg-dark-700 dark:text-gray-300"
          >
            +{{ getDisplayBindings(row).length - 2 }}
          </span>
        </div>
        <button
          type="button"
          class="text-xs text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400"
          @click="$emit('edit', row)"
        >
          {{ isAdminMode ? t("admin.users.editGroupBindings") : t("keys.editKey") }}
        </button>
      </div>
      <button
        v-else
        type="button"
        class="text-sm text-gray-400 transition-colors hover:text-primary-500 dark:text-dark-500"
        @click="$emit('edit', row)"
      >
        {{ t("keys.noGroup") }}
      </button>
    </template>

    <template #cell-usage="{ row }">
      <div class="text-sm">
        <div class="flex items-center gap-1.5">
          <span class="text-gray-500 dark:text-gray-400">{{ t("keys.today") }}:</span>
          <span class="font-medium text-gray-900 dark:text-white">
            ${{ (usageStats[row.id]?.today_actual_cost ?? 0).toFixed(4) }}
          </span>
        </div>
        <div class="mt-0.5 flex items-center gap-1.5">
          <span class="text-gray-500 dark:text-gray-400">{{ t("keys.total") }}:</span>
          <span class="font-medium text-gray-900 dark:text-white">
            ${{ (usageStats[row.id]?.total_actual_cost ?? 0).toFixed(4) }}
          </span>
        </div>
        <div v-if="row.quota > 0" class="mt-1.5">
          <div class="flex items-center gap-1.5">
            <span class="text-gray-500 dark:text-gray-400">{{ t("keys.quota") }}:</span>
            <span
              :class="[
                'font-medium',
                row.quota_used >= row.quota
                  ? 'text-red-500'
                  : row.quota_used >= row.quota * 0.8
                    ? 'text-yellow-500'
                    : 'text-gray-900 dark:text-white',
              ]"
            >
              ${{ row.quota_used?.toFixed(2) || "0.00" }} / ${{ row.quota?.toFixed(2) }}
            </span>
          </div>
          <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
            <div
              :class="[
                'h-full rounded-full transition-all',
                row.quota_used >= row.quota
                  ? 'bg-red-500'
                  : row.quota_used >= row.quota * 0.8
                    ? 'bg-yellow-500'
                    : 'bg-primary-500',
              ]"
              :style="{ width: Math.min((row.quota_used / row.quota) * 100, 100) + '%' }"
            />
          </div>
        </div>
      </div>
    </template>

    <template #cell-rate_limit="{ row }">
      <div
        v-if="row.rate_limit_5h > 0 || row.rate_limit_1d > 0 || row.rate_limit_7d > 0"
        class="space-y-1.5 min-w-[140px]"
      >
        <ApiKeyRateLimitWindow
          v-if="row.rate_limit_5h > 0"
          label="5h"
          :usage="row.usage_5h"
          :limit="row.rate_limit_5h"
          :reset-at="row.reset_5h_at"
          :format-reset-time="formatResetTime"
        />
        <ApiKeyRateLimitWindow
          v-if="row.rate_limit_1d > 0"
          label="1d"
          :usage="row.usage_1d"
          :limit="row.rate_limit_1d"
          :reset-at="row.reset_1d_at"
          :format-reset-time="formatResetTime"
        />
        <ApiKeyRateLimitWindow
          v-if="row.rate_limit_7d > 0"
          label="7d"
          :usage="row.usage_7d"
          :limit="row.rate_limit_7d"
          :reset-at="row.reset_7d_at"
          :format-reset-time="formatResetTime"
        />
        <button
          v-if="row.usage_5h > 0 || row.usage_1d > 0 || row.usage_7d > 0"
          @click.stop="$emit('reset-rate-limit', row)"
          class="mt-0.5 inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
          :title="t('keys.resetRateLimitUsage')"
        >
          <Icon name="refresh" size="xs" />
          {{ t("keys.resetUsage") }}
        </button>
      </div>
      <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
    </template>

    <template #cell-expires_at="{ value }">
      <span
        v-if="value"
        :class="[
          'text-sm',
          new Date(value) < new Date()
            ? 'text-red-500 dark:text-red-400'
            : 'text-gray-500 dark:text-dark-400',
        ]"
      >
        {{ formatDateTime(value) }}
      </span>
      <span v-else class="text-sm text-gray-400 dark:text-dark-500">{{ t("keys.noExpiration") }}</span>
    </template>

    <template #cell-status="{ value }">
      <span
        :class="[
          'badge',
          value === 'active'
            ? 'badge-success'
            : value === 'quota_exhausted'
              ? 'badge-warning'
              : value === 'expired'
                ? 'badge-danger'
                : 'badge-gray',
        ]"
      >
        {{ t("keys.status." + value) }}
      </span>
    </template>

    <template #cell-last_used_at="{ value }">
      <span v-if="value" class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
      <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
    </template>

    <template #cell-created_at="{ value }">
      <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
    </template>

    <template #cell-actions="{ row }">
      <div class="flex items-center gap-1">
        <button
          @click="$emit('use-key', row)"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400"
        >
          <Icon name="terminal" size="sm" />
          <span class="text-xs">{{ t("keys.useKey") }}</span>
        </button>
        <button
          v-if="!hideCcsImportButton"
          @click="$emit('import-ccswitch', row)"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
        >
          <Icon name="upload" size="sm" />
          <span class="text-xs">{{ t("keys.importToCcSwitch") }}</span>
        </button>
        <button
          @click="$emit('toggle-status', row)"
          :class="[
            'flex flex-col items-center gap-0.5 rounded-lg p-1.5 transition-colors',
            row.status === 'active'
              ? 'text-gray-500 hover:bg-yellow-50 hover:text-yellow-600 dark:hover:bg-yellow-900/20 dark:hover:text-yellow-400'
              : 'text-gray-500 hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400',
          ]"
        >
          <Icon v-if="row.status === 'active'" name="ban" size="sm" />
          <Icon v-else name="checkCircle" size="sm" />
          <span class="text-xs">{{ row.status === "active" ? t("keys.disable") : t("keys.enable") }}</span>
        </button>
        <button
          @click="$emit('edit', row)"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
        >
          <Icon name="edit" size="sm" />
          <span class="text-xs">{{ t("common.edit") }}</span>
        </button>
        <button
          @click="$emit('delete', row)"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
        >
          <Icon name="trash" size="sm" />
          <span class="text-xs">{{ t("common.delete") }}</span>
        </button>
      </div>
    </template>

    <template #empty>
      <EmptyState
        :title="t('keys.noKeysYet')"
        :description="t('keys.createFirstKey')"
        :action-text="t('keys.createKey')"
        @action="$emit('create')"
      />
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import DataTable from "@/components/common/DataTable.vue";
import EmptyState from "@/components/common/EmptyState.vue";
import GroupBadge from "@/components/common/GroupBadge.vue";
import Icon from "@/components/icons/Icon.vue";
import type { BatchApiKeyUsageStats } from "@/api/usage";
import type { Column } from "@/components/common/types";
import type { ApiKey, Group } from "@/types";
import { formatDateTime } from "@/utils/format";
import type { ApiKeyGroup } from "@/types";
import ApiKeyRateLimitWindow from "./ApiKeyRateLimitWindow.vue";

defineProps<{
  columns: Column[];
  apiKeys: ApiKey[];
  loading: boolean;
  copiedKeyId: number | null;
  usageStats: Record<string, BatchApiKeyUsageStats>;
  userGroupRates: Record<number, number>;
  isAdminMode: boolean;
  hideCcsImportButton: boolean;
  resolveGroup: (groupId: number | null | undefined) => Group | undefined;
  getDisplayBindings: (key: ApiKey) => ApiKeyGroup[];
  maskKey: (key: string) => string;
  formatResetTime: (resetAt: string | null) => string;
}>();

defineEmits<{
  create: [];
  copy: [value: string, keyId: number];
  edit: [key: ApiKey];
  delete: [key: ApiKey];
  "use-key": [key: ApiKey];
  "import-ccswitch": [key: ApiKey];
  "toggle-status": [key: ApiKey];
  "reset-rate-limit": [key: ApiKey];
}>();

const { t } = useI18n();
</script>
