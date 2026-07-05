<template>
  <tr
    class="group cursor-pointer transition-colors hover:bg-gray-50/80 dark:hover:bg-dark-800/50"
    @click="emit('openErrorDetail', log.id)"
  >
    <td v-if="isColumnVisible('created_at')" class="whitespace-nowrap px-4 py-2">
      <el-tooltip :content="log.request_id || log.client_request_id" placement="top" :show-after="500">
        <span class="font-mono text-xs font-medium text-gray-900 dark:text-gray-200">
          {{ formatDateTime(log.created_at).split(' ')[1] }}
        </span>
      </el-tooltip>
    </td>
    <td v-if="isColumnVisible('type')" class="whitespace-nowrap px-4 py-2">
      <span :class="['inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-bold ring-1 ring-inset', getTypeBadge(log, tt).className]">
        {{ getTypeBadge(log, tt).label }}
      </span>
    </td>
    <td v-if="isColumnVisible('category')" class="whitespace-nowrap px-4 py-2">
      <span class="text-xs font-medium text-gray-700 dark:text-gray-300">
        {{ t('usage.errors.categories.' + mapErrorCategory(log.phase, log.type)) }}
      </span>
    </td>
    <td v-if="isColumnVisible('platform')" class="whitespace-nowrap px-4 py-2">
      <span class="inline-flex items-center rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-bold uppercase text-gray-600 dark:bg-dark-700 dark:text-gray-300">
        {{ log.platform || '-' }}
      </span>
    </td>
    <td v-if="isColumnVisible('endpoint')" class="px-4 py-2">
      <div class="max-w-[180px]" :title="getEndpointTooltip(log)">
        <div class="truncate font-mono text-[11px] text-gray-700 dark:text-gray-300">{{ getInboundEndpoint(log) || '-' }}</div>
        <div v-if="shouldShowUpstreamEndpoint(log)" class="truncate text-[10px] text-gray-400 dark:text-gray-500">-> {{ log.upstream_endpoint }}</div>
        <div v-if="log.gemini_surface || log.probe_action" class="truncate text-[10px] text-gray-400 dark:text-gray-500">
          {{ [log.gemini_surface, log.probe_action].filter(Boolean).join(' / ') }}
        </div>
      </div>
    </td>
    <td v-if="isColumnVisible('model')" class="px-4 py-2">
      <div class="max-w-[140px]" :title="getModelTooltip(log)">
        <span v-if="getRequestedModel(log)" class="block truncate font-mono text-[11px] text-gray-700 dark:text-gray-300">{{ getRequestedModel(log) }}</span>
        <span v-if="shouldShowModelMapping(log)" class="block truncate text-[10px] text-gray-400 dark:text-gray-500">-> {{ log.upstream_model }}</span>
        <span v-if="log.billing_rule_id" class="block truncate text-[10px] text-gray-400 dark:text-gray-500">{{ t('admin.ops.errorLog.billingRule') }}: {{ log.billing_rule_id }}</span>
        <span v-if="!getRequestedModel(log)" class="text-xs text-gray-400">-</span>
      </div>
    </td>
    <td v-if="isColumnVisible('group')" class="px-4 py-2">
      <el-tooltip v-if="log.group_id" :content="t('admin.ops.errorLog.id') + ' ' + log.group_id" placement="top" :show-after="500">
        <span class="max-w-[100px] truncate text-xs font-medium text-gray-900 dark:text-gray-200">{{ log.group_name || '-' }}</span>
      </el-tooltip>
      <span v-else class="text-xs text-gray-400">-</span>
    </td>
    <td v-if="isColumnVisible('user')" class="px-4 py-2">
      <el-tooltip v-if="userTooltip" :content="userTooltip" placement="top" :show-after="500">
        <span class="max-w-[100px] truncate text-xs font-medium text-gray-900 dark:text-gray-200">{{ userLabel }}</span>
      </el-tooltip>
      <span v-else class="text-xs text-gray-400">-</span>
    </td>
    <td v-if="isColumnVisible('api_key')" class="px-4 py-2">
      <div v-if="log.api_key_id || log.api_key_name" class="max-w-[140px]">
        <span class="block truncate text-xs font-medium text-gray-900 dark:text-gray-200" :title="log.api_key_name || String(log.api_key_id || '')">{{ log.api_key_name || ('#' + log.api_key_id) }}</span>
        <span v-if="log.api_key_deleted" class="mt-1 inline-flex rounded bg-rose-100 px-1 py-px text-[10px] font-medium text-rose-700 dark:bg-rose-500/15 dark:text-rose-300">
          {{ t('admin.ops.errorLog.keyDeletedBadge') }}
        </span>
      </div>
      <span v-else class="text-xs text-gray-400">-</span>
    </td>
    <td v-if="isColumnVisible('status')" class="whitespace-nowrap px-4 py-2">
      <div class="flex flex-wrap items-center gap-1.5">
        <span :class="['inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-bold ring-1 ring-inset', getStatusClass(log.status_code)]">{{ log.status_code }}</span>
        <span v-if="log.request_type != null" :class="['rounded px-1.5 py-0.5 text-[10px] font-bold', getRequestTypeClass(log.request_type)]">{{ formatRequestType(log.request_type, tt) }}</span>
        <span v-if="log.severity" :class="['rounded px-1.5 py-0.5 text-[10px] font-bold', getSeverityClass(log.severity)]">{{ log.severity }}</span>
      </div>
    </td>
    <td v-if="isColumnVisible('message')" class="px-4 py-2">
      <div class="max-w-[200px]">
        <p class="truncate text-[11px] font-medium text-gray-600 dark:text-gray-400" :title="log.message">{{ formatSmartMessage(log.message, tt) || '-' }}</p>
      </div>
    </td>
    <td v-if="isColumnVisible('client_ip')" class="px-4 py-2"><span class="font-mono text-[11px] text-gray-600 dark:text-gray-400">{{ log.client_ip || '-' }}</span></td>
    <td v-if="isColumnVisible('user_agent')" class="px-4 py-2"><span class="block max-w-[220px] truncate text-[11px] text-gray-600 dark:text-gray-400" :title="log.user_agent || ''">{{ log.user_agent || '-' }}</span></td>
    <td class="whitespace-nowrap px-4 py-2 text-right" @click.stop>
      <button type="button" class="text-xs font-bold text-primary-600 hover:text-primary-700 dark:text-primary-400" @click="emit('openErrorDetail', log.id)">
        {{ t('admin.ops.errorLog.details') }}
      </button>
    </td>
  </tr>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OpsErrorLog } from '@/api/admin/ops'
