<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OpsRequestTraceFilter } from '@/api/admin/ops'
import { createDefaultRequestTraceFilter } from '../helpers'

const props = defineProps<{
  modelValue: OpsRequestTraceFilter
  loading: boolean
  rawExportAllowed: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: OpsRequestTraceFilter): void
  (e: 'apply'): void
  (e: 'reset'): void
  (e: 'export', includeRaw: boolean): void
}>()

const { t } = useI18n()
const draft = reactive<OpsRequestTraceFilter>(createDefaultRequestTraceFilter())

function syncDraft(source: OpsRequestTraceFilter) {
  Object.assign(draft, createDefaultRequestTraceFilter(), source)
}

watch(
  () => props.modelValue,
  (value) => syncDraft(value),
  { immediate: true, deep: true }
)

function emitFilter() {
  emit('update:modelValue', { ...draft })
}

function updateField<K extends keyof OpsRequestTraceFilter>(key: K, value: OpsRequestTraceFilter[K]) {
  draft[key] = value
  if (key !== 'page') {
    draft.page = 1
  }
  emitFilter()
}

const timeRangeOptions = computed(() => ([
  { value: '5m', label: t('admin.requestDetails.filters.timeRangeOptions.5m') },
  { value: '30m', label: t('admin.requestDetails.filters.timeRangeOptions.30m') },
  { value: '1h', label: t('admin.requestDetails.filters.timeRangeOptions.1h') },
  { value: '6h', label: t('admin.requestDetails.filters.timeRangeOptions.6h') },
  { value: '24h', label: t('admin.requestDetails.filters.timeRangeOptions.24h') },
  { value: '7d', label: t('admin.requestDetails.filters.timeRangeOptions.7d') },
  { value: '30d', label: t('admin.requestDetails.filters.timeRangeOptions.30d') }
]))

const booleanOptions = computed(() => ([
  { value: '', label: t('admin.requestDetails.filters.any') },
  { value: 'true', label: t('common.yes') },
  { value: 'false', label: t('common.no') }
]))

function readBoolean(value: string): boolean | undefined {
  if (value === 'true') return true
  if (value === 'false') return false
  return undefined
}

