<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Pagination from '@/components/common/Pagination.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import type { OpsRequestTraceListItem } from '@/api/admin/ops'
import { formatDateTime, formatNumber } from '@/utils/format'
import {
  formatDurationMs,
  getProtocolPairLabel,
  getRequestTraceCaptureReasonLabel,
  getRequestTraceFlagBadges,
  getRequestTraceFinishReasonLabel,
  getRequestTraceRequestTypeLabel,
  getRequestTraceStatusLabel,
  getRequestTraceSubjectFields,
  getRequestTraceRouteFields,
  getStatusBadgeClass,
  resolveRequestTraceModelPresentation
} from '../helpers'

defineProps<{
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

const getRequestedModel = (item: OpsRequestTraceListItem) => resolveRequestTraceModelPresentation(item.requested_model)
const getUpstreamModel = (item: OpsRequestTraceListItem) =>
  resolveRequestTraceModelPresentation(item.actual_upstream_model || item.upstream_model)
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
      <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
        <thead class="bg-gray-50 dark:bg-dark-900">
          <tr>
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
            <th class="px-4 py-3 text-right text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.actions') }}
            </th>
          </tr>
        </thead>

        <tbody class="divide-y divide-gray-200 dark:divide-dark-700">
          <tr v-if="loading" v-for="i in 8" :key="i">
            <td v-for="j in 7" :key="j" class="px-4 py-4">
              <div class="h-4 animate-pulse rounded bg-gray-100 dark:bg-dark-700"></div>
            </td>
          </tr>

          <tr v-else-if="items.length === 0">
            <td colspan="7" class="px-4 py-14 text-center text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.empty') }}
            </td>
          </tr>

          <tr
            v-for="item in items"
            :key="item.id"
            class="cursor-pointer align-top hover:bg-gray-50 dark:hover:bg-dark-900/60"
            :class="{ 'bg-blue-50/60 dark:bg-blue-900/10': selectedId === item.id }"
            @click="emit('select', item)"
          >
            <td class="min-w-[280px] px-4 py-4 align-top">
              <div class="text-sm font-medium text-gray-900 dark:text-white">
                {{ formatDateTime(item.created_at) }}
              </div>
              <div class="mt-2 space-y-2">
                <div>
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ t('admin.requestDetails.presentation.labels.requestId') }}
                  </div>
                  <div class="truncate font-mono text-xs text-gray-700 dark:text-gray-200" :title="item.request_id">
                    {{ item.request_id || '-' }}
                  </div>
                </div>
                <div>
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ t('admin.requestDetails.presentation.labels.clientRequestId') }}
                  </div>
                  <div class="truncate font-mono text-xs text-gray-500 dark:text-gray-400" :title="item.client_request_id">
                    {{ item.client_request_id || '-' }}
                  </div>
                </div>
                <div>
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ t('admin.requestDetails.presentation.labels.upstreamRequestId') }}
                  </div>
                  <div class="truncate font-mono text-xs text-gray-500 dark:text-gray-400" :title="item.upstream_request_id">
                    {{ item.upstream_request_id || '-' }}
                  </div>
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.requestDetails.presentation.labels.requestType') }}:
                  {{ getRequestTraceRequestTypeLabel(t, item.request_type) }}
                </div>
              </div>
            </td>

            <td class="min-w-[220px] px-4 py-4 align-top">
              <div class="space-y-2">
                <div
                  v-for="field in getRequestTraceSubjectFields(t, item)"
                  :key="field.label"
                  class="rounded-xl bg-gray-50 px-3 py-2 dark:bg-dark-900/60"
                >
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ field.label }}
                  </div>
                  <div class="mt-1 text-sm text-gray-700 dark:text-gray-200" :class="{ 'font-mono text-xs': field.mono }">
                    {{ field.value }}
                  </div>
                </div>
              </div>
            </td>

            <td class="min-w-[220px] px-4 py-4 align-top">
              <div class="space-y-2">
                <div
                  v-for="field in getRequestTraceRouteFields(t, item)"
                  :key="field.label"
                  class="rounded-xl bg-gray-50 px-3 py-2 dark:bg-dark-900/60"
                >
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ field.label }}
                  </div>
                  <div class="mt-1 text-sm text-gray-700 dark:text-gray-200">
                    {{ field.value }}
                  </div>
                </div>
              </div>
            </td>

            <td class="min-w-[260px] px-4 py-4 align-top">
              <div class="space-y-3">
                <div v-if="getRequestedModel(item)" class="rounded-2xl border border-gray-100 p-3 dark:border-dark-700">
                  <div class="mb-2 text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ t('admin.requestDetails.presentation.labels.requestedModel') }}
                  </div>
                  <div class="flex items-center gap-3">
                    <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gray-100 dark:bg-dark-800">
                      <ModelIcon
                        :model="getRequestedModel(item)?.modelId || item.requested_model"
                        :provider="getRequestedModel(item)?.provider"
                        :display-name="getRequestedModel(item)?.displayName"
                        size="18px"
                      />
                    </span>
                    <div class="min-w-0">
                      <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
                        {{ getRequestedModel(item)?.displayName || item.requested_model || '-' }}
                      </div>
                      <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                        {{ getRequestedModel(item)?.modelId || item.requested_model || '-' }}
                      </div>
                    </div>
                  </div>
                </div>

                <div v-if="getUpstreamModel(item)" class="rounded-2xl border border-gray-100 p-3 dark:border-dark-700">
                  <div class="mb-2 text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ t('admin.requestDetails.presentation.labels.upstreamModel') }}
                  </div>
                  <div class="flex items-center gap-3">
                    <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gray-100 dark:bg-dark-800">
                      <ModelIcon
                        :model="getUpstreamModel(item)?.modelId || (item.actual_upstream_model || item.upstream_model)"
                        :provider="getUpstreamModel(item)?.provider"
                        :display-name="getUpstreamModel(item)?.displayName"
                        size="18px"
                      />
                    </span>
                    <div class="min-w-0">
                      <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
                        {{ getUpstreamModel(item)?.displayName || item.actual_upstream_model || item.upstream_model || '-' }}
                      </div>
                      <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                        {{ getUpstreamModel(item)?.modelId || item.actual_upstream_model || item.upstream_model || '-' }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </td>

            <td class="min-w-[250px] px-4 py-4 align-top">
              <div class="flex items-center gap-2">
                <span class="badge" :class="getStatusBadgeClass(item.status)">
                  {{ getRequestTraceStatusLabel(t, item.status) }}
                </span>
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  {{ item.status_code }}
                  <template v-if="item.upstream_status_code">
                    / {{ item.upstream_status_code }}
                  </template>
                </span>
              </div>
              <div class="mt-3 space-y-2 text-sm text-gray-700 dark:text-gray-300">
                <div>
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ t('admin.requestDetails.presentation.labels.finishReason') }}
                  </div>
                  <div>{{ getRequestTraceFinishReasonLabel(t, item.finish_reason) }}</div>
                </div>
                <div>
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ t('admin.requestDetails.presentation.labels.captureReason') }}
                  </div>
                  <div>{{ getRequestTraceCaptureReasonLabel(t, item.capture_reason) }}</div>
                </div>
                <div class="grid grid-cols-2 gap-2">
                  <div class="rounded-xl bg-gray-50 px-3 py-2 dark:bg-dark-900/60">
                    <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                      {{ t('admin.requestDetails.presentation.labels.duration') }}
                    </div>
                    <div class="mt-1">{{ formatDurationMs(item.duration_ms) }}</div>
                  </div>
                  <div class="rounded-xl bg-gray-50 px-3 py-2 dark:bg-dark-900/60">
                    <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                      {{ t('admin.requestDetails.presentation.labels.ttft') }}
                    </div>
                    <div class="mt-1">{{ formatDurationMs(item.ttft_ms) }}</div>
                  </div>
                </div>
                <div class="rounded-xl bg-gray-50 px-3 py-2 dark:bg-dark-900/60">
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ t('admin.requestDetails.presentation.labels.totalTokens') }}
                  </div>
                  <div class="mt-1 font-medium text-gray-900 dark:text-white">
                    {{ formatNumber(item.total_tokens || 0) }}
                  </div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ formatNumber(item.input_tokens || 0) }} / {{ formatNumber(item.output_tokens || 0) }}
                  </div>
                </div>
              </div>
            </td>

            <td class="min-w-[250px] px-4 py-4 align-top">
              <div class="flex flex-wrap gap-1.5">
                <span
                  v-for="badge in getRequestTraceFlagBadges(t, item)"
                  :key="badge.key"
                  class="badge"
                  :class="badge.className"
                >
                  {{ badge.label }}
                </span>
              </div>
              <div class="mt-3 space-y-2 text-sm text-gray-700 dark:text-gray-300">
                <div>
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ t('admin.requestDetails.presentation.labels.protocolPair') }}
                  </div>
                  <div>{{ getProtocolPairLabel(t, item.protocol_in, item.protocol_out) }}</div>
                </div>
                <div>
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ t('admin.requestDetails.presentation.labels.thinkingLevel') }}
                  </div>
                  <div>{{ item.has_thinking ? item.thinking_level || '-' : t('admin.requestDetails.presentation.flags.thinkingDisabled') }}</div>
                </div>
                <div v-if="item.tool_kinds?.length">
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ t('admin.requestDetails.presentation.labels.toolKinds') }}
                  </div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">
                    {{ item.tool_kinds.join(', ') }}
                  </div>
                </div>
              </div>
            </td>

            <td class="px-4 py-4 text-right align-top">
              <button class="btn btn-secondary btn-sm" type="button" @click.stop="emit('select', item)">
                {{ t('admin.requestDetails.table.view') }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <Pagination
      :total="total"
      :page="page"
      :page-size="pageSize"
      :page-size-options="[20, 50, 100, 200]"
      :show-jump="true"
      @update:page="emit('update:page', $event)"
      @update:page-size="emit('update:pageSize', $event)"
    />
  </div>
</template>
