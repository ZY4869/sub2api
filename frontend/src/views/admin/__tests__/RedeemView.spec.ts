import { describe, expect, it, beforeEach, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import RedeemView from '../RedeemView.vue'

const {
  list,
  generate,
  getAllGroups,
  showError,
  showSuccess,
  showInfo,
} = vi.hoisted(() => ({
  list: vi.fn(),
  generate: vi.fn(),
  getAllGroups: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showInfo: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    redeem: {
      list,
      generate,
    },
    groups: {
      getAll: getAllGroups,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning: vi.fn(),
    showInfo,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn().mockResolvedValue(true),
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  const messages: Record<string, string> = {
    'admin.redeem.generateCodes': 'Generate Codes',
    'admin.redeem.generateCodesTitle': 'Generate Redeem Codes',
    'admin.redeem.generate': 'Generate',
    'admin.redeem.codeType': 'Code Type',
    'admin.redeem.amount': 'Amount ($)',
    'admin.redeem.columns.value': 'Value',
    'admin.redeem.count': 'Count',
    'admin.redeem.adjustmentHint': 'Positive adds, negative subtracts, 0 is invalid',
    'admin.redeem.nonZeroValueRequired': 'Please enter a non-zero value',
    'admin.redeem.groupRequired': 'Please select a subscription group',
    'common.cancel': 'Cancel',
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const AppLayoutStub = defineComponent({
  name: 'AppLayout',
  template: '<div><slot /></div>',
})

const TablePageLayoutStub = defineComponent({
  name: 'TablePageLayout',
  template: `
    <div>
      <slot name="filters" />
      <slot name="table" />
      <slot name="pagination" />
    </div>
  `,
})

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: {
    modelValue: {
      type: [String, Number, Boolean],
      default: null,
    },
  },
  emits: ['update:modelValue', 'change'],
  template: '<div class="select-stub" />',
})

describe('RedeemView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    list.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0,
    })
    generate.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
  })

  function mountView() {
    return mount(RedeemView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          TablePageLayout: TablePageLayoutStub,
          DataTable: true,
          Pagination: true,
          ConfirmDialog: true,
          Select: SelectStub,
          GroupBadge: true,
          GroupOptionItem: true,
          Icon: true,
          Teleport: true,
        },
      },
    })
  }

  async function openGenerateDialog(wrapper: ReturnType<typeof mount>) {
    const button = wrapper
      .findAll('button')
      .find((candidate) => candidate.text() === 'Generate Codes')
    expect(button).toBeTruthy()
    await button!.trigger('click')
    await flushPromises()
  }

  it('blocks zero adjustments before submitting', async () => {
    const wrapper = mountView()
    await flushPromises()
    await openGenerateDialog(wrapper)

    const amountInput = wrapper.find('form input[type="number"]')
    await amountInput.setValue('0')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(generate).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('Please enter a non-zero value')
  })

  it('submits negative balance adjustments', async () => {
    const wrapper = mountView()
    await flushPromises()
    await openGenerateDialog(wrapper)

    const amountInput = wrapper.find('form input[type="number"]')
    await amountInput.setValue('-5')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(generate).toHaveBeenCalledWith(1, 'balance', -5, undefined, undefined)
  })
})