function toLocalDateTimeValue(value?: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hour}:${minute}`
}

const localStartTime = computed({
  get: () => toLocalDateTimeValue(draft.start_time),
  set: (value: string) => updateField('start_time', value ? new Date(value).toISOString() : undefined)
})

const localEndTime = computed({
  get: () => toLocalDateTimeValue(draft.end_time),
  set: (value: string) => updateField('end_time', value ? new Date(value).toISOString() : undefined)
})

function resetFilters() {
  emit('reset')
}
</script>

<template>
  <div class="rounded-3xl bg-white p-6 shadow-sm ring-1 ring-gray-900/5 dark:bg-dark-800 dark:ring-dark-700">
    <div class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('admin.requestDetails.filters.title') }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.requestDetails.filters.description') }}
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <button class="btn btn-secondary" type="button" @click="resetFilters">
          {{ t('common.reset') }}
        </button>
        <button class="btn btn-secondary" type="button" :disabled="loading" @click="emit('export', false)">
          {{ t('admin.requestDetails.actions.exportMasked') }}
        </button>
        <button
          v-if="rawExportAllowed"
          class="btn btn-secondary"
          type="button"
          :disabled="loading"
          @click="emit('export', true)"
        >
          {{ t('admin.requestDetails.actions.exportRaw') }}
        </button>
        <button class="btn btn-primary" type="button" :disabled="loading" @click="emit('apply')">
          {{ t('common.search') }}
        </button>
      </div>
    </div>

    <div class="mt-6 grid grid-cols-1 gap-4 lg:grid-cols-2 xl:grid-cols-5">
      <div class="xl:col-span-2">
        <label class="input-label">{{ t('admin.requestDetails.filters.q') }}</label>
        <input
          :value="draft.q || ''"
          type="text"
          class="input"
          :placeholder="t('admin.requestDetails.filters.qPlaceholder')"
          @input="updateField('q', String(($event.target as HTMLInputElement).value || ''))"
          @keydown.enter="emit('apply')"
        />
      </div>

      <div>
        <label class="input-label">{{ t('admin.requestDetails.filters.timeRange') }}</label>
        <select class="input" :value="draft.time_range || '1h'" @change="updateField('time_range', ($event.target as HTMLSelectElement).value as OpsRequestTraceFilter['time_range'])">
          <option v-for="option in timeRangeOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
      </div>

      <div>
        <label class="input-label">{{ t('admin.requestDetails.filters.startTime') }}</label>
        <input v-model="localStartTime" type="datetime-local" class="input" />
      </div>

      <div>
        <label class="input-label">{{ t('admin.requestDetails.filters.endTime') }}</label>
        <input v-model="localEndTime" type="datetime-local" class="input" />
      </div>
    </div>

    <details class="mt-4 rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900">
      <summary class="cursor-pointer text-sm font-semibold text-gray-800 dark:text-gray-100">
        {{ t('admin.requestDetails.filters.advanced') }}
      </summary>

      <div class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.platform') }}</label>
          <input :value="draft.platform || ''" type="text" class="input" @input="updateField('platform', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.protocolIn') }}</label>
          <input :value="draft.protocol_in || ''" type="text" class="input" @input="updateField('protocol_in', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.protocolOut') }}</label>
          <input :value="draft.protocol_out || ''" type="text" class="input" @input="updateField('protocol_out', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.channel') }}</label>
          <input :value="draft.channel || ''" type="text" class="input" @input="updateField('channel', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.routePath') }}</label>
          <input :value="draft.route_path || ''" type="text" class="input" @input="updateField('route_path', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.status') }}</label>
          <input :value="draft.status || ''" type="text" class="input" @input="updateField('status', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.finishReason') }}</label>
          <input :value="draft.finish_reason || ''" type="text" class="input" @input="updateField('finish_reason', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.requestType') }}</label>
          <input :value="draft.request_type || ''" type="text" class="input" @input="updateField('request_type', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.userId') }}</label>
          <input :value="draft.user_id || ''" type="number" min="1" class="input" @input="updateField('user_id', Number.parseInt(($event.target as HTMLInputElement).value, 10) || undefined)" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.apiKeyId') }}</label>
          <input :value="draft.api_key_id || ''" type="number" min="1" class="input" @input="updateField('api_key_id', Number.parseInt(($event.target as HTMLInputElement).value, 10) || undefined)" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.accountId') }}</label>
          <input :value="draft.account_id || ''" type="number" min="1" class="input" @input="updateField('account_id', Number.parseInt(($event.target as HTMLInputElement).value, 10) || undefined)" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.groupId') }}</label>
          <input :value="draft.group_id || ''" type="number" min="1" class="input" @input="updateField('group_id', Number.parseInt(($event.target as HTMLInputElement).value, 10) || undefined)" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.requestedModel') }}</label>
          <input :value="draft.requested_model || ''" type="text" class="input" @input="updateField('requested_model', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.upstreamModel') }}</label>
          <input :value="draft.upstream_model || ''" type="text" class="input" @input="updateField('upstream_model', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.requestId') }}</label>
          <input :value="draft.request_id || ''" type="text" class="input" @input="updateField('request_id', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.upstreamRequestId') }}</label>
          <input :value="draft.upstream_request_id || ''" type="text" class="input" @input="updateField('upstream_request_id', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.clientRequestId') }}</label>
          <input :value="draft.client_request_id || ''" type="text" class="input" @input="updateField('client_request_id', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.geminiSurface') }}</label>
          <input :value="draft.gemini_surface || ''" type="text" class="input" @input="updateField('gemini_surface', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.billingRuleId') }}</label>
          <input :value="draft.billing_rule_id || ''" type="text" class="input" @input="updateField('billing_rule_id', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.probeAction') }}</label>
          <input :value="draft.probe_action || ''" type="text" class="input" @input="updateField('probe_action', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.captureReason') }}</label>
          <input :value="draft.capture_reason || ''" type="text" class="input" @input="updateField('capture_reason', String(($event.target as HTMLInputElement).value || ''))" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.stream') }}</label>
          <select class="input" :value="draft.stream == null ? '' : String(draft.stream)" @change="updateField('stream', readBoolean(($event.target as HTMLSelectElement).value))">
            <option v-for="option in booleanOptions" :key="`stream-${option.value}`" :value="option.value">{{ option.label }}</option>
          </select>
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.hasTools') }}</label>
          <select class="input" :value="draft.has_tools == null ? '' : String(draft.has_tools)" @change="updateField('has_tools', readBoolean(($event.target as HTMLSelectElement).value))">
            <option v-for="option in booleanOptions" :key="`tool-${option.value}`" :value="option.value">{{ option.label }}</option>
          </select>
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.hasThinking') }}</label>
          <select class="input" :value="draft.has_thinking == null ? '' : String(draft.has_thinking)" @change="updateField('has_thinking', readBoolean(($event.target as HTMLSelectElement).value))">
            <option v-for="option in booleanOptions" :key="`thinking-${option.value}`" :value="option.value">{{ option.label }}</option>
          </select>
        </div>
        <div>
          <label class="input-label">{{ t('admin.requestDetails.filters.rawAvailable') }}</label>
          <select class="input" :value="draft.raw_available == null ? '' : String(draft.raw_available)" @change="updateField('raw_available', readBoolean(($event.target as HTMLSelectElement).value))">
            <option v-for="option in booleanOptions" :key="`raw-${option.value}`" :value="option.value">{{ option.label }}</option>
          </select>
        </div>
      </div>
    </details>
  </div>
</template>
