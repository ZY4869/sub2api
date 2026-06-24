<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <UsageStatsCards
          :stats="usageStats"
          :stats-card-style="pagePreferences.stats_card_style"
        />
      </template>

      <template #filters>
        <UsageFilters
          v-model:filters="filters"
          v-model:start-date="startDate"
          v-model:end-date="endDate"
          :api-key-options="apiKeyOptions"
          :platform-options="platformOptions"
          :loading="loading"
          :exporting="exporting"
          @apply="applyFilters"
          @reset="resetFilters"
          @export="exportToCSV"
          @date-range-change="onDateRangeChange"
        >
          <template #display-settings>
            <UsageDisplaySettingsMenu
              :preferences="pagePreferences"
              :hidden-columns="hiddenColumns"
              :columns="allColumns"
              :always-visible-columns="ALWAYS_VISIBLE"
              :usage-model-display-mode="usageModelDisplayMode"
              :updating-usage-model-display-mode="updatingUsageModelDisplayMode"
              :disabled="updatingUsageViewPreferences"
              @update-preference="handleUsageViewPreferenceChange"
              @toggle-column="toggleUsageColumn"
              @update-usage-model-display-mode="handleUsageModelDisplayModeChange"
            />
          </template>
        </UsageFilters>
      </template>

      <template #table>
        <ApiKeyDailyUsageCard
          :selected-api-key-i-d="selectedApiKeyID"
          :selected-api-key-name="selectedApiKeyName"
          :api-key-daily-range-label="apiKeyDailyRangeLabel"
          :rows="apiKeyDailyRows"
          :loading="apiKeyDailyLoading"
          :format-tokens="formatTokens"
          :format-currency-breakdown="formatCurrencyBreakdown"
        />
        <UsageTable
          :columns="visibleColumns"
          :usage-logs="usageLogs"
          :loading="loading"
          :usage-model-display-mode="usageModelDisplayMode"
          :table-density="pagePreferences.table_density"
          :format-currency-breakdown="formatCurrencyBreakdown"
          :format-tokens="formatTokens"
          :format-cache-tokens="formatCacheTokens"
          :format-duration="formatDuration"
          :format-user-agent="formatUserAgent"
          :get-status-badge-class="getStatusBadgeClass"
          :get-status-label="getStatusLabel"
          :get-simulated-client-label="getSimulatedClientLabel"
          :truncate-usage-error-message="truncateUsageErrorMessage"
          :format-reasoning-effort-pair="formatReasoningEffortPair"
          :format-usage-million-context-lines="formatUsageMillionContextLines"
          :format-thinking-enabled="formatThinkingEnabled"
          :format-usage-endpoints="formatUsageEndpoints"
          :get-request-type-badge-class="getRequestTypeBadgeClass"
          :get-request-type-label="getRequestTypeLabel"
          :get-charge-label="getChargeLabel"
          :get-charge-badge-class="getChargeBadgeClass"
          @show-token-tooltip="showTokenTooltip"
          @hide-token-tooltip="hideTokenTooltip"
          @show-cost-tooltip="showTooltip"
          @hide-cost-tooltip="hideTooltip"
          @open-request-preview="openRequestPreview"
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
  </AppLayout>

  <UsageRequestPreviewModal
    :show="requestPreviewOpen"
    :usage-log="selectedPreviewUsage"
    @close="closeRequestPreview"
  />

  <UsageTokenTooltip
    :visible="tokenTooltipVisible"
    :position="tokenTooltipPosition"
    :data="tokenTooltipData"
    :get-cache-read-label="getCacheReadLabel"
    :get-cache-creation-label="getCacheCreationLabel"
  />

  <UsageCostTooltip
    :visible="tooltipVisible"
    :position="tooltipPosition"
    :data="tooltipData"
  />
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useAppStore } from "@/stores/app";
import { usageAPI } from "@/api";
import AppLayout from "@/components/layout/AppLayout.vue";
import TablePageLayout from "@/components/layout/TablePageLayout.vue";
import Pagination from "@/components/common/Pagination.vue";
import UsageStatsCards from "@/components/user/usage/UsageStatsCards.vue";
import UsageRequestPreviewModal from "@/components/user/usage/UsageRequestPreviewModal.vue";
import UsageDisplaySettingsMenu from "@/components/usage/UsageDisplaySettingsMenu.vue";
import type {
  UsageLog,
  UsageModelDisplayMode,
  UsageQueryParams,
  UsageStatsResponse,
  TrendDataPoint,
} from "@/types";
import type { UsageFilterApiKey } from "@/api/usage";
import type { Column } from "@/components/common/types";
import { getPersistedPageSize } from "@/composables/usePersistedPageSize";
import { useTokenDisplayMode } from "@/composables/useTokenDisplayMode";
import { useUsageModelDisplayModePreference } from "@/composables/useUsageModelDisplayModePreference";
import { useUsageViewPreferences } from "@/composables/useUsageViewPreferences";
import {
  formatReasoningEffortPair,
  formatThinkingEnabled,
} from "@/utils/format";
import {
  formatUsageEndpointDisplay,
  formatUsageMillionContextDisplay,
  formatUsageMillionContextExportFields,
  formatUsageUserAgentDisplay,
} from "@/utils/usageDisplay";
import { formatUsageProtocolExportText } from "@/utils/protocolDisplay";
import {
  getUsageChargeBadgeClass,
  getUsageChargeLabel,
  getUsageOperationBadgeClass,
  getUsageOperationLabel,
} from "@/utils/usageOperation";
import {
  formatUsageAmount,
  formatUsageMultiplier,
} from "@/utils/usageCost";
import { FILTER_PLATFORM_ORDER, getPlatformEnglishName } from "@/utils/platformBranding";
import ApiKeyDailyUsageCard from "./usage/ApiKeyDailyUsageCard.vue";
import UsageCostTooltip from "./usage/UsageCostTooltip.vue";
import UsageFilters from "./usage/UsageFilters.vue";
import UsageTable from "./usage/UsageTable.vue";
import UsageTokenTooltip from "./usage/UsageTokenTooltip.vue";

