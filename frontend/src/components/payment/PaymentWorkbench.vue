<template>
  <div class="grid min-h-0 grid-cols-1 gap-6 overflow-auto p-6 lg:grid-cols-[minmax(0,1fr)_minmax(320px,420px)]">
    <section class="space-y-5">
      <div>
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
          {{ t('purchase.workbenchTitle') }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
          {{ t('purchase.workbenchDesc') }}
        </p>
      </div>

      <div class="grid grid-cols-2 gap-3">
        <button
          type="button"
          class="product-tab"
          :class="productType === 'balance_topup' && 'product-tab-active'"
          @click="productType = 'balance_topup'"
        >
          <Icon name="dollar" size="md" />
          <span>{{ t('purchase.balanceTopup') }}</span>
        </button>
        <button
          type="button"
          class="product-tab"
          :class="productType === 'subscription' && 'product-tab-active'"
          @click="productType = 'subscription'"
        >
          <Icon name="calendar" size="md" />
          <span>{{ t('purchase.subscription') }}</span>
        </button>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <label class="space-y-2">
          <span class="text-sm font-medium text-gray-700 dark:text-dark-200">
            {{ t('purchase.currency') }}
          </span>
          <select v-model="selectedCurrency" class="input">
            <option v-for="currency in allowedCurrencies" :key="currency" :value="currency">
              {{ currency }}
            </option>
          </select>
        </label>
        <label class="space-y-2">
          <span class="text-sm font-medium text-gray-700 dark:text-dark-200">
            {{ t('purchase.countryCode') }}
          </span>
          <input
            v-model.trim="countryCode"
            type="text"
            maxlength="2"
            class="input uppercase"
            :placeholder="t('purchase.countryCodePlaceholder')"
          />
        </label>
      </div>

      <label v-if="productType === 'balance_topup'" class="space-y-2">
        <span class="text-sm font-medium text-gray-700 dark:text-dark-200">
          {{ t('purchase.topupAmount') }}
        </span>
        <input
          v-model.number="topupAmount"
          type="number"
          class="input"
          :min="settings.payment_min_topup_amount"
          :max="settings.payment_max_topup_amount"
          step="0.01"
        />
        <span class="block text-xs text-gray-500 dark:text-dark-400">
          {{ t('purchase.topupRange', { min: settings.payment_min_topup_amount, max: settings.payment_max_topup_amount }) }}
        </span>
      </label>

      <div v-else class="space-y-3">
        <p class="text-sm font-medium text-gray-700 dark:text-dark-200">
          {{ t('purchase.plan') }}
        </p>
        <div v-if="enabledPlans.length" class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <button
            v-for="plan in enabledPlans"
            :key="plan.plan_id"
            type="button"
            class="plan-card"
            :class="selectedPlanId === plan.plan_id && 'plan-card-active'"
            @click="selectedPlanId = plan.plan_id"
          >
            <span class="font-semibold text-gray-900 dark:text-white">{{ plan.name }}</span>
            <span class="text-xs text-gray-500 dark:text-dark-400">
              {{ t('purchase.validityDays', { days: plan.validity_days }) }}
            </span>
            <span class="mt-2 text-sm font-semibold text-primary-600 dark:text-primary-300">
              {{ formatAmount(plan.prices_by_currency[selectedCurrency] || 0, selectedCurrency) }}
            </span>
          </button>
        </div>
        <p v-else class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
          {{ t('purchase.noPlans') }}
        </p>
      </div>

      <div v-if="error" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200">
        {{ t('purchase.orderFailed') }}
      </div>

      <div class="flex flex-wrap gap-3">
        <button type="button" class="btn btn-primary" :disabled="!canCreate" @click="createOrder">
          <Icon name="creditCard" size="sm" class="mr-2" />
          {{ creating ? t('purchase.creating') : t('purchase.createOrder') }}
        </button>
        <button v-if="order" type="button" class="btn btn-secondary" :disabled="refreshing" @click="refreshOrder">
          <Icon name="refresh" size="sm" class="mr-2" />
          {{ t('purchase.refreshStatus') }}
        </button>
        <button v-if="order && order.status === 'pending'" type="button" class="btn btn-secondary" :disabled="cancelling" @click="cancelOrder">
          <Icon name="x" size="sm" class="mr-2" />
          {{ t('purchase.cancelOrder') }}
        </button>
      </div>
    </section>

    <aside class="space-y-4">
      <PaymentStatusPanel :order="order" />
      <AirwallexPaymentElement v-if="createResult" :order="createResult" @confirmed="refreshOrder" />
      <div v-else class="rounded-lg border border-dashed border-gray-200 p-5 text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
        {{ t('purchase.createFirst') }}
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { usePaymentWorkbench } from '@/composables/usePaymentWorkbench'
import Icon from '@/components/icons/Icon.vue'
import AirwallexPaymentElement from './AirwallexPaymentElement.vue'
import PaymentStatusPanel from './PaymentStatusPanel.vue'
import type { PublicSettings } from '@/types'

const props = defineProps<{
  settings: PublicSettings
}>()

const { t, locale } = useI18n()
const {
  productType,
  selectedCurrency,
  selectedPlanId,
  topupAmount,
  countryCode,
  creating,
  refreshing,
  cancelling,
  error,
  createResult,
  order,
  allowedCurrencies,
  enabledPlans,
  canCreate,
  createOrder,
  refreshOrder,
  cancelOrder
} = usePaymentWorkbench(() => props.settings)

function formatAmount(amount: number, currency: string): string {
  if (!amount || !currency) return t('purchase.currencyUnavailable')
  try {
    return new Intl.NumberFormat(locale.value, { style: 'currency', currency }).format(amount)
  } catch {
    return `${amount.toFixed(2)} ${currency}`
  }
}
</script>

<style scoped>
.product-tab {
  @apply flex h-14 items-center justify-center gap-2 rounded-lg border border-gray-200 bg-white px-4 text-sm font-semibold text-gray-700 transition hover:border-primary-300 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-200;
}

.product-tab-active {
  @apply border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-500/15 dark:text-primary-200;
}

.plan-card {
  @apply flex min-h-28 flex-col rounded-lg border border-gray-200 bg-white p-4 text-left transition hover:border-primary-300 dark:border-dark-700 dark:bg-dark-800;
}

.plan-card-active {
  @apply border-primary-500 ring-2 ring-primary-500/20 dark:border-primary-400;
}
</style>
