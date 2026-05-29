<template>
  <DataTable
    :columns="columns"
    :data="codes"
    :loading="loading"
    server-side-sort
    default-sort-key="id"
    default-sort-order="desc"
    @sort="(key, order) => emit('sort', key, order)"
  >
    <template #header-select>
      <input
        type="checkbox"
        class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        :checked="allVisibleSelected"
        @change="emit('toggle-visible-selection', ($event.target as HTMLInputElement).checked)"
      />
    </template>
    <template #cell-select="{ row }">
      <input
        type="checkbox"
        class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        :checked="selectedIds.includes(row.id)"
        @change="emit('toggle-selection', row.id)"
      />
    </template>
    <template #cell-code="{ value }">
      <div class="flex items-center space-x-2">
        <code class="font-mono text-sm text-gray-900 dark:text-gray-100">{{ value }}</code>
        <button
          @click="emit('copy', value)"
          :class="[
            'flex items-center transition-colors',
            copiedCode === value
              ? 'text-green-500'
              : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-300'
          ]"
          :title="copiedCode === value ? t('admin.redeem.copied') : t('keys.copyToClipboard')"
        >
          <Icon v-if="copiedCode !== value" name="copy" size="sm" :stroke-width="2" />
          <svg v-else class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M5 13l4 4L19 7"
            />
          </svg>
        </button>
      </div>
    </template>

    <template #cell-type="{ value }">
      <span
        :class="[
          'badge',
          value === 'balance'
            ? 'badge-success'
            : value === 'subscription'
              ? 'badge-warning'
              : 'badge-primary'
        ]"
      >
        {{ t('admin.redeem.types.' + value) }}
      </span>
    </template>

    <template #cell-value="{ value, row }">
      <span class="text-sm font-medium text-gray-900 dark:text-white">
        <template v-if="row.type === 'balance'">${{ value.toFixed(2) }}</template>
        <template v-else-if="row.type === 'subscription'">
          {{ row.validity_days || 30 }} {{ t('admin.redeem.days') }}
          <span v-if="row.group" class="ml-1 text-xs text-gray-500 dark:text-gray-400"
            >({{ row.group.name }})</span
          >
        </template>
        <template v-else>{{ value }}</template>
      </span>
    </template>

    <template #cell-status="{ value }">
      <span
        :class="[
          'badge',
          value === 'unused'
            ? 'badge-success'
            : value === 'used'
              ? 'badge-gray'
              : value === 'disabled'
                ? 'badge-warning'
                : 'badge-danger'
        ]"
      >
        {{ t('admin.redeem.status.' + value) }}
      </span>
    </template>

    <template #cell-used_by="{ value, row }">
      <span class="text-sm text-gray-500 dark:text-dark-400">
        {{ row.user?.email || (value ? t('admin.redeem.userPrefix', { id: value }) : '-') }}
      </span>
    </template>

    <template #cell-used_at="{ value }">
      <span class="text-sm text-gray-500 dark:text-dark-400">{{
        value ? formatDateTime(value) : '-'
      }}</span>
    </template>

    <template #cell-expires_at="{ value }">
      <span class="text-sm text-gray-500 dark:text-dark-400">{{
        value ? formatDateTime(value) : '-'
      }}</span>
    </template>

    <template #cell-actions="{ row }">
      <div class="flex items-center space-x-2">
        <button
          v-if="row.status === 'unused'"
          @click="emit('delete', row)"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
        >
          <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
            />
          </svg>
          <span class="text-xs">{{ t('common.delete') }}</span>
        </button>
        <span v-else class="text-gray-400 dark:text-dark-500">-</span>
      </div>
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { SortOrder } from '@/api/admin/redeem'
import { formatDateTime } from '@/utils/format'
import type { RedeemCode } from '@/types'
import type { Column } from '@/components/common/types'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  columns: Column[]
  codes: RedeemCode[]
  loading: boolean
  allVisibleSelected: boolean
  selectedIds: number[]
  copiedCode: string | null
}>()

const emit = defineEmits<{
  sort: [key: string, order: SortOrder]
  'toggle-visible-selection': [checked: boolean]
  'toggle-selection': [id: number]
  copy: [code: string]
  delete: [code: RedeemCode]
}>()

const { t } = useI18n()
</script>
