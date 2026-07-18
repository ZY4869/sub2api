<template>
  <section class="card overflow-hidden">
    <div class="border-b border-gray-100 p-4 dark:border-dark-700">
      <div class="grid grid-cols-1 gap-3 md:grid-cols-3 xl:grid-cols-5">
        <input v-model="filters.request_id" class="input font-mono text-sm" :placeholder="t('admin.promptAudit.filters.requestId')" @keyup.enter="apply" />
        <input v-model="filters.prompt_hash" class="input font-mono text-sm" :placeholder="t('admin.promptAudit.filters.promptHash')" @keyup.enter="apply" />
        <input v-model="filters.keyword" class="input" :placeholder="t('admin.promptAudit.filters.keyword')" @keyup.enter="apply" />
        <select v-model="filters.decision" class="input" @change="apply">
          <option value="">{{ t('admin.promptAudit.filters.allDecisions') }}</option>
          <option value="pass">{{ t('admin.promptAudit.decision.pass') }}</option>
          <option value="flag">{{ t('admin.promptAudit.decision.flag') }}</option>
          <option value="critical">{{ t('admin.promptAudit.decision.critical') }}</option>
        </select>
        <select v-model="filters.risk_level" class="input" @change="apply">
          <option value="">{{ t('admin.promptAudit.filters.allRisks') }}</option>
          <option value="low">{{ t('admin.promptAudit.risk.low') }}</option>
          <option value="medium">{{ t('admin.promptAudit.risk.medium') }}</option>
          <option value="high">{{ t('admin.promptAudit.risk.high') }}</option>
          <option value="critical">{{ t('admin.promptAudit.risk.critical') }}</option>
        </select>
      </div>
      <div class="mt-3 flex flex-wrap gap-2">
        <button class="btn btn-primary" type="button" @click="apply">{{ t('common.search') }}</button>
        <button class="btn btn-secondary" type="button" @click="$emit('reset')">{{ t('common.reset') }}</button>
        <button class="btn btn-secondary" type="button" :disabled="loading" @click="$emit('refresh')">{{ t('common.refresh') }}</button>
        <button class="btn btn-danger" type="button" :disabled="loading || total === 0" @click="$emit('preview-delete')">
          {{ t('admin.promptAudit.events.previewDelete') }}
        </button>
      </div>
      <div v-if="deletePreview" class="mt-3 rounded-lg bg-amber-50 p-3 text-sm text-amber-800 dark:bg-amber-500/10 dark:text-amber-200">
        {{ t('admin.promptAudit.events.deletePreview', { count: deletePreview.matched }) }}
        <button class="btn btn-danger btn-sm ml-3" type="button" @click="$emit('delete-filter')">
          {{ t('admin.promptAudit.events.confirmDeleteFilter') }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('common.loading') }}
    </div>
    <div v-else-if="events.length === 0" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('admin.promptAudit.events.empty') }}
    </div>
    <div v-else class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
        <thead class="bg-gray-50 dark:bg-dark-800/60">
          <tr>
            <th v-for="column in columns" :key="column" class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
              {{ t(`admin.promptAudit.events.columns.${column}`) }}
            </th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
          <tr v-for="event in events" :key="event.id" class="hover:bg-gray-50 dark:hover:bg-dark-800/40">
            <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">{{ formatDate(event.created_at) }}</td>
            <td class="px-4 py-3 text-sm">
              <span class="rounded-full px-2 py-1 text-xs font-medium" :class="decisionClass(event.decision)">
                {{ t(`admin.promptAudit.decision.${event.decision}`) }}
              </span>
              <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t(`admin.promptAudit.risk.${event.risk_level}`) }}</div>
            </td>
            <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
              <div class="font-medium">{{ event.provider || '-' }}</div>
              <div class="font-mono text-xs text-gray-500 dark:text-gray-400">{{ event.model || '-' }}</div>
            </td>
            <td class="max-w-md px-4 py-3 text-sm text-gray-700 dark:text-gray-200">{{ event.redacted_preview || '-' }}</td>
            <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
              <div class="font-mono text-xs">{{ event.request_id || '-' }}</div>
              <div class="font-mono text-xs text-gray-500 dark:text-gray-400">{{ event.prompt_hash || '-' }}</div>
            </td>
            <td class="px-4 py-3 text-right text-sm">
              <button class="btn btn-ghost btn-sm" type="button" @click="$emit('open', event.id)">{{ t('common.view') }}</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <Pagination
      v-if="total > 0"
      :page="filters.page || 1"
      :total="total"
      :page-size="filters.page_size || 20"
      @update:page="$emit('page', $event)"
      @update:pageSize="$emit('page-size', $event)"
    />
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { PromptAuditDeletePreview, PromptAuditEvent, PromptAuditEventFilter } from '@/api/admin/promptAudit'
import Pagination from '@/components/common/Pagination.vue'

const filters = defineModel<PromptAuditEventFilter>('filters', { required: true })

defineProps<{
  events: PromptAuditEvent[]
  total: number
  loading: boolean
  deletePreview: PromptAuditDeletePreview | null
}>()

const emit = defineEmits<{
  refresh: []
  reset: []
  open: [id: number]
  page: [page: number]
  'page-size': [pageSize: number]
  'preview-delete': []
  'delete-filter': []
}>()

const { t } = useI18n()
const columns = ['createdAt', 'decision', 'target', 'preview', 'request', 'actions']

function apply() {
  emit('page', 1)
}

function formatDate(value: string) {
  if (!value) return '-'
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}

function decisionClass(decision: string) {
  if (decision === 'critical') return 'bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300'
  if (decision === 'flag') return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
  return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
}
</script>
