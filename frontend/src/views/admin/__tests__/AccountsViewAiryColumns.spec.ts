import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { createPinia } from 'pinia'
import AccountsView from '../AccountsView.vue'

const { listAccounts } = vi.hoisted(() => ({
  listAccounts: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listArchivedGroups: vi.fn().mockResolvedValue([]),
      getSummary: vi.fn().mockResolvedValue({ total: 0, by_status: {}, by_platform: {} }),
      getRuntimeSummary: vi.fn().mockResolvedValue({ in_use: 0 }),
      getDaily5HTriggerSettings: vi.fn().mockResolvedValue({
        settings: {
          enabled: false,
          selected_account_types: [],
          include_paused_accounts: false,
          openai_model_mode: { mode: 'auto' },
          anthropic_model_mode: { mode: 'auto' },
          gemini_model_mode: { mode: 'auto' },
        },
        candidates: [],
      }),
      getBatchTodayStats: vi.fn().mockResolvedValue({ stats: {} }),
    },
    groups: {
      getAll: vi.fn().mockResolvedValue([]),
    },
    proxies: {
      getAll: vi.fn().mockResolvedValue([]),
    },
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: { account_realtime_countdown_enabled: true },
    isSimpleMode: false,
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ replace: vi.fn(), push: vi.fn() }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/composables/useRealtimeCountdownNow', () => ({
  useRealtimeCountdownNow: () => ({
    enabled: ref(true),
    nowMs: ref(0),
    nowDate: ref(new Date(0)),
  }),
}))

vi.mock('@/composables/useAccountVisualStylePreference', () => ({
  useAccountVisualStylePreference: () => ({
    resolvedAccountVisualPreset: ref('airy'),
    accountVisualPresetOverride: ref(null),
    updatingAccountVisualStyle: ref(false),
    setAccountVisualPresetOverride: vi.fn(),
  }),
}))

const TablePageLayoutStub = defineComponent({
  template: `
    <div>
      <slot name="toolbar" />
      <slot name="table" />
    </div>
  `,
})

const ToolbarStub = defineComponent({
  props: ['toggleableColumns'],
  template: '<div class="toolbar-columns">{{ toggleableColumns.map((column) => column.key).join(",") }}</div>',
})

const AccountsViewTableStub = defineComponent({
  props: ['columns'],
  template: '<div class="table-columns">{{ columns.map((column) => column.key).join(",") }}</div>',
})

describe('AccountsView airy columns', () => {
  beforeEach(() => {
    localStorage.clear()
    listAccounts.mockResolvedValue({
      data: [],
      total: 0,
      page: 1,
      page_size: 20,
    })
  })

  it('hides the duplicate schedulable column from airy table columns', async () => {
    const wrapper = mount(AccountsView, {
      global: {
        plugins: [createPinia()],
        stubs: {
          TablePageLayout: TablePageLayoutStub,
          AccountsViewToolbar: ToolbarStub,
          AccountsViewTable: AccountsViewTableStub,
          AccountGroupedView: true,
          AccountCardGrid: true,
          AccountBulkActionsBar: true,
          Pagination: true,
          AccountsViewDialogsHost: true,
          AccountLimitedSummaryBar: true,
          AccountStatusSummaryBar: true,
          AccountPlatformTabs: true,
          AccountDaily5HTriggerSettingsDialog: true,
          ArchivedAccountGroupsPanel: true,
          ScheduledTestsPanel: true,
          AccountActionMenu: true,
        },
      },
    })

    await vi.dynamicImportSettled()

    expect(wrapper.get('.table-columns').text()).not.toContain('schedulable')
  })
})
