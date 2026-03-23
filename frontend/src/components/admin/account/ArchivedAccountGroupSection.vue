<template>
  <section class="rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <div class="flex flex-wrap items-center justify-between gap-4 px-4 py-4">
      <button
        type="button"
        class="flex min-w-0 flex-1 items-center gap-3 text-left"
        @click="toggleExpanded"
      >
        <span class="flex h-9 w-9 items-center justify-center rounded-full bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200">
          <svg
            class="h-4 w-4 transition-transform"
            :class="expanded ? 'rotate-90' : ''"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            stroke-width="1.8"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="m9 5 7 7-7 7" />
          </svg>
        </span>
        <div class="min-w-0">
          <div class="truncate text-base font-semibold text-gray-900 dark:text-white">
            {{ summary.group_name }}
          </div>
          <div class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.archivedGroupAccountCount', { count: summary.total_count }) }}
          </div>
        </div>
      </button>

      <div class="ml-auto flex flex-wrap items-center justify-end gap-3">
        <div class="grid min-w-[8rem] gap-2 sm:grid-cols-2">
          <div class="rounded-xl bg-emerald-50 px-3 py-2 text-right dark:bg-emerald-900/20">
            <div class="text-[0.7rem] font-medium uppercase tracking-[0.2em] text-emerald-700 dark:text-emerald-300">
              {{ t('admin.accounts.archivedGroupAvailable') }}
            </div>
            <div class="text-3xl font-semibold leading-none text-emerald-800 dark:text-emerald-200">
              {{ paddedAvailableCount }}
            </div>
          </div>
          <div class="rounded-xl bg-rose-50 px-3 py-2 text-right dark:bg-rose-900/20">
            <div class="text-[0.7rem] font-medium uppercase tracking-[0.2em] text-rose-700 dark:text-rose-300">
              {{ t('admin.accounts.archivedGroupInvalid') }}
            </div>
            <div class="text-3xl font-semibold leading-none text-rose-800 dark:text-rose-200">
              {{ paddedInvalidCount }}
            </div>
          </div>
        </div>

        <button
          type="button"
          class="btn btn-secondary"
          :disabled="unarchivingGroup || loadingGroupIds"
          @click="handleUnarchiveGroup"
        >
          {{ t('admin.accounts.unarchiveGroup') }}
        </button>
      </div>
    </div>

    <div v-if="expanded" class="border-t border-gray-100 px-4 py-4 dark:border-dark-700">
      <div v-if="detailsLoading" class="py-6 text-sm text-gray-500 dark:text-gray-400">
        {{ t('common.loading') }}
      </div>
      <div v-else-if="detailsError" class="py-6 text-sm text-red-500">
        {{ detailsError }}
      </div>
      <div v-else>
        <AccountsViewTable
          :columns="columns"
          :accounts="accounts"
          :loading="detailsLoading"
          :all-visible-selected="false"
          :selected-ids="[]"
          :toggling-schedulable="togglingSchedulable"
          :today-stats-by-account-id="todayStatsByAccountId"
          :today-stats-loading="todayStatsLoading"
          :today-stats-error="todayStatsError"
          :usage-manual-refresh-token="usageManualRefreshToken"
          :sort-storage-key="sortStorageKey"
          :pagination="pagination"
          @show-temp-unsched="emit('show-temp-unsched', $event)"
          @toggle-schedulable="emit('toggle-schedulable', $event)"
          @edit="emit('edit', $event)"
          @delete="emit('delete', $event)"
          @open-menu="emit('open-menu', $event)"
          @page-change="handlePageChange"
          @page-size-change="handlePageSizeChange"
        >
          <template #row-actions="{ row }">
            <div class="flex items-center gap-2">
              <button
                type="button"
                class="rounded-lg border border-amber-200 px-2.5 py-1 text-xs font-medium text-amber-700 transition hover:bg-amber-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-amber-800 dark:text-amber-300 dark:hover:bg-amber-900/20"
                :disabled="unarchivingAccountId === row.id || unarchivingGroup"
                @click="handleUnarchiveAccount(row.id)"
              >
                {{ t('admin.accounts.unarchive') }}
              </button>
              <AccountsViewRowActions
                @edit="emit('edit', row)"
                @delete="emit('delete', row)"
                @more="emit('open-menu', { account: row, event: $event })"
              />
            </div>
          </template>
        </AccountsViewTable>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import AccountsViewRowActions from './AccountsViewRowActions.vue'
import AccountsViewTable from './AccountsViewTable.vue'
import type { Column } from '@/components/common/types'
import type { Account, ArchivedAccountGroupSummary, WindowStats } from '@/types'

const props = defineProps<{
  summary: ArchivedAccountGroupSummary
  filters: {
    platform?: string
    type?: string
    status?: string
    group?: string
    search?: string
  }
  columns: Column[]
  togglingSchedulable: number | null
  todayStatsByAccountId: Record<string, WindowStats>
  todayStatsLoading: boolean
  todayStatsError: string | null
  usageManualRefreshToken: number
  sortStorageKey: string
  refreshToken: number
}>()

