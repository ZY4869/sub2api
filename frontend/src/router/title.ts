import { i18n } from '@/i18n'

function resolveLocaleMessageValue(message: unknown, path: string): string | null {
  if (!message || typeof message !== 'object' || !path.trim()) {
    return null
  }

  const value = path.split('.').reduce<unknown>((current, segment) => {
    if (!current || typeof current !== 'object') {
      return null
    }
    return (current as Record<string, unknown>)[segment]
  }, message)

  return typeof value === 'string' && value.trim() ? value.trim() : null
}

function resolveLocalizedRouteTitle(titleKey: string): string | null {
  const currentLocale = String(i18n.global.locale.value || '')
  const fallbackLocale = String(i18n.global.fallbackLocale)

  return (
    resolveLocaleMessageValue(i18n.global.getLocaleMessage(currentLocale), titleKey) ||
    resolveLocaleMessageValue(i18n.global.getLocaleMessage(fallbackLocale), titleKey)
  )
}

/**
 * 统一生成页面标题，避免多处写入 document.title 产生覆盖冲突。
 * 优先使用 titleKey 通过 i18n 翻译，fallback 到静态 routeTitle。
 */
export function resolveDocumentTitle(routeTitle: unknown, siteName?: string, titleKey?: string): string {
  const normalizedSiteName = typeof siteName === 'string' && siteName.trim() ? siteName.trim() : 'Sub2API'

  if (typeof titleKey === 'string' && titleKey.trim()) {
    const translated = resolveLocalizedRouteTitle(titleKey)
    if (translated) {
      return `${translated} - ${normalizedSiteName}`
    }
  }

  if (typeof routeTitle === 'string' && routeTitle.trim()) {
    return `${routeTitle.trim()} - ${normalizedSiteName}`
  }

  return normalizedSiteName
}
