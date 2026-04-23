import { flushPromises, mount } from '@vue/test-utils'
import { computed, defineComponent, toValue } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockState = vi.hoisted(() => ({
  summaryParamsSource: null as any,
  runtimeParamsSource: null as any,
  tableParams: null as any,
  tableItems: [] as any[],
  routerPush: vi.fn(),
  debouncedReload: vi.fn(),
  refreshAccountSummary: vi.fn(),
  refreshRuntimeSummary: vi.fn(),
  load: vi.fn(() => Promise.resolve()),
  reload: vi.fn(() => Promise.resolve()),
  refreshAccountsIncrementally: vi.fn(() => Promise.resolve()),
  refreshTodayStats: vi.fn(() => Promise.resolve()),
  syncPendingListChanges: vi.fn(() => Promise.resolve()),
  setAutoRefreshEnabled: vi.fn(),
  handleAutoRefreshIntervalChange: vi.fn(),
  enterAutoRefreshSilentWindow: vi.fn(),
  pauseAutoRefresh: vi.fn(),
  resumeAutoRefresh: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
  showError: vi.fn(),
  showInfo: vi.fn(),
  getAllGroups: vi.fn(),
  getAllProxies: vi.fn(),
  getById: vi.fn()
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockState.routerPush
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: mockState.showSuccess,
    showWarning: mockState.showWarning,
    showError: mockState.showError,
    showInfo: mockState.showInfo
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isSimpleMode: false
  })
}))

vi.mock('@/stores', () => ({
  useModelInventoryStore: () => ({
    invalidate: vi.fn()
  })
}))

vi.mock('@/stores/modelRegistry', () => ({
  invalidateModelRegistry: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: vi.fn(),
      getById: mockState.getById
    },
    groups: {
      getAll: mockState.getAllGroups
    },
    proxies: {
      getAll: mockState.getAllProxies
    }
  }
}))

vi.mock('@/composables/useTableLoader', async () => {
  const { reactive, ref } = await import('vue')
  return {
    useTableLoader: vi.fn((options: { initialParams?: Record<string, unknown> }) => {
      const params = reactive({ ...(options.initialParams || {}) })
      mockState.tableParams = params
      return {
        items: ref(mockState.tableItems),
        loading: ref(false),
        params,
        pagination: reactive({
          page: 1,
          page_size: 20,
          total: 0,
          pages: 0
        }),
        load: vi.fn(() => Promise.resolve()),
        reload: vi.fn(() => Promise.resolve()),
        debouncedReload: vi.fn(),
        handlePageChange: vi.fn(),
        handlePageSizeChange: vi.fn()
      }
    })
  }
})

vi.mock('@/composables/useAccountStatusSummary', async () => {
  const { ref } = await import('vue')
  return {
    useAccountStatusSummary: vi.fn((paramsSource: unknown) => {
      mockState.summaryParamsSource = paramsSource
      return {
        summary: ref({
          total: 88,
          by_status: {
            active: 67,
            inactive: 10,
            error: 4
          },
          rate_limited: 5,
          temp_unschedulable: 1,
          overloaded: 1,
          paused: 2,
          in_use: 0,
          remaining_available: 11,
          by_platform: {
            openai: 1,
            protocol_gateway: 100,
            gemini: 99
          },
          limited_breakdown: {
            total: 5,
            rate_429: 2,
            usage_5h: 2,
            usage_7d: 1,
            usage_7d_all: 0
          }
        }),
        loading: ref(false),
        error: ref(null),
        refresh: mockState.refreshAccountSummary
      }
    })
  }
})

vi.mock('@/composables/useAccountsRuntimeSummary', async () => {
  const { ref } = await import('vue')
  return {
    useAccountsRuntimeSummary: vi.fn((paramsSource: unknown) => {
      mockState.runtimeParamsSource = paramsSource
      return {
        summary: ref({
          in_use: 3
        }),
        refresh: mockState.refreshRuntimeSummary
      }
    })
  }
})

