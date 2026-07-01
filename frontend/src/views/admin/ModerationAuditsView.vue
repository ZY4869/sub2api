<template>
  <AppLayout>
    <div class="space-y-6">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ t('admin.moderation.title') }}
        </h1>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.moderation.description') }}
        </p>
      </div>

      <div class="card p-4">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
          <input
            v-model="filters.request_id"
            type="text"
            class="input"
            :placeholder="t('admin.moderation.filters.requestId')"
            @keyup.enter="applyFilters"
          />
          <input
            v-model="filters.client_request_id"
            type="text"
            class="input"
            :placeholder="t('admin.moderation.filters.clientRequestId')"
            @keyup.enter="applyFilters"
          />
          <input
            v-model="filters.provider"
            type="text"
            class="input"
            :placeholder="t('admin.moderation.filters.provider')"
            @keyup.enter="applyFilters"
          />
          <input
            v-model="filters.model"
            type="text"
            class="input"
            :placeholder="t('admin.moderation.filters.model')"
            @keyup.enter="applyFilters"
          />
          <input
            v-model="filters.source_endpoint"
            type="text"
            class="input"
            :placeholder="t('admin.moderation.filters.sourceEndpoint')"
            @keyup.enter="applyFilters"
          />
          <input
            v-model="filters.content_hash"
            type="text"
            class="input font-mono text-sm"
            :placeholder="t('admin.moderation.filters.contentHash')"
            @keyup.enter="applyFilters"
          />
          <input
            v-model.number="filters.user_id"
            type="number"
            min="1"
            class="input"
            :placeholder="t('admin.moderation.filters.userId')"
            @keyup.enter="applyFilters"
          />
          <Select v-model="hitFilter" :options="hitOptions" @change="applyFilters" />
        </div>
        <div class="mt-3 flex flex-wrap gap-2">
          <button class="btn btn-primary" @click="applyFilters">
            {{ t('common.search') }}
          </button>
          <button class="btn btn-secondary" @click="resetFilters">
            {{ t('common.reset') }}
          </button>
          <button class="btn btn-secondary" :disabled="loading" @click="loadAudits">
            {{ t('common.refresh') }}
          </button>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div v-if="loading" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('common.loading') }}
        </div>
        <div v-else-if="audits.length === 0" class="p-8 text-center text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.moderation.empty') }}
        </div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800/60">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.moderation.columns.createdAt') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.moderation.columns.provider') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.moderation.columns.sourceEndpoint') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.moderation.columns.summary') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.moderation.columns.matchedKeyword') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.moderation.columns.request') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.moderation.columns.status') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.moderation.columns.latency') }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr
                v-for="audit in audits"
                :key="audit.id"
                class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-800/40"
                @click="openDetail(audit.id)"
              >
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
                  {{ formatDateTime(audit.created_at) }}
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
                  <div class="font-medium">{{ audit.provider || '-' }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ audit.model || '-' }}</div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
                  <span class="rounded bg-gray-100 px-2 py-1 font-mono text-xs dark:bg-dark-800">
                    {{ audit.source_endpoint }}
                  </span>
                </td>
                <td class="max-w-xl px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
                  {{ audit.content_summary || '-' }}
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
                  <span class="rounded bg-gray-100 px-2 py-1 font-mono text-xs dark:bg-dark-800">
                    {{ audit.matched_keyword || '-' }}
                  </span>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
                  <div class="font-mono text-xs">{{ audit.request_id || '-' }}</div>
                  <div class="font-mono text-xs text-gray-500 dark:text-gray-400">{{ audit.client_request_id || '-' }}</div>
                </td>
                <td class="px-4 py-3 text-sm">
                  <div class="flex flex-wrap gap-2">
                    <span
                      class="rounded-full px-2 py-1 text-xs font-medium"
                      :class="audit.hit ? 'bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300' : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'"
                    >
                      {{ audit.hit ? t('admin.moderation.status.hit') : t('admin.moderation.status.pass') }}
                    </span>
                    <span
                      v-if="audit.dedupe_hit"
                      class="rounded-full bg-amber-100 px-2 py-1 text-xs font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300"
                    >
                      {{ t('admin.moderation.status.dedupe') }}
                    </span>
                    <span
                      v-for="category in audit.categories || []"
                      :key="category"
                      class="rounded-full bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300"
                    >
                      {{ category }}
                    </span>
                    <span
                      v-if="audit.error_reason"
                      class="rounded-full bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-800 dark:text-gray-300"
                    >
                      {{ t('admin.moderation.status.error') }}
                    </span>
                  </div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-200">
                  {{ audit.latency_ms }} ms
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="pagination.total > 0" class="border-t border-gray-200 p-4 dark:border-dark-700">
          <Pagination
            :page="pagination.page"
            :total="pagination.total"
            :page-size="pagination.page_size"
            @update:page="handlePageChange"
            @update:pageSize="handlePageSizeChange"
          />
        </div>
      </div>

      <div v-if="selectedAudit" class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('admin.moderation.detail.title') }} #{{ selectedAudit.id }}
          </h2>
        </div>
        <div class="grid grid-cols-1 gap-4 p-6 md:grid-cols-2">
          <div>
            <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.moderation.detail.requestId') }}</div>
            <div class="mt-1 font-mono text-sm text-gray-800 dark:text-gray-100">{{ selectedAudit.request_id || '-' }}</div>
          </div>
          <div>
            <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.moderation.detail.clientRequestId') }}</div>
            <div class="mt-1 font-mono text-sm text-gray-800 dark:text-gray-100">{{ selectedAudit.client_request_id || '-' }}</div>
          </div>
          <div>
            <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.moderation.detail.userId') }}</div>
            <div class="mt-1 text-sm text-gray-800 dark:text-gray-100">{{ selectedAudit.user_id ?? '-' }}</div>
          </div>
          <div>
            <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.moderation.detail.apiKeyId') }}</div>
            <div class="mt-1 text-sm text-gray-800 dark:text-gray-100">{{ selectedAudit.api_key_id ?? '-' }}</div>
          </div>
          <div class="md:col-span-2">
            <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.moderation.detail.contentHash') }}</div>
            <div class="mt-1 break-all font-mono text-sm text-gray-800 dark:text-gray-100">{{ selectedAudit.content_hash }}</div>
          </div>
          <div class="md:col-span-2">
            <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.moderation.detail.summary') }}</div>
            <div class="mt-1 rounded-xl bg-gray-50 p-4 text-sm text-gray-800 dark:bg-dark-800 dark:text-gray-100">
              {{ selectedAudit.content_summary || '-' }}
            </div>
          </div>
          <div>
            <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.moderation.detail.latency') }}</div>
            <div class="mt-1 text-sm text-gray-800 dark:text-gray-100">{{ selectedAudit.latency_ms }} ms</div>
          </div>
          <div>
            <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.moderation.detail.errorReason') }}</div>
            <div class="mt-1 text-sm text-gray-800 dark:text-gray-100">{{ selectedAudit.error_reason || '-' }}</div>
          </div>
          <div>
            <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.moderation.detail.matchedKeyword') }}</div>
            <div class="mt-1 font-mono text-sm text-gray-800 dark:text-gray-100">{{ selectedAudit.matched_keyword || '-' }}</div>
          </div>
          <div class="md:col-span-2">
            <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.moderation.detail.categories') }}</div>
            <div class="mt-2 flex flex-wrap gap-2">
              <span
                v-for="category in selectedAudit.categories || []"
                :key="category"
                class="rounded-full bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300"
              >
                {{ category }}
              </span>
              <span v-if="!selectedAudit.categories?.length" class="text-sm text-gray-800 dark:text-gray-100">-</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { ContentModerationAudit, PaginatedResponse } from '@/types'
