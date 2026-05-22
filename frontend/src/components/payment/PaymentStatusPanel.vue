<template>
  <div v-if="order" class="rounded-lg border border-gray-100 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/60">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <p class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('purchase.orderStatus') }}
        </p>
        <p class="mt-1 font-mono text-xs text-gray-500 dark:text-dark-400">
          {{ order.order_no }}
        </p>
      </div>
      <span :class="statusClass">
        {{ t(`purchase.status.${order.status}`) }}
      </span>
    </div>

    <dl class="mt-4 grid grid-cols-2 gap-3 text-sm">
      <div>
        <dt class="text-gray-500 dark:text-dark-400">{{ t('purchase.amount') }}</dt>
        <dd class="mt-1 font-semibold text-gray-900 dark:text-white">
          {{ amountText }}
        </dd>
      </div>
      <div>
        <dt class="text-gray-500 dark:text-dark-400">{{ t('purchase.provider') }}</dt>
        <dd class="mt-1 font-semibold text-gray-900 dark:text-white">
          {{ order.provider }} · {{ order.provider_env }}
        </dd>
      </div>
    </dl>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PaymentOrder } from '@/types'

const props = defineProps<{
  order: PaymentOrder | null
}>()

const { t, locale } = useI18n()

const amountText = computed(() => {
  if (!props.order) return ''
  try {
    return new Intl.NumberFormat(locale.value, {
      style: 'currency',
      currency: props.order.currency
    }).format(props.order.amount)
  } catch {
    return `${props.order.amount.toFixed(2)} ${props.order.currency}`
  }
})

const statusClass = computed(() => {
  const base = 'inline-flex rounded-full px-2.5 py-1 text-xs font-semibold'
  switch (props.order?.status) {
    case 'paid':
      return `${base} bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300`
    case 'failed':
    case 'cancelled':
    case 'expired':
      return `${base} bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300`
    default:
      return `${base} bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300`
  }
})
</script>
