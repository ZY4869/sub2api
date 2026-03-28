<template>
  <div class="space-y-2">
    <div
      v-if="loading"
      class="grid grid-cols-3 gap-2 sm:grid-cols-7"
    >
      <div
        v-for="item in 7"
        :key="item"
        class="h-11 animate-pulse rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
      ></div>
    </div>

    <div
      v-else
      class="grid grid-cols-3 gap-2 sm:grid-cols-7"
    >
      <button
        v-for="card in cards"
        :key="card.key"
        type="button"
        class="flex items-center justify-between gap-3 rounded-xl border px-3 py-2 text-left shadow-sm transition"
        :class="[card.bgClass, cardClasses(card)]"
        :data-card-key="card.key"
        @click="handleCardClick(card)"
      >
        <div class="flex min-w-0 items-center gap-2">
          <span class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-white/70 shadow-sm ring-1 ring-white/80 dark:bg-white/10 dark:ring-white/10">
            <Icon :name="card.iconName" size="sm" :class="card.iconClass" :stroke-width="2" />
          </span>
          <span class="min-w-0 text-sm font-semibold leading-tight" :class="card.labelClass">
            {{ card.label }}
          </span>
        </div>
        <div class="shrink-0 text-lg font-semibold leading-none" :class="card.countClass">
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
import Icon from '@/components/icons/Icon.vue'
import type { AccountStatusSummary } from '@/types'

const props = withDefaults(defineProps<{
  summary: AccountStatusSummary
  loading?: boolean
  error?: string | null
  activeStatus?: string
  activeRuntimeView?: string
}>(), {
  loading: false,
  error: null,
  activeStatus: '',
  activeRuntimeView: 'all'
})

const emit = defineEmits<{
  'select-status': [status: string]
  'select-runtime-view': [runtimeView: string]
}>()

const { t } = useI18n()

type SummaryCard = {
  key: string
  label: string
  count: number
  statusValue: string
  runtimeValue?: string
  clickMode?: 'status' | 'runtime' | 'none'
  iconName: 'database' | 'sparkles' | 'exclamationTriangle' | 'bolt' | 'lock' | 'play' | 'checkCircle'
  iconClass: string
  labelClass: string
  countClass: string
  bgClass: string
}

const cards = computed<SummaryCard[]>(() => [
  {
    key: 'total',
    label: t('admin.accounts.summary.total'),
    count: props.summary.total,
    statusValue: '',
    clickMode: 'status',
    iconName: 'database',
    iconClass: 'text-primary-600 dark:text-primary-300',
    labelClass: 'text-primary-700 dark:text-primary-200',
    countClass: 'text-primary-700 dark:text-primary-300',
    bgClass: 'bg-gradient-to-br from-primary-50 to-primary-100/60 dark:from-primary-950/40 dark:to-primary-900/20'
  },
  {
    key: 'active',
    label: t('admin.accounts.summary.active'),
    count: props.summary.by_status.active,
    statusValue: 'active',
    clickMode: 'status',
    iconName: 'sparkles',
    iconClass: 'text-emerald-600 dark:text-emerald-300',
    labelClass: 'text-emerald-700 dark:text-emerald-200',
    countClass: 'text-emerald-700 dark:text-emerald-300',
    bgClass: 'bg-gradient-to-br from-emerald-50 to-emerald-100/60 dark:from-emerald-950/40 dark:to-emerald-900/20'
  },
  {
    key: 'remaining_available',
    label: t('admin.accounts.summary.remainingAvailable'),
    count: props.summary.remaining_available,
    statusValue: '',
    clickMode: 'none',
    iconName: 'checkCircle',
    iconClass: 'text-teal-600 dark:text-teal-300',
    labelClass: 'text-teal-700 dark:text-teal-200',
    countClass: 'text-teal-700 dark:text-teal-300',
    bgClass: 'bg-gradient-to-br from-teal-50 to-teal-100/60 dark:from-teal-950/40 dark:to-teal-900/20'
  },
  {
    key: 'in_use',
    label: t('admin.accounts.summary.inUse'),
    count: props.summary.in_use,
    statusValue: '',
    runtimeValue: 'in_use_only',
    clickMode: 'runtime',
    iconName: 'play',
    iconClass: 'text-sky-600 dark:text-sky-300',
    labelClass: 'text-sky-700 dark:text-sky-200',
    countClass: 'text-sky-700 dark:text-sky-300',
    bgClass: 'bg-gradient-to-br from-sky-50 to-sky-100/60 dark:from-sky-950/40 dark:to-sky-900/20'
  },
  {
    key: 'error',
    label: t('admin.accounts.summary.error'),
    count: props.summary.by_status.error,
    statusValue: 'error',
    clickMode: 'status',
    iconName: 'exclamationTriangle',
    iconClass: 'text-red-600 dark:text-red-300',
    labelClass: 'text-red-700 dark:text-red-200',
    countClass: 'text-red-700 dark:text-red-300',
    bgClass: 'bg-gradient-to-br from-red-50 to-red-100/60 dark:from-red-950/40 dark:to-red-900/20'
  },
  {
    key: 'rate_limited',
    label: t('admin.accounts.summary.rateLimited'),
    count: props.summary.rate_limited,
    statusValue: 'rate_limited',
    clickMode: 'status',
    iconName: 'bolt',
    iconClass: 'text-amber-600 dark:text-amber-300',
    labelClass: 'text-amber-700 dark:text-amber-200',
    countClass: 'text-amber-700 dark:text-amber-300',
    bgClass: 'bg-gradient-to-br from-amber-50 to-amber-100/60 dark:from-amber-950/40 dark:to-amber-900/20'
  },
  {
    key: 'paused',
    label: t('admin.accounts.summary.paused'),
    count: props.summary.paused,
    statusValue: 'paused',
    clickMode: 'status',
    iconName: 'lock',
    iconClass: 'text-slate-600 dark:text-slate-300',
    labelClass: 'text-slate-700 dark:text-slate-200',
    countClass: 'text-slate-700 dark:text-slate-300',
    bgClass: 'bg-gradient-to-br from-slate-50 to-slate-100/60 dark:from-slate-800/40 dark:to-slate-700/20'
  }
])

const cardClasses = (card: { key: string; statusValue: string; clickMode?: 'status' | 'runtime' | 'none' }) => {
  if (card.clickMode === 'none') {
    return 'border-transparent hover:border-transparent cursor-default'
  }
  const isActive = card.key === 'in_use'
    ? props.activeRuntimeView === 'in_use_only'
    : card.key === 'total'
      ? !props.activeStatus && props.activeRuntimeView !== 'in_use_only'
      : props.activeStatus === card.statusValue
  return isActive
    ? 'border-primary-300 ring-2 ring-primary-400/50 dark:border-primary-500'
    : 'border-transparent hover:border-gray-300 dark:border-transparent dark:hover:border-dark-500'
}

const handleCardClick = (card: SummaryCard) => {
  if (card.clickMode === 'none') {
    return
  }
  if (card.runtimeValue) {
    emit('select-runtime-view', card.runtimeValue)
    return
  }
  emit('select-status', card.statusValue)
}
</script>
