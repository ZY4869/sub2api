export interface ParsedKiroTokenImport {
  credentials: Record<string, unknown>
  extra?: Record<string, unknown>
  suggestedName?: string
}

type JsonRecord = Record<string, unknown>

const TOKEN_CONTAINER_KEYS = ['credentials', 'credential', 'token', 'tokens', 'auth', 'oauth'] as const
const USER_CONTAINER_KEYS = ['user', 'profile', 'account', 'identity'] as const

export function parseKiroTokenImport(rawInput: string): ParsedKiroTokenImport {
  const trimmed = rawInput.trim()
  if (!trimmed) {
    throw new Error('请输入 Kiro token 或 token JSON。')
  }

  if (!looksLikeJSONObject(trimmed)) {
    return {
      credentials: { access_token: trimmed },
      extra: {
        provider: 'kiro',
        source: 'kiro_import'
      }
    }
  }

  const parsed = parseJSONObject(trimmed)
  const sources = collectCandidateObjects(parsed)

  const accessToken = pickString(sources, ['access_token', 'accessToken'])
  if (!accessToken) {
    throw new Error('未找到 access_token，请粘贴包含 access_token 的 Kiro token JSON。')
  }

  const credentials: Record<string, unknown> = {
    access_token: accessToken
  }

  assignIfPresent(credentials, 'refresh_token', pickString(sources, ['refresh_token', 'refreshToken']))
  assignIfPresent(credentials, 'expires_at', normalizeScalar(pickValue(sources, ['expires_at', 'expiresAt', 'expiration', 'expires'])))
  assignIfPresent(credentials, 'auth_method', pickString(sources, ['auth_method', 'authMethod']))
  assignIfPresent(credentials, 'client_id', pickString(sources, ['client_id', 'clientId']))
  assignIfPresent(credentials, 'client_secret', pickString(sources, ['client_secret', 'clientSecret']))
  assignIfPresent(credentials, 'client_id_hash', pickString(sources, ['client_id_hash', 'clientIdHash']))
  assignIfPresent(credentials, 'start_url', pickString(sources, ['start_url', 'startUrl']))
  assignIfPresent(credentials, 'api_region', pickString(sources, ['api_region', 'apiRegion', 'region']))
  assignIfPresent(credentials, 'profile_arn', pickString(sources, ['profile_arn', 'profileArn']))

  const extra: Record<string, unknown> = {}
  assignIfPresent(extra, 'email', pickString(sources, ['email']))
  assignIfPresent(extra, 'username', pickString(sources, ['username', 'login', 'user_name']))
  assignIfPresent(extra, 'display_name', pickString(sources, ['display_name', 'displayName', 'name']))
  assignIfPresent(extra, 'provider', pickString(sources, ['provider']))
  assignIfPresent(extra, 'source', pickString(sources, ['source']))

  if (!extra.provider) {
    extra.provider = 'kiro'
  }
  if (!extra.source) {
    extra.source = 'kiro_import'
  }

  const suggestedName = firstNonEmptyString(
    extra.email,
    extra.username,
    extra.display_name
  )

  return {
    credentials,
    extra: Object.keys(extra).length > 0 ? extra : undefined,
    suggestedName
  }
}

function looksLikeJSONObject(value: string): boolean {
  return value.startsWith('{') || value.startsWith('[')
}

function parseJSONObject(value: string): JsonRecord {
  let parsed: unknown
  try {
    parsed = JSON.parse(value)
  } catch {
    throw new Error('Kiro token JSON 解析失败，请检查格式是否正确。')
  }

  if (Array.isArray(parsed)) {
    if (parsed.length !== 1 || !isRecord(parsed[0])) {
      throw new Error('暂不支持批量导入多个 Kiro token JSON，请一次粘贴一个。')
    }
    return parsed[0]
  }

  if (!isRecord(parsed)) {
    throw new Error('Kiro token JSON 必须是对象格式。')
  }

  return parsed
}

function collectCandidateObjects(root: JsonRecord): JsonRecord[] {
  const sources: JsonRecord[] = [root]

  for (const key of TOKEN_CONTAINER_KEYS) {
    const nested = root[key]
    if (isRecord(nested)) {
      sources.push(nested)
    }
  }

  for (const key of USER_CONTAINER_KEYS) {
    const nested = root[key]
    if (isRecord(nested)) {
      sources.push(nested)
    }
  }

  return sources
}

function pickValue(sources: JsonRecord[], keys: string[]): unknown {
  for (const source of sources) {
    for (const key of keys) {
      const value = source[key]
      if (value !== undefined && value !== null && value !== '') {
        return value
      }
    }
  }
  return undefined
}

function pickString(sources: JsonRecord[], keys: string[]): string | undefined {
  const value = pickValue(sources, keys)
  return normalizeScalar(value)
}

function normalizeScalar(value: unknown): string | undefined {
  if (typeof value === 'string') {
    const trimmed = value.trim()
    return trimmed || undefined
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(Math.trunc(value))
  }
  return undefined
}

function assignIfPresent(target: Record<string, unknown>, key: string, value: unknown) {
  if (value !== undefined && value !== null && value !== '') {
    target[key] = value
  }
}

function firstNonEmptyString(...values: unknown[]): string | undefined {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) {
      return value.trim()
    }
  }
  return undefined
}

function isRecord(value: unknown): value is JsonRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
