import { apiClient } from '@/api/client'
import { getLocale } from '@/i18n'

export const MODEL_CATALOG_PUBLISHED_EVENT = 'model-catalog:published'
export const MODEL_CATALOG_PUBLISHED_SSE_EVENT = 'model_catalog.published'

export interface ModelCatalogPublishedEventPayload {
  etag: string
  published_at?: string
  model_count: number
  changed_count: number
}

export interface ModelCatalogPublishedEventSubscription {
  close: () => void
}

export function emitModelCatalogPublished(payload: ModelCatalogPublishedEventPayload) {
  if (typeof window === 'undefined') {
    return
  }
  window.dispatchEvent(new CustomEvent(MODEL_CATALOG_PUBLISHED_EVENT, { detail: payload }))
}

export function subscribeModelCatalogPublishedEvents(): ModelCatalogPublishedEventSubscription {
  if (typeof window === 'undefined') {
    return { close: () => {} }
  }

  let closed = false
  let retryTimer: number | null = null
  let controller: AbortController | null = null

  const clearRetry = () => {
    if (retryTimer !== null) {
      window.clearTimeout(retryTimer)
      retryTimer = null
    }
  }

  const start = () => {
    if (closed) return
    const token = localStorage.getItem('auth_token')
    if (!token) return

    controller = new AbortController()
    void readModelCatalogPublishedEventStream(controller.signal, token).catch((error) => {
      if (closed || isAbortError(error)) return
      clearRetry()
      retryTimer = window.setTimeout(start, 5000)
    })
  }

  start()

  return {
    close: () => {
      closed = true
      clearRetry()
      controller?.abort()
      controller = null
    },
  }
}

async function readModelCatalogPublishedEventStream(signal: AbortSignal, token: string) {
  const response = await fetch(resolveModelCatalogEventsURL(), {
    method: 'GET',
    headers: buildModelCatalogEventHeaders(token),
    signal,
  })

  if (!response.ok || !response.body) {
    throw new Error(`model catalog event stream failed: ${response.status}`)
  }

  await readSSEStream(response.body, (eventName, payload) => {
    if (eventName !== MODEL_CATALOG_PUBLISHED_SSE_EVENT) {
      return
    }
    emitModelCatalogPublished(normalizePublishedEventPayload(payload))
  })
}

function resolveModelCatalogEventsURL(): string {
  const baseURL = String(apiClient.defaults.baseURL || '/api/v1').trim()
  const normalizedPath = '/model-catalog/events'
  if (/^https?:\/\//i.test(baseURL)) {
    return `${baseURL.replace(/\/+$/g, '')}${normalizedPath}`
  }
  if (typeof window !== 'undefined' && window.location?.origin) {
    return `${window.location.origin.replace(/\/+$/g, '')}/${baseURL.replace(/^\/+|\/+$/g, '')}${normalizedPath}`
  }
  return `/api/v1${normalizedPath}`
}

function buildModelCatalogEventHeaders(token: string): HeadersInit {
  return {
    Accept: 'text/event-stream',
    'Accept-Language': getLocale(),
    Authorization: `Bearer ${token}`,
  }
}

async function readSSEStream(
  stream: ReadableStream<Uint8Array>,
  onEvent: (eventName: string, payload: unknown) => void,
) {
  const reader = stream.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const chunks = buffer.split('\n\n')
    buffer = chunks.pop() || ''
    for (const chunk of chunks) {
      emitSSEChunk(chunk, onEvent)
    }
  }

  if (buffer.trim()) {
    emitSSEChunk(buffer, onEvent)
  }
}

function emitSSEChunk(
  chunk: string,
  onEvent: (eventName: string, payload: unknown) => void,
) {
  let eventName = ''
  const dataLines: string[] = []

  for (const line of chunk.split(/\r?\n/)) {
    if (line.startsWith(':')) continue
    if (line.startsWith('event:')) {
      eventName = line.slice('event:'.length).trim()
      continue
    }
    if (line.startsWith('data:')) {
      dataLines.push(line.slice('data:'.length).trim())
    }
  }

  if (!eventName || dataLines.length === 0) return

  const raw = dataLines.join('\n')
  try {
    onEvent(eventName, JSON.parse(raw))
  } catch {
    onEvent(eventName, raw)
  }
}

function normalizePublishedEventPayload(payload: unknown): ModelCatalogPublishedEventPayload {
  const value = typeof payload === 'object' && payload !== null
    ? payload as Record<string, unknown>
    : {}
  return {
    etag: String(value.etag || ''),
    published_at: typeof value.published_at === 'string' ? value.published_at : undefined,
    model_count: Number(value.model_count || 0),
    changed_count: Number(value.changed_count || 0),
  }
}

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === 'AbortError'
}
