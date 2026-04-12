<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Pagination from '@/components/common/Pagination.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import type { OpsRequestTraceListItem } from '@/api/admin/ops'
import { formatDateTime, formatNumber } from '@/utils/format'
import TruncatedCopyText from './TruncatedCopyText.vue'
import {
  formatDurationMs,
  getProtocolPairLabel,
  getRequestTraceCaptureReasonLabel,
  getRequestTraceFlagBadges,
  getRequestTraceFinishReasonLabel,
  getRequestTraceStatusLabel,
  getStatusBadgeClass,
  resolveRequestTraceModelPresentation
} from '../helpers'

const props = defineProps<{
  items: OpsRequestTraceListItem[]
  total: number
  page: number
  pageSize: number
  loading: boolean
  selectedId?: number | null
}>()

const emit = defineEmits<{
  (e: 'select', value: OpsRequestTraceListItem): void
  (e: 'update:page', value: number): void
  (e: 'update:pageSize', value: number): void
}>()

const { t } = useI18n()

function getRequestedModel(item: OpsRequestTraceListItem) {
  return resolveRequestTraceModelPresentation(item.requested_model)
}

function getUpstreamModel(item: OpsRequestTraceListItem) {
  return resolveRequestTraceModelPresentation(item.actual_upstream_model || item.upstream_model)
}

function hasDifferentUpstreamModel(item: OpsRequestTraceListItem): boolean {
  const requested = getRequestedModel(item)?.modelId || item.requested_model || ''
  const upstream = getUpstreamModel(item)?.modelId || item.actual_upstream_model || item.upstream_model || ''
  return requested.trim() !== upstream.trim() && upstream.trim().length > 0
}

function joinSummaryParts(parts: Array<string | null | undefined>): string {
  return parts
    .map((part) => String(part || '').trim())
    .filter(Boolean)
    .join(' · ') || '-'
}

function getSubjectSummary(item: OpsRequestTraceListItem): string {
  return joinSummaryParts([
    item.user_id ? t('admin.requestDetails.table.summary.user', { id: item.user_id }) : '',
    item.api_key_id ? t('admin.requestDetails.table.summary.apiKey', { id: item.api_key_id }) : '',
    item.account_id ? t('admin.requestDetails.table.summary.account', { id: item.account_id }) : '',
    item.group_id ? t('admin.requestDetails.table.summary.group', { id: item.group_id }) : ''
  ])
}

function getRouteSummary(item: OpsRequestTraceListItem): string {
  return joinSummaryParts([
    item.route_path,
    item.channel,
    item.platform,
    item.gemini_surface,
    item.probe_action,
    item.billing_rule_id,
    getProtocolPairLabel(t, item.protocol_in, item.protocol_out)
  ])
}

function getRequestIdTooltip(item: OpsRequestTraceListItem): string {
  return [
    `${t('admin.requestDetails.presentation.labels.requestId')}: ${item.request_id || '-'}`,
    `${t('admin.requestDetails.presentation.labels.clientRequestId')}: ${item.client_request_id || '-'}`,
    `${t('admin.requestDetails.presentation.labels.upstreamRequestId')}: ${item.upstream_request_id || '-'}`,
    `${t('admin.requestDetails.presentation.labels.billingRuleId')}: ${item.billing_rule_id || '-'}`,
    `${t('admin.requestDetails.presentation.labels.geminiSurface')}: ${item.gemini_surface || '-'}`,
    `${t('admin.requestDetails.presentation.labels.probeAction')}: ${item.probe_action || '-'}`
  ].join('\n')
}

function getStatusReasonSummary(item: OpsRequestTraceListItem): string {
  const statusCodes = joinSummaryParts([
    item.status_code ? String(item.status_code) : '',
    item.upstream_status_code ? String(item.upstream_status_code) : ''
  ])
  return joinSummaryParts([
    statusCodes === '-' ? '' : statusCodes.replace(' · ', ' / '),
    getRequestTraceFinishReasonLabel(t, item.finish_reason),
    getRequestTraceCaptureReasonLabel(t, item.capture_reason)
  ])
}

function getPerformanceSummary(item: OpsRequestTraceListItem): string {
  return joinSummaryParts([
    formatDurationMs(item.duration_ms),
    t('admin.requestDetails.table.summary.ttft', { value: formatDurationMs(item.ttft_ms) }),
    t('admin.requestDetails.table.summary.tokens', { value: formatNumber(item.total_tokens || 0) })
  ])
}

function getModelDisplayText(modelId?: string | null) {
  return String(modelId || '').trim() || '-'
}

function getModelTitle(modelId?: string | null) {
  const presentation = resolveRequestTraceModelPresentation(modelId)
  if (!presentation) return '-'
  if (presentation.displayName === presentation.modelId) {
    return presentation.modelId
  }
  return `${presentation.displayName} (${presentation.modelId})`
}
</script>

