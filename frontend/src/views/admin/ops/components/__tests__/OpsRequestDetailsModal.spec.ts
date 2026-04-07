import { describe, expect, it, beforeEach, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import OpsRequestDetailsModal from '../OpsRequestDetailsModal.vue'

const mockListRequestDetails = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listRequestDetails: (...args: any[]) => mockListRequestDetails(...args),
  },
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn().mockResolvedValue(true),
  }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showWarning: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn(),
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, any>) => {
        if (key === 'admin.ops.requestDetails.rangeHours') {
          return `${params?.n} hours`
        }
        if (key === 'admin.ops.requestDetails.rangeMinutes') {
          return `${params?.n} minutes`
        }
        return key
      },
    }),
  }
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: { type: Boolean, default: false },
    title: { type: String, default: '' },
  },
  emits: ['close'],
  template: '<div v-if="show"><slot /></div>',
})

describe('OpsRequestDetailsModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockListRequestDetails.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 10,
      pages: 0,
    })
  })

  it('passes channel_id when loading request details', async () => {
    const wrapper = mount(OpsRequestDetailsModal, {
      props: {
        modelValue: false,
        timeRange: '1h',
        preset: {
          title: 'Request Details',
          kind: 'all',
          sort: 'created_at_desc',
        },
        platform: 'openai',
        groupId: 7,
        channelId: 9,
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Pagination: true,
        },
      },
    })

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(mockListRequestDetails).toHaveBeenCalledWith(
      expect.objectContaining({
        platform: 'openai',
        group_id: 7,
        channel_id: 9,
        kind: 'all',
        sort: 'created_at_desc',
      })
    )
  })
})
