<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  opsAPI,
  type OpsRequestTraceDetail,
  type OpsRequestTraceFilter,
  type OpsRequestTraceListItem,
  type OpsRequestTraceRawDetail,
  type OpsRequestTraceSummary
} from '@/api/admin/ops'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores'
import RequestDetailsBreakdownChart from './RequestDetailsBreakdownChart.vue'
import RequestDetailsDrawer from './RequestDetailsDrawer.vue'
import RequestDetailsFilterPanel from './RequestDetailsFilterPanel.vue'
import RequestDetailsSummaryCards from './RequestDetailsSummaryCards.vue'
import RequestDetailsTable from './RequestDetailsTable.vue'
import RequestDetailsTrendChart from './RequestDetailsTrendChart.vue'
import {
  buildCopyableRequestTraceErrorSummary,
  buildRequestTraceQuery,
  createDefaultRequestTraceFilter,
  parseRequestTraceFilterFromQuery
} from '../helpers'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const filters = ref<OpsRequestTraceFilter>(parseRequestTraceFilterFromQuery(route.query))
const items = ref<OpsRequestTraceListItem[]>([])
const total = ref(0)
const summary = ref<OpsRequestTraceSummary | null>(null)
const detail = ref<OpsRequestTraceDetail | null>(null)
const rawDetail = ref<OpsRequestTraceRawDetail | null>(null)
const selectedId = ref<number | null>(null)
const errorMessage = ref('')

const loadingList = ref(false)
const loadingSummary = ref(false)
const loadingDetail = ref(false)
const loadingRaw = ref(false)
const refreshing = ref(false)

let listController: AbortController | null = null
let summaryController: AbortController | null = null
let detailController: AbortController | null = null
let rawController: AbortController | null = null
let syncingRoute = false
let applyingRoute = false
let syncRouteTimer: ReturnType<typeof setTimeout> | null = null
let refreshTimer: ReturnType<typeof setTimeout> | null = null

const drawerOpen = computed(() => selectedId.value != null)
const rawExportAllowed = computed(() => summary.value?.raw_access_allowed ?? false)

function closeDrawer() {
  selectedId.value = null
  detail.value = null
  rawDetail.value = null
}

function isCanceled(error: unknown): boolean {
  return !!error && typeof error === 'object' && 'code' in error && (error as { code?: string }).code === 'ERR_CANCELED'
}

function requestFilter(): OpsRequestTraceFilter {
  return { ...filters.value }
}

function summaryFilter(): OpsRequestTraceFilter {
  const next = { ...filters.value }
  delete next.page
  delete next.page_size
  return next
}

function buildTraceRouteQuery(): Record<string, string> {
  return {
    ...buildRequestTraceQuery(filters.value),
    tab: 'trace',
  }
}

async function fetchList() {
  listController?.abort()
  listController = new AbortController()
  loadingList.value = true
  errorMessage.value = ''
  try {
    const response = await opsAPI.listRequestTraces(requestFilter(), { signal: listController.signal })
    items.value = response.items || []
    total.value = response.total || 0
  } catch (error: any) {
    if (isCanceled(error)) return
    errorMessage.value = error?.message || t('admin.requestDetails.messages.listFailed')
  } finally {
    loadingList.value = false
  }
}

async function fetchSummary() {
  summaryController?.abort()
  summaryController = new AbortController()
  loadingSummary.value = true
  try {
    summary.value = await opsAPI.getRequestTraceSummary(summaryFilter(), { signal: summaryController.signal })
  } catch (error: any) {
    if (isCanceled(error)) return
    appStore.showError(error?.message || t('admin.requestDetails.messages.summaryFailed'))
  } finally {
    loadingSummary.value = false
  }
}

async function fetchDetail(id: number) {
  detailController?.abort()
  detailController = new AbortController()
  loadingDetail.value = true
  rawDetail.value = null
  try {
    detail.value = await opsAPI.getRequestTraceDetail(id, { signal: detailController.signal })
  } catch (error: any) {
    if (isCanceled(error)) return
    appStore.showError(error?.message || t('admin.requestDetails.messages.detailFailed'))
  } finally {
    loadingDetail.value = false
  }
}

async function fetchRawDetail() {
  if (!selectedId.value) return
  rawController?.abort()
  rawController = new AbortController()
  loadingRaw.value = true
  try {
    rawDetail.value = await opsAPI.getRequestTraceRawDetail(selectedId.value, { signal: rawController.signal })
  } catch (error: any) {
    if (isCanceled(error)) return
    appStore.showError(error?.message || t('admin.requestDetails.messages.rawFailed'))
  } finally {
    loadingRaw.value = false
  }
}

async function fetchAllNow() {
  await Promise.all([fetchList(), fetchSummary()])
}

async function handleManualRefresh() {
  if (refreshing.value) return
  refreshing.value = true
  try {
    const selectedTraceID = selectedId.value
    const shouldRefreshRaw = Boolean(selectedTraceID && rawDetail.value)
    await fetchAllNow()
    if (selectedTraceID) {
      await fetchDetail(selectedTraceID)
      if (shouldRefreshRaw) {
        await fetchRawDetail()
      }
    }
  } finally {
    refreshing.value = false
  }
}

function syncRouteQuery() {
  if (syncRouteTimer) {
    clearTimeout(syncRouteTimer)
  }
  syncRouteTimer = setTimeout(async () => {
    syncingRoute = true
    try {
      await router.replace({ query: buildTraceRouteQuery() })
    } finally {
      syncingRoute = false
    }
  }, 200)
}

