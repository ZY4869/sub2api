<template>
  <div class="card p-5">
    <div class="mb-4 flex items-center justify-between gap-3">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('imageBatches.jobsTitle') }}
      </h2>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="$emit('refresh')">
        {{ t('common.refresh') }}
      </button>
    </div>

    <div v-if="loading" class="flex h-40 items-center justify-center text-sm text-gray-500">
      {{ t('common.loading') }}
    </div>
    <div v-else-if="jobs.length === 0" class="flex h-40 items-center justify-center text-sm text-gray-500">
      {{ t('imageBatches.noJobs') }}
    </div>
    <div v-else class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-100 text-left text-xs text-gray-500 dark:border-dark-700">
            <th class="py-2 pr-3">{{ t('imageBatches.job') }}</th>
            <th class="py-2 pr-3">{{ t('imageBatches.model') }}</th>
            <th class="py-2 pr-3">{{ t('imageBatches.status') }}</th>
            <th class="py-2 pr-3 text-right">{{ t('imageBatches.counts') }}</th>
            <th class="py-2 text-right">{{ t('imageBatches.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="job in jobs"
            :key="job.id"
            class="border-b border-gray-50 dark:border-dark-800"
            :class="selectedJobId === job.id ? 'bg-primary-50/60 dark:bg-primary-950/20' : ''"
          >
            <td class="py-3 pr-3">
              <button
                type="button"
                class="font-mono text-xs text-primary-600 hover:text-primary-700"
                @click="$emit('select', job)"
              >
                {{ shortID(job.id) }}
              </button>
              <div class="text-xs text-gray-400">{{ formatDate(job.created_at) }}</div>
            </td>
            <td class="py-3 pr-3">
              <span class="inline-flex max-w-[220px] items-center gap-1 truncate">
                <ModelIcon
                  :model="job.display_model_id"
                  :provider="job.provider"
                  :display-name="job.display_model_id"
                  size="14px"
                />
                <span class="truncate">{{ job.display_model_id }}</span>
              </span>
            </td>
            <td class="py-3 pr-3">
              <span :class="statusClass(job.status)" class="rounded-full px-2 py-1 text-xs font-medium">
                {{ t(`imageBatches.statuses.${job.status}`) }}
              </span>
              <div v-if="job.friendly_error" class="mt-1 max-w-[220px] truncate text-xs text-red-500">
                {{ job.friendly_error }}
              </div>
            </td>
            <td class="py-3 pr-3 text-right font-mono text-xs">
              {{ job.counts.succeeded }}/{{ job.counts.items }}
            </td>
            <td class="py-3 text-right">
              <div class="flex flex-wrap justify-end gap-2">
                <button type="button" class="btn btn-secondary btn-xs" @click="$emit('select', job)">
                  {{ t('imageBatches.items') }}
                </button>
                <button type="button" class="btn btn-secondary btn-xs" @click="$emit('download', job)">
                  {{ t('imageBatches.download') }}
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-xs"
                  :disabled="!canCancel(job.status)"
                  @click="$emit('cancel', job)"
                >
                  {{ t('common.cancel') }}
                </button>
                <button type="button" class="btn btn-secondary btn-xs" @click="$emit('deleteOutputs', job)">
                  {{ t('imageBatches.deleteOutputs') }}
                </button>
                <button type="button" class="btn btn-danger btn-xs" @click="$emit('delete', job)">
                  {{ t('common.delete') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import ModelIcon from '@/components/common/ModelIcon.vue'
import type { ImageBatchJob, ImageBatchJobStatus } from '@/api/imageBatches'

defineProps<{
  jobs: ImageBatchJob[]
  loading: boolean
  selectedJobId?: string
  t: (key: string, params?: Record<string, unknown>) => string
}>()

defineEmits<{
  refresh: []
  select: [job: ImageBatchJob]
  download: [job: ImageBatchJob]
  cancel: [job: ImageBatchJob]
  delete: [job: ImageBatchJob]
  deleteOutputs: [job: ImageBatchJob]
}>()

const terminalStatuses = new Set<ImageBatchJobStatus>([
  'completed',
  'failed',
  'cancelled',
  'output_deleted',
])

const shortID = (id: string) => id.length > 12 ? `${id.slice(0, 8)}...${id.slice(-4)}` : id
const canCancel = (status: ImageBatchJobStatus) => !terminalStatuses.has(status)
const formatDate = (value: string) => new Date(value).toLocaleString()

const statusClass = (status: ImageBatchJobStatus) => {
  if (status === 'completed') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (status === 'failed') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  if (status === 'cancelled') return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
  if (status === 'output_deleted') return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
}
</script>
