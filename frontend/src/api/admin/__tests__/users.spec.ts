import { beforeEach, describe, expect, it, vi } from 'vitest'

const postMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    post: postMock,
  },
}))

describe('admin users api', () => {
  beforeEach(() => {
    postMock.mockReset()
  })

  it('sends Idempotency-Key when batch updating concurrency', async () => {
    postMock.mockResolvedValue({
      data: {
        matched: 2,
        success_count: 2,
        failed_count: 0,
        concurrency: 4,
        results: [],
      },
    })

    const { batchUpdateConcurrency } = await import('../users')
    await batchUpdateConcurrency(
      {
        concurrency: 4,
        search: 'alpha',
      },
      'users-batch-concurrency-test-1',
    )

    expect(postMock).toHaveBeenCalledWith(
      '/admin/users/batch-concurrency',
      {
        concurrency: 4,
        search: 'alpha',
      },
      {
        headers: {
          'Idempotency-Key': 'users-batch-concurrency-test-1',
        },
      },
    )
  })
})
