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
    vi.mocked(adminAPI.accounts.list).mockReset()
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
})
