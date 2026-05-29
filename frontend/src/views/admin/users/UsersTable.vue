<template>
  <DataTable :columns="columns" :data="users" :loading="loading" :actions-count="7">
    <template #cell-email="{ value, row }">
      <div class="flex items-center gap-2">
        <div class="relative flex h-8 w-8 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30">
          <span class="text-sm font-medium text-primary-700 dark:text-primary-300">
            {{ value.charAt(0).toUpperCase() }}
          </span>
          <span v-if="row.role === 'admin'" class="absolute -right-1 -top-1 rounded-full bg-amber-400 p-0.5 text-white shadow-sm dark:bg-amber-500">
            <Icon name="crown" size="xs" class="h-3 w-3" />
          </span>
        </div>
        <div class="min-w-0">
          <div class="font-medium text-gray-900 dark:text-white">{{ value }}</div>
          <div v-if="row.role === 'admin' || row.admin_free_billing" class="mt-0.5 flex flex-wrap items-center gap-1">
            <span v-if="row.role === 'admin'" class="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300">
              <Icon name="crown" size="xs" class="h-3 w-3" />
              管理员
            </span>
            <span v-if="row.admin_free_billing" class="inline-flex items-center rounded-full bg-emerald-100 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300">
              免扣
            </span>
          </div>
        </div>
      </div>
    </template>

    <template #cell-username="{ value }">
      <span class="text-sm text-gray-700 dark:text-gray-300">{{ value || '-' }}</span>
    </template>

    <template #cell-notes="{ value }">
      <div class="max-w-xs">
        <span
          v-if="value"
          :title="value.length > 30 ? value : undefined"
          class="block truncate text-sm text-gray-600 dark:text-gray-400"
        >
          {{ value.length > 30 ? value.substring(0, 25) + '...' : value }}
        </span>
        <span v-else class="text-sm text-gray-400">-</span>
      </div>
    </template>

    <template
      v-for="def in attributeDefinitions.filter(d => d.enabled)"
      :key="def.id"
      #[`cell-attr_${def.id}`]="{ row }"
    >
      <div class="max-w-xs">
        <span
          class="block truncate text-sm text-gray-700 dark:text-gray-300"
          :title="getAttributeValue(row.id, def.id)"
        >
          {{ getAttributeValue(row.id, def.id) }}
        </span>
      </div>
    </template>

    <template #cell-role="{ value }">
      <span :class="['badge', value === 'admin' ? 'badge-purple' : 'badge-gray']">
        {{ t('admin.users.roles.' + value) }}
      </span>
    </template>

    <template #cell-groups="{ row }">
      <div v-if="allGroups.length > 0" class="flex flex-col gap-1">
        <span
          v-if="getUserGroups(row).exclusive.length > 0"
          class="group/ex relative inline-flex cursor-pointer items-center gap-1 whitespace-nowrap text-xs"
          @click.stop="emit('toggle-expanded-group', row.id)"
        >
          <Icon name="shield" size="xs" class="h-3.5 w-3.5 text-purple-500 dark:text-purple-400" />
          <span class="font-medium text-purple-600 dark:text-purple-400">{{ getUserGroups(row).exclusive.length }}</span>
          <span class="text-gray-500 dark:text-dark-400">{{ t('admin.users.exclusiveLabel') }}</span>
          <div
            v-if="expandedGroupUserId !== row.id"
            class="pointer-events-none absolute left-0 top-full z-50 mt-1.5 rounded bg-gray-900 px-2.5 py-1.5 text-xs text-white opacity-0 shadow-lg transition-opacity duration-75 group-hover/ex:opacity-100 dark:bg-dark-600"
          >
            <div class="absolute left-4 bottom-full border-4 border-transparent border-b-gray-900 dark:border-b-dark-600"></div>
            <div class="flex flex-col gap-0.5 whitespace-nowrap">
              <span v-for="g in getUserGroups(row).exclusive" :key="g.id">{{ g.name }}</span>
            </div>
          </div>
          <div
            v-if="expandedGroupUserId === row.id"
            class="absolute left-0 top-full z-50 mt-1.5 min-w-[160px] overflow-hidden rounded-lg border border-gray-200 bg-white py-1 text-xs shadow-xl dark:border-dark-600 dark:bg-dark-700"
          >
            <div class="border-b border-gray-100 px-3 py-1.5 text-[10px] font-medium uppercase tracking-wider text-gray-400 dark:border-dark-600 dark:text-dark-400">
              {{ t('admin.users.clickToReplace') }}
            </div>
            <div
              v-for="g in getUserGroups(row).exclusive"
              :key="g.id"
              class="flex cursor-pointer items-center gap-2 px-3 py-2 text-gray-700 transition-colors hover:bg-primary-50 hover:text-primary-600 dark:text-dark-200 dark:hover:bg-primary-900/30 dark:hover:text-primary-400"
              @click.stop="emit('open-group-replace', row, g)"
            >
              <Icon name="swap" size="xs" class="h-3.5 w-3.5 flex-shrink-0 opacity-50" />
              <span class="flex-1">{{ g.name }}</span>
            </div>
          </div>
        </span>
        <span
          v-if="getUserGroups(row).publicGroups.length > 0"
          class="group/pub relative inline-flex cursor-default items-center gap-1 whitespace-nowrap text-xs"
        >
          <Icon name="globe" size="xs" class="h-3.5 w-3.5 text-gray-400 dark:text-dark-500" />
          <span class="font-medium text-gray-600 dark:text-dark-300">{{ getUserGroups(row).publicGroups.length }}</span>
          <span class="text-gray-400 dark:text-dark-500">{{ t('admin.users.publicLabel') }}</span>
          <div class="pointer-events-none absolute left-0 top-full z-50 mt-1.5 rounded bg-gray-900 px-2.5 py-1.5 text-xs text-white opacity-0 shadow-lg transition-opacity duration-75 group-hover/pub:opacity-100 dark:bg-dark-600">
            <div class="absolute left-4 bottom-full border-4 border-transparent border-b-gray-900 dark:border-b-dark-600"></div>
            <div class="flex flex-col gap-0.5 whitespace-nowrap">
              <span v-for="g in getUserGroups(row).publicGroups" :key="g.id">{{ g.name }}</span>
            </div>
          </div>
        </span>
        <span
          v-if="getUserGroups(row).exclusive.length === 0 && getUserGroups(row).publicGroups.length === 0"
          class="text-xs text-gray-400 dark:text-dark-500"
        >-</span>
      </div>
      <span v-else class="text-xs text-gray-400 dark:text-dark-500">-</span>
    </template>

    <template #cell-subscriptions="{ row }">
      <div
        v-if="row.subscriptions && row.subscriptions.length > 0"
        class="flex flex-wrap gap-1.5"
      >
        <GroupBadge
          v-for="sub in row.subscriptions"
          :key="sub.id"
          :name="sub.group?.name || ''"
          :platform="sub.group?.platform"
          :subscription-type="sub.group?.subscription_type"
          :rate-multiplier="sub.group?.rate_multiplier"
          :days-remaining="sub.expires_at ? getDaysRemaining(sub.expires_at) : null"
          :title="sub.expires_at ? formatDateTime(sub.expires_at) : ''"
        />
      </div>
      <span
        v-else
        class="inline-flex items-center gap-1.5 rounded-md bg-gray-50 px-2 py-1 text-xs text-gray-400 dark:bg-dark-700/50 dark:text-dark-500"
      >
        <Icon name="ban" size="xs" class="h-3.5 w-3.5" />
        <span>{{ t('admin.users.noSubscription') }}</span>
      </span>
    </template>

    <template #cell-balance="{ value, row }">
      <div class="flex items-center gap-2">
        <div class="group relative">
          <button
            class="font-medium text-gray-900 underline decoration-dashed decoration-gray-300 underline-offset-4 transition-colors hover:text-primary-600 dark:text-white dark:decoration-dark-500 dark:hover:text-primary-400"
            @click="emit('balance-history', row)"
          >
            ${{ value.toFixed(2) }}
          </button>
          <div class="pointer-events-none absolute bottom-full left-1/2 z-50 mb-1.5 -translate-x-1/2 whitespace-nowrap rounded bg-gray-900 px-2 py-1 text-xs text-white opacity-0 shadow-lg transition-opacity duration-75 group-hover:opacity-100 dark:bg-dark-600">
            {{ t('admin.users.balanceHistoryTip') }}
            <div class="absolute left-1/2 top-full -translate-x-1/2 border-4 border-transparent border-t-gray-900 dark:border-t-dark-600"></div>
          </div>
        </div>
        <button
          @click.stop="emit('deposit', row)"
          class="rounded px-2 py-0.5 text-xs font-medium text-emerald-600 transition-colors hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-900/20"
          :title="t('admin.users.deposit')"
        >
          {{ t('admin.users.deposit') }}
        </button>
      </div>
    </template>

    <template #cell-usage="{ row }">
      <div class="text-sm">
        <div class="flex items-center gap-1.5">
          <span class="text-gray-500 dark:text-gray-400">{{ t('admin.users.today') }}:</span>
          <span class="font-medium text-gray-900 dark:text-white">
            ${{ (usageStats[row.id]?.today_actual_cost ?? 0).toFixed(4) }}
          </span>
        </div>
        <div class="mt-0.5 flex items-center gap-1.5">
          <span class="text-gray-500 dark:text-gray-400">{{ t('admin.users.total') }}:</span>
          <span class="font-medium text-gray-900 dark:text-white">
            ${{ (usageStats[row.id]?.total_actual_cost ?? 0).toFixed(4) }}
          </span>
        </div>
      </div>
    </template>

    <template #cell-concurrency="{ row }">
      <UserConcurrencyCell
        :current="row.current_concurrency ?? 0"
        :max="row.concurrency"
      />
    </template>

    <template #cell-status="{ value }">
      <div class="flex items-center gap-1.5">
        <span
          :class="[
            'inline-block h-2 w-2 rounded-full',
            value === 'active' ? 'bg-green-500' : 'bg-red-500'
          ]"
        ></span>
        <span class="text-sm text-gray-700 dark:text-gray-300">
          {{ value === 'active' ? t('common.active') : t('admin.users.disabled') }}
        </span>
      </div>
    </template>

    <template #cell-created_at="{ value }">
      <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
    </template>

    <template #cell-actions="{ row }">
      <div class="flex items-center gap-1">
        <button
          @click="emit('edit', row)"
          class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
        >
          <Icon name="edit" size="sm" />
          <span class="text-xs">{{ t('common.edit') }}</span>
        </button>
        <button
          v-if="row.role !== 'admin'"
          @click="emit('toggle-status', row)"
          :class="[
            'flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors',
            row.status === 'active'
              ? 'hover:bg-orange-50 hover:text-orange-600 dark:hover:bg-orange-900/20 dark:hover:text-orange-400'
              : 'hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400'
          ]"
        >
          <Icon v-if="row.status === 'active'" name="ban" size="sm" />
          <Icon v-else name="checkCircle" size="sm" />
          <span class="text-xs">{{ row.status === 'active' ? t('admin.users.disable') : t('admin.users.enable') }}</span>
        </button>
        <button
          @click="emit('open-menu', row, $event)"
          class="action-menu-trigger flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:hover:bg-dark-700 dark:hover:text-white"
          :class="{ 'bg-gray-100 text-gray-900 dark:bg-dark-700 dark:text-white': activeMenuId === row.id }"
        >
          <Icon name="more" size="sm" />
          <span class="text-xs">{{ t('common.more') }}</span>
        </button>
      </div>
    </template>

    <template #empty>
      <EmptyState
        :title="t('admin.users.noUsersYet')"
        :description="t('admin.users.createFirstUser')"
        :action-text="t('admin.users.createUser')"
        @action="emit('create')"
      />
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { AdminGroup, AdminUser, UserAttributeDefinition } from '@/types'
import type { BatchUserUsageStats } from '@/api/admin/dashboard'
import type { Column } from '@/components/common/types'
import { formatDateTime } from '@/utils/format'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import UserConcurrencyCell from '@/components/user/UserConcurrencyCell.vue'

defineProps<{
  columns: Column[]
  users: AdminUser[]
  loading: boolean
  attributeDefinitions: UserAttributeDefinition[]
  allGroups: AdminGroup[]
  usageStats: Record<string, BatchUserUsageStats>
  activeMenuId: number | null
  expandedGroupUserId: number | null
  getAttributeValue: (userId: number, attrId: number) => string
  getUserGroups: (user: AdminUser) => {
    exclusive: AdminGroup[]
    publicGroups: AdminGroup[]
  }
  getDaysRemaining: (expiresAt: string) => number
}>()

const emit = defineEmits<{
  create: []
  edit: [user: AdminUser]
  'toggle-status': [user: AdminUser]
  'open-menu': [user: AdminUser, event: MouseEvent]
  deposit: [user: AdminUser]
  'balance-history': [user: AdminUser]
  'toggle-expanded-group': [userId: number]
  'open-group-replace': [user: AdminUser, group: { id: number; name: string }]
}>()

const { t } = useI18n()
</script>
