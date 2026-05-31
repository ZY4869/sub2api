<template>
  <div class="flex items-center gap-2">
    <span
      class="inline-flex h-7 w-7 items-center justify-center rounded-full"
      :class="statusClass"
      :title="title"
      :aria-label="title"
    >
      <Icon :name="iconName" size="sm" />
    </span>
    <span class="font-mono text-xs text-gray-700 dark:text-gray-200">
      {{ label }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { UsageLog } from '@/types'

const props = defineProps<{
  row: Partial<Pick<UsageLog, 'model_success_rate_7d' | 'model_success_status'>>
}>()

const { t } = useI18n()

const normalizedStatus = computed(() => {
  const status = String(props.row.model_success_status || 'unknown').toLowerCase()
  return ['healthy', 'warning', 'error'].includes(status) ? status : 'unknown'
})

const label = computed(() => {
  const rate = props.row.model_success_rate_7d
  if (rate == null || !Number.isFinite(rate)) return '-'
  return `${(rate * 100).toFixed(1)}%`
})

const title = computed(() =>
  t(`usage.modelSuccessRateStatuses.${normalizedStatus.value}`, { rate: label.value }),
)

const iconName = computed(() => {
  switch (normalizedStatus.value) {
    case 'healthy':
      return 'checkCircle'
    case 'warning':
      return 'exclamationTriangle'
    case 'error':
      return 'xCircle'
    default:
      return 'questionCircle'
  }
})

const statusClass = computed(() => {
  switch (normalizedStatus.value) {
    case 'healthy':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
    case 'warning':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
    case 'error':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300'
    default:
      return 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-300'
  }
})
</script>
