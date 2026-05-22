import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import { useAccountsPagePrefetch } from '@/composables/useAccountsPagePrefetch'

const accountListMock = vi.hoisted(() => vi.fn())

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: accountListMock,
    },
  },
}))

describe('useAccountsPagePrefetch', () => {
  afterEach(() => {
    vi.clearAllMocks()
  })

  it('stores and reads cached pages by page + page size + filters', () => {
    const scope = effectScope()
    const subject = scope.run(() => useAccountsPagePrefetch())!
    const params = {
      search: 'demo',
      status: 'active',
      lite: '1',
    }

    subject.storePageSnapshot({
      items: [{ id: 1, name: 'A' } as any],
      total: 40,
      page: 2,
      page_size: 20,
      pages: 2,
    }, params as any)

    expect(subject.getCachedPage(2, 20, params as any)?.items).toEqual([
      { id: 1, name: 'A' },
    ])
    expect(subject.getCachedPage(1, 20, params as any)).toBeNull()
    scope.stop()
  })

  it('deduplicates inflight prefetch requests and caches the response', async () => {
    const scope = effectScope()
    const subject = scope.run(() => useAccountsPagePrefetch())!
    const params = { search: 'demo' }

    accountListMock.mockResolvedValue({
      items: [{ id: 3, name: 'Prefetched' }],
      total: 60,
      page: 3,
      page_size: 20,
      pages: 3,
    })

    const [first, second] = await Promise.all([
      subject.prefetchPage(3, 20, params as any),
      subject.prefetchPage(3, 20, params as any),
    ])

    expect(accountListMock).toHaveBeenCalledTimes(1)
    expect(first?.items[0]?.id).toBe(3)
    expect(second?.items[0]?.id).toBe(3)
    expect(subject.getCachedPage(3, 20, params as any)?.items[0]?.id).toBe(3)
    scope.stop()
  })
})
