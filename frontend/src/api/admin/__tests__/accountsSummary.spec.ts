import { beforeEach, describe, expect, it, vi } from 'vitest'

const getMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    get: getMock
  }
}))

describe('admin accounts summary api', () => {
  beforeEach(() => {
    getMock.mockReset()
  })

  it('normalizes account summary payload fields', async () => {
    getMock.mockResolvedValue({
      data: {
        Total: 9,
        ByStatus: {
          active: 6,
          inactive: 2,
          error: 1
        },
        RateLimited: 3,
        TempUnschedulable: 2,
        Overloaded: 1,
        Paused: 4,
        ByPlatform: {
          openai: 5,
          kiro: 4
        }
      }
    })

    const { getStatusSummary } = await import('../accounts')
    const result = await getStatusSummary()

    expect(result).toEqual({
      total: 9,
      by_status: {
        active: 6,
        inactive: 2,
        error: 1
      },
      rate_limited: 3,
      temp_unschedulable: 2,
      overloaded: 1,
      paused: 4,
      by_platform: {
        openai: 5,
        kiro: 4
      }
    })
  })
})
