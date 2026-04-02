<template>
  <DataTable
    :columns="columns"
    :data="accounts"
    :loading="loading"
    row-key="id"
    default-sort-key="name"
    default-sort-order="asc"
    :sort-storage-key="sortStorageKey"
    :preserve-input-order="preserveInputOrder"
  >
    <template #header-select>
      <input
        type="checkbox"
        class="h-4 w-4 cursor-pointer rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        :checked="allVisibleSelected"
        @click.stop
        @change="handleToggleSelectAllVisible"
      />
    </template>

    <template #cell-select="{ row }">
      <input
        type="checkbox"
        class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        :checked="selectedIds.includes(row.id)"
        @change="emit('toggle-selected', row.id)"
      />
    </template>

    <template #cell-name="{ row, value }">
      <div class="flex flex-col">
        <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
        <span
          v-if="row.extra?.email_address"
          class="max-w-[200px] truncate text-xs text-gray-500 dark:text-gray-400"
          :title="row.extra.email_address"
        >
          {{ row.extra.email_address }}
        </span>
      </div>
    </template>

    <template #cell-notes="{ value }">
      <span
        v-if="value"
        :title="value"
        class="block max-w-xs truncate text-sm text-gray-600 dark:text-gray-300"
      >
        {{ value }}
      </span>
      <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
    </template>

    <template #cell-platform_type="{ row }">
      <PlatformTypeBadge
        :platform="row.platform"
        :gateway-protocol="row.gateway_protocol"
        :type="row.type"
        :plan-type="row.credentials?.plan_type"
        :privacy-mode="String(row.extra?.privacy_mode || '') || undefined"
      />
    </template>

    <template #cell-capacity="{ row }">
      <AccountCapacityCell :account="row" />
    </template>

    <template #cell-status="{ row }">
      <AccountStatusIndicator :account="row" @show-temp-unsched="emit('show-temp-unsched', row)" />
    </template>

    <template #cell-schedulable="{ row }">
      <button
        class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-dark-800"
        :class="[
          row.schedulable
            ? 'bg-primary-500 hover:bg-primary-600'
            : 'bg-gray-200 hover:bg-gray-300 dark:bg-dark-600 dark:hover:bg-dark-500'
        ]"
        :disabled="togglingSchedulable === row.id"
        :title="
          row.schedulable
            ? t('admin.accounts.schedulableEnabled')
            : t('admin.accounts.schedulableDisabled')
        "
        @click="emit('toggle-schedulable', row)"
      >
        <span
          class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
          :class="[row.schedulable ? 'translate-x-4' : 'translate-x-0']"
        />
      </button>
    </template>

    <template #cell-today_stats="{ row }">
      <AccountTodayStatsCell
        :stats="todayStatsByAccountId[String(row.id)] ?? null"
        :loading="todayStatsLoading"
        :error="todayStatsError"
      />
    </template>

    <template #cell-groups="{ row }">
      <AccountGroupsCell :groups="row.groups" :max-display="4" />
    </template>

    <template #cell-usage="{ row }">
      <AccountUsageCell
        :account="row"
        :today-stats="todayStatsByAccountId[String(row.id)] ?? null"
        :today-stats-loading="todayStatsLoading"
        :manual-refresh-token="usageManualRefreshToken"
      />
    </template>

    <template #cell-usage_reset_dates="{ row }">
      <AccountUsageResetCell :account="row" />
    </template>

    <template #cell-proxy="{ row }">
      <div v-if="row.proxy" class="flex items-center gap-2">
        <span class="text-sm text-gray-700 dark:text-gray-300">{{ row.proxy.name }}</span>
        <span v-if="row.proxy.country_code" class="text-xs text-gray-500 dark:text-gray-400">
          ({{ formatCountryLabel(row.proxy.country_code, row.proxy.country, locale) }})
        </span>
      </div>
      <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
    </template>

    <template #cell-rate_multiplier="{ row }">
      <span class="text-sm font-mono text-gray-700 dark:text-gray-300">
        {{ (row.rate_multiplier ?? 1).toFixed(2) }}x
      </span>
    </template>

    <template #cell-priority="{ value }">
      <span class="text-sm text-gray-700 dark:text-gray-300">{{ value }}</span>
    </template>

    <template #cell-last_used_at="{ value }">
      <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatRelativeTime(value) }}</span>
    </template>

    <template #cell-expires_at="{ row, value }">
      <div class="flex flex-col items-start gap-1">
        <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatExpiresAt(value) }}</span>
        <div
          v-if="isExpired(value) || (row.auto_pause_on_expired && value)"
          class="flex items-center gap-1"
        >
          <span
            v-if="isExpired(value)"
            class="inline-flex items-center rounded-md bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
          >
            {{ t('admin.accounts.expired') }}
          </span>
          <span
            v-if="row.auto_pause_on_expired && value"
            class="inline-flex items-center rounded-md bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
          >
            {{ t('admin.accounts.autoPauseOnExpired') }}
          </span>
        </div>
      </div>
    </template>

    <template #cell-actions="{ row }">
      <slot name="row-actions" :row="row">
        <AccountsViewRowActions
          @edit="emit('edit', row)"
          @delete="emit('delete', row)"
          @more="emit('open-menu', { account: row, event: $event })"
        />
      </slot>
    </template>
  </DataTable>

  <Pagination
    v-if="showPagination && pagination.total > 0"
    :page="pagination.page"
    :total="pagination.total"
    :page-size="pagination.page_size"
    @update:page="emit('page-change', $event)"
    @update:page-size="emit('page-size-change', $event)"
  />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Column } from '@/components/common/types'
import type { Account, WindowStats } from '@/types'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import AccountCapacityCell from '@/components/account/AccountCapacityCell.vue'
import AccountGroupsCell from '@/components/account/AccountGroupsCell.vue'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import AccountTodayStatsCell from '@/components/account/AccountTodayStatsCell.vue'
import AccountUsageCell from '@/components/account/AccountUsageCell.vue'
import AccountUsageResetCell from '@/components/account/AccountUsageResetCell.vue'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import { formatCountryLabel } from '@/utils/displayLabels'
import AccountsViewRowActions from './AccountsViewRowActions.vue'

withDefaults(defineProps<{
  columns: Column[]
  accounts: Account[]
  loading: boolean
  allVisibleSelected: boolean
  selectedIds: number[]
  togglingSchedulable: number | null
  todayStatsByAccountId: Record<string, WindowStats>
  todayStatsLoading: boolean
  todayStatsError: string | null
  usageManualRefreshToken: number
  sortStorageKey: string
  preserveInputOrder?: boolean
  pagination: {
    total: number
    page: number
    page_size: number
  }
  showPagination?: boolean
}>(), {
  preserveInputOrder: false,
  showPagination: true
})

const emit = defineEmits<{
  'toggle-select-all-visible': [checked: boolean]
  'toggle-selected': [id: number]
  'show-temp-unsched': [account: Account]
  'toggle-schedulable': [account: Account]
  edit: [account: Account]
  delete: [account: Account]
  'open-menu': [payload: { account: Account; event: MouseEvent }]
  'page-change': [page: number]
  'page-size-change': [size: number]
}>()

const { t, locale } = useI18n()

const handleToggleSelectAllVisible = (event: Event) => {
  emit('toggle-select-all-visible', (event.target as HTMLInputElement).checked)
}

const formatExpiresAt = (value: number | null) => {
  if (!value) return '-'
  return formatDateTime(
    new Date(value * 1000),
    {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    },
    'sv-SE'
  )
}

const isExpired = (value: number | null) => {
  if (!value) return false
  return value * 1000 <= Date.now()
}
</script>
