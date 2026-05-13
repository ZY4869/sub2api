import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import PurchaseSubscriptionView from '../PurchaseSubscriptionView.vue'

const testState = vi.hoisted(() => ({
  appStoreState: {
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
    cachedPublicSettings: {
      purchase_subscription_enabled: true,
      purchase_subscription_url: 'https://pay.example.com/checkout?plan=pro',
      purchase_subscription_provider: 'airwallex',
      purchase_subscription_default_currency: 'USD',
      purchase_subscription_default_country_code: 'US',
      purchase_subscription_payment_env: 'sandbox',
      purchase_subscription_extra_params: {
        merchant_region: 'global',
      },
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

describe('PurchaseSubscriptionView', () => {
  beforeEach(() => {
    testState.appStoreState.fetchPublicSettings.mockReset()
    testState.appStoreState.publicSettingsLoaded = true
    testState.appStoreState.cachedPublicSettings.purchase_subscription_enabled = true
    testState.appStoreState.cachedPublicSettings.purchase_subscription_url = 'https://pay.example.com/checkout?plan=pro'
    testState.appStoreState.cachedPublicSettings.purchase_subscription_provider = 'airwallex'
    testState.appStoreState.cachedPublicSettings.purchase_subscription_default_currency = 'USD'
    testState.appStoreState.cachedPublicSettings.purchase_subscription_default_country_code = 'US'
    testState.appStoreState.cachedPublicSettings.purchase_subscription_payment_env = 'sandbox'
    testState.appStoreState.cachedPublicSettings.purchase_subscription_extra_params = {
      merchant_region: 'global',
    }
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
    expect(src).toContain('currency=USD')
    expect(src).toContain('country_code=US')
    expect(src).toContain('payment_env=sandbox')
    expect(src).toContain('merchant_region=global')
    expect(src).not.toContain('token=')
    expect(src).not.toContain('user_id=')
    expect(src).not.toContain('src_host=')
    expect(src).not.toContain('src_url=')
  })
})
