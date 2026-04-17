<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <!-- Total Requests -->
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-blue-100 p-2 dark:bg-blue-900/30">
                <Icon
                  name="document"
                  size="md"
                  class="text-blue-600 dark:text-blue-400"
                />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t("usage.totalRequests") }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ usageStats?.total_requests?.toLocaleString() || "0" }}
                </p>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t("usage.inSelectedRange") }}
                </p>
              </div>
            </div>
          </div>

          <!-- Total Tokens -->
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-amber-100 p-2 dark:bg-amber-900/30">
                <Icon
                  name="cube"
                  size="md"
                  class="text-amber-600 dark:text-amber-400"
                />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t("usage.totalTokens") }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ formatTokens(usageStats?.total_tokens || 0) }}
                </p>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t("usage.in") }}:
                  {{ formatTokens(usageStats?.total_input_tokens || 0) }} /
                  {{ t("usage.out") }}:
                  {{ formatTokens(usageStats?.total_output_tokens || 0) }}
                </p>
              </div>
            </div>
          </div>

          <!-- Total Cost -->
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-green-100 p-2 dark:bg-green-900/30">
                <Icon
                  name="dollar"
                  size="md"
                  class="text-green-600 dark:text-green-400"
                />
              </div>
              <div class="min-w-0 flex-1">
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t("usage.totalCost") }}
                </p>
                <p class="text-xl font-bold text-green-600 dark:text-green-400">
                  ${{ formatUsageAmount(usageStats?.total_actual_cost, 4) }}
                </p>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t("usage.actualCost") }} /
                  <span class="line-through"
                    >${{ formatUsageAmount(usageStats?.total_cost, 4) }}</span
                  >
                  {{ t("usage.standardCost") }}
                </p>
                <p
                  v-if="usageStats?.admin_free_requests"
                  class="mt-1 text-[11px] text-emerald-500 dark:text-emerald-300"
                >
                  管理员免扣
                  {{ usageStats.admin_free_requests.toLocaleString() }} 次 / ${{
                    formatUsageAmount(usageStats.admin_free_standard_cost, 4)
                  }}
                  标准成本
                </p>
              </div>
            </div>
          </div>

          <!-- Average Duration -->
          <div class="card p-4">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-purple-100 p-2 dark:bg-purple-900/30">
                <Icon
                  name="clock"
                  size="md"
                  class="text-purple-600 dark:text-purple-400"
                />
              </div>
              <div>
                <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
                  {{ t("usage.avgDuration") }}
                </p>
                <p class="text-xl font-bold text-gray-900 dark:text-white">
                  {{ formatDuration(usageStats?.average_duration_ms || 0) }}
                </p>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t("usage.perRequest") }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="card">
          <div class="px-6 py-4">
            <div class="flex flex-wrap items-end gap-4">
              <!-- API Key Filter -->
              <div class="min-w-[180px]">
                <label class="input-label">{{ t("usage.apiKeyFilter") }}</label>
                <Select
                  v-model="filters.api_key_id"
                  :options="apiKeyOptions"
                  :placeholder="t('usage.allApiKeys')"
                  @change="applyFilters"
                />
              </div>

              <!-- Date Range Filter -->
              <div>
                <label class="input-label">{{ t("usage.timeRange") }}</label>
                <DateRangePicker
                  v-model:start-date="startDate"
                  v-model:end-date="endDate"
                  @change="onDateRangeChange"
                />
              </div>

              <!-- Actions -->
              <div class="ml-auto flex items-center gap-3">
                <TokenDisplayModeToggle />
                <button
                  @click="applyFilters"
                  :disabled="loading"
                  class="btn btn-secondary"
                >
                  {{ t("common.refresh") }}
                </button>
                <button @click="resetFilters" class="btn btn-secondary">
                  {{ t("common.reset") }}
                </button>
                <button
                  @click="exportToCSV"
                  :disabled="exporting"
                  class="btn btn-primary"
                >
                  <svg
                    v-if="exporting"
                    class="-ml-1 mr-2 h-4 w-4 animate-spin"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      class="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      stroke-width="4"
                    ></circle>
                    <path
                      class="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
                  </svg>
                  {{ exporting ? t("usage.exporting") : t("usage.exportCsv") }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="usageLogs"
          :loading="loading"
          :virtual-scroll="false"
          row-key="request_id"
        >
          <template #cell-api_key="{ row }">
            <span class="text-sm text-gray-900 dark:text-white">{{
              row.api_key?.name || "-"
            }}</span>
          </template>

          <template #cell-model="{ row }">
            <div class="flex items-start gap-2">
              <ModelIcon :model="row.model" size="16px" />
              <div class="min-w-0">
                <div
                  class="break-all font-medium text-gray-900 dark:text-white"
                >
                  {{ row.model }}
                </div>
                <div
                  v-if="row.upstream_model && row.upstream_model !== row.model"
                  class="break-all text-xs text-gray-500 dark:text-gray-400"
                >
                  <span class="mr-1">-></span>{{ row.upstream_model }}
                </div>
              </div>
            </div>
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
            <span class="text-sm text-gray-900 dark:text-white">
              {{ formatReasoningEffort(row.reasoning_effort) }}
            </span>
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
                :key="`${row.request_id}-${line.key}`"
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
                @mouseenter="showTokenTooltip($event, row)"
                @mouseleave="hideTokenTooltip"
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
                ${{ formatUsageAmount(row.actual_cost) }}
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
                @mouseenter="showTooltip($event, row)"
                @mouseleave="hideTooltip"
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
              @click="openRequestPreview(row)"
            >
              {{ t("usage.requestPreview.action") }}
            </button>
          </template>

          <template #empty>
            <EmptyState :message="t('usage.noRecords')" />
          </template>
        </DataTable>
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

  <!-- Token Tooltip Portal -->
  <Teleport to="body">
    <div
      v-if="tokenTooltipVisible"
      class="fixed z-[9999] pointer-events-none -translate-y-1/2"
      :style="{
        left: tokenTooltipPosition.x + 'px',
        top: tokenTooltipPosition.y + 'px',
      }"
    >
      <div
        class="whitespace-nowrap rounded-lg border border-gray-700 bg-gray-900 px-3 py-2.5 text-xs text-white shadow-xl dark:border-gray-600 dark:bg-gray-800"
      >
        <div class="space-y-1.5">
          <!-- Token Breakdown -->
          <div>
            <div class="text-xs font-semibold text-gray-300 mb-1">
              {{ t("usage.tokenDetails") }}
            </div>
            <div
              v-if="tokenTooltipData && tokenTooltipData.input_tokens > 0"
              class="flex items-center justify-between gap-4"
            >
              <span class="text-gray-400">{{
                t("admin.usage.inputTokens")
              }}</span>
              <span class="font-medium text-white">{{
                tokenTooltipData.input_tokens.toLocaleString()
              }}</span>
            </div>
            <div
              v-if="tokenTooltipData && tokenTooltipData.output_tokens > 0"
              class="flex items-center justify-between gap-4"
            >
              <span class="text-gray-400">{{
                t("admin.usage.outputTokens")
              }}</span>
              <span class="font-medium text-white">{{
                tokenTooltipData.output_tokens.toLocaleString()
              }}</span>
            </div>
            <div
              v-if="
                tokenTooltipData && tokenTooltipData.cache_creation_tokens > 0
              "
            >
              <!-- 有 5m/1h 明细时，展开显示 -->
              <template
                v-if="
                  tokenTooltipData.cache_creation_5m_tokens > 0 ||
                  tokenTooltipData.cache_creation_1h_tokens > 0
                "
              >
                <div
                  v-if="tokenTooltipData.cache_creation_5m_tokens > 0"
                  class="flex items-center justify-between gap-4"
                >
                  <span class="text-gray-400 flex items-center gap-1.5">
                    {{ t("admin.usage.cacheCreation5mTokens") }}
                    <span
                      class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-amber-500/20 text-amber-400 ring-1 ring-inset ring-amber-500/30"
                      >5m</span
                    >
                  </span>
                  <span class="font-medium text-white">{{
                    tokenTooltipData.cache_creation_5m_tokens.toLocaleString()
                  }}</span>
                </div>
                <div
                  v-if="tokenTooltipData.cache_creation_1h_tokens > 0"
                  class="flex items-center justify-between gap-4"
                >
                  <span class="text-gray-400 flex items-center gap-1.5">
                    {{ t("admin.usage.cacheCreation1hTokens") }}
                    <span
                      class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-orange-500/20 text-orange-400 ring-1 ring-inset ring-orange-500/30"
                      >1h</span
                    >
                  </span>
                  <span class="font-medium text-white">{{
                    tokenTooltipData.cache_creation_1h_tokens.toLocaleString()
                  }}</span>
                </div>
              </template>
              <!-- 无明细时，只显示聚合值 -->
              <div v-else class="flex items-center justify-between gap-4">
                <span class="text-gray-400">{{
                  t("admin.usage.cacheCreationTokens")
                }}</span>
                <span class="font-medium text-white">{{
                  tokenTooltipData.cache_creation_tokens.toLocaleString()
                }}</span>
              </div>
            </div>
            <div
              v-if="tokenTooltipData && tokenTooltipData.cache_ttl_overridden"
              class="flex items-center justify-between gap-4"
            >
              <span class="text-gray-400 flex items-center gap-1.5">
                {{ t("usage.cacheTtlOverriddenLabel") }}
                <span
                  class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-rose-500/20 text-rose-400 ring-1 ring-inset ring-rose-500/30"
                  >R-{{
                    tokenTooltipData.cache_creation_1h_tokens > 0 ? "5m" : "1H"
                  }}</span
                >
              </span>
              <span class="font-medium text-rose-400">{{
                tokenTooltipData.cache_creation_1h_tokens > 0
                  ? t("usage.cacheTtlOverridden1h")
                  : t("usage.cacheTtlOverridden5m")
              }}</span>
            </div>
            <div
              v-if="tokenTooltipData && tokenTooltipData.cache_read_tokens > 0"
              class="flex items-center justify-between gap-4"
            >
              <span class="text-gray-400">{{
                t("admin.usage.cacheReadTokens")
              }}</span>
              <span class="font-medium text-white">{{
                tokenTooltipData.cache_read_tokens.toLocaleString()
              }}</span>
            </div>
          </div>
          <!-- Total -->
          <div
            class="flex items-center justify-between gap-6 border-t border-gray-700 pt-1.5"
          >
            <span class="text-gray-400">{{ t("usage.totalTokens") }}</span>
            <span class="font-semibold text-blue-400">{{
              (
                (tokenTooltipData?.input_tokens || 0) +
                (tokenTooltipData?.output_tokens || 0) +
                (tokenTooltipData?.cache_creation_tokens || 0) +
                (tokenTooltipData?.cache_read_tokens || 0)
              ).toLocaleString()
            }}</span>
          </div>
        </div>
        <!-- Tooltip Arrow (left side) -->
        <div
          class="absolute right-full top-1/2 h-0 w-0 -translate-y-1/2 border-b-[6px] border-r-[6px] border-t-[6px] border-b-transparent border-r-gray-900 border-t-transparent dark:border-r-gray-800"
        ></div>
      </div>
    </div>
  </Teleport>

  <!-- Tooltip Portal -->
  <Teleport to="body">
    <div
      v-if="tooltipVisible"
      class="fixed z-[9999] pointer-events-none -translate-y-1/2"
      :style="{
        left: tooltipPosition.x + 'px',
        top: tooltipPosition.y + 'px',
      }"
    >
      <div
        class="whitespace-nowrap rounded-lg border border-gray-700 bg-gray-900 px-3 py-2.5 text-xs text-white shadow-xl dark:border-gray-600 dark:bg-gray-800"
      >
        <div class="space-y-1.5">
          <!-- Cost Breakdown -->
          <div class="mb-2 border-b border-gray-700 pb-1.5">
            <div class="text-xs font-semibold text-gray-300 mb-1">
              {{ t("usage.costDetails") }}
            </div>
            <div
              v-if="
                tooltipData && hasPositiveUsageAmount(tooltipData.input_cost)
              "
              class="flex items-center justify-between gap-4"
            >
              <span class="text-gray-400">{{
                t("admin.usage.inputCost")
              }}</span>
              <span class="font-medium text-white"
                >${{ formatUsageAmount(tooltipData.input_cost) }}</span
              >
            </div>
            <div
              v-if="
                tooltipData && hasPositiveUsageAmount(tooltipData.output_cost)
              "
              class="flex items-center justify-between gap-4"
            >
              <span class="text-gray-400">{{
                t("admin.usage.outputCost")
              }}</span>
              <span class="font-medium text-white"
                >${{ formatUsageAmount(tooltipData.output_cost) }}</span
              >
            </div>
            <div
              v-if="tooltipData && tooltipData.input_tokens > 0"
              class="flex items-center justify-between gap-4"
            >
              <span class="text-gray-400">{{
                t("usage.inputTokenPrice")
              }}</span>
              <span class="font-medium text-sky-300"
                >{{
                  formatTokenPricePerMillion(
                    tooltipData.input_cost,
                    tooltipData.input_tokens,
                  )
                }}
                {{ t("usage.perMillionTokens") }}</span
              >
            </div>
            <div
              v-if="tooltipData && tooltipData.output_tokens > 0"
              class="flex items-center justify-between gap-4"
            >
              <span class="text-gray-400">{{
                t("usage.outputTokenPrice")
              }}</span>
              <span class="font-medium text-violet-300"
                >{{
                  formatTokenPricePerMillion(
                    tooltipData.output_cost,
                    tooltipData.output_tokens,
                  )
                }}
                {{ t("usage.perMillionTokens") }}</span
              >
            </div>
            <div
              v-if="
                tooltipData &&
                hasPositiveUsageAmount(tooltipData.cache_creation_cost)
              "
              class="flex items-center justify-between gap-4"
            >
              <span class="text-gray-400">{{
                t("admin.usage.cacheCreationCost")
              }}</span>
              <span class="font-medium text-white"
                >${{ formatUsageAmount(tooltipData.cache_creation_cost) }}</span
              >
            </div>
            <div
              v-if="
                tooltipData &&
                hasPositiveUsageAmount(tooltipData.cache_read_cost)
              "
              class="flex items-center justify-between gap-4"
            >
              <span class="text-gray-400">{{
                t("admin.usage.cacheReadCost")
              }}</span>
              <span class="font-medium text-white"
                >${{ formatUsageAmount(tooltipData.cache_read_cost) }}</span
              >
            </div>
          </div>
          <!-- Rate and Summary -->
          <div class="flex items-center justify-between gap-6">
            <span class="text-gray-400">{{ t("usage.serviceTier") }}</span>
            <span class="font-semibold text-cyan-300">{{
              getUsageServiceTierLabel(tooltipData?.service_tier, t)
            }}</span>
          </div>
          <div class="flex items-center justify-between gap-6">
            <span class="text-gray-400">{{ t("usage.rate") }}</span>
            <span class="font-semibold text-blue-400"
              >{{ formatUsageMultiplier(tooltipData?.rate_multiplier) }}x</span
            >
          </div>
          <div class="flex items-center justify-between gap-6">
            <span class="text-gray-400">{{ t("usage.original") }}</span>
            <span class="font-medium text-white"
              >${{ formatUsageAmount(tooltipData?.total_cost) }}</span
            >
          </div>
          <div
            class="flex items-center justify-between gap-6 border-t border-gray-700 pt-1.5"
          >
            <span class="text-gray-400">{{ t("usage.billed") }}</span>
            <span class="font-semibold text-green-400"
              >${{ formatUsageAmount(tooltipData?.actual_cost) }}</span
            >
          </div>
          <div
            v-if="tooltipData?.billing_exempt_reason === 'admin_free'"
            class="flex items-center justify-between gap-6"
          >
            <span class="text-gray-400">免扣原因</span>
            <span
              class="inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[11px] font-medium text-emerald-300"
            >
              <Icon name="crown" size="xs" class="h-3 w-3" />
              管理员免费
            </span>
          </div>
        </div>
        <!-- Tooltip Arrow (left side) -->
        <div
          class="absolute right-full top-1/2 h-0 w-0 -translate-y-1/2 border-b-[6px] border-r-[6px] border-t-[6px] border-b-transparent border-r-gray-900 border-t-transparent dark:border-r-gray-800"
        ></div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { useAppStore } from "@/stores/app";
