<template>
  <div class="space-y-2">
    <div
      v-if="loading"
      class="grid grid-cols-2 gap-2 sm:grid-cols-4"
    >
      <div
        v-for="item in 4"
        :key="item"
        class="h-11 animate-pulse rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
      ></div>
    </div>

    <div
      v-else
      class="grid grid-cols-2 gap-2 sm:grid-cols-4"
    >
      <button
        v-for="card in cards"
        :key="card.key"
        type="button"
        class="rounded-xl border bg-white px-3 py-2 text-left shadow-sm transition dark:bg-dark-800"
        :class="cardClasses(card)"
        @click="emit('select-reason', card.reasonValue)"
      >
        <div class="text-[10px] font-semibold uppercase tracking-wider" :class="card.eyebrowClass">
          {{ card.label }}
        </div>
        <div class="mt-0.5 text-lg font-semibold leading-none text-gray-900 dark:text-white">
          {{ card.count }}
        </div>
      </button>
    </div>

    <div v-if="error" class="text-sm text-red-500">
      {{ error }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountRateLimitReason, AccountStatusSummary } from '@/types'

type LimitedReasonCard = {
  key: string
  label: string
  count: number
  reasonValue: AccountRateLimitReason | ''
  eyebrowClass: string
}

const props = withDefaults(defineProps<{
  summary: AccountStatusSummary
  loading?: boolean
  error?: string | null
  activeReason?: AccountRateLimitReason | ''
}>(), {
  loading: false,
  error: null,
  activeReason: ''
})

const emit = defineEmits<{
  'select-reason': [reason: AccountRateLimitReason | '']
}>()

const { t } = useI18n()

const cards = computed<LimitedReasonCard[]>(() => [
  {
    key: 'all',
    label: t('admin.accounts.limited.summary.all'),
    count: props.summary.limited_breakdown.total,
    reasonValue: '',
    eyebrowClass: 'text-primary-600 dark:text-primary-400'
  },
  {
    key: 'rate_429',
    label: t('admin.accounts.limited.summary.rate429'),
    count: props.summary.limited_breakdown.rate_429,
    reasonValue: 'rate_429' as const,
    eyebrowClass: 'text-amber-600 dark:text-amber-400'
  },
  {
    key: 'usage_5h',
    label: t('admin.accounts.limited.summary.usage5h'),
    count: props.summary.limited_breakdown.usage_5h,
    reasonValue: 'usage_5h' as const,
    eyebrowClass: 'text-indigo-600 dark:text-indigo-400'
  },
  {
    key: 'usage_7d',
    label: t('admin.accounts.limited.summary.usage7d'),
    count: props.summary.limited_breakdown.usage_7d,
    reasonValue: 'usage_7d' as const,
    eyebrowClass: 'text-emerald-600 dark:text-emerald-400'
  }
])

const cardClasses = (card: { reasonValue: AccountRateLimitReason | '' }) => {
  const isActive = props.activeReason === card.reasonValue
  return isActive
    ? 'border-primary-300 ring-2 ring-primary-500/70 dark:border-primary-500'
    : 'border-gray-200 hover:border-gray-300 dark:border-dark-700 dark:hover:border-dark-500'
}
</script>
