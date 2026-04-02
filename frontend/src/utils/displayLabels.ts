type SupportedLocale = 'en' | 'zh'

type LocalizedLabel = {
  en: string
  zh: string
}

const TIMEZONE_LABELS: Record<string, LocalizedLabel> = {
  UTC: { en: 'UTC', zh: '协调世界时' },
  'Asia/Shanghai': { en: 'Shanghai', zh: '中国上海' },
  'Asia/Tokyo': { en: 'Tokyo', zh: '日本东京' },
  'Asia/Seoul': { en: 'Seoul', zh: '韩国首尔' },
  'Asia/Singapore': { en: 'Singapore', zh: '新加坡' },
  'Asia/Kolkata': { en: 'Kolkata', zh: '印度加尔各答' },
  'Asia/Dubai': { en: 'Dubai', zh: '阿联酋迪拜' },
  'Europe/London': { en: 'London', zh: '英国伦敦' },
  'Europe/Paris': { en: 'Paris', zh: '法国巴黎' },
  'Europe/Berlin': { en: 'Berlin', zh: '德国柏林' },
  'Europe/Moscow': { en: 'Moscow', zh: '俄罗斯莫斯科' },
  'America/New_York': { en: 'New York', zh: '美国纽约' },
  'America/Chicago': { en: 'Chicago', zh: '美国芝加哥' },
  'America/Denver': { en: 'Denver', zh: '美国丹佛' },
  'America/Los_Angeles': { en: 'Los Angeles', zh: '美国洛杉矶' },
  'America/Sao_Paulo': { en: 'Sao Paulo', zh: '巴西圣保罗' },
  'Australia/Sydney': { en: 'Sydney', zh: '澳大利亚悉尼' },
  'Pacific/Auckland': { en: 'Auckland', zh: '新西兰奥克兰' },
}

const AWS_REGION_LABELS: Record<string, LocalizedLabel> = {
  auto: { en: 'Auto', zh: '自动' },
  'us-east-1': { en: 'US East 1', zh: '美国东部 1' },
  'us-east-2': { en: 'US East 2', zh: '美国东部 2' },
  'us-west-1': { en: 'US West 1', zh: '美国西部 1' },
  'us-west-2': { en: 'US West 2', zh: '美国西部 2' },
  'ap-northeast-1': { en: 'Asia Pacific Northeast 1', zh: '亚太东北 1' },
  'ap-northeast-2': { en: 'Asia Pacific Northeast 2', zh: '亚太东北 2' },
  'ap-southeast-1': { en: 'Asia Pacific Southeast 1', zh: '亚太东南 1' },
  'ap-southeast-2': { en: 'Asia Pacific Southeast 2', zh: '亚太东南 2' },
  'ap-south-1': { en: 'Asia Pacific South 1', zh: '亚太南部 1' },
  'eu-west-1': { en: 'EU West 1', zh: '欧洲西部 1' },
  'eu-west-2': { en: 'EU West 2', zh: '欧洲西部 2' },
  'eu-central-1': { en: 'EU Central 1', zh: '欧洲中部 1' },
  'sa-east-1': { en: 'South America East 1', zh: '南美东部 1' },
}

const VERTEX_LOCATION_LABELS: Record<string, LocalizedLabel> = {
  global: { en: 'Global', zh: '全球' },
  'us-west1': { en: 'Oregon', zh: '美国西部 1' },
  'us-west4': { en: 'Las Vegas', zh: '美国西部 4' },
  'us-central1': { en: 'Iowa', zh: '美国中部 1' },
  'us-east1': { en: 'South Carolina', zh: '美国东部 1' },
  'us-east4': { en: 'Northern Virginia', zh: '美国东部 4' },
  'us-east5': { en: 'Columbus', zh: '美国东部 5' },
  'us-south1': { en: 'Dallas', zh: '美国南部 1' },
  'northamerica-northeast1': { en: 'Montreal', zh: '北美东北 1' },
  'southamerica-east1': { en: 'Sao Paulo', zh: '南美东部 1' },
  'europe-west1': { en: 'Belgium', zh: '欧洲西部 1' },
  'europe-west2': { en: 'London', zh: '欧洲西部 2' },
  'europe-west3': { en: 'Frankfurt', zh: '欧洲西部 3' },
  'europe-west4': { en: 'Netherlands', zh: '欧洲西部 4' },
  'europe-west6': { en: 'Zurich', zh: '欧洲西部 6' },
  'europe-west8': { en: 'Milan', zh: '欧洲西部 8' },
  'europe-west9': { en: 'Paris', zh: '欧洲西部 9' },
  'europe-north1': { en: 'Finland', zh: '欧洲北部 1' },
  'europe-central2': { en: 'Warsaw', zh: '欧洲中部 2' },
  'europe-southwest1': { en: 'Madrid', zh: '欧洲西南 1' },
  'asia-east1': { en: 'Taiwan', zh: '亚洲东部 1' },
  'asia-east2': { en: 'Hong Kong', zh: '亚洲东部 2' },
  'asia-northeast1': { en: 'Tokyo', zh: '亚洲东北 1' },
  'asia-northeast3': { en: 'Seoul', zh: '亚洲东北 3' },
  'asia-south1': { en: 'Mumbai', zh: '亚洲南部 1' },
  'asia-southeast1': { en: 'Singapore', zh: '亚洲东南 1' },
  'australia-southeast1': { en: 'Sydney', zh: '澳大利亚东南 1' },
}

