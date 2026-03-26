<template>
  <div class="mb-2 space-y-2">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('admin.accounts.archivedGroupsTitle') }}
        </h3>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.archivedGroupsDescription') }}
        </p>
      </div>
    </div>

    <div v-if="loading" class="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-6 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-gray-400">
      {{ t('common.loading') }}
    </div>
    <div v-else-if="groups.length === 0" class="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-6 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-gray-400">
      {{ t('admin.accounts.archivedGroupsEmpty') }}
    </div>

    <ArchivedAccountGroupSection
      v-for="group in groups"
      :key="group.group_id"
      :summary="group"
      :filters="filters"
      :columns="columns"
      :toggling-schedulable="togglingSchedulable"
      :today-stats-by-account-id="todayStatsByAccountId"
      :today-stats-loading="todayStatsLoading"
      :today-stats-error="todayStatsError"
      :usage-manual-refresh-token="usageManualRefreshToken"
      :sort-storage-key="sortStorageKey"
      :refresh-token="refreshToken"
      @edit="emit('edit', $event)"
      @delete="emit('delete', $event)"
      @open-menu="emit('open-menu', $event)"
      @show-temp-unsched="emit('show-temp-unsched', $event)"
      @toggle-schedulable="emit('toggle-schedulable', $event)"
      @changed="handleChanged"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import ArchivedAccountGroupSection from './ArchivedAccountGroupSection.vue'
import type { Column } from '@/components/common/types'
import type { Account, ArchivedAccountGroupSummary, WindowStats } from '@/types'

const props = defineProps<{
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

const groups = ref<ArchivedAccountGroupSummary[]>([])
const loading = ref(false)

const loadGroups = async () => {
  loading.value = true
  try {
    groups.value = await adminAPI.accounts.listArchivedGroups({
      platform: props.filters.platform || '',
      type: props.filters.type || '',
      status: props.filters.status || '',
      group: props.filters.group || '',
      search: props.filters.search || ''
    })
  } catch (error) {
    console.error('Failed to load archived account groups:', error)
    groups.value = []
  } finally {
    loading.value = false
  }
}

const handleChanged = async () => {
  await loadGroups()
  emit('changed')
}

watch(
  () => [props.filters.platform, props.filters.type, props.filters.status, props.filters.group, props.filters.search, props.refreshToken],
  () => {
    loadGroups().catch((error) => {
      console.error('Failed to refresh archived account groups:', error)
    })
  },
  { immediate: true }
)
</script>
