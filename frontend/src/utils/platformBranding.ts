import type { AccountPlatform, GroupPlatform } from '@/types'
import { buildLobeIconSources, resolveLobeBadgeText } from '@/utils/lobeIconResolver'

type PlatformKey = AccountPlatform | GroupPlatform

interface PlatformBranding {
  badge: string
  iconKey: string
  englishName: string
}

export const PRIMARY_PLATFORM_ORDER = [
  'anthropic',
  'antigravity',
  'baidu_document_ai',
  'deepseek',
  'gemini',
  'grok',
  'kiro',
  'openai',
  'protocol_gateway'
] as const

export const ACCOUNT_PLATFORM_ORDER: AccountPlatform[] = [...PRIMARY_PLATFORM_ORDER]
export const GROUP_PLATFORM_ORDER: GroupPlatform[] = PRIMARY_PLATFORM_ORDER.filter(
  (platform): platform is GroupPlatform => platform !== 'protocol_gateway'
)
export const FILTER_PLATFORM_ORDER = [...PRIMARY_PLATFORM_ORDER]

export function getPlatformOrderIndex(platform: string): number {
  const index = PRIMARY_PLATFORM_ORDER.indexOf(platform as (typeof PRIMARY_PLATFORM_ORDER)[number])
  return index >= 0 ? index : Number.MAX_SAFE_INTEGER
}

const PLATFORM_BRANDING: Record<AccountPlatform, PlatformBranding> = {
  anthropic: { badge: 'An', iconKey: 'anthropic', englishName: 'Anthropic' },
  antigravity: { badge: 'AG', iconKey: 'antigravity', englishName: 'Antigravity' },
  baidu_document_ai: { badge: 'BD', iconKey: 'baidu', englishName: 'Baidu Document AI' },
  deepseek: { badge: 'DS', iconKey: 'deepseek', englishName: 'DeepSeek' },
  gemini: { badge: 'Go', iconKey: 'google', englishName: 'Google' },
  grok: { badge: 'Gr', iconKey: 'xai', englishName: 'Grok' },
  kiro: { badge: 'Ki', iconKey: 'kiro', englishName: 'Kiro' },
  openai: { badge: 'OA', iconKey: 'openai', englishName: 'OpenAI' },
  protocol_gateway: { badge: 'PG', iconKey: 'openrouter', englishName: 'Protocol Gateway' }
}

export function getPlatformIconSources(platform?: PlatformKey | string | null): string[] {
  const iconKey = platform ? PLATFORM_BRANDING[String(platform) as AccountPlatform]?.iconKey : ''
  return iconKey ? buildLobeIconSources([iconKey]) : []
}

export function getPlatformBadgeText(platform?: PlatformKey | string | null): string {
  const normalized = String(platform || '').trim()
  return PLATFORM_BRANDING[normalized as AccountPlatform]?.badge || resolveLobeBadgeText(normalized)
}

export function getPlatformEnglishName(platform?: PlatformKey | string | null): string {
  const normalized = String(platform || '').trim()
  return PLATFORM_BRANDING[normalized as AccountPlatform]?.englishName || normalized
}
