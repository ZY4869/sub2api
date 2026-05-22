import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { usePaymentWorkbench } from '../usePaymentWorkbench'
import type { PublicSettings } from '@/types'

const createPaymentOrderMock = vi.hoisted(() => vi.fn())
const getPaymentOrderMock = vi.hoisted(() => vi.fn())
const cancelPaymentOrderMock = vi.hoisted(() => vi.fn())

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    createPaymentOrder: createPaymentOrderMock,
    getPaymentOrder: getPaymentOrderMock,
    cancelPaymentOrder: cancelPaymentOrderMock
  }
}))

function makeSettings(): PublicSettings {
  return {
    registration_enabled: true,
    email_verify_enabled: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: true,
    password_reset_enabled: false,
    invitation_code_enabled: false,
    turnstile_enabled: false,
    turnstile_site_key: '',
    site_name: 'Sub2API',
    site_logo: '',
    site_subtitle: '',
    visual_preset_default: 'classic',
    api_base_url: '',
    contact_info: '',
    doc_url: '',
    home_content: '',
    hide_ccs_import_button: false,
    available_channels_enabled: false,
    channel_monitor_enabled: false,
    public_model_catalog_enabled: true,
    affiliate_enabled: false,
    purchase_subscription_enabled: true,
    purchase_subscription_url: '',
    payment_provider_airwallex_enabled: true,
    payment_allowed_currencies: ['USD', 'HKD'],
    payment_default_currency: 'HKD',
    payment_min_topup_amount: 1,
    payment_max_topup_amount: 5000,
    payment_subscription_plans: [
      {
        plan_id: 'pro',
        name: 'Pro',
        group_id: 1,
        validity_days: 30,
        prices_by_currency: { USD: 12, HKD: 88 },
        enabled: true
      }
    ],
    custom_menu_items: [],
    login_agreement_enabled: false,
    login_agreement_mode: 'checkbox',
    login_agreement_updated_at: '',
    login_agreement_documents: [],
    linuxdo_oauth_enabled: false,
    github_oauth_enabled: false,
    google_oauth_enabled: false,
    backend_mode_enabled: false,
    maintenance_mode_enabled: false,
    version: 'test'
  }
}

describe('usePaymentWorkbench', () => {
  beforeEach(() => {
    createPaymentOrderMock.mockReset()
    getPaymentOrderMock.mockReset()
    cancelPaymentOrderMock.mockReset()
  })

  it('selects configured default currency and creates a top-up order', async () => {
    createPaymentOrderMock.mockResolvedValue({
      order: {
        order_no: 'pay_1',
        product_type: 'balance_topup',
        status: 'pending',
        provider: 'airwallex',
        provider_env: 'demo',
        amount_minor: 1000,
        amount: 10,
        currency: 'HKD'
      },
      client_secret: 'secret',
      client_id: 'client',
      intent_id: 'int_1',
      resume_token: 'resume',
      provider_env: 'demo'
    })
    const subject = usePaymentWorkbench(() => makeSettings())
    await nextTick()

    expect(subject.selectedCurrency.value).toBe('HKD')
    subject.topupAmount.value = 25
    subject.countryCode.value = 'hk'
    await subject.createOrder()

    expect(createPaymentOrderMock).toHaveBeenCalledWith(
      expect.objectContaining({
        product_type: 'balance_topup',
        amount: 25,
        currency: 'HKD',
        country_code: 'hk'
      }),
      expect.any(String)
    )
    expect(subject.order.value?.order_no).toBe('pay_1')
  })

  it('switches to subscription plans and blocks unavailable currency prices', async () => {
    const subject = usePaymentWorkbench(() => makeSettings())
    await nextTick()

    subject.productType.value = 'subscription'
    await nextTick()
    expect(subject.selectedPlanId.value).toBe('pro')
    expect(subject.payableAmount.value).toBe(88)

    subject.selectedCurrency.value = 'EUR'
    await nextTick()
    expect(subject.canCreate.value).toBe(false)
  })

  it('refreshes and cancels the current order', async () => {
    const subject = usePaymentWorkbench(() => makeSettings())
    subject.order.value = {
      order_no: 'pay_1',
      product_type: 'balance_topup',
      status: 'pending',
      provider: 'airwallex',
      provider_env: 'demo',
      amount_minor: 1000,
      amount: 10,
      currency: 'USD'
    }
    getPaymentOrderMock.mockResolvedValue({ ...subject.order.value, status: 'cancelled' })

    await subject.cancelOrder()

    expect(cancelPaymentOrderMock).toHaveBeenCalledWith('pay_1')
    expect(getPaymentOrderMock).toHaveBeenCalledWith('pay_1')
    expect(subject.order.value?.status).toBe('cancelled')
  })
})
