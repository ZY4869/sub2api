<template>
  <span
    v-if="label"
    class="inline-flex items-center rounded-full bg-sky-100 px-2.5 py-1 text-xs font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300"
  >
    {{ label }}
  </span>
  <span
    v-else
    class="text-sm text-gray-400 dark:text-gray-500"
  >
    -
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AdminUsageLog, UsageLog } from '@/types'
import { formatContextWindowLabel } from '@/utils/usageModelPresentation'

const props = defineProps<{
  row: Pick<UsageLog | AdminUsageLog, 'request_context_length_tokens'>
}>()

const label = computed(() => {
  const tokens = Number(props.row?.request_context_length_tokens || 0)
  if (!Number.isFinite(tokens) || tokens <= 0) {
    return ''
  }
  return formatContextWindowLabel(tokens)
})
</script>
