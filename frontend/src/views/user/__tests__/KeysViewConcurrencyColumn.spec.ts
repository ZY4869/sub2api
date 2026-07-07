import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import KeysView from '../KeysView.vue'

const mocks = vi.hoisted(() => ({
  listKeys: vi.fn(),
  getAvailableGroups: vi.fn(),
  getModelOptions: vi.fn(),
  getModelCatalog: vi.fn(),
  getUserGroupRates: vi.fn(),
  getPublicSettings: vi.fn(),
  getUsage: vi.fn(),
  showError: vi.fn(),
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

vi.mock('@/api', () => ({
  keysAPI: {
    list: mocks.listKeys,
    getModelCatalog: vi.fn(),
    createWithPayload: vi.fn(),
    update: vi.fn(),
    toggleStatus: vi.fn(),
    delete: vi.fn(),
  },
  authAPI: {
    getPublicSettings: mocks.getPublicSettings,
  },
  usageAPI: {
    getDashboardApiKeysUsage: mocks.getUsage,
  },
  userGroupsAPI: {
    getAvailable: mocks.getAvailableGroups,
    getModelOptions: mocks.getModelOptions,
    getModelCatalog: mocks.getModelCatalog,
    getUserGroupRates: mocks.getUserGroupRates,
  },
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      getAll: vi.fn(),
    },
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isAdmin: false,
    user: {
      id: 1,
      role: 'user',
    },
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: mocks.showError,
    showSuccess: vi.fn(),
  }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep: vi.fn(() => false),
    nextStep: vi.fn(),
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(),
  }),
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20,
}))

const KeysTableStub = defineComponent({
  name: 'KeysTable',
  props: ['columns', 'apiKeys'],
  setup(props) {
    return () =>
      h('div', { 'data-testid': 'keys-table' }, [
        h(
          'div',
          { 'data-testid': 'keys-table-columns' },
          props.columns.map((column: { key: string; label: string }) => `${column.key}:${column.label}`).join('|'),
        ),
        h(
          'div',
          { 'data-testid': 'keys-table-concurrency' },
          props.apiKeys.map((key: { current_concurrency?: number }) => String(key.current_concurrency ?? 0)).join(','),
        ),
      ])
  },
})

function setupMocks() {
  localStorage.clear()
  mocks.listKeys.mockResolvedValue({
    items: [
      {
        id: 7,
        user_id: 1,
        key: 'sk-test',
        name: 'realtime-key',
        group_id: null,
        status: 'active',
        ip_whitelist: [],
        ip_blacklist: [],
        last_used_at: null,
        current_concurrency: 3,
        quota: 0,
        quota_used: 0,
        image_only_enabled: false,
        image_count_billing_enabled: false,
        image_max_count: 0,
        image_count_used: 0,
        image_count_weights: { '1K': 1, '2K': 1, '4K': 1 },
        expires_at: null,
        created_at: '2026-07-07T00:00:00Z',
        updated_at: '2026-07-07T00:00:00Z',
        rate_limit_5h: 0,
        rate_limit_1d: 0,
        rate_limit_7d: 0,
        usage_5h: 0,
        usage_1d: 0,
        usage_7d: 0,
        window_5h_start: null,
        window_1d_start: null,
        window_7d_start: null,
        reset_5h_at: null,
        reset_1d_at: null,
        reset_7d_at: null,
      },
    ],
    total: 1,
    pages: 1,
  })
  mocks.getUsage.mockResolvedValue({ stats: {} })
  mocks.getAvailableGroups.mockResolvedValue([])
  mocks.getModelOptions.mockResolvedValue([])
  mocks.getModelCatalog.mockResolvedValue({ items: [] })
  mocks.getUserGroupRates.mockResolvedValue({})
  mocks.getPublicSettings.mockResolvedValue({})
  mocks.showError.mockReset()
}

describe('KeysView current concurrency column', () => {
  beforeEach(() => {
    setupMocks()
  })

  it('includes the current_concurrency column and passes API key concurrency snapshots to the table', async () => {
    const wrapper = mount(KeysView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="actions" /><slot name="table" /><slot name="pagination" /></div>',
          },
          KeysTable: KeysTableStub,
          KeysColumnSettingsMenu: { template: '<div />' },
          Pagination: { template: '<div />' },
          KeyFormDialog: { template: '<div />' },
          ConfirmDialog: { template: '<div />' },
          Select: { template: '<div />' },
          SearchInput: { template: '<input />' },
          Icon: { template: '<span />' },
          UseKeyModal: { template: '<div />' },
          CcsClientSelectDialog: { template: '<div />' },
        },
      },
    })

    await flushPromises()

    expect(wrapper.get('[data-testid="keys-table-columns"]').text()).toContain(
      'current_concurrency:keys.currentConcurrency',
    )
    expect(wrapper.get('[data-testid="keys-table-concurrency"]').text()).toBe('3')
  })
})
