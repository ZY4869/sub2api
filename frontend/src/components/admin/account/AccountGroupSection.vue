<template>
  <section :class="sectionClass">
    <div class="flex flex-wrap items-center justify-between gap-4 px-4 py-4">
      <button
        type="button"
        class="flex min-w-0 flex-1 items-center gap-3 text-left"
        @click="expanded = !expanded"
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
            {{ title }}
          </div>
          <div class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.groupView.stats', { count: accounts.length }) }}
          </div>
          <div class="text-xs text-gray-400 dark:text-gray-500">
            {{ t('admin.accounts.groupView.currentPageScope') }}
          </div>
        </div>
      </button>

      <div class="ml-auto flex flex-wrap items-center justify-end gap-2 text-xs font-medium">
        <span :class="activeBadgeClass">
          {{ t('admin.accounts.summary.active') }} {{ activeCount }}
        </span>
        <span :class="errorBadgeClass">
          {{ t('admin.accounts.summary.error') }} {{ errorCount }}
        </span>
        <span :class="rateLimitedBadgeClass">
          {{ t('admin.accounts.summary.rateLimited') }} {{ rateLimitedCount }}
        </span>
      </div>
    </div>

    <div v-if="expanded" :class="bodyClass">
      <AccountsViewTable
        v-if="viewMode === 'table'"
        :columns="columns"
        :accounts="accounts"
        :loading="false"
        :all-visible-selected="allVisibleSelected"
        :selected-ids="selectedIds"
        :toggling-schedulable="togglingSchedulable"
        :today-stats-by-account-id="todayStatsByAccountId"
        :today-stats-loading="todayStatsLoading"
        :today-stats-error="todayStatsError"
        :usage-manual-refresh-token="usageManualRefreshToken"
        :sort-storage-key="sectionSortStorageKey"
        :preserve-input-order="preserveInputOrder"
        :visual-style="visualStyle"
        :white-surface-enabled="whiteSurfaceEnabled"
        :account-today-stats-windows="accountTodayStatsWindows"
        :account-group-display-mode="accountGroupDisplayMode"
        :pagination="pagination"
        :show-pagination="false"
        @toggle-select-all-visible="emit('toggle-section-selected', { ids: accountIds, checked: $event })"
        @toggle-selected="emit('toggle-selected', $event)"
        @show-temp-unsched="emit('show-temp-unsched', $event)"
        @toggle-schedulable="emit('toggle-schedulable', $event)"
        @edit="emit('edit', $event)"
        @delete="emit('delete', $event)"
        @open-menu="emit('open-menu', $event)"
      />

      <AccountCardGrid
        v-else
        :accounts="accounts"
        :loading="false"
        :selected-ids="selectedIds"
        :toggling-schedulable="togglingSchedulable"
        :today-stats-by-account-id="todayStatsByAccountId"
        :today-stats-loading="todayStatsLoading"
        :usage-manual-refresh-token="usageManualRefreshToken"
        :visual-style="visualStyle"
        :white-surface-enabled="whiteSurfaceEnabled"
        @toggle-selected="emit('toggle-selected', $event)"
        @show-temp-unsched="emit('show-temp-unsched', $event)"
        @toggle-schedulable="emit('toggle-schedulable', $event)"
        @edit="emit('edit', $event)"
        @delete="emit('delete', $event)"
        @open-menu="emit('open-menu', $event)"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Column } from '@/components/common/types'
import type {
  Account,
  AccountGroupDisplayMode,
  AccountTodayStatsWindow,
  AccountViewMode,
  AccountVisualStyle,
  WindowStats,
} from '@/types'
import AccountCardGrid from './AccountCardGrid.vue'
import AccountsViewTable from './AccountsViewTable.vue'

