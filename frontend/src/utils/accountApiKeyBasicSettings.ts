import type { AccountPlatform, GatewayProtocol } from '@/types'
import {
  isProtocolGatewayPlatform,
  resolveGatewayProtocolDescriptor
} from '@/utils/accountProtocolGateway'

export type AccountApiKeySettingsMode = 'create' | 'edit'

function resolveProtocolGatewaySetting(platform: AccountPlatform, gatewayProtocol?: GatewayProtocol) {
  if (!isProtocolGatewayPlatform(platform)) {
    return null
  }
  return resolveGatewayProtocolDescriptor(gatewayProtocol)
}

export function resolveAccountApiKeyDefaultBaseUrl(
  platform: AccountPlatform,
  gatewayProtocol?: GatewayProtocol
): string {
  const descriptor = resolveProtocolGatewaySetting(platform, gatewayProtocol)
  if (descriptor) return descriptor.defaultBaseUrl
  if (platform === 'openai' || platform === 'sora') return 'https://api.openai.com'
  if (platform === 'copilot') return 'https://api.githubcopilot.com'
  if (platform === 'gemini') return 'https://generativelanguage.googleapis.com'
  if (platform === 'antigravity') return 'https://cloudcode-pa.googleapis.com'
  return 'https://api.anthropic.com'
}

export function resolveAccountApiKeyPlaceholder(
  platform: AccountPlatform,
  gatewayProtocol?: GatewayProtocol
): string {
  const descriptor = resolveProtocolGatewaySetting(platform, gatewayProtocol)
  if (descriptor) return descriptor.apiKeyPlaceholder
  if (platform === 'openai' || platform === 'sora') return 'sk-proj-...'
  if (platform === 'copilot') return 'ghu_...'
  if (platform === 'gemini') return 'AIza...'
  if (platform === 'antigravity') return 'sk-...'
  return 'sk-ant-...'
}

export function resolveAccountApiKeyBaseUrlHintKey(
  platform: AccountPlatform,
  mode: AccountApiKeySettingsMode,
  gatewayProtocol?: GatewayProtocol
): string {
  const descriptor = resolveProtocolGatewaySetting(platform, gatewayProtocol)
  if (descriptor) return descriptor.baseUrlHintKey
  if (mode === 'create' && platform === 'sora') return 'admin.accounts.soraUpstreamBaseUrlHint'
  if (platform === 'openai' || platform === 'sora' || platform === 'copilot') return 'admin.accounts.openai.baseUrlHint'
  if (platform === 'gemini') return 'admin.accounts.gemini.baseUrlHint'
  if (platform === 'antigravity') return 'admin.accounts.upstream.baseUrlHint'
  return 'admin.accounts.baseUrlHint'
}

export function resolveAccountApiKeyHintKey(
  platform: AccountPlatform,
  mode: AccountApiKeySettingsMode,
  gatewayProtocol?: GatewayProtocol
): string {
  if (mode === 'edit') return 'admin.accounts.leaveEmptyToKeep'
  const descriptor = resolveProtocolGatewaySetting(platform, gatewayProtocol)
  if (descriptor) return descriptor.apiKeyHintKey
  if (platform === 'openai' || platform === 'sora' || platform === 'copilot') return 'admin.accounts.openai.apiKeyHint'
  if (platform === 'gemini') return 'admin.accounts.gemini.apiKeyHint'
  if (platform === 'antigravity') return 'admin.accounts.upstream.apiKeyHint'
  return 'admin.accounts.apiKeyHint'
}

export function shouldSuggestProtocolGateway(
  platform: AccountPlatform,
  baseUrl: string | null | undefined
): boolean {
  if (isProtocolGatewayPlatform(platform)) {
    return false
  }
  if (platform !== 'openai' && platform !== 'anthropic' && platform !== 'gemini') {
    return false
  }
  const trimmed = String(baseUrl || '').trim()
  if (!trimmed) {
    return false
  }
  return trimmed !== resolveAccountApiKeyDefaultBaseUrl(platform)
}
