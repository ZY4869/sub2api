<template>
  <DataTable :columns="columns" :data="groups" :loading="loading">
          <template #cell-name="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
          </template>

          <template #cell-platform="{ value }">
            <span
              :class="[
                'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium',
                value === 'anthropic'
                  ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
                  : value === 'kiro'
                    ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                  : value === 'openai'
                    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                    : value === 'grok'
                        ? 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-200'
                      : value === 'deepseek'
                        ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400'
                      : value === 'antigravity'
                        ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                        : 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
              ]"
            >
              <PlatformIcon :platform="value" size="xs" />
              {{ t('admin.groups.platforms.' + value) }}
            </span>
          </template>

          <template #cell-billing_type="{ row }">
            <div class="space-y-1">
              <span
                :class="[
                  'inline-block rounded-full px-2 py-0.5 text-xs font-medium',
                  row.subscription_type === 'subscription'
                    ? 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400'
                    : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
                ]"
              >
                {{
                  row.subscription_type === 'subscription'
                    ? t('admin.groups.subscription.subscription')
                    : t('admin.groups.subscription.standard')
                }}
              </span>
              <div
                v-if="row.subscription_type === 'subscription'"
                class="text-xs text-gray-500 dark:text-gray-400"
              >
                <template
                  v-if="row.daily_limit_usd || row.weekly_limit_usd || row.monthly_limit_usd"
                >
                  <span v-if="row.daily_limit_usd">${{ row.daily_limit_usd }}/{{ t('admin.groups.limitDay') }}</span>
                  <span
                    v-if="row.daily_limit_usd && (row.weekly_limit_usd || row.monthly_limit_usd)"
                    class="mx-1 text-gray-300 dark:text-gray-600"
                  >/</span>
                  <span v-if="row.weekly_limit_usd">${{ row.weekly_limit_usd }}/{{ t('admin.groups.limitWeek') }}</span>
                  <span
                    v-if="row.weekly_limit_usd && row.monthly_limit_usd"
                    class="mx-1 text-gray-300 dark:text-gray-600"
                  >/</span>
                  <span v-if="row.monthly_limit_usd">${{ row.monthly_limit_usd }}/{{ t('admin.groups.limitMonth') }}</span>
                </template>
                <span v-else class="text-gray-400 dark:text-gray-500">{{
                  t('admin.groups.subscription.noLimit')
                }}</span>








              </div>
            </div>
          </template>

          <template #cell-rate_multiplier="{ value }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ value }}x</span>
          </template>

          <template #cell-is_exclusive="{ value }">
            <span :class="['badge', value ? 'badge-primary' : 'badge-gray']">
              {{ value ? t('admin.groups.exclusive') : t('admin.groups.public') }}
            </span>
          </template>

          <template #cell-account_count="{ row }">
            <div class="flex min-w-[7.5rem] justify-end gap-3 text-right">
              <div>
                <div class="text-[1.55rem] font-bold leading-none text-emerald-600 dark:text-emerald-400">
                  {{ formatGroupAccountValue(getGroupAvailableAccounts(row), row) }}
                </div>
                <div class="mt-1 text-[11px] tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.accountsAvailable') }}
                </div>
              </div>
              <div>
                <div class="text-[1.55rem] font-bold leading-none text-rose-600 dark:text-rose-400">
                  {{ formatGroupAccountValue(row.rate_limited_account_count || 0, row) }}
                </div>
                <div class="mt-1 text-[11px] tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.accountsRateLimited') }}
                </div>
              </div>
              <div>
                <div class="text-[1.55rem] font-bold leading-none text-gray-700 dark:text-gray-200">
                  {{ formatGroupAccountValue(row.account_count || 0, row) }}
                </div>
                <div class="mt-1 text-[11px] tracking-wide text-gray-500 dark:text-gray-400">
                  {{ t('admin.groups.accountsTotal') }}
                </div>
              </div>
            </div>
          </template>

          <template #cell-capacity="{ row }">
            <GroupCapacityBadge
              v-if="capacityMap.get(row.id)"
              :concurrency-used="capacityMap.get(row.id)!.concurrencyUsed"
              :concurrency-max="capacityMap.get(row.id)!.concurrencyMax"
              :sessions-used="capacityMap.get(row.id)!.sessionsUsed"
              :sessions-max="capacityMap.get(row.id)!.sessionsMax"
              :rpm-used="capacityMap.get(row.id)!.rpmUsed"
              :rpm-max="capacityMap.get(row.id)!.rpmMax"
            />
            <span v-else class="text-xs text-gray-400">-</span>
          </template>

          <template #cell-usage="{ row }">
            <div v-if="usageLoading" class="text-xs text-gray-400">-</div>
            <div v-else class="space-y-0.5 text-xs">
              <div class="text-gray-500 dark:text-gray-400">
                <span class="text-gray-400 dark:text-gray-500">{{ t('admin.groups.usageToday') }}</span>
                <span class="ml-1 font-medium text-gray-700 dark:text-gray-300">${{ formatCost(usageMap.get(row.id)?.today_cost ?? 0) }}</span>
              </div>
              <div class="text-gray-500 dark:text-gray-400">
                <span class="text-gray-400 dark:text-gray-500">{{ t('admin.groups.usageTotal') }}</span>
                <span class="ml-1 font-medium text-gray-700 dark:text-gray-300">${{ formatCost(usageMap.get(row.id)?.total_cost ?? 0) }}</span>
              </div>
            </div>
          </template>

          <template #cell-status="{ value }">
            <span :class="['badge', value === 'active' ? 'badge-success' : 'badge-danger']">
              {{ t('admin.accounts.status.' + value) }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button
                @click="handleEdit(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
              >
                <Icon name="edit" size="sm" />
                <span class="text-xs">{{ t('common.edit') }}</span>
              </button>
              <button
                @click="handleRateMultipliers(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-purple-600 dark:hover:bg-dark-700 dark:hover:text-purple-400"
              >
                <Icon name="dollar" size="sm" />
                <span class="text-xs">{{ t('admin.groups.rateMultipliers') }}</span>
              </button>
              <button
                @click="handleDelete(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
              >
                <Icon name="trash" size="sm" />
                <span class="text-xs">{{ t('common.delete') }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.groups.noGroupsYet')"
              :description="t('admin.groups.createFirstGroup')"
              :action-text="t('admin.groups.createGroup')"
              @action="showCreateModal = true"
            />
          </template>
  </DataTable>
</template>

<script setup lang="ts">
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import GroupCapacityBadge from '@/components/common/GroupCapacityBadge.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ ctx: any }>()
const {
  t,
  columns,
  groups,
  loading,
  usageMap,
  usageLoading,
  capacityMap,
  showCreateModal,
  handleEdit,
  handleRateMultipliers,
  handleDelete,
  formatCost,
  getGroupAvailableAccounts,
  formatGroupAccountValue
} = props.ctx
</script>