const { t } = useI18n();
const appStore = useAppStore();
const { formatTokenDisplay } = useTokenDisplayMode();
const {
  usageModelDisplayMode,
  updatingUsageModelDisplayMode,
  setUsageModelDisplayMode,
} = useUsageModelDisplayModePreference();
const {
  pagePreferences,
  hiddenColumns,
  updatingUsageViewPreferences,
  patchPagePreferences,
  toggleColumn: toggleUsageColumn,
} = useUsageViewPreferences("user");
const formatCurrencyBreakdown = (
  values: Record<string, number> | null | undefined,
  fallbackUSD: number | null | undefined,
  decimals = 4,
) => {
  const entries = Object.entries(values || {})
    .filter(([, value]) => Number.isFinite(value))
    .sort(([left], [right]) => left.localeCompare(right));
  if (entries.length === 0) {
    return `$${formatUsageAmount(fallbackUSD, decimals)}`;
  }
  return entries
    .map(([currency, value]) => {
      const normalized = currency.toUpperCase();
      return `${normalized === "CNY" ? "¥" : "$"}${formatUsageAmount(value, decimals)}`;
    })
    .join(" / ");
};

let abortController: AbortController | null = null;

// Tooltip state
const tooltipVisible = ref(false);
const tooltipPosition = ref({ x: 0, y: 0 });
const tooltipData = ref<UsageLog | null>(null);

// Token tooltip state
const tokenTooltipVisible = ref(false);
const tokenTooltipPosition = ref({ x: 0, y: 0 });
const tokenTooltipData = ref<UsageLog | null>(null);

// Usage stats from API
const usageStats = ref<UsageStatsResponse | null>(null);

const ALWAYS_VISIBLE = ["created_at"];

const allColumns = computed<Column[]>(() => [
  { key: "api_key", label: t("usage.apiKeyFilter"), sortable: false },
  { key: "model", label: t("usage.model"), sortable: true },
  { key: "success_rate", label: t("usage.modelSuccessRate"), sortable: false },
  { key: "status", label: t("usage.status"), sortable: false },
  { key: "thinking_enabled", label: t("usage.thinkingMode"), sortable: false },
  {
    key: "reasoning_effort",
    label: t("usage.reasoningEffort"),
    sortable: false,
  },
  {
    key: "request_protocol",
    label: t("usage.requestProtocol"),
    sortable: false,
  },
  { key: "endpoint", label: t("usage.endpoint"), sortable: false },
  { key: "stream", label: t("usage.type"), sortable: false },
  { key: "tokens", label: t("usage.tokens"), sortable: false },
  { key: "cache_hit", label: t("usage.cacheHit"), sortable: false },
  { key: "cost", label: t("usage.cost"), sortable: false },
  { key: "first_token", label: t("usage.firstToken"), sortable: false },
  { key: "duration", label: t("usage.duration"), sortable: false },
  { key: "created_at", label: t("usage.time"), sortable: true },
  { key: "user_agent", label: t("usage.userAgent"), sortable: false },
  { key: "actions", label: t("common.actions"), sortable: false },
]);

