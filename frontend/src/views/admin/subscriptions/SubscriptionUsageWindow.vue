<template>
  <div class="usage-row">
    <div class="flex items-center gap-2">
      <span class="usage-label">{{ label }}</span>
      <div class="h-1.5 flex-1 rounded-full bg-gray-200 dark:bg-dark-600">
        <div
          class="h-1.5 rounded-full transition-all"
          :class="getSubscriptionProgressClass(used, limit)"
          :style="{ width: getSubscriptionProgressWidth(used, limit) }"
        />
      </div>
      <span class="usage-amount">
        ${{ used?.toFixed(2) || '0.00' }}
        <span class="text-gray-400">/</span>
        ${{ limit?.toFixed(2) }}
      </span>
    </div>
    <div class="reset-info" v-if="windowStart">
      <svg
        class="h-3 w-3"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
        />
      </svg>
      <span>{{ formatSubscriptionResetTime(windowStart, period, t) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  formatSubscriptionResetTime,
  getSubscriptionProgressClass,
  getSubscriptionProgressWidth
} from './utils'

defineProps<{
  label: string
  used: number | null | undefined
  limit: number | null
  windowStart?: string | null
  period: 'daily' | 'weekly' | 'monthly'
}>()

const { t } = useI18n()
</script>

<style scoped>
.usage-row {
  @apply space-y-1;
}

.usage-label {
  @apply w-10 flex-shrink-0 text-xs font-medium text-gray-500 dark:text-gray-400;
}

.usage-amount {
  @apply whitespace-nowrap text-xs tabular-nums text-gray-600 dark:text-gray-300;
}

.reset-info {
  @apply flex items-center gap-1 pl-12 text-[10px] text-blue-600 dark:text-blue-400;
}
</style>