import { useAppStore } from '@/stores/app'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const audits = ref<ContentModerationAudit[]>([])
const selectedAudit = ref<ContentModerationAudit | null>(null)
const hitFilter = ref<string>('')
const pagination = reactive<PaginatedResponse<ContentModerationAudit>>({
  items: [],
  total: 0,
  page: 1,
  page_size: 20,
  pages: 1
})

const filters = reactive({
  request_id: '',
  client_request_id: '',
  provider: '',
  model: '',
  source_endpoint: '',
  content_hash: '',
  user_id: undefined as number | undefined
})

const hitOptions = computed(() => [
  { value: '', label: t('admin.moderation.filters.allHits') },
  { value: 'true', label: t('admin.moderation.filters.hitOnly') },
  { value: 'false', label: t('admin.moderation.filters.passOnly') }
])

function buildQuery() {
  return {
    page: pagination.page,
    page_size: pagination.page_size,
    request_id: filters.request_id || undefined,
    client_request_id: filters.client_request_id || undefined,
    provider: filters.provider || undefined,
    model: filters.model || undefined,
    source_endpoint: filters.source_endpoint || undefined,
    content_hash: filters.content_hash || undefined,
    user_id: filters.user_id || undefined,
    hit: hitFilter.value === '' ? undefined : hitFilter.value === 'true'
  }
}

function formatDateTime(value: string) {
  if (!value) return '-'
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short'
  }).format(new Date(value))
}

async function loadAudits() {
  loading.value = true
  try {
    const data = await adminAPI.moderation.listAudits(buildQuery())
    audits.value = data.items
    Object.assign(pagination, data)
  } catch (error: any) {
    appStore.showError(`${t('admin.moderation.loadFailed')}: ${error.message || t('common.unknownError')}`)
  } finally {
    loading.value = false
  }
}

async function openDetail(id: number) {
  try {
    selectedAudit.value = await adminAPI.moderation.getAuditDetail(id)
  } catch (error: any) {
    appStore.showError(`${t('admin.moderation.detailFailed')}: ${error.message || t('common.unknownError')}`)
  }
}

function applyFilters() {
  pagination.page = 1
  void loadAudits()
}

function resetFilters() {
  filters.request_id = ''
  filters.client_request_id = ''
  filters.provider = ''
  filters.model = ''
  filters.source_endpoint = ''
  filters.content_hash = ''
  filters.user_id = undefined
  hitFilter.value = ''
  selectedAudit.value = null
  pagination.page = 1
  void loadAudits()
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadAudits()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadAudits()
}

onMounted(() => {
  void loadAudits()
})
</script>