vi.mock('@/composables/useAccountsViewLiveSync', async () => {
  const { ref } = await import('vue')
  return {
    useAccountsViewLiveSync: vi.fn(() => ({
      autoRefreshIntervals: [5, 10, 30],
      autoRefreshEnabled: ref(false),
      autoRefreshIntervalSeconds: ref(10),
      autoRefreshCountdown: ref(0),
      hasPendingListSync: ref(false),
      todayStatsByAccountId: ref({}),
      todayStatsLoading: ref(false),
      todayStatsError: ref(null),
      load: mockState.load,
      reload: mockState.reload,
      refreshAccountsIncrementally: mockState.refreshAccountsIncrementally,
      debouncedReload: mockState.debouncedReload,
      handlePageChange: vi.fn(),
      handlePageSizeChange: vi.fn(),
      refreshTodayStats: mockState.refreshTodayStats,
      syncPendingListChanges: mockState.syncPendingListChanges,
      setAutoRefreshEnabled: mockState.setAutoRefreshEnabled,
      handleAutoRefreshIntervalChange: mockState.handleAutoRefreshIntervalChange,
      enterAutoRefreshSilentWindow: mockState.enterAutoRefreshSilentWindow,
      pauseAutoRefresh: mockState.pauseAutoRefresh,
      resumeAutoRefresh: mockState.resumeAutoRefresh
    }))
  }
})

vi.mock('@/composables/useAccountActionMenu', async () => {
  const { reactive } = await import('vue')
  return {
    useAccountActionMenu: vi.fn(() => ({
      menu: reactive({ show: false, acc: null, pos: { x: 0, y: 0 } }),
      openMenu: vi.fn(),
      closeMenu: vi.fn(),
      syncMenuAccount: vi.fn(),
      clearMenuAccount: vi.fn()
    }))
  }
})

vi.mock('@/composables/useAccountViewMode', async () => {
  const { ref } = await import('vue')
  return {
    useAccountViewMode: vi.fn(() => ({
      viewMode: ref('table'),
      groupViewEnabled: ref(false)
    }))
  }
})

vi.mock('@/composables/useAccountsViewListPatching', () => ({
  useAccountsViewListPatching: vi.fn(() => ({
    patchAccountInList: vi.fn()
  }))
}))

vi.mock('@/composables/useSwipeSelect', () => ({
  useSwipeSelect: vi.fn()
}))

vi.mock('@/composables/useTableSelection', async () => {
  const { ref } = await import('vue')
  return {
    useTableSelection: vi.fn(() => ({
      selectedIds: ref([]),
      allVisibleSelected: ref(false),
      isSelected: vi.fn(() => false),
      setSelectedIds: vi.fn(),
      select: vi.fn(),
      deselect: vi.fn(),
      toggle: vi.fn(),
      clear: vi.fn(),
      removeMany: vi.fn(),
      toggleVisible: vi.fn(),
      selectVisible: vi.fn()
    }))
  }
})

vi.mock('@/composables/useAccountUsagePresentation', () => ({
  canAccountFetchUsage: vi.fn(() => false),
  invalidateAccountUsagePresentationCache: vi.fn(),
  refreshAccountUsagePresentation: vi.fn(),
  resolveActualUsageRefreshLoadOptions: vi.fn()
}))

vi.mock('@/composables/useModelImportExposureSync', async () => {
  const { ref } = await import('vue')
  return {
    useModelImportExposureSync: vi.fn(() => ({
      syncDialogOpen: ref(false),
      syncDialogModels: ref([]),
      syncDialogSubmitting: ref(false),
      handleImportedModels: vi.fn(),
      closeSyncDialog: vi.fn(),
      submitSyncDialog: vi.fn()
    }))
  }
})

vi.mock('@/utils/accountModelImport', () => ({
  buildAccountModelImportToastPayload: vi.fn(),
  resolveAccountModelImportErrorMessage: vi.fn(() => 'error'),
  shouldInvalidateModelInventory: vi.fn(() => false)
}))

import AccountsView from '../AccountsView.vue'
import {
  canAccountFetchUsage,
  invalidateAccountUsagePresentationCache,
  refreshAccountUsagePresentation,
  resolveActualUsageRefreshLoadOptions
} from '@/composables/useAccountUsagePresentation'