function refreshData() {
  if (refreshTimer) {
    clearTimeout(refreshTimer)
  }
  refreshTimer = setTimeout(() => {
    void fetchAllNow()
  }, 250)
}

watch(
  () => route.query,
  (query) => {
    if (syncingRoute) return
    applyingRoute = true
    filters.value = parseRequestTraceFilterFromQuery(query)
    void fetchAllNow()
    applyingRoute = false
  },
  { immediate: true }
)

watch(
  filters,
  () => {
    if (applyingRoute) return
    syncRouteQuery()
    refreshData()
  },
  { deep: true }
)

watch(selectedId, (id) => {
  if (!id) return
  void fetchDetail(id)
})

function handleApply() {
  void fetchAllNow()
}

function handleReset() {
  filters.value = createDefaultRequestTraceFilter()
  closeDrawer()
}

function handleSelect(item: OpsRequestTraceListItem) {
  selectedId.value = item.id
}

async function handleCopyError(item: OpsRequestTraceListItem) {
  try {
    const resolvedDetail = detail.value?.id === item.id
      ? detail.value
      : await opsAPI.getRequestTraceDetail(item.id)
    await copyToClipboard(buildCopyableRequestTraceErrorSummary(resolvedDetail))
  } catch (error: any) {
    appStore.showError(error?.message || t('common.copyFailed'))
  }
}

function handlePage(page: number) {
  filters.value = { ...filters.value, page }
}

function handlePageSize(pageSize: number) {
  filters.value = { ...filters.value, page: 1, page_size: pageSize }
}

async function handleExport(includeRaw: boolean) {
  try {
    const { blob, filename } = await opsAPI.exportRequestTracesCSV(summaryFilter(), includeRaw)
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = filename
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
    URL.revokeObjectURL(url)
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.requestDetails.messages.exportFailed'))
  }
}

onUnmounted(() => {
  listController?.abort()
  summaryController?.abort()
  detailController?.abort()
  rawController?.abort()
  if (syncRouteTimer) clearTimeout(syncRouteTimer)
  if (refreshTimer) clearTimeout(refreshTimer)
})
</script>

<template>
  <div class="space-y-6 pb-12">
    <div v-if="errorMessage" class="rounded-2xl bg-red-50 px-4 py-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-300">
      {{ errorMessage }}
    </div>

    <RequestDetailsFilterPanel
      v-model="filters"
      :loading="loadingList || loadingSummary"
      :raw-export-allowed="rawExportAllowed"
      @apply="handleApply"
      @reset="handleReset"
      @export="handleExport"
    />

    <RequestDetailsSummaryCards :summary="summary" :loading="loadingSummary" />

    <div class="grid grid-cols-1 gap-6 xl:grid-cols-3">
      <div class="xl:col-span-2">
        <RequestDetailsTrendChart
          :points="summary?.trend ?? []"
          :loading="loadingSummary"
          :time-range="filters.time_range || '1h'"
        />
      </div>
      <div class="space-y-6">
        <RequestDetailsBreakdownChart
          :title="t('admin.requestDetails.charts.statusTitle')"
          :description="t('admin.requestDetails.charts.statusDescription')"
          :items="summary?.status_distribution ?? []"
          :total="summary?.totals.request_count ?? 0"
          :loading="loadingSummary"
        />
        <RequestDetailsBreakdownChart
          :title="t('admin.requestDetails.charts.protocolTitle')"
          :description="t('admin.requestDetails.charts.protocolDescription')"
          :items="summary?.protocol_pair_distribution ?? []"
          :total="summary?.totals.request_count ?? 0"
          :loading="loadingSummary"
        />
      </div>
    </div>

    <div class="grid grid-cols-1 gap-6 xl:grid-cols-3">
      <RequestDetailsBreakdownChart
        :title="t('admin.requestDetails.charts.finishReasonTitle')"
        :description="t('admin.requestDetails.charts.finishReasonDescription')"
        :items="summary?.finish_reason_distribution ?? []"
        :total="summary?.totals.request_count ?? 0"
        :loading="loadingSummary"
      />
      <RequestDetailsBreakdownChart
        :title="t('admin.requestDetails.charts.modelTitle')"
        :description="t('admin.requestDetails.charts.modelDescription')"
        :items="summary?.model_distribution ?? []"
        :total="summary?.totals.request_count ?? 0"
        :loading="loadingSummary"
      />
      <RequestDetailsBreakdownChart
        :title="t('admin.requestDetails.charts.capabilityTitle')"
        :description="t('admin.requestDetails.charts.capabilityDescription')"
        :items="summary?.capability_distribution ?? []"
        :total="summary?.totals.request_count ?? 0"
        :loading="loadingSummary"
      />
    </div>

    <RequestDetailsTable
      :items="items"
      :total="total"
      :page="filters.page || 1"
      :page-size="filters.page_size || 20"
      :loading="loadingList"
      :refreshing="refreshing"
      :selected-id="selectedId"
      @refresh="handleManualRefresh"
      @select="handleSelect"
      @copy-error="handleCopyError"
      @update:page="handlePage"
      @update:page-size="handlePageSize"
    />

    <RequestDetailsDrawer
      :open="drawerOpen"
      :detail="detail"
      :raw-detail="rawDetail"
      :loading="loadingDetail"
      :raw-loading="loadingRaw"
      @close="closeDrawer"
      @load-raw="fetchRawDetail"
    />
  </div>
</template>
