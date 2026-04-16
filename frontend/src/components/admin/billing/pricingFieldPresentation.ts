import type { BillingPricingSimpleSpecial } from '@/api/admin/billing'

export type RootNumberField =
  | 'input_price'
  | 'output_price'
  | 'cache_price'
  | 'input_price_above_threshold'
  | 'output_price_above_threshold'

export type SpecialNumberField = keyof BillingPricingSimpleSpecial
export type BillingPricingFieldId = RootNumberField | SpecialNumberField

export type PricingFieldUnit = 'per_million_tokens' | 'per_request' | 'per_image'

const REQUEST_FIELD_IDS: BillingPricingFieldId[] = [
  'grounding_search',
  'grounding_maps',
]

const IMAGE_FIELD_IDS: BillingPricingFieldId[] = [
  'output_price',
  'batch_output_price',
]

export function resolvePricingFieldUnit(
  fieldId: BillingPricingFieldId,
  outputChargeSlot: string = 'text_output',
): PricingFieldUnit {
  if (REQUEST_FIELD_IDS.includes(fieldId)) {
    return 'per_request'
  }

  if (IMAGE_FIELD_IDS.includes(fieldId) && outputChargeSlot === 'image_output') {
    return 'per_image'
  }

  if (IMAGE_FIELD_IDS.includes(fieldId) && outputChargeSlot === 'video_request') {
    return 'per_request'
  }

  return 'per_million_tokens'
}

export function pricingFieldUnitLabel(unit: PricingFieldUnit): string {
  switch (unit) {
    case 'per_request':
      return '$ / 次'
    case 'per_image':
      return '$ / 张'
    default:
      return '$ / M Tokens'
  }
}

export function pricingFieldUnitLabelForField(
  fieldId: BillingPricingFieldId,
  outputChargeSlot?: string,
): string {
  return pricingFieldUnitLabel(resolvePricingFieldUnit(fieldId, outputChargeSlot))
}
