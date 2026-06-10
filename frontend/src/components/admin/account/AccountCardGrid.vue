<template>
  <div v-if="loading" class="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-10 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-gray-400">
    {{ t('common.loading') }}
  </div>
  <div v-else-if="accounts.length === 0" class="rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-4 py-10 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-900/40 dark:text-gray-400">
    {{ emptyText }}
  </div>
  <div v-else ref="gridRootRef">
    <div
      v-if="shouldFallbackToDirectRows"
      class="space-y-4"
    >
      <div
        v-for="(row, rowIndex) in directRows"
        :key="`direct-${rowIndex}`"
        class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
      >
        <AccountCard
          v-for="account in row"
          :key="account.id"
          :account="account"
          :selected="selectedIds.includes(account.id)"
          :toggling-schedulable="togglingSchedulable"
          :today-stats-by-account-id="todayStatsByAccountId"
          :today-stats-loading="todayStatsLoading"
          :usage-manual-refresh-token="usageManualRefreshToken"
          :visual-style="visualStyle"
          :white-surface-enabled="whiteSurfaceEnabled"
          :account-group-display-mode="accountGroupDisplayMode"
          :account-status-display-mode="accountStatusDisplayMode"
          @toggle-selected="emit('toggle-selected', $event)"
          @show-temp-unsched="emit('show-temp-unsched', $event)"
          @toggle-schedulable="emit('toggle-schedulable', $event)"
          @edit="emit('edit', $event)"
          @delete="emit('delete', $event)"
          @open-menu="emit('open-menu', $event)"
        />
      </div>
    </div>

    <div
      v-else
      class="relative"
      :style="{ height: `${totalHeight}px` }"
    >
      <div
        v-for="row in renderedRows"
        :key="row.key"
        :ref="(element) => measureRow(element as Element | null)"
        class="absolute left-0 top-0 w-full pb-4"
        :style="{ transform: `translateY(${row.start}px)` }"
      >
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          <AccountCard
            v-for="account in row.items"
            :key="account.id"
            :account="account"
            :selected="selectedIds.includes(account.id)"
            :toggling-schedulable="togglingSchedulable"
            :today-stats-by-account-id="todayStatsByAccountId"
            :today-stats-loading="todayStatsLoading"
            :usage-manual-refresh-token="usageManualRefreshToken"
            :visual-style="visualStyle"
            :white-surface-enabled="whiteSurfaceEnabled"
            :account-group-display-mode="accountGroupDisplayMode"
            :account-status-display-mode="accountStatusDisplayMode"
            @toggle-selected="emit('toggle-selected', $event)"
            @show-temp-unsched="emit('show-temp-unsched', $event)"
            @toggle-schedulable="emit('toggle-schedulable', $event)"
            @edit="emit('edit', $event)"
            @delete="emit('delete', $event)"
            @open-menu="emit('open-menu', $event)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  Account,
  AccountGroupDisplayMode,
  AccountStatusDisplayMode,
  AccountVisualStyle,
  WindowStats,
} from '@/types'
import AccountCard from './AccountCard.vue'
import { useVirtualAccountCardRows } from './useVirtualAccountCardRows'

const props = withDefaults(defineProps<{
  accounts: Account[]
  loading: boolean
  selectedIds: number[]
  togglingSchedulable: number | null
  todayStatsByAccountId: Record<string, WindowStats>
  todayStatsLoading: boolean
  usageManualRefreshToken: number
  emptyText?: string
  visualStyle?: AccountVisualStyle
  whiteSurfaceEnabled?: boolean
  accountGroupDisplayMode?: AccountGroupDisplayMode
  accountStatusDisplayMode?: AccountStatusDisplayMode
}>(), {
  emptyText: '',
  visualStyle: 'airy',
  whiteSurfaceEnabled: false,
  accountGroupDisplayMode: 'full',
  accountStatusDisplayMode: 'detailed'
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
const {
  rootRef: gridRootRef,
  directRows,
  renderedRows,
  shouldFallbackToDirectRows,
  totalHeight,
  measureRow,
} = useVirtualAccountCardRows({
  items: computed(() => props.accounts),
  estimateRowHeight: 420,
  overscan: 2,
})
</script>