const visibleColumns = computed<Column[]>(() =>
  allColumns.value.filter(
    (column) => ALWAYS_VISIBLE.includes(column.key) || !hiddenColumns.value.has(column.key),
  ),
);

const usageLogs = ref<UsageLog[]>([]);
const apiKeys = ref<UsageFilterApiKey[]>([]);
const apiKeyDailyRows = ref<TrendDataPoint[]>([]);
const apiKeyDailyLoading = ref(false);
const loading = ref(false);
const exporting = ref(false);
const requestPreviewOpen = ref(false);
const selectedPreviewUsage = ref<UsageLog | null>(null);

const apiKeyOptions = computed(() => {
  return [
    { value: null, label: t("usage.allApiKeys") },
    ...apiKeys.value.map((key) => ({
      value: key.id,
      label: key.deleted
        ? `${key.name} (${t("usage.deletedApiKeySuffix")})`
        : key.name,
    })),
  ];
});

const platformOptions = computed(() => [
  { value: null, label: t("usage.allPlatforms") },
  ...FILTER_PLATFORM_ORDER.map((platform) => ({
    value: platform,
    label: getPlatformEnglishName(platform),
  })),
]);

const selectedApiKeyID = computed(() => {
  const value = filters.value.api_key_id;
  const numeric = value == null ? 0 : Number(value);
  return Number.isFinite(numeric) && numeric > 0 ? numeric : null;
});

const selectedApiKeyName = computed(() => {
  const id = selectedApiKeyID.value;
  if (!id) return t("usage.allApiKeys");
  const selected = apiKeys.value.find((key) => key.id === id);
  if (!selected) return `#${id}`;
  return selected.deleted
    ? `${selected.name} (${t("usage.deletedApiKeySuffix")})`
    : selected.name;
});

const apiKeyDailyRangeLabel = computed(() =>
  `${filters.value.start_date || startDate.value} - ${filters.value.end_date || endDate.value}`,
);

// Helper function to format date in local timezone
const formatLocalDate = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
};

// Initialize date range immediately
const now = new Date();
const weekAgo = new Date(now);
weekAgo.setDate(weekAgo.getDate() - 6);

// Date range state
const startDate = ref(formatLocalDate(weekAgo));
const endDate = ref(formatLocalDate(now));

const filters = ref<UsageQueryParams>({
  api_key_id: undefined,
  platform: undefined,
  start_date: undefined,
  end_date: undefined,
});

// Initialize filters with date range
filters.value.start_date = startDate.value;
filters.value.end_date = endDate.value;

// Handle date range change from DateRangePicker
const onDateRangeChange = (range: {
  startDate: string;
  endDate: string;
  preset: string | null;
}) => {
  filters.value.start_date = range.startDate;
  filters.value.end_date = range.endDate;
  applyFilters();
};

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0,
});

const formatDuration = (ms: number | null | undefined): string => {
  if (typeof ms !== "number" || !Number.isFinite(ms)) return "-";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
};

const formatUserAgent = (ua: string): string => {
  return formatUsageUserAgentDisplay(ua);
};

const isDeepSeekUsageRow = (
  row: Pick<UsageLog, "upstream_service"> | null | undefined,
): boolean => String(row?.upstream_service || "").trim().toLowerCase() === "deepseek";

const getCacheReadLabel = (
  row: Pick<UsageLog, "upstream_service"> | null | undefined,
): string => (isDeepSeekUsageRow(row) ? "Cache Hit" : t("admin.usage.cacheReadTokens"));

const getCacheCreationLabel = (
  row: Pick<UsageLog, "upstream_service"> | null | undefined,
): string => (isDeepSeekUsageRow(row) ? "Cache Miss" : t("admin.usage.cacheCreationTokens"));

const getChargeLabel = (row: UsageLog): string | null =>
  getUsageChargeLabel(row, t);

const getChargeBadgeClass = (row: UsageLog): string =>
  getUsageChargeBadgeClass(row);

const getRequestTypeLabel = (log: UsageLog): string => {
  return getUsageOperationLabel(log, t);
};

