<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import OpsErrorLogTable from './OpsErrorLogTable.vue'
import { opsAPI, type OpsErrorLog } from '@/api/admin/ops'
import UsageColumnSettingsMenu from '@/components/usage/UsageColumnSettingsMenu.vue'
import type { Column } from '@/components/common/types'
import { COMMON_ERROR_STATUS_CODES } from '@/utils/errorBadges'

interface Props {
  show: boolean
  timeRange: string
  platform?: string
  groupId?: number | null
  channelId?: number | null
  errorType: 'request' | 'upstream'
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'openErrorDetail', errorId: number): void
}>()

const { t } = useI18n()


const loading = ref(false)
const rows = ref<OpsErrorLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const q = ref('')
const statusCode = ref<number | 'other' | null>(null)
const phase = ref<string>('')
const errorOwner = ref<string>('')
const geminiSurface = ref('')
const billingRuleId = ref('')
const probeAction = ref('')
const category = ref('')
const viewMode = ref<'errors' | 'excluded' | 'all'>('errors')
const sortBy = ref('created_at')
const sortOrder = ref<'asc' | 'desc'>('desc')
const hiddenColumns = ref<Set<string>>(new Set(['user_agent']))


const modalTitle = computed(() => {
  return props.errorType === 'upstream' ? t('admin.ops.errorDetails.upstreamErrors') : t('admin.ops.errorDetails.requestErrors')
})

const statusCodeSelectOptions = computed(() => {
  return [
    { value: null, label: t('common.all') },
    ...COMMON_ERROR_STATUS_CODES.map((c) => ({ value: c, label: String(c) })),
    { value: 'other', label: t('admin.ops.errorDetails.statusCodeOther') || 'Other' }
  ]
})

const categorySelectOptions = computed(() => {
  const codes = ['auth', 'rate_limit', 'quota', 'invalid_request', 'service_unavailable', 'upstream', 'internal', 'cyber']
  return [
    { value: '', label: t('usage.errors.allCategories') },
    ...codes.map((code) => ({ value: code, label: t(`usage.errors.categories.${code}`) }))
  ]
})

const ownerSelectOptions = computed(() => {
  return [
    { value: '', label: t('common.all') },
    { value: 'provider', label: t('admin.ops.errorDetails.owner.provider') || 'provider' },
    { value: 'client', label: t('admin.ops.errorDetails.owner.client') || 'client' },
    { value: 'platform', label: t('admin.ops.errorDetails.owner.platform') || 'platform' }
  ]
})


const viewModeSelectOptions = computed(() => {
  return [
    { value: 'errors', label: t('admin.ops.errorDetails.viewErrors') || 'errors' },
    { value: 'excluded', label: t('admin.ops.errorDetails.viewExcluded') || 'excluded' },
    { value: 'all', label: t('common.all') }
  ]
})

const phaseSelectOptions = computed(() => {
  const options = [
    { value: '', label: t('common.all') },
    { value: 'request', label: t('admin.ops.errorDetails.phase.request') || 'request' },
    { value: 'auth', label: t('admin.ops.errorDetails.phase.auth') || 'auth' },
    { value: 'routing', label: t('admin.ops.errorDetails.phase.routing') || 'routing' },
    { value: 'upstream', label: t('admin.ops.errorDetails.phase.upstream') || 'upstream' },
    { value: 'network', label: t('admin.ops.errorDetails.phase.network') || 'network' },
    { value: 'internal', label: t('admin.ops.errorDetails.phase.internal') || 'internal' }
  ]
  return options
})

const tableColumns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.ops.errorLog.time') },
  { key: 'type', label: t('admin.ops.errorLog.type') },
  { key: 'category', label: t('usage.errors.category') },
  { key: 'platform', label: t('admin.ops.errorLog.platform') },
  { key: 'endpoint', label: t('admin.ops.errorLog.endpoint') },
  { key: 'model', label: t('admin.ops.errorLog.model') },
  { key: 'group', label: t('admin.ops.errorLog.group') },
  { key: 'user', label: t('admin.ops.errorLog.user') },
  { key: 'api_key', label: t('admin.ops.errorLog.apiKey') },
  { key: 'status', label: t('admin.ops.errorLog.status') },
  { key: 'message', label: t('admin.ops.errorLog.message') },
  { key: 'client_ip', label: t('admin.ops.errorLog.ip') },
  { key: 'user_agent', label: t('usage.userAgent') }
])

