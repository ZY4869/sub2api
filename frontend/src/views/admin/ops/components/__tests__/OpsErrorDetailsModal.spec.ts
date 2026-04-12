import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import OpsErrorDetailsModal from '../OpsErrorDetailsModal.vue'

const mockListRequestErrors = vi.fn()
const mockListUpstreamErrors = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listRequestErrors: (...args: any[]) => mockListRequestErrors(...args),
    listUpstreamErrors: (...args: any[]) => mockListUpstreamErrors(...args)
  }
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialogStub',
  props: {
    show: { type: Boolean, default: false }
  },
  emits: ['close'],
  template: '<div v-if="show"><slot /></div>'
})

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: ['modelValue', 'options'],
  template: '<div class="select-stub" />'
})

const OpsErrorLogTableStub = defineComponent({
  name: 'OpsErrorLogTableStub',
  template: '<div class="ops-error-log-table-stub" />'
})

describe('OpsErrorDetailsModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockListRequestErrors.mockResolvedValue({ items: [], total: 0 })
    mockListUpstreamErrors.mockResolvedValue({ items: [], total: 0 })
  })

  it('passes trimmed gemini filters to listRequestErrors', async () => {
    const wrapper = mount(OpsErrorDetailsModal, {
      props: {
        show: true,
        timeRange: '1h',
        errorType: 'request'
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Select: SelectStub,
          OpsErrorLogTable: OpsErrorLogTableStub
        }
      }
    })

    await flushPromises()
    mockListRequestErrors.mockClear()

    await wrapper.get('input[placeholder="admin.ops.errorDetails.filters.geminiSurface"]').setValue(' live ')
    await wrapper.get('input[placeholder="admin.ops.errorDetails.filters.billingRuleId"]').setValue(' rule-88 ')
    await wrapper.get('input[placeholder="admin.ops.errorDetails.filters.probeAction"]').setValue(' recovery_probe ')
    await flushPromises()

    expect(mockListRequestErrors).toHaveBeenLastCalledWith(
      expect.objectContaining({
        gemini_surface: 'live',
        billing_rule_id: 'rule-88',
        probe_action: 'recovery_probe'
      })
    )
  })
})