const getRequestTypeBadgeClass = (log: UsageLog): string => {
  return getUsageOperationBadgeClass(log);
};

const getStatusLabel = (status: UsageLog["status"]): string =>
  status === "failed" ? t("usage.statusFailed") : t("usage.statusSucceeded");

const getStatusBadgeClass = (status: UsageLog["status"]): string =>
  status === "failed"
    ? "bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300"
    : "bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300";

const getSimulatedClientLabel = (
  client: UsageLog["simulated_client"],
): string => {
  if (client === "gemini_cli") {
    return t("usage.simulatedClientGeminiCli");
  }
  return t("usage.simulatedClientCodex");
};

const truncateUsageErrorMessage = (message: string): string => {
  const trimmed = message.trim();
  if (trimmed.length <= 120) {
    return trimmed;
  }
  return `${trimmed.slice(0, 117)}...`;
};

const getRequestTypeExportText = (log: UsageLog): string => {
  switch (log.operation_type) {
    case "batch_create":
      return t("usage.operationTypeBatchCreate");
    case "batch_settlement":
      return t("usage.operationTypeBatchSettlement");
    case "batch_status":
      return t("usage.operationTypeBatchStatus");
    case "get_file_metadata":
      return t("usage.operationTypeGetFileMetadata");
    case "official_result_download":
      return t("usage.operationTypeOfficialResultDownload");
    case "local_archive_download":
      return t("usage.operationTypeLocalArchiveDownload");
    default: {
      const label = getUsageOperationLabel(log, t);
      if (label === t("usage.ws")) return t("usage.ws");
      if (label === t("usage.stream")) return t("usage.stream");
      if (label === t("usage.sync")) return t("usage.sync");
      return t("usage.unknown");
    }
  }
};

const formatUsageEndpoints = (
  log: Pick<UsageLog, "inbound_endpoint" | "upstream_endpoint">,
) => formatUsageEndpointDisplay(log);

const formatUsageMillionContextLines = (
  row: Pick<
    UsageLog,
    | "million_context_requested"
    | "million_context_effective"
    | "million_context_source"
    | "million_context_beta_token"
  >,
) => formatUsageMillionContextDisplay(row);

const formatTokens = (value: number): string => formatTokenDisplay(value);

// Compact format for cache tokens in table cells
const formatCacheTokens = (value: number): string => {
  return formatTokenDisplay(value);
};

const getCacheCreationTotal = (
  log: Pick<
    UsageLog,
    "cache_creation_tokens" | "cache_creation_5m_tokens" | "cache_creation_1h_tokens"
  >,
): number =>
  (log.cache_creation_tokens || 0) +
  (log.cache_creation_5m_tokens || 0) +
  (log.cache_creation_1h_tokens || 0);

const loadUsageLogs = async () => {
  if (abortController) {
    abortController.abort();
  }
  const currentAbortController = new AbortController();
  abortController = currentAbortController;
  const { signal } = currentAbortController;
  loading.value = true;
  try {
    const params: UsageQueryParams = {
      page: pagination.page,
      page_size: pagination.page_size,
      ...filters.value,
    };

    const response = await usageAPI.query(params, { signal });
    if (signal.aborted) {
      return;
    }
    usageLogs.value = response.items;
    pagination.total = response.total;
    pagination.pages = response.pages;
  } catch (error) {
    if (signal.aborted) {
      return;
    }
    const abortError = error as { name?: string; code?: string };
    if (
      abortError?.name === "AbortError" ||
      abortError?.code === "ERR_CANCELED"
    ) {
      return;
    }
    appStore.showError(t("usage.failedToLoad"));
  } finally {
    if (abortController === currentAbortController) {
      loading.value = false;
    }
  }
};

const loadApiKeys = async () => {
  try {
    const currentSelectedID = filters.value.api_key_id
      ? Number(filters.value.api_key_id)
      : undefined;
    const previousSelected = currentSelectedID
      ? apiKeys.value.find((key) => key.id === currentSelectedID)
      : undefined;
    const items = await usageAPI.listFilterApiKeys({
      start_date: filters.value.start_date || startDate.value,
      end_date: filters.value.end_date || endDate.value,
    });
    if (
      currentSelectedID &&
      previousSelected &&
      !items.some((key) => key.id === currentSelectedID)
    ) {
      apiKeys.value = [previousSelected, ...items];
      return;
    }
    apiKeys.value = items;
  } catch (error) {
    console.error("Failed to load API keys:", error);
  }
};

