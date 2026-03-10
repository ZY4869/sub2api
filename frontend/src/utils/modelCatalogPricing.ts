import type { ModelCatalogExchangeRate, ModelCatalogPricing } from '@/api/admin/models'
import { TOKENS_PER_MILLION } from './usagePricing'

export type ModelCatalogPricingKey = keyof ModelCatalogPricing
export type ModelCatalogPricingUnit = 'token' | 'image' | 'threshold'
export type ModelCatalogPricingGroup = 'inputTier' | 'outputTier' | 'cache' | 'image'

export interface ModelCatalogPricingField {
  key: ModelCatalogPricingKey
  labelKey: string
  unit: ModelCatalogPricingUnit
  group: ModelCatalogPricingGroup
}

export const MODEL_CATALOG_PRICING_GROUPS: Array<{ key: ModelCatalogPricingGroup; labelKey: string }> = [
  { key: 'inputTier', labelKey: 'admin.models.groups.inputTier' },
  { key: 'outputTier', labelKey: 'admin.models.groups.outputTier' },
  { key: 'cache', labelKey: 'admin.models.groups.cache' },
  { key: 'image', labelKey: 'admin.models.groups.image' }
]

export const MODEL_CATALOG_PRICING_FIELDS: ModelCatalogPricingField[] = [
  { key: 'input_token_threshold', labelKey: 'admin.models.fields.inputThreshold', unit: 'threshold', group: 'inputTier' },
  { key: 'input_cost_per_token', labelKey: 'admin.models.fields.inputCost', unit: 'token', group: 'inputTier' },
  { key: 'input_cost_per_token_above_threshold', labelKey: 'admin.models.fields.inputCostAboveThreshold', unit: 'token', group: 'inputTier' },
  { key: 'input_cost_per_token_priority', labelKey: 'admin.models.fields.inputPriorityCost', unit: 'token', group: 'inputTier' },
  { key: 'input_cost_per_token_priority_above_threshold', labelKey: 'admin.models.fields.inputPriorityCostAboveThreshold', unit: 'token', group: 'inputTier' },
  { key: 'output_token_threshold', labelKey: 'admin.models.fields.outputThreshold', unit: 'threshold', group: 'outputTier' },
  { key: 'output_cost_per_token', labelKey: 'admin.models.fields.outputCost', unit: 'token', group: 'outputTier' },
  { key: 'output_cost_per_token_above_threshold', labelKey: 'admin.models.fields.outputCostAboveThreshold', unit: 'token', group: 'outputTier' },
  { key: 'output_cost_per_token_priority', labelKey: 'admin.models.fields.outputPriorityCost', unit: 'token', group: 'outputTier' },
  { key: 'output_cost_per_token_priority_above_threshold', labelKey: 'admin.models.fields.outputPriorityCostAboveThreshold', unit: 'token', group: 'outputTier' },
  { key: 'cache_creation_input_token_cost', labelKey: 'admin.models.fields.cacheCreationCost', unit: 'token', group: 'cache' },
  { key: 'cache_creation_input_token_cost_above_1hr', labelKey: 'admin.models.fields.cacheCreationCostAbove1h', unit: 'token', group: 'cache' },
  { key: 'cache_read_input_token_cost', labelKey: 'admin.models.fields.cacheReadCost', unit: 'token', group: 'cache' },
  { key: 'cache_read_input_token_cost_priority', labelKey: 'admin.models.fields.cacheReadPriorityCost', unit: 'token', group: 'cache' },
  { key: 'output_cost_per_image', labelKey: 'admin.models.fields.imageCost', unit: 'image', group: 'image' }
]

export function tokenPriceToMillion(value?: number): number | null {
  return typeof value === 'number' ? value * TOKENS_PER_MILLION : null
}

export function millionPriceToToken(value?: number | null): number | undefined {
  return typeof value === 'number' ? value / TOKENS_PER_MILLION : undefined
}

function normalizeDisplayValue(value: number, unit: ModelCatalogPricingUnit): number {
  return unit === 'token' ? value * TOKENS_PER_MILLION : value
}

const CNY_SYMBOL = '\u00A5'
const APPROX_PREFIX = '\u2248 '

function formatCurrency(amount: number, symbol: string, precision = amount >= 1 ? 4 : 6): string {
  return `${symbol}${amount.toFixed(precision)}`
}

export function formatModelCatalogPrice(value?: number, unit: ModelCatalogPricingUnit = 'token'): string {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '-'
  }
  if (unit === 'threshold') {
    return `${Math.trunc(value)} tokens`
  }
  return formatCurrency(normalizeDisplayValue(value, unit), '$')
}

export function formatModelCatalogCNYReference(
  value?: number,
  unit: ModelCatalogPricingUnit = 'token',
  exchangeRate?: Pick<ModelCatalogExchangeRate, 'rate'> | null
): string | null {
  if (typeof value !== 'number' || Number.isNaN(value) || !exchangeRate?.rate || unit === 'threshold') {
    return null
  }
  const cnyValue = normalizeDisplayValue(value, unit) * exchangeRate.rate
  return `${APPROX_PREFIX}${formatCurrency(cnyValue, CNY_SYMBOL, cnyValue >= 1 ? 2 : 4)}`
}

export function formatModelCatalogPricePair(
  value?: number,
  unit: ModelCatalogPricingUnit = 'token',
  exchangeRate?: Pick<ModelCatalogExchangeRate, 'rate'> | null
) {
  return {
    usd: formatModelCatalogPrice(value, unit),
    cny: formatModelCatalogCNYReference(value, unit, exchangeRate)
  }
}

export function pricingInputValue(value: number | undefined, unit: ModelCatalogPricingUnit): string {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return ''
  }
  if (unit === 'threshold') {
    return String(Math.trunc(value))
  }
  const normalized = unit === 'token' ? tokenPriceToMillion(value) : value
  return normalized == null ? '' : String(normalized)
}

export function parsePricingInput(value: string, unit: ModelCatalogPricingUnit): number | undefined {
  if (value.trim() === '') {
    return undefined
  }
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed < 0) {
    return undefined
  }
  if (unit === 'threshold') {
    return Number.isInteger(parsed) && parsed > 0 ? parsed : undefined
  }
  return unit === 'token' ? millionPriceToToken(parsed) : parsed
}

export function hasTieredPricing(pricing?: ModelCatalogPricing): boolean {
  if (!pricing) {
    return false
  }
  return Boolean(
    (pricing.input_token_threshold && pricing.input_cost_per_token_above_threshold) ||
      (pricing.output_token_threshold && pricing.output_cost_per_token_above_threshold)
  )
}