import { usageAPI } from "@/api";
import AppLayout from "@/components/layout/AppLayout.vue";
import TablePageLayout from "@/components/layout/TablePageLayout.vue";
import DataTable from "@/components/common/DataTable.vue";
import Pagination from "@/components/common/Pagination.vue";
import EmptyState from "@/components/common/EmptyState.vue";
import Select from "@/components/common/Select.vue";
import DateRangePicker from "@/components/common/DateRangePicker.vue";
import ModelIcon from "@/components/common/ModelIcon.vue";
import TokenDisplayModeToggle from "@/components/common/TokenDisplayModeToggle.vue";
import UsageProtocolCell from "@/components/common/UsageProtocolCell.vue";
import Icon from "@/components/icons/Icon.vue";
import UsageRequestPreviewModal from "@/components/user/usage/UsageRequestPreviewModal.vue";
import type { UsageLog, UsageQueryParams, UsageStatsResponse } from "@/types";
import type { UsageFilterApiKey } from "@/api/usage";
import type { Column } from "@/components/common/types";
import { getPersistedPageSize } from "@/composables/usePersistedPageSize";
import { useTokenDisplayMode } from "@/composables/useTokenDisplayMode";
import {
  formatDateTime,
  formatReasoningEffort,
  formatThinkingEnabled,
} from "@/utils/format";
import { formatTokenPricePerMillion } from "@/utils/usagePricing";
import { getUsageServiceTierLabel } from "@/utils/usageServiceTier";
import {
  formatUsageEndpointDisplay,
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
  hasPositiveUsageAmount,
} from "@/utils/usageCost";

