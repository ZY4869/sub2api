<template>
  <DataTable
    :columns="columns"
    :data="subscriptions"
    :loading="loading"
    :server-side-sort="true"
    @sort="(key, order) => emit('sort', key, order)"
  >
    <template #cell-user="{ row }">
      <div class="flex items-center gap-2">
        <div
          class="flex h-8 w-8 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30"
        >
          <span class="text-sm font-medium text-primary-700 dark:text-primary-300">
            {{ userColumnMode === 'email'
              ? (row.user?.email?.charAt(0).toUpperCase() || '?')
              : (row.user?.username?.charAt(0).toUpperCase() || '?')
            }}
          </span>
        </div>
        <span class="font-medium text-gray-900 dark:text-white">
          {{ userColumnMode === 'email'
            ? (row.user?.email || t('admin.redeem.userPrefix', { id: row.user_id }))
            : (row.user?.username || '-')
          }}
        </span>
      </div>
    </template>

    <template #cell-group="{ row }">
      <GroupBadge
        v-if="row.group"
        :name="row.group.name"
        :platform="row.group.platform"
        :subscription-type="row.group.subscription_type"
        :rate-multiplier="row.group.rate_multiplier"
        :show-rate="false"
      />
      <span v-else class="text-sm text-gray-400 dark:text-dark-500">-</span>
    </template>

    <template #cell-usage="{ row }">
      <div class="min-w-[280px] space-y-2">
        <SubscriptionUsageWindow
          v-if="row.group?.daily_limit_usd"
          :label="t('admin.subscriptions.daily')"
          :used="row.daily_usage_usd"
          :limit="row.group?.daily_limit_usd"
          :window-start="row.daily_window_start"
          period="daily"
        />
        <SubscriptionUsageWindow
          v-if="row.group?.weekly_limit_usd"
          :label="t('admin.subscriptions.weekly')"
          :used="row.weekly_usage_usd"
          :limit="row.group?.weekly_limit_usd"
          :window-start="row.weekly_window_start"
          period="weekly"
        />
        <SubscriptionUsageWindow
          v-if="row.group?.monthly_limit_usd"
          :label="t('admin.subscriptions.monthly')"
          :used="row.monthly_usage_usd"
          :limit="row.group?.monthly_limit_usd"
          :window-start="row.monthly_window_start"
          period="monthly"
        />
        <div
          v-if="
            !row.group?.daily_limit_usd &&
            !row.group?.weekly_limit_usd &&
            !row.group?.monthly_limit_usd
          "
          class="flex items-center gap-2 rounded-lg bg-gradient-to-r from-emerald-50 to-teal-50 px-3 py-2 dark:from-emerald-900/20 dark:to-teal-900/20"
        >
          <span class="text-lg text-emerald-600 dark:text-emerald-400">∞</span>
          <span class="text-xs font-medium text-emerald-700 dark:text-emerald-300">
            {{ t('admin.subscriptions.unlimited') }}
          </span>
        </div>
      </div>
    </template>

    <template #cell-expires_at="{ value }">
      <div v-if="value">
        <span
          class="text-sm"
          :class="
            isSubscriptionExpiringSoon(value)
              ? 'text-orange-600 dark:text-orange-400'
              : 'text-gray-700 dark:text-gray-300'
          "
        >
          {{ formatDateOnly(value) }}
        </span>
        <div v-if="getSubscriptionDaysRemaining(value) !== null" class="text-xs text-gray-500">
          {{ getSubscriptionDaysRemaining(value) }} {{ t('admin.subscriptions.daysRemaining') }}
        </div>
      </div>
      <span v-else class="text-sm text-gray-500">{{
        t('admin.subscriptions.noExpiration')
      }}</span>
    </template>

    <template #cell-status="{ value }">
      <span
        :class="[
          'badge',
          value === 'active'
            ? 'badge-success'
            : value === 'expired'
              ? 'badge-warning'
              : 'badge-danger'
        ]"
      >
        {{ t(`admin.subscriptions.status.${value}`) }}
      </span>
    </template>

    <template #cell-actions="{ row }">
      <div class="flex items-center gap-1">
        <button
          v-if="row.status === 'active' || row.status === 'expired'"
          @click="emit('extend', row)"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
        >
          <Icon name="calendar" size="sm" />
          <span class="text-xs">{{ t('admin.subscriptions.adjust') }}</span>
        </button>
        <button
          v-if="row.status === 'active'"
          @click="emit('reset-quota', row)"
          :disabled="resettingQuota && resettingSubscriptionId === row.id"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-orange-50 hover:text-orange-600 dark:hover:bg-orange-900/20 dark:hover:text-orange-400 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <Icon name="refresh" size="sm" />
          <span class="text-xs">{{ t('admin.subscriptions.resetQuota') }}</span>
        </button>
        <button
          v-if="row.status === 'active'"
          @click="emit('revoke', row)"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
        >
          <Icon name="ban" size="sm" />
          <span class="text-xs">{{ t('admin.subscriptions.revoke') }}</span>
        </button>
      </div>
    </template>

    <template #empty>
      <EmptyState
        :title="t('admin.subscriptions.noSubscriptionsYet')"
        :description="t('admin.subscriptions.assignFirstSubscription')"
        :action-text="t('admin.subscriptions.assignSubscription')"
        @action="emit('assign')"
      />
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { UserSubscription } from '@/types'
import type { Column } from '@/components/common/types'
import { formatDateOnly } from '@/utils/format'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import SubscriptionUsageWindow from './SubscriptionUsageWindow.vue'
import {
  getSubscriptionDaysRemaining,
  isSubscriptionExpiringSoon
} from './utils'

defineProps<{
  columns: Column[]
  subscriptions: UserSubscription[]
  loading: boolean
  userColumnMode: 'email' | 'username'
  resettingQuota: boolean
  resettingSubscriptionId: number | null
}>()

const emit = defineEmits<{
  sort: [key: string, order: 'asc' | 'desc']
  extend: [subscription: UserSubscription]
  'reset-quota': [subscription: UserSubscription]
  revoke: [subscription: UserSubscription]
  assign: []
}>()

const { t } = useI18n()
</script>
