<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ t('purchase.resultTitle') }}
        </h1>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ t('purchase.resultDesc') }}
        </p>
      </div>

      <div v-if="loading" class="flex justify-center py-16">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div v-else-if="error" class="rounded-lg border border-red-200 bg-red-50 p-5 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200">
        <p class="font-medium">{{ t('purchase.resultFailed') }}</p>
        <p class="mt-1">{{ error }}</p>
        <button type="button" class="btn btn-secondary mt-4" @click="loadOrder">
          <Icon name="refresh" size="sm" class="mr-2" />
          {{ t('purchase.retry') }}
        </button>
      </div>

      <section v-else class="space-y-4 rounded-lg border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
        <PaymentStatusPanel :order="order" />
        <p class="text-sm text-gray-600 dark:text-dark-300">
          {{ resultMessage }}
        </p>
        <div class="flex flex-wrap gap-3">
          <button
            v-if="isRefreshable"
            type="button"
            class="btn btn-secondary"
            :disabled="refreshing"
            @click="refreshOrder"
          >
            <Icon name="refresh" size="sm" class="mr-2" />
            {{ t('purchase.refreshStatus') }}
          </button>
          <RouterLink v-if="resumeHref" :to="resumeHref" class="btn btn-primary">
            <Icon name="creditCard" size="sm" class="mr-2" />
            {{ t('purchase.resumePayment') }}
          </RouterLink>
          <button
            v-else-if="isRefreshable"
            type="button"
            class="btn btn-primary"
            :disabled="resuming"
            @click="resumeByOrderNo"
          >
            <Icon name="creditCard" size="sm" class="mr-2" />
            {{ resuming ? t('purchase.resumingPayment') : t('purchase.resumePayment') }}
          </button>
          <RouterLink to="/purchase" class="btn btn-secondary">
            <Icon name="arrowLeft" size="sm" class="mr-2" />
            {{ retryLabel }}
          </RouterLink>
        </div>
        <AirwallexPaymentElement
          v-if="canShowPaymentElement"
          class="pt-2"
          :order="resumeResult"
          @confirmed="refreshOrder"
        />
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
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
const resuming = ref(false)
const error = ref('')
const order = ref<PaymentOrder | null>(null)
const resumeResult = ref<PaymentResumeOrderResponse | null>(null)
let pollTimer: number | null = null

const orderNo = computed(() => String(route.params.orderNo || ''))
const resumeToken = computed(() => String(route.query.resume_token || route.query.resumeToken || ''))
const isRefreshable = computed(() => ['created', 'pending'].includes(order.value?.status || ''))
const resumeHref = computed(() =>
  resumeToken.value && isRefreshable.value
    ? `/payment/resume/${encodeURIComponent(resumeToken.value)}`
    : ''
)
const canShowPaymentElement = computed(() =>
  Boolean(resumeResult.value?.client_secret && isRefreshable.value)
)
const retryLabel = computed(() =>
  order.value?.status === 'failed' || order.value?.status === 'cancelled'
    ? t('purchase.retryPurchase')
    : t('purchase.backToPurchase')
)
const resultMessage = computed(() => {
  const status = order.value?.status
  if (status === 'paid') return t('purchase.resultPaid')
  if (status === 'failed') return t('purchase.resultFailedStatus')
  if (status === 'cancelled') return t('purchase.resultCancelled')
  if (status === 'expired') return t('purchase.resultExpired')
  if (status === 'refunded' || status === 'partial_refunded') return t('purchase.resultRefunded')
  return t('purchase.resultPending')
})

async function loadOrder() {
  loading.value = true
  error.value = ''
  try {
    order.value = await paymentAPI.getPaymentOrder(orderNo.value)
    schedulePoll()
  } catch (err) {
    error.value = resolveErrorMessage(err)
  } finally {
    loading.value = false
  }
}

async function refreshOrder() {
  refreshing.value = true
  error.value = ''
  try {
    order.value = await paymentAPI.getPaymentOrder(orderNo.value)
    if (resumeResult.value) {
      resumeResult.value = { ...resumeResult.value, order: order.value }
    }
    schedulePoll()
  } catch (err) {
    error.value = resolveErrorMessage(err)
  } finally {
    refreshing.value = false
  }
}

async function resumeByOrderNo() {
  if (!orderNo.value) return
  resuming.value = true
  error.value = ''
  try {
    resumeResult.value = await paymentAPI.resumePaymentOrderByOrderNo(orderNo.value)
    order.value = resumeResult.value.order
    schedulePoll()
  } catch (err) {
    error.value = resolveErrorMessage(err)
  } finally {
    resuming.value = false
  }
}

function schedulePoll() {
  clearPoll()
  if (!isRefreshable.value) return
  if (typeof window === 'undefined') return
  pollTimer = window.setTimeout(() => {
    void refreshOrder()
  }, 5000)
}

function clearPoll() {
  if (!pollTimer) return
  if (typeof window === 'undefined') {
    pollTimer = null
    return
  }
  window.clearTimeout(pollTimer)
  pollTimer = null
}

function resolveErrorMessage(err: unknown): string {
  if (err && typeof err === 'object' && 'message' in err) {
    return String((err as { message?: unknown }).message || t('purchase.resultFailed'))
  }
  return t('purchase.resultFailed')
}

onMounted(loadOrder)
onUnmounted(clearPoll)
</script>