const props = withDefaults(defineProps<{
  sectionKey: string
  title: string
  accounts: Account[]
  viewMode: AccountViewMode
  columns: Column[]
  selectedIds: number[]
  togglingSchedulable: number | null
  todayStatsByAccountId: Record<string, WindowStats>
  todayStatsLoading: boolean
  todayStatsError: string | null
  usageManualRefreshToken: number
  sortStorageKey: string
  preserveInputOrder?: boolean
  visualStyle?: AccountVisualStyle
  whiteSurfaceEnabled?: boolean
  accountTodayStatsWindows?: AccountTodayStatsWindow[]
  accountGroupDisplayMode?: AccountGroupDisplayMode
}>(), {
  visualStyle: 'airy',
  whiteSurfaceEnabled: false
})

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
const expanded = ref(false)

const accountIds = computed(() => props.accounts.map((account) => account.id))
const allVisibleSelected = computed(() => props.accounts.length > 0 && props.accounts.every((account) => props.selectedIds.includes(account.id)))
const activeCount = computed(() => props.accounts.filter((account) => account.status === 'active').length)
const errorCount = computed(() => props.accounts.filter((account) => account.status === 'error').length)
const sectionClass = computed(() =>
  props.visualStyle === 'airy'
    ? props.whiteSurfaceEnabled
      ? 'rounded-[1.75rem] border border-slate-200/85 bg-white shadow-[0_16px_34px_rgba(15,23,42,0.05)] dark:border-slate-700/80 dark:bg-slate-900'
      : 'rounded-[1.75rem] border border-slate-200/75 bg-[linear-gradient(135deg,rgba(255,255,255,0.97),rgba(248,250,252,0.92))] shadow-[0_18px_38px_rgba(148,163,184,0.14)] dark:border-slate-700/80 dark:bg-[linear-gradient(135deg,rgba(30,41,59,0.82),rgba(15,23,42,0.7))]'
    : 'rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800'
)
const bodyClass = computed(() =>
  props.visualStyle === 'airy'
    ? 'border-t border-slate-200/75 px-4 py-4 dark:border-slate-700/80'
    : 'border-t border-gray-100 px-4 py-4 dark:border-dark-700'
)
const activeBadgeClass = computed(() =>
  props.visualStyle === 'airy'
    ? 'rounded-full border border-emerald-200/80 bg-emerald-50/95 px-2.5 py-1 text-emerald-700 dark:border-emerald-400/25 dark:bg-emerald-500/12 dark:text-emerald-200'
    : 'rounded-full bg-emerald-50 px-2.5 py-1 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
)
const errorBadgeClass = computed(() =>
  props.visualStyle === 'airy'
    ? 'rounded-full border border-rose-200/80 bg-rose-50/95 px-2.5 py-1 text-rose-700 dark:border-rose-400/25 dark:bg-rose-500/12 dark:text-rose-200'
    : 'rounded-full bg-red-50 px-2.5 py-1 text-red-700 dark:bg-red-900/20 dark:text-red-300'
)
const rateLimitedBadgeClass = computed(() =>
  props.visualStyle === 'airy'
    ? 'rounded-full border border-amber-200/80 bg-amber-50/95 px-2.5 py-1 text-amber-700 dark:border-amber-400/25 dark:bg-amber-500/12 dark:text-amber-100'
    : 'rounded-full bg-amber-50 px-2.5 py-1 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300'
)
const rateLimitedCount = computed(() => props.accounts.filter((account) => {
  if (!account.rate_limit_reset_at) {
    return false
  }
  const resetAt = new Date(account.rate_limit_reset_at).getTime()
  return Number.isFinite(resetAt) && resetAt > Date.now()
}).length)
const sectionSortStorageKey = computed(() => `${props.sortStorageKey}-${props.sectionKey}`)
const pagination = computed(() => ({
  total: props.accounts.length,
  page: 1,
  page_size: Math.max(props.accounts.length, 1)
}))
</script>

<style scoped>
section {
  content-visibility: auto;
  contain-intrinsic-size: 320px;
}
</style>