const { t } = useI18n();
const appStore = useAppStore();
const { formatTokenDisplay } = useTokenDisplayMode();

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

const columns = computed<Column[]>(() => [
  { key: "api_key", label: t("usage.apiKeyFilter"), sortable: false },
  { key: "model", label: t("usage.model"), sortable: true },
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
  { key: "cost", label: t("usage.cost"), sortable: false },
  { key: "first_token", label: t("usage.firstToken"), sortable: false },
  { key: "duration", label: t("usage.duration"), sortable: false },
  { key: "created_at", label: t("usage.time"), sortable: true },
  { key: "user_agent", label: t("usage.userAgent"), sortable: false },
  { key: "actions", label: t("common.actions"), sortable: false },
]);

const usageLogs = ref<UsageLog[]>([]);
const apiKeys = ref<UsageFilterApiKey[]>([]);
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

const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
};

const formatUserAgent = (ua: string): string => {
  return formatUsageUserAgentDisplay(ua);
};

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
      return "Batch Create";
    case "batch_settlement":
      return "Batch Settlement";
    case "batch_status":
      return "Batch Status";
    case "get_file_metadata":
      return "File Metadata";
    case "official_result_download":
      return "Official Download";
    case "local_archive_download":
      return "Local Archive Download";
    default: {
      const label = getUsageOperationLabel(log, t);
      if (label === t("usage.ws")) return "WS";
      if (label === t("usage.stream")) return "Stream";
      if (label === t("usage.sync")) return "Sync";
      return "Unknown";
    }
  }
};

