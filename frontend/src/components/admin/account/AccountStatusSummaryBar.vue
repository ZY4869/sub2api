<template>
  <div class="space-y-2">
    <div
      v-if="loading"
      class="grid grid-cols-3 gap-2 sm:grid-cols-5"
    >
      <div
        v-for="item in 5"
        :key="item"
        class="h-11 animate-pulse rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
      ></div>
    </div>

    <div
      v-else
      class="grid grid-cols-3 gap-2 sm:grid-cols-5"
    >
      <button
        v-for="card in cards"
        :key="card.key"
        type="button"
        class="rounded-xl border px-3 py-2 text-left shadow-sm transition"
        :class="[card.bgClass, cardClasses(card)]"
        @click="emit('select-status', card.statusValue)"
      >
        <div class="text-[10px] font-semibold uppercase tracking-wider" :class="card.eyebrowClass">
          {{ card.label }}
        </div>
        <div class="mt-0.5 text-lg font-semibold leading-none" :class="card.countClass">
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
    eyebrowClass: 'text-primary-600 dark:text-primary-400',
    countClass: 'text-primary-700 dark:text-primary-300',
    bgClass: 'bg-gradient-to-br from-primary-50 to-primary-100/60 dark:from-primary-950/40 dark:to-primary-900/20'
  },
  {
    key: 'active',
    label: t('admin.accounts.summary.active'),
    count: props.summary.by_status.active,
    statusValue: 'active',
    eyebrowClass: 'text-emerald-600 dark:text-emerald-400',
    countClass: 'text-emerald-700 dark:text-emerald-300',
    bgClass: 'bg-gradient-to-br from-emerald-50 to-emerald-100/60 dark:from-emerald-950/40 dark:to-emerald-900/20'
  },
  {
    key: 'error',
    label: t('admin.accounts.summary.error'),
    count: props.summary.by_status.error,
    statusValue: 'error',
    eyebrowClass: 'text-red-600 dark:text-red-400',
    countClass: 'text-red-700 dark:text-red-300',
    bgClass: 'bg-gradient-to-br from-red-50 to-red-100/60 dark:from-red-950/40 dark:to-red-900/20'
  },
  {
    key: 'rate_limited',
    label: t('admin.accounts.summary.rateLimited'),
    count: props.summary.rate_limited,
    statusValue: 'rate_limited',
    eyebrowClass: 'text-amber-600 dark:text-amber-400',
    countClass: 'text-amber-700 dark:text-amber-300',
    bgClass: 'bg-gradient-to-br from-amber-50 to-amber-100/60 dark:from-amber-950/40 dark:to-amber-900/20'
  },
  {
    key: 'paused',
    label: t('admin.accounts.summary.paused'),
    count: props.summary.paused,
    statusValue: 'paused',
    eyebrowClass: 'text-slate-600 dark:text-slate-400',
    countClass: 'text-slate-700 dark:text-slate-300',
    bgClass: 'bg-gradient-to-br from-slate-50 to-slate-100/60 dark:from-slate-800/40 dark:to-slate-700/20'
  }
])

const cardClasses = (card: { key: string; statusValue: string }) => {
  const isActive = card.statusValue === ''
    ? !props.activeStatus
    : props.activeStatus === card.statusValue
  return isActive
    ? 'border-primary-300 ring-2 ring-primary-400/50 dark:border-primary-500'
    : 'border-transparent hover:border-gray-300 dark:border-transparent dark:hover:border-dark-500'
}
</script>
