import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import UsageRequestPreviewModal from '../UsageRequestPreviewModal.vue'

const translations: Record<string, string> = {
  'common.close': 'Close',
  'common.loading': 'Loading',
  'common.refresh': 'Refresh',
  'usage.requestPreview.title': 'Request Details',
  'usage.requestPreview.description': 'Review the captured preview for this usage request.',
  'usage.requestPreview.metaRequestId': 'Request ID',
  'usage.requestPreview.metaCapturedAt': 'Captured At',
  'usage.requestPreview.metaSource': 'Source',
  'usage.requestPreview.previewReady': 'Preview is ready',
  'usage.requestPreview.empty': 'No content available',
  'usage.requestPreview.capturedEmptyStatus': 'Captured, but empty',
  'usage.requestPreview.capturedEmptyDescription': 'This section was captured, but the payload was empty.',
  'usage.requestPreview.rawOnlyStatus': 'Only raw fallback content is available',
  'usage.requestPreview.rawOnlyDescription': 'Only a raw fallback marker was recorded for this section.',
  'usage.requestPreview.rawOnlyNotice': 'This section is showing raw fallback content because structured capture was unavailable.',
  'usage.requestPreview.truncatedNotice': 'This preview was truncated to the capture preview limit.',
  'usage.requestPreview.rawOnlyBadge': 'Raw fallback',
  'usage.requestPreview.truncatedBadge': 'Truncated',
  'usage.requestPreview.failedToLoad': 'Failed to load request details',
  'usage.requestPreview.unavailableTitle': 'No request details captured',
  'usage.requestPreview.unavailableDescription': 'Unavailable',
  'usage.requestPreview.sections.inbound': 'Inbound Request',
  'usage.requestPreview.sections.normalized': 'Normalized Request',
  'usage.requestPreview.sections.upstreamRequest': 'Upstream Request',
  'usage.requestPreview.sections.upstreamResponse': 'Upstream Response',
  'usage.requestPreview.sections.gatewayResponse': 'Gateway Response',
  'usage.requestPreview.sections.tools': 'Tools / Thinking',
  'usage.requestPreview.emptyStates.inbound': 'No inbound request preview was captured for this request.',
  'usage.requestPreview.emptyStates.normalized': 'No normalized request content is available for this request.',
  'usage.requestPreview.emptyStates.upstreamRequest': 'No upstream request content is available for this request.',
  'usage.requestPreview.emptyStates.upstreamResponse': 'No upstream response content is available for this request.',
  'usage.requestPreview.emptyStates.gatewayResponse': 'No gateway response content is available for this request.',
  'usage.requestPreview.emptyStates.tools': 'No tool or thinking trace was captured for this request.',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        const template = translations[key] ?? key
        return Object.entries(params || {}).reduce(
          (result, [paramKey, value]) => result.replace(`{${paramKey}}`, String(value)),
          template
        )
      }
    })
  }
})

describe('UsageRequestPreviewModal', () => {
  it('uses an injected preview loader and renders envelope metadata', async () => {
    const previewLoader = vi.fn().mockResolvedValue({
      available: true,
      request_id: 'req-123',
      captured_at: '2026-04-19T00:00:00Z',
      inbound_request_json: JSON.stringify({
        state: 'raw_only',
        source: 'inbound_request_fallback',
        truncated: true,
        payload: { prompt: 'hello' }
      }),
      normalized_request_json: '',
      upstream_request_json: '',
      upstream_response_json: '',
      gateway_response_json: '',
      tool_trace_json: '',
    })

    const wrapper = mount(UsageRequestPreviewModal, {
      props: {
        show: true,
        usageLog: { id: 7, request_id: 'req-123' },
        previewLoader,
      },
      attachTo: document.body,
    })

    await flushPromises()
    const pageText = document.body.textContent || ''

    expect(previewLoader).toHaveBeenCalledWith(7, expect.objectContaining({ signal: expect.any(AbortSignal) }))
    expect(pageText).toContain('Source: inbound_request_fallback')
    expect(pageText).toContain('Raw fallback')
    expect(pageText).toContain('Truncated')
    expect(pageText).toContain('"prompt": "hello"')

    wrapper.unmount()
  })
})
