<template>
  <div>
    <div v-if="loading" class="flex items-center justify-center py-12">
      <LoadingSpinner />
    </div>

    <div v-else-if="!featureEnabled" class="flex items-center justify-center p-10 text-center">
      <div class="max-w-md">
        <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
          <Icon name="chart" size="lg" class="text-gray-400" />
        </div>
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('channelStatus.featureDisabled') }}
        </h3>
      </div>
    </div>

    <div v-else-if="items.length === 0" class="flex items-center justify-center p-10 text-center">
      <div class="max-w-md">
        <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
          <Icon name="database" size="lg" class="text-gray-400" />
        </div>
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('empty.noData', 'No data') }}
        </h3>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {{ t('common.tryAgainLater', 'Please try again later.') }}
        </p>
      </div>
    </div>

    <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      <button
        v-for="m in items"
        :key="m.id"
        type="button"
        class="rounded-2xl border border-gray-200 bg-white p-5 text-left transition hover:border-primary-300 hover:shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:hover:border-primary-600"
        @click="$emit('openDetail', m.id)"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex items-center gap-2">
              <ModelPlatformIcon :platform="m.provider" size="sm" />
              <div class="truncate text-base font-semibold text-gray-900 dark:text-white">
                {{ m.name }}
              </div>
            </div>
            <div class="mt-2 flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
              <span>{{ t('channelStatus.primaryModel') }}:</span>
              <ModelIcon :model="m.primary_model_id" :provider="m.provider" :display-name="m.primary_model_id" size="14px" />
              <span class="font-mono truncate">{{ m.primary_model_id }}</span>
            </div>
          </div>

          <div class="flex-shrink-0">
            <StatusBadge :status="statusVariant(m.primary_last?.status)" :label="statusLabel(m.primary_last?.status)" />
          </div>
        </div>

        <div class="mt-4 flex items-center justify-between gap-4">
          <div class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('channelStatus.availability7d') }}:
            <span class="font-semibold text-gray-900 dark:text-white">
              {{ formatRate(m.primary_availability_7d) }}
            </span>
          </div>
          <div class="flex items-center gap-1">
            <span
              v-for="(dot, idx) in (m.timeline || []).slice(0, 16)"
              :key="idx"
              class="h-2 w-2 rounded-full"
              :class="timelineDotClass(dot.status)"
              :title="dot.status"
            ></span>
          </div>
        </div>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import type { ChannelMonitorUserListItem } from '@/api/channelMonitors'

defineProps<{
  loading: boolean
  featureEnabled: boolean
  items: ChannelMonitorUserListItem[]
}>()

defineEmits<{
  (e: 'openDetail', id: number): void
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

function timelineDotClass(status?: string): string {
  switch (status) {
    case 'success':
      return 'bg-green-500'
    case 'degraded':
      return 'bg-yellow-500'
    case 'failure':
      return 'bg-red-500'
    default:
      return 'bg-gray-400'
  }
}

function formatRate(v?: number): string {
  if (v == null || Number.isNaN(v)) return '-'
  return `${(v * 100).toFixed(1)}%`
}
</script>
