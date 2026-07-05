<template>
  <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
    <div class="mb-2 text-sm font-medium text-gray-900 dark:text-white">
      {{ t('admin.accounts.codexImport.result') }}
    </div>
    <p class="text-sm text-gray-700 dark:text-dark-300">
      {{ t('admin.accounts.codexImport.resultSummary', resultSummaryParams) }}
    </p>
    <div v-if="visibleMessages.length" class="mt-3 max-h-44 overflow-auto text-xs">
      <div
        v-for="message in visibleMessages"
        :key="`${message.kind}-${message.index}-${message.message}`"
        :class="[
          'mb-1 rounded px-2 py-1',
          message.kind === 'error'
            ? 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'
            : 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300'
        ]"
      >
        {{ message.index }} {{ message.name || '-' }}: {{ message.message }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CodexSessionImportResult } from '@/api/admin/accounts'

const props = defineProps<{
  result: CodexSessionImportResult
}>()

const { t } = useI18n()

const resultSummaryParams = computed<Record<string, number>>(() => ({
  total: props.result.total,
  created: props.result.created,
  updated: props.result.updated,
  skipped: props.result.skipped,
  failed: props.result.failed
}))

const visibleMessages = computed(() => [
  ...(props.result.errors || []).map((item) => ({ ...item, kind: 'error' as const })),
  ...(props.result.warnings || []).map((item) => ({ ...item, kind: 'warning' as const }))
])
</script>
