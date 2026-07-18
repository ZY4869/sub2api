import { beforeEach, describe, expect, it, vi } from 'vitest'

const getMock = vi.fn()
const postMock = vi.fn()
const putMock = vi.fn()
const deleteMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    get: getMock,
    post: postMock,
    put: putMock,
    delete: deleteMock
  }
}))

describe('admin prompt audit api', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
    putMock.mockReset()
    deleteMock.mockReset()
  })

  it('sends step-up TOTP when updating config', async () => {
    putMock.mockResolvedValue({ data: { config_version: 2 } })
    const { updateConfig } = await import('../promptAudit')

    await updateConfig(
      {
        expected_config_version: 1,
        enabled: false,
        blocking_enabled: false,
        store_pass_events: false,
        strategy: 'priority',
        worker_count: 4,
        queue_capacity: 32768,
        scanners: ['jailbreak'],
        all_groups: true,
        group_ids: [],
        endpoints: []
      },
      { stepUpTotp: '123456' }
    )

    expect(putMock).toHaveBeenCalledWith(
      '/admin/prompt-audit/config',
      expect.objectContaining({ expected_config_version: 1 }),
      { headers: { 'X-Sub2API-Step-Up-TOTP': '123456' } }
    )
  })

  it('passes delete preview token and step-up header for filtered delete', async () => {
    postMock.mockResolvedValue({ data: { deleted: 3 } })
    const { deleteByFilter } = await import('../promptAudit')

    await deleteByFilter(
      { page: 1, page_size: 20, decision: 'critical' },
      {
        matched: 3,
        snapshot_max_id: 99,
        filter_hash: 'hash-1',
        confirmation_token: 'token-1'
      },
      { stepUpTotp: '654321' }
    )

    expect(postMock).toHaveBeenCalledWith(
      '/admin/prompt-audit/events/delete-by-filter',
      {
        filter: { page: 1, page_size: 20, decision: 'critical' },
        snapshot_max_id: 99,
        filter_hash: 'hash-1',
        confirmation_token: 'token-1',
        confirm: true
      },
      { headers: { 'X-Sub2API-Step-Up-TOTP': '654321' } }
    )
  })
})
