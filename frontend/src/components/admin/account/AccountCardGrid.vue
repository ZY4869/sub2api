<template>
  <div v-if="loading" class="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-10 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-gray-400">
    {{ t('common.loading') }}
  </div>
  <div v-else-if="accounts.length === 0" class="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-10 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-gray-400">
    {{ emptyText }}
  </div>
  <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
    <AccountCard
      v-for="account in accounts"
      :key="account.id"
      :account="account"
      :selected="selectedIds.includes(account.id)"
      :toggling-schedulable="togglingSchedulable"
      :today-stats-by-account-id="todayStatsByAccountId"
      :today-stats-loading="todayStatsLoading"
      :usage-manual-refresh-token="usageManualRefreshToken"
      @toggle-selected="emit('toggle-selected', $event)"
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
import type { Account, WindowStats } from '@/types'
import AccountCard from './AccountCard.vue'

const props = withDefaults(defineProps<{
  accounts: Account[]
  loading: boolean
  selectedIds: number[]
  togglingSchedulable: number | null
  todayStatsByAccountId: Record<string, WindowStats>
  todayStatsLoading: boolean
  usageManualRefreshToken: number
  emptyText?: string
}>(), {
  emptyText: ''
})

const emit = defineEmits<{
  'toggle-selected': [id: number]
  'show-temp-unsched': [account: Account]
  'toggle-schedulable': [account: Account]
  edit: [account: Account]
  delete: [account: Account]
  'open-menu': [payload: { account: Account; event: MouseEvent }]
}>()

const { t } = useI18n()

const emptyText = computed(() => props.emptyText || t('admin.accounts.noAccounts'))
</script>
