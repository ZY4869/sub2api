import { beforeEach, describe, expect, it, vi } from 'vitest'

const getMock = vi.fn()
const postMock = vi.fn()
const putMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    get: getMock,
    post: postMock,
    put: putMock,
  },
}))

describe('admin users api', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
    putMock.mockReset()
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

  it('reads and updates user platform quotas', async () => {
    getMock.mockResolvedValue({ data: [{ platform: 'openai' }] })
    putMock.mockResolvedValue({ data: [{ platform: 'gemini' }] })

    const { getUserPlatformQuotas, updateUserPlatformQuotas } = await import('../users')
    const read = await getUserPlatformQuotas(123)
    const saved = await updateUserPlatformQuotas(123, [
      {
        platform: 'openai',
        daily_limit_usd: 10,
        weekly_limit_usd: null,
        monthly_limit_usd: 100,
      },
    ])

    expect(read).toEqual([{ platform: 'openai' }])
    expect(saved).toEqual([{ platform: 'gemini' }])
    expect(getMock).toHaveBeenCalledWith('/admin/users/123/platform-quotas')
    expect(putMock).toHaveBeenCalledWith('/admin/users/123/platform-quotas', {
      items: [
        {
          platform: 'openai',
          daily_limit_usd: 10,
          weekly_limit_usd: null,
          monthly_limit_usd: 100,
        },
      ],
    })
  })
})
