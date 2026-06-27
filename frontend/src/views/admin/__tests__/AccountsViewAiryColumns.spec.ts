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
          ignore_free_accounts: false,
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

vi.mock('@/composables/useAccountDisplayPreferences', () => ({
  useAccountDisplayPreferences: () => ({
    accountTodayStatsWindows: ref(['today', 'weekly', 'total']),
    accountGroupDisplayMode: ref('full'),
    updatingAccountDisplayPreferences: ref(false),
    setAccountDisplayPreferences: vi.fn(),
  }),
}))

const TablePageLayoutStub = defineComponent({
  template: `
    <div>
      <slot name="filters" />
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
  template: '<div class="table-columns">{{ columns.map((column) => `${column.key}:${column.class || ""}`).join("|") }}</div>',
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
    expect(wrapper.get('.table-columns').text()).toContain('name:w-[clamp(184px,18vw,220px)]')
    expect(wrapper.get('.table-columns').text()).toContain(
      'platform_type:w-[clamp(144px,13vw,168px)] min-w-[140px] max-w-[172px]',
    )
    expect(wrapper.get('.table-columns').text()).toContain(
      'capacity:w-[clamp(164px,14vw,184px)] min-w-[156px] max-w-[188px]',
    )
    expect(wrapper.get('.table-columns').text()).toContain(
      'status:w-[clamp(176px,14vw,192px)] min-w-[168px] max-w-[196px]',
    )
    expect(wrapper.get('.table-columns').text()).toContain(
      'last_used_at:w-[112px] min-w-[104px] max-w-[120px] whitespace-nowrap',
    )
    expect(wrapper.get('.table-columns').text()).toContain(
      'created_at:w-[clamp(136px,12vw,152px)] min-w-[132px] max-w-[156px] whitespace-nowrap',
    )
    expect(wrapper.get('.table-columns').text()).toContain(
      'expires_at:w-[clamp(164px,13vw,184px)] min-w-[156px] max-w-[188px] whitespace-nowrap',
    )
    expect(wrapper.get('.table-columns').text()).toContain('usage_reset_dates:w-[clamp(216px,18vw,248px)]')
    expect(wrapper.get('.toolbar-columns').text()).toContain('usage_reset_dates')
  })
})
