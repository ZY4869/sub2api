import type {
  BillingPricingCurrency,
  BillingPricingLayerForm,
  BillingPricingSheetDetail,
  BillingPricingStatus,
} from '@/api/admin/billing'

export type BillingPricingLayerPatchV1 = Partial<
  Pick<
    BillingPricingLayerForm,
    | 'input_price'
    | 'output_price'
    | 'cache_price'
    | 'tier_threshold_tokens'
    | 'input_price_above_threshold'
    | 'output_price_above_threshold'
  >
>

export interface BillingPricingPatchPayloadV1 {
  official?: BillingPricingLayerPatchV1
  sale?: BillingPricingLayerPatchV1
}

export interface BillingPricingCurrentPayloadV1 {
  official: BillingPricingLayerForm
  sale: BillingPricingLayerForm
}

export interface BillingPricingPatchModelV1 {
  model: string
  display_name?: string
  provider?: string
  mode?: string
  currency?: BillingPricingCurrency
  pricing_status?: BillingPricingStatus
  pricing_warnings?: string[]
  current: BillingPricingCurrentPayloadV1
  patch: BillingPricingPatchPayloadV1
  notes: string
}

export interface BillingPricingPatchFileV1 {
  version: 1
  kind: 'billing_pricing_patch'
  generated_at: string
  models: BillingPricingPatchModelV1[]
}

const patchKeys = [
  'input_price',
  'output_price',
  'cache_price',
  'tier_threshold_tokens',
  'input_price_above_threshold',
  'output_price_above_threshold',
] as const

type PatchKey = (typeof patchKeys)[number]

function hasOwn(obj: unknown, key: string): boolean {
  return typeof obj === 'object' && obj !== null && Object.prototype.hasOwnProperty.call(obj, key)
}

function normalizePatchNumber(value: unknown, key: PatchKey): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    throw new Error(`patch.${key} must be a finite number`)
  }
  if (value < 0) {
    throw new Error(`patch.${key} must be >= 0`)
  }
  if (key === 'tier_threshold_tokens' && !Number.isInteger(value)) {
    throw new Error('patch.tier_threshold_tokens must be an integer')
  }
  return value
}

function tierPatchEnabled(patch: BillingPricingLayerPatchV1): boolean {
  return (
    hasOwn(patch, 'tier_threshold_tokens')
    || hasOwn(patch, 'input_price_above_threshold')
    || hasOwn(patch, 'output_price_above_threshold')
  )
}

export function billingPricingLayerPatchHasChanges(
  base: BillingPricingLayerForm,
  patch?: BillingPricingLayerPatchV1,
): boolean {
  if (!patch) {
    return false
  }
  if (tierPatchEnabled(patch) && !base.tiered_enabled) {
    return true
  }
  return patchKeys.some((key) => hasOwn(patch, key) && base[key] !== patch[key])
}

export function applyBillingPricingLayerPatch(
  base: BillingPricingLayerForm,
  patch?: BillingPricingLayerPatchV1,
): BillingPricingLayerForm {
  if (!patch || typeof patch !== 'object') {
    return base
  }

  if (!billingPricingLayerPatchHasChanges(base, patch)) {
    return base
  }

  const next: BillingPricingLayerForm = {
    ...base,
    special: { ...(base.special || {}) },
    item_multipliers: { ...(base.item_multipliers || {}) },
  }

  if (tierPatchEnabled(patch)) {
    next.tiered_enabled = true
  }

  patchKeys.forEach((key) => {
    if (!hasOwn(patch, key)) {
      return
    }
    const rawValue = patch[key] as unknown
    const value = normalizePatchNumber(rawValue, key)
    next[key] = value
  })

  return next
}

export function buildBillingPricingPatchFileV1(details: BillingPricingSheetDetail[]): BillingPricingPatchFileV1 {
  return {
    version: 1,
    kind: 'billing_pricing_patch',
    generated_at: new Date().toISOString(),
    models: (details || []).map((detail) => ({
      model: detail.model,
      display_name: detail.display_name,
      provider: detail.provider,
      mode: detail.mode,
      currency: detail.currency,
      pricing_status: detail.pricing_status,
      pricing_warnings: detail.pricing_warnings || [],
      current: {
        official: detail.official_form,
        sale: detail.sale_form,
      },
      patch: {},
      notes: '',
    })),
  }
}

export function parseBillingPricingPatchFileV1(raw: unknown): BillingPricingPatchFileV1 {
  if (typeof raw !== 'object' || raw === null) {
    throw new Error('Invalid JSON: root must be an object')
  }
  if (!hasOwn(raw, 'version') || (raw as { version?: unknown }).version !== 1) {
    throw new Error('Invalid JSON: version must be 1')
  }
  if (!hasOwn(raw, 'kind') || (raw as { kind?: unknown }).kind !== 'billing_pricing_patch') {
    throw new Error('Invalid JSON: kind must be billing_pricing_patch')
  }
  if (!hasOwn(raw, 'models') || !Array.isArray((raw as { models?: unknown }).models)) {
    throw new Error('Invalid JSON: models must be an array')
  }

  const models = (raw as { models: unknown[] }).models
  models.forEach((entry, index) => {
    if (typeof entry !== 'object' || entry === null) {
      throw new Error(`Invalid JSON: models[${index}] must be an object`)
    }
    if (!hasOwn(entry, 'model') || typeof (entry as { model?: unknown }).model !== 'string') {
      throw new Error(`Invalid JSON: models[${index}].model must be a string`)
    }
  })

  return raw as BillingPricingPatchFileV1
}
