<template>
  <div class="card p-5">
    <div class="mb-4 flex items-center justify-between gap-3">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('imageBatches.itemsTitle') }}
      </h2>
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        :disabled="!jobId || loading"
        @click="$emit('refresh')"
      >
        {{ t('common.refresh') }}
      </button>
    </div>

    <div v-if="!jobId" class="flex h-40 items-center justify-center text-sm text-gray-500">
      {{ t('imageBatches.selectJob') }}
    </div>
    <div v-else-if="loading" class="flex h-40 items-center justify-center text-sm text-gray-500">
      {{ t('common.loading') }}
    </div>
    <div v-else-if="items.length === 0" class="flex h-40 items-center justify-center text-sm text-gray-500">
      {{ t('imageBatches.noItems') }}
    </div>
    <div v-else class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-100 text-left text-xs text-gray-500 dark:border-dark-700">
            <th class="py-2 pr-3">{{ t('imageBatches.customID') }}</th>
            <th class="py-2 pr-3">{{ t('imageBatches.status') }}</th>
            <th class="py-2 pr-3 text-right">{{ t('imageBatches.outputs') }}</th>
            <th class="py-2 text-right">{{ t('imageBatches.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in items" :key="item.custom_id" class="border-b border-gray-50 dark:border-dark-800">
            <td class="py-3 pr-3 font-mono text-xs">{{ item.custom_id }}</td>
            <td class="py-3 pr-3">
              <span :class="statusClass(item.status)" class="rounded-full px-2 py-1 text-xs font-medium">
                {{ t(`imageBatches.itemStatuses.${item.status}`) }}
              </span>
              <div v-if="item.friendly_error" class="mt-1 max-w-[220px] truncate text-xs text-red-500">
                {{ item.friendly_error }}
              </div>
            </td>
            <td class="py-3 pr-3 text-right">{{ item.output_count }}</td>
            <td class="py-3 text-right">
              <button
                type="button"
                class="btn btn-secondary btn-xs"
                :disabled="item.output_count <= 0"
                @click="$emit('downloadItem', item)"
              >
                {{ t('imageBatches.download') }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ImageBatchItem, ImageBatchItemStatus } from '@/api/imageBatches'

defineProps<{
  jobId?: string
  items: ImageBatchItem[]
  loading: boolean
  t: (key: string, params?: Record<string, unknown>) => string
}>()

defineEmits<{
  refresh: []
  downloadItem: [item: ImageBatchItem]
}>()

const statusClass = (status: ImageBatchItemStatus) => {
  if (status === 'success') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (status === 'failed') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  if (status === 'cancelled') return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
  return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
}
</script>
