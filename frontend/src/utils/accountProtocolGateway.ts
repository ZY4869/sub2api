import type {
  Account,
  AccountPlatform,
  AccountType,
  GatewayAcceptedProtocol,
  GatewayClientProfile,
  GatewayClientRoute,
  GatewayProtocol
} from '@/types'

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
  targetGroupPlatform: GatewayAcceptedProtocol | ''
}

export const PROTOCOL_GATEWAY_PLATFORM = 'protocol_gateway' as const

export const PROTOCOL_GATEWAY_PROTOCOLS = ['openai', 'anthropic', 'gemini', 'mixed'] as const
export const PROTOCOL_GATEWAY_ACCEPTED_PROTOCOLS = ['openai', 'anthropic', 'gemini'] as const
export const PROTOCOL_GATEWAY_CLIENT_PROFILES = ['codex', 'gemini_cli'] as const
export const PROTOCOL_GATEWAY_GEMINI_BATCH_REQUEST_FORMATS = [
  '/upload/v1beta/files',
  '/v1beta/files',
  '/v1beta/models/{model}:batchGenerateContent',
  '/v1beta/batches/{batch}',
  '/v1/projects/{project}/locations/{location}/batchPredictionJobs'
] as const

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
  },
  mixed: {
    id: 'mixed',
    displayName: 'Mixed',
    requestFormats: [
      '/v1/chat/completions',
      '/v1/responses',
      '/v1/messages',
      '/v1beta/models/{model}:generateContent'
    ],
    defaultBaseUrl: '',
    apiKeyPlaceholder: 'gateway-key-...',
    baseUrlHintKey: 'admin.accounts.protocolGateway.protocols.mixed.baseUrlHint',
    apiKeyHintKey: 'admin.accounts.protocolGateway.protocols.mixed.apiKeyHint',
    modelImportStrategy: 'mixed',
    testStrategy: 'mixed',
    targetGroupPlatform: ''
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

export function isGatewayAcceptedProtocol(value: unknown): value is GatewayAcceptedProtocol {
  return typeof value === 'string' && (PROTOCOL_GATEWAY_ACCEPTED_PROTOCOLS as readonly string[]).includes(value)
}

export function normalizeGatewayAcceptedProtocol(value: unknown): GatewayAcceptedProtocol | '' {
  if (typeof value !== 'string') {
    return ''
  }
  const normalized = value.trim().toLowerCase()
  return isGatewayAcceptedProtocol(normalized) ? normalized : ''
}

export function normalizeGatewayAcceptedProtocols(
  gatewayProtocol: GatewayProtocol | '' | undefined,
  acceptedProtocols?: unknown
): GatewayAcceptedProtocol[] {
  if (gatewayProtocol && gatewayProtocol !== 'mixed') {
    return [gatewayProtocol]
  }
  const rawValues = Array.isArray(acceptedProtocols) ? acceptedProtocols : []
  const normalized = rawValues
    .map((value) => normalizeGatewayAcceptedProtocol(value))
    .filter((value): value is GatewayAcceptedProtocol => Boolean(value))
  const unique = [...new Set(normalized)]
  return unique.length > 0 ? unique : [...PROTOCOL_GATEWAY_ACCEPTED_PROTOCOLS]
}

export function isGatewayClientProfile(value: unknown): value is GatewayClientProfile {
  return typeof value === 'string' && (PROTOCOL_GATEWAY_CLIENT_PROFILES as readonly string[]).includes(value)
}

export function normalizeGatewayClientProfile(value: unknown): GatewayClientProfile | '' {
  if (typeof value !== 'string') {
    return ''
  }
  const normalized = value.trim().toLowerCase()
  return isGatewayClientProfile(normalized) ? normalized : ''
}

export function normalizeGatewayBatchEnabled(value: unknown): boolean {
  if (typeof value === 'boolean') {
    return value
  }
  if (typeof value === 'number') {
    return value !== 0
  }
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    return normalized === 'true' || normalized === '1' || normalized === 'yes' || normalized === 'on'
  }
  return false
}

