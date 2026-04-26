import { describe, expect, it } from 'vitest'
import {
  applyBillingPricingLayerPatch,
  billingPricingLayerPatchHasChanges,
  parseBillingPricingPatchFileV1,
} from '../billingPricingPatch'

function createBaseForm() {
  return {
    input_price: 1,
    output_price: 2,
    cache_price: 0.1,
    special_enabled: false,
    special: {},
    tiered_enabled: false,
    multiplier_enabled: false,
    item_multipliers: {},
  }
}

describe('billingPricingPatch', () => {
  it('applies only provided patch fields', () => {
    const base = createBaseForm()
    expect(billingPricingLayerPatchHasChanges(base as any, { input_price: 1 })).toBe(false)
    expect(billingPricingLayerPatchHasChanges(base as any, { input_price: 1.25 })).toBe(true)

    const next = applyBillingPricingLayerPatch(base as any, { input_price: 1.25 })
    expect(next).toMatchObject({
      input_price: 1.25,
      output_price: 2,
      cache_price: 0.1,
      tiered_enabled: false,
    })
  })

  it('auto-enables tiered pricing when tier fields are present', () => {
    const base = createBaseForm()
    const next = applyBillingPricingLayerPatch(base as any, {
      tier_threshold_tokens: 200000,
      input_price_above_threshold: 2,
      output_price_above_threshold: 3,
    })
    expect(next.tiered_enabled).toBe(true)
    expect(next.tier_threshold_tokens).toBe(200000)
    expect(next.input_price_above_threshold).toBe(2)
    expect(next.output_price_above_threshold).toBe(3)
  })

  it('validates the basic patch file envelope', () => {
    expect(() => parseBillingPricingPatchFileV1(null)).toThrow()
    expect(() => parseBillingPricingPatchFileV1({ version: 2 })).toThrow()
    expect(() =>
      parseBillingPricingPatchFileV1({ version: 1, kind: 'billing_pricing_patch', models: [{ model: 'gpt-5.4' }] }),
    ).not.toThrow()
  })
})

