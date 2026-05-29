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
        class="group rounded-[20px] border border-slate-200/80 bg-white p-5 text-left shadow-[0_4px_20px_-4px_rgba(0,0,0,0.03)] transition-all duration-300 hover:-translate-y-1 hover:border-slate-300/80 hover:shadow-[0_12px_30px_-8px_rgba(0,0,0,0.08)] dark:border-dark-700 dark:bg-dark-900 dark:hover:border-dark-600"
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
            <span class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-[11px] font-bold" :class="statusBadgeClass(m.primary_last?.status)">
              {{ statusLabel(m.primary_last?.status) }}
            </span>
          </div>
        </div>

        <div class="mt-5 grid grid-cols-2 gap-3">
          <div class="rounded-xl border border-slate-100 bg-slate-50/80 p-3 dark:border-dark-700 dark:bg-dark-800/70">
            <div class="mb-2 flex items-center justify-between gap-2">
              <span class="text-[10px] font-black uppercase tracking-widest text-slate-400">
                {{ t('channelStatus.availability7d') }}
              </span>
              <PublicModelSuccessBars :rate="m.primary_availability_7d" :label="t('channelStatus.availability7d')" />
            </div>
            <div class="font-mono text-lg font-black" :class="rateColor(m.primary_availability_7d)">
              {{ formatRate(m.primary_availability_7d) }}
            </div>
          </div>
          <div class="rounded-xl border border-slate-100 bg-slate-50/80 p-3 dark:border-dark-700 dark:bg-dark-800/70">
            <div class="mb-2 text-[10px] font-black uppercase tracking-widest text-slate-400">
              {{ t('channelStatus.latestLatency') }}
            </div>
            <div class="font-mono text-lg font-black text-slate-800 dark:text-white">
              {{ formatLatency(m.primary_last?.latency_ms) }}
            </div>
          </div>
        </div>

        <div class="mt-5 flex items-end justify-between gap-4 border-t border-slate-100 pt-4 dark:border-dark-700">
          <span class="text-[10px] font-black uppercase tracking-widest text-slate-400">
            {{ t('channelStatus.timeline') }}
          </span>
          <div class="flex items-end gap-1">
            <span
              v-for="(dot, idx) in (m.timeline || []).slice(0, 16)"
              :key="idx"
              class="h-5 w-2 rounded-[2px] transition-colors"
              :class="timelineDotClass(dot.status)"
              :title="statusLabel(dot.status)"
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
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import PublicModelSuccessBars from '@/components/models/public-catalog/PublicModelSuccessBars.vue'
import {
  formatLatency,
  formatRate,
  rateColor,
} from '@/components/models/public-catalog/publicModelCatalogView'
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

function statusBadgeClass(status?: string): string {
  if (status === 'success') return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200'
  if (status === 'degraded') return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200'
  if (status === 'failure') return 'border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-200'
  return 'border-slate-200 bg-slate-50 text-slate-600 dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300'
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

</script>
