import type { AccountPlatform } from '@/types'

export type AccountApiKeySettingsMode = 'create' | 'edit'

export function resolveAccountApiKeyDefaultBaseUrl(platform: AccountPlatform): string {
  if (platform === 'openai' || platform === 'sora') return 'https://api.openai.com'
  if (platform === 'gemini') return 'https://generativelanguage.googleapis.com'
  if (platform === 'antigravity') return 'https://cloudcode-pa.googleapis.com'
  return 'https://api.anthropic.com'
}

export function resolveAccountApiKeyPlaceholder(platform: AccountPlatform): string {
  if (platform === 'openai' || platform === 'sora') return 'sk-proj-...'
  if (platform === 'gemini') return 'AIza...'
  if (platform === 'antigravity') return 'sk-...'
  return 'sk-ant-...'
}

export function resolveAccountApiKeyBaseUrlHintKey(
  platform: AccountPlatform,
  mode: AccountApiKeySettingsMode
): string {
  if (mode === 'create' && platform === 'sora') return 'admin.accounts.soraUpstreamBaseUrlHint'
  if (platform === 'openai' || platform === 'sora') return 'admin.accounts.openai.baseUrlHint'
  if (platform === 'gemini') return 'admin.accounts.gemini.baseUrlHint'
  if (platform === 'antigravity') return 'admin.accounts.upstream.baseUrlHint'
  return 'admin.accounts.baseUrlHint'
}

export function resolveAccountApiKeyHintKey(
  platform: AccountPlatform,
  mode: AccountApiKeySettingsMode
): string {
  if (mode === 'edit') return 'admin.accounts.leaveEmptyToKeep'
  if (platform === 'openai' || platform === 'sora') return 'admin.accounts.openai.apiKeyHint'
  if (platform === 'gemini') return 'admin.accounts.gemini.apiKeyHint'
  if (platform === 'antigravity') return 'admin.accounts.upstream.apiKeyHint'
  return 'admin.accounts.apiKeyHint'
}
