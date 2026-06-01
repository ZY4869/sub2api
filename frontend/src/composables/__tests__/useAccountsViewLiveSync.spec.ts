import { computed, reactive, ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAccountsViewLiveSync } from '@/composables/useAccountsViewLiveSync'
import type { AccountListRequestParams } from '@/utils/accountListSync'
import type { Account } from '@/types'

const intervalMocks = vi.hoisted(() => ({
  pause: vi.fn(),
  resume: vi.fn()
}))

const adminMocks = vi.hoisted(() => ({
  getBatchTodayStats: vi.fn(),
  listWithEtag: vi.fn(),
  list: vi.fn(),
}))

vi.mock('@vueuse/core', () => ({
  useIntervalFn: vi.fn(() => ({
    pause: intervalMocks.pause,
    resume: intervalMocks.resume
  }))
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getBatchTodayStats: adminMocks.getBatchTodayStats,
      listWithEtag: adminMocks.listWithEtag,
      list: adminMocks.list,
    }
  }
}))

const createAccount = (overrides: Partial<Account> = {}): Account => ({
  id: 1,
  name: 'Primary Account',
  platform: 'openai',
  type: 'oauth',
  proxy_id: null,
  concurrency: 1,
  priority: 0,
  status: 'active',
  error_message: null,
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: false,
  auto_renew_enabled: false,
  auto_renew_period: 'month',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  schedulable: true,
  rate_limited_at: null,
  rate_limit_reset_at: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  session_window_start: null,
  session_window_end: null,
  session_window_status: null,
  ...overrides
})

function createLiveSync(options: {
  baseLoad?: () => Promise<void>
  baseReload?: () => Promise<void>
} = {}) {
  const accounts = ref<Account[]>([])
  const loading = ref(false)
  const params = reactive<AccountListRequestParams>({
    platform: '',
    type: '',
    status: '',
    group: '',
    search: ''
  })
  const pagination = reactive({
    page: 1,
    page_size: 20,
    total: 0,
    pages: 0
  })
  const hiddenColumns = reactive(new Set<string>())
  const baseLoad = vi.fn(options.baseLoad ?? (async () => {}))
  const baseReload = vi.fn(options.baseReload ?? (async () => {}))
  const syncAccountRefs = vi.fn()

  const liveSync = useAccountsViewLiveSync({
    accounts,
    loading,
    params,
    pagination,
    hiddenColumns,
    baseLoad,
    baseReload,
    baseDebouncedReload: vi.fn(),
    baseHandlePageChange: vi.fn(),
    baseHandlePageSizeChange: vi.fn(),
    isAnyModalOpen: computed(() => false),
    isActionMenuOpen: computed(() => false),
    syncAccountRefs
  })

  return {
    accounts,
    params,
    pagination,
    baseLoad,
    liveSync
  }
}

describe('useAccountsViewLiveSync', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('loads first page with lite flag and refreshes today stats', async () => {
    adminMocks.getBatchTodayStats.mockResolvedValue({
      stats: {
        '1': {
          requests: 2,
          tokens: 4,
          cost: 6
        }
      }
    })

    const { accounts, params, pagination, baseLoad, liveSync } = createLiveSync({
      baseLoad: async () => {
        expect(params.lite).toBe('1')
        accounts.value = [createAccount()]
        pagination.total = 1
        pagination.pages = 1
      }
    })

    await liveSync.load()

    expect(baseLoad).toHaveBeenCalledTimes(1)
    expect(params.lite).toBeUndefined()
    expect(liveSync.todayStatsByAccountId.value['1']).toEqual({
      requests: 2,
      tokens: 4,
      cost: 6
    })
  })

  it('restores and updates auto-refresh preferences', () => {
    localStorage.setItem(
      'account-auto-refresh',
      JSON.stringify({ enabled: true, interval_seconds: 10 })
    )

    const { liveSync } = createLiveSync()

    expect(liveSync.autoRefreshEnabled.value).toBe(true)
    expect(liveSync.autoRefreshIntervalSeconds.value).toBe(10)

    liveSync.setAutoRefreshEnabled(false)
    liveSync.handleAutoRefreshIntervalChange(5)
    liveSync.setAutoRefreshEnabled(true)

    expect(liveSync.autoRefreshIntervalSeconds.value).toBe(5)
    expect(intervalMocks.pause).toHaveBeenCalled()
    expect(intervalMocks.resume).toHaveBeenCalled()
  })

  it('hydrates the next page from prefetch cache before falling back to base page change', async () => {
    adminMocks.getBatchTodayStats.mockResolvedValue({ stats: {} })
    adminMocks.list.mockResolvedValue({
      items: [createAccount({ id: 2, name: 'Prefetched Account' })],
      total: 40,
      page: 2,
      page_size: 20,
      pages: 2,
    })

    const { accounts, pagination, baseLoad, liveSync } = createLiveSync({
      baseLoad: async () => {
        accounts.value = [createAccount({ id: 1, name: 'Current Page Account' })]
        pagination.total = 40
        pagination.page = 1
        pagination.page_size = 20
        pagination.pages = 2
      },
    })

    await liveSync.load()
    await liveSync.handlePageChange(2)
    await Promise.resolve()
    await Promise.resolve()

    expect(baseLoad).toHaveBeenCalledTimes(1)
    expect(accounts.value[0]?.id).toBe(2)
    expect(pagination.page).toBe(2)
  })
})
