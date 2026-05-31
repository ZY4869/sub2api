<template>
<DataTable
  :columns="columns"
  :data="usageLogs"
  :loading="loading"
  :virtual-scroll="false"
  row-key="id"
>
  <template #cell-api_key="{ row }">
    <span class="text-sm text-gray-900 dark:text-white">{{
      row.api_key?.name || "-"
    }}</span>
  </template>

  <template #cell-model="{ row }">
    <UsageModelCell
      :row="row"
      :mode="usageModelDisplayMode"
    />
  </template>

  <template #cell-success_rate="{ row }">
    <UsageSuccessRateCell :row="row" />
  </template>

  <template #cell-status="{ row }">
    <div class="max-w-[280px] space-y-1">
      <div class="flex flex-wrap items-center gap-1.5">
        <span
          class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
          :class="getStatusBadgeClass(row.status)"
        >
          {{ getStatusLabel(row.status) }}
        </span>
        <span
          v-if="row.simulated_client"
          class="inline-flex items-center rounded-full bg-primary-500/10 px-2 py-0.5 text-xs font-medium text-primary-700 dark:text-primary-300"
        >
          {{ getSimulatedClientLabel(row.simulated_client) }}
        </span>
      </div>
      <div
        v-if="row.status === 'failed'"
        class="space-y-1 text-xs text-rose-600 dark:text-rose-300"
      >
        <div class="flex flex-wrap gap-x-3 gap-y-1">
          <span v-if="row.http_status != null">
            <span class="font-medium"
              >{{ t("usage.httpStatus") }}:</span
            >
            {{ row.http_status }}
          </span>
          <span v-if="row.error_code">
            <span class="font-medium">{{ t("usage.errorCode") }}:</span>
            {{ row.error_code }}
          </span>
        </div>
        <div
          v-if="row.error_message"
          :title="row.error_message"
          class="truncate"
        >
          <span class="font-medium"
            >{{ t("usage.errorMessage") }}:</span
          >
          <span class="ml-1">{{
            truncateUsageErrorMessage(row.error_message)
          }}</span>
        </div>
      </div>
    </div>
  </template>

  <template #cell-reasoning_effort="{ row }">
    <div class="space-y-1">
      <div class="text-sm text-gray-900 dark:text-white">
        {{ formatReasoningEffortPair(row.reasoning_effort_raw, row.reasoning_effort_effective, row.reasoning_effort) }}
      </div>
      <div
        v-if="formatUsageMillionContextLines(row).length > 0"
        class="space-y-0.5 text-xs text-gray-500 dark:text-gray-400"
      >
        <span
          v-for="line in formatUsageMillionContextLines(row)"
          :key="`${row.id}-${line.key}`"
          class="block break-all"
          :title="line.raw"
        >
          <span class="font-medium text-gray-400 dark:text-gray-500"
            >{{ t(line.labelKey) }}:</span
          >
          <span class="ml-1">{{ line.display }}</span>
        </span>
      </div>
    </div>
  </template>

  <template #cell-thinking_enabled="{ row }">
    <span class="text-sm text-gray-900 dark:text-white">
      {{ formatThinkingEnabled(row.thinking_enabled) }}
    </span>
  </template>

  <template #cell-request_protocol="{ row }">
    <UsageProtocolCell
      :inbound-path="row.inbound_endpoint"
      :upstream-path="row.upstream_endpoint"
    />
  </template>

  <template #cell-endpoint="{ row }">
    <div
      class="block max-w-[320px] space-y-1 text-sm text-gray-600 dark:text-gray-300"
    >
      <div
        v-for="line in formatUsageEndpoints(row)"
        :key="`${row.id}-${line.key}`"
        class="whitespace-normal break-all"
      >
        <span class="font-medium text-gray-500 dark:text-gray-400"
          >{{ t(line.labelKey) }}:</span
        >
        <span class="ml-1" :title="line.raw">{{ line.display }}</span>
      </div>
      <div
        v-if="formatUsageEndpoints(row).length === 0"
        class="text-sm text-gray-400 dark:text-gray-500"
      >
        -
      </div>
    </div>
  </template>

  <template #cell-stream="{ row }">
    <span
      class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium"
      :class="getRequestTypeBadgeClass(row)"
    >
      {{ getRequestTypeLabel(row) }}
    </span>
  </template>

  <template #cell-tokens="{ row }">
    <!-- 图片生成请求 -->
    <div v-if="row.image_count > 0" class="flex items-center gap-1.5">
      <svg
        class="h-4 w-4 text-indigo-500"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
        />
      </svg>
      <span class="font-medium text-gray-900 dark:text-white"
        >{{ row.image_count }}{{ $t("usage.imageUnit") }}</span
      >
      <span class="text-gray-400">({{ row.image_size || "2K" }})</span>
    </div>
    <!-- Token 请求 -->
    <div v-else class="flex items-center gap-1.5">
      <div class="space-y-1.5 text-sm">
        <!-- Input / Output Tokens -->
        <div class="flex items-center gap-2">
          <!-- Input -->
          <div class="inline-flex items-center gap-1">
            <Icon name="arrowDown" size="sm" class="text-emerald-500" />
            <span
              class="font-medium text-gray-900 dark:text-white"
              :title="row.input_tokens.toLocaleString()"
              >{{ formatTokens(row.input_tokens) }}</span
            >
          </div>
          <!-- Output -->
          <div class="inline-flex items-center gap-1">
            <Icon name="arrowUp" size="sm" class="text-violet-500" />
            <span
              class="font-medium text-gray-900 dark:text-white"
              :title="row.output_tokens.toLocaleString()"
              >{{ formatTokens(row.output_tokens) }}</span
            >
          </div>
        </div>
        <!-- Cache Tokens (Read + Write) -->
        <div
          v-if="
            row.cache_read_tokens > 0 || row.cache_creation_tokens > 0
          "
          class="flex items-center gap-2"
        >
          <!-- Cache Read -->
          <div
            v-if="row.cache_read_tokens > 0"
            class="inline-flex items-center gap-1"
          >
            <Icon name="inbox" size="sm" class="text-sky-500" />
            <span
              class="font-medium text-sky-600 dark:text-sky-400"
              :title="row.cache_read_tokens.toLocaleString()"
              >{{ formatCacheTokens(row.cache_read_tokens) }}</span
            >
          </div>
          <!-- Cache Write -->
          <div
            v-if="row.cache_creation_tokens > 0"
            class="inline-flex items-center gap-1"
          >
            <Icon name="edit" size="sm" class="text-amber-500" />
            <span
              class="font-medium text-amber-600 dark:text-amber-400"
              :title="row.cache_creation_tokens.toLocaleString()"
              >{{ formatCacheTokens(row.cache_creation_tokens) }}</span
            >
            <span
              v-if="row.cache_creation_1h_tokens > 0"
              class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-orange-100 text-orange-600 ring-1 ring-inset ring-orange-200 dark:bg-orange-500/20 dark:text-orange-400 dark:ring-orange-500/30"
              >1h</span
            >
            <span
              v-if="row.cache_ttl_overridden"
              :title="t('usage.cacheTtlOverriddenHint')"
              class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-rose-100 text-rose-600 ring-1 ring-inset ring-rose-200 dark:bg-rose-500/20 dark:text-rose-400 dark:ring-rose-500/30 cursor-help"
              >R</span
            >
          </div>
        </div>
      </div>
      <!-- Token Detail Tooltip -->
      <div
        class="group relative"
        @mouseenter="$emit('show-token-tooltip', $event, row)"
        @mouseleave="$emit('hide-token-tooltip')"
      >
        <div
          class="flex h-4 w-4 cursor-help items-center justify-center rounded-full bg-gray-100 transition-colors group-hover:bg-blue-100 dark:bg-gray-700 dark:group-hover:bg-blue-900/50"
        >
          <Icon
            name="infoCircle"
            size="xs"
            class="text-gray-400 group-hover:text-blue-500 dark:text-gray-500 dark:group-hover:text-blue-400"
          />
        </div>
      </div>
    </div>
  </template>

  <template #cell-cost="{ row }">
    <div class="flex items-center gap-1.5 text-sm">
      <span class="font-medium text-green-600 dark:text-green-400">
        {{ formatCurrencyBreakdown(row.actual_cost_by_currency, row.actual_cost) }}
      </span>
      <span
        v-if="getChargeLabel(row)"
        class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium"
        :class="getChargeBadgeClass(row)"
      >
        {{ getChargeLabel(row) }}
      </span>
      <span
        v-if="row.billing_exempt_reason === 'admin_free'"
        class="inline-flex items-center gap-1 rounded-full bg-emerald-100 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300"
      >
        <Icon name="crown" size="xs" class="h-3 w-3" />
        免扣
      </span>
      <!-- Cost Detail Tooltip -->
      <div
        class="group relative"
        @mouseenter="$emit('show-cost-tooltip', $event, row)"
        @mouseleave="$emit('hide-cost-tooltip')"
      >
        <div
          class="flex h-4 w-4 cursor-help items-center justify-center rounded-full bg-gray-100 transition-colors group-hover:bg-blue-100 dark:bg-gray-700 dark:group-hover:bg-blue-900/50"
        >
          <Icon
            name="infoCircle"
            size="xs"
            class="text-gray-400 group-hover:text-blue-500 dark:text-gray-500 dark:group-hover:text-blue-400"
          />
        </div>
      </div>
    </div>
  </template>

  <template #cell-first_token="{ row }">
    <span
      v-if="row.first_token_ms != null"
      class="text-sm text-gray-600 dark:text-gray-400"
    >
      {{ formatDuration(row.first_token_ms) }}
    </span>
    <span v-else class="text-sm text-gray-400 dark:text-gray-500"
      >-</span
    >
  </template>

  <template #cell-duration="{ row }">
    <span class="text-sm text-gray-600 dark:text-gray-400">{{
      formatDuration(row.duration_ms)
    }}</span>
  </template>

  <template #cell-created_at="{ value }">
    <span class="text-sm text-gray-600 dark:text-gray-400">{{
      formatDateTime(value)
    }}</span>
  </template>

  <template #cell-user_agent="{ row }">
    <span
      v-if="row.user_agent"
      class="text-sm text-gray-600 dark:text-gray-400 block max-w-[320px] whitespace-normal break-all"
      :title="row.user_agent"
      >{{ formatUserAgent(row.user_agent) }}</span
    >
    <span v-else class="text-sm text-gray-400 dark:text-gray-500"
      >-</span
    >
  </template>

  <template #cell-actions="{ row }">
    <button
      class="btn btn-secondary btn-sm"
      type="button"
      :disabled="!row.id"
      @click="$emit('open-request-preview', row)"
    >
      {{ t("usage.requestPreview.action") }}
    </button>
  </template>

  <template #empty>
    <EmptyState :message="t('usage.noRecords')" />
  </template>
