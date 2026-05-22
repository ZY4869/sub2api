import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import PaymentResultView from '../PaymentResultView.vue'
import type { PaymentOrder } from '@/types'

const getPaymentOrderMock = vi.hoisted(() => vi.fn())
const resumePaymentOrderByOrderNoMock = vi.hoisted(() => vi.fn())
const routeState = vi.hoisted(() => ({
  params: { orderNo: 'pay_test' },
  query: {} as Record<string, string>
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getPaymentOrder: getPaymentOrderMock,
    resumePaymentOrderByOrderNo: resumePaymentOrderByOrderNoMock
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
      locale: { value: 'en' },
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
    template: '<div data-test="airwallex-element">{{ order && order.client_secret }}</div>'
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
  return mount(PaymentResultView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        Icon: { template: '<span />' }
      }
    }
  })
}

describe('PaymentResultView', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    getPaymentOrderMock.mockReset()
    resumePaymentOrderByOrderNoMock.mockReset()
    routeState.params.orderNo = 'pay_test'
    routeState.query = {}
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows success state without resume action for paid orders', async () => {
    getPaymentOrderMock.mockResolvedValue(makeOrder('paid'))

    const wrapper = mountView()
    await flushPromises()

    expect(getPaymentOrderMock).toHaveBeenCalledWith('pay_test')
    expect(wrapper.text()).toContain('paid')
    expect(wrapper.text()).toContain('purchase.resultPaid')
    expect(wrapper.find('a[href^="/payment/resume/"]').exists()).toBe(false)
  })

  it('polls pending orders and exposes resume link when a resume token is present', async () => {
    routeState.query = { resume_token: 'resume_test' }
    getPaymentOrderMock.mockResolvedValue(makeOrder('pending'))

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('a[href="/payment/resume/resume_test"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('purchase.resultPending')

    await vi.advanceTimersByTimeAsync(5000)
    await flushPromises()

    expect(getPaymentOrderMock).toHaveBeenCalledTimes(2)
  })

  it('resumes a pending order by order number when no resume token is present', async () => {
    getPaymentOrderMock.mockResolvedValue(makeOrder('pending'))
    resumePaymentOrderByOrderNoMock.mockResolvedValue({
      order: makeOrder('pending'),
      client_secret: 'secret_from_order_resume',
      client_id: 'client',
      intent_id: 'int_123',
      provider_env: 'demo'
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(resumePaymentOrderByOrderNoMock).toHaveBeenCalledWith('pay_test')
    expect(wrapper.find('[data-test="airwallex-element"]').text()).toContain('secret_from_order_resume')
  })

  it('uses retry purchase label for failed orders', async () => {
    getPaymentOrderMock.mockResolvedValue(makeOrder('failed'))

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('purchase.resultFailedStatus')
    expect(wrapper.find('a[href="/purchase"]').text()).toContain('purchase.retryPurchase')
  })
})
