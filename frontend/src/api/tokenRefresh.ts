import axios from 'axios'
import type { ApiResponse } from '@/types'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

const AUTH_TOKEN_KEY = 'auth_token'
const REFRESH_TOKEN_KEY = 'refresh_token'
const TOKEN_EXPIRES_AT_KEY = 'token_expires_at'
const AUTH_USER_KEY = 'auth_user'
const TOKEN_REFRESH_LOCK = 'sub2api-token-refresh'
const FRESH_TOKEN_BUFFER_MS = 30 * 1000

export interface RefreshTokenResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: string
}

export interface TokenRefreshOptions {
  force?: boolean
  staleAccessToken?: string | null
}

let inFlight: Promise<RefreshTokenResponse> | null = null

function storedRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}

function storedAccessToken(): string | null {
  return localStorage.getItem(AUTH_TOKEN_KEY)
}

function storedExpiresAt(): number | null {
  const raw = localStorage.getItem(TOKEN_EXPIRES_AT_KEY)
  if (!raw) return null
  const parsed = Number.parseInt(raw, 10)
  return Number.isFinite(parsed) ? parsed : null
}

function persistTokenPair(data: RefreshTokenResponse): void {
  localStorage.setItem(AUTH_TOKEN_KEY, data.access_token)
  localStorage.setItem(REFRESH_TOKEN_KEY, data.refresh_token)
  localStorage.setItem(TOKEN_EXPIRES_AT_KEY, String(Date.now() + data.expires_in * 1000))
}

function existingFreshToken(options?: TokenRefreshOptions): RefreshTokenResponse | null {
  const accessToken = storedAccessToken()
  const refreshToken = storedRefreshToken()
  const expiresAt = storedExpiresAt()
  if (!accessToken || !refreshToken || !expiresAt) return null
  if (options?.staleAccessToken && accessToken !== options.staleAccessToken) {
    return {
      access_token: accessToken,
      refresh_token: refreshToken,
      expires_in: Math.max(1, Math.floor((expiresAt - Date.now()) / 1000)),
      token_type: 'Bearer'
    }
  }
  if (!options?.force && expiresAt - Date.now() > FRESH_TOKEN_BUFFER_MS) {
    return {
      access_token: accessToken,
      refresh_token: refreshToken,
      expires_in: Math.max(1, Math.floor((expiresAt - Date.now()) / 1000)),
      token_type: 'Bearer'
    }
  }
  return null
}

async function withRefreshLock<T>(fn: () => Promise<T>): Promise<T> {
  const locks = (navigator as Navigator & {
    locks?: { request: <R>(name: string, callback: () => Promise<R>) => Promise<R> }
  }).locks
  if (!locks?.request) {
    return fn()
  }
  return locks.request(TOKEN_REFRESH_LOCK, fn)
}

async function requestRefresh(options?: TokenRefreshOptions): Promise<RefreshTokenResponse> {
  const cached = existingFreshToken(options)
  if (cached) return cached

  const refreshToken = storedRefreshToken()
  if (!refreshToken) {
    throw new Error('No refresh token available')
  }

  const response = await axios.post<ApiResponse<RefreshTokenResponse>>(
    `${API_BASE_URL}/auth/refresh`,
    { refresh_token: refreshToken },
    { headers: { 'Content-Type': 'application/json' } }
  )
  const apiResponse = response.data
  if (apiResponse.code !== 0 || !apiResponse.data) {
    throw new Error(apiResponse.message || 'Token refresh failed')
  }
  persistTokenPair(apiResponse.data)
  return apiResponse.data
}

export async function refreshAccessToken(options?: TokenRefreshOptions): Promise<RefreshTokenResponse> {
  if (inFlight) {
    return inFlight
  }
  inFlight = withRefreshLock(async () => requestRefresh(options)).finally(() => {
    inFlight = null
  })
  return inFlight
}

export function clearAuthStorage(): void {
  localStorage.removeItem(AUTH_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem(AUTH_USER_KEY)
  localStorage.removeItem(TOKEN_EXPIRES_AT_KEY)
}