export function supportedGatewayClientProfilesForProtocol(protocol: GatewayAcceptedProtocol): GatewayClientProfile[] {
  switch (protocol) {
    case 'openai':
      return ['codex']
    case 'gemini':
      return ['gemini_cli']
    default:
      return []
  }
}

export function normalizeGatewayClientRoutes(value: unknown): GatewayClientRoute[] {
  if (!Array.isArray(value)) {
    return []
  }
  return value
    .map((item) => {
      if (!item || typeof item !== 'object') {
        return null
      }
      const route = item as Record<string, unknown>
      const protocol = normalizeGatewayAcceptedProtocol(route.protocol)
      const matchType = route.match_type === 'exact' || route.match_type === 'prefix' ? route.match_type : ''
      const matchValue = typeof route.match_value === 'string' ? route.match_value.trim() : ''
      const clientProfile = normalizeGatewayClientProfile(route.client_profile)
      if (!protocol || !matchType || !matchValue || !clientProfile) {
        return null
      }
      return {
        protocol,
        match_type: matchType,
        match_value: matchValue,
        client_profile: clientProfile
      } satisfies GatewayClientRoute
    })
    .filter((value): value is GatewayClientRoute => Boolean(value))
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
  const normalizedProtocol = normalizeGatewayProtocol(gatewayProtocol)
  if (!normalizedProtocol || normalizedProtocol === 'mixed') {
    return platform
  }
  return normalizedProtocol
}