</DataTable>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import DataTable from "@/components/common/DataTable.vue";
import EmptyState from "@/components/common/EmptyState.vue";
import UsageModelCell from "@/components/common/UsageModelCell.vue";
import UsageSuccessRateCell from "@/components/common/UsageSuccessRateCell.vue";
import UsageProtocolCell from "@/components/common/UsageProtocolCell.vue";
import Icon from "@/components/icons/Icon.vue";
import type {
  UsageLog,
  UsageModelDisplayMode,
} from "@/types";
import type { Column } from "@/components/common/types";
import type {
  UsageEndpointDisplayLine,
  UsageMillionContextDisplayLine,
} from "@/utils/usageDisplay";
import { formatDateTime } from "@/utils/format";

defineProps<{
  columns: Column[];
  usageLogs: UsageLog[];
  loading: boolean;
  usageModelDisplayMode: UsageModelDisplayMode;
  formatCurrencyBreakdown: (
    values: Record<string, number> | null | undefined,
    fallbackUSD: number | null | undefined,
    decimals?: number,
  ) => string;
  formatTokens: (value: number) => string;
  formatCacheTokens: (value: number) => string;
  formatDuration: (ms: number | null | undefined) => string;
  formatUserAgent: (ua: string) => string;
  getStatusBadgeClass: (status: UsageLog["status"]) => string;
  getStatusLabel: (status: UsageLog["status"]) => string;
  getSimulatedClientLabel: (client: UsageLog["simulated_client"]) => string;
  truncateUsageErrorMessage: (message: string) => string;
  formatReasoningEffortPair: (raw?: string | null, effective?: string | null, legacy?: string | null) => string;
  formatUsageMillionContextLines: (row: UsageLog) => UsageMillionContextDisplayLine[];
  formatThinkingEnabled: (value: boolean | null | undefined) => string;
  formatUsageEndpoints: (row: UsageLog) => UsageEndpointDisplayLine[];
  getRequestTypeBadgeClass: (log: UsageLog) => string;
  getRequestTypeLabel: (log: UsageLog) => string;
  getChargeLabel: (row: UsageLog) => string | null;
  getChargeBadgeClass: (row: UsageLog) => string;
}>();

defineEmits<{
  "show-token-tooltip": [event: MouseEvent, row: UsageLog];
  "hide-token-tooltip": [];
  "show-cost-tooltip": [event: MouseEvent, row: UsageLog];
  "hide-cost-tooltip": [];
  "open-request-preview": [row: UsageLog];
}>();

const { t } = useI18n();
</script>
