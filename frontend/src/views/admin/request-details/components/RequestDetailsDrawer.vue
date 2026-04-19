<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ProtocolPairDisplay from '@/components/common/ProtocolPairDisplay.vue'
import type { OpsRequestTraceDetail, OpsRequestTraceRawDetail } from '@/api/admin/ops'
import { parseRequestPreviewContent } from '@/utils/requestPreview'
import { formatDateTime, formatNumber } from '@/utils/format'
import RequestDetailsContentDialog from './RequestDetailsContentDialog.vue'
import RequestDetailsPayloadPanel from './RequestDetailsPayloadPanel.vue'
import {
  formatDurationMs,
  formatPrettyJSON,
  getRequestTraceCapabilityFields,
  getRequestTraceExecutionFields,
  getRequestTraceFlagBadges,
  getRequestTraceIdentityFields,
  getRequestTraceRequestTypeLabel,
  getRequestTraceRouteFields,
  getRequestTraceStatusLabel,
  getRequestTraceSubjectFields,
  getStatusBadgeClass,
  resolveRequestTraceModelPresentation
} from '../helpers'

type PayloadTabKey = 'inbound' | 'normalized' | 'upstreamRequest' | 'upstreamResponse' | 'gatewayResponse' | 'tools'
type DialogPayloadKey = PayloadTabKey | 'rawRequest' | 'rawResponse'

interface FullDialogState {
  show: boolean
  title: string
  content: string
  notice: string
  emptyMessage: string
  loading: boolean
}

interface PayloadPanelState {
  title: string
  content: string
  emptyMessage: string
  notice: string
  canOpenFull: boolean
  source: string
  truncated: boolean
  state: 'uncollected' | 'empty' | 'raw_only' | 'ready'
}

const payloadTabKeys: PayloadTabKey[] = [
  'inbound',
  'normalized',
  'upstreamRequest',
  'upstreamResponse',
  'gatewayResponse',
  'tools'
]

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
const pendingRawDialogKey = ref<'inbound' | 'rawRequest' | 'rawResponse' | null>(null)
const fullDialog = ref<FullDialogState>(createEmptyDialogState())

function createEmptyDialogState(): FullDialogState {
  return {
    show: false,
    title: '',
    content: '',
    notice: '',
    emptyMessage: '',
    loading: false
  }
}

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

const requestedModel = computed(() =>
  resolveRequestTraceModelPresentation(props.detail?.requested_model)
)
const upstreamModel = computed(() =>
  resolveRequestTraceModelPresentation(props.detail?.actual_upstream_model || props.detail?.upstream_model)
)

const identityFields = computed(() => {
  if (!props.detail) return []
  return [
    {
      label: t('admin.requestDetails.presentation.labels.createdAt'),
      value: formatDateTime(props.detail.created_at),
      mono: false
    },
    {
      label: t('admin.requestDetails.presentation.labels.requestType'),
      value: getRequestTraceRequestTypeLabel(t, props.detail.request_type),
      mono: false
    },
    ...getRequestTraceRouteFields(t, props.detail)
  ]
})

const identifierFields = computed(() => {
  if (!props.detail) return []
  return [...getRequestTraceIdentityFields(t, props.detail), ...getRequestTraceSubjectFields(t, props.detail)]
})

const executionFields = computed(() => {
  if (!props.detail) return []
  return getRequestTraceExecutionFields(t, props.detail)
})

const capabilityFields = computed(() => {
  if (!props.detail) return []
  return getRequestTraceCapabilityFields(t, props.detail)
})

const flagBadges = computed(() => {
  if (!props.detail) return []
  return getRequestTraceFlagBadges(t, props.detail)
})

const requestHeaders = computed(() => formatPrettyJSON(props.detail?.request_headers_json))
const responseHeaders = computed(() => formatPrettyJSON(props.detail?.response_headers_json))
const rawBackedInboundAvailable = computed(() => Boolean(props.detail?.raw_access_allowed && props.detail?.raw_available))