const alwaysVisibleColumns = ['created_at', 'status']

function toggleColumn(key: string) {
  const next = new Set(hiddenColumns.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  hiddenColumns.value = next
}

function handleSort(nextSortBy: string, nextSortOrder: 'asc' | 'desc') {
  sortBy.value = nextSortBy
  sortOrder.value = nextSortOrder
  page.value = 1
  fetchErrorLogs()
}

function close() {
  emit('update:show', false)
}

async function fetchErrorLogs() {
  if (!props.show) return

  loading.value = true
  try {
    const params: Record<string, any> = {
      page: page.value,
      page_size: pageSize.value,
      time_range: props.timeRange,
      view: viewMode.value
    }

    const platform = String(props.platform || '').trim()
    if (platform) params.platform = platform
    if (typeof props.groupId === 'number' && props.groupId > 0) params.group_id = props.groupId

    if (q.value.trim()) params.q = q.value.trim()
    if (statusCode.value === 'other') params.status_codes_other = '1'
    else if (typeof statusCode.value === 'number') params.status_codes = String(statusCode.value)

    const phaseVal = String(phase.value || '').trim()
    if (phaseVal) params.phase = phaseVal

    const ownerVal = String(errorOwner.value || '').trim()
    if (ownerVal) params.error_owner = ownerVal
    if (geminiSurface.value.trim()) params.gemini_surface = geminiSurface.value.trim()
    if (billingRuleId.value.trim()) params.billing_rule_id = billingRuleId.value.trim()
    if (probeAction.value.trim()) params.probe_action = probeAction.value.trim()
    if (category.value.trim()) params.category = category.value.trim()
    params.sort_by = sortBy.value
    params.sort_order = sortOrder.value


    const res = props.errorType === 'upstream'
      ? await opsAPI.listUpstreamErrors(params)
      : await opsAPI.listRequestErrors(params)
    rows.value = res.items || []
    total.value = res.total || 0
  } catch (err) {
    console.error('[OpsErrorDetailsModal] Failed to fetch error logs', err)
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

  function resetFilters() {
    q.value = ''
    statusCode.value = null
    phase.value = props.errorType === 'upstream' ? 'upstream' : ''
    errorOwner.value = ''
    geminiSurface.value = ''
    billingRuleId.value = ''
    probeAction.value = ''
    category.value = ''
    viewMode.value = 'errors'
    sortBy.value = 'created_at'
    sortOrder.value = 'desc'
    page.value = 1
    fetchErrorLogs()
  }


watch(
  () => props.show,
  (open) => {
    if (!open) return
    page.value = 1
    pageSize.value = 10
    resetFilters()
  }
)

watch(
  () => [props.timeRange, props.platform, props.groupId] as const,
  () => {
    if (!props.show) return
    page.value = 1
    fetchErrorLogs()
  }
)

watch(
  () => [page.value, pageSize.value] as const,
  () => {
    if (!props.show) return
    fetchErrorLogs()
  }
)

let searchTimeout: number | null = null
watch(
  () => q.value,
  () => {
    if (!props.show) return
    if (searchTimeout) window.clearTimeout(searchTimeout)
    searchTimeout = window.setTimeout(() => {
      page.value = 1
      fetchErrorLogs()
    }, 350)
  }
)

watch(
  () => [statusCode.value, phase.value, errorOwner.value, geminiSurface.value, billingRuleId.value, probeAction.value, category.value, viewMode.value] as const,
  () => {
    if (!props.show) return
    page.value = 1
    fetchErrorLogs()
  }
)
</script>

<template>
  <BaseDialog :show="show" :title="modalTitle" width="full" @close="close">
    <div class="flex h-full min-h-0 flex-col">
      <div
        v-if="typeof channelId === 'number' && channelId > 0"
        class="mb-4 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
      >
        {{ t('admin.ops.errorDetails.channelFilterNotice') }}
      </div>

      <!-- Filters -->
      <div class="mb-4 flex-shrink-0 border-b border-gray-200 pb-4 dark:border-dark-700">
        <div class="grid grid-cols-1 gap-2 md:grid-cols-3 xl:grid-cols-8">
          <div class="col-span-2 compact-select">
            <div class="relative group">
              <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                <svg
                  class="h-3.5 w-3.5 text-gray-400 transition-colors group-focus-within:text-blue-500"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
              </div>
              <input
                v-model="q"
                type="text"
                class="w-full rounded-lg border-gray-200 bg-gray-50/50 py-1.5 pl-9 pr-3 text-xs font-medium text-gray-700 transition-all focus:border-blue-500 focus:bg-white focus:ring-2 focus:ring-blue-500/10 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-300 dark:focus:bg-dark-800"
                :placeholder="t('admin.ops.errorDetails.searchPlaceholder')"
              />
            </div>
          </div>

          <div class="compact-select">
            <Select :model-value="statusCode" :options="statusCodeSelectOptions" @update:model-value="statusCode = $event as any" />
          </div>

          <div class="compact-select">
            <Select :model-value="phase" :options="phaseSelectOptions" @update:model-value="phase = String($event ?? '')" />
          </div>

          <div class="compact-select">
            <Select :model-value="errorOwner" :options="ownerSelectOptions" @update:model-value="errorOwner = String($event ?? '')" />
          </div>

          <div class="compact-select">
            <Select :model-value="category" :options="categorySelectOptions" @update:model-value="category = String($event ?? '')" />
          </div>

          <div class="compact-select">
            <input
              v-model="geminiSurface"
              type="text"
              class="input"
              :placeholder="t('admin.ops.errorDetails.filters.geminiSurface')"
            />
          </div>

          <div class="compact-select">
            <input
              v-model="billingRuleId"
              type="text"
              class="input"
              :placeholder="t('admin.ops.errorDetails.filters.billingRuleId')"
            />
          </div>

          <div class="compact-select">
            <input
              v-model="probeAction"
              type="text"
              class="input"
              :placeholder="t('admin.ops.errorDetails.filters.probeAction')"
            />
          </div>



          <div class="compact-select">
            <Select :model-value="viewMode" :options="viewModeSelectOptions" @update:model-value="viewMode = $event as any" />
          </div>

          <div class="flex items-center justify-end">
            <div class="flex items-center gap-2">
              <UsageColumnSettingsMenu
                :hidden-columns="hiddenColumns"
                :columns="tableColumns"
                :always-visible-columns="alwaysVisibleColumns"
                @toggle-column="toggleColumn"
              />
              <button type="button" class="rounded-lg bg-gray-100 px-3 py-1.5 text-xs font-semibold text-gray-700 transition-colors hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600" @click="resetFilters">
                {{ t('common.reset') }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Body -->
      <div class="flex min-h-0 flex-1 flex-col">
        <div class="mb-2 flex-shrink-0 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.ops.errorDetails.total') }} {{ total }}
        </div>

          <OpsErrorLogTable
            class="min-h-0 flex-1"
            :rows="rows"
            :total="total"
            :loading="loading"
            :page="page"
            :page-size="pageSize"
            :hidden-columns="hiddenColumns"
            :sort-by="sortBy"
            :sort-order="sortOrder"
            @openErrorDetail="emit('openErrorDetail', $event)"
            @sort="handleSort"
            @update:page="page = $event"
            @update:pageSize="pageSize = $event"
          />

      </div>
    </div>
  </BaseDialog>
</template>

<style>
.compact-select .select-trigger {
  @apply py-1.5 px-3 text-xs rounded-lg;
}
</style>
