<template>
  <div
    v-if="visibleSummary"
    class="rounded-xl border px-3 py-2 text-xs"
    :class="containerClass"
  >
    <div class="flex items-center gap-2">
      <span class="font-semibold">{{ headline }}</span>
      <span
        v-if="visibleSummary?.blacklisted"
        class="rounded-full px-2 py-0.5 text-[10px] font-semibold"
        :class="badgeClass"
      >
        {{ t('admin.accounts.autoRecoveryProbe.autoBlacklisted') }}
      </span>
    </div>
    <p class="mt-1 leading-5">{{ summaryText }}</p>
    <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-[11px] opacity-80">
      <span v-if="visibleSummary?.checked_at">
        {{ t('admin.accounts.autoRecoveryProbe.checkedAt', { time: formatDateTime(visibleSummary.checked_at) }) }}
      </span>
      <span v-if="visibleSummary?.next_retry_at">
        {{ t('admin.accounts.autoRecoveryProbe.nextRetryAt', { time: formatDateTime(visibleSummary.next_retry_at) }) }}
      </span>
      <span v-if="visibleSummary?.error_code">
        {{ t('admin.accounts.autoRecoveryProbe.errorCode', { code: visibleSummary.error_code }) }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountAutoRecoveryProbeSummary } from '@/types'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{
  summary?: AccountAutoRecoveryProbeSummary | null
  lifecycleState?: string | null
}>()

const { t } = useI18n()
const hasRestoredFromBlacklisted = computed(() => {
  const lifecycleState = String(props.lifecycleState || '').trim().toLowerCase()
  if (!lifecycleState || lifecycleState === 'blacklisted') {
    return false
  }
  return props.summary?.blacklisted || props.summary?.status === 'blacklisted'
})

const visibleSummary = computed(() =>
  props.summary && props.summary.status !== 'success' && !hasRestoredFromBlacklisted.value
    ? props.summary
    : null
)

const statusKey = computed(() => {
  switch (visibleSummary.value?.status) {
    case 'success':
    case 'retry_scheduled':
    case 'blacklisted':
      return visibleSummary.value.status
    default:
      return 'unknown'
  }
})

const headline = computed(() =>
  t('admin.accounts.autoRecoveryProbe.headline', {
    status: t(`admin.accounts.autoRecoveryProbe.statuses.${statusKey.value}`)
  })
)

const summaryText = computed(() => {
  const text = String(visibleSummary.value?.summary || '').trim()
  if (text) {
    return text
  }
  return t(`admin.accounts.autoRecoveryProbe.summaries.${statusKey.value}`)
})

const containerClass = computed(() => {
  if (visibleSummary.value?.blacklisted || visibleSummary.value?.status === 'blacklisted') {
    return 'border-red-200 bg-red-50 text-red-800 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200'
  }
  if (visibleSummary.value?.status === 'retry_scheduled') {
    return 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-100'
  }
  return 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-100'
})

const badgeClass = computed(() => {
  if (visibleSummary.value?.blacklisted || visibleSummary.value?.status === 'blacklisted') {
    return 'bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-100'
  }
  return 'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-100'
})
</script>