const SummaryBarStub = defineComponent({
  name: 'AccountStatusSummaryBar',
  props: ['summary', 'activeStatus', 'activeRuntimeView'],
  emits: ['select-status', 'select-runtime-view'],
  template: `
    <div>
      <div class="summary-total">{{ summary.total }}</div>
      <div class="summary-active">{{ summary.by_status.active }}</div>
      <div class="summary-in-use">{{ summary.in_use }}</div>
      <div class="summary-active-status">{{ activeStatus }}</div>
      <div class="summary-active-runtime">{{ activeRuntimeView }}</div>
      <button class="summary-total-button" @click="$emit('select-status', '')" />
      <button class="summary-active-button" @click="$emit('select-status', 'active')" />
      <button class="summary-rate-limited-button" @click="$emit('select-status', 'rate_limited')" />
      <button class="summary-remaining-available-button" @click="$emit('select-runtime-view', 'available_only')" />
      <button class="summary-in-use-button" @click="$emit('select-runtime-view', 'in_use_only')" />
    </div>
  `
})

const ToolbarStub = defineComponent({
  name: 'AccountsViewToolbar',
  props: ['platformCountSortOrder'],
  emits: ['refresh-usage', 'update:platform-count-sort-order'],
  template: `
    <div>
      <div class="toolbar-platform-sort">{{ platformCountSortOrder }}</div>
      <button class="toolbar-refresh-usage" @click="$emit('refresh-usage')" />
      <button
        class="toolbar-platform-sort-toggle"
        @click="$emit('update:platform-count-sort-order', 'count_desc')"
      />
    </div>
  `
})

const PlatformTabsStub = defineComponent({
  name: 'AccountPlatformTabs',
  inheritAttrs: false,
  props: ['platformCounts'],
  template: `
    <div>
      <div class="platform-tabs-order">all,anthropic,kiro,openai,copilot,grok,protocol_gateway,gemini,antigravity</div>
      <div class="platform-tabs-sort-attr">{{ $attrs['sort-order'] || '' }}</div>
    </div>
  `
})

const AccountsViewTableStub = defineComponent({
  name: 'AccountsViewTable',
  props: ['accounts', 'preserveInputOrder'],
  emits: ['edit'],
  setup(props: { accounts?: Array<{ name: string }>; preserveInputOrder?: boolean }) {
    const accountOrder = computed(() => (props.accounts ?? []).map((account) => account.name).join(','))
    return {
      accountOrder
    }
  },
  template: `
    <div>
      <div class="table-account-order">{{ accountOrder }}</div>
      <div class="table-preserve-input-order">{{ preserveInputOrder }}</div>
      <button
        v-if="accounts && accounts.length > 0"
        class="edit-first-account"
        @click="$emit('edit', accounts[0])"
      />
    </div>
  `
})

const DialogsHostStub = defineComponent({
  name: 'AccountsViewDialogsHost',
  props: ['showEdit', 'editLoading', 'editingAccount'],
  emits: ['close-edit'],
  template: `
    <div>
      <div class="dialog-show-edit">{{ String(showEdit) }}</div>
      <div class="dialog-edit-loading">{{ String(editLoading) }}</div>
      <div class="dialog-edit-account">{{ editingAccount?.name || '' }}</div>
      <button class="dialog-close-edit" @click="$emit('close-edit')" />
    </div>
  `
})

const mountView = () =>
  mount(AccountsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
        AccountsViewToolbar: ToolbarStub,
        AccountPlatformTabs: PlatformTabsStub,
        AccountStatusSummaryBar: SummaryBarStub,
        AccountLimitedSummaryBar: true,
        AccountBulkActionsBar: true,
        AccountGroupedView: true,
        AccountCardGrid: true,
        AccountsViewTable: AccountsViewTableStub,
        AccountsViewDialogsHost: DialogsHostStub,
        Pagination: true
      }
    }
  })