const loadUsageStats = async () => {
  try {
    const apiKeyId = filters.value.api_key_id
      ? Number(filters.value.api_key_id)
      : undefined;
    const stats = await usageAPI.getStatsByDateRange(
      filters.value.start_date || startDate.value,
      filters.value.end_date || endDate.value,
      apiKeyId,
      filters.value.platform,
    );
    usageStats.value = stats;
  } catch (error) {
    console.error("Failed to load usage stats:", error);
  }
};

const loadApiKeyDailyUsage = async () => {
  const apiKeyId = selectedApiKeyID.value;
  if (!apiKeyId) {
    apiKeyDailyRows.value = [];
    return;
  }
  apiKeyDailyLoading.value = true;
  try {
    const response = await usageAPI.getDashboardApiKeyDailyUsage(apiKeyId, {
      start_date: filters.value.start_date || startDate.value,
      end_date: filters.value.end_date || endDate.value,
    });
    apiKeyDailyRows.value = response.daily_details || [];
  } catch (error) {
    console.error("Failed to load API key daily usage:", error);
    apiKeyDailyRows.value = [];
    appStore.showError(t("usage.apiKeyDailyFailed"));
  } finally {
    apiKeyDailyLoading.value = false;
  }
};

const applyFilters = () => {
  pagination.page = 1;
  loadApiKeys();
  loadUsageLogs();
  loadUsageStats();
  loadApiKeyDailyUsage();
};

const resetFilters = () => {
  filters.value = {
    api_key_id: undefined,
    platform: undefined,
    start_date: undefined,
    end_date: undefined,
  };
  // Reset date range to default (last 7 days)
  const now = new Date();
  const weekAgo = new Date(now);
  weekAgo.setDate(weekAgo.getDate() - 6);
  startDate.value = formatLocalDate(weekAgo);
  endDate.value = formatLocalDate(now);
  filters.value.start_date = startDate.value;
  filters.value.end_date = endDate.value;
  pagination.page = 1;
  loadApiKeys();
  loadUsageLogs();
  loadUsageStats();
  loadApiKeyDailyUsage();
};

const handlePageChange = (page: number) => {
  pagination.page = page;
  loadUsageLogs();
};

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize;
  pagination.page = 1;
  loadUsageLogs();
};

const handleUsageModelDisplayModeChange = async (
  mode: UsageModelDisplayMode,
) => {
  await setUsageModelDisplayMode(mode);
};

const handleUsageViewPreferenceChange = async (
  key: "hidden_columns" | "token_display_mode" | "table_density" | "stats_card_style",
  value: string,
) => {
  await patchPagePreferences({ [key]: value } as any);
};

const formatModelSuccessRateExport = (
  rate: number | null | undefined,
): string => {
  if (rate == null || !Number.isFinite(rate)) {
    return "";
  }
  return `${(rate * 100).toFixed(1)}%`;
};

const openRequestPreview = (usageLog: UsageLog) => {
  if (!usageLog.id) {
    return;
  }
  selectedPreviewUsage.value = usageLog;
  requestPreviewOpen.value = true;
};

const closeRequestPreview = () => {
  requestPreviewOpen.value = false;
  selectedPreviewUsage.value = null;
};

/**
 * Escape CSV value to prevent injection and handle special characters
 */
