import { describe, expect, it, vi, beforeEach } from 'vitest'
import { useAirwallexElements } from '@/composables/useAirwallexElements'
import type { PaymentCreateOrderResponse } from '@/types'

const sdkMock = vi.hoisted(() => ({
  init: vi.fn(),
  createElement: vi.fn()
}))

vi.mock('@airwallex/components-sdk', () => sdkMock)

function makeOrder(overrides: Partial<PaymentCreateOrderResponse> = {}): PaymentCreateOrderResponse {
  return {
    order: {
      order_no: 'pay_test',
      product_type: 'balance_topup',
      status: 'pending',
      provider: 'airwallex',
      provider_env: 'demo',
      amount_minor: 1000,
      amount: 10,
      currency: 'USD'
    },
    client_secret: 'secret_test',
    client_id: 'client_test',
    intent_id: 'int_test',
    resume_token: 'resume_test',
    provider_env: 'demo',
    ...overrides
  }
}

describe('useAirwallexElements', () => {
  beforeEach(() => {
    sdkMock.init.mockReset()
    sdkMock.createElement.mockReset()
  })

  it('initializes Airwallex payments with client id and normalized locale', async () => {
    const element = {
      mount: vi.fn(),
      confirm: vi.fn().mockResolvedValue({ id: 'int_test' }),
      unmount: vi.fn(),
      destroy: vi.fn()
    }
    sdkMock.createElement.mockResolvedValue(element)
    const target = document.createElement('div')
    const airwallex = useAirwallexElements()

    await airwallex.mount(target, makeOrder(), 'zh-CN')
    const result = await airwallex.confirm(makeOrder())

    expect(sdkMock.init).toHaveBeenCalledWith({
      env: 'demo',
      locale: 'zh',
      clientId: 'client_test',
      enabledElements: ['payments']
    })
    expect(sdkMock.createElement).toHaveBeenCalledWith('card')
    expect(element.mount).toHaveBeenCalledWith(target)
    expect(element.confirm).toHaveBeenCalledWith({
      intent_id: 'int_test',
      client_secret: 'secret_test'
    })
    expect(result).toEqual({ id: 'int_test' })
    expect(airwallex.mounted.value).toBe(true)
    expect(airwallex.error.value).toBe('')
  })

  it('maps unknown environments to demo and preserves production explicitly', async () => {
    sdkMock.createElement.mockResolvedValue({ mount: vi.fn() })
    const target = document.createElement('div')
    const airwallex = useAirwallexElements()

    await airwallex.mount(target, makeOrder({ provider_env: 'sandbox' }), 'en-US')
    await airwallex.mount(target, makeOrder({ provider_env: 'prod' }), 'zh-HK')

    expect(sdkMock.init).toHaveBeenNthCalledWith(1, expect.objectContaining({
      env: 'demo',
      locale: 'en'
    }))
    expect(sdkMock.init).toHaveBeenNthCalledWith(2, expect.objectContaining({
      env: 'prod',
      locale: 'zh-HK'
    }))
  })
})
