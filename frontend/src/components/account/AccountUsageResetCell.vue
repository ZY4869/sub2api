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
    <div
      v-for="row in presentation.resetRows"
      :key="row.key"
      class="flex items-center gap-1 text-[10px] tabular-nums"
    >
      <span
        class="w-[32px] shrink-0 rounded px-1 py-0 text-center font-medium text-gray-500 dark:text-gray-400"
      >
        {{ row.label }}
      </span>

      <span
        v-if="formatResetValue(row.resetsAt, row.remainingSeconds)"
        class="flex min-w-0 items-center gap-1.5 text-gray-700 dark:text-gray-300"
        :title="formatResetValue(row.resetsAt, row.remainingSeconds)?.tooltip || undefined"
      >
        <Icon
          name="clock"
          size="xs"
          class="shrink-0 text-gray-400 dark:text-gray-500"
        />
        <span class="shrink-0 font-medium">
          {{ formatResetValue(row.resetsAt, row.remainingSeconds)?.countdown }}
        </span>
        <span class="shrink-0 text-gray-400 dark:text-gray-500">·</span>
        <span class="min-w-0 truncate text-gray-500 dark:text-gray-400">
          {{ formatResetValue(row.resetsAt, row.remainingSeconds)?.absolute }}
        </span>
      </span>

      <span v-else class="text-gray-400 dark:text-gray-500">-</span>
    </div>
  </div>

  <div v-else class="text-xs text-gray-400">-</div>
</template>

<script setup lang="ts">
import { useAccountUsagePresentation } from '@/composables/useAccountUsagePresentation'
import { useUiNow } from '@/composables/useUiNow'
import {
  formatLocalAbsoluteTime,
  formatLocalTimestamp,
  formatResetCountdown,
  parseEffectiveResetAt,
} from '@/utils/usageResetTime'
import type { Account } from '@/types'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  account: Account
}>()

const { t } = useI18n()
const { nowDate } = useUiNow()
const { presentation } = useAccountUsagePresentation(() => props.account)

function formatResetValue(
  resetsAt: string | null,
  remainingSeconds?: number | null,
): {
  countdown: string
  absolute: string
  tooltip: string
} | null {
  const effectiveResetAt = parseEffectiveResetAt(
    resetsAt,
    remainingSeconds ?? null,
    nowDate.value,
  )
  if (!effectiveResetAt) return null

  return {
    countdown: formatResetCountdown(
      effectiveResetAt,
      nowDate.value,
      t('admin.accounts.usageWindow.now'),
    ),
    absolute: formatLocalAbsoluteTime(effectiveResetAt, nowDate.value, {
      today: t('dates.today'),
      tomorrow: t('dates.tomorrow'),
    }),
    tooltip: formatLocalTimestamp(effectiveResetAt),
  }
}
</script>
