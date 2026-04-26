export const BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL =
  'https://paddleocr.aistudio-app.com/api/v2/ocr'

export function isBaiduDocumentAIPlatform(platform: unknown): boolean {
  const normalized = String(platform || '').trim().toLowerCase()
  return (
    normalized === 'baidu_document_ai' ||
    normalized === 'baidu' ||
    normalized === 'document-ai' ||
    normalized === 'document_ai'
  )
}

export function parseBaiduDocumentAIDirectApiUrlsInput(
  raw: string
): Record<string, string> {
  const trimmed = String(raw || '').trim()
  if (!trimmed) {
    return {}
  }

  let parsed: unknown
  try {
    parsed = JSON.parse(trimmed)
  } catch {
    throw new Error('invalid_json')
  }

  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('invalid_shape')
  }

  const normalized: Record<string, string> = {}
  for (const [modelId, value] of Object.entries(parsed as Record<string, unknown>)) {
    const nextModelId = String(modelId || '').trim()
    const nextURL = String(value || '').trim()
    if (!nextModelId || !nextURL) {
      continue
    }
    normalized[nextModelId] = nextURL.replace(/\/+$/, '')
  }
  return normalized
}

export function stringifyBaiduDocumentAIDirectApiUrls(
  value: unknown
): string {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return ''
  }
  const normalized: Record<string, string> = {}
  for (const [modelId, rawURL] of Object.entries(value as Record<string, unknown>)) {
    const nextModelId = String(modelId || '').trim()
    const nextURL = String(rawURL || '').trim()
    if (!nextModelId || !nextURL) {
      continue
    }
    normalized[nextModelId] = nextURL.replace(/\/+$/, '')
  }
  if (Object.keys(normalized).length === 0) {
    return ''
  }
  return JSON.stringify(normalized, null, 2)
}
