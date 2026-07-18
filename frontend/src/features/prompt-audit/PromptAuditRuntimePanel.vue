<template>
  <section class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
    <div v-for="item in summary" :key="item.key" class="card p-4">
      <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">
        {{ item.label }}
      </div>
      <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
        {{ item.value }}
      </div>
      <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {{ item.detail }}
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PromptAuditRuntime } from '@/api/admin/promptAudit'

const props = defineProps<{ runtime: PromptAuditRuntime | null }>()
const { t } = useI18n()

const summary = computed(() => {
  const rt = props.runtime
  return [
    {
      key: 'mode',
      label: t('admin.promptAudit.runtime.mode'),
      value: rt?.effective_mode || 'off',
      detail: rt?.process_status || '-'
    },
    {
      key: 'queue',
      label: t('admin.promptAudit.runtime.queue'),
      value: rt ? rt.queue.queued + rt.queue.retry : 0,
      detail: t('admin.promptAudit.runtime.queueDetail', {
        processing: rt?.queue.processing || 0,
        failed: rt?.queue.failed || 0
      })
    },
    {
      key: 'workers',
      label: t('admin.promptAudit.runtime.workers'),
      value: rt ? `${rt.worker_active}/${rt.worker_total}` : '0/0',
      detail: t('admin.promptAudit.runtime.capacity', { capacity: rt?.queue_capacity || 0 })
    },
    {
      key: 'metrics',
      label: t('admin.promptAudit.runtime.metrics'),
      value: rt?.guard_metrics.block || 0,
      detail: t('admin.promptAudit.runtime.metricsDetail', {
        allow: rt?.guard_metrics.allow || 0,
        flag: rt?.guard_metrics.flag || 0
      })
    }
  ]
})
</script>
