import { beforeEach, describe, expect, it, vi } from 'vitest'
import axios from 'axios'
import { clearAuthStorage, refreshAccessToken } from '../tokenRefresh'

vi.mock('axios', () => ({
  default: {
    post: vi.fn()
  }
}))

const postMock = vi.mocked(axios.post)

function successfulRefresh(accessToken: string, refreshToken = 'refresh-next', expiresIn = 3600) {
  return {
    data: {
      code: 0,
      data: {
        access_token: accessToken,
        refresh_token: refreshToken,
        expires_in: expiresIn,
        token_type: 'Bearer'
      }
    }
  }
}

function setStoredTokens(accessToken = 'access-old', refreshToken = 'refresh-old', expiresAt = Date.now() - 1000) {
  localStorage.setItem('auth_token', accessToken)
  localStorage.setItem('refresh_token', refreshToken)
  localStorage.setItem('token_expires_at', String(expiresAt))
}

describe('tokenRefresh', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    clearAuthStorage()
    Object.defineProperty(navigator, 'locks', {
      configurable: true,
      value: undefined
    })
  })

  it('dedupes same-tab in-flight refresh requests', async () => {
    setStoredTokens()
    let resolveRefresh!: (value: unknown) => void
    postMock.mockReturnValue(new Promise((resolve) => {
      resolveRefresh = resolve
    }) as ReturnType<typeof axios.post>)

    const first = refreshAccessToken({ force: true })
    const second = refreshAccessToken({ force: true })
    expect(postMock).toHaveBeenCalledTimes(1)

    resolveRefresh(successfulRefresh('access-new'))
    await expect(Promise.all([first, second])).resolves.toEqual([
      expect.objectContaining({ access_token: 'access-new' }),
      expect.objectContaining({ access_token: 'access-new' })
    ])
    expect(localStorage.getItem('auth_token')).toBe('access-new')
  })

  it('uses Web Locks when available', async () => {
    setStoredTokens()
    const requestLock = vi.fn(async (_name: string, callback: () => Promise<unknown>) => callback())
    Object.defineProperty(navigator, 'locks', {
      configurable: true,
      value: { request: requestLock }
    })
    postMock.mockResolvedValue(successfulRefresh('access-lock'))

    await refreshAccessToken({ force: true })

    expect(requestLock).toHaveBeenCalledWith('sub2api-token-refresh', expect.any(Function))
    expect(postMock).toHaveBeenCalledTimes(1)
  })

  it('reuses a newer token written by another tab', async () => {
    setStoredTokens('access-newer', 'refresh-newer', Date.now() + 5 * 60 * 1000)

    const result = await refreshAccessToken({ force: true, staleAccessToken: 'access-old' })

    expect(result.access_token).toBe('access-newer')
    expect(result.refresh_token).toBe('refresh-newer')
    expect(postMock).not.toHaveBeenCalled()
  })

  it('clears failed in-flight state so a later refresh can retry', async () => {
    setStoredTokens()
    postMock.mockRejectedValueOnce(new Error('network down'))

    await expect(refreshAccessToken({ force: true })).rejects.toThrow('network down')

    postMock.mockResolvedValueOnce(successfulRefresh('access-after-retry'))
    await expect(refreshAccessToken({ force: true })).resolves.toEqual(
      expect.objectContaining({ access_token: 'access-after-retry' })
    )
    expect(postMock).toHaveBeenCalledTimes(2)
  })
})
