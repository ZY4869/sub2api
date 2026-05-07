import type {
  BillingPricingCurrency,
  BillingPricingLayerForm,
  BillingPricingMultiplierMode,
  BillingPricingSheetDetail,
  BillingPricingStatus,
} from '@/api/admin/billing'
import {
  cloneBillingPricingLayerForm,
  createEmptyBillingPricingSpecial,
} from '@/components/admin/billing/pricingOptions'

type BillingPricingSpecialPatchV1 = Partial<Record<
  keyof BillingPricingLayerForm['special'],
  number | null
>>

type BillingPricingSpecialPatchNormalized = Partial<Record<
  keyof BillingPricingLayerForm['special'],
  number
>>

interface BillingPricingLayerPatchNormalized {
  input_price?: number
  output_price?: number
  cache_price?: number
  special_enabled?: boolean
  special?: BillingPricingSpecialPatchNormalized
  tiered_enabled?: boolean
  tier_threshold_tokens?: number
  input_price_above_threshold?: number
  output_price_above_threshold?: number
  multiplier_enabled?: boolean
  multiplier_mode?: BillingPricingMultiplierMode
  shared_multiplier?: number
  item_multipliers?: Record<string, number | null>
}

export interface BillingPricingLayerPatchV1 {
  input_price?: number | null
  output_price?: number | null
  cache_price?: number | null
  special_enabled?: boolean | null
  special?: BillingPricingSpecialPatchV1
  tiered_enabled?: boolean | null
  tier_threshold_tokens?: number | null
  input_price_above_threshold?: number | null
  output_price_above_threshold?: number | null
  multiplier_enabled?: boolean | null
  multiplier_mode?: BillingPricingMultiplierMode | null
  shared_multiplier?: number | null
  item_multipliers?: Record<string, number | null> | null
}

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
  export_mode?: 'issue_worklist' | 'executable_template'
  models: BillingPricingPatchModelV1[]
}

export interface BillingPricingPatchMaterializationResult {
  file: BillingPricingPatchFileV1
  updated: number
  skipped: number
}

const rootNumberKeys = [
  'input_price',
  'output_price',
  'cache_price',
  'tier_threshold_tokens',
  'input_price_above_threshold',
  'output_price_above_threshold',
  'shared_multiplier',
] as const

const rootBooleanKeys = [
  'special_enabled',
  'tiered_enabled',
  'multiplier_enabled',
] as const

const specialKeys = [
  'batch_input_price',
  'batch_output_price',
  'batch_cache_price',
  'grounding_search',
  'grounding_maps',
  'file_search_embedding',
  'file_search_retrieval',
] as const

const multiplierModes = new Set<BillingPricingMultiplierMode>(['shared', 'item'])

type RootNumberKey = (typeof rootNumberKeys)[number]
type RootBooleanKey = (typeof rootBooleanKeys)[number]
type SpecialKey = (typeof specialKeys)[number]

function hasOwn(obj: unknown, key: string): boolean {
  return typeof obj === 'object' && obj !== null && Object.prototype.hasOwnProperty.call(obj, key)
}

function isFiniteNonNegativeNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

function normalizePatchNumber(value: unknown, key: RootNumberKey | SpecialKey): number | undefined {
  if (value === null) {
    return undefined
  }
  if (!isFiniteNonNegativeNumber(value)) {
    throw new Error(`patch.${key} must be a finite non-negative number or null`)
  }
  if (key === 'tier_threshold_tokens' && !Number.isInteger(value)) {
    throw new Error('patch.tier_threshold_tokens must be an integer')
  }
  return value
}

function normalizePatchBoolean(value: unknown, key: RootBooleanKey): boolean | undefined {
  if (value === null) {
    return undefined
  }
  if (typeof value !== 'boolean') {
    throw new Error(`patch.${key} must be a boolean or null`)
  }
  return value
}

function normalizePatchMultiplierMode(value: unknown): BillingPricingMultiplierMode | undefined {
  if (value === null) {
    return undefined
  }
  if (typeof value !== 'string' || !multiplierModes.has(value as BillingPricingMultiplierMode)) {
    throw new Error('patch.multiplier_mode must be "shared", "item", or null')
  }
  return value as BillingPricingMultiplierMode
}

