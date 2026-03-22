import type { ParsedKiroTokenImport } from '@/utils/kiroTokenImport'

export type KiroOAuthMethod = 'builder_id' | 'idc' | 'github' | 'google'

export interface KiroAuthUrlResult {
  auth_url: string
  session_id: string
  redirect_uri: string
  state: string
}

export interface KiroExchangeCodeResult {
  access_token: string
  refresh_token?: string
  expires_at?: string
  auth_method?: string
  provider?: string
  client_id?: string
  client_secret?: string
  client_id_hash?: string
  start_url?: string
  region?: string
  profile_arn?: string
  email?: string
  username?: string
  display_name?: string
}

export interface ParsedKiroOAuthCallback {
  code: string
  state?: string
}

export function parseKiroOAuthCallback(rawValue: string): ParsedKiroOAuthCallback {
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

export function buildKiroOAuthPayload(tokenInfo: KiroExchangeCodeResult): ParsedKiroTokenImport {
  const credentials: Record<string, unknown> = {
    access_token: tokenInfo.access_token
  }

  assignIfPresent(credentials, 'refresh_token', tokenInfo.refresh_token)
  assignIfPresent(credentials, 'expires_at', tokenInfo.expires_at)
  assignIfPresent(credentials, 'auth_method', tokenInfo.auth_method)
  assignIfPresent(credentials, 'client_id', tokenInfo.client_id)
  assignIfPresent(credentials, 'client_secret', tokenInfo.client_secret)
  assignIfPresent(credentials, 'client_id_hash', tokenInfo.client_id_hash)
  assignIfPresent(credentials, 'start_url', tokenInfo.start_url)
  assignIfPresent(credentials, 'region', tokenInfo.region)
  assignIfPresent(credentials, 'profile_arn', tokenInfo.profile_arn)

  const extra: Record<string, unknown> = {
    source: 'kiro_browser_oauth'
  }
  assignIfPresent(extra, 'provider', normalizeProvider(tokenInfo.provider, tokenInfo.auth_method))
  assignIfPresent(extra, 'email', tokenInfo.email)
  assignIfPresent(extra, 'username', tokenInfo.username)
  assignIfPresent(extra, 'display_name', tokenInfo.display_name)

  const suggestedName = firstNonEmptyString(tokenInfo.email, tokenInfo.username, tokenInfo.display_name)

  return {
    credentials,
    extra,
    suggestedName
  }
}

function normalizeProvider(provider?: string, authMethod?: string): string {
  const normalizedProvider = provider?.trim().toLowerCase()
  if (normalizedProvider) {
    return normalizedProvider
  }
  const normalizedMethod = authMethod?.trim().toLowerCase()
  if (normalizedMethod === 'builder_id' || normalizedMethod === 'idc') {
    return 'aws'
  }
  return normalizedMethod || 'kiro'
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
