<template>
  <BaseDialog
    :show="open"
    :title="detail?.name || t('channelStatus.viewDetails')"
    width="wide"
    @close="$emit('close')"
  >
    <div class="space-y-4">
      <div v-if="loading" class="flex items-center justify-center py-8">
        <LoadingSpinner />
      </div>

      <div v-else-if="detail" class="space-y-3">
        <div class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
          <ModelPlatformIcon :platform="detail.provider" size="sm" />
          <span class="font-semibold">{{ detail.name }}</span>
        </div>

        <div class="overflow-hidden rounded-xl border border-gray-200 dark:border-dark-700">
          <table class="w-full text-sm">
            <thead class="bg-gray-50 text-gray-600 dark:bg-dark-900 dark:text-gray-300">
              <tr>
                <th class="px-4 py-2 text-left">Model</th>
                <th class="px-4 py-2 text-left">Last</th>
                <th class="px-4 py-2 text-right">Latency</th>
                <th class="px-4 py-2 text-right">7d</th>
                <th class="px-4 py-2 text-right">15d</th>
                <th class="px-4 py-2 text-right">30d</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="m in detail.models"
                :key="m.model_id"
                class="border-t border-gray-100 dark:border-dark-700"
              >
                <td class="px-4 py-2">
                  <div class="flex items-center gap-2">
                    <ModelIcon :model="m.model_id" :provider="detail.provider" :display-name="m.model_id" size="14px" />
                    <span class="font-mono">{{ m.model_id }}</span>
                  </div>
                </td>
                <td class="px-4 py-2">
                  <StatusBadge :status="statusVariant(m.last?.status)" :label="statusLabel(m.last?.status)" />
                </td>
                <td class="px-4 py-2 text-right font-mono text-gray-700 dark:text-gray-200">
                  {{ m.last?.latency_ms != null ? `${m.last.latency_ms}ms` : '-' }}
                </td>
                <td class="px-4 py-2 text-right font-mono text-gray-700 dark:text-gray-200">
                  {{ formatRate(m.availability_7d) }}
                </td>
                <td class="px-4 py-2 text-right font-mono text-gray-700 dark:text-gray-200">
                  {{ formatRate(m.availability_15d) }}
                </td>
                <td class="px-4 py-2 text-right font-mono text-gray-700 dark:text-gray-200">
                  {{ formatRate(m.availability_30d) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import type { ChannelMonitorUserDetail } from '@/api/channelMonitors'

defineProps<{
  open: boolean
  loading: boolean
  detail: ChannelMonitorUserDetail | null
}>()

defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()

function statusVariant(status?: string): string {
  if (status === 'success') return 'success'
  if (status === 'degraded') return 'warning'
  if (status === 'failure') return 'error'
  return 'inactive'
}

function statusLabel(status?: string): string {
  switch (status) {
    case 'success':
      return t('channelStatus.status.success')
    case 'degraded':
      return t('channelStatus.status.degraded')
    case 'failure':
      return t('channelStatus.status.failure')
    default:
      return t('channelStatus.status.unknown')
  }
}

function formatRate(v?: number): string {
  if (v == null || Number.isNaN(v)) return '-'
  return `${(v * 100).toFixed(1)}%`
}
</script>

