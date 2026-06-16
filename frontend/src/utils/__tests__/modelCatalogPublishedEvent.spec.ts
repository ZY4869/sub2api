import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  MODEL_CATALOG_PUBLISHED_EVENT,
  subscribeModelCatalogPublishedEvents,
} from '../modelCatalogPublishedEvent'

function sseResponse(body: string): Response {
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      controller.enqueue(new TextEncoder().encode(body))
      controller.close()
    },
  })
  return new Response(stream, {
    status: 200,
    headers: { 'Content-Type': 'text/event-stream' },
  })
}

describe('modelCatalogPublishedEvent', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('dispatches a local refresh event from backend SSE payload', async () => {
    localStorage.setItem('auth_token', 'token-1')
    const fetchMock = vi.fn().mockResolvedValue(sseResponse([
      'event: model_catalog.published',
      'data: {"etag":"W/\\"etag-1\\"","published_at":"2026-06-16T10:00:00Z","model_count":2,"changed_count":1}',
      '',
      '',
    ].join('\n')))
    vi.stubGlobal('fetch', fetchMock)
    const listener = vi.fn()
    window.addEventListener(MODEL_CATALOG_PUBLISHED_EVENT, listener)

    const subscription = subscribeModelCatalogPublishedEvents()
    await vi.waitFor(() => expect(listener).toHaveBeenCalledTimes(1))
    subscription.close()
    window.removeEventListener(MODEL_CATALOG_PUBLISHED_EVENT, listener)

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/model-catalog/events'),
      expect.objectContaining({
        method: 'GET',
        headers: expect.objectContaining({
          Accept: 'text/event-stream',
          Authorization: 'Bearer token-1',
        }),
      }),
    )
    expect(listener.mock.calls[0][0].detail).toEqual({
      etag: 'W/"etag-1"',
      published_at: '2026-06-16T10:00:00Z',
      model_count: 2,
      changed_count: 1,
    })
  })

  it('aborts the backend stream when closed', () => {
    localStorage.setItem('auth_token', 'token-1')
    let signal: AbortSignal | undefined
    vi.stubGlobal('fetch', vi.fn((_url, init?: RequestInit) => {
      signal = init?.signal
      return new Promise<Response>(() => {})
    }))

    const subscription = subscribeModelCatalogPublishedEvents()
    subscription.close()

    expect(signal?.aborted).toBe(true)
  })
})
