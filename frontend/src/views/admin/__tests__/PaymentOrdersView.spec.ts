import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PaymentOrdersView from '../PaymentOrdersView.vue'
import type { VueWrapper } from '@vue/test-utils'

const listOrdersMock = vi.hoisted(() => vi.fn())
const refundOrderMock = vi.hoisted(() => vi.fn())
const appStoreMock = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    payment: {
      listOrders: listOrdersMock,
      refundOrder: refundOrderMock
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMock
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'en' },
      t: (key: string, params?: Record<string, unknown>) =>
        params?.amount ? `${key} ${params.amount}` : key
    })
  }
})

vi.mock('@/components/common/DataTable.vue', () => ({
  default: {
    props: ['columns', 'data', 'loading'],
    template: `
      <div>
        <div v-for="row in data" :key="row.order_no" data-test="order-row">
          <slot name="cell-order" :row="row" />
          <slot name="cell-status" :row="row" />
          <slot name="cell-actions" :row="row" />
        </div>
      </div>
    `
  }
}))

vi.mock('@/components/common/BaseDialog.vue', () => ({
  default: {
    props: ['show', 'title'],
    template: `
      <section v-if="show" data-test="refund-dialog">
        <slot />
        <slot name="footer" />
      </section>
    `
  }
}))

function makeOrder() {
  return {
    order_no: 'pay_1',
    user_id: 7,
    product_type: 'balance_topup',
    status: 'paid',
    provider: 'airwallex',
    provider_env: 'demo',
    provider_intent_id: 'int_123',
    amount_minor: 1000,
    amount: 10,
    refunded_amount_minor: 0,
    refundable_amount_minor: 1000,
    currency: 'USD',
    created_at: '2026-05-22T00:00:00Z'
  }
}

function mountView() {
  return mount(PaymentOrdersView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
        Pagination: { template: '<nav />' },
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue', 'change'],
          template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value); $emit(\'change\')"></select>'
        },
        Icon: { template: '<span />' },
        PaymentStatusPanel: { props: ['order'], template: '<div data-test="status-panel">{{ order && order.order_no }}</div>' }
      }
    }
  })
}

describe('PaymentOrdersView', () => {
  beforeEach(() => {
    listOrdersMock.mockReset()
    refundOrderMock.mockReset()
    appStoreMock.showError.mockReset()
    appStoreMock.showSuccess.mockReset()
  })

  it('loads orders and submits a bounded refund', async () => {
    listOrdersMock.mockResolvedValue({ items: [makeOrder()], total: 1, page: 1, page_size: 20, pages: 1 })
    refundOrderMock.mockResolvedValue({ refund_no: 'rf_1' })

    const wrapper = mountView()
    await flushPromises()

    expect(listOrdersMock).toHaveBeenCalledWith(1, 20, {
      status: undefined,
      product_type: undefined,
      user_id: undefined
    })
    await wrapper.find('button.btn-sm').trigger('click')
    await refundAmountInput(wrapper).setValue('500')
    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(refundOrderMock).toHaveBeenCalledWith(
      'pay_1',
      { amount_minor: 500, reason: undefined },
      expect.any(String)
    )
    expect(appStoreMock.showSuccess).toHaveBeenCalledWith('admin.payment.refund.success')
  })

  it('rejects over-refund before calling the API', async () => {
    listOrdersMock.mockResolvedValue({ items: [makeOrder()], total: 1, page: 1, page_size: 20, pages: 1 })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('button.btn-sm').trigger('click')
    await refundAmountInput(wrapper).setValue('1001')
    await wrapper.find('form').trigger('submit.prevent')

    expect(refundOrderMock).not.toHaveBeenCalled()
    expect(appStoreMock.showError).toHaveBeenCalledWith('admin.payment.refund.invalidAmount')
  })
})

function refundAmountInput(wrapper: VueWrapper) {
  return wrapper.find('#payment-refund-form input[type="number"]')
}
