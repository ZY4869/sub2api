export type RequestPreviewEnvelopeState = 'captured' | 'empty' | 'raw_only'
export type RequestPreviewRenderState = 'uncollected' | 'empty' | 'raw_only' | 'ready'

export interface RequestPreviewEnvelope {
  state?: RequestPreviewEnvelopeState | string
  source?: string
  truncated?: boolean
  content_type?: string
  key_fields?: Record<string, string | number | boolean | null>
  payload?: unknown
}

export interface ParsedRequestPreviewContent {
  raw: string
  displayContent: string
  renderState: RequestPreviewRenderState
  hasContent: boolean
  isEnvelope: boolean
  source: string
  truncated: boolean
  contentType: string
  keyFields: Record<string, string | number | boolean | null>
  payload: unknown
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isEnvelope(value: unknown): value is RequestPreviewEnvelope {
  if (!isRecord(value)) {
    return false
  }
  return (
    'state' in value
    || 'payload' in value
    || 'source' in value
    || 'truncated' in value
    || 'content_type' in value
  )
}

function normalizeEnvelopeState(value: unknown): RequestPreviewEnvelopeState {
  switch (String(value || '').trim()) {
    case 'empty':
      return 'empty'
    case 'raw_only':
      return 'raw_only'
    default:
      return 'captured'
  }
}

function formatPayload(payload: unknown): string {
  if (typeof payload === 'string') {
    const trimmed = payload.trim()
    if (!trimmed) {
      return ''
    }
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2)
    } catch {
      return trimmed
    }
  }
  if (typeof payload === 'undefined' || payload === null) {
    return ''
  }
  try {
    return JSON.stringify(payload, null, 2)
  } catch {
    return String(payload)
  }
}

function normalizeKeyFields(
  value: unknown,
): Record<string, string | number | boolean | null> {
  if (!isRecord(value)) {
    return {}
  }
  const out: Record<string, string | number | boolean | null> = {}
  for (const [key, fieldValue] of Object.entries(value)) {
    const trimmedKey = key.trim()
    if (!trimmedKey) continue
    if (
      fieldValue === null
      || typeof fieldValue === 'string'
      || typeof fieldValue === 'number'
      || typeof fieldValue === 'boolean'
    ) {
      out[trimmedKey] = fieldValue
    }
  }
  return out
}

export function parseRequestPreviewContent(raw?: string | null): ParsedRequestPreviewContent {
  const normalizedRaw = String(raw || '').trim()
  if (!normalizedRaw) {
    return {
      raw: '',
      displayContent: '',
      renderState: 'uncollected',
      hasContent: false,
      isEnvelope: false,
      source: '',
      truncated: false,
      contentType: '',
      payload: null,
      keyFields: {},
    }
  }

  try {
    const parsed = JSON.parse(normalizedRaw) as unknown
    if (isEnvelope(parsed)) {
      const payloadText = formatPayload(parsed.payload)
      const state = normalizeEnvelopeState(parsed.state)
      return {
        raw: normalizedRaw,
        displayContent: payloadText,
        renderState: payloadText
          ? (state === 'raw_only' ? 'raw_only' : 'ready')
          : (state === 'empty' ? 'empty' : state === 'raw_only' ? 'raw_only' : 'uncollected'),
        hasContent: payloadText.length > 0,
        isEnvelope: true,
        source: String(parsed.source || '').trim(),
        truncated: Boolean(parsed.truncated),
        contentType: String(parsed.content_type || '').trim(),
        keyFields: normalizeKeyFields(parsed.key_fields),
        payload: typeof parsed.payload === 'undefined' ? null : parsed.payload,
      }
    }

    return {
      raw: normalizedRaw,
      displayContent: JSON.stringify(parsed, null, 2),
      renderState: 'ready',
      hasContent: true,
      isEnvelope: false,
      source: '',
      truncated: false,
      contentType: 'application/json',
      keyFields: {},
      payload: parsed,
    }
  } catch {
    return {
      raw: normalizedRaw,
      displayContent: normalizedRaw,
      renderState: 'ready',
      hasContent: true,
      isEnvelope: false,
      source: '',
      truncated: false,
      contentType: 'text/plain',
      keyFields: {},
      payload: normalizedRaw,
    }
  }
}