const parsedPanels = computed(() => ({
  inbound: parseRequestPreviewContent(props.detail?.inbound_request_json),
  normalized: parseRequestPreviewContent(props.detail?.normalized_request_json),
  upstreamRequest: parseRequestPreviewContent(props.detail?.upstream_request_json),
  upstreamResponse: parseRequestPreviewContent(props.detail?.upstream_response_json),
  gatewayResponse: parseRequestPreviewContent(props.detail?.gateway_response_json),
  tools: parseRequestPreviewContent(props.detail?.tool_trace_json),
  rawRequest: parseRequestPreviewContent(props.rawDetail?.raw_request),
  rawResponse: parseRequestPreviewContent(props.rawDetail?.raw_response)
}))

function resolvePanelEmptyMessage(tab: DialogPayloadKey): string {
  const state = parsedPanels.value[tab].renderState
  if (state === 'empty') {
    return t('admin.requestDetails.drawer.payload.collectedEmpty')
  }
  if (state === 'raw_only') {
    return t('admin.requestDetails.drawer.payload.rawOnlyEmpty')
  }
  switch (tab) {
    case 'inbound':
      return t('admin.requestDetails.drawer.emptyStates.inbound')
    case 'normalized':
      return t('admin.requestDetails.drawer.emptyStates.normalized')
    case 'upstreamRequest':
      return t('admin.requestDetails.drawer.emptyStates.upstreamRequest')
    case 'upstreamResponse':
      return t('admin.requestDetails.drawer.emptyStates.upstreamResponse')
    case 'gatewayResponse':
      return t('admin.requestDetails.drawer.emptyStates.gatewayResponse')
    case 'tools':
      return t('admin.requestDetails.drawer.emptyStates.tools')
    case 'rawRequest':
      return t('admin.requestDetails.drawer.emptyStates.rawRequest')
    case 'rawResponse':
      return t('admin.requestDetails.drawer.emptyStates.rawResponse')
    default:
      return t('common.noData')
  }
}

const payloadPanels = computed<Record<DialogPayloadKey, PayloadPanelState>>(() => ({
  inbound: {
    title: tabLabel('inbound'),
    content: parsedPanels.value.inbound.displayContent,
    emptyMessage: resolvePanelEmptyMessage('inbound'),
    notice: parsedPanels.value.inbound.hasContent && !rawBackedInboundAvailable.value
      ? t('admin.requestDetails.drawer.previewOnlyNotice')
      : '',
    canOpenFull: rawBackedInboundAvailable.value || parsedPanels.value.inbound.hasContent,
    source: parsedPanels.value.inbound.source,
    truncated: parsedPanels.value.inbound.truncated,
    state: parsedPanels.value.inbound.renderState
  },
  normalized: {
    title: tabLabel('normalized'),
    content: parsedPanels.value.normalized.displayContent,
    emptyMessage: resolvePanelEmptyMessage('normalized'),
    notice: '',
    canOpenFull: parsedPanels.value.normalized.hasContent,
    source: parsedPanels.value.normalized.source,
    truncated: parsedPanels.value.normalized.truncated,
    state: parsedPanels.value.normalized.renderState
  },
  upstreamRequest: {
    title: tabLabel('upstreamRequest'),
    content: parsedPanels.value.upstreamRequest.displayContent,
    emptyMessage: resolvePanelEmptyMessage('upstreamRequest'),
    notice: '',
    canOpenFull: parsedPanels.value.upstreamRequest.hasContent,
    source: parsedPanels.value.upstreamRequest.source,
    truncated: parsedPanels.value.upstreamRequest.truncated,
    state: parsedPanels.value.upstreamRequest.renderState
  },
  upstreamResponse: {
    title: tabLabel('upstreamResponse'),
    content: parsedPanels.value.upstreamResponse.displayContent,
    emptyMessage: resolvePanelEmptyMessage('upstreamResponse'),
    notice: '',
    canOpenFull: parsedPanels.value.upstreamResponse.hasContent,
    source: parsedPanels.value.upstreamResponse.source,
    truncated: parsedPanels.value.upstreamResponse.truncated,
    state: parsedPanels.value.upstreamResponse.renderState
  },
  gatewayResponse: {
    title: tabLabel('gatewayResponse'),
    content: parsedPanels.value.gatewayResponse.displayContent,
    emptyMessage: resolvePanelEmptyMessage('gatewayResponse'),
    notice: '',
    canOpenFull: parsedPanels.value.gatewayResponse.hasContent,
    source: parsedPanels.value.gatewayResponse.source,
    truncated: parsedPanels.value.gatewayResponse.truncated,
    state: parsedPanels.value.gatewayResponse.renderState
  },
  tools: {
    title: tabLabel('tools'),
    content: parsedPanels.value.tools.displayContent,
    emptyMessage: resolvePanelEmptyMessage('tools'),
    notice: '',
    canOpenFull: parsedPanels.value.tools.hasContent,
    source: parsedPanels.value.tools.source,
    truncated: parsedPanels.value.tools.truncated,
    state: parsedPanels.value.tools.renderState
  },
  rawRequest: {
    title: t('admin.requestDetails.presentation.labels.rawRequest'),
    content: parsedPanels.value.rawRequest.displayContent,
    emptyMessage: resolvePanelEmptyMessage('rawRequest'),
    notice: '',
    canOpenFull: parsedPanels.value.rawRequest.hasContent,
    source: parsedPanels.value.rawRequest.source,
    truncated: parsedPanels.value.rawRequest.truncated,
    state: parsedPanels.value.rawRequest.renderState
  },
  rawResponse: {
    title: t('admin.requestDetails.presentation.labels.rawResponse'),
    content: parsedPanels.value.rawResponse.displayContent,
    emptyMessage: resolvePanelEmptyMessage('rawResponse'),
    notice: '',
    canOpenFull: parsedPanels.value.rawResponse.hasContent,
    source: parsedPanels.value.rawResponse.source,
    truncated: parsedPanels.value.rawResponse.truncated,
    state: parsedPanels.value.rawResponse.renderState
  }
}))

