<template>
  <div class="space-y-2">
    <div
      v-if="loading"
      class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5"
    >
      <div
        v-for="item in 5"
        :key="item"
        class="h-20 animate-pulse rounded-2xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
      ></div>
    </div>

    <div
      v-else
      class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5"
    >
      <button
        v-for="card in cards"
        :key="card.key"
        type="button"
        class="rounded-2xl border bg-white px-4 py-4 text-left shadow-sm transition hover:-translate-y-0.5 dark:bg-dark-800"
        :class="cardClasses(card)"
        @click="emit('select-status', card.statusValue)"
      >
        <div class="text-xs font-semibold uppercase tracking-[0.18em]" :class="card.eyebrowClass">
          {{ card.label }}
        </div>
        <div class="mt-3 text-3xl font-bold leading-none text-gray-900 dark:text-white">
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
import type { AccountStatusSummary } from '@/types'

const props = withDefaults(defineProps<{
  summary: AccountStatusSummary
  loading?: boolean
  error?: string | null
  activeStatus?: string
}>(), {
  loading: false,
  error: null,
  activeStatus: ''
})

const emit = defineEmits<{
  'select-status': [status: string]
}>()

const { t } = useI18n()

const cards = computed(() => [
  {
    key: 'total',
    label: t('admin.accounts.summary.total'),
    count: props.summary.total,
    statusValue: '',
    eyebrowClass: 'text-primary-600 dark:text-primary-400'
  },
  {
    key: 'active',
    label: t('admin.accounts.summary.active'),
    count: props.summary.by_status.active,
    statusValue: 'active',
    eyebrowClass: 'text-emerald-600 dark:text-emerald-400'
  },
  {
    key: 'error',
    label: t('admin.accounts.summary.error'),
    count: props.summary.by_status.error,
    statusValue: 'error',
    eyebrowClass: 'text-red-600 dark:text-red-400'
  },
  {
    key: 'rate_limited',
    label: t('admin.accounts.summary.rateLimited'),
    count: props.summary.rate_limited,
    statusValue: 'rate_limited',
    eyebrowClass: 'text-amber-600 dark:text-amber-400'
  },
  {
    key: 'paused',
    label: t('admin.accounts.summary.paused'),
    count: props.summary.paused,
    statusValue: 'paused',
    eyebrowClass: 'text-slate-600 dark:text-slate-300'
  }
])

const cardClasses = (card: { key: string; statusValue: string }) => {
  const isActive = card.statusValue === ''
    ? !props.activeStatus
    : props.activeStatus === card.statusValue
  return isActive
    ? 'border-primary-300 ring-2 ring-primary-500/70 dark:border-primary-500'
    : 'border-gray-200 hover:border-gray-300 dark:border-dark-700 dark:hover:border-dark-500'
}
</script>
