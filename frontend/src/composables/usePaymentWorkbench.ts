import { computed, ref, watch } from 'vue'
import { paymentAPI } from '@/api/payment'
import type {
  PaymentCreateOrderResponse,
  PaymentOrder,
  PaymentProductType,
  PaymentSubscriptionPlan,
  PublicSettings
} from '@/types'

function randomIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `pay-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export function usePaymentWorkbench(settings: () => PublicSettings | null) {
  const productType = ref<PaymentProductType>('balance_topup')
  const selectedCurrency = ref('')
  const selectedPlanId = ref('')
  const topupAmount = ref(10)
  const countryCode = ref('')
  const creating = ref(false)
  const refreshing = ref(false)
  const cancelling = ref(false)
  const error = ref('')
  const createResult = ref<PaymentCreateOrderResponse | null>(null)
  const order = ref<PaymentOrder | null>(null)

  const allowedCurrencies = computed(() => {
    const currencies = settings()?.payment_allowed_currencies || []
    return currencies.length > 0 ? currencies : ['USD', 'CNY', 'HKD']
  })

  const enabledPlans = computed<PaymentSubscriptionPlan[]>(() =>
    (settings()?.payment_subscription_plans || []).filter((plan) => plan.enabled)
  )

  const selectedPlan = computed(() =>
    enabledPlans.value.find((plan) => plan.plan_id === selectedPlanId.value) || null
  )

  const payableAmount = computed(() => {
    if (productType.value === 'balance_topup') return topupAmount.value
    const plan = selectedPlan.value
    return plan ? planPriceForCurrency(plan, selectedCurrency.value) : 0
  })

  function planPriceForCurrency(plan: PaymentSubscriptionPlan, currency: string): number {
    const normalizedCurrency = currency.trim().toUpperCase()
    const directPrice = plan.prices_by_currency?.[normalizedCurrency] || 0
    if (directPrice > 0) return directPrice
    const rate = Number(settings()?.payment_subscription_usd_to_cny_rate) || 0
    const usdPrice = plan.prices_by_currency?.USD || 0
    if (normalizedCurrency !== 'CNY' || rate <= 0 || usdPrice <= 0) return 0
    return Math.round(usdPrice * rate * 100) / 100
  }

  const canCreate = computed(() => {
    if (!selectedCurrency.value || creating.value) return false
    if (productType.value === 'balance_topup') return topupAmount.value > 0
    return Boolean(selectedPlan.value && payableAmount.value > 0)
  })

  watch(
    allowedCurrencies,
    (currencies) => {
      const preferred = settings()?.payment_default_currency || currencies[0]
      selectedCurrency.value = currencies.includes(selectedCurrency.value)
        ? selectedCurrency.value
        : currencies.includes(preferred)
          ? preferred
          : currencies[0]
    },
    { immediate: true }
  )

  watch(
    enabledPlans,
    (plans) => {
      if (selectedPlanId.value && plans.some((plan) => plan.plan_id === selectedPlanId.value)) return
      selectedPlanId.value = plans[0]?.plan_id || ''
    },
    { immediate: true }
  )

  watch(productType, () => {
    error.value = ''
    createResult.value = null
    order.value = null
  })

  async function createOrder() {
    if (!canCreate.value) return
    creating.value = true
    error.value = ''
    try {
      const payload =
        productType.value === 'balance_topup'
          ? {
              product_type: productType.value,
              amount: topupAmount.value,
              currency: selectedCurrency.value,
              country_code: countryCode.value,
              return_url: buildPaymentResultURL()
            }
          : {
              product_type: productType.value,
              plan_id: selectedPlanId.value,
              currency: selectedCurrency.value,
              country_code: countryCode.value,
              return_url: buildPaymentResultURL()
            }
      createResult.value = await paymentAPI.createPaymentOrder(payload, randomIdempotencyKey())
      order.value = createResult.value.order
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
    } finally {
      creating.value = false
    }
  }

  async function refreshOrder() {
    if (!order.value) return
    refreshing.value = true
    try {
      order.value = await paymentAPI.getPaymentOrder(order.value.order_no)
      if (createResult.value) {
        createResult.value = { ...createResult.value, order: order.value }
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
    } finally {
      refreshing.value = false
    }
  }

  async function cancelOrder() {
    if (!order.value) return
    cancelling.value = true
    try {
      await paymentAPI.cancelPaymentOrder(order.value.order_no)
      await refreshOrder()
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
    } finally {
      cancelling.value = false
    }
  }

  return {
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
    selectedPlan,
    payableAmount,
    planPriceForCurrency,
    canCreate,
    createOrder,
    refreshOrder,
    cancelOrder
  }
}

function buildPaymentResultURL(): string | undefined {
  if (typeof window === 'undefined') return undefined
  return `${window.location.origin}/payment/result/__ORDER_NO__`
}
