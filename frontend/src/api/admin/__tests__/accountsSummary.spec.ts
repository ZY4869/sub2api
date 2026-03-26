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
        },
        LimitedBreakdown: {
          total: 3,
          rate_429: 1,
          usage_5h: 1,
          usage_7d: 1
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
      },
      limited_breakdown: {
        total: 3,
        rate_429: 1,
        usage_5h: 1,
        usage_7d: 1
      }
    })
  })

  it('passes limited filters through summary request params', async () => {
    getMock.mockResolvedValue({
      data: {
        total: 0
      }
    })

    const { getStatusSummary } = await import('../accounts')
    await getStatusSummary({
      limited_view: 'limited_only',
      limited_reason: 'usage_7d'
    })

    expect(getMock).toHaveBeenCalledWith('/admin/accounts/summary', {
      params: {
        limited_view: 'limited_only',
        limited_reason: 'usage_7d'
      }
    })
  })
})