const escapeCSVValue = (value: unknown): string => {
  if (value == null) return "";

  const str = String(value);
  const escaped = str.replace(/"/g, '""');

  // Prevent formula injection by prefixing dangerous characters with single quote
  if (/^[=+\-@\t\r]/.test(str)) {
    return `"\'${escaped}"`;
  }

  // Escape values containing comma, quote, or newline
  if (/[,"\n\r]/.test(str)) {
    return `"${escaped}"`;
  }

  return str;
};

const exportToCSV = async () => {
  if (pagination.total === 0) {
    appStore.showWarning(t("usage.noDataToExport"));
    return;
  }

  exporting.value = true;
  appStore.showInfo(t("usage.preparingExport"));

  try {
    const allLogs: UsageLog[] = [];
    const pageSize = 100; // Use a larger page size for export to reduce requests
    const totalRequests = Math.ceil(pagination.total / pageSize);

    for (let page = 1; page <= totalRequests; page++) {
      const params: UsageQueryParams = {
        page: page,
        page_size: pageSize,
        ...filters.value,
      };
      const response = await usageAPI.query(params);
      allLogs.push(...response.items);
    }

    if (allLogs.length === 0) {
      appStore.showWarning(t("usage.noDataToExport"));
      return;
    }

    const headers = [
      t("usage.time"),
      t("usage.apiKeyFilter"),
      t("usage.model"),
      t("usage.status"),
      t("usage.modelSuccessRate"),
      t("usage.simulatedClient"),
      t("usage.thinkingMode"),
      t("usage.reasoningEffort"),
      t("usage.millionContextRequested"),
      t("usage.millionContextEffective"),
      t("usage.millionContextSource"),
      t("usage.millionContextBetaToken"),
      t("usage.requestProtocol"),
      t("usage.inboundEndpoint"),
      t("usage.type"),
      t("usage.httpStatus"),
      t("usage.errorCode"),
      t("usage.errorMessage"),
      t("usage.inputTokens"),
      t("usage.outputTokens"),
      t("usage.cacheReadTokens"),
      t("usage.cacheCreationTokens"),
      t("usage.rate"),
      t("usage.billed"),
      t("usage.original"),
      t("usage.billingExemptReason"),
      t("usage.firstToken"),
      t("usage.duration"),
    ];

    const rows = allLogs.map((log) => {
      const millionContext = formatUsageMillionContextExportFields(log);
      return [
        log.created_at,
        log.api_key?.name || "",
        log.model,
        getStatusLabel(log.status),
        formatModelSuccessRateExport(log.model_success_rate_7d),
        log.simulated_client
          ? getSimulatedClientLabel(log.simulated_client)
          : "",
        formatThinkingEnabled(log.thinking_enabled),
        formatReasoningEffortPair(log.reasoning_effort_raw, log.reasoning_effort_effective, log.reasoning_effort),
        millionContext.requested,
        millionContext.effective,
        millionContext.source,
        millionContext.betaToken,
        formatUsageProtocolExportText(
          log.inbound_endpoint,
          log.upstream_endpoint,
        ),
        log.inbound_endpoint || "",
        getRequestTypeExportText(log),
        log.http_status ?? "",
        log.error_code || "",
        log.error_message || "",
        log.input_tokens,
        log.output_tokens,
        log.cache_read_tokens,
        getCacheCreationTotal(log),
        formatUsageMultiplier(log.rate_multiplier),
        formatUsageAmount(log.actual_cost, 8),
        formatUsageAmount(log.total_cost, 8),
        log.billing_exempt_reason || "",
        log.first_token_ms ?? "",
        log.duration_ms,
      ].map(escapeCSVValue);
    });

    const csvContent = [
      headers.map(escapeCSVValue).join(","),
      ...rows.map((row) => row.join(",")),
    ].join("\n");

    const blob = new Blob([`\uFEFF${csvContent}`], {
      type: "text/csv;charset=utf-8;",
    });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `usage_${filters.value.start_date}_to_${filters.value.end_date}.csv`;
    link.click();
    window.URL.revokeObjectURL(url);

    appStore.showSuccess(t("usage.exportSuccess"));
  } catch (error) {
    appStore.showError(t("usage.exportFailed"));
    console.error("CSV Export failed:", error);
  } finally {
    exporting.value = false;
  }
};

// Tooltip functions
const showTooltip = (event: MouseEvent, row: UsageLog) => {
  const target = event.currentTarget as HTMLElement;
  const rect = target.getBoundingClientRect();

  tooltipData.value = row;
  // Position to the right of the icon, vertically centered
  tooltipPosition.value.x = rect.right + 8;
  tooltipPosition.value.y = rect.top + rect.height / 2;
  tooltipVisible.value = true;
};

const hideTooltip = () => {
  tooltipVisible.value = false;
  tooltipData.value = null;
};

// Token tooltip functions
const showTokenTooltip = (event: MouseEvent, row: UsageLog) => {
  const target = event.currentTarget as HTMLElement;
  const rect = target.getBoundingClientRect();

  tokenTooltipData.value = row;
  tokenTooltipPosition.value.x = rect.right + 8;
  tokenTooltipPosition.value.y = rect.top + rect.height / 2;
  tokenTooltipVisible.value = true;
};

const hideTokenTooltip = () => {
  tokenTooltipVisible.value = false;
  tokenTooltipData.value = null;
};

onMounted(() => {
  loadApiKeys();
  loadUsageLogs();
  loadUsageStats();
  loadApiKeyDailyUsage();
});
</script>
