<template>
  <BaseDialog :show="!!account" :title="t('admin.accounts.daily5h.details')" width="wide" @close="$emit('close')">
    <div v-if="account" class="min-w-0 space-y-5 break-words">
      <div class="flex min-w-0 items-start gap-3">
        <PlatformTypeBadge :platform="account.platform" :type="account.type" />
        <div class="min-w-0">
          <div class="font-semibold [overflow-wrap:anywhere]">{{ account.name }}</div>
          <div class="text-xs text-gray-500">#{{ account.id }}</div>
        </div>
      </div>
      <div class="grid min-w-0 gap-4 sm:grid-cols-2">
        <div class="min-w-0"><div class="input-label">{{ t('admin.accounts.columns.status') }}</div><AccountStatusIndicator :account="account" /></div>
        <div class="min-w-0"><div class="input-label">{{ t('admin.accounts.columns.capacity') }}</div><AccountCapacityCell :account="account" /></div>
        <div class="min-w-0 sm:col-span-2"><div class="input-label">{{ t('admin.accounts.columns.groups') }}</div><AccountGroupsCell :groups="account.groups" :max-display="100" /></div>
        <div class="min-w-0"><div class="input-label">{{ t('admin.accounts.columns.usageWindows') }}</div><AccountUsageCell :account="account" :today-stats="todayStats" :today-stats-loading="false" :manual-refresh-token="0" /></div>
        <div class="min-w-0"><div class="input-label">{{ t('admin.accounts.columns.usageResetDates') }}</div><AccountUsageResetCell :account="account" /></div>
        <div v-if="todayStats" class="min-w-0 sm:col-span-2"><div class="input-label">{{ t('admin.accounts.columns.todayStats') }}</div><AccountTodayStatsCell :stats="todayStats" :loading="false" :error="null" /></div>
        <div v-for="field in fields" :key="field.key" class="min-w-0">
          <div class="input-label">{{ t(`admin.accounts.columns.${field.key}`) }}</div>
          <div class="text-sm [overflow-wrap:anywhere]">{{ field.value || '—' }}</div>
        </div>
      </div>
      <section class="min-w-0 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
        <h4 class="mb-3 font-semibold">{{ t('admin.accounts.daily5h.toolbarLabel') }}</h4>
        <div class="grid gap-3 text-sm sm:grid-cols-2">
          <div><div class="input-label">{{ t('admin.accounts.daily5h.summary') }}</div>{{ statusLabel }}</div>
          <div><div class="input-label">{{ t('admin.accounts.daily5h.checkedAt') }}</div>{{ formatTime(extra.daily_5h_trigger_last_checked_at) }}</div>
          <div><div class="input-label">{{ t('admin.accounts.daily5h.nextRetryAt') }}</div>{{ formatTime(extra.daily_5h_trigger_next_retry_at) }}</div>
          <div><div class="input-label">{{ t('admin.accounts.daily5h.attempts') }}</div>{{ extra.daily_5h_trigger_task_date || '—' }} · {{ extra.daily_5h_trigger_attempts || 0 }}/3</div>
          <div v-if="extra.daily_5h_trigger_last_model_id" class="min-w-0 sm:col-span-2">
            <div class="input-label">{{ t('admin.accounts.daily5h.model') }}</div>
            <div class="flex min-w-0 items-center gap-2"><ModelIcon :model="String(extra.daily_5h_trigger_last_model_id)" size="18px" /><span class="[overflow-wrap:anywhere]">{{ extra.daily_5h_trigger_last_model_id }}</span></div>
          </div>
          <p v-if="extra.daily_5h_trigger_last_summary" class="min-w-0 text-gray-500 [overflow-wrap:anywhere] sm:col-span-2">{{ extra.daily_5h_trigger_last_summary }}</p>
        </div>
      </section>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account, WindowStats } from '@/types'
import { formatDateTime, formatRelativeTime } from '@/utils/format'
import BaseDialog from '@/components/common/BaseDialog.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import AccountCapacityCell from '@/components/account/AccountCapacityCell.vue'
import AccountGroupsCell from '@/components/account/AccountGroupsCell.vue'
import AccountUsageCell from '@/components/account/AccountUsageCell.vue'
import AccountTodayStatsCell from '@/components/account/AccountTodayStatsCell.vue'
import AccountUsageResetCell from '@/components/account/AccountUsageResetCell.vue'

const props = defineProps<{ account: Account | null; todayStats?: WindowStats | null }>()
defineEmits<{ close: [] }>()
const { t } = useI18n()
const extra = computed(() => props.account?.extra || {})
const formatTime = (value: unknown) => typeof value === 'string' && value ? formatDateTime(value) : '—'
const statusLabel = computed(() => {
  const status = String(extra.value.daily_5h_trigger_last_status || '')
  return ['success', 'failed', 'skipped', 'waiting', 'running'].includes(status)
    ? t(`admin.accounts.daily5h.${status}`) : t('admin.accounts.daily5h.notRun')
})
const fields = computed(() => {
  const account = props.account
  if (!account) return []
  return [
    { key: 'proxy', value: account.proxy?.name },
    { key: 'schedulerScore', value: account.scheduler_score ? JSON.stringify(account.scheduler_score) : '—' },
    { key: 'schedulable', value: account.schedulable ? t('admin.accounts.schedulableEnabled') : t('admin.accounts.schedulableDisabled') },
    { key: 'priority', value: String(account.priority ?? 0) },
    { key: 'billingRateMultiplier', value: `${account.rate_multiplier ?? 1}x` },
    { key: 'lastUsed', value: formatRelativeTime(account.last_used_at) },
    { key: 'createdAt', value: formatTime(account.created_at) },
    { key: 'expiresAt', value: formatTime(account.expires_at) },
    { key: 'notes', value: account.notes },
  ]
})
</script>
