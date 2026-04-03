<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Pagination from '@/components/common/Pagination.vue'
import type { OpsRequestTraceListItem } from '@/api/admin/ops'
import { formatDateTime, formatNumber } from '@/utils/format'
import { formatDurationMs, getProtocolPairLabel, getStatusBadgeClass } from '../helpers'

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
              {{ t('admin.requestDetails.table.columns.time') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.requestId') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.protocolPair') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.route') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.subject') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.models') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.status') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.latency') }}
            </th>
            <th class="px-4 py-3 text-left text-[11px] font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.columns.tokens') }}
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
            <td v-for="j in 11" :key="j" class="px-4 py-4">
              <div class="h-4 animate-pulse rounded bg-gray-100 dark:bg-dark-700"></div>
            </td>
          </tr>

          <tr v-else-if="items.length === 0">
            <td colspan="11" class="px-4 py-14 text-center text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.requestDetails.table.empty') }}
            </td>
          </tr>

          <tr
            v-for="item in items"
            :key="item.id"
            class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-900/60"
            :class="{ 'bg-blue-50/60 dark:bg-blue-900/10': selectedId === item.id }"
            @click="emit('select', item)"
          >
            <td class="whitespace-nowrap px-4 py-4 text-sm text-gray-600 dark:text-gray-300">
              {{ formatDateTime(item.created_at) }}
            </td>
            <td class="max-w-[220px] px-4 py-4 align-top">
              <div class="truncate font-mono text-xs text-gray-700 dark:text-gray-200" :title="item.request_id">
                {{ item.request_id || '-' }}
              </div>
              <div class="mt-1 truncate font-mono text-[11px] text-gray-400 dark:text-gray-500" :title="item.upstream_request_id">
                {{ item.upstream_request_id || '-' }}
              </div>
            </td>
            <td class="whitespace-nowrap px-4 py-4 text-sm text-gray-600 dark:text-gray-300">
              {{ getProtocolPairLabel(item.protocol_in, item.protocol_out) }}
            </td>
            <td class="max-w-[200px] px-4 py-4 align-top">
              <div class="truncate text-sm font-medium text-gray-800 dark:text-gray-100" :title="item.route_path">
                {{ item.route_path || '-' }}
              </div>
              <div class="mt-1 text-xs uppercase tracking-wide text-gray-400 dark:text-gray-500">
                {{ item.channel || item.platform || '-' }}
              </div>
            </td>
            <td class="px-4 py-4 text-xs text-gray-600 dark:text-gray-300">
              <div>U {{ item.user_id ?? '-' }}</div>
              <div>K {{ item.api_key_id ?? '-' }}</div>
              <div>A {{ item.account_id ?? '-' }}</div>
              <div>G {{ item.group_id ?? '-' }}</div>
            </td>
            <td class="max-w-[240px] px-4 py-4 align-top">
              <div class="truncate text-sm font-medium text-gray-800 dark:text-gray-100" :title="item.requested_model">
                {{ item.requested_model || '-' }}
              </div>
              <div class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400" :title="item.actual_upstream_model || item.upstream_model">
                {{ item.actual_upstream_model || item.upstream_model || '-' }}
              </div>
            </td>
            <td class="px-4 py-4 text-sm">
              <span class="badge" :class="getStatusBadgeClass(item.status)">
                {{ item.status || '-' }}
              </span>
              <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                {{ item.status_code }} / {{ item.finish_reason || '-' }}
              </div>
            </td>
            <td class="px-4 py-4 text-sm text-gray-600 dark:text-gray-300">
              <div>{{ formatDurationMs(item.duration_ms) }}</div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                TTFT {{ formatDurationMs(item.ttft_ms) }}
              </div>
            </td>
            <td class="px-4 py-4 text-sm text-gray-600 dark:text-gray-300">
              <div>{{ formatNumber(item.total_tokens || 0) }}</div>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ formatNumber(item.input_tokens || 0) }} / {{ formatNumber(item.output_tokens || 0) }}
              </div>
            </td>
            <td class="px-4 py-4 text-xs text-gray-600 dark:text-gray-300">
              <div class="flex flex-wrap gap-1">
                <span class="badge badge-gray">{{ item.stream ? 'stream' : 'sync' }}</span>
                <span class="badge" :class="item.has_tools ? 'badge-primary' : 'badge-gray'">tool</span>
                <span class="badge" :class="item.has_thinking ? 'badge-warning' : 'badge-gray'">thinking</span>
                <span class="badge" :class="item.raw_available ? 'badge-success' : 'badge-gray'">raw</span>
              </div>
              <div class="mt-2 truncate text-[11px] text-gray-400 dark:text-gray-500" :title="item.capture_reason">
                {{ item.capture_reason || '-' }}
              </div>
            </td>
            <td class="px-4 py-4 text-right">
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
