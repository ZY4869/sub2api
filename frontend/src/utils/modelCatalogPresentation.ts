export const MODEL_CATALOG_DEFAULT_THRESHOLD = 200000
export const MODEL_CATALOG_PAGE_SIZE = 500
export const MODEL_CATALOG_PRICE_DISPLAY_MODE_STORAGE_KEY = 'admin_model_catalog_price_display_mode'

export type ModelCatalogPriceDisplayMode = 'usd' | 'dual'

const MODEL_CATALOG_PROVIDER_LABELS: Record<string, string> = {
  anthropic: 'Anthropic',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  sora: 'Sora'
}

export function resolveModelCatalogDisplayName(model: string, displayName?: string): string {
  return displayName || model
}

export function formatModelCatalogProvider(provider?: string): string {
  if (!provider) {
    return '-'
  }
  const normalized = provider.trim()
  if (!normalized) {
    return '-'
  }
  const lower = normalized.toLowerCase()
  return MODEL_CATALOG_PROVIDER_LABELS[lower] || `${normalized.charAt(0).toUpperCase()}${normalized.slice(1)}`
}

export function formatModelCatalogPlatforms(platforms?: string[]): string[] {
  if (!platforms?.length) {
    return []
  }
  return platforms.map((platform) => formatModelCatalogProvider(platform))
}

export function getModelCatalogPriceDisplayMode(): ModelCatalogPriceDisplayMode {
  if (typeof window === 'undefined') {
    return 'usd'
  }
  const storedMode = window.localStorage.getItem(MODEL_CATALOG_PRICE_DISPLAY_MODE_STORAGE_KEY)
  return storedMode === 'dual' ? 'dual' : 'usd'
}

export function setModelCatalogPriceDisplayMode(mode: ModelCatalogPriceDisplayMode) {
  if (typeof window === 'undefined') {
    return
  }
  window.localStorage.setItem(MODEL_CATALOG_PRICE_DISPLAY_MODE_STORAGE_KEY, mode)
}

export function buildModelCatalogTierDescription(threshold = MODEL_CATALOG_DEFAULT_THRESHOLD) {
  return {
    low: `<= ${threshold.toLocaleString()}`,
    high: `>= ${(threshold + 1).toLocaleString()}`
  }
}
