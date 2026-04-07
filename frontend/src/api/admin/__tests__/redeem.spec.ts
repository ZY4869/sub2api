import { describe, expect, it, beforeEach, vi } from 'vitest'

import { generate } from '../redeem'

const mockPost = vi.fn()

vi.mock('../../client', () => ({
  apiClient: {
    post: (...args: any[]) => mockPost(...args),
  },
}))

describe('redeem API', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockPost.mockResolvedValue({ data: [] })
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
})
