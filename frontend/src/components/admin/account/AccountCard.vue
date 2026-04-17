<template>
  <article class="card card-hover relative overflow-hidden p-4">
    <div class="absolute left-4 top-4">
      <input
        type="checkbox"
        class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        :checked="selected"
        @change="emit('toggle-selected', account.id)"
      />
    </div>

    <div class="pl-8">
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0">
          <div class="flex items-center gap-2">
            <div class="truncate text-base font-semibold text-gray-900 dark:text-white">
              {{ account.name }}
            </div>
            <span
              v-if="showAutoRecoverySuccess"
              class="inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-500/15 dark:text-emerald-300"
              :title="t('admin.accounts.autoRecoveryProbe.successIndicator')"
              :aria-label="t('admin.accounts.autoRecoveryProbe.successIndicator')"
            >
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.4">
                <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
              </svg>
            </span>
          </div>
          <div
            v-if="email"
            class="truncate text-xs text-gray-500 dark:text-gray-400"
            :title="email"
          >
            {{ email }}
          </div>
        </div>
        <PlatformTypeBadge
          :platform="account.platform"
          :gateway-protocol="account.gateway_protocol"
          :type="account.type"
          :plan-type="String(account.credentials?.plan_type || '') || undefined"
          :privacy-mode="String(account.extra?.privacy_mode || '') || undefined"
          :subscription-expires-at="
            String(account.credentials?.subscription_expires_at || '') || undefined
          "
        />
      </div>

      <div class="mt-4">
        <AccountStatusIndicator :account="account" @show-temp-unsched="emit('show-temp-unsched', account)" />
      </div>

      <AccountAutoRecoveryProbeNotice
        v-if="account.auto_recovery_probe"
        class="mt-4"
        :summary="account.auto_recovery_probe"
      />

      <div class="mt-4 grid gap-3 sm:grid-cols-2">
        <div class="rounded-xl bg-gray-50 px-3 py-3 dark:bg-dark-900/40">
          <div class="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.columns.capacity') }}
          </div>
          <AccountCapacityCell :account="account" />
        </div>
        <div class="rounded-xl bg-gray-50 px-3 py-3 dark:bg-dark-900/40">
          <div class="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.columns.lastUsed') }}
          </div>
          <div class="text-sm text-gray-700 dark:text-gray-200">
            {{ formatRelativeTime(account.last_used_at) }}
          </div>
        </div>
      </div>

      <div class="mt-4 rounded-xl border border-gray-100 px-3 py-3 dark:border-dark-700">
        <div class="mb-2 text-xs font-semibold uppercase tracking-[0.16em] text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.columns.usageWindows') }}
        </div>
        <AccountUsageCell
          :account="account"
          :today-stats="todayStatsByAccountId[String(account.id)] ?? null"
          :today-stats-loading="todayStatsLoading"
          :manual-refresh-token="usageManualRefreshToken"
        />
      </div>

      <div v-if="account.groups?.length" class="mt-4">
        <AccountGroupsCell :groups="account.groups" :max-display="4" />
      </div>

      <div class="mt-4 flex items-center justify-between gap-3">
        <button
          class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 dark:focus:ring-offset-dark-800"
          :class="[
            account.schedulable
              ? 'bg-primary-500 hover:bg-primary-600'
              : 'bg-gray-200 hover:bg-gray-300 dark:bg-dark-600 dark:hover:bg-dark-500'
          ]"
          :disabled="togglingSchedulable === account.id"
          :title="account.schedulable ? t('admin.accounts.schedulableEnabled') : t('admin.accounts.schedulableDisabled')"
          @click="emit('toggle-schedulable', account)"
        >
          <span
            class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
            :class="[account.schedulable ? 'translate-x-4' : 'translate-x-0']"
          />
        </button>

        <AccountsViewRowActions
          @edit="emit('edit', account)"
          @delete="emit('delete', account)"
          @more="emit('open-menu', { account, event: $event })"
        />
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import AccountCapacityCell from '@/components/account/AccountCapacityCell.vue'
import AccountGroupsCell from '@/components/account/AccountGroupsCell.vue'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import AccountUsageCell from '@/components/account/AccountUsageCell.vue'
import type { Account, WindowStats } from '@/types'
import { formatRelativeTime } from '@/utils/format'
import AccountAutoRecoveryProbeNotice from './AccountAutoRecoveryProbeNotice.vue'
import AccountsViewRowActions from './AccountsViewRowActions.vue'

const props = defineProps<{
  account: Account
  selected: boolean
  togglingSchedulable: number | null
  todayStatsByAccountId: Record<string, WindowStats>
  todayStatsLoading: boolean
  usageManualRefreshToken: number
}>()

const emit = defineEmits<{
  'toggle-selected': [id: number]
  'show-temp-unsched': [account: Account]
  'toggle-schedulable': [account: Account]
  edit: [account: Account]
  delete: [account: Account]
  'open-menu': [payload: { account: Account; event: MouseEvent }]
}>()

const { t } = useI18n()

const email = computed(() => String(props.account.extra?.email_address || '').trim())
const showAutoRecoverySuccess = computed(() => props.account.auto_recovery_probe?.status === 'success')
</script>
