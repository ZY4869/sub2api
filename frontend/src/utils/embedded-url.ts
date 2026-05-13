/**
 * Shared URL builder for iframe-embedded pages.
 * Used by PurchaseSubscriptionView and CustomPageView to build consistent URLs
 * with theme, lang, and ui_mode parameters only.
 */

const EMBEDDED_THEME_QUERY_KEY = 'theme'
const EMBEDDED_LANG_QUERY_KEY = 'lang'
const EMBEDDED_UI_MODE_QUERY_KEY = 'ui_mode'
const EMBEDDED_UI_MODE_VALUE = 'embedded'

const BLOCKED_EXTRA_PARAM_KEYS = new Set([
  'token',
  'access_token',
  'refresh_token',
  'api_key',
  'apikey',
  'authorization',
  'user_id',
  'src_host',
  'src_url',
])

interface BuildEmbeddedUrlOptions {
  extraParams?: Record<string, string | undefined>
}

export function buildEmbeddedUrl(
  baseUrl: string,
  theme: 'light' | 'dark' = 'light',
  lang?: string,
  options: BuildEmbeddedUrlOptions = {},
): string {
  if (!baseUrl) return baseUrl
  try {
    const url = new URL(baseUrl)
    url.searchParams.set(EMBEDDED_THEME_QUERY_KEY, theme)
    if (lang) {
      url.searchParams.set(EMBEDDED_LANG_QUERY_KEY, lang)
    }
    url.searchParams.set(EMBEDDED_UI_MODE_QUERY_KEY, EMBEDDED_UI_MODE_VALUE)
    for (const [rawKey, rawValue] of Object.entries(options.extraParams ?? {})) {
      const key = rawKey.trim()
      const value = typeof rawValue === 'string' ? rawValue.trim() : ''
      if (!key || !value) continue
      if (BLOCKED_EXTRA_PARAM_KEYS.has(key.toLowerCase())) continue
      url.searchParams.set(key, value)
    }
    return url.toString()
  } catch {
    return baseUrl
  }
}

export function detectTheme(): 'light' | 'dark' {
  if (typeof document === 'undefined') return 'light'
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}
