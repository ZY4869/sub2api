<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OpsRequestTraceSummaryBreakdownItem } from '@/api/admin/ops'
import { formatPercent } from '../helpers'
import { formatNumber } from '@/utils/format'

const props = defineProps<{
  title: string
  description: string
  items: OpsRequestTraceSummaryBreakdownItem[]
  total: number
  loading: boolean
}>()

const { t } = useI18n()
const palette = ['bg-blue-600', 'bg-teal-500', 'bg-amber-500', 'bg-red-500', 'bg-violet-500', 'bg-lime-500']

const rows = computed(() => props.items.slice(0, 6))
const maxCount = computed(() => Math.max(...rows.value.map((item) => item.count), 1))
</script>

<template>
  <div class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700">
    <div class="mb-4">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ title }}</h3>
      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ description }}</p>
    </div>

    <div v-if="loading" class="flex h-[220px] items-center justify-center text-sm text-gray-400">
      {{ t('common.loading') }}
    </div>
    <div v-else-if="rows.length === 0" class="flex h-[220px] items-center justify-center text-sm text-gray-400">
      {{ t('common.noData') }}
    </div>
    <div v-else class="space-y-4">
      <div
        v-for="(item, index) in rows"
        :key="`${item.key}-${index}`"
        class="space-y-2"
      >
        <div class="flex items-center justify-between gap-4 text-sm">
          <div class="truncate text-gray-700 dark:text-gray-200">{{ item.label || item.key }}</div>
          <div class="flex-shrink-0 text-right text-xs text-gray-500 dark:text-gray-400">
            {{ formatNumber(item.count) }} / {{ formatPercent(item.count, total) }}
          </div>
        </div>
        <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-900">
          <div
            class="h-full rounded-full"
            :class="palette[index % palette.length]"
            :style="{ width: `${(item.count / maxCount) * 100}%` }"
          ></div>
        </div>
      </div>
    </div>
  </div>
</template>
