<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ t('purchase.resumeTitle') }}
        </h1>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ t('purchase.resumeDesc') }}
        </p>
      </div>

      <div v-if="loading" class="flex justify-center py-16">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div v-else-if="error" class="rounded-lg border border-red-200 bg-red-50 p-5 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200">
        <p class="font-medium">{{ t('purchase.resumeFailed') }}</p>
        <p class="mt-1">{{ error }}</p>
        <button type="button" class="btn btn-secondary mt-4" @click="loadOrder">
          <Icon name="refresh" size="sm" class="mr-2" />
          {{ t('purchase.retry') }}
        </button>
      </div>

      <div v-else class="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(320px,420px)]">
        <section class="space-y-4 rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
          <PaymentStatusPanel :order="order" />
          <div class="flex flex-wrap gap-3">
            <button type="button" class="btn btn-secondary" :disabled="refreshing" @click="refreshOrder">
              <Icon name="refresh" size="sm" class="mr-2" />
              {{ t('purchase.refreshStatus') }}
            </button>
            <RouterLink to="/purchase" class="btn btn-secondary">
              <Icon name="creditCard" size="sm" class="mr-2" />
              {{ t('purchase.backToPurchase') }}
            </RouterLink>
          </div>
        </section>

        <aside class="space-y-4">
          <AirwallexPaymentElement
            v-if="canResumePayment"
            :order="resumeResult"
            @confirmed="refreshOrder"
          />
          <div v-else class="rounded-lg border border-gray-200 bg-gray-50 p-5 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-800/60 dark:text-dark-300">
            {{ terminalMessage }}
          </div>
        </aside>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import AirwallexPaymentElement from '@/components/payment/AirwallexPaymentElement.vue'
import PaymentStatusPanel from '@/components/payment/PaymentStatusPanel.vue'
import { paymentAPI } from '@/api/payment'
import type { PaymentOrder, PaymentResumeOrderResponse } from '@/types'

const route = useRoute()
const { t } = useI18n()

const loading = ref(false)
const refreshing = ref(false)
const error = ref('')
const resumeResult = ref<PaymentResumeOrderResponse | null>(null)
const order = ref<PaymentOrder | null>(null)

const resumeToken = computed(() => String(route.params.resumeToken || ''))
const canResumePayment = computed(() =>
  Boolean(resumeResult.value?.client_secret && ['created', 'pending'].includes(order.value?.status || ''))
)
const terminalMessage = computed(() => {
  const status = order.value?.status
  if (status === 'paid') return t('purchase.resumePaid')
  if (status === 'failed') return t('purchase.resumeFailedStatus')
  if (status === 'cancelled') return t('purchase.resumeCancelled')
  if (status === 'refunded' || status === 'partial_refunded') return t('purchase.resumeRefunded')
  return t('purchase.resumeUnavailable')
})

async function loadOrder() {
  loading.value = true
  error.value = ''
  try {
    resumeResult.value = await paymentAPI.resumePaymentOrder(resumeToken.value)
    order.value = resumeResult.value.order
  } catch (err) {
    error.value = resolveErrorMessage(err)
  } finally {
    loading.value = false
  }
}

async function refreshOrder() {
  if (!order.value) return
  refreshing.value = true
  error.value = ''
  try {
    order.value = await paymentAPI.getPaymentOrder(order.value.order_no)
    if (resumeResult.value) {
      resumeResult.value = { ...resumeResult.value, order: order.value }
    }
  } catch (err) {
    error.value = resolveErrorMessage(err)
  } finally {
    refreshing.value = false
  }
}

function resolveErrorMessage(err: unknown): string {
  if (err && typeof err === 'object' && 'message' in err) {
    return String((err as { message?: unknown }).message || t('purchase.resumeFailed'))
  }
  return t('purchase.resumeFailed')
}

onMounted(loadOrder)
</script>
