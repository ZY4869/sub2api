<template>
  <BaseDialog :show="show" :title="t('admin.accounts.archivedModalTitle')" width="full" @close="emit('close')">
    <div class="space-y-4">
      <div class="flex flex-wrap items-center gap-3">
        <SearchInput
          :model-value="String(params.search || '')"
          :placeholder="t('admin.accounts.searchAccounts')"
          class="w-full sm:w-64"
          @update:model-value="handleSearchUpdate"
          @search="reload"
        />
        <Select :model-value="params.platform" class="w-40" :options="platformOptions" @update:model-value="handlePlatformUpdate" @change="reload" />
        <Select :model-value="params.group" class="w-48" :options="groupOptions" @update:model-value="handleGroupUpdate" @change="reload" />
      </div>

      <DataTable :columns="columns" :data="accounts" :loading="loading" row-key="id">
        <template #cell-name="{ row }">
          <div class="flex flex-col">
            <span class="font-medium text-gray-900 dark:text-white">{{ row.name }}</span>
            <span v-if="row.lifecycle_reason_message" class="max-w-md truncate text-xs text-gray-500 dark:text-gray-400" :title="row.lifecycle_reason_message">
              {{ row.lifecycle_reason_message }}
            </span>
          </div>
        </template>

        <template #cell-platform="{ row }">
          <PlatformTypeBadge :platform="row.platform" :type="row.type" :plan-type="row.credentials?.plan_type" />
        </template>

        <template #cell-status="{ row }">
          <span class="inline-flex items-center rounded-md bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
            {{ resolveStatusLabel(row.status) }}
          </span>
        </template>

        <template #cell-groups="{ row }">
          <AccountGroupsCell :groups="row.groups" :max-display="4" />
        </template>

        <template #cell-updated_at="{ value }">
          <span class="text-sm text-gray-500 dark:text-dark-400" :title="formatDateTime(value)">
            {{ formatRelativeTime(value) }}
          </span>
        </template>
      </DataTable>

      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useTableLoader } from '@/composables/useTableLoader'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import type { Column } from '@/components/common/types'
import type { Account, AdminGroup } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import AccountGroupsCell from '@/components/account/AccountGroupsCell.vue'

const props = defineProps<{
  show: boolean
  groups: AdminGroup[]
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()

const {
  items: accounts,
  loading,
  params,
  pagination,
  reload,
  handlePageChange,
  handlePageSizeChange
} = useTableLoader<Account, { platform: string; group: string; search: string; lifecycle: string }>({
  fetchFn: adminAPI.accounts.list,
  initialParams: {
    platform: '',
    group: '',
    search: '',
    lifecycle: 'archived'
  }
})

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.accounts.columns.name'), sortable: true },
  { key: 'platform', label: t('admin.accounts.columns.platformType') },
  { key: 'status', label: t('admin.accounts.columns.status') },
  { key: 'groups', label: t('admin.accounts.columns.groups') },
  { key: 'updated_at', label: t('admin.accounts.columns.updatedAt'), sortable: true }
])

const platformOptions = computed(() => [
  { value: '', label: t('admin.accounts.allPlatforms') },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'kiro', label: 'Kiro' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'copilot', label: 'Copilot' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
  { value: 'sora', label: 'Sora' }
])

const groupOptions = computed(() => [
  { value: '', label: t('admin.accounts.allGroups') },
  { value: 'ungrouped', label: t('admin.accounts.ungroupedGroup') },
  ...props.groups.map((group) => ({ value: String(group.id), label: group.name }))
])

const handleSearchUpdate = (value: string) => {
  params.search = value
}

const handlePlatformUpdate = (value: string | number | boolean | null) => {
  params.platform = String(value || '')
}

const handleGroupUpdate = (value: string | number | boolean | null) => {
  params.group = String(value || '')
}

const resolveStatusLabel = (status: string) => {
  if (status === 'inactive') {
    return t('admin.accounts.status.inactive')
  }
  if (status === 'error') {
    return t('admin.accounts.status.error')
  }
  return t('admin.accounts.status.active')
}

watch(
  () => props.show,
  (open) => {
    if (open) {
      reload()
    }
  }
)
</script>