const createDeferred = <T,>() => {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('AccountsView', () => {
  beforeEach(() => {
    localStorage.clear()
    mockState.summaryParamsSource = null
    mockState.runtimeParamsSource = null
    mockState.tableParams = null
    mockState.tableItems = [
      { id: 1, name: 'OpenAI-1', platform: 'openai', type: 'apikey', status: 'active', schedulable: true },
      { id: 2, name: 'Gemini-1', platform: 'gemini', type: 'apikey', status: 'active', schedulable: true },
      { id: 3, name: 'OpenAI-2', platform: 'openai', type: 'apikey', status: 'active', schedulable: true },
      { id: 4, name: 'Gateway-1', platform: 'protocol_gateway', type: 'apikey', status: 'active', schedulable: true }
    ]
    mockState.routerPush.mockReset().mockResolvedValue(undefined)
    mockState.debouncedReload.mockReset()
    mockState.refreshAccountSummary.mockReset()
    mockState.refreshRuntimeSummary.mockReset()
    mockState.load.mockClear()
    mockState.reload.mockClear()
    mockState.refreshAccountsIncrementally.mockClear()
    mockState.refreshTodayStats.mockClear()
    mockState.syncPendingListChanges.mockClear()
    mockState.setAutoRefreshEnabled.mockClear()
    mockState.handleAutoRefreshIntervalChange.mockClear()
    mockState.enterAutoRefreshSilentWindow.mockClear()
    mockState.pauseAutoRefresh.mockClear()
    mockState.resumeAutoRefresh.mockClear()
    mockState.showSuccess.mockReset()
    mockState.showWarning.mockReset()
    mockState.showError.mockReset()
    mockState.showInfo.mockReset()
    mockState.getAllGroups.mockReset().mockResolvedValue([])
    mockState.getAllProxies.mockReset().mockResolvedValue([])
    mockState.getById.mockReset()
    vi.mocked(canAccountFetchUsage).mockImplementation(() => false)
    vi.mocked(invalidateAccountUsagePresentationCache).mockReset()
    vi.mocked(refreshAccountUsagePresentation).mockReset()
    vi.mocked(resolveActualUsageRefreshLoadOptions).mockImplementation(() => ({}))
  })

  it('keeps global summary counts independent from runtime_view and merges in-use separately', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(toValue(mockState.summaryParamsSource)).toEqual({
      platform: '',
      type: '',
      group: '',
      privacy_mode: '',
      search: '',
      lifecycle: 'normal',
      limited_view: 'all',
      limited_reason: ''
    })
    expect(toValue(mockState.runtimeParamsSource)).toEqual({
      platform: '',
      type: '',
      group: '',
      privacy_mode: '',
      search: '',
      lifecycle: 'normal',
      limited_view: 'all',
      limited_reason: '',
      runtime_view: 'all'
    })
    expect(wrapper.get('.summary-total').text()).toBe('88')
    expect(wrapper.get('.summary-active').text()).toBe('67')
    expect(wrapper.get('.summary-in-use').text()).toBe('3')

    await wrapper.get('.summary-in-use-button').trigger('click')
    await flushPromises()

    expect(toValue(mockState.summaryParamsSource)).toEqual({
      platform: '',
      type: '',
      group: '',
      privacy_mode: '',
      search: '',
      lifecycle: 'normal',
      limited_view: 'all',
      limited_reason: ''
    })
    expect(toValue(mockState.runtimeParamsSource)).toEqual({
      platform: '',
      type: '',
      group: '',
      privacy_mode: '',
      search: '',
      lifecycle: 'normal',
      limited_view: 'all',
      limited_reason: '',
      runtime_view: 'in_use_only'
    })
    expect(wrapper.get('.summary-total').text()).toBe('88')
    expect(wrapper.get('.summary-active').text()).toBe('67')
    expect(wrapper.get('.summary-in-use').text()).toBe('3')
    expect(wrapper.get('.summary-active-runtime').text()).toBe('in_use_only')
    expect(mockState.debouncedReload).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })

  it('switches to available-only runtime view when remaining available is selected', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.summary-remaining-available-button').trigger('click')
    await flushPromises()

    expect(mockState.tableParams.runtime_view).toBe('available_only')
    expect(mockState.tableParams.status).toBe('')
    expect(toValue(mockState.summaryParamsSource)).toEqual({
      platform: '',
      type: '',
      group: '',
      privacy_mode: '',
      search: '',
      lifecycle: 'normal',
      limited_view: 'all',
      limited_reason: ''
    })
    expect(toValue(mockState.runtimeParamsSource)).toEqual({
      platform: '',
      type: '',
      group: '',
      privacy_mode: '',
      search: '',
      lifecycle: 'normal',
      limited_view: 'all',
      limited_reason: '',
      runtime_view: 'available_only'
    })
    expect(wrapper.get('.summary-active-runtime').text()).toBe('available_only')
    expect(mockState.debouncedReload).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })

  it('clears runtime_view and status when returning to total', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.summary-in-use-button').trigger('click')
    await wrapper.get('.summary-total-button').trigger('click')
    await flushPromises()

    expect(mockState.tableParams.runtime_view).toBe('all')
    expect(mockState.tableParams.status).toBe('')
    expect(wrapper.get('.summary-active-status').text()).toBe('')
    expect(wrapper.get('.summary-active-runtime').text()).toBe('all')
    expect(mockState.debouncedReload).toHaveBeenCalledTimes(2)

    wrapper.unmount()
  })

  it('clears runtime_view before applying a normal status filter', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.summary-in-use-button').trigger('click')
    await wrapper.get('.summary-active-button').trigger('click')
    await flushPromises()

    expect(mockState.tableParams.runtime_view).toBe('all')
    expect(mockState.tableParams.status).toBe('active')
    expect(wrapper.get('.summary-active-status').text()).toBe('active')
    expect(wrapper.get('.summary-active-runtime').text()).toBe('all')

    wrapper.unmount()
  })

  it('filters rate-limited accounts in place when limited hiding is disabled', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.summary-rate-limited-button').trigger('click')
    await flushPromises()

    expect(mockState.routerPush).not.toHaveBeenCalled()
    expect(mockState.tableParams.status).toBe('rate_limited')
    expect(mockState.tableParams.limited_view).toBe('all')
    expect(mockState.debouncedReload).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })

  it('keeps limited accounts hidden on board selection when always-hide is enabled', async () => {
    localStorage.setItem('account-always-hide-limited-accounts', 'true')
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.summary-rate-limited-button').trigger('click')
    await flushPromises()

    expect(mockState.tableParams.status).toBe('rate_limited')
    expect(mockState.tableParams.limited_view).toBe('normal_only')

    wrapper.unmount()
  })

  it('defaults platform sort order to count_asc and sorts by the visible account list counts instead of summary totals', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('.toolbar-platform-sort').text()).toBe('count_asc')
    expect(wrapper.get('.table-account-order').text()).toBe('Gateway-1,Gemini-1,OpenAI-1,OpenAI-2')
    expect(wrapper.get('.table-preserve-input-order').text()).toBe('true')
    expect(wrapper.get('.platform-tabs-order').text()).toBe('all,anthropic,kiro,openai,copilot,grok,protocol_gateway,gemini,antigravity')
    expect(wrapper.get('.platform-tabs-sort-attr').text()).toBe('')

    wrapper.unmount()
  })

  it('restores the saved platform sort order from localStorage and still uses visible account list counts', async () => {
    localStorage.setItem('account-platform-count-sort-order', 'count_desc')
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('.toolbar-platform-sort').text()).toBe('count_desc')
    expect(wrapper.get('.table-account-order').text()).toBe('OpenAI-1,OpenAI-2,Gateway-1,Gemini-1')
    expect(wrapper.get('.platform-tabs-sort-attr').text()).toBe('')

    wrapper.unmount()
  })

  it('updates platform sort order from the toolbar and persists the reordered list locally', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.toolbar-platform-sort-toggle').trigger('click')
    await flushPromises()

    expect(wrapper.get('.toolbar-platform-sort').text()).toBe('count_desc')
    expect(wrapper.get('.table-account-order').text()).toBe('OpenAI-1,OpenAI-2,Gateway-1,Gemini-1')
    expect(localStorage.getItem('account-platform-count-sort-order')).toBe('count_desc')

    wrapper.unmount()
  })

  it('shows success toast details for actual usage refresh when all accounts succeed', async () => {
    vi.mocked(canAccountFetchUsage).mockReturnValue(true)
    vi.mocked(refreshAccountUsagePresentation).mockResolvedValue({
      total: 4,
      success: 4,
      activeSuccess: 3,
      fallbackSuccess: 1,
      failed: 0
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.toolbar-refresh-usage').trigger('click')
    await flushPromises()

    expect(invalidateAccountUsagePresentationCache).toHaveBeenCalledWith([1, 2, 3, 4])
    expect(refreshAccountUsagePresentation).toHaveBeenCalledWith(
      expect.any(Array),
      expect.objectContaining({
        force: true,
        concurrency: 4,
        resolveLoadOptions: expect.any(Function)
      })
    )
    expect(mockState.showSuccess).toHaveBeenCalledWith(
      'admin.accounts.refreshActualUsageSuccess',
      expect.objectContaining({
        details: [
          { text: 'admin.accounts.refreshActualUsageDetailActive', tone: 'success' },
          { text: 'admin.accounts.refreshActualUsageDetailFallback', tone: 'warning' }
        ]
      })
    )
    expect(mockState.showWarning).not.toHaveBeenCalled()
    expect(mockState.showError).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('shows warning toast details for partial actual usage refresh failures', async () => {
    vi.mocked(canAccountFetchUsage).mockReturnValue(true)
    vi.mocked(refreshAccountUsagePresentation).mockResolvedValue({
      total: 4,
      success: 3,
      activeSuccess: 2,
      fallbackSuccess: 1,
      failed: 1
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.toolbar-refresh-usage').trigger('click')
    await flushPromises()

    expect(mockState.showWarning).toHaveBeenCalledWith(
      'admin.accounts.refreshActualUsagePartial',
      expect.objectContaining({
        details: [
          { text: 'admin.accounts.refreshActualUsageDetailActive', tone: 'success' },
          { text: 'admin.accounts.refreshActualUsageDetailFallback', tone: 'warning' },
          { text: 'admin.accounts.refreshActualUsageDetailFailed', tone: 'error' }
        ]
      })
    )
    expect(mockState.showSuccess).not.toHaveBeenCalled()
    expect(mockState.showError).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('shows error toast details when actual usage refresh fully fails', async () => {
    vi.mocked(canAccountFetchUsage).mockReturnValue(true)
    vi.mocked(refreshAccountUsagePresentation).mockResolvedValue({
      total: 4,
      success: 0,
      activeSuccess: 0,
      fallbackSuccess: 0,
      failed: 4
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.toolbar-refresh-usage').trigger('click')
    await flushPromises()

    expect(mockState.showError).toHaveBeenCalledWith(
      'admin.accounts.refreshActualUsageFailedCount',
      expect.objectContaining({
        details: [{ text: 'admin.accounts.refreshActualUsageDetailFailed', tone: 'error' }]
      })
    )
    expect(mockState.showSuccess).not.toHaveBeenCalled()
    expect(mockState.showWarning).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('loads full account detail before populating the edit dialog', async () => {
    const deferred = createDeferred<any>()
    mockState.getById.mockReturnValue(deferred.promise)

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.edit-first-account').trigger('click')

    expect(mockState.getById).toHaveBeenCalledWith(
      4,
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
    expect(wrapper.get('.dialog-show-edit').text()).toBe('true')
    expect(wrapper.get('.dialog-edit-loading').text()).toBe('true')
    expect(wrapper.get('.dialog-edit-account').text()).toBe('')

    deferred.resolve({
      id: 4,
      name: 'Gateway-1 detail',
      platform: 'protocol_gateway',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      proxy_id: null,
      concurrency: 1,
      priority: 0,
      auto_pause_on_expired: false,
      error_message: null,
      last_used_at: null,
      expires_at: null,
      created_at: '2026-04-03T00:00:00Z',
      updated_at: '2026-04-03T00:00:00Z',
      rate_limited_at: null,
      rate_limit_reset_at: null,
      overload_until: null,
      temp_unschedulable_until: null,
      temp_unschedulable_reason: null,
      session_window_start: null,
      session_window_end: null,
      session_window_status: null,
      credentials: {
        api_key: 'sk-live-detail'
      },
      extra: {
        privacy_mode: 'strict'
      }
    })
    await flushPromises()

    expect(wrapper.get('.dialog-edit-loading').text()).toBe('false')
    expect(wrapper.get('.dialog-edit-account').text()).toBe('Gateway-1 detail')

    wrapper.unmount()
  })

  it('aborts the edit detail request when the dialog closes and does not show an error', async () => {
    const deferred = createDeferred<any>()
    let capturedSignal: AbortSignal | undefined
    mockState.getById.mockImplementation((_id: number, options?: { signal?: AbortSignal }) => {
      capturedSignal = options?.signal
      capturedSignal?.addEventListener('abort', () => {
        deferred.reject({ name: 'CanceledError', code: 'ERR_CANCELED' })
      })
      return deferred.promise
    })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.edit-first-account').trigger('click')
    expect(wrapper.get('.dialog-show-edit').text()).toBe('true')
    expect(wrapper.get('.dialog-edit-loading').text()).toBe('true')

    await wrapper.get('.dialog-close-edit').trigger('click')
    await flushPromises()

    expect(capturedSignal?.aborted).toBe(true)
    expect(wrapper.get('.dialog-show-edit').text()).toBe('false')
    expect(wrapper.get('.dialog-edit-loading').text()).toBe('false')
    expect(mockState.showError).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('cancels the previous edit detail request when edit is triggered again', async () => {
    const first = createDeferred<any>()
    const second = createDeferred<any>()
    const signals: AbortSignal[] = []

    mockState.getById
      .mockImplementationOnce((_id: number, options?: { signal?: AbortSignal }) => {
        if (options?.signal) {
          signals.push(options.signal)
          options.signal.addEventListener('abort', () => {
            first.reject({ name: 'CanceledError', code: 'ERR_CANCELED' })
          })
        }
        return first.promise
      })
      .mockImplementationOnce((_id: number, options?: { signal?: AbortSignal }) => {
        if (options?.signal) {
          signals.push(options.signal)
        }
        return second.promise
      })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.edit-first-account').trigger('click')
    await wrapper.get('.edit-first-account').trigger('click')

    expect(mockState.getById).toHaveBeenNthCalledWith(
      1,
      4,
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
    expect(mockState.getById).toHaveBeenNthCalledWith(
      2,
      4,
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
    expect(signals[0]?.aborted).toBe(true)
    expect(signals[1]?.aborted).toBe(false)

    second.resolve({
      id: 4,
      name: 'Gateway-1 newest detail',
      platform: 'protocol_gateway',
      type: 'apikey',
      status: 'active',
      schedulable: true,
      proxy_id: null,
      concurrency: 1,
      priority: 0,
      auto_pause_on_expired: false,
      error_message: null,
      last_used_at: null,
      expires_at: null,
      created_at: '2026-04-03T00:00:00Z',
      updated_at: '2026-04-03T00:00:00Z',
      rate_limited_at: null,
      rate_limit_reset_at: null,
      overload_until: null,
      temp_unschedulable_until: null,
      temp_unschedulable_reason: null,
      session_window_start: null,
      session_window_end: null,
      session_window_status: null,
      credentials: {},
      extra: {}
    })
    await flushPromises()

    expect(wrapper.get('.dialog-edit-loading').text()).toBe('false')
    expect(wrapper.get('.dialog-edit-account').text()).toBe('Gateway-1 newest detail')
    expect(mockState.showError).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('closes the edit dialog and reports an error when detail loading fails', async () => {
    const deferred = createDeferred<any>()
    mockState.getById.mockReturnValue(deferred.promise)

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.edit-first-account').trigger('click')
    expect(wrapper.get('.dialog-show-edit').text()).toBe('true')
    expect(wrapper.get('.dialog-edit-loading').text()).toBe('true')

    deferred.reject(new Error('detail failed'))
    await flushPromises()

    expect(wrapper.get('.dialog-show-edit').text()).toBe('false')
    expect(wrapper.get('.dialog-edit-loading').text()).toBe('false')
    expect(wrapper.get('.dialog-edit-account').text()).toBe('')
    expect(mockState.showError).toHaveBeenCalledWith('detail failed')

    wrapper.unmount()
  })
})
