import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, toValue } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockState = vi.hoisted(() => ({
  summaryParamsSource: null as any,
  runtimeParamsSource: null as any,
  tableParams: null as any,
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
  getAllGroups: vi.fn(),
  getAllProxies: vi.fn()
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
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
    showError: vi.fn(),
    showInfo: vi.fn()
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
      list: vi.fn()
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
        items: ref([]),
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
            openai: 50
          },
          limited_breakdown: {
            total: 5,
            rate_429: 2,
            usage_5h: 2,
            usage_7d: 1
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
      <button class="summary-in-use-button" @click="$emit('select-runtime-view', 'in_use_only')" />
    </div>
  `
})

const ToolbarStub = defineComponent({
  name: 'AccountsViewToolbar',
  props: ['platformCountSortOrder'],
  emits: ['update:platform-count-sort-order'],
  template: `
    <div>
      <div class="toolbar-platform-sort">{{ platformCountSortOrder }}</div>
      <button
        class="toolbar-platform-sort-toggle"
        @click="$emit('update:platform-count-sort-order', 'count_desc')"
      />
    </div>
  `
})

const PlatformTabsStub = defineComponent({
  name: 'AccountPlatformTabs',
  props: ['sortOrder'],
  template: '<div class="platform-tabs-sort-order">{{ sortOrder }}</div>'
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
        AccountsViewTable: true,
        AccountsViewDialogsHost: true,
        Pagination: true
      }
    }
  })

describe('AccountsView', () => {
  beforeEach(() => {
    localStorage.clear()
    mockState.summaryParamsSource = null
    mockState.runtimeParamsSource = null
    mockState.tableParams = null
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
    mockState.getAllGroups.mockReset().mockResolvedValue([])
    mockState.getAllProxies.mockReset().mockResolvedValue([])
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

  it('defaults platform tab sort order to count_asc and passes it to the toolbar and tabs', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('.toolbar-platform-sort').text()).toBe('count_asc')
    expect(wrapper.get('.platform-tabs-sort-order').text()).toBe('count_asc')

    wrapper.unmount()
  })

  it('restores the saved platform tab sort order from localStorage', async () => {
    localStorage.setItem('account-platform-count-sort-order', 'count_desc')
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('.toolbar-platform-sort').text()).toBe('count_desc')
    expect(wrapper.get('.platform-tabs-sort-order').text()).toBe('count_desc')

    wrapper.unmount()
  })

  it('updates platform tab sort order from the toolbar and persists it locally', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.toolbar-platform-sort-toggle').trigger('click')
    await flushPromises()

    expect(wrapper.get('.toolbar-platform-sort').text()).toBe('count_desc')
    expect(wrapper.get('.platform-tabs-sort-order').text()).toBe('count_desc')
    expect(localStorage.getItem('account-platform-count-sort-order')).toBe('count_desc')

    wrapper.unmount()
  })
})
