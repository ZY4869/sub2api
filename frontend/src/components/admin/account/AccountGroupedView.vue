<template>
  <div v-if="loading" class="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-10 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-gray-400">
    {{ t('common.loading') }}
  </div>
  <div v-else-if="sections.length === 0" class="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-10 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-gray-400">
    {{ t('admin.accounts.noAccounts') }}
  </div>
  <div v-else class="space-y-4">
    <AccountGroupSection
      v-for="section in sections"
      :key="section.key"
      :section-key="section.key"
      :title="section.title"
      :accounts="section.accounts"
      :view-mode="viewMode"
      :columns="columns"
      :selected-ids="selectedIds"
      :toggling-schedulable="togglingSchedulable"
      :today-stats-by-account-id="todayStatsByAccountId"
      :today-stats-loading="todayStatsLoading"
      :today-stats-error="todayStatsError"
      :usage-manual-refresh-token="usageManualRefreshToken"
      :sort-storage-key="sortStorageKey"
      @toggle-selected="emit('toggle-selected', $event)"
      @toggle-section-selected="emit('toggle-section-selected', $event)"
      @show-temp-unsched="emit('show-temp-unsched', $event)"
      @toggle-schedulable="emit('toggle-schedulable', $event)"
      @edit="emit('edit', $event)"
      @delete="emit('delete', $event)"
      @open-menu="emit('open-menu', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Column } from '@/components/common/types'
import type { Account, AccountViewMode, AdminGroup, WindowStats } from '@/types'
import AccountGroupSection from './AccountGroupSection.vue'

type GroupSection = {
  key: string
  title: string
  accounts: Account[]
}

const props = defineProps<{
  accounts: Account[]
  groups: AdminGroup[]
  groupFilter: string
  viewMode: AccountViewMode
  columns: Column[]
  selectedIds: number[]
  loading: boolean
  togglingSchedulable: number | null
  todayStatsByAccountId: Record<string, WindowStats>
  todayStatsLoading: boolean
  todayStatsError: string | null
  usageManualRefreshToken: number
  sortStorageKey: string
}>()

const emit = defineEmits<{
  'toggle-selected': [id: number]
  'toggle-section-selected': [payload: { ids: number[]; checked: boolean }]
  'show-temp-unsched': [account: Account]
  'toggle-schedulable': [account: Account]
  edit: [account: Account]
  delete: [account: Account]
  'open-menu': [payload: { account: Account; event: MouseEvent }]
}>()

const { t } = useI18n()

const groupNameMap = computed(() => {
  return props.groups.reduce<Map<number, string>>((acc, group) => {
    acc.set(group.id, group.name)
    return acc
  }, new Map<number, string>())
})

const resolveAccountGroups = (account: Account) => {
  const groups = new Map<number, string>()
  for (const group of account.groups || []) {
    if (group?.id) {
      groups.set(group.id, group.name)
    }
  }
  for (const groupID of account.group_ids || []) {
    if (groupID > 0) {
      groups.set(groupID, groups.get(groupID) || groupNameMap.value.get(groupID) || String(groupID))
    }
  }
  return [...groups.entries()].map(([id, name]) => ({ id, name }))
}

const appendSectionAccount = (sections: Map<string, GroupSection>, key: string, title: string, account: Account) => {
  const section = sections.get(key)
  if (section) {
    section.accounts.push(account)
    return
  }
  sections.set(key, {
    key,
    title,
    accounts: [account]
  })
}

const sections = computed(() => {
  const groupFilter = String(props.groupFilter || '').trim()
  const nextSections = new Map<string, GroupSection>()

  for (const account of props.accounts) {
    const accountGroups = resolveAccountGroups(account)

    if (groupFilter === 'ungrouped') {
      if (accountGroups.length === 0) {
        appendSectionAccount(nextSections, 'ungrouped', t('admin.accounts.groupView.ungrouped'), account)
      }
      continue
    }

    if (groupFilter) {
      const parsedGroupID = Number.parseInt(groupFilter, 10)
      if (Number.isFinite(parsedGroupID) && parsedGroupID > 0 && accountGroups.some((group) => group.id === parsedGroupID)) {
        appendSectionAccount(nextSections, `group-${parsedGroupID}`, groupNameMap.value.get(parsedGroupID) || accountGroups.find((group) => group.id === parsedGroupID)?.name || String(parsedGroupID), account)
      }
      continue
    }

    if (accountGroups.length === 0) {
      appendSectionAccount(nextSections, 'ungrouped', t('admin.accounts.groupView.ungrouped'), account)
      continue
    }

    for (const group of accountGroups) {
      appendSectionAccount(nextSections, `group-${group.id}`, group.name, account)
    }
  }

  return [...nextSections.values()].sort((a, b) => {
    if (a.key === 'ungrouped') return 1
    if (b.key === 'ungrouped') return -1
    return a.title.localeCompare(b.title)
  })
})
</script>
