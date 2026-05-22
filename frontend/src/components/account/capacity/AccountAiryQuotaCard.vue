<template>
  <AccountAiryCapacityMetricCard
    :label="metricLabel"
    :value="value"
    :title="tooltip"
    :tone="tone"
    :tag="resolvedLabel"
    :white-surface-enabled="whiteSurfaceEnabled"
  >
    <template #icon>
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8">
        <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z" />
      </svg>
    </template>
  </AccountAiryCapacityMetricCard>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import AccountAiryCapacityMetricCard from './AccountAiryCapacityMetricCard.vue'
import { formatQuotaCurrency } from './presentation'

const props = withDefaults(defineProps<{
  used: number
  limit: number
  kind?: 'daily' | 'weekly' | 'total'
  label?: string
  whiteSurfaceEnabled?: boolean
}>(), {
  kind: 'total',
  label: '',
  whiteSurfaceEnabled: false
})

const { t } = useI18n()

const resolvedKind = computed(() => props.kind || 'total')
const resolvedLabel = computed(() => {
  if (props.label) {
    return props.label
  }
  switch (resolvedKind.value) {
    case 'daily':
      return 'D'
    case 'weekly':
      return 'W'
    default:
      return 'T'
  }
})

const tone = computed(() => {
  if (props.used >= props.limit) {
    return 'danger'
  }
  if (props.used >= props.limit * 0.8) {
    return 'warning'
  }
  return 'safe'
})

const tooltip = computed(() => {
  if (props.used >= props.limit) {
    return t(`admin.accounts.capacity.quota.${resolvedKind.value}Exceeded`)
  }
  return t(`admin.accounts.capacity.quota.${resolvedKind.value}Normal`)
})

const value = computed(() => `$${formatQuotaCurrency(props.used)} / $${formatQuotaCurrency(props.limit)}`)
const metricLabel = computed(() => t('admin.accounts.capacity.cards.quota', { label: resolvedLabel.value }))
</script>
