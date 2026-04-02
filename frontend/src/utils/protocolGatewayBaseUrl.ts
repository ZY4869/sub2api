export type ProtocolGatewayBaseUrlStatus = 'empty' | 'valid' | 'invalid' | 'loopback'

export interface ProtocolGatewayBaseUrlCheckResult {
  status: ProtocolGatewayBaseUrlStatus
  input: string
  normalizedUrl: string
  hostname: string
  displayHost: string
}

const LOOPBACK_HOSTS = new Set(['localhost', '127.0.0.1', '0.0.0.0', '::1'])

function normalizeHostname(hostname: string): string {
  return hostname.replace(/^\[(.*)\]$/, '$1').toLowerCase()
}

export function checkProtocolGatewayBaseUrl(
  value: string | null | undefined
): ProtocolGatewayBaseUrlCheckResult {
  const input = String(value || '').trim()
  if (!input) {
    return {
      status: 'empty',
      input,
      normalizedUrl: '',
      hostname: '',
      displayHost: ''
    }
  }

  if (!/^https?:\/\//i.test(input)) {
    return {
      status: 'invalid',
      input,
      normalizedUrl: '',
      hostname: '',
      displayHost: ''
    }
  }

  try {
    const parsed = new URL(input)
    const protocol = parsed.protocol.toLowerCase()
    if (protocol !== 'http:' && protocol !== 'https:') {
      return {
        status: 'invalid',
        input,
        normalizedUrl: '',
        hostname: '',
        displayHost: ''
      }
    }

    const hostname = normalizeHostname(parsed.hostname)
    const displayHost = parsed.host || hostname
    return {
      status: LOOPBACK_HOSTS.has(hostname) ? 'loopback' : 'valid',
      input,
      normalizedUrl: parsed.toString(),
      hostname,
      displayHost
    }
  } catch {
    return {
      status: 'invalid',
      input,
      normalizedUrl: '',
      hostname: '',
      displayHost: ''
    }
  }
}