function normalizePatchItemMultipliers(
  value: unknown,
): Record<string, number | null> | undefined {
  if (value === null) {
    return {}
  }
  if (value === undefined) {
    return undefined
  }
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    throw new Error('patch.item_multipliers must be an object or null')
  }

  const entries = Object.entries(value as Record<string, unknown>)
  const next: Record<string, number | null> = {}
  entries.forEach(([fieldId, multiplier]) => {
    if (!fieldId.trim()) {
      throw new Error('patch.item_multipliers keys must be non-empty strings')
    }
    if (multiplier === null) {
      next[fieldId] = null
      return
    }
    if (!isFiniteNonNegativeNumber(multiplier)) {
      throw new Error(`patch.item_multipliers.${fieldId} must be a finite non-negative number or null`)
    }
    next[fieldId] = multiplier
  })
  return next
}

function billingPricingLayerPatchTouchesTier(
  patch: BillingPricingLayerPatchNormalized,
): boolean {
  return (
    hasOwn(patch, 'tiered_enabled')
    || hasOwn(patch, 'tier_threshold_tokens')
    || hasOwn(patch, 'input_price_above_threshold')
    || hasOwn(patch, 'output_price_above_threshold')
  )
}

function billingPricingLayerPatchTouchesSpecial(
  patch: BillingPricingLayerPatchNormalized,
): boolean {
  return hasOwn(patch, 'special_enabled') || hasOwn(patch, 'special')
}

function billingPricingLayerPatchTouchesMultiplier(
  patch: BillingPricingLayerPatchNormalized,
): boolean {
  return (
    hasOwn(patch, 'multiplier_enabled')
    || hasOwn(patch, 'multiplier_mode')
    || hasOwn(patch, 'shared_multiplier')
    || hasOwn(patch, 'item_multipliers')
  )
}

function normalizePatchLayer(patch?: BillingPricingLayerPatchV1): BillingPricingLayerPatchNormalized | undefined {
  if (!patch) {
    return undefined
  }

  const next: BillingPricingLayerPatchNormalized = {}

  rootNumberKeys.forEach((key) => {
    if (!hasOwn(patch, key)) {
      return
    }
    next[key] = normalizePatchNumber(patch[key], key)
  })

  rootBooleanKeys.forEach((key) => {
    if (!hasOwn(patch, key)) {
      return
    }
    next[key] = normalizePatchBoolean(patch[key], key)
  })

  if (hasOwn(patch, 'multiplier_mode')) {
    next.multiplier_mode = normalizePatchMultiplierMode(patch.multiplier_mode)
  }

  if (hasOwn(patch, 'special')) {
    const raw = patch.special
    if (raw === null || raw === undefined) {
      next.special = {}
    } else if (typeof raw !== 'object' || Array.isArray(raw)) {
      throw new Error('patch.special must be an object or null')
    } else {
      const special: BillingPricingSpecialPatchNormalized = {}
      specialKeys.forEach((key) => {
        if (!hasOwn(raw, key)) {
          return
        }
        const rawValue = (raw as BillingPricingSpecialPatchV1)[key]
        if (rawValue === undefined) {
          return
        }
        special[key] = normalizePatchNumber(rawValue, key)
      })
      next.special = special
    }
  }

  if (hasOwn(patch, 'item_multipliers')) {
    next.item_multipliers = normalizePatchItemMultipliers(patch.item_multipliers)
  }

  return next
}

function patchLayerFieldHasChanges<T>(
  baseValue: T,
  patchValue: T | undefined,
  keyPresent: boolean,
): boolean {
  return keyPresent && baseValue !== patchValue
}

export function billingPricingLayerPatchHasChanges(
  base: BillingPricingLayerForm,
  patch?: BillingPricingLayerPatchV1,
): boolean {
  const normalizedPatch = normalizePatchLayer(patch)
  if (!normalizedPatch) {
    return false
  }

  if (rootNumberKeys.some((key) => patchLayerFieldHasChanges(base[key], normalizedPatch[key], hasOwn(normalizedPatch, key)))) {
    return true
  }
  if (rootBooleanKeys.some((key) => patchLayerFieldHasChanges(base[key], normalizedPatch[key], hasOwn(normalizedPatch, key)))) {
    return true
  }
  if (patchLayerFieldHasChanges(base.multiplier_mode, normalizedPatch.multiplier_mode, hasOwn(normalizedPatch, 'multiplier_mode'))) {
    return true
  }
  if (hasOwn(normalizedPatch, 'special')) {
    const patchSpecial = normalizedPatch.special || {}
    if (specialKeys.some((key) => hasOwn(patchSpecial, key) && base.special?.[key] !== (patchSpecial[key] ?? undefined))) {
      return true
    }
  }
  if (hasOwn(normalizedPatch, 'item_multipliers')) {
    const patchMultipliers = normalizedPatch.item_multipliers || {}
    const baseMultipliers = base.item_multipliers || {}
    const keys = new Set([...Object.keys(baseMultipliers), ...Object.keys(patchMultipliers)])
    for (const key of keys) {
      if (!hasOwn(patchMultipliers, key)) {
        continue
      }
      const patchValue = patchMultipliers[key]
      if ((patchValue ?? undefined) !== baseMultipliers[key]) {
        return true
      }
    }
  }
  return false
}

