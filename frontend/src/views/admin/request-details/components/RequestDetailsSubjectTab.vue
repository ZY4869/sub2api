<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { adminAPI } from '@/api/admin'
import { adminUsageAPI, type AdminUsageQueryParams } from '@/api/admin/usage'
import {
  opsAPI,
  type OpsRequestSubjectInsights,
  type OpsRequestSubjectInsightsParams,
  type OpsRequestSubjectType,
} from '@/api/admin/ops'
import { useTokenDisplayMode } from '@/composables/useTokenDisplayMode'
import type { AdminUsageLog } from '@/types'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import EndpointDistributionChart from '@/components/charts/EndpointDistributionChart.vue'
import RequestDetailsSubjectTrendChart from './RequestDetailsSubjectTrendChart.vue'
import RequestDetailsSubjectUsageTable from './RequestDetailsSubjectUsageTable.vue'

type SubjectTimeRange = '5m' | '30m' | '1h' | '6h' | '24h' | '7d' | '30d' | 'custom'

interface SubjectFilters {
  subject_type: OpsRequestSubjectType
  subject_id: string
  time_range: SubjectTimeRange
  start_time: string
  end_time: string
}

const { t } = useI18n()
const route = useRoute()
const { formatTokenDisplay } = useTokenDisplayMode()

const insights = ref<OpsRequestSubjectInsights | null>(null)
const usageRows = ref<AdminUsageLog[]>([])
const usageTotal = ref(0)
const usagePage = ref(1)
const usagePageSize = ref(20)
const loadingInsights = ref(false)
const loadingUsage = ref(false)
const queryError = ref('')
const subjectSearchRef = ref<HTMLElement | null>(null)
const subjectKeyword = ref('')
const subjectOptions = ref<Array<{ id: number; label: string }>>([])
const loadingSubjectOptions = ref(false)
const showSubjectDropdown = ref(false)
const cachedGroups = ref<Array<{ id: number; name: string }>>([])

const filters = ref<SubjectFilters>(createInitialFilters())

const timeRangeOptions = [
  { value: '5m', label: t('admin.requestDetails.filters.timeRangeOptions.5m') },
  { value: '30m', label: t('admin.requestDetails.filters.timeRangeOptions.30m') },
  { value: '1h', label: t('admin.requestDetails.filters.timeRangeOptions.1h') },
  { value: '6h', label: t('admin.requestDetails.filters.timeRangeOptions.6h') },
  { value: '24h', label: t('admin.requestDetails.filters.timeRangeOptions.24h') },
  { value: '7d', label: t('admin.requestDetails.filters.timeRangeOptions.7d') },
  { value: '30d', label: t('admin.requestDetails.filters.timeRangeOptions.30d') },
  { value: 'custom', label: t('admin.requestDetails.subject.filters.customRange') },
]

const canQuery = computed(() => Number.parseInt(filters.value.subject_id, 10) > 0)
const subjectPlaceholder = computed(() => {
  if (filters.value.subject_type === 'group') {
    return t('admin.groups.searchGroups')
  }
  if (filters.value.subject_type === 'api_key') {
    return t('admin.usage.searchApiKeyPlaceholder')
  }
  return t('admin.usage.searchAccountPlaceholder')
})
const subjectEmptyMessage = computed(() =>
  loadingSubjectOptions.value ? t('common.loading') : t('common.noData')
)

const summaryCards = computed(() => {
  if (!insights.value) return []
  return [
    { key: 'account', label: t('admin.requestDetails.subject.summary.totalAccountCost'), value: `$${formatCurrency(insights.value.summary.total_account_cost)}` },
    { key: 'user', label: t('admin.requestDetails.subject.summary.totalUserCost'), value: `$${formatCurrency(insights.value.summary.total_user_cost)}` },
    { key: 'standard', label: t('admin.requestDetails.subject.summary.totalStandardCost'), value: `$${formatCurrency(insights.value.summary.total_standard_cost)}` },
    { key: 'requests', label: t('admin.requestDetails.subject.summary.totalRequests'), value: formatNumber(insights.value.summary.total_requests) },
    { key: 'tokens', label: t('admin.requestDetails.subject.summary.totalTokens'), value: formatTokenDisplay(insights.value.summary.total_tokens) },
    { key: 'duration', label: t('admin.requestDetails.subject.summary.avgDurationMs'), value: `${Math.round(insights.value.summary.avg_duration_ms || 0)} ms` },
  ]
})