const formatUsageEndpoints = (
  log: Pick<UsageLog, "inbound_endpoint" | "upstream_endpoint">,
) => formatUsageEndpointDisplay(log);

const formatTokens = (value: number): string => formatTokenDisplay(value);

// Compact format for cache tokens in table cells
const formatCacheTokens = (value: number): string => {
  return formatTokenDisplay(value);
};

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
    );
    usageStats.value = stats;
  } catch (error) {
    console.error("Failed to load usage stats:", error);
  }
};

const applyFilters = () => {
  pagination.page = 1;
  loadApiKeys();
  loadUsageLogs();
  loadUsageStats();
};

const resetFilters = () => {
  filters.value = {
    api_key_id: undefined,
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
      "Time",
      "API Key Name",
      "Model",
      "Status",
      "Simulated Client",
      "Thinking Mode",
      "Reasoning Effort",
      "Request Protocol",
      "Inbound Endpoint",
      "Type",
      "HTTP Status",
      "Error Code",
      "Error Message",
      "Input Tokens",
      "Output Tokens",
      "Cache Read Tokens",
      "Cache Creation Tokens",
      "Rate Multiplier",
      "Billed Cost",
      "Original Cost",
      "免扣原因",
      "First Token (ms)",
      "Duration (ms)",
    ];
    headers[20] = "Billing Exempt Reason";

    const rows = allLogs.map((log) =>
      [
        log.created_at,
        log.api_key?.name || "",
        log.model,
        getStatusLabel(log.status),
        log.simulated_client
          ? getSimulatedClientLabel(log.simulated_client)
          : "",
        formatThinkingEnabled(log.thinking_enabled),
        formatReasoningEffort(log.reasoning_effort),
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
        log.cache_creation_tokens,
        formatUsageMultiplier(log.rate_multiplier),
        formatUsageAmount(log.actual_cost, 8),
        formatUsageAmount(log.total_cost, 8),
        log.billing_exempt_reason || "",
        log.first_token_ms ?? "",
        log.duration_ms,
      ].map(escapeCSVValue),
    );

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
});
</script>
