<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OpsRequestTraceDetail, OpsRequestTraceRawDetail } from '@/api/admin/ops'
import { formatDateTime, formatNumber } from '@/utils/format'
import { formatDurationMs, formatPrettyJSON, getProtocolPairLabel } from '../helpers'

const props = defineProps<{
  open: boolean
  detail: OpsRequestTraceDetail | null
  rawDetail: OpsRequestTraceRawDetail | null
  loading: boolean
  rawLoading: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'loadRaw'): void
}>()

const { t } = useI18n()
const activeTab = ref('overview')

const tabs = computed(() => {
  const base = [
    'overview',
    'inbound',
    'normalized',
    'upstreamRequest',
    'upstreamResponse',
    'gatewayResponse',
    'tools',
    'audits'
  ]
  if (props.detail?.raw_access_allowed) {
    base.push('raw')
  }
  return base
})

watch(
  () => props.open,
  (open) => {
    if (open) {
      activeTab.value = 'overview'
    }
  }
)

const jsonPanels = computed(() => ({
  inbound: formatPrettyJSON(props.detail?.inbound_request_json),
  normalized: formatPrettyJSON(props.detail?.normalized_request_json),
  upstreamRequest: formatPrettyJSON(props.detail?.upstream_request_json),
  upstreamResponse: formatPrettyJSON(props.detail?.upstream_response_json),
  gatewayResponse: formatPrettyJSON(props.detail?.gateway_response_json),
  tools: formatPrettyJSON(props.detail?.tool_trace_json),
  rawRequest: formatPrettyJSON(props.rawDetail?.raw_request),
  rawResponse: formatPrettyJSON(props.rawDetail?.raw_response)
}))

function tabLabel(tab: string): string {
  return t(`admin.requestDetails.drawer.tabs.${tab}`)
}
</script>