const currentPayloadPanel = computed(() => {
  if (!isPayloadTab(activeTab.value)) return null
  return payloadPanels.value[activeTab.value]
})

function isPayloadTab(tab: string): tab is PayloadTabKey {
  return payloadTabKeys.includes(tab as PayloadTabKey)
}

function tabLabel(tab: string): string {
  return t(`admin.requestDetails.drawer.tabs.${tab}`)
}

function buildFullDialogTitle(tabTitle: string): string {
  return t('admin.requestDetails.drawer.fullDialogTitle', {
    tab: tabTitle,
    id: props.detail?.request_id || '-'
  })
}

function openFullDialog(next: Omit<FullDialogState, 'show'>) {
  fullDialog.value = {
    show: true,
    ...next
  }
}

function closeFullDialog() {
  fullDialog.value = createEmptyDialogState()
}

function handleOpenPayloadFull(tab: PayloadTabKey) {
  if (tab === 'inbound' && rawBackedInboundAvailable.value) {
    if (payloadPanels.value.rawRequest.content) {
      openFullDialog({
        title: buildFullDialogTitle(payloadPanels.value.inbound.title),
        content: payloadPanels.value.rawRequest.content,
        notice: '',
        emptyMessage: payloadPanels.value.rawRequest.emptyMessage,
        loading: false
      })
      return
    }

    pendingRawDialogKey.value = 'inbound'
    openFullDialog({
      title: buildFullDialogTitle(payloadPanels.value.inbound.title),
      content: '',
      notice: '',
      emptyMessage: payloadPanels.value.rawRequest.emptyMessage,
      loading: true
    })
    emit('loadRaw')
    return
  }

  const panel = payloadPanels.value[tab]
  openFullDialog({
    title: buildFullDialogTitle(panel.title),
    content: panel.content,
    notice: panel.notice,
    emptyMessage: panel.emptyMessage,
    loading: false
  })
}

function handleOpenRawFull(tab: 'rawRequest' | 'rawResponse') {
  const panel = payloadPanels.value[tab]
  openFullDialog({
    title: buildFullDialogTitle(panel.title),
    content: panel.content,
    notice: '',
    emptyMessage: panel.emptyMessage,
    loading: false
  })
}

