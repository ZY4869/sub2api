<template>
  <div
    v-if="visibleSummary"
    class="inline-flex items-center"
  >
    <AccountErrorTooltipButton
      :message="detailText"
      :ariaLabel="headline"
      :button-class="buttonClass"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountAutoRecoveryProbeSummary } from '@/types'
import AccountErrorTooltipButton from '@/components/account/AccountErrorTooltipButton.vue'
import { formatDateTime } from '@/utils/format'
import { resolveAccountAiryIssueSummary } from './accountAiryIssueText'
import type { AiryStatusKind } from './accountAiryStatusTypes'

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

const summaryKind = computed<AiryStatusKind>(() => {
  if (statusKey.value === 'blacklisted') return 'banned'
  if (statusKey.value === 'retry_scheduled') return 'syncing'
  return 'error'
})

const headline = computed(() =>
  t('admin.accounts.autoRecoveryProbe.headline', {
    status: t(`admin.accounts.autoRecoveryProbe.statuses.${statusKey.value}`)
  })
)

const summaryText = computed(() => {
  const text = String(visibleSummary.value?.summary || '').trim()
  if (text || visibleSummary.value?.error_code) {
    const summary = resolveAccountAiryIssueSummary(summaryKind.value, [
      text,
      visibleSummary.value?.error_code,
    ])
    if (summary.helperKey) return t(summary.helperKey)
    if (summary.helper) return summary.helper
  }
  return t(`admin.accounts.autoRecoveryProbe.summaries.${statusKey.value}`)
})

const detailText = computed(() => {
  if (!visibleSummary.value) return ''
  const lines = [
    headline.value,
    summaryText.value,
  ]
  if (visibleSummary.value.blacklisted) {
    lines.push(t('admin.accounts.autoRecoveryProbe.autoBlacklisted'))
  }
  if (visibleSummary.value.checked_at) {
    lines.push(t('admin.accounts.autoRecoveryProbe.checkedAt', {
      time: formatDateTime(visibleSummary.value.checked_at)
    }))
  }
  if (visibleSummary.value.next_retry_at) {
    lines.push(t('admin.accounts.autoRecoveryProbe.nextRetryAt', {
      time: formatDateTime(visibleSummary.value.next_retry_at)
    }))
  }
  if (visibleSummary.value.error_code) {
    lines.push(t('admin.accounts.autoRecoveryProbe.errorCode', {
      code: visibleSummary.value.error_code
    }))
  }
  return lines.filter(Boolean).join('\n')
})

const buttonClass = computed(() => {
  if (visibleSummary.value?.blacklisted || visibleSummary.value?.status === 'blacklisted') {
    return 'rounded-full border border-red-200/80 bg-red-50 px-1.5 py-1 text-red-600 transition hover:text-red-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-red-400/60 dark:border-red-400/20 dark:bg-red-500/10 dark:text-red-200 dark:hover:text-red-100'
  }
  if (visibleSummary.value?.status === 'retry_scheduled') {
    return 'rounded-full border border-amber-200/80 bg-amber-50 px-1.5 py-1 text-amber-600 transition hover:text-amber-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-amber-400/60 dark:border-amber-400/20 dark:bg-amber-500/10 dark:text-amber-100 dark:hover:text-amber-50'
  }
  return 'rounded-full border border-rose-200/80 bg-rose-50 px-1.5 py-1 text-rose-600 transition hover:text-rose-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-rose-400/60 dark:border-rose-400/20 dark:bg-rose-500/10 dark:text-rose-200 dark:hover:text-rose-100'
})
</script>
