import type {
  BillingPricingFieldId,
} from './pricingFieldPresentation'
import type {
  BillingPricingLayerForm,
  BillingPricingMultiplierMode,
  BillingPricingSheetDetail,
  BillingPricingSimpleSpecial,
} from '@/api/admin/billing'
import { normalizeBillingPricingCurrency } from './pricingCurrency'

export const BILLING_DISCOUNT_FIELD_IDS = {
  input_price: 'input_price',
  output_price: 'output_price',
  cache_price: 'cache_price',
  input_price_above_threshold: 'input_price_above_threshold',
  output_price_above_threshold: 'output_price_above_threshold',
  batch_input_price: 'batch_input_price',
  batch_output_price: 'batch_output_price',
  batch_cache_price: 'batch_cache_price',
  grounding_search: 'grounding_search',
  grounding_maps: 'grounding_maps',
  file_search_embedding: 'file_search_embedding',
  file_search_retrieval: 'file_search_retrieval',
} as const

export function createEmptyBillingPricingSpecial(): BillingPricingSimpleSpecial {
  return {}
}

export function createEmptyBillingPricingLayerForm(seed: Partial<BillingPricingLayerForm> = {}): BillingPricingLayerForm {
  return {
    input_price: seed.input_price,
    output_price: seed.output_price,
    cache_price: seed.cache_price,
    special_enabled: seed.special_enabled ?? false,
    special: {
      ...createEmptyBillingPricingSpecial(),
      ...(seed.special || {}),
    },
    tiered_enabled: seed.tiered_enabled ?? false,
    tier_threshold_tokens: seed.tier_threshold_tokens,
    input_price_above_threshold: seed.input_price_above_threshold,
    output_price_above_threshold: seed.output_price_above_threshold,
    multiplier_enabled: seed.multiplier_enabled ?? false,
    multiplier_mode: seed.multiplier_mode,
    shared_multiplier: seed.shared_multiplier,
    item_multipliers: seed.item_multipliers ? { ...seed.item_multipliers } : {},
  }
}

export function cloneBillingPricingLayerForm(form?: Partial<BillingPricingLayerForm>): BillingPricingLayerForm {
  return createEmptyBillingPricingLayerForm(form || {})
}

export function normalizeBillingPricingSheetDetail(detail: BillingPricingSheetDetail): BillingPricingSheetDetail {
  return {
    ...detail,
    currency: normalizeBillingPricingCurrency(detail.currency),
    official_form: cloneBillingPricingLayerForm(detail.official_form),
    sale_form: cloneBillingPricingLayerForm(detail.sale_form),
  }
}

export function outputPriceLabel(outputChargeSlot?: string): string {
  switch (outputChargeSlot) {
    case 'image_output':
      return '图片输出定价'
    case 'video_request':
      return '视频请求定价'
    default:
      return '输出定价'
  }
}

export function billingLayerHasValues(form?: Partial<BillingPricingLayerForm>): boolean {
  if (!form) return false
  return countConfiguredBillingFields(form) > 0
}

export function billingLayerHasSpecialValues(form?: Partial<BillingPricingLayerForm>): boolean {
  if (!form) return false

  return [
    form.special?.batch_input_price,
    form.special?.batch_output_price,
    form.special?.batch_cache_price,
    form.special?.grounding_search,
    form.special?.grounding_maps,
    form.special?.file_search_embedding,
    form.special?.file_search_retrieval,
  ].some((value) => value != null)
}

export function countConfiguredBillingFields(form?: Partial<BillingPricingLayerForm>): number {
  if (!form) return 0

  const rootValues = [
    form.input_price,
    form.output_price,
    form.cache_price,
    form.input_price_above_threshold,
    form.output_price_above_threshold,
  ]
  const specialValues = [
    form.special?.batch_input_price,
    form.special?.batch_output_price,
    form.special?.batch_cache_price,
    form.special?.grounding_search,
    form.special?.grounding_maps,
    form.special?.file_search_embedding,
    form.special?.file_search_retrieval,
  ]

  return [...rootValues, ...specialValues].filter((value) => value != null).length
}

export function normalizeBillingPricingMultiplierMode(
  mode?: BillingPricingMultiplierMode,
): BillingPricingMultiplierMode {
  return mode === 'item' ? 'item' : 'shared'
}

export function resolveBillingPricingFieldValue(
  form: Partial<BillingPricingLayerForm> | undefined,
  fieldId: BillingPricingFieldId,
): number | undefined {
  if (!form) {
    return undefined
  }

  switch (fieldId) {
    case 'input_price':
    case 'output_price':
    case 'cache_price':
      return form[fieldId]
    case 'input_price_above_threshold':
    case 'output_price_above_threshold':
      return form.tiered_enabled ? form[fieldId] : undefined
    default:
      if (!form.special_enabled && !form.special?.[fieldId]) {
        return undefined
      }
      return form.special?.[fieldId]
  }
}

export function resolveBillingPricingFieldMultiplier(
  form: Partial<BillingPricingLayerForm> | undefined,
  fieldId: BillingPricingFieldId,
): number | undefined {
  if (!form?.multiplier_enabled) {
    return undefined
  }
  if (normalizeBillingPricingMultiplierMode(form.multiplier_mode) === 'item') {
    return form.item_multipliers?.[fieldId] ?? 1
  }
  return form.shared_multiplier ?? 1
}

export function resolveEffectiveBillingPricingFieldValue(
  form: Partial<BillingPricingLayerForm> | undefined,
  fieldId: BillingPricingFieldId,
): number | undefined {
  const baseValue = resolveBillingPricingFieldValue(form, fieldId)
  if (baseValue == null) {
    return undefined
  }
  const multiplier = resolveBillingPricingFieldMultiplier(form, fieldId)
  if (multiplier == null) {
    return baseValue
  }
  return baseValue * multiplier
}
