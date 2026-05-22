import { ref } from 'vue'
import type { PaymentCreateOrderResponse, PaymentResumeOrderResponse } from '@/types'

type AirwallexModule = typeof import('@airwallex/components-sdk')

interface AirwallexPaymentElementHandle {
  mount: (target: string | HTMLElement) => unknown
  confirm?: (options: { client_secret: string; intent_id?: string }) => Promise<unknown>
  unmount?: () => void
  destroy?: () => void
}

export function useAirwallexElements() {
  const mounted = ref(false)
  const loading = ref(false)
  const confirming = ref(false)
  const error = ref('')
  let element: AirwallexPaymentElementHandle | null = null

  async function mount(
    target: HTMLElement,
    order: PaymentCreateOrderResponse | PaymentResumeOrderResponse,
    locale: string
  ) {
    loading.value = true
    error.value = ''
    destroy()
    try {
      const sdk = (await import('@airwallex/components-sdk')) as AirwallexModule
      const env = order.provider_env === 'prod' ? 'prod' : 'demo'
      await sdk.init({
        env,
        locale: normalizeAirwallexLocale(locale),
        clientId: order.client_id,
        enabledElements: ['payments']
      })
      const created = (await sdk.createElement('card')) as AirwallexPaymentElementHandle | null
      if (!created || typeof created.mount !== 'function') {
        throw new Error('Airwallex card element is unavailable')
      }
      created.mount(target)
      element = created
      mounted.value = true
    } catch (err) {
      mounted.value = false
      error.value = err instanceof Error ? err.message : String(err)
    } finally {
      loading.value = false
    }
  }

  async function confirm(order: PaymentCreateOrderResponse | PaymentResumeOrderResponse) {
    if (!element?.confirm) {
      error.value = 'Airwallex confirm is unavailable'
      return null
    }
    confirming.value = true
    error.value = ''
    try {
      return await element.confirm({
        intent_id: order.intent_id,
        client_secret: order.client_secret
      })
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
      return null
    } finally {
      confirming.value = false
    }
  }

  function destroy() {
    try {
      element?.unmount?.()
      element?.destroy?.()
    } finally {
      element = null
      mounted.value = false
    }
  }

  return {
    mounted,
    loading,
    confirming,
    error,
    mount,
    confirm,
    destroy
  }
}

function normalizeAirwallexLocale(locale: string) {
  const normalized = locale.toLowerCase()
  if (normalized.startsWith('zh-hk')) return 'zh-HK'
  if (normalized.startsWith('zh')) return 'zh'
  if (normalized.startsWith('ja')) return 'ja'
  if (normalized.startsWith('ko')) return 'ko'
  if (normalized.startsWith('fr')) return 'fr'
  if (normalized.startsWith('es')) return 'es'
  if (normalized.startsWith('de')) return 'de'
  if (normalized.startsWith('it')) return 'it'
  if (normalized.startsWith('nl')) return 'nl'
  if (normalized.startsWith('pt')) return 'pt'
  if (normalized.startsWith('ru')) return 'ru'
  if (normalized.startsWith('da')) return 'da'
  if (normalized.startsWith('id')) return 'id'
  if (normalized.startsWith('ms')) return 'ms'
  if (normalized.startsWith('sv')) return 'sv'
  if (normalized.startsWith('pl')) return 'pl'
  if (normalized.startsWith('fi')) return 'fi'
  if (normalized.startsWith('ro')) return 'ro'
  if (normalized.startsWith('ar')) return 'ar'
  return 'en'
}
