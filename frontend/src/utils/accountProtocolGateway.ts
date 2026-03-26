import type { Account, AccountPlatform, GatewayProtocol } from '@/types'

export interface ProtocolGatewayDescriptor {
  id: GatewayProtocol
  displayName: string
  requestFormats: string[]
  defaultBaseUrl: string
  apiKeyPlaceholder: string
  baseUrlHintKey: string
  apiKeyHintKey: string
  modelImportStrategy: GatewayProtocol
  testStrategy: GatewayProtocol
  targetGroupPlatform: GatewayProtocol
}

export const PROTOCOL_GATEWAY_PLATFORM = 'protocol_gateway' as const

export const PROTOCOL_GATEWAY_PROTOCOLS = ['openai', 'anthropic', 'gemini'] as const

export const PROTOCOL_GATEWAY_DESCRIPTORS: Record<GatewayProtocol, ProtocolGatewayDescriptor> = {
  openai: {
    id: 'openai',
    displayName: 'OpenAI',
    requestFormats: ['/v1/chat/completions', '/v1/responses'],
    defaultBaseUrl: 'https://api.openai.com',
    apiKeyPlaceholder: 'sk-proj-...',
    baseUrlHintKey: 'admin.accounts.protocolGateway.protocols.openai.baseUrlHint',
    apiKeyHintKey: 'admin.accounts.protocolGateway.protocols.openai.apiKeyHint',
    modelImportStrategy: 'openai',
    testStrategy: 'openai',
    targetGroupPlatform: 'openai'
  },
  anthropic: {
    id: 'anthropic',
    displayName: 'Anthropic',
    requestFormats: ['/v1/messages'],
    defaultBaseUrl: 'https://api.anthropic.com',
    apiKeyPlaceholder: 'sk-ant-...',
    baseUrlHintKey: 'admin.accounts.protocolGateway.protocols.anthropic.baseUrlHint',
    apiKeyHintKey: 'admin.accounts.protocolGateway.protocols.anthropic.apiKeyHint',
    modelImportStrategy: 'anthropic',
    testStrategy: 'anthropic',
    targetGroupPlatform: 'anthropic'
  },
  gemini: {
    id: 'gemini',
    displayName: 'Gemini',
    requestFormats: ['/v1beta/models/{model}:generateContent'],
    defaultBaseUrl: 'https://generativelanguage.googleapis.com',
    apiKeyPlaceholder: 'AIza...',
    baseUrlHintKey: 'admin.accounts.protocolGateway.protocols.gemini.baseUrlHint',
    apiKeyHintKey: 'admin.accounts.protocolGateway.protocols.gemini.apiKeyHint',
    modelImportStrategy: 'gemini',
    testStrategy: 'gemini',
    targetGroupPlatform: 'gemini'
  }
}

export function isProtocolGatewayPlatform(platform: string | null | undefined): platform is typeof PROTOCOL_GATEWAY_PLATFORM {
  return String(platform || '').trim().toLowerCase() === PROTOCOL_GATEWAY_PLATFORM
}

export function isGatewayProtocol(value: unknown): value is GatewayProtocol {
  return typeof value === 'string' && value in PROTOCOL_GATEWAY_DESCRIPTORS
}

export function normalizeGatewayProtocol(value: unknown): GatewayProtocol | '' {
  if (typeof value !== 'string') {
    return ''
  }
  const normalized = value.trim().toLowerCase()
  return isGatewayProtocol(normalized) ? normalized : ''
}

export function resolveGatewayProtocolDescriptor(
  gatewayProtocol: unknown
): ProtocolGatewayDescriptor | null {
  const normalized = normalizeGatewayProtocol(gatewayProtocol)
  return normalized ? PROTOCOL_GATEWAY_DESCRIPTORS[normalized] : null
}

export function resolveAccountGatewayProtocol(
  account?: Pick<Account, 'platform' | 'gateway_protocol' | 'extra'> | null
): GatewayProtocol | '' {
  if (!account || !isProtocolGatewayPlatform(account.platform)) {
    return ''
  }
  return (
    normalizeGatewayProtocol(account.gateway_protocol) ||
    normalizeGatewayProtocol(account.extra?.gateway_protocol)
  )
}

export function resolveEffectiveAccountPlatform(
  platform: AccountPlatform,
  gatewayProtocol?: unknown
): AccountPlatform {
  if (!isProtocolGatewayPlatform(platform)) {
    return platform
  }
  return normalizeGatewayProtocol(gatewayProtocol) || platform
}

export function resolveEffectiveAccountPlatformFromAccount(
  account?: Pick<Account, 'platform' | 'gateway_protocol' | 'extra'> | null
): AccountPlatform {
  if (!account) {
    return 'anthropic'
  }
  return resolveEffectiveAccountPlatform(
    account.platform,
    resolveAccountGatewayProtocol(account)
  )
}

export function resolveGatewayProtocolLabel(gatewayProtocol?: unknown): string {
  return resolveGatewayProtocolDescriptor(gatewayProtocol)?.displayName || ''
}
