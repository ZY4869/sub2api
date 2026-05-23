import { describe, expect, it, beforeEach, vi } from 'vitest'

import { batchUpdate, generate, list } from '../redeem'

const mockPost = vi.fn()
const mockGet = vi.fn()

vi.mock('../../client', () => ({
  apiClient: {
    get: (...args: any[]) => mockGet(...args),
    post: (...args: any[]) => mockPost(...args),
  },
}))

describe('redeem API', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockPost.mockResolvedValue({ data: [] })
    mockGet.mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20, pages: 0 } })
  })

  it('passes negative subscription validity_days through to the backend', async () => {
    await generate(2, 'subscription', 10, 3, -7)

    expect(mockPost).toHaveBeenCalledWith('/admin/redeem-codes/generate', {
      count: 2,
      type: 'subscription',
      value: 10,
      group_id: 3,
      validity_days: -7,
    })
  })

  it('passes redeem code expiration through to the backend', async () => {
    await generate(1, 'balance', 5, null, undefined, '2026-06-01T08:30')

    expect(mockPost).toHaveBeenCalledWith('/admin/redeem-codes/generate', {
      count: 1,
      type: 'balance',
      value: 5,
      expires_at: '2026-06-01T08:30',
    })
  })

  it('passes relative expiration days through to the backend', async () => {
    await generate(1, 'balance', 5, null, undefined, null, 14)

    expect(mockPost).toHaveBeenCalledWith('/admin/redeem-codes/generate', {
      count: 1,
      type: 'balance',
      value: 5,
      expires_in_days: 14,
    })
  })

  it('passes sort params when listing redeem codes', async () => {
    await list(2, 50, { status: 'disabled', sort_by: 'expires_at', sort_order: 'asc' })

    expect(mockGet).toHaveBeenCalledWith('/admin/redeem-codes', {
      params: {
        page: 2,
        page_size: 50,
        status: 'disabled',
        sort_by: 'expires_at',
        sort_order: 'asc',
      },
      signal: undefined,
    })
  })

  it('passes subscription sort params when listing redeem codes', async () => {
    await list(1, 20, { sort_by: 'validity_days', sort_order: 'desc' })

    expect(mockGet).toHaveBeenCalledWith('/admin/redeem-codes', {
      params: {
        page: 1,
        page_size: 20,
        sort_by: 'validity_days',
        sort_order: 'desc',
      },
      signal: undefined,
    })
  })

  it('passes group sort params when exporting redeem codes', async () => {
    mockGet.mockResolvedValueOnce({ data: new Blob() })

    const { exportCodes } = await import('../redeem')
    await exportCodes({ sort_by: 'group_id', sort_order: 'asc' })

    expect(mockGet).toHaveBeenCalledWith('/admin/redeem-codes/export', {
      params: {
        sort_by: 'group_id',
        sort_order: 'asc',
      },
      responseType: 'blob',
    })
  })

  it('posts batch update fields without local mutation', async () => {
    await batchUpdate([1, 2], { status: 'disabled', expires_at: null })

    expect(mockPost).toHaveBeenCalledWith('/admin/redeem-codes/batch-update', {
      ids: [1, 2],
      fields: {
        status: 'disabled',
        expires_at: null,
      },
    })
  })
})