export type DisplayOption = {
  value: string
  label: string
}

export const QUOTA_RESET_TIMEZONES = Object.keys(TIMEZONE_LABELS)
export const VERTEX_LOCATION_CODES = Object.keys(VERTEX_LOCATION_LABELS)

function resolveLocale(localeCode?: string): SupportedLocale {
  return String(localeCode || '').toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

function formatWithCode(value: string, label: string, localeCode?: string): string {
  if (!value || !label || label === value) return value
  return resolveLocale(localeCode) === 'zh' ? `${label}（${value}）` : `${label} (${value})`
}

function resolveMappedLabel(
  value: string,
  labels: Record<string, LocalizedLabel>,
  localeCode?: string,
): string | null {
  const normalized = String(value || '').trim()
  if (!normalized) return ''
  const resolved = labels[normalized]
  if (!resolved) return null
  return formatWithCode(normalized, resolved[resolveLocale(localeCode)], localeCode)
}

export function resolveUsageWindowLabel(value: string, t: (key: string) => string): string {
  switch (String(value || '').trim().toLowerCase()) {
    case '5h':
      return t('ui.usageWindow.fiveHour')
    case '1d':
      return t('ui.usageWindow.daily')
    case '7d':
      return t('ui.usageWindow.weekly')
    case 'total':
      return t('ui.usageWindow.total')
    case 'pro':
      return t('ui.usageWindow.pro')
    case 'flash':
      return t('ui.usageWindow.flash')
    default:
      return value
  }
}

export function buildTimezoneOptions(localeCode?: string): DisplayOption[] {
  return QUOTA_RESET_TIMEZONES.map((value) => ({
    value,
    label: formatTimezoneLabel(value, localeCode),
  }))
}

export function formatTimezoneLabel(value: string, localeCode?: string): string {
  return resolveMappedLabel(value, TIMEZONE_LABELS, localeCode) || String(value || '').trim()
}

export function formatAwsRegionLabel(value: string, localeCode?: string): string {
  return resolveMappedLabel(value, AWS_REGION_LABELS, localeCode) || String(value || '').trim()
}

export function formatVertexLocationLabel(value: string, localeCode?: string): string {
  return resolveMappedLabel(value, VERTEX_LOCATION_LABELS, localeCode) || String(value || '').trim()
}

export function formatGenericRegionLabel(value: string, localeCode?: string): string {
  const normalized = String(value || '').trim()
  if (!normalized) return ''
  return (
    resolveMappedLabel(normalized, AWS_REGION_LABELS, localeCode) ||
    resolveMappedLabel(normalized, VERTEX_LOCATION_LABELS, localeCode) ||
    resolveMappedLabel(normalized, TIMEZONE_LABELS, localeCode) ||
    normalized
  )
}

export function formatCountryLabel(countryCode?: string | null, fallbackName?: string | null, localeCode?: string): string {
  const locale = resolveLocale(localeCode) === 'zh' ? 'zh-Hans' : 'en'
  const normalizedCode = String(countryCode || '').trim().toUpperCase()
  const fallback = String(fallbackName || '').trim()

  if (!normalizedCode) return fallback

  try {
    const displayNames = new Intl.DisplayNames([locale], { type: 'region' })
    const localized = displayNames.of(normalizedCode)
    if (localized && localized !== normalizedCode) {
      return formatWithCode(normalizedCode, localized, localeCode)
    }
  } catch {
    // Ignore unsupported Intl environments and fall back to backend-provided text/code.
  }

  if (fallback && fallback.toLowerCase() !== normalizedCode.toLowerCase()) {
    return formatWithCode(normalizedCode, fallback, localeCode)
  }

  return normalizedCode
}

export function formatProxyLocationLabel(
  proxy: {
    country?: string | null
    country_code?: string | null
    city?: string | null
  },
  localeCode?: string,
): string {
  const country = formatCountryLabel(proxy.country_code, proxy.country, localeCode)
  const city = String(proxy.city || '').trim()
  return [country, city].filter(Boolean).join(' · ')
}
