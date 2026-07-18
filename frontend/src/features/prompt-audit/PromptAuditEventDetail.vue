<template>
  <section v-if="event" class="card overflow-hidden">
    <div class="flex flex-col gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
      <div>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('admin.promptAudit.detail.title', { id: event.id }) }}
        </h2>
        <p class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400">{{ event.request_id }}</p>
      </div>
      <button class="btn btn-danger btn-sm" type="button" :disabled="deleting" @click="$emit('delete', event.id)">
        {{ t('common.delete') }}
      </button>
    </div>
    <div class="grid grid-cols-1 gap-4 p-5 md:grid-cols-2">
      <InfoItem :label="t('admin.promptAudit.detail.decision')" :value="decisionText" />
      <InfoItem :label="t('admin.promptAudit.detail.target')" :value="`${event.provider || '-'} / ${event.model || '-'}`" />
      <InfoItem :label="t('admin.promptAudit.detail.user')" :value="userText" />
      <InfoItem :label="t('admin.promptAudit.detail.group')" :value="groupText" />
      <InfoItem :label="t('admin.promptAudit.detail.endpoint')" :value="event.endpoint || '-'" />
      <InfoItem :label="t('admin.promptAudit.detail.guard')" :value="event.guard_endpoint_id || '-'" />
      <InfoItem class="md:col-span-2" :label="t('admin.promptAudit.detail.promptHash')" :value="event.prompt_hash || '-'" mono />
      <div class="md:col-span-2">
        <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.promptAudit.detail.categories') }}</div>
        <div class="mt-2 flex flex-wrap gap-2">
          <span v-for="category in event.categories || []" :key="category" class="rounded-full bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300">
            {{ category }}
          </span>
          <span v-if="!event.categories?.length" class="text-sm text-gray-700 dark:text-gray-200">-</span>
        </div>
      </div>
      <div class="md:col-span-2">
        <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.promptAudit.detail.preview') }}</div>
        <div class="mt-1 rounded-lg bg-gray-50 p-4 text-sm text-gray-800 dark:bg-dark-800 dark:text-gray-100">
          {{ event.redacted_preview || '-' }}
        </div>
      </div>
      <div v-if="event.full_prompt" class="md:col-span-2">
        <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.promptAudit.detail.fullPrompt') }}</div>
        <pre class="mt-1 max-h-96 overflow-auto rounded-lg bg-gray-950 p-4 text-xs text-gray-100">{{ event.full_prompt }}</pre>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PromptAuditEvent } from '@/api/admin/promptAudit'
import InfoItem from './PromptAuditInfoItem.vue'

const props = defineProps<{
  event: PromptAuditEvent | null
  deleting: boolean
}>()

defineEmits<{ delete: [id: number] }>()

const { t } = useI18n()

const decisionText = computed(() => {
  const event = props.event
  return event ? `${t(`admin.promptAudit.decision.${event.decision}`)} / ${t(`admin.promptAudit.risk.${event.risk_level}`)}` : '-'
})

const userText = computed(() => {
  const event = props.event
  if (!event) return '-'
  return event.username_snapshot || event.user_email_snapshot || String(event.user_id ?? '-')
})

const groupText = computed(() => {
  const event = props.event
  if (!event) return '-'
  return event.group_name || String(event.group_id ?? '-')
})
</script>