const subjectMeta = computed(() => {
  if (!insights.value) return []
  const subject = insights.value.subject
  return [
    `${t('admin.requestDetails.subject.subjectType')}: ${subject.type}`,
    `${t('admin.requestDetails.subject.subjectId')}: ${subject.id}`,
    subject.user_email ? `${t('admin.requestDetails.subject.userEmail')}: ${subject.user_email}` : '',
    subject.group_name ? `${t('admin.requestDetails.subject.groupName')}: ${subject.group_name}` : '',
  ].filter(Boolean)
})

function createInitialFilters(): SubjectFilters {
  if (route.query.group_id) {
    return { subject_type: 'group', subject_id: String(route.query.group_id), time_range: '30d', start_time: '', end_time: '' }
  }
  if (route.query.api_key_id) {
    return { subject_type: 'api_key', subject_id: String(route.query.api_key_id), time_range: '30d', start_time: '', end_time: '' }
  }
  return {
    subject_type: 'account',
    subject_id: route.query.account_id ? String(route.query.account_id) : '',
    time_range: '30d',
    start_time: '',
    end_time: '',
  }
}

function formatNumber(value: number): string {
  return value.toLocaleString()
}

function formatCurrency(value: number): string {
  return Number(value || 0).toFixed(4)
}

function formatLocalDate(value: Date): string {
  const year = value.getFullYear()
  const month = String(value.getMonth() + 1).padStart(2, '0')
  const day = String(value.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function toRFC3339(value?: string): string | undefined {
  const trimmed = String(value || '').trim()
  if (!trimmed) return undefined
  const parsed = new Date(trimmed)
  if (Number.isNaN(parsed.getTime())) return undefined
  return parsed.toISOString()
}

function shiftRange(range: SubjectTimeRange): { start_time?: string; end_time?: string } {
  if (range === 'custom') {
    return {
      start_time: toRFC3339(filters.value.start_time),
      end_time: toRFC3339(filters.value.end_time),
    }
  }
  return {}
}

function resolveUsageDateRange(): { start_date?: string; end_date?: string } {
  if (filters.value.start_time || filters.value.end_time) {
    return {
      start_date: filters.value.start_time ? filters.value.start_time.slice(0, 10) : undefined,
      end_date: filters.value.end_time ? filters.value.end_time.slice(0, 10) : undefined,
    }
  }

  const end = new Date()
  const start = new Date()
  switch (filters.value.time_range) {
    case '5m':
      start.setMinutes(end.getMinutes() - 5)
      break
    case '30m':
      start.setMinutes(end.getMinutes() - 30)
      break
    case '1h':
      start.setHours(end.getHours() - 1)
      break
    case '6h':
      start.setHours(end.getHours() - 6)
      break
    case '24h':
      start.setDate(end.getDate() - 1)
      break
    case '7d':
      start.setDate(end.getDate() - 7)
      break
    default:
      start.setDate(end.getDate() - 30)
      break
  }

  return {
    start_date: formatLocalDate(start),
    end_date: formatLocalDate(end),
  }
}

async function loadInsights() {
  if (!canQuery.value) {
    insights.value = null
    return
  }

  loadingInsights.value = true
  queryError.value = ''
  try {
    const params: OpsRequestSubjectInsightsParams = {
      subject_type: filters.value.subject_type,
      subject_id: Number(filters.value.subject_id),
      ...(filters.value.time_range !== 'custom' ? { time_range: filters.value.time_range } : {}),
      ...shiftRange(filters.value.time_range),
    }
    insights.value = await opsAPI.getSubjectInsights(params)
  } catch (error: any) {
    queryError.value = error?.message || t('admin.requestDetails.subject.messages.loadFailed')
    insights.value = null
  } finally {
    loadingInsights.value = false
  }
}

async function loadUsage() {
  if (!canQuery.value) {
    usageRows.value = []
    usageTotal.value = 0
    return
  }

  loadingUsage.value = true
  try {
    const params: AdminUsageQueryParams = {
      page: usagePage.value,
      page_size: usagePageSize.value,
      include_preview_availability: true,
      exact_total: true,
      ...resolveUsageDateRange(),
    }
    const subjectID = Number(filters.value.subject_id)
    if (filters.value.subject_type === 'account') params.account_id = subjectID
    if (filters.value.subject_type === 'group') params.group_id = subjectID
    if (filters.value.subject_type === 'api_key') params.api_key_id = subjectID
    const response = await adminUsageAPI.list(params)
    usageRows.value = response.items || []
    usageTotal.value = response.total || 0
  } catch (error: any) {
    queryError.value = error?.message || t('admin.requestDetails.subject.messages.loadFailed')
    usageRows.value = []
    usageTotal.value = 0
  } finally {
    loadingUsage.value = false
  }
}

async function applyFilters() {
  usagePage.value = 1
  await Promise.all([loadInsights(), loadUsage()])
}

function resetFilters() {
  filters.value = createInitialFilters()
  insights.value = null
  usageRows.value = []
  usageTotal.value = 0
  queryError.value = ''
  usagePage.value = 1
  resetSubjectSelector()
  void hydrateSubjectLabel()
}

async function handlePage(page: number) {
  usagePage.value = page
  await loadUsage()
}

async function handlePageSize(pageSize: number) {
  usagePageSize.value = pageSize
  usagePage.value = 1
  await loadUsage()
}

function formatSubjectOptionLabel(option: { id: number; label: string }) {
  return `${option.label} · #${option.id}`
}

function syncSubjectIDFromKeyword() {
  const trimmed = subjectKeyword.value.trim()
  filters.value.subject_id = /^\d+$/.test(trimmed) ? trimmed : ''
}

function resetSubjectSelector() {
  subjectKeyword.value = ''
  subjectOptions.value = []
  showSubjectDropdown.value = false
}

async function ensureGroupsLoaded() {
  if (cachedGroups.value.length > 0) {
    return
  }
  const response = await adminAPI.groups.list(1, 1000)
  cachedGroups.value = (response.items || []).map((group) => ({
    id: Number(group.id),
    name: String(group.name || '').trim() || `#${group.id}`
  }))
}

async function loadSubjectOptions(keyword = '') {
  loadingSubjectOptions.value = true
  try {
    const trimmed = keyword.trim()
    if (filters.value.subject_type === 'account') {
      const response = await adminAPI.accounts.list(1, 20, { search: trimmed })
      subjectOptions.value = (response.items || []).map((account) => ({
        id: Number(account.id),
        label: String(account.name || '').trim() || `#${account.id}`
      }))
      return
    }

    if (filters.value.subject_type === 'group') {
      await ensureGroupsLoaded()
      const normalizedKeyword = trimmed.toLowerCase()
      subjectOptions.value = cachedGroups.value
        .filter((group) => {
          if (!normalizedKeyword) {
            return true
          }
          return (
            group.name.toLowerCase().includes(normalizedKeyword) ||
            String(group.id).includes(normalizedKeyword)
          )
        })
        .slice(0, 30)
        .map((group) => ({
          id: group.id,
          label: group.name
        }))
      return
    }

    const results = await adminAPI.usage.searchApiKeys(undefined, trimmed)
    subjectOptions.value = results.map((item) => ({
      id: Number(item.id),
      label: String(item.name || '').trim() || `#${item.id}`
    }))
  } catch {
    subjectOptions.value = []
  } finally {
    loadingSubjectOptions.value = false
  }
}

async function hydrateSubjectLabel() {
  const subjectID = Number(filters.value.subject_id)
  if (!Number.isFinite(subjectID) || subjectID <= 0) {
    subjectKeyword.value = ''
    return
  }

  try {
    if (filters.value.subject_type === 'account') {
      const account = await adminAPI.accounts.getById(subjectID)
      subjectKeyword.value = formatSubjectOptionLabel({
        id: subjectID,
        label: String(account.name || '').trim() || `#${subjectID}`
      })
      return
    }
    if (filters.value.subject_type === 'group') {
      const group = await adminAPI.groups.getById(subjectID)
      subjectKeyword.value = formatSubjectOptionLabel({
        id: subjectID,
        label: String(group.name || '').trim() || `#${subjectID}`
      })
      return
    }
  } catch {
    // Fall back to raw IDs when the referenced subject cannot be resolved.
  }

  subjectKeyword.value = `#${subjectID}`
}

async function handleSubjectFocus() {
  showSubjectDropdown.value = true
  await loadSubjectOptions(subjectKeyword.value)
}

async function handleSubjectInput() {
  showSubjectDropdown.value = true
  syncSubjectIDFromKeyword()
  await loadSubjectOptions(subjectKeyword.value)
}

async function selectSubjectOption(option: { id: number; label: string }) {
  filters.value.subject_id = String(option.id)
  subjectKeyword.value = formatSubjectOptionLabel(option)
  showSubjectDropdown.value = false
  await applyFilters()
}

function clearSubjectSelection() {
  filters.value.subject_id = ''
  subjectKeyword.value = ''
  subjectOptions.value = []
  showSubjectDropdown.value = false
}

function handleDocumentClick(event: MouseEvent) {
  const target = event.target as Node | null
  if (!target) {
    return
  }
  if (!(subjectSearchRef.value?.contains(target) ?? false)) {
    showSubjectDropdown.value = false
  }
}

watch(
  () => filters.value.subject_type,
  () => {
    filters.value.subject_id = ''
    subjectKeyword.value = ''
    subjectOptions.value = []
    showSubjectDropdown.value = false
  }
)

onMounted(() => {
  document.addEventListener('click', handleDocumentClick)
  void hydrateSubjectLabel()
  if (canQuery.value) {
    void applyFilters()
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick)
})
</script>

<template>
  <div class="space-y-6 pb-12">
    <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="flex flex-wrap items-end gap-4">
        <div class="min-w-[180px] flex-1">
          <div class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ t('admin.requestDetails.subject.filters.subjectType') }}
          </div>
          <select v-model="filters.subject_type" class="input w-full">
            <option value="account">{{ t('admin.requestDetails.subject.filters.account') }}</option>
            <option value="group">{{ t('admin.requestDetails.subject.filters.group') }}</option>
            <option value="api_key">{{ t('admin.requestDetails.subject.filters.apiKey') }}</option>
          </select>
        </div>
        <div ref="subjectSearchRef" class="relative min-w-[220px] flex-1">
          <div class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ t('admin.requestDetails.subject.filters.subjectId') }}
          </div>
          <input
            v-model.trim="subjectKeyword"
            class="input w-full pr-10"
            type="text"
            :placeholder="subjectPlaceholder"
            @focus="handleSubjectFocus"
            @input="handleSubjectInput"
          />
          <button
            v-if="subjectKeyword || filters.subject_id"
            type="button"
            class="absolute right-3 top-[2.55rem] text-sm text-gray-400 transition hover:text-gray-600 dark:hover:text-gray-200"
            @click="clearSubjectSelection"
          >
            ×
          </button>
          <div
            v-if="showSubjectDropdown && (loadingSubjectOptions || subjectOptions.length > 0 || subjectKeyword)"
            class="absolute z-40 mt-2 max-h-64 w-full overflow-auto rounded-2xl border border-gray-200 bg-white p-2 shadow-xl dark:border-dark-600 dark:bg-dark-800"
          >
            <div
              v-if="loadingSubjectOptions"
              class="px-3 py-2 text-sm text-gray-500 dark:text-gray-400"
            >
              {{ subjectEmptyMessage }}
            </div>
            <button
              v-for="option in subjectOptions"
              :key="`${filters.subject_type}-${option.id}`"
              type="button"
              class="flex w-full items-center justify-between gap-3 rounded-xl px-3 py-2 text-left text-sm transition hover:bg-gray-100 dark:hover:bg-dark-700"
              @click="selectSubjectOption(option)"
            >
              <span class="truncate">{{ option.label }}</span>
              <span class="shrink-0 text-xs text-gray-400">#{{ option.id }}</span>
            </button>
            <div
              v-if="!loadingSubjectOptions && subjectOptions.length === 0"
              class="px-3 py-2 text-sm text-gray-500 dark:text-gray-400"
            >
              {{ subjectEmptyMessage }}
            </div>
          </div>
        </div>
        <div class="min-w-[180px] flex-1">
          <div class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ t('admin.requestDetails.subject.filters.timeRange') }}
          </div>
          <select v-model="filters.time_range" class="input w-full">
            <option v-for="option in timeRangeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </div>
        <div class="min-w-[220px] flex-1">
          <div class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ t('admin.requestDetails.filters.startTime') }}
          </div>
          <input v-model="filters.start_time" class="input w-full" type="datetime-local" @input="filters.time_range = 'custom'" />
        </div>
        <div class="min-w-[220px] flex-1">
          <div class="mb-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ t('admin.requestDetails.filters.endTime') }}
          </div>
          <input v-model="filters.end_time" class="input w-full" type="datetime-local" @input="filters.time_range = 'custom'" />
        </div>
        <button class="btn btn-primary" type="button" :disabled="loadingInsights || loadingUsage || !canQuery" @click="applyFilters">
          {{ t('common.search') }}
        </button>
        <button class="btn btn-secondary" type="button" @click="resetFilters">
          {{ t('common.reset') }}
        </button>
      </div>
    </section>

    <div v-if="queryError" class="rounded-2xl bg-red-50 px-4 py-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-300">
      {{ queryError }}
    </div>

    <section
      v-if="!canQuery && !loadingInsights"
      class="flex min-h-[220px] items-center justify-center rounded-3xl border border-dashed border-gray-200 bg-white px-6 text-sm text-gray-500 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400"
    >
      {{ t('admin.requestDetails.subject.emptyPrompt') }}
    </section>

    <template v-else>
      <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div v-if="loadingInsights" class="flex h-28 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('common.loading') }}
        </div>
        <template v-else-if="insights">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div>
              <div class="text-xs font-medium uppercase tracking-[0.2em] text-primary-500">
                {{ t('admin.requestDetails.subject.headerEyebrow') }}
              </div>
              <h2 class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">
                {{ insights.subject.name || `${insights.subject.type} #${insights.subject.id}` }}
              </h2>
              <div class="mt-3 flex flex-wrap gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span v-for="item in subjectMeta" :key="item" class="rounded-full bg-gray-100 px-3 py-1 dark:bg-dark-700">
                  {{ item }}
                </span>
              </div>
            </div>
            <div class="rounded-2xl bg-gray-50 px-4 py-3 text-sm text-gray-600 dark:bg-dark-700 dark:text-gray-300">
              {{ t('admin.requestDetails.subject.summary.activeDays') }}:
              <span class="font-semibold text-gray-900 dark:text-white">{{ insights.summary.active_days }}</span>
              /
              {{ insights.summary.window_days }}
            </div>
          </div>
        </template>
      </section>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div v-for="card in summaryCards" :key="card.key" class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ card.label }}
          </div>
          <div class="mt-3 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ card.value }}
          </div>
        </div>
      </div>

      <div v-if="insights" class="grid grid-cols-1 gap-6 xl:grid-cols-3">
        <div class="xl:col-span-2">
          <RequestDetailsSubjectTrendChart :history="insights.history" :loading="loadingInsights" />
        </div>
        <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('admin.requestDetails.subject.previewCoverage.title') }}
          </h3>
          <div class="mt-4 space-y-3 text-sm text-gray-600 dark:text-gray-300">
            <div class="flex items-center justify-between">
              <span>{{ t('admin.requestDetails.subject.previewCoverage.availableRate') }}</span>
              <span class="font-semibold text-gray-900 dark:text-white">
                {{ (insights.request_preview_coverage.preview_available_rate * 100).toFixed(1) }}%
              </span>
            </div>
            <div class="flex items-center justify-between">
              <span>{{ t('admin.requestDetails.subject.previewCoverage.previewAvailableCount') }}</span>
              <span>{{ formatNumber(insights.request_preview_coverage.preview_available_count) }}</span>
            </div>
            <div class="flex items-center justify-between">
              <span>{{ t('admin.requestDetails.subject.previewCoverage.normalizedCount') }}</span>
              <span>{{ formatNumber(insights.request_preview_coverage.normalized_count) }}</span>
            </div>
            <div class="flex items-center justify-between">
              <span>{{ t('admin.requestDetails.subject.previewCoverage.upstreamRequestCount') }}</span>
              <span>{{ formatNumber(insights.request_preview_coverage.upstream_request_count) }}</span>
            </div>
            <div class="flex items-center justify-between">
              <span>{{ t('admin.requestDetails.subject.previewCoverage.upstreamResponseCount') }}</span>
              <span>{{ formatNumber(insights.request_preview_coverage.upstream_response_count) }}</span>
            </div>
            <div class="flex items-center justify-between">
              <span>{{ t('admin.requestDetails.subject.previewCoverage.gatewayResponseCount') }}</span>
              <span>{{ formatNumber(insights.request_preview_coverage.gateway_response_count) }}</span>
            </div>
          </div>
        </section>
      </div>

      <div v-if="insights" class="grid grid-cols-1 gap-6 xl:grid-cols-3">
        <ModelDistributionChart :model-stats="insights.models" :loading="loadingInsights" />
        <EndpointDistributionChart
          :endpoint-stats="insights.endpoints"
          :loading="loadingInsights"
          :title="t('usage.inboundEndpoint')"
        />
        <EndpointDistributionChart
          :endpoint-stats="insights.upstream_endpoints"
          :loading="loadingInsights"
          :title="t('usage.upstreamEndpoint')"
        />
      </div>

      <RequestDetailsSubjectUsageTable
        :items="usageRows"
        :total="usageTotal"
        :page="usagePage"
        :page-size="usagePageSize"
        :loading="loadingUsage"
        @update:page="handlePage"
        @update:page-size="handlePageSize"
      />
    </template>
  </div>
</template>
