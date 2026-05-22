import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import PurchaseSubscriptionView from '../PurchaseSubscriptionView.vue'

const createPaymentOrderMock = vi.hoisted(() => vi.fn())

const testState = vi.hoisted(() => ({
  appStoreState: {
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
    cachedPublicSettings: {
      purchase_subscription_enabled: true,
      purchase_subscription_url: 'https://pay.example.com/checkout?plan=pro',
      payment_provider_airwallex_enabled: false,
      payment_allowed_currencies: ['USD', 'CNY', 'HKD'],
      payment_default_currency: 'USD',
      payment_min_topup_amount: 1,
      payment_max_topup_amount: 5000,
      payment_subscription_plans: [],
    },
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN' },
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => testState.appStoreState,
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    createPaymentOrder: createPaymentOrderMock,
    getPaymentOrder: vi.fn(),
    resumePaymentOrderByOrderNo: vi.fn(),
    cancelPaymentOrder: vi.fn(),
  },
}))

vi.mock('@/components/payment/AirwallexPaymentElement.vue', () => ({
  default: { template: '<div data-test="airwallex-element" />' },
}))

describe('PurchaseSubscriptionView', () => {
  beforeEach(() => {
    testState.appStoreState.fetchPublicSettings.mockReset()
    createPaymentOrderMock.mockReset()
    testState.appStoreState.publicSettingsLoaded = true
    testState.appStoreState.cachedPublicSettings.purchase_subscription_enabled = true
    testState.appStoreState.cachedPublicSettings.purchase_subscription_url = 'https://pay.example.com/checkout?plan=pro'
    testState.appStoreState.cachedPublicSettings.payment_provider_airwallex_enabled = false
    testState.appStoreState.cachedPublicSettings.payment_allowed_currencies = ['USD', 'CNY', 'HKD']
    testState.appStoreState.cachedPublicSettings.payment_default_currency = 'USD'
    testState.appStoreState.cachedPublicSettings.payment_min_topup_amount = 1
    testState.appStoreState.cachedPublicSettings.payment_max_topup_amount = 5000
    testState.appStoreState.cachedPublicSettings.payment_subscription_plans = []
    document.documentElement.classList.remove('dark')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('embeds purchase pages without forwarding tokens or user identifiers', async () => {
    const wrapper = mount(PurchaseSubscriptionView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: { template: '<span />' },
        },
      },
    })

    await flushPromises()

    const iframe = wrapper.get('iframe')
    const src = iframe.attributes('src')
    expect(src).toContain('https://pay.example.com/checkout?plan=pro')
    expect(src).toContain('theme=light')
    expect(src).toContain('lang=zh-CN')
    expect(src).toContain('ui_mode=embedded')
    expect(src).not.toContain('token=')
    expect(src).not.toContain('user_id=')
    expect(src).not.toContain('src_host=')
    expect(src).not.toContain('src_url=')
  })

  it('uses built-in payment mode without forwarding tokens or user identifiers', async () => {
    testState.appStoreState.cachedPublicSettings.payment_provider_airwallex_enabled = true
    createPaymentOrderMock.mockResolvedValue({
      order: {
        order_no: 'pay_test',
        product_type: 'balance_topup',
        status: 'pending',
        provider: 'airwallex',
        provider_env: 'demo',
        amount_minor: 1000,
        amount: 10,
        currency: 'USD',
      },
      client_secret: 'secret_test',
      intent_id: 'int_test',
      resume_token: 'resume_test',
      provider_env: 'demo',
    })

    const wrapper = mount(PurchaseSubscriptionView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: { template: '<span />' },
        },
      },
    })

    await flushPromises()
    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()

    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(createPaymentOrderMock).toHaveBeenCalledWith(
      expect.not.objectContaining({ token: expect.anything(), user_id: expect.anything() }),
      expect.any(String)
    )
    expect(wrapper.html()).not.toContain('token=')
    expect(wrapper.html()).not.toContain('user_id=')
  })
})
