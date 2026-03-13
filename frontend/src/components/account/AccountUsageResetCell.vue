<template>
  <div v-if="presentation.state === 'loading'" class="space-y-1.5">
    <div v-for="index in presentation.meta.loadingRows" :key="index" class="flex items-center gap-2">
      <div class="h-3 w-[32px] animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
      <div class="h-3 w-20 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
    </div>
  </div>

  <div v-else-if="presentation.state === 'error'" class="text-xs text-red-500">
    {{ presentation.error }}
  </div>

  <div v-else-if="presentation.resetRows.length > 0" class="space-y-1">
    <div v-for="row in presentation.resetRows" :key="row.key" class="flex items-center gap-2 text-[10px] tabular-nums">
      <span class="w-[32px] shrink-0 rounded px-1 text-center font-medium text-gray-500 dark:text-gray-400">
        {{ row.label }}
      </span>
      <span class="text-gray-700 dark:text-gray-300">
        {{ formatResetValue(row.resetsAt, row.remainingSeconds) }}
      </span>
    </div>
  </div>

  <div v-else class="text-xs text-gray-400">-</div>
</template>

<script setup lang="ts">
import { useAccountUsagePresentation } from '@/composables/useAccountUsagePresentation'
import { formatLocalAbsoluteTime, parseEffectiveResetAt } from '@/utils/usageResetTime'
import type { Account } from '@/types'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  account: Account
}>()

const { t } = useI18n()
const { presentation } = useAccountUsagePresentation(() => props.account)

function formatResetValue(resetsAt: string | null, remainingSeconds?: number | null): string {
  const effectiveResetAt = parseEffectiveResetAt(resetsAt, remainingSeconds ?? null)
  if (!effectiveResetAt) return '-'

  return formatLocalAbsoluteTime(effectiveResetAt, new Date(), {
    today: t('dates.today'),
    tomorrow: t('dates.tomorrow'),
  })
}
</script>
