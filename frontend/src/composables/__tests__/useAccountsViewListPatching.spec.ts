import { reactive, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useAccountsViewListPatching } from '@/composables/useAccountsViewListPatching'
import type { AccountListRequestParams } from '@/utils/accountListSync'
import type { Account } from '@/types'

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

describe('useAccountsViewListPatching', () => {
  it('patches in-place rows and syncs linked refs', () => {
    const accounts = ref([
      createAccount({ id: 1, current_concurrency: 2 })
    ])
    const pagination = reactive({
      page: 1,
      page_size: 20,
      total: 1,
      pages: 1
    })
    const hasPendingListSync = ref(false)
    const syncAccountRefs = vi.fn()

    const { patchAccountInList } = useAccountsViewListPatching({
      accounts,
      params: reactive<AccountListRequestParams>({
        platform: '',
        type: '',
        status: '',
        group: '',
        search: ''
      }),
      pagination,
      hasPendingListSync,
      removeSelectedAccounts: vi.fn(),
      syncAccountRefs,
      clearRemovedAccount: vi.fn()
    })

    patchAccountInList(createAccount({ id: 1, updated_at: '2026-01-02T00:00:00Z' }))

    expect(accounts.value[0]).toEqual(
      expect.objectContaining({
        id: 1,
        updated_at: '2026-01-02T00:00:00Z',
        current_concurrency: 2
      })
    )
    expect(syncAccountRefs).toHaveBeenCalledWith(expect.objectContaining({ id: 1 }))
    expect(hasPendingListSync.value).toBe(false)
  })

  it('removes rows that no longer match current filters and marks list for sync', () => {
    const accounts = ref([
      createAccount({ id: 1, group_ids: [2] }),
      createAccount({ id: 2, group_ids: [2] })
    ])
    const pagination = reactive({
      page: 1,
      page_size: 20,
      total: 2,
      pages: 1
    })
    const hasPendingListSync = ref(false)
    const removeSelectedAccounts = vi.fn()
    const clearRemovedAccount = vi.fn()

    const { patchAccountInList } = useAccountsViewListPatching({
      accounts,
      params: reactive<AccountListRequestParams>({
        platform: '',
        type: '',
        status: '',
        group: '2',
        search: ''
      }),
      pagination,
      hasPendingListSync,
      removeSelectedAccounts,
      syncAccountRefs: vi.fn(),
      clearRemovedAccount
    })

    patchAccountInList(createAccount({ id: 1, group_ids: [3] }))

    expect(accounts.value.map(account => account.id)).toEqual([2])
    expect(pagination.total).toBe(1)
    expect(hasPendingListSync.value).toBe(true)
    expect(removeSelectedAccounts).toHaveBeenCalledWith([1])
    expect(clearRemovedAccount).toHaveBeenCalledWith(1)
  })
})
