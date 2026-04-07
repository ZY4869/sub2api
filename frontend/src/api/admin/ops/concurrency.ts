import { apiClient } from '../../client'
import type { OpsAccountAvailabilityStatsResponse, OpsConcurrencyStatsResponse, OpsRealtimeTrafficSummaryResponse, OpsUserConcurrencyStatsResponse, OpsWSStatus, SubscribeQPSOptions } from './types'

export async function getConcurrencyStats(platform?: string, groupId?: number | null): Promise<OpsConcurrencyStatsResponse> {
  const params: Record<string, any> = {}
  if (platform) {
    params.platform = platform
  }
  if (typeof groupId === 'number' && groupId > 0) {
    params.group_id = groupId
  }

  const { data } = await apiClient.get<OpsConcurrencyStatsResponse>('/admin/ops/concurrency', { params })
  return data
}

export async function getUserConcurrencyStats(): Promise<OpsUserConcurrencyStatsResponse> {
  const { data } = await apiClient.get<OpsUserConcurrencyStatsResponse>('/admin/ops/user-concurrency')
  return data
}

export async function getAccountAvailabilityStats(platform?: string, groupId?: number | null): Promise<OpsAccountAvailabilityStatsResponse> {
  const params: Record<string, any> = {}
  if (platform) {
    params.platform = platform
  }
  if (typeof groupId === 'number' && groupId > 0) {
    params.group_id = groupId
  }
  const { data } = await apiClient.get<OpsAccountAvailabilityStatsResponse>('/admin/ops/account-availability', { params })
  return data
}

export async function getRealtimeTrafficSummary(
  window: string,
  platform?: string,
  groupId?: number | null,
  channelId?: number | null
): Promise<OpsRealtimeTrafficSummaryResponse> {
  const params: Record<string, any> = { window }
  if (platform) {
    params.platform = platform
  }
  if (typeof groupId === 'number' && groupId > 0) {
    params.group_id = groupId
  }
  if (typeof channelId === 'number' && channelId > 0) {
    params.channel_id = channelId
  }

  const { data } = await apiClient.get<OpsRealtimeTrafficSummaryResponse>('/admin/ops/realtime-traffic', { params })
  return data
}

export const OPS_WS_CLOSE_CODES = {
  REALTIME_DISABLED: 4001
} as const

const OPS_WS_BASE_PROTOCOL = 'sub2api-admin'

export function subscribeQPS(onMessage: (data: any) => void, options: SubscribeQPSOptions = {}): () => void {
  let ws: WebSocket | null = null
  let reconnectAttempts = 0
  const maxReconnectAttempts = Number.isFinite(options.maxReconnectAttempts as number)
    ? (options.maxReconnectAttempts as number)
    : Infinity
  const baseDelayMs = options.reconnectBaseDelayMs ?? 1000
  const maxDelayMs = options.reconnectMaxDelayMs ?? 30000
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let shouldReconnect = true
  let isConnecting = false
  let hasConnectedOnce = false
  let lastMessageAt = 0
  const staleTimeoutMs = options.staleTimeoutMs ?? 120_000
  const staleCheckIntervalMs = options.staleCheckIntervalMs ?? 30_000
  let staleTimer: ReturnType<typeof setInterval> | null = null

  const setStatus = (status: OpsWSStatus) => {
    options.onStatusChange?.(status)
  }

  const clearReconnectTimer = () => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  const clearStaleTimer = () => {
    if (staleTimer) {
      clearInterval(staleTimer)
      staleTimer = null
    }
  }

  const startStaleTimer = () => {
    clearStaleTimer()
    if (!staleTimeoutMs || staleTimeoutMs <= 0) return
    staleTimer = setInterval(() => {
      if (!shouldReconnect) return
      if (!ws || ws.readyState !== WebSocket.OPEN) return
      if (!lastMessageAt) return
      const ageMs = Date.now() - lastMessageAt
      if (ageMs > staleTimeoutMs) {
        // Treat as a half-open connection; closing triggers the normal reconnect path.
        ws.close()
      }
    }, staleCheckIntervalMs)
  }

  const scheduleReconnect = () => {
    if (!shouldReconnect) return
    if (hasConnectedOnce && reconnectAttempts >= maxReconnectAttempts) return

    // If we're offline, wait for the browser to come back online.
    if (typeof navigator !== 'undefined' && 'onLine' in navigator && !navigator.onLine) {
      setStatus('offline')
      return
    }

    const expDelay = baseDelayMs * Math.pow(2, reconnectAttempts)
    const delay = Math.min(expDelay, maxDelayMs)
    const jitter = Math.floor(Math.random() * 250)
    clearReconnectTimer()
    reconnectTimer = setTimeout(() => {
      reconnectAttempts++
      connect()
    }, delay + jitter)
    options.onReconnectScheduled?.({ attempt: reconnectAttempts + 1, delayMs: delay + jitter })
  }

  const handleOnline = () => {
    if (!shouldReconnect) return
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return
    connect()
  }

  const handleOffline = () => {
    setStatus('offline')
  }

  const connect = () => {
    if (!shouldReconnect) return
    if (isConnecting) return
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return
    if (hasConnectedOnce && reconnectAttempts >= maxReconnectAttempts) return

    isConnecting = true
    setStatus(hasConnectedOnce ? 'reconnecting' : 'connecting')
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsBaseUrl = options.wsBaseUrl || import.meta.env.VITE_WS_BASE_URL || window.location.host
    const wsURL = new URL(`${protocol}//${wsBaseUrl}/api/v1/admin/ops/ws/qps`)

    // Do NOT put admin JWT in the URL query string (it can leak via access logs, proxies, etc).
    // Browsers cannot set Authorization headers for WebSockets, so we pass the token via
    // Sec-WebSocket-Protocol (subprotocol list): ["sub2api-admin", "jwt.<token>"].
    const rawToken = String(options.token ?? localStorage.getItem('auth_token') ?? '').trim()
    const protocols: string[] = [OPS_WS_BASE_PROTOCOL]
    if (rawToken) protocols.push(`jwt.${rawToken}`)

    ws = new WebSocket(wsURL.toString(), protocols)

    ws.onopen = () => {
      reconnectAttempts = 0
      isConnecting = false
      hasConnectedOnce = true
      clearReconnectTimer()
      lastMessageAt = Date.now()
      startStaleTimer()
      setStatus('connected')
      options.onOpen?.()
    }

    ws.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data)
        lastMessageAt = Date.now()
        onMessage(data)
      } catch (err) {
        console.warn('[OpsWS] Failed to parse message:', err)
      }
    }

    ws.onerror = (error) => {
      console.error('[OpsWS] Connection error:', error)
      options.onError?.(error)
    }

    ws.onclose = (event) => {
      isConnecting = false
      options.onClose?.(event)
      clearStaleTimer()
      ws = null

      // If the server explicitly tells us to stop reconnecting, honor it.
      if (event && typeof event.code === 'number' && event.code === OPS_WS_CLOSE_CODES.REALTIME_DISABLED) {
        shouldReconnect = false
        clearReconnectTimer()
        setStatus('closed')
        options.onFatalClose?.(event)
        return
      }

      scheduleReconnect()
    }
  }

  window.addEventListener('online', handleOnline)
  window.addEventListener('offline', handleOffline)
  connect()

  return () => {
    shouldReconnect = false
    window.removeEventListener('online', handleOnline)
    window.removeEventListener('offline', handleOffline)
    clearReconnectTimer()
    clearStaleTimer()
    if (ws) ws.close()
    ws = null
    setStatus('closed')
  }
}