<template>
  <transition name="fade">
    <div v-if="open" class="fixed inset-0 z-40 bg-black/40" @click="emit('close')"></div>
  </transition>

  <transition name="slide-left">
    <aside
      v-if="open"
      class="fixed right-0 top-0 z-50 flex h-full w-full max-w-[960px] flex-col border-l border-gray-200 bg-white shadow-2xl dark:border-dark-700 dark:bg-dark-900"
    >
      <div class="flex items-start justify-between border-b border-gray-100 px-6 py-5 dark:border-dark-700">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('admin.requestDetails.drawer.title') }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ detail?.request_id || t('admin.requestDetails.drawer.noSelection') }}
          </p>
        </div>
        <button class="btn btn-secondary btn-sm" type="button" @click="emit('close')">
          {{ t('common.close') }}
        </button>
      </div>

      <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
        <div class="flex flex-wrap gap-2">
          <button
            v-for="tab in tabs"
            :key="tab"
            class="rounded-full px-3 py-1.5 text-sm font-medium"
            :class="activeTab === tab ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-700 dark:bg-dark-800 dark:text-gray-300'"
            type="button"
            @click="activeTab = tab"
          >
            {{ tabLabel(tab) }}
          </button>
        </div>
      </div>

      <div class="min-h-0 flex-1 overflow-y-auto px-6 py-5">
        <div v-if="loading" class="flex h-full items-center justify-center text-sm text-gray-400">
          {{ t('common.loading') }}
        </div>

        <div v-else-if="!detail" class="flex h-full items-center justify-center text-sm text-gray-400">
          {{ t('admin.requestDetails.drawer.noSelection') }}
        </div>

        <div v-else-if="activeTab === 'overview'" class="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <div class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-800">
            <div class="text-xs uppercase tracking-wide text-gray-400">{{ t('admin.requestDetails.drawer.sections.identity') }}</div>
            <div class="mt-3 space-y-2 text-sm text-gray-700 dark:text-gray-300">
              <div>{{ formatDateTime(detail.created_at) }}</div>
              <div>{{ getProtocolPairLabel(detail.protocol_in, detail.protocol_out) }}</div>
              <div>{{ detail.route_path || '-' }}</div>
              <div>{{ detail.channel || detail.platform || '-' }}</div>
              <div class="font-mono text-xs">{{ detail.request_id || '-' }}</div>
              <div class="font-mono text-xs">{{ detail.upstream_request_id || '-' }}</div>
            </div>
          </div>

          <div class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-800">
            <div class="text-xs uppercase tracking-wide text-gray-400">{{ t('admin.requestDetails.drawer.sections.execution') }}</div>
            <div class="mt-3 space-y-2 text-sm text-gray-700 dark:text-gray-300">
              <div>{{ detail.status }} / {{ detail.status_code }}</div>
              <div>{{ detail.finish_reason || '-' }}</div>
              <div>{{ formatDurationMs(detail.duration_ms) }} / TTFT {{ formatDurationMs(detail.ttft_ms) }}</div>
              <div>{{ detail.requested_model || '-' }}</div>
              <div>{{ detail.actual_upstream_model || detail.upstream_model || '-' }}</div>
              <div>{{ formatNumber(detail.total_tokens || 0) }} tokens</div>
            </div>
          </div>

          <div class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-800">
            <div class="text-xs uppercase tracking-wide text-gray-400">{{ t('admin.requestDetails.drawer.sections.flags') }}</div>
            <div class="mt-3 flex flex-wrap gap-2 text-xs">
              <span class="badge badge-gray">{{ detail.stream ? 'stream' : 'sync' }}</span>
              <span class="badge" :class="detail.has_tools ? 'badge-primary' : 'badge-gray'">tool</span>
              <span class="badge" :class="detail.has_thinking ? 'badge-warning' : 'badge-gray'">thinking</span>
              <span class="badge" :class="detail.raw_available ? 'badge-success' : 'badge-gray'">raw</span>
            </div>
            <div class="mt-3 space-y-2 text-sm text-gray-700 dark:text-gray-300">
              <div>{{ detail.capture_reason || '-' }}</div>
              <div>{{ detail.thinking_source || '-' }} / {{ detail.thinking_level || '-' }}</div>
              <div>{{ detail.media_resolution || '-' }}</div>
              <div>{{ detail.count_tokens_source || '-' }}</div>
            </div>
          </div>

          <div class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-800">
            <div class="text-xs uppercase tracking-wide text-gray-400">{{ t('admin.requestDetails.drawer.sections.headers') }}</div>
            <pre class="mt-3 max-h-[220px] overflow-auto rounded-xl bg-white p-3 text-xs text-gray-800 dark:bg-dark-900 dark:text-gray-200"><code>{{ formatPrettyJSON(detail.request_headers_json || detail.response_headers_json) }}</code></pre>
          </div>
        </div>

        <div v-else-if="activeTab === 'audits'" class="space-y-3">
          <div
            v-for="audit in detail.audits"
            :key="audit.id"
            class="rounded-2xl border border-gray-200 p-4 dark:border-dark-700"
          >
            <div class="flex items-center justify-between gap-4">
              <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ audit.action }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(audit.created_at) }}</div>
            </div>
            <div class="mt-2 text-sm text-gray-600 dark:text-gray-300">
              {{ t('admin.requestDetails.drawer.auditOperator', { id: audit.operator_id }) }}
            </div>
            <pre class="mt-3 overflow-auto rounded-xl bg-gray-50 p-3 text-xs text-gray-700 dark:bg-dark-800 dark:text-gray-200"><code>{{ formatPrettyJSON(audit.meta_json) }}</code></pre>
          </div>
          <div v-if="detail.audits.length === 0" class="text-sm text-gray-400">
            {{ t('common.noData') }}
          </div>
        </div>

        <div v-else-if="activeTab === 'raw'" class="space-y-4">
          <div v-if="!detail.raw_access_allowed" class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-200">
            {{ t('admin.requestDetails.drawer.rawNotAllowed') }}
          </div>
          <template v-else>
            <div class="flex justify-end">
              <button class="btn btn-secondary btn-sm" type="button" :disabled="rawLoading" @click="emit('loadRaw')">
                {{ rawDetail ? t('common.refresh') : t('admin.requestDetails.drawer.loadRaw') }}
              </button>
            </div>
            <div v-if="rawLoading" class="text-sm text-gray-400">{{ t('common.loading') }}</div>
            <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
              <div class="rounded-2xl border border-gray-200 p-4 dark:border-dark-700">
                <div class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.requestDetails.drawer.tabs.raw') }} / Request
                </div>
                <pre class="max-h-[520px] overflow-auto rounded-xl bg-gray-50 p-3 text-xs text-gray-800 dark:bg-dark-800 dark:text-gray-200"><code>{{ jsonPanels.rawRequest || '-' }}</code></pre>
              </div>
              <div class="rounded-2xl border border-gray-200 p-4 dark:border-dark-700">
                <div class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.requestDetails.drawer.tabs.raw') }} / Response
                </div>
                <pre class="max-h-[520px] overflow-auto rounded-xl bg-gray-50 p-3 text-xs text-gray-800 dark:bg-dark-800 dark:text-gray-200"><code>{{ jsonPanels.rawResponse || '-' }}</code></pre>
              </div>
            </div>
          </template>
        </div>

        <div v-else class="space-y-4">
          <pre class="max-h-[720px] overflow-auto rounded-2xl border border-gray-200 bg-gray-50 p-4 text-xs text-gray-800 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200"><code>{{
            activeTab === 'inbound'
              ? jsonPanels.inbound
              : activeTab === 'normalized'
                ? jsonPanels.normalized
                : activeTab === 'upstreamRequest'
                  ? jsonPanels.upstreamRequest
                  : activeTab === 'upstreamResponse'
                    ? jsonPanels.upstreamResponse
                    : activeTab === 'gatewayResponse'
                      ? jsonPanels.gatewayResponse
                      : jsonPanels.tools
          }}</code></pre>
        </div>
      </div>
    </aside>
  </transition>
</template>

<style scoped>
.slide-left-enter-active,
.slide-left-leave-active {
  transition: transform 0.2s ease;
}

.slide-left-enter-from,
.slide-left-leave-to {
  transform: translateX(100%);
}
</style>
