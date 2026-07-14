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

export interface GrokDeviceFlowStartResult {
  session_id: string
  user_code: string
  verification_uri: string
  verification_uri_complete?: string
  interval: number
  expires_at: number
}

export type GrokDevicePollStatus =
  | 'pending'
  | 'slow_down'
  | 'authorized'
  | 'denied'
  | 'expired'
  | string

export interface GrokDevicePollResult {
  status: GrokDevicePollStatus
  interval: number
  expires_at: number
  token_info?: GrokExchangeCodeResult
  error_message?: string
}

export type GrokAuthorizationInputKind =
  | 'unknown'
  | 'callback_url'
  | 'query_string'
  | 'bare_auth_code'
  | 'device_user_code'
  | 'device_url'

export interface ParsedGrokOAuthCallback {
  code: string
  state?: string
}

export interface ParsedGrokAuthorizationInput {
  kind: GrokAuthorizationInputKind
  code: string
  state?: string
  userCode?: string
  requiresState: boolean
}

export interface ParsedGrokOAuthPayload {
  credentials: Record<string, unknown>
  extra?: Record<string, unknown>
  suggestedName?: string
}

export function parseGrokOAuthCallback(rawValue: string): ParsedGrokOAuthCallback {
  const parsed = parseGrokAuthorizationInput(rawValue)
  return { code: parsed.code, state: parsed.state }
}

export function parseGrokAuthorizationInput(rawValue: string): ParsedGrokAuthorizationInput {
  const trimmed = rawValue.trim()
  if (!trimmed) {
    return emptyParsedInput('unknown')
  }

  const fromUrl = parseGrokInputFromUrl(trimmed)
  if (fromUrl) return fromUrl

  const queryCandidate = trimmed.startsWith('?') ? trimmed.slice(1) : trimmed
  if (queryCandidate.includes('=')) {
    const fromQuery = parseGrokInputFromQuery(queryCandidate)
    if (fromQuery) return fromQuery
  }

  const userCode = normalizeGrokDeviceUserCode(trimmed)
  if (userCode) {
    return { ...emptyParsedInput('device_user_code'), userCode }
  }

  return { ...emptyParsedInput('bare_auth_code'), code: trimmed }
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

const grokReauthorizationOverwriteKeys = [
  'access_token',
  'refresh_token',
  'id_token',
  'token_type',
  'scope',
  'client_id',
  'expires_in',
  'expires_at',
  'email',
  'subject',
  'name'
]

export function mergeGrokReauthorizationCredentials(
  existingCredentials: Record<string, unknown> | undefined | null,
  oauthCredentials: Record<string, unknown>
): Record<string, unknown> {
  const merged: Record<string, unknown> = { ...(existingCredentials || {}) }
  for (const key of grokReauthorizationOverwriteKeys) {
    if (Object.prototype.hasOwnProperty.call(oauthCredentials, key)) {
      merged[key] = oauthCredentials[key]
    }
  }
  if (!Object.prototype.hasOwnProperty.call(merged, 'base_url') && oauthCredentials.base_url) {
    merged.base_url = oauthCredentials.base_url
  }
  return merged
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

function emptyParsedInput(kind: GrokAuthorizationInputKind): ParsedGrokAuthorizationInput {
  return { kind, code: '', requiresState: false }
}

function parseGrokInputFromUrl(value: string): ParsedGrokAuthorizationInput | null {
  try {
    const url = new URL(value)
    const code = url.searchParams.get('code')?.trim() || ''
    if (code) {
      return {
        kind: 'callback_url',
        code,
        state: url.searchParams.get('state')?.trim() || undefined,
        requiresState: true
      }
    }
    const userCode = normalizeGrokDeviceUserCode(url.searchParams.get('user_code') || '')
    return userCode ? { ...emptyParsedInput('device_url'), userCode } : null
  } catch {
    return null
  }
}

function parseGrokInputFromQuery(value: string): ParsedGrokAuthorizationInput | null {
  const params = new URLSearchParams(value)
  const code = params.get('code')?.trim() || ''
  if (code) {
    return {
      kind: 'query_string',
      code,
      state: params.get('state')?.trim() || undefined,
      requiresState: true
    }
  }
  const userCode = normalizeGrokDeviceUserCode(params.get('user_code') || '')
  return userCode ? { ...emptyParsedInput('device_url'), userCode } : null
}

export function normalizeGrokDeviceUserCode(value: string): string {
  const trimmed = value.trim().toUpperCase()
  if (!trimmed || !/^[A-Z0-9-]+$/.test(trimmed)) return ''
  const parts = trimmed.split('-')
  if (parts.length === 1) {
    return parts[0].length === 8 ? trimmed : ''
  }
  return parts.every((part) => part.length === 4) ? trimmed : ''
}