export function applyBillingPricingLayerPatch(
  base: BillingPricingLayerForm,
  patch?: BillingPricingLayerPatchV1,
): BillingPricingLayerForm {
  const normalizedPatch = normalizePatchLayer(patch)
  if (!normalizedPatch) {
    return base
  }

  if (!billingPricingLayerPatchHasChanges(base, normalizedPatch)) {
    return base
  }

  const next = cloneBillingPricingLayerForm(base)

  rootNumberKeys.forEach((key) => {
    if (!hasOwn(normalizedPatch, key)) {
      return
    }
    next[key] = normalizedPatch[key]
  })

  rootBooleanKeys.forEach((key) => {
    if (!hasOwn(normalizedPatch, key)) {
      return
    }
    const value = normalizedPatch[key]
    if (value !== undefined) {
      next[key] = value
    }
  })

  if (billingPricingLayerPatchTouchesTier(normalizedPatch) && !hasOwn(normalizedPatch, 'tiered_enabled')) {
    next.tiered_enabled = true
  }

  if (billingPricingLayerPatchTouchesSpecial(normalizedPatch) && !hasOwn(normalizedPatch, 'special_enabled')) {
    next.special_enabled = true
  }

  if (billingPricingLayerPatchTouchesMultiplier(normalizedPatch) && !hasOwn(normalizedPatch, 'multiplier_enabled')) {
    next.multiplier_enabled = true
  }

  if (hasOwn(normalizedPatch, 'special')) {
    const patchSpecial = normalizedPatch.special || {}
    next.special = {
      ...next.special,
    }
    specialKeys.forEach((key) => {
      if (!hasOwn(patchSpecial, key)) {
        return
      }
      next.special[key] = patchSpecial[key] ?? undefined
    })
  }

  if (hasOwn(normalizedPatch, 'multiplier_mode')) {
    next.multiplier_mode = normalizedPatch.multiplier_mode
  }

  if (hasOwn(normalizedPatch, 'item_multipliers')) {
    const patchMultipliers = normalizedPatch.item_multipliers || {}
    const nextItemMultipliers: Record<string, number> = {
      ...(next.item_multipliers || {}),
    }
    Object.entries(patchMultipliers).forEach(([fieldId, value]) => {
      if (value == null) {
        delete nextItemMultipliers[fieldId]
      } else {
        nextItemMultipliers[fieldId] = value
      }
    })
    next.item_multipliers = nextItemMultipliers
  }

  if (next.special_enabled === false) {
    next.special = createEmptyBillingPricingSpecial()
  }

  if (next.tiered_enabled === false) {
    next.tier_threshold_tokens = undefined
    next.input_price_above_threshold = undefined
    next.output_price_above_threshold = undefined
  }

  if (next.multiplier_enabled === false) {
    next.multiplier_mode = undefined
    next.shared_multiplier = undefined
    next.item_multipliers = {}
  }

  return next
}

function createTemplatePatchLayer(form: BillingPricingLayerForm): BillingPricingLayerPatchV1 {
  return {
    input_price: form.input_price ?? null,
    output_price: form.output_price ?? null,
    cache_price: form.cache_price ?? null,
    special_enabled: form.special_enabled,
    special: {
      batch_input_price: form.special.batch_input_price ?? null,
      batch_output_price: form.special.batch_output_price ?? null,
      batch_cache_price: form.special.batch_cache_price ?? null,
      grounding_search: form.special.grounding_search ?? null,
      grounding_maps: form.special.grounding_maps ?? null,
      file_search_embedding: form.special.file_search_embedding ?? null,
      file_search_retrieval: form.special.file_search_retrieval ?? null,
    },
    tiered_enabled: form.tiered_enabled,
    tier_threshold_tokens: form.tier_threshold_tokens ?? null,
    input_price_above_threshold: form.input_price_above_threshold ?? null,
    output_price_above_threshold: form.output_price_above_threshold ?? null,
    multiplier_enabled: form.multiplier_enabled,
    multiplier_mode: form.multiplier_mode ?? null,
    shared_multiplier: form.shared_multiplier ?? null,
    item_multipliers: Object.keys(form.item_multipliers || {}).length > 0
      ? Object.fromEntries(Object.entries(form.item_multipliers || {}).map(([key, value]) => [key, value]))
      : {},
  }
}