import { getSeverityClass, formatDateTime } from '../utils/opsFormatters'
import { mapErrorCategory } from '@/utils/errorCategory'
import { formatRequestType, formatSmartMessage, getEndpointTooltip, getInboundEndpoint, getModelTooltip, getRequestedModel, getRequestTypeClass, getStatusClass, getTypeBadge, isUpstreamRow, shouldShowModelMapping, shouldShowUpstreamEndpoint } from './opsErrorLogTableHelpers'

const props = defineProps<{
  log: OpsErrorLog
  hiddenColumns: Set<string>
}>()

const emit = defineEmits<{
  (e: 'openErrorDetail', id: number): void
}>()

const { t } = useI18n()
const tt = (key: string) => t(key)

const userTooltip = computed(() => {
  if (isUpstreamRow(props.log) && props.log.account_id) return t('admin.ops.errorLog.accountId') + ' ' + props.log.account_id
  const userID = props.log.user_id || props.log.deleted_key_owner_user_id
  return userID ? t('admin.ops.errorLog.userId') + ' ' + userID : ''
})

const userLabel = computed(() => {
  if (isUpstreamRow(props.log)) return props.log.account_name || '-'
  return props.log.user_email || props.log.deleted_key_owner_email || '-'
})

function isColumnVisible(key: string): boolean {
  return !props.hiddenColumns.has(key)
}
</script>
