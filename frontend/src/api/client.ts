/**
 * Axios HTTP Client Configuration
 * Base client with interceptors for authentication, token refresh, and error handling
 */

import axios, { AxiosInstance, AxiosError, InternalAxiosRequestConfig, AxiosResponse } from 'axios'
import type { ApiResponse } from '@/types'
import { getLocale } from '@/i18n'
import { clearAuthStorage, refreshAccessToken } from './tokenRefresh'

// ==================== Axios Instance Configuration ====================

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

export const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

const MODEL_IMPORT_UPSTREAM_UNAUTHORIZED = 'MODEL_IMPORT_UPSTREAM_UNAUTHORIZED'

function normalizeResponseMetadata(apiData: Record<string, any>): Record<string, string> | undefined {
  return typeof apiData.metadata === 'object' && apiData.metadata !== null
    ? apiData.metadata as Record<string, string>
    : undefined
}

function buildStructuredApiError(
  status: number,
  apiData: Record<string, any>,
  fallbackMessage: string
) {
  return {
    status,
    code: apiData.code,
    error: apiData.error,
    message: apiData.message || apiData.detail || fallbackMessage,
    reason: typeof apiData.reason === 'string' ? apiData.reason : undefined,
    metadata: normalizeResponseMetadata(apiData)
  }
}

function isModelImportUpstreamUnauthorized(apiData: Record<string, any>): boolean {
  return apiData.reason === MODEL_IMPORT_UPSTREAM_UNAUTHORIZED ||
    apiData.error === MODEL_IMPORT_UPSTREAM_UNAUTHORIZED ||
    apiData.code === MODEL_IMPORT_UPSTREAM_UNAUTHORIZED
}

function extractBearerToken(headers: unknown): string | null {
  const value = (headers as Record<string, unknown> | undefined)?.Authorization ??
    (headers as Record<string, unknown> | undefined)?.authorization
  if (typeof value !== 'string') return null
  const trimmed = value.trim()
  if (!trimmed.toLowerCase().startsWith('bearer ')) return null
  return trimmed.slice(7).trim() || null
}

// ==================== Request Interceptor ====================

// Get user's timezone
const getUserTimezone = (): string => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return 'UTC'
  }
}

apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // Attach token from localStorage
    const token = localStorage.getItem('auth_token')
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }

    // Attach locale for backend translations
    if (config.headers) {
      config.headers['Accept-Language'] = getLocale()
    }

    // Attach timezone for all GET requests (backend may use it for default date ranges)
    if (config.method === 'get') {
      if (!config.params) {
        config.params = {}
      }
      config.params.timezone = getUserTimezone()
    }

    return config
  },
  (error: unknown) => {
    return Promise.reject(error)
  }
)

// ==================== Response Interceptor ====================

apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    // Unwrap standard API response format { code, message, data }
    const apiResponse = response.data as ApiResponse<unknown>
    if (apiResponse && typeof apiResponse === 'object' && 'code' in apiResponse) {
      if (apiResponse.code === 0) {
        // Success - return the data portion
        response.data = apiResponse.data
      } else {
        // API error
        const metadata = (typeof (apiResponse as any).metadata === 'object' && (apiResponse as any).metadata !== null
          ? (apiResponse as any).metadata
          : undefined) as Record<string, string> | undefined
        return Promise.reject({
          status: response.status,
          code: apiResponse.code,
          message: apiResponse.message || 'Unknown error',
          reason: typeof (apiResponse as any).reason === 'string' ? (apiResponse as any).reason : undefined,
          metadata
        })
      }
    }
    return response
  },
  async (error: AxiosError<ApiResponse<unknown>>) => {
    // Request cancellation: keep the original axios cancellation error so callers can ignore it.
    // Otherwise we'd misclassify it as a generic "network error".
    if (error.code === 'ERR_CANCELED' || axios.isCancel(error)) {
      return Promise.reject(error)
    }

    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }

    // Handle common errors
    if (error.response) {
      const { status, data } = error.response
      const url = String(error.config?.url || '')

      // Validate `data` shape to avoid HTML error pages breaking our error handling.
      const apiData = (typeof data === 'object' && data !== null ? data : {}) as Record<string, any>
      const structuredError = buildStructuredApiError(status, apiData, error.message)

      // Ops monitoring disabled: treat as feature-flagged 404, and proactively redirect away
      // from ops pages to avoid broken UI states.
      if (status === 404 && apiData.message === 'Ops monitoring is disabled') {
        try {
          localStorage.setItem('ops_monitoring_enabled_cached', 'false')
        } catch {
          // ignore localStorage failures
        }
        try {
          window.dispatchEvent(new CustomEvent('ops-monitoring-disabled'))
        } catch {
          // ignore event failures
        }

        if (window.location.pathname.startsWith('/admin/ops')) {
          window.location.href = '/admin/settings'
        }

        return Promise.reject({
          status,
          code: 'OPS_DISABLED',
          message: apiData.message || error.message,
          url
        })
      }

      // 401: Try to refresh the token if we have a refresh token
      // This handles TOKEN_EXPIRED, INVALID_TOKEN, TOKEN_REVOKED, etc.
      if (status === 401 && isModelImportUpstreamUnauthorized(apiData)) {
        return Promise.reject(structuredError)
      }

      if (status === 401 && !originalRequest._retry) {
        const refreshToken = localStorage.getItem('refresh_token')
        const isAuthEndpoint =
          url.includes('/auth/login') || url.includes('/auth/register') || url.includes('/auth/refresh')

        // If we have a refresh token and this is not an auth endpoint, try to refresh
        if (refreshToken && !isAuthEndpoint) {
          originalRequest._retry = true

          try {
            const refreshed = await refreshAccessToken({
              force: true,
              staleAccessToken: extractBearerToken(originalRequest.headers)
            })

            if (originalRequest.headers) {
              originalRequest.headers.Authorization = `Bearer ${refreshed.access_token}`
            }

            return apiClient(originalRequest)
          } catch {
            clearAuthStorage()
            sessionStorage.setItem('auth_expired', '1')

            if (!window.location.pathname.includes('/login')) {
              window.location.href = '/login'
            }

            return Promise.reject({
              status: 401,
              code: 'TOKEN_REFRESH_FAILED',
              message: 'Session expired. Please log in again.'
            })
          }
        }

        // No refresh token or is auth endpoint - clear auth and redirect
        const hasToken = !!localStorage.getItem('auth_token')
        const headers = error.config?.headers as Record<string, unknown> | undefined
        const authHeader = headers?.Authorization ?? headers?.authorization
        const sentAuth =
          typeof authHeader === 'string'
            ? authHeader.trim() !== ''
            : Array.isArray(authHeader)
              ? authHeader.length > 0
              : !!authHeader

        clearAuthStorage()
        if ((hasToken || sentAuth) && !isAuthEndpoint) {
          sessionStorage.setItem('auth_expired', '1')
        }
        // Only redirect if not already on login page
        if (!window.location.pathname.includes('/login')) {
          window.location.href = '/login'
        }
      }

      // Return structured error
      return Promise.reject(structuredError)
    }

    // Network error
    return Promise.reject({
      status: 0,
      message: 'Network error. Please check your connection.'
    })
  }
)

export default apiClient
