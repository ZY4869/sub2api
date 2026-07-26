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
  'openrouter',
  'protocol_gateway'
] as const
export const GROUP_ONLY_PLATFORM_ORDER = ['composite'] as const

export const ACCOUNT_PLATFORM_ORDER: AccountPlatform[] = [...PRIMARY_PLATFORM_ORDER]
export const GROUP_PLATFORM_ORDER: GroupPlatform[] = [...PRIMARY_PLATFORM_ORDER]
export const GROUP_MANAGEMENT_PLATFORM_ORDER: GroupPlatform[] = [
  ...PRIMARY_PLATFORM_ORDER,
  ...GROUP_ONLY_PLATFORM_ORDER
]
export const FILTER_PLATFORM_ORDER = [...PRIMARY_PLATFORM_ORDER]

export function getPlatformOrderIndex(platform: string): number {
  const index = PRIMARY_PLATFORM_ORDER.indexOf(platform as (typeof PRIMARY_PLATFORM_ORDER)[number])
  return index >= 0 ? index : Number.MAX_SAFE_INTEGER
}

export function isAccountPlatform(platform?: string | null): platform is AccountPlatform {
  return PRIMARY_PLATFORM_ORDER.includes(platform as (typeof PRIMARY_PLATFORM_ORDER)[number])
}

const PLATFORM_BRANDING: Record<GroupPlatform, PlatformBranding> = {
  anthropic: { badge: 'An', iconKey: 'anthropic', englishName: 'Anthropic' },
  antigravity: { badge: 'AG', iconKey: 'antigravity', englishName: 'Antigravity' },
  baidu_document_ai: { badge: 'BD', iconKey: 'baidu', englishName: 'Baidu Document AI' },
  composite: { badge: 'CP', iconKey: 'route', englishName: 'Composite' },
  deepseek: { badge: 'DS', iconKey: 'deepseek', englishName: 'DeepSeek' },
  gemini: { badge: 'Go', iconKey: 'google', englishName: 'Google' },
  grok: { badge: 'Gr', iconKey: 'xai', englishName: 'Grok' },
  kiro: { badge: 'Ki', iconKey: 'kiro', englishName: 'Kiro' },
  openai: { badge: 'OA', iconKey: 'openai', englishName: 'OpenAI' },
  openrouter: { badge: 'OR', iconKey: 'openrouter', englishName: 'OpenRouter' },
  protocol_gateway: { badge: 'PG', iconKey: 'newapi', englishName: 'Protocol Gateway' }
}

export function getPlatformIconSources(platform?: PlatformKey | string | null): string[] {
  const iconKey = platform ? PLATFORM_BRANDING[String(platform) as GroupPlatform]?.iconKey : ''
  return iconKey ? buildLobeIconSources([iconKey]) : []
}

export function getPlatformBadgeText(platform?: PlatformKey | string | null): string {
  const normalized = String(platform || '').trim()
  return PLATFORM_BRANDING[normalized as GroupPlatform]?.badge || resolveLobeBadgeText(normalized)
}

export function getPlatformEnglishName(platform?: PlatformKey | string | null): string {
  const normalized = String(platform || '').trim()
  return PLATFORM_BRANDING[normalized as GroupPlatform]?.englishName || normalized
}
