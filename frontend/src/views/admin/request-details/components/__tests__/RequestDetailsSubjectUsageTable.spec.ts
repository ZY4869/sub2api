import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import RequestDetailsSubjectUsageTable from '../RequestDetailsSubjectUsageTable.vue'

const setUsageModelDisplayMode = vi.fn()
const setUsageContextBadgeDisplayMode = vi.fn()

const messages: Record<string, string> = {
  'admin.requestDetails.subject.ledger.title': 'Subject Usage',
  'admin.requestDetails.subject.ledger.description': 'Usage rows for the selected subject',
  'admin.requestDetails.subject.ledger.columns.createdAt': 'Created At',
  'admin.requestDetails.subject.ledger.columns.requestId': 'Request ID',
  'admin.requestDetails.subject.ledger.columns.apiKeyId': 'API Key ID',
  'admin.requestDetails.subject.ledger.columns.accountId': 'Account ID',
  'admin.requestDetails.subject.ledger.columns.groupId': 'Group ID',
  'admin.requestDetails.subject.ledger.columns.models': 'Models',
  'admin.requestDetails.subject.ledger.columns.nativeContext': 'Native Context',
  'admin.requestDetails.subject.ledger.columns.status': 'Status',
  'admin.requestDetails.subject.ledger.columns.totalTokens': 'Total Tokens',
  'admin.requestDetails.subject.ledger.columns.totalStandardCost': 'Standard Cost',
  'admin.requestDetails.subject.ledger.columns.totalUserCost': 'User Cost',
  'admin.requestDetails.subject.ledger.columns.durationMs': 'Duration',
  'admin.requestDetails.subject.ledger.columns.previewAvailable': 'Preview',
  'admin.requestDetails.subject.ledger.columns.actions': 'Actions',
  'admin.requestDetails.subject.ledger.empty': 'No subject usage',
  'usage.requestPreview.action': 'Request Details',
  'common.yes': 'Yes',
  'common.no': 'No',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

vi.mock('@/composables/useTokenDisplayMode', () => ({
  useTokenDisplayMode: () => ({
    formatTokenDisplay: (value: number) => `${value}`,
  }),
}))

vi.mock('@/composables/useUsageModelDisplayModePreference', () => ({
  useUsageModelDisplayModePreference: () => ({
    usageModelDisplayMode: 'display_and_model',
    updatingUsageModelDisplayMode: false,
    setUsageModelDisplayMode,
  }),
}))

vi.mock('@/composables/useUsageContextBadgeDisplayModePreference', () => ({
  useUsageContextBadgeDisplayModePreference: () => ({
    usageContextBadgeDisplayMode: 'request_only',
    updatingUsageContextBadgeDisplayMode: false,
    setUsageContextBadgeDisplayMode,
  }),
}))

vi.mock('@/api/admin/usage', () => ({
  default: {
    getRequestPreview: vi.fn(),
  },
  adminUsageAPI: {
    getRequestPreview: vi.fn(),
  },
}))

const DataTableStub = {
  props: ['columns', 'data'],
  template: `
    <div>
      <slot name="header-models" :column="{ key: 'models', label: 'Models' }" />
      <div v-for="(row, index) in data" :key="row.id ?? index">
        <slot name="cell-models" :row="row" />
        <slot name="cell-native_context" :row="row" />
        <slot name="cell-preview_available" :value="row.preview_available" />
        <slot name="cell-actions" :row="row" />
      </div>
      <slot name="empty" />
    </div>
  `,
}

describe('RequestDetailsSubjectUsageTable', () => {
  it('renders toggle and shared model cell with the current display mode', async () => {
    const wrapper = mount(RequestDetailsSubjectUsageTable, {
      props: {
        items: [
          {
            id: 1,
            request_id: 'req-subject-1',
            model: 'deepseek-v4-pro',
            upstream_model: 'deepseek-v4-pro',
            status: 'success',
            total_tokens: 123,
            total_cost: 0.1,
            actual_cost: 0.1,
            duration_ms: 220,
            preview_available: true,
            created_at: '2026-05-06T12:00:00Z',
          },
        ],
        total: 1,
        page: 1,
        pageSize: 20,
        loading: false,
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Pagination: true,
          UsageRequestPreviewModal: true,
          UsageModelCell: {
            props: ['row', 'mode'],
            template: '<div data-testid="usage-model-cell">{{ row.model }}|{{ mode }}</div>',
          },
          UsageModelDisplayModeToggle: {
            props: ['modelValue', 'disabled', 'showLabel', 'compact'],
            template: `
              <button
                data-testid="usage-mode-toggle"
                @click="$emit('update:modelValue', 'display_only')"
              >
                {{ modelValue }}
              </button>
            `,
          },
          UsageContextBadgeDisplayModeToggle: {
            props: ['modelValue', 'disabled', 'showLabel', 'compact'],
            template: `
              <button
                data-testid="context-mode-toggle"
                @click="$emit('update:modelValue', 'native_only')"
              >
                {{ modelValue }}
              </button>
            `,
          },
        },
      },
    })

    expect(wrapper.get('[data-testid="usage-model-cell"]').text()).toContain('deepseek-v4-pro|display_and_model')
    expect(wrapper.findAll('[data-testid="usage-mode-toggle"]').length).toBeGreaterThan(0)
    expect(wrapper.text()).toContain('1M')
    expect(wrapper.findAll('[data-testid="context-mode-toggle"]').length).toBeGreaterThan(0)

    await wrapper.get('[data-testid="usage-mode-toggle"]').trigger('click')
    expect(setUsageModelDisplayMode).toHaveBeenCalledWith('display_only')

    await wrapper.get('[data-testid="context-mode-toggle"]').trigger('click')
    expect(setUsageContextBadgeDisplayMode).toHaveBeenCalledWith('native_only')
  })
})