<template>
  <div class="rounded-3xl bg-white shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700">
    <div class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <div>
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.requestDetails.table.title') }}
        </h3>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.requestDetails.table.description') }}
        </p>
      </div>
      <div class="text-xs text-gray-400 dark:text-gray-500">
        {{ t('common.total') }}: {{ formatNumber(total) }}
      </div>
    </div>

    <div class="overflow-x-auto">
      <table class="min-w-[1480px] divide-y divide-gray-200 whitespace-nowrap dark:divide-dark-700">
        <thead class="bg-gray-50 dark:bg-dark-900">
          <tr>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.time') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.requestId') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.subject') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.route') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.models') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.status') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.flags') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.performance') }}
            </th>
            <th class="px-4 py-3 text-right text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.actions') }}
            </th>
          </tr>
        </thead>

        <tbody class="divide-y divide-gray-200 dark:divide-dark-700">
          <tr v-if="loading" v-for="i in 8" :key="i">
            <td v-for="j in 9" :key="j" class="px-4 py-4">
              <div class="h-4 animate-pulse rounded bg-gray-100 dark:bg-dark-700"></div>
            </td>
          </tr>

          <tr v-else-if="props.items.length === 0">
            <td colspan="9" class="px-4 py-14 text-center text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.empty') }}
            </td>
          </tr>

          <tr
            v-for="item in props.items"
            :key="item.id"
            class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-900/60"
            :class="{ 'bg-blue-50/60 dark:bg-blue-900/10': props.selectedId === item.id }"
            @click="emit('select', item)"
          >
            <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
              {{ formatDateTime(item.created_at) }}
            </td>

            <td class="px-4 py-3">
              <TruncatedCopyText
                class="block max-w-[220px] text-xs text-gray-800 dark:text-gray-200"
                :value="item.request_id"
                :title-text="getRequestIdTooltip(item)"
                mono
              />
            </td>

            <td class="px-4 py-3">
              <TruncatedCopyText
                class="block max-w-[260px] text-sm text-gray-700 dark:text-gray-200"
                :display-text="getSubjectSummary(item)"
                :copy-value="getSubjectSummary(item)"
                :title-text="getSubjectSummary(item)"
              />
            </td>

            <td class="px-4 py-3">
              <TruncatedCopyText
                class="block max-w-[320px] text-sm text-gray-700 dark:text-gray-200"
                :display-text="getRouteSummary(item)"
                :copy-value="getRouteSummary(item)"
                :title-text="getRouteSummary(item)"
              />
            </td>

            <td class="px-4 py-3">
              <div class="flex min-w-0 max-w-[360px] items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
                <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-gray-100 dark:bg-dark-700">
                  <ModelIcon
                    :model="getRequestedModel(item)?.modelId || item.requested_model"
                    :provider="getRequestedModel(item)?.provider"
                    :display-name="getRequestedModel(item)?.displayName"
                    size="14px"
                  />
                </span>
                <TruncatedCopyText
                  class="min-w-0 max-w-[120px]"
                  :display-text="getRequestedModel(item)?.displayName || getModelDisplayText(item.requested_model)"
                  :copy-value="getRequestedModel(item)?.modelId || item.requested_model"
                  :title-text="getModelTitle(item.requested_model)"
                />

                <template v-if="hasDifferentUpstreamModel(item)">
                  <span class="shrink-0 text-gray-300 dark:text-gray-600">→</span>
                  <span class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-gray-100 dark:bg-dark-700">
                    <ModelIcon
                      :model="getUpstreamModel(item)?.modelId || (item.actual_upstream_model || item.upstream_model)"
                      :provider="getUpstreamModel(item)?.provider"
                      :display-name="getUpstreamModel(item)?.displayName"
                      size="14px"
                    />
                  </span>
                  <TruncatedCopyText
                    class="min-w-0 max-w-[120px]"
                    :display-text="getUpstreamModel(item)?.displayName || getModelDisplayText(item.actual_upstream_model || item.upstream_model)"
                    :copy-value="getUpstreamModel(item)?.modelId || item.actual_upstream_model || item.upstream_model"
                    :title-text="getModelTitle(item.actual_upstream_model || item.upstream_model)"
                  />
                </template>
              </div>
            </td>

            <td class="px-4 py-3">
              <div class="flex min-w-0 max-w-[320px] items-center gap-2">
                <span class="badge shrink-0" :class="getStatusBadgeClass(item.status)">
                  {{ getRequestTraceStatusLabel(t, item.status) }}
                </span>
                <TruncatedCopyText
                  class="min-w-0 max-w-[240px] text-sm text-gray-700 dark:text-gray-200"
                  :display-text="getStatusReasonSummary(item)"
                  :copy-value="getStatusReasonSummary(item)"
                  :title-text="getStatusReasonSummary(item)"
                />
              </div>
            </td>

            <td class="px-4 py-3">
              <div class="flex max-w-[260px] items-center gap-1 overflow-hidden">
                <span
                  v-for="badge in getRequestTraceFlagBadges(t, item)"
                  :key="badge.key"
                  class="badge shrink-0"
                  :class="badge.className"
                  :title="badge.label"
                >
                  {{ badge.label }}
                </span>
              </div>
            </td>

            <td class="px-4 py-3">
              <TruncatedCopyText
                class="block max-w-[220px] text-sm text-gray-700 dark:text-gray-200"
                :display-text="getPerformanceSummary(item)"
                :copy-value="getPerformanceSummary(item)"
                :title-text="getPerformanceSummary(item)"
              />
            </td>

            <td class="px-4 py-3 text-right">
              <button class="btn btn-secondary btn-sm" type="button" @click.stop="emit('select', item)">
                {{ t('admin.requestDetails.table.view') }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <Pagination
      :total="props.total"
      :page="props.page"
      :page-size="props.pageSize"
      :page-size-options="[20, 50, 100, 200]"
      :show-jump="true"
      @update:page="emit('update:page', $event)"
      @update:page-size="emit('update:pageSize', $event)"
    />
  </div>
</template>
