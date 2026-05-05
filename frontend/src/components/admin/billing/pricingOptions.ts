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

export type BillingPricingValidationErrors = Record<string, string>

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

export function normalizeBillingPricingLayerFormForSave(
  form?: Partial<BillingPricingLayerForm>,
): BillingPricingLayerForm {
  const next = cloneBillingPricingLayerForm(form)

  if (!next.special_enabled) {
    next.special = createEmptyBillingPricingSpecial()
  }
  if (!next.tiered_enabled) {
    next.tier_threshold_tokens = undefined
    next.input_price_above_threshold = undefined
    next.output_price_above_threshold = undefined
  }
  if (!next.multiplier_enabled) {
    next.multiplier_mode = undefined
    next.shared_multiplier = undefined
    next.item_multipliers = {}
    return next
  }

  next.multiplier_mode = normalizeBillingPricingMultiplierMode(next.multiplier_mode)
  if (next.multiplier_mode === 'shared') {
    next.item_multipliers = {}
    return next
  }

  next.shared_multiplier = undefined
  next.item_multipliers = Object.fromEntries(
    Object.entries(next.item_multipliers || {}).filter(([fieldId, value]) =>
      fieldId.trim() && typeof value === 'number' && Number.isFinite(value),
    ),
  )
  return next
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

export function validateBillingPricingLayerFormForSave(
  form?: Partial<BillingPricingLayerForm>,
): BillingPricingValidationErrors {
  const normalized = normalizeBillingPricingLayerFormForSave(form)
  const errors: BillingPricingValidationErrors = {}

  const validateNonNegative = (value: number | undefined, fieldId: string, label: string) => {
    if (value == null) {
      return
    }
    if (!Number.isFinite(value) || value < 0) {
      errors[fieldId] = `${label}必须是非负数`
    }
  }

  validateNonNegative(normalized.input_price, 'input_price', '输入定价')
  validateNonNegative(normalized.output_price, 'output_price', '输出定价')
  validateNonNegative(normalized.cache_price, 'cache_price', '缓存定价')
  validateNonNegative(normalized.input_price_above_threshold, 'input_price_above_threshold', '输入阈值后定价')
  validateNonNegative(normalized.output_price_above_threshold, 'output_price_above_threshold', '输出阈值后定价')
  validateNonNegative(normalized.special.batch_input_price, 'batch_input_price', 'Batch 输入定价')
  validateNonNegative(normalized.special.batch_output_price, 'batch_output_price', 'Batch 输出定价')
  validateNonNegative(normalized.special.batch_cache_price, 'batch_cache_price', 'Batch 缓存定价')
  validateNonNegative(normalized.special.grounding_search, 'grounding_search', 'Grounding Search')
  validateNonNegative(normalized.special.grounding_maps, 'grounding_maps', 'Grounding Maps')
  validateNonNegative(normalized.special.file_search_embedding, 'file_search_embedding', 'File Search Embedding')
  validateNonNegative(normalized.special.file_search_retrieval, 'file_search_retrieval', 'File Search Retrieval')

  if (normalized.tiered_enabled) {
    if (!Number.isInteger(normalized.tier_threshold_tokens) || (normalized.tier_threshold_tokens || 0) <= 0) {
      errors.tier_threshold_tokens = '共享阈值必须是正整数'
    }
    if (normalized.input_price_above_threshold == null && normalized.output_price_above_threshold == null) {
      const message = '至少填写一个阈值后价格'
      errors.input_price_above_threshold = message
      errors.output_price_above_threshold = message
    }
  }

  if (normalized.multiplier_enabled) {
    if (normalized.multiplier_mode === 'shared') {
      validateNonNegative(normalized.shared_multiplier, 'shared_multiplier', '统一倍率')
    } else {
      Object.entries(normalized.item_multipliers || {}).forEach(([fieldId, value]) => {
        if (!Number.isFinite(value) || value < 0) {
          errors[`item_multipliers.${fieldId}`] = `${fieldLabelForValidation(fieldId)}倍率必须是非负数`
        }
      })
    }
  }

  return errors
}

export function hasBillingPricingValidationErrors(errors: BillingPricingValidationErrors): boolean {
  return Object.keys(errors).length > 0
}

function fieldLabelForValidation(fieldId: string): string {
  switch (fieldId) {
    case 'input_price':
      return '输入定价'
    case 'output_price':
      return '输出定价'
    case 'cache_price':
      return '缓存定价'
    case 'input_price_above_threshold':
      return '输入阈值后定价'
    case 'output_price_above_threshold':
      return '输出阈值后定价'
    case 'batch_input_price':
      return 'Batch 输入定价'
    case 'batch_output_price':
      return 'Batch 输出定价'
    case 'batch_cache_price':
      return 'Batch 缓存定价'
    case 'grounding_search':
      return 'Grounding Search'
    case 'grounding_maps':
      return 'Grounding Maps'
    case 'file_search_embedding':
      return 'File Search Embedding'
    case 'file_search_retrieval':
      return 'File Search Retrieval'
    default:
      return fieldId
  }
}
