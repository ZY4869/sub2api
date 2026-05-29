import type { Proxy, ProxyProtocol } from '@/types'

export const flagUrl = (code: string) =>
  `https://unpkg.com/flag-icons/flags/4x3/${code.toLowerCase()}.svg`

export function buildProxyUrl(row: Proxy): string {
  return `${row.protocol}://${buildAuthPart(row)}${row.host}:${row.port}`
}

export function getCopyFormats(row: Proxy) {
  const hasAuth = row.username || row.password
  const fullUrl = buildProxyUrl(row)
  const formats = [
    { label: fullUrl, value: fullUrl },
  ]
  if (hasAuth) {
    const withoutProtocol = fullUrl.replace(/^[^:]+:\/\//, '')
    formats.push({ label: withoutProtocol, value: withoutProtocol })
  }
  formats.push({ label: `${row.host}:${row.port}`, value: `${row.host}:${row.port}` })
  return formats
}

export interface ParsedProxyInput {
  protocol: ProxyProtocol
  host: string
  port: number
  username: string
  password: string
}

export function parseProxyUrl(line: string): ParsedProxyInput | null {
  const trimmed = line.trim()
  if (!trimmed) return null

  const regex = /^(https?|socks5h?):\/\/(?:([^:@]+):([^@]+)@)?([^:]+):(\d+)$/i
  const match = trimmed.match(regex)
  if (!match) return null

  const [, protocol, username, password, host, port] = match
  const portNum = parseInt(port, 10)
  if (portNum < 1 || portNum > 65535) return null

  return {
    protocol: protocol.toLowerCase() as ProxyProtocol,
    host: host.trim(),
    port: portNum,
    username: username?.trim() || '',
    password: password?.trim() || ''
  }
}

export function parseProxyBatchInput(input: string) {
  const lines = input.split('\n').filter((line) => line.trim())
  const seen = new Set<string>()
  const proxies: ParsedProxyInput[] = []
  let invalid = 0
  let duplicate = 0

  for (const line of lines) {
    const parsed = parseProxyUrl(line)
    if (!parsed) {
      invalid++
      continue
    }

    const key = `${parsed.host}:${parsed.port}:${parsed.username}:${parsed.password}`
    if (seen.has(key)) {
      duplicate++
      continue
    }
    seen.add(key)
    proxies.push(parsed)
  }

  return {
    total: lines.length,
    valid: proxies.length,
    invalid,
    duplicate,
    proxies
  }
}

export function formatProxyExportTimestamp(date = new Date()) {
  const pad2 = (value: number) => String(value).padStart(2, '0')
  return `${date.getFullYear()}${pad2(date.getMonth() + 1)}${pad2(date.getDate())}${pad2(date.getHours())}${pad2(date.getMinutes())}${pad2(date.getSeconds())}`
}

export function qualityStatusClass(status: string) {
  if (status === 'pass') return 'badge-success'
  if (status === 'warn') return 'badge-warning'
  if (status === 'challenge') return 'badge-danger'
  return 'badge-danger'
}

export function qualityOverallClass(status?: string) {
  if (status === 'healthy') return 'badge-success'
  if (status === 'warn') return 'badge-warning'
  if (status === 'challenge') return 'badge-danger'
  return 'badge-danger'
}

export type ProxyTranslate = (key: string) => string

export function qualityStatusLabel(status: string, t: ProxyTranslate) {
  if (status === 'pass') return t('admin.proxies.qualityStatusPass')
  if (status === 'warn') return t('admin.proxies.qualityStatusWarn')
  if (status === 'challenge') return t('admin.proxies.qualityStatusChallenge')
  return t('admin.proxies.qualityStatusFail')
}

export function qualityOverallLabel(status: string | undefined, t: ProxyTranslate) {
  if (status === 'healthy') return t('admin.proxies.qualityStatusHealthy')
  if (status === 'warn') return t('admin.proxies.qualityStatusWarn')
  if (status === 'challenge') return t('admin.proxies.qualityStatusChallenge')
  return t('admin.proxies.qualityStatusFail')
}

export function qualityTargetLabel(target: string, t: ProxyTranslate) {
  switch (target) {
    case 'base_connectivity':
      return t('admin.proxies.qualityTargetBase')
    case 'openai':
      return t('admin.accounts.platforms.openai')
    case 'anthropic':
      return t('admin.accounts.platforms.anthropic')
    case 'gemini':
      return t('admin.accounts.platforms.gemini')
    default:
      return target
  }
}

function buildAuthPart(row: Proxy): string {
  const user = row.username ? encodeURIComponent(row.username) : ''
  const pass = row.password ? encodeURIComponent(row.password) : ''
  if (user && pass) return `${user}:${pass}@`
  if (user) return `${user}@`
  if (pass) return `:${pass}@`
  return ''
}
