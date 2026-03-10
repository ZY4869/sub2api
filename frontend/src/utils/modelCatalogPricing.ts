import { TOKENS_PER_MILLION } from './usagePricing'
import type { ModelCatalogPricing } from '@/api/admin/models'

export type ModelCatalogPricingKey = keyof ModelCatalogPricing
export type ModelCatalogPricingUnit = 'token' | 'image'

export const MODEL_CATALOG_PRICING_FIELDS: Array<{
  key: ModelCatalogPricingKey
  labelKey: string
  unit: ModelCatalogPricingUnit
}> = [
  { key: 'input_cost_per_token', labelKey: 'admin.models.fields.inputCost', unit: 'token' },
  { key: 'input_cost_per_token_priority', labelKey: 'admin.models.fields.inputPriorityCost', unit: 'token' },
  { key: 'output_cost_per_token', labelKey: 'admin.models.fields.outputCost', unit: 'token' },
  { key: 'output_cost_per_token_priority', labelKey: 'admin.models.fields.outputPriorityCost', unit: 'token' },
  { key: 'cache_creation_input_token_cost', labelKey: 'admin.models.fields.cacheCreationCost', unit: 'token' },
  { key: 'cache_creation_input_token_cost_above_1hr', labelKey: 'admin.models.fields.cacheCreationCostAbove1h', unit: 'token' },
  { key: 'cache_read_input_token_cost', labelKey: 'admin.models.fields.cacheReadCost', unit: 'token' },
  { key: 'cache_read_input_token_cost_priority', labelKey: 'admin.models.fields.cacheReadPriorityCost', unit: 'token' },
  { key: 'output_cost_per_image', labelKey: 'admin.models.fields.imageCost', unit: 'image' }
]

export function tokenPriceToMillion(value?: number): number | null {
  return typeof value === 'number' ? value * TOKENS_PER_MILLION : null
}

export function millionPriceToToken(value?: number | null): number | undefined {
  return typeof value === 'number' ? value / TOKENS_PER_MILLION : undefined
}

export function formatModelCatalogPrice(value?: number, unit: ModelCatalogPricingUnit = 'token'): string {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '-'
  }
  const normalized = unit === 'token' ? value * TOKENS_PER_MILLION : value
  return `$${normalized.toFixed(normalized >= 1 ? 4 : 6)}`
}

export function pricingInputValue(value: number | undefined, unit: ModelCatalogPricingUnit): string {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return ''
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
  return unit === 'token' ? millionPriceToToken(parsed) : parsed
}
