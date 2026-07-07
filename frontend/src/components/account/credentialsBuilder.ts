export function applyInterceptWarmup(
  credentials: Record<string, unknown>,
  enabled: boolean,
  mode: 'create' | 'edit'
): void {
  if (enabled) {
    credentials.intercept_warmup_requests = true
  } else if (mode === 'edit') {
    delete credentials.intercept_warmup_requests
  }
}

const forbiddenRequestHeaderNames = new Set([
  'authorization',
  'x-api-key',
  'api-key',
  'x-goog-api-key',
  'cookie',
  'host',
  'content-length',
  'connection',
  'transfer-encoding',
  'proxy-authorization',
  'proxy-authenticate',
  'te',
  'trailer',
  'upgrade',
  'keep-alive'
])

const requestHeaderNamePattern = /^[!#$%&'*+\-.^_`|~0-9a-z]+$/i

export type AccountRequestHeadersResult =
  | { ok: true; headers: Record<string, string> }
  | { ok: false; errorKey: string }

export function buildAccountRequestHeaders(input: string): AccountRequestHeadersResult {
  const raw = input.trim()
  if (!raw) {
    return { ok: true, headers: {} }
  }

  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    return { ok: false, errorKey: 'admin.accounts.requestHeadersInvalidJson' }
  }
  if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
    return { ok: false, errorKey: 'admin.accounts.requestHeadersInvalidShape' }
  }

  const headers: Record<string, string> = {}
  for (const [name, value] of Object.entries(parsed as Record<string, unknown>)) {
    const normalizedName = name.trim().toLowerCase()
    if (!normalizedName) {
      continue
    }
    if (!requestHeaderNamePattern.test(normalizedName)) {
      return { ok: false, errorKey: 'admin.accounts.requestHeadersInvalidName' }
    }
    if (forbiddenRequestHeaderNames.has(normalizedName)) {
      return { ok: false, errorKey: 'admin.accounts.requestHeadersForbiddenName' }
    }
    const normalizedValue = String(value ?? '').trim()
    if (!normalizedValue) {
      continue
    }
    headers[normalizedName] = normalizedValue
  }

  return { ok: true, headers }
}

export function applyAccountRequestHeaders(
  credentials: Record<string, unknown>,
  input: string
): string | null {
  const result = buildAccountRequestHeaders(input)
  if (!result.ok) {
    return result.errorKey
  }
  if (Object.keys(result.headers).length > 0) {
    credentials.request_headers = result.headers
  } else {
    delete credentials.request_headers
  }
  delete credentials.request_header_overrides
  delete credentials.header_overrides
  return null
}

export function formatAccountRequestHeaders(source: unknown): string {
  const sourceRecord = (source && typeof source === 'object' && !Array.isArray(source))
    ? source as Record<string, unknown>
    : {}
  const rawHeaders = sourceRecord.request_headers ||
    sourceRecord.request_header_overrides ||
    sourceRecord.header_overrides
  if (!rawHeaders || typeof rawHeaders !== 'object' || Array.isArray(rawHeaders)) {
    return ''
  }
  const entries = Object.entries(rawHeaders as Record<string, unknown>)
    .map(([name, value]) => [name.trim().toLowerCase(), String(value ?? '').trim()] as const)
    .filter(([name, value]) => name && value && !forbiddenRequestHeaderNames.has(name))
    .sort(([a], [b]) => a.localeCompare(b))
  if (entries.length === 0) {
    return ''
  }
  return JSON.stringify(Object.fromEntries(entries), null, 2)
}
