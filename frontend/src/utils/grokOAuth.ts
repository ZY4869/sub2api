export interface GrokAuthUrlResult {
  auth_url: string
  session_id: string
  redirect_uri: string
  state: string
}

export interface GrokExchangeCodeResult {
  access_token: string
  refresh_token?: string
  id_token?: string
  token_type?: string
  expires_in?: number
  expires_at?: number
  scope?: string
  client_id?: string
  base_url?: string
  email?: string
  subject?: string
  name?: string
  email_verified?: boolean
}

export interface ParsedGrokOAuthCallback {
  code: string
  state?: string
}

export interface ParsedGrokOAuthPayload {
  credentials: Record<string, unknown>
  extra?: Record<string, unknown>
  suggestedName?: string
}

export function parseGrokOAuthCallback(rawValue: string): ParsedGrokOAuthCallback {
  const trimmed = rawValue.trim()
  if (!trimmed) {
    return { code: '' }
  }
  if (!trimmed.includes('code=')) {
    return { code: trimmed }
  }

  try {
    const url = new URL(trimmed)
    return {
      code: url.searchParams.get('code')?.trim() || '',
      state: url.searchParams.get('state')?.trim() || undefined
    }
  } catch {
    const codeMatch = trimmed.match(/[?&]code=([^&]+)/)
    const stateMatch = trimmed.match(/[?&]state=([^&]+)/)
    return {
      code: decodeURIComponent(codeMatch?.[1] || '').trim(),
      state: decodeURIComponent(stateMatch?.[1] || '').trim() || undefined
    }
  }
}

export function buildGrokOAuthPayload(tokenInfo: GrokExchangeCodeResult): ParsedGrokOAuthPayload {
  const credentials: Record<string, unknown> = {
    access_token: tokenInfo.access_token
  }
  assignIfPresent(credentials, 'refresh_token', tokenInfo.refresh_token)
  assignIfPresent(credentials, 'id_token', tokenInfo.id_token)
  assignIfPresent(credentials, 'token_type', tokenInfo.token_type)
  assignIfPresent(credentials, 'scope', tokenInfo.scope)
  assignIfPresent(credentials, 'client_id', tokenInfo.client_id)
  assignIfPresent(credentials, 'base_url', tokenInfo.base_url)
  assignIfPresent(credentials, 'email', tokenInfo.email)
  assignIfPresent(credentials, 'subject', tokenInfo.subject)
  assignIfPresent(credentials, 'name', tokenInfo.name)
  if (typeof tokenInfo.expires_in === 'number' && Number.isFinite(tokenInfo.expires_in)) {
    credentials.expires_in = tokenInfo.expires_in
  }
  if (typeof tokenInfo.expires_at === 'number' && Number.isFinite(tokenInfo.expires_at)) {
    credentials.expires_at = Math.floor(tokenInfo.expires_at)
  }

  const extra: Record<string, unknown> = {
    provider: 'xai',
    source: 'grok_browser_oauth'
  }
  assignIfPresent(extra, 'email', tokenInfo.email)
  assignIfPresent(extra, 'subject', tokenInfo.subject)
  assignIfPresent(extra, 'display_name', tokenInfo.name)

  return {
    credentials,
    extra,
    suggestedName: firstNonEmptyString(tokenInfo.email, tokenInfo.name, tokenInfo.subject)
  }
}

function assignIfPresent(target: Record<string, unknown>, key: string, value: unknown) {
  if (typeof value === 'string' && value.trim()) {
    target[key] = value.trim()
  }
}

function firstNonEmptyString(...values: Array<string | undefined>): string | undefined {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) {
      return value.trim()
    }
  }
  return undefined
}