watch(
  () => props.open,
  (open) => {
    activeTab.value = 'overview'
    pendingRawDialogKey.value = null
    if (!open) {
      closeFullDialog()
    }
  }
)

watch(
  () => [props.rawLoading, props.rawDetail?.raw_request, props.rawDetail?.raw_response, props.open] as const,
  ([rawLoading, , , open]) => {
    if (!open || !pendingRawDialogKey.value || rawLoading) return
    const pendingKey = pendingRawDialogKey.value
    pendingRawDialogKey.value = null

    const sourcePanel = pendingKey === 'inbound' ? payloadPanels.value.rawRequest : payloadPanels.value[pendingKey]
    const title = pendingKey === 'inbound'
      ? buildFullDialogTitle(payloadPanels.value.inbound.title)
      : buildFullDialogTitle(sourcePanel.title)

    openFullDialog({
      title,
      content: sourcePanel.content,
      notice: '',
      emptyMessage: sourcePanel.emptyMessage,
      loading: false
    })
  }
)
</script>

<template>
  <transition name="fade">
    <div v-if="open" class="fixed inset-0 z-40 bg-black/40" @click="emit('close')"></div>
  </transition>

  <transition name="slide-left">
    <aside
      v-if="open"
      class="fixed right-0 top-0 z-50 flex h-full w-full max-w-[1040px] flex-col border-l border-gray-200 bg-white shadow-2xl dark:border-dark-700 dark:bg-dark-900"
    >
      <div class="flex items-start justify-between border-b border-gray-100 px-6 py-5 dark:border-dark-700">
        <div class="min-w-0">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('admin.requestDetails.drawer.title') }}
          </h2>
          <div v-if="detail" class="mt-2 flex flex-wrap items-center gap-2">
            <span class="badge" :class="getStatusBadgeClass(detail.status)">
              {{ getRequestTraceStatusLabel(t, detail.status) }}
            </span>
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ getRequestTraceRequestTypeLabel(t, detail.request_type) }}
            </span>
            <span class="truncate font-mono text-xs text-gray-500 dark:text-gray-400">
              {{ detail.request_id || t('admin.requestDetails.drawer.noSelection') }}
            </span>
          </div>
          <p v-else class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.requestDetails.drawer.noSelection') }}
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

        <div v-else-if="activeTab === 'overview'" class="space-y-6">
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
            <div class="rounded-3xl bg-gradient-to-br from-blue-50 to-white p-4 ring-1 ring-blue-100 dark:from-blue-950/20 dark:to-dark-800 dark:ring-blue-900/40">
              <div class="text-[11px] font-medium uppercase tracking-wide text-blue-500 dark:text-blue-300">
                {{ t('admin.requestDetails.presentation.cards.status') }}
              </div>
              <div class="mt-3 flex items-center gap-2">
                <span class="badge" :class="getStatusBadgeClass(detail.status)">
                  {{ getRequestTraceStatusLabel(t, detail.status) }}
                </span>
                <span class="text-sm text-gray-500 dark:text-gray-400">{{ detail.status_code }}</span>
              </div>
              <div class="mt-3">
                <ProtocolPairDisplay
                  compact
                  :protocol-in="detail.protocol_in"
                  :protocol-out="detail.protocol_out"
                />
              </div>
            </div>

            <div class="rounded-3xl bg-gradient-to-br from-violet-50 to-white p-4 ring-1 ring-violet-100 dark:from-violet-950/20 dark:to-dark-800 dark:ring-violet-900/40">
              <div class="text-[11px] font-medium uppercase tracking-wide text-violet-500 dark:text-violet-300">
                {{ t('admin.requestDetails.presentation.cards.requestedModel') }}
              </div>
              <div class="mt-3 flex items-center gap-3">
                <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-white shadow-sm ring-1 ring-violet-100 dark:bg-dark-800 dark:ring-violet-900/40">
                  <ModelIcon
                    :model="requestedModel?.modelId || detail.requested_model"
                    :provider="requestedModel?.provider"
                    :display-name="requestedModel?.displayName"
                    size="18px"
                  />
                </span>
                <div class="min-w-0">
                  <div class="truncate text-sm font-semibold text-gray-900 dark:text-white">
                    {{ requestedModel?.displayName || detail.requested_model || '-' }}
                  </div>
                  <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                    {{ requestedModel?.modelId || detail.requested_model || '-' }}
                  </div>
                </div>
              </div>
            </div>

            <div class="rounded-3xl bg-gradient-to-br from-emerald-50 to-white p-4 ring-1 ring-emerald-100 dark:from-emerald-950/20 dark:to-dark-800 dark:ring-emerald-900/40">
              <div class="text-[11px] font-medium uppercase tracking-wide text-emerald-500 dark:text-emerald-300">
                {{ t('admin.requestDetails.presentation.cards.upstreamModel') }}
              </div>
              <div class="mt-3 flex items-center gap-3">
                <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-white shadow-sm ring-1 ring-emerald-100 dark:bg-dark-800 dark:ring-emerald-900/40">
                  <ModelIcon
                    :model="upstreamModel?.modelId || (detail.actual_upstream_model || detail.upstream_model)"
                    :provider="upstreamModel?.provider"
                    :display-name="upstreamModel?.displayName"
                    size="18px"
                  />
                </span>
                <div class="min-w-0">
                  <div class="truncate text-sm font-semibold text-gray-900 dark:text-white">
                    {{ upstreamModel?.displayName || detail.actual_upstream_model || detail.upstream_model || '-' }}
                  </div>
                  <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                    {{ upstreamModel?.modelId || detail.actual_upstream_model || detail.upstream_model || '-' }}
                  </div>
                </div>
              </div>
            </div>

            <div class="rounded-3xl bg-gradient-to-br from-amber-50 to-white p-4 ring-1 ring-amber-100 dark:from-amber-950/20 dark:to-dark-800 dark:ring-amber-900/40">
              <div class="text-[11px] font-medium uppercase tracking-wide text-amber-500 dark:text-amber-300">
                {{ t('admin.requestDetails.presentation.cards.performance') }}
              </div>
              <div class="mt-3 text-lg font-semibold text-gray-900 dark:text-white">
                {{ formatDurationMs(detail.duration_ms) }}
              </div>
              <div class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                TTFT {{ formatDurationMs(detail.ttft_ms) }}
              </div>
              <div class="mt-3 text-sm text-gray-600 dark:text-gray-300">
                {{ t('admin.requestDetails.presentation.labels.totalTokens') }}:
                {{ formatNumber(detail.total_tokens || 0) }}
              </div>
            </div>
          </div>

          <div class="grid grid-cols-1 gap-6 xl:grid-cols-2">
            <section class="rounded-3xl border border-gray-200 p-5 dark:border-dark-700">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.requestDetails.drawer.sections.basicInfo') }}
              </h3>
              <div class="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
                <div
                  v-for="field in identityFields"
                  :key="field.label"
                  class="rounded-2xl bg-gray-50 px-4 py-3 dark:bg-dark-800"
                >
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ field.label }}
                  </div>
                  <div class="mt-2 text-sm text-gray-700 dark:text-gray-200" :class="{ 'font-mono text-xs': field.mono }">
                    {{ field.value }}
                  </div>
                </div>
              </div>
            </section>

            <section class="rounded-3xl border border-gray-200 p-5 dark:border-dark-700">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.requestDetails.drawer.sections.identifiers') }}
              </h3>
              <div class="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
                <div
                  v-for="field in identifierFields"
                  :key="field.label"
                  class="rounded-2xl bg-gray-50 px-4 py-3 dark:bg-dark-800"
                >
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ field.label }}
                  </div>
                  <div class="mt-2 text-sm text-gray-700 dark:text-gray-200" :class="{ 'font-mono text-xs': field.mono }">
                    {{ field.value }}
                  </div>
                </div>
              </div>
            </section>

            <section class="rounded-3xl border border-gray-200 p-5 dark:border-dark-700">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.requestDetails.drawer.sections.execution') }}
              </h3>
              <div class="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
                <div
                  v-for="field in executionFields"
                  :key="field.label"
                  class="rounded-2xl bg-gray-50 px-4 py-3 dark:bg-dark-800"
                >
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ field.label }}
                  </div>
                  <div class="mt-2 text-sm text-gray-700 dark:text-gray-200" :class="{ 'font-mono text-xs': field.mono }">
                    {{ field.value }}
                  </div>
                </div>
              </div>
            </section>

            <section class="rounded-3xl border border-gray-200 p-5 dark:border-dark-700">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.requestDetails.drawer.sections.flags') }}
              </h3>
              <div class="mt-4 flex flex-wrap gap-2">
                <span
                  v-for="badge in flagBadges"
                  :key="badge.key"
                  class="badge"
                  :class="badge.className"
                >
                  {{ badge.label }}
                </span>
              </div>
              <div class="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
                <div
                  v-for="field in capabilityFields"
                  :key="field.label"
                  class="rounded-2xl bg-gray-50 px-4 py-3 dark:bg-dark-800"
                >
                  <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                    {{ field.label }}
                  </div>
                  <div class="mt-2 text-sm text-gray-700 dark:text-gray-200">
                    {{ field.value }}
                  </div>
                </div>
              </div>
            </section>
          </div>

          <section class="rounded-3xl border border-gray-200 p-5 dark:border-dark-700">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.requestDetails.drawer.sections.headers') }}
            </h3>
            <div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-2">
              <div>
                <div class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                  {{ t('admin.requestDetails.presentation.labels.requestHeaders') }}
                </div>
                <pre class="max-h-[260px] overflow-auto rounded-2xl bg-gray-50 p-4 text-xs text-gray-800 dark:bg-dark-800 dark:text-gray-200"><code>{{ requestHeaders || '-' }}</code></pre>
              </div>
              <div>
                <div class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                  {{ t('admin.requestDetails.presentation.labels.responseHeaders') }}
                </div>
                <pre class="max-h-[260px] overflow-auto rounded-2xl bg-gray-50 p-4 text-xs text-gray-800 dark:bg-dark-800 dark:text-gray-200"><code>{{ responseHeaders || '-' }}</code></pre>
              </div>
            </div>
          </section>
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
            <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
              <RequestDetailsPayloadPanel
                data-test="raw-request-panel"
                :title="payloadPanels.rawRequest.title"
                :content="payloadPanels.rawRequest.content"
                :loading="rawLoading"
                :empty-message="payloadPanels.rawRequest.emptyMessage"
                :source="payloadPanels.rawRequest.source"
                :truncated="payloadPanels.rawRequest.truncated"
                :state="payloadPanels.rawRequest.state"
                @open-full="handleOpenRawFull('rawRequest')"
              />
              <RequestDetailsPayloadPanel
                data-test="raw-response-panel"
                :title="payloadPanels.rawResponse.title"
                :content="payloadPanels.rawResponse.content"
                :loading="rawLoading"
                :empty-message="payloadPanels.rawResponse.emptyMessage"
                :source="payloadPanels.rawResponse.source"
                :truncated="payloadPanels.rawResponse.truncated"
                :state="payloadPanels.rawResponse.state"
                @open-full="handleOpenRawFull('rawResponse')"
              />
            </div>
          </template>
        </div>

        <RequestDetailsPayloadPanel
          v-else-if="currentPayloadPanel"
          :data-test="`${activeTab}-panel`"
          :title="currentPayloadPanel.title"
          :content="currentPayloadPanel.content"
          :empty-message="currentPayloadPanel.emptyMessage"
          :notice="currentPayloadPanel.notice"
          :can-open-full="currentPayloadPanel.canOpenFull"
          :source="currentPayloadPanel.source"
          :truncated="currentPayloadPanel.truncated"
          :state="currentPayloadPanel.state"
          @open-full="handleOpenPayloadFull(activeTab as PayloadTabKey)"
        />
      </div>
    </aside>
  </transition>

  <RequestDetailsContentDialog
    :show="fullDialog.show"
    :title="fullDialog.title"
    :content="fullDialog.content"
    :notice="fullDialog.notice"
    :loading="fullDialog.loading"
    :empty-message="fullDialog.emptyMessage"
    @close="closeFullDialog"
  />
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
