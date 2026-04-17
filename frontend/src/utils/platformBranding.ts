import type { AccountPlatform, GroupPlatform } from '@/types'
import { buildLobeIconSources, resolveLobeBadgeText } from '@/utils/lobeIconResolver'

type PlatformKey = AccountPlatform | GroupPlatform

interface PlatformBranding {
  badge: string
  iconKey: string
}

export const ACCOUNT_PLATFORM_ORDER: AccountPlatform[] = [
  'anthropic',
  'kiro',
  'openai',
  'copilot',
  'grok',
  'protocol_gateway',
  'gemini',
  'antigravity',
  'baidu_document_ai'
]

const PLATFORM_BRANDING: Record<AccountPlatform, PlatformBranding> = {
  anthropic: { badge: 'An', iconKey: 'anthropic' },
  kiro: { badge: 'Ki', iconKey: 'kiro' },
  openai: { badge: 'OA', iconKey: 'openai' },
  copilot: { badge: 'GH', iconKey: 'githubcopilot' },
  grok: { badge: 'Gr', iconKey: 'xai' },
  protocol_gateway: { badge: 'PG', iconKey: 'openrouter' },
  gemini: { badge: 'Go', iconKey: 'google' },
  antigravity: { badge: 'AG', iconKey: 'antigravity' },
  baidu_document_ai: { badge: 'BD', iconKey: 'baidu' }
}

export function getPlatformIconSources(platform?: PlatformKey | string | null): string[] {
  const iconKey = platform ? PLATFORM_BRANDING[String(platform) as AccountPlatform]?.iconKey : ''
  return iconKey ? buildLobeIconSources([iconKey]) : []
}

export function getPlatformBadgeText(platform?: PlatformKey | string | null): string {
  const normalized = String(platform || '').trim()
  return PLATFORM_BRANDING[normalized as AccountPlatform]?.badge || resolveLobeBadgeText(normalized)
}
