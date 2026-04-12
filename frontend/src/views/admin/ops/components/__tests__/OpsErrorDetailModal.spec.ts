import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { OpsErrorDetail } from '@/api/admin/ops'
import OpsErrorDetailModal from '../OpsErrorDetailModal.vue'

const mockGetRequestErrorDetail = vi.fn()
const mockGetUpstreamErrorDetail = vi.fn()
const mockListRequestErrorUpstreamErrors = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getRequestErrorDetail: (...args: any[]) => mockGetRequestErrorDetail(...args),
    getUpstreamErrorDetail: (...args: any[]) => mockGetUpstreamErrorDetail(...args),
    listRequestErrorUpstreamErrors: (...args: any[]) => mockListRequestErrorUpstreamErrors(...args)
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn()
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, any>) => {
        if (key === 'admin.ops.errorDetail.titleWithId') return `detail-${params?.id}`
        return key
      }
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

function createErrorDetail(overrides: Partial<OpsErrorDetail> = {}): OpsErrorDetail {
  return {
    id: 1,
    created_at: '2026-04-12T10:00:00Z',
    phase: 'request',
    type: 'request_error',
    error_owner: 'client',
    error_source: 'client_request',
    severity: 'warn',
    status_code: 400,
    platform: 'gemini',
    model: 'gemini-2.5-flash',
    is_retryable: false,
    retry_count: 0,
    resolved: false,
    client_request_id: 'client-1',
    request_id: 'req-1',
    message: 'request failed',
    user_email: 'user@example.com',
    account_name: '',
    group_name: 'Gemini Group',
    request_path: '/v1beta/models/gemini-2.5-flash:generateContent',
    inbound_endpoint: '/v1beta/models',
    upstream_endpoint: '/v1beta/models',
    requested_model: 'gemini-2.5-flash',
    upstream_model: 'gemini-2.5-flash',
    request_type: 1,
    gemini_surface: 'interactions',
    billing_rule_id: 'rule-77',
    probe_action: 'recovery_probe',
    error_body: '{"error":"bad request"}',
    user_agent: 'vitest',
    request_body: '{"contents":[]}',
    request_body_truncated: false,
    is_business_limited: false,
    ...overrides
  }
}

function findSummaryValue(wrapper: VueWrapper<any>, label: string): string {
  const cards = wrapper.findAll('div.rounded-xl.bg-gray-50.p-4')
  const card = cards.find((item) => item.find('div.text-xs').exists() && item.find('div.text-xs').text() === label)
  if (!card) {
    throw new Error(`summary card not found for ${label}`)
  }
  return card.find('.mt-1').text().replace(/\s+/g, ' ').trim()
}

describe('OpsErrorDetailModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetUpstreamErrorDetail.mockResolvedValue(createErrorDetail())
    mockListRequestErrorUpstreamErrors.mockResolvedValue({ items: [] })
  })

  it('shows gemini metadata fields from request error detail', async () => {
    mockGetRequestErrorDetail.mockResolvedValue(createErrorDetail())

    const wrapper = mount(OpsErrorDetailModal, {
      props: {
        show: true,
        errorId: 123,
        errorType: 'request'
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(mockGetRequestErrorDetail).toHaveBeenCalledWith(123)
    expect(findSummaryValue(wrapper, 'admin.ops.errorDetail.geminiSurface')).toBe('interactions')
    expect(findSummaryValue(wrapper, 'admin.ops.errorDetail.billingRuleId')).toBe('rule-77')
    expect(findSummaryValue(wrapper, 'admin.ops.errorDetail.probeAction')).toBe('recovery_probe')
  })

  it('shows hyphen placeholders when gemini metadata is empty', async () => {
    mockGetRequestErrorDetail.mockResolvedValue(
      createErrorDetail({
        gemini_surface: '',
        billing_rule_id: '',
        probe_action: ''
      })
    )

    const wrapper = mount(OpsErrorDetailModal, {
      props: {
        show: true,
        errorId: 456,
        errorType: 'request'
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(findSummaryValue(wrapper, 'admin.ops.errorDetail.geminiSurface')).toBe('-')
    expect(findSummaryValue(wrapper, 'admin.ops.errorDetail.billingRuleId')).toBe('-')
    expect(findSummaryValue(wrapper, 'admin.ops.errorDetail.probeAction')).toBe('-')
  })
})