export function resolveEffectiveAccountPlatforms(
  platform: AccountPlatform,
  gatewayProtocol?: unknown,
  acceptedProtocols?: unknown
): AccountPlatform[] {
  if (!isProtocolGatewayPlatform(platform)) {
    return [platform]
  }

  const normalizedProtocol = normalizeGatewayProtocol(gatewayProtocol)
  if (normalizedProtocol && normalizedProtocol !== 'mixed') {
    return [normalizedProtocol]
  }

  const normalizedAccepted = normalizeGatewayAcceptedProtocols(
    normalizedProtocol || 'mixed',
    acceptedProtocols
  )

  return normalizedAccepted.length > 0 ? [...normalizedAccepted] : ['openai']
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

const CLAUDE_MIMIC_ENABLED_KEY = 'claude_code_mimic_enabled'
const CLAUDE_TLS_FINGERPRINT_KEY = 'enable_tls_fingerprint'
const CLAUDE_SESSION_MASKING_KEY = 'session_id_masking_enabled'

export function supportsProtocolGatewayClaudeClientMimic(options: {
  platform?: AccountPlatform | string | null
  type?: AccountType | string | null
  gatewayProtocol?: unknown
  acceptedProtocols?: unknown
}): boolean {
  if (!isProtocolGatewayPlatform(options.platform)) {
    return false
  }
  if (String(options.type || '').trim().toLowerCase() !== 'apikey') {
    return false
  }

  const acceptedProtocols = normalizeGatewayAcceptedProtocols(
    normalizeGatewayProtocol(options.gatewayProtocol) || 'mixed',
    options.acceptedProtocols
  )

  return acceptedProtocols.includes('anthropic')
}

export function supportsProtocolGatewayGeminiBatch(options: {
  platform?: AccountPlatform | string | null
  type?: AccountType | string | null
  gatewayProtocol?: unknown
  acceptedProtocols?: unknown
}): boolean {
  if (!isProtocolGatewayPlatform(options.platform)) {
    return false
  }
  if (String(options.type || '').trim().toLowerCase() !== 'apikey') {
    return false
  }

  const acceptedProtocols = normalizeGatewayAcceptedProtocols(
    normalizeGatewayProtocol(options.gatewayProtocol) || 'mixed',
    options.acceptedProtocols
  )

  return acceptedProtocols.includes('gemini')
}

export function supportsProtocolGatewayGeminiBatchAccount(
  account?: Pick<Account, 'platform' | 'type' | 'gateway_protocol' | 'extra'> | null
): boolean {
  if (!account) {
    return false
  }

  return supportsProtocolGatewayGeminiBatch({
    platform: account.platform,
    type: account.type,
    gatewayProtocol: account.gateway_protocol ?? account.extra?.gateway_protocol,
    acceptedProtocols: account.extra?.gateway_accepted_protocols
  })
}

export function resolveProtocolGatewayBatchRequestFormats(options: {
  gatewayProtocol?: unknown
  acceptedProtocols?: unknown
}): string[] {
  const acceptedProtocols = normalizeGatewayAcceptedProtocols(
    normalizeGatewayProtocol(options.gatewayProtocol) || 'mixed',
    options.acceptedProtocols
  )

  return acceptedProtocols.includes('gemini')
    ? [...PROTOCOL_GATEWAY_GEMINI_BATCH_REQUEST_FORMATS]
    : []
}

export function applyProtocolGatewayGeminiBatchExtra(
  base: Record<string, unknown> | undefined,
  options: {
    platform?: AccountPlatform | string | null
    type?: AccountType | string | null
    gatewayProtocol?: unknown
    acceptedProtocols?: unknown
    gatewayBatchEnabled?: boolean
  }
): Record<string, unknown> | undefined {
  const nextExtra: Record<string, unknown> = { ...(base || {}) }
  const supported = supportsProtocolGatewayGeminiBatch(options)

  if (!supported || !options.gatewayBatchEnabled) {
    delete nextExtra.gateway_batch_enabled
    return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
  }

  nextExtra.gateway_batch_enabled = true
  return nextExtra
}

export function supportsProtocolGatewayClaudeClientMimicAccount(
  account?: Pick<Account, 'platform' | 'type' | 'gateway_protocol' | 'extra'> | null
): boolean {
  if (!account) {
    return false
  }

  return supportsProtocolGatewayClaudeClientMimic({
    platform: account.platform,
    type: account.type,
    gatewayProtocol: account.gateway_protocol ?? account.extra?.gateway_protocol,
    acceptedProtocols: account.extra?.gateway_accepted_protocols
  })
}

export function applyProtocolGatewayClaudeClientMimicExtra(
  base: Record<string, unknown> | undefined,
  options: {
    platform?: AccountPlatform | string | null
    type?: AccountType | string | null
    gatewayProtocol?: unknown
    acceptedProtocols?: unknown
    claudeCodeMimicEnabled?: boolean
    enableTLSFingerprint?: boolean
    sessionIDMaskingEnabled?: boolean
  }
): Record<string, unknown> | undefined {
  const nextExtra: Record<string, unknown> = { ...(base || {}) }
  const supported = supportsProtocolGatewayClaudeClientMimic(options)

  if (!supported) {
    delete nextExtra[CLAUDE_MIMIC_ENABLED_KEY]
    delete nextExtra[CLAUDE_TLS_FINGERPRINT_KEY]
    delete nextExtra[CLAUDE_SESSION_MASKING_KEY]
    return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
  }

  if (!options.claudeCodeMimicEnabled) {
    delete nextExtra[CLAUDE_MIMIC_ENABLED_KEY]
    delete nextExtra[CLAUDE_TLS_FINGERPRINT_KEY]
    delete nextExtra[CLAUDE_SESSION_MASKING_KEY]
    return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
  }

  nextExtra[CLAUDE_MIMIC_ENABLED_KEY] = true

  if (options.enableTLSFingerprint) {
    nextExtra[CLAUDE_TLS_FINGERPRINT_KEY] = true
  } else {
    delete nextExtra[CLAUDE_TLS_FINGERPRINT_KEY]
  }

  if (options.sessionIDMaskingEnabled) {
    nextExtra[CLAUDE_SESSION_MASKING_KEY] = true
  } else {
    delete nextExtra[CLAUDE_SESSION_MASKING_KEY]
  }

  return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
}
