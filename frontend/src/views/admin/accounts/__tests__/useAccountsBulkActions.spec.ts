import { describe, expect, it, vi, beforeEach } from 'vitest'
import { ref, reactive } from 'vue'
import { useAccountsBulkActions } from '../useAccountsBulkActions'
import { adminAPI } from '@/api/admin'

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: vi.fn(),
      delete: vi.fn(),
      batchClearError: vi.fn(),
      batchRefresh: vi.fn(),
      bulkUpdate: vi.fn()
    }
  }
}))

function createCtx(overrides: Record<string, unknown> = {}) {
  return {
    accounts: ref([]),
    appStore: {
      showWarning: vi.fn(),
      showError: vi.fn(),
      showSuccess: vi.fn()
    },
    batchTestAccounts: ref([]),
    batchTestDefaultModelStrategy: ref('auto'),
    batchTestDefaultTestMode: ref('health_check'),
    bulkEditFilters: ref(null),
    bulkEditFiltersTotal: ref(null),
    clearSelection: vi.fn(),
    load: vi.fn(),
    params: reactive({
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      group: '',
      search: 'team-a',
      lifecycle: 'normal',
      privacy_mode: '',
      limited_view: 'all',
      limited_reason: '',
      runtime_view: 'available_only'
    }),
    reload: vi.fn(),
    selIds: ref([]),
    selPlatforms: ref([]),
    showArchiveSelected: ref(false),
    showBatchTest: ref(false),
    showBulkEdit: ref(false),
    setSelectedIds: vi.fn(),
    t: (key: string) => key,
    refreshAccountSummarySafe: vi.fn(),
    usageRefreshing: ref(false),
    ...overrides
  }
}

describe('useAccountsBulkActions', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    vi.mocked(adminAPI.accounts.list).mockReset()
    vi.mocked(adminAPI.accounts.delete).mockReset()
  })

  it('adds ungrouped filter and resolves target total before opening filtered bulk edit', async () => {
    vi.mocked(adminAPI.accounts.list).mockResolvedValue({
      items: [],
      total: 12,
      page: 1,
      page_size: 1,
      pages: 12
    } as any)
    const ctx = createCtx()
    const actions = useAccountsBulkActions(ctx)

    await actions.openBulkEditFilteredModal({ excludeGrouped: true })

    expect(adminAPI.accounts.list).toHaveBeenCalledWith(1, 1, {
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      group: 'ungrouped',
      search: 'team-a',
      lifecycle: 'normal',
      privacy_mode: undefined,
      limited_view: 'all',
      limited_reason: undefined,
      runtime_view: 'available_only'
    })
    expect(ctx.bulkEditFilters.value).toEqual(expect.objectContaining({
      group: 'ungrouped'
    }))
    expect(ctx.bulkEditFiltersTotal.value).toBe(12)
    expect(ctx.showBulkEdit.value).toBe(true)
  })

  it('does not override a concrete group filter when exclude grouped is requested', async () => {
    vi.mocked(adminAPI.accounts.list).mockResolvedValue({
      items: [],
      total: 3,
      page: 1,
      page_size: 1,
      pages: 3
    } as any)
    const ctx = createCtx({
      params: reactive({
        platform: 'openai',
        type: '',
        status: '',
        group: '9',
        search: '',
        lifecycle: 'normal',
        privacy_mode: '',
        limited_view: 'all',
        limited_reason: '',
        runtime_view: 'all'
      })
    })
    const actions = useAccountsBulkActions(ctx)

    await actions.openBulkEditFilteredModal({ excludeGrouped: true })

    expect(ctx.appStore.showWarning).toHaveBeenCalledWith(
      'admin.accounts.bulkEdit.excludeGroupedSpecificGroupDisabled'
    )
    expect(adminAPI.accounts.list).toHaveBeenCalledWith(1, 1, expect.objectContaining({
      group: '9'
    }))
    expect(ctx.bulkEditFilters.value).toEqual(expect.objectContaining({
      group: '9'
    }))
  })

  it('limits bulk delete concurrency and refreshes after success', async () => {
    vi.stubGlobal('confirm', vi.fn(() => true))
    let releaseDeletes!: () => void
    const deleteGate = new Promise<void>((resolve) => {
      releaseDeletes = resolve
    })
    let activeDeletes = 0
    let maxActiveDeletes = 0

    vi.mocked(adminAPI.accounts.delete).mockImplementation(async () => {
      activeDeletes += 1
      maxActiveDeletes = Math.max(maxActiveDeletes, activeDeletes)
      await deleteGate
      activeDeletes -= 1
      return { message: 'ok' }
    })

    const ctx = createCtx({
      selIds: ref([1, 2, 3, 4, 5, 6, 7, 8, 9, 10])
    })
    const actions = useAccountsBulkActions(ctx)

    const promise = actions.handleBulkDelete()

    expect(adminAPI.accounts.delete).toHaveBeenCalledTimes(4)
    expect(maxActiveDeletes).toBe(4)

    releaseDeletes()
    await promise

    expect(adminAPI.accounts.delete).toHaveBeenCalledTimes(10)
    expect(maxActiveDeletes).toBeLessThanOrEqual(4)
    expect(ctx.clearSelection).toHaveBeenCalled()
    expect(ctx.reload).toHaveBeenCalled()
  })

  it('selects all accounts matching the current filters across pages', async () => {
    vi.mocked(adminAPI.accounts.list)
      .mockResolvedValueOnce({
        items: [{ id: 1 }, { id: 2 }],
        total: 1001,
        page: 1,
        page_size: 1000,
        pages: 2
      } as any)
      .mockResolvedValueOnce({
        items: [{ id: 2 }, { id: 3 }],
        total: 1001,
        page: 2,
        page_size: 1000,
        pages: 2
      } as any)
    const ctx = createCtx()
    const actions = useAccountsBulkActions(ctx)

    await actions.handleSelectFilteredAccounts()

    expect(adminAPI.accounts.list).toHaveBeenNthCalledWith(1, 1, 1000, {
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      group: undefined,
      search: 'team-a',
      lifecycle: 'normal',
      privacy_mode: undefined,
      limited_view: 'all',
      limited_reason: undefined,
      runtime_view: 'available_only'
    })
    expect(adminAPI.accounts.list).toHaveBeenNthCalledWith(2, 2, 1000, {
      platform: 'openai',
      type: 'apikey',
      status: 'active',
      group: undefined,
      search: 'team-a',
      lifecycle: 'normal',
      privacy_mode: undefined,
      limited_view: 'all',
      limited_reason: undefined,
      runtime_view: 'available_only'
    })
    expect(ctx.setSelectedIds).toHaveBeenCalledWith([1, 2, 3])
    expect(ctx.appStore.showSuccess).toHaveBeenCalledWith(
      'admin.accounts.bulkActions.selectFilteredSuccess'
    )
  })
})