function hasAnyPatchLayer(patch?: BillingPricingPatchPayloadV1): boolean {
  if (!patch || typeof patch !== 'object') {
    return false
  }
  return Object.keys(patch).some((layerKey) => {
    const layer = patch[layerKey as keyof BillingPricingPatchPayloadV1]
    return typeof layer === 'object' && layer !== null && Object.keys(layer).length > 0
  })
}

function materializePatchLayerFromCurrent(form?: BillingPricingLayerForm): BillingPricingLayerPatchV1 | undefined {
  if (!form) {
    return undefined
  }
  const layer = createTemplatePatchLayer(form)
  const next: BillingPricingLayerPatchV1 = {}
  let hasValue = false

  ;(['input_price', 'output_price', 'cache_price'] as const).forEach((key) => {
    const value = layer[key]
    if (typeof value === 'number' && Number.isFinite(value)) {
      next[key] = value
      hasValue = true
    }
  })

  if (!hasValue) {
    return undefined
  }

  return next
}

function materializePatchModelEntry(entry: BillingPricingPatchModelV1): BillingPricingPatchModelV1 | null {
  if (hasAnyPatchLayer(entry.patch)) {
    return entry
  }

  const official = materializePatchLayerFromCurrent(entry.current?.official)
  const sale = materializePatchLayerFromCurrent(entry.current?.sale)
  if (!official && !sale) {
    return null
  }

  const patch: BillingPricingPatchPayloadV1 = {}
  if (official) {
    patch.official = official
  }
  if (sale) {
    patch.sale = sale
  }

  return {
    ...entry,
    currency: entry.currency,
    patch,
    notes: entry.notes || 'Auto-built confirmed patch from current.official/current.sale known price fields only.',
  }
}

export function materializeBillingPricingPatchFileV1(
  file: BillingPricingPatchFileV1,
): BillingPricingPatchMaterializationResult {
  let updated = 0
  let skipped = 0

  const models = (file.models || []).reduce<BillingPricingPatchModelV1[]>((result, entry) => {
    const next = materializePatchModelEntry(entry)
    if (!next) {
      skipped += 1
      return result
    }
    if (next !== entry) {
      updated += 1
    }
    result.push(next)
    return result
  }, [])

  return {
    file: {
      ...file,
      export_mode: 'executable_template',
      models,
    },
    updated,
    skipped,
  }
}

export function buildBillingPricingPatchFileV1(
  details: BillingPricingSheetDetail[],
  options: { executableTemplate?: boolean } = {},
): BillingPricingPatchFileV1 {
  const executableTemplate = options.executableTemplate === true
  return {
    version: 1,
    kind: 'billing_pricing_patch',
    generated_at: new Date().toISOString(),
    export_mode: executableTemplate ? 'executable_template' : 'issue_worklist',
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
      patch: executableTemplate
        ? {
            official: createTemplatePatchLayer(detail.official_form),
            sale: createTemplatePatchLayer(detail.sale_form),
          }
        : {},
      notes: executableTemplate
        ? '可执行模板：保留全字段结构，可直接填值或填 null 清空。'
        : '待处理工作清单：默认 patch 为空，请按需填写官方价或 sale 价。',
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

  const models = (raw as { models: unknown[] }).models.map((entry, index) => {
    if (typeof entry !== 'object' || entry === null) {
      throw new Error(`Invalid JSON: models[${index}] must be an object`)
    }
    if (!hasOwn(entry, 'model') || typeof (entry as { model?: unknown }).model !== 'string') {
      throw new Error(`Invalid JSON: models[${index}].model must be a string`)
    }

    const patch = hasOwn(entry, 'patch') ? (entry as { patch?: unknown }).patch : {}
    if (patch !== undefined && (typeof patch !== 'object' || patch === null || Array.isArray(patch))) {
      throw new Error(`Invalid JSON: models[${index}].patch must be an object`)
    }

    if (patch && hasOwn(patch, 'official')) {
      normalizePatchLayer((patch as { official?: BillingPricingLayerPatchV1 }).official)
    }
    if (patch && hasOwn(patch, 'sale')) {
      normalizePatchLayer((patch as { sale?: BillingPricingLayerPatchV1 }).sale)
    }
    return entry as BillingPricingPatchModelV1
  })

  return {
    version: 1,
    kind: 'billing_pricing_patch',
    generated_at: String((raw as { generated_at?: unknown }).generated_at || ''),
    export_mode: (raw as { export_mode?: BillingPricingPatchFileV1['export_mode'] }).export_mode,
    models,
  }
}
