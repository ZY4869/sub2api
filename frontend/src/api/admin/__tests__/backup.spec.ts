import { beforeEach, describe, expect, it, vi } from 'vitest'

const putMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    put: putMock
  }
}))

describe('admin backup api', () => {
  beforeEach(() => {
    putMock.mockReset()
  })

  it('sends step-up TOTP when updating S3 config', async () => {
    putMock.mockResolvedValue({ data: { bucket: 'b1' } })
    const { updateS3Config } = await import('../backup')

    await updateS3Config(
      {
        endpoint: 'https://r2.example.com',
        region: 'auto',
        bucket: 'b1',
        access_key_id: 'ak',
        secret_access_key: 'secret',
        prefix: 'backups/',
        force_path_style: false
      },
      { stepUpTotp: '112233' }
    )

    expect(putMock).toHaveBeenCalledWith(
      '/admin/backups/s3-config',
      expect.objectContaining({ bucket: 'b1' }),
      { headers: { 'X-Sub2API-Step-Up-TOTP': '112233' } }
    )
  })
})