const emit = defineEmits<{
  edit: [account: Account]
  delete: [account: Account]
  'open-menu': [payload: { account: Account; event: MouseEvent }]
  'show-temp-unsched': [account: Account]
  'toggle-schedulable': [account: Account]
  changed: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

const expanded = ref(false)
const hasLoaded = ref(false)
const detailsLoading = ref(false)
const detailsError = ref('')
const loadingGroupIds = ref(false)
const unarchivingGroup = ref(false)
const unarchivingAccountId = ref<number | null>(null)
const accounts = ref<Account[]>([])
const pagination = reactive({
  total: 0,
  page: 1,
  page_size: 10,
  pages: 0
})

const countPadWidth = computed(() => Math.max(2, String(Math.max(props.summary.total_count, 0)).length))
const paddedAvailableCount = computed(() => String(props.summary.available_count).padStart(countPadWidth.value, '0'))
const paddedInvalidCount = computed(() => String(props.summary.invalid_count).padStart(countPadWidth.value, '0'))

const buildListFilters = () => ({
  platform: props.filters.platform || '',
  type: props.filters.type || '',
  status: props.filters.status || '',
  search: props.filters.search || '',
  group: String(props.summary.group_id),
  lifecycle: 'archived'
})

const resetDetails = () => {
  hasLoaded.value = false
  accounts.value = []
  detailsError.value = ''
  pagination.total = 0
  pagination.page = 1
  pagination.pages = 0
}

const loadAccounts = async (page = pagination.page, pageSize = pagination.page_size) => {
  detailsLoading.value = true
  detailsError.value = ''
  try {
    const response = await adminAPI.accounts.list(page, pageSize, buildListFilters())
    accounts.value = response.items || []
    pagination.total = response.total || 0
    pagination.page = response.page || page
    pagination.page_size = response.page_size || pageSize
    pagination.pages = response.pages || 0
    hasLoaded.value = true
  } catch (error: any) {
    detailsError.value = error?.message || t('common.error')
  } finally {
    detailsLoading.value = false
  }
}

const toggleExpanded = async () => {
  expanded.value = !expanded.value
  if (expanded.value && !hasLoaded.value) {
    await loadAccounts(1, pagination.page_size)
  }
}

const handlePageChange = async (page: number) => {
  await loadAccounts(page, pagination.page_size)
}

const handlePageSizeChange = async (pageSize: number) => {
  await loadAccounts(1, pageSize)
}

const collectArchivedAccountIds = async () => {
  loadingGroupIds.value = true
  try {
    const pageSize = 200
    const collected: number[] = []
    let page = 1
    while (true) {
      const response = await adminAPI.accounts.list(page, pageSize, buildListFilters())
      const ids = (response.items || []).map((account) => account.id)
      collected.push(...ids)
      if (!response.items?.length || collected.length >= (response.total || 0)) {
        break
      }
      page += 1
    }
    return collected
  } finally {
    loadingGroupIds.value = false
  }
}

const handleUnarchiveAccount = async (accountId: number) => {
  unarchivingAccountId.value = accountId
  try {
    const result = await adminAPI.accounts.unarchiveAccounts([accountId])
    if (result.restored_count > 0) {
      if (expanded.value) {
        await loadAccounts(1, pagination.page_size)
      }
      appStore.showSuccess(t('admin.accounts.unarchiveSuccess', { count: result.restored_count }))
      emit('changed')
      return
    }
    appStore.showError(result.results[0]?.error_message || t('admin.accounts.unarchiveFailed'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.unarchiveFailed'))
  } finally {
    unarchivingAccountId.value = null
  }
}

const handleUnarchiveGroup = async () => {
  unarchivingGroup.value = true
  try {
    const accountIds = await collectArchivedAccountIds()
    if (accountIds.length === 0) {
      appStore.showWarning(t('admin.accounts.unarchiveNoAccounts'))
      return
    }
    const result = await adminAPI.accounts.unarchiveAccounts(accountIds)
    if (result.failed_count > 0 && result.restored_count > 0) {
      if (expanded.value) {
        await loadAccounts(1, pagination.page_size)
      }
      appStore.showWarning(
        t('admin.accounts.unarchivePartial', {
          restored: result.restored_count,
          failed: result.failed_count
        })
      )
      emit('changed')
      return
    }
    if (result.restored_count > 0) {
      if (expanded.value) {
        await loadAccounts(1, pagination.page_size)
      }
      appStore.showSuccess(t('admin.accounts.unarchiveSuccess', { count: result.restored_count }))
      emit('changed')
      return
    }
    appStore.showError(t('admin.accounts.unarchiveFailed'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.unarchiveFailed'))
  } finally {
    unarchivingGroup.value = false
  }
}

watch(
  () => [props.filters.platform, props.filters.type, props.filters.status, props.filters.group, props.filters.search],
  async () => {
    resetDetails()
    if (expanded.value) {
      await loadAccounts(1, pagination.page_size)
    }
  }
)

watch(
  () => props.refreshToken,
  async () => {
    resetDetails()
    if (expanded.value) {
      await loadAccounts(1, pagination.page_size)
    }
  }
)
</script>
