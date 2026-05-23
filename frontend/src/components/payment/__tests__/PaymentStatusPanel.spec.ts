import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import PaymentStatusPanel from '../PaymentStatusPanel.vue'
import type { PaymentOrder } from '@/types'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'en' },
    t: (key: string, params?: Record<string, unknown>) =>
      params?.currency ? `${key}:${params.currency}` : key
  })
}))

function makeOrder(amount: number): PaymentOrder {
  return {
    order_no: 'pay_test',
    product_type: 'balance_topup',
    status: 'pending',
    provider: 'airwallex',
    provider_env: 'demo',
    amount_minor: 0,
    amount,
    currency: 'USD'
  }
}

describe('PaymentStatusPanel', () => {
  it('does not render NaN when the order amount is invalid', () => {
    const wrapper = mount(PaymentStatusPanel, {
      props: {
        order: makeOrder(Number.NaN)
      }
    })

    expect(wrapper.text()).toContain('purchase.amountUnavailableWithCurrency:USD')
    expect(wrapper.text()).not.toContain('NaN')
  })
})
