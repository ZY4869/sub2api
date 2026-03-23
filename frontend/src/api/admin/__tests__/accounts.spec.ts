import { beforeEach, describe, expect, it, vi } from 'vitest'

const getMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    get: getMock
  }
}))

describe('admin accounts api', () => {
  beforeEach(() => {
    getMock.mockReset()
  })

  it('normalizes archived group summaries from legacy PascalCase fields', async () => {
    getMock.mockResolvedValue({
      data: [
        {
          GroupID: 9,
          GroupName: 'OpenAI Archive',
          TotalCount: 12,
          AvailableCount: 7,
          InvalidCount: 5,
          LatestUpdatedAt: '2026-03-23T01:02:03Z'
        }
      ]
    })

    const { listArchivedGroups } = await import('../accounts')
    const result = await listArchivedGroups()

    expect(result).toEqual([
      {
        group_id: 9,
        group_name: 'OpenAI Archive',
        total_count: 12,
        available_count: 7,
        invalid_count: 5,
        latest_updated_at: '2026-03-23T01:02:03Z'
      }
    ])
  })
})
