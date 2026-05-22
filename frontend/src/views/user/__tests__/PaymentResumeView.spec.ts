import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PaymentResumeView from '../PaymentResumeView.vue'
import type { PaymentOrder } from '@/types'

const resumePaymentOrderMock = vi.hoisted(() => vi.fn())
const getPaymentOrderMock = vi.hoisted(() => vi.fn())
const routeState = vi.hoisted(() => ({
  params: { resumeToken: 'resume_test' }
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    resumePaymentOrder: resumePaymentOrderMock,
    getPaymentOrder: getPaymentOrderMock
  }
}))

vi.mock('vue-router', () => ({
  RouterLink: {
    props: ['to'],
    template: '<a :href="to"><slot /></a>'
  },
  useRoute: () => routeState
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/components/payment/PaymentStatusPanel.vue', () => ({
  default: {
    props: ['order'],
    template: '<div data-test="status-panel">{{ order && order.status }}</div>'
  }
}))

vi.mock('@/components/payment/AirwallexPaymentElement.vue', () => ({
  default: {
    props: ['order'],
    emits: ['confirmed'],
    template: '<button data-test="airwallex-element" @click="$emit(\'confirmed\')">{{ order && order.client_secret }}</button>'
  }
}))

function makeOrder(status: PaymentOrder['status']): PaymentOrder {
  return {
    order_no: 'pay_test',
    product_type: 'balance_topup',
    status,
    provider: 'airwallex',
    provider_env: 'demo',
    amount_minor: 1000,
    amount: 10,
    currency: 'USD'
  }
}

function mountView() {
  return mount(PaymentResumeView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        Icon: { template: '<span />' }
      }
    }
  })
}

describe('PaymentResumeView', () => {
  beforeEach(() => {
    resumePaymentOrderMock.mockReset()
    getPaymentOrderMock.mockReset()
    routeState.params.resumeToken = 'resume_test'
  })

  it('loads a resumable order and refreshes after payment confirmation', async () => {
    resumePaymentOrderMock.mockResolvedValue({
      order: makeOrder('pending'),
      client_secret: 'secret',
      client_id: 'client',
      intent_id: 'int_123',
      provider_env: 'demo'
    })
    getPaymentOrderMock.mockResolvedValue(makeOrder('paid'))

    const wrapper = mountView()
    await flushPromises()

    expect(resumePaymentOrderMock).toHaveBeenCalledWith('resume_test')
    expect(wrapper.find('[data-test="airwallex-element"]').text()).toContain('secret')

    await wrapper.find('[data-test="airwallex-element"]').trigger('click')
    await flushPromises()

    expect(getPaymentOrderMock).toHaveBeenCalledWith('pay_test')
    expect(wrapper.find('[data-test="status-panel"]').text()).toContain('paid')
  })

  it('shows terminal text for paid orders without mounting payment element', async () => {
    resumePaymentOrderMock.mockResolvedValue({
      order: makeOrder('paid'),
      client_secret: '',
      client_id: 'client',
      intent_id: 'int_123',
      provider_env: 'demo'
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-test="airwallex-element"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('purchase.resumePaid')
  })
})
