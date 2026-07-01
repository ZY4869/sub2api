import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ModerationAuditsView from '../ModerationAuditsView.vue'

const testState = vi.hoisted(() => ({
  listAuditsMock: vi.fn(),
  getAuditDetailMock: vi.fn(),
  showErrorMock: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/api/admin', () => ({
  adminAPI: {
    moderation: {
      listAudits: testState.listAuditsMock,
      getAuditDetail: testState.getAuditDetailMock,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: testState.showErrorMock,
  }),
}))

describe('ModerationAuditsView', () => {
  beforeEach(() => {
    testState.listAuditsMock.mockReset()
    testState.getAuditDetailMock.mockReset()
    testState.showErrorMock.mockReset()
  })

  it('loads audits on mount and shows detail when a row is clicked', async () => {
    testState.listAuditsMock.mockResolvedValue({
      items: [
        {
          id: 9,
          request_id: 'req-1',
          client_request_id: 'client-1',
          user_id: 5,
          api_key_id: 8,
          provider: 'openai',
          model: 'omni-moderation-latest',
          source_endpoint: 'openai_messages',
          content_hash: 'hash-1',
          content_summary: 'masked summary',
          matched_keyword: 'blocked phrase',
          hit: false,
          dedupe_hit: false,
          error_reason: '',
          latency_ms: 12,
          created_at: '2025-01-02T03:04:05Z',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    testState.getAuditDetailMock.mockResolvedValue({
      id: 9,
      request_id: 'req-1',
      client_request_id: 'client-1',
      user_id: 5,
      api_key_id: 8,
      provider: 'openai',
      model: 'omni-moderation-latest',
      source_endpoint: 'openai_messages',
      content_hash: 'hash-1',
      content_summary: 'masked summary detail',
      matched_keyword: 'blocked phrase',
      hit: false,
      dedupe_hit: false,
      error_reason: '',
      latency_ms: 12,
      created_at: '2025-01-02T03:04:05Z',
    })

    const wrapper = mount(ModerationAuditsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Pagination: true,
          Select: {
            props: ['modelValue', 'options'],
            emits: ['update:modelValue', 'change'],
            template: '<select />',
          },
        },
      },
    })

    await flushPromises()

    expect(testState.listAuditsMock).toHaveBeenCalledWith({
      page: 1,
      page_size: 20,
      request_id: undefined,
      client_request_id: undefined,
      provider: undefined,
      model: undefined,
      source_endpoint: undefined,
      content_hash: undefined,
      user_id: undefined,
      hit: undefined,
    })

    await wrapper.get('tbody tr').trigger('click')
    await flushPromises()

    expect(testState.getAuditDetailMock).toHaveBeenCalledWith(9)
    expect(wrapper.text()).toContain('masked summary detail')
    expect(wrapper.text()).toContain('blocked phrase')
  })

  it('shows an empty state when there are no audits', async () => {
    testState.listAuditsMock.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const wrapper = mount(ModerationAuditsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Pagination: true,
          Select: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('admin.moderation.empty')
  })
})
