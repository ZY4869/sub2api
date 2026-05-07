import { describe, expect, it } from 'vitest'
import {
  applyBillingPricingLayerPatch,
  billingPricingLayerPatchHasChanges,
  buildBillingPricingPatchFileV1,
  materializeBillingPricingPatchFileV1,
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
    tier_threshold_tokens: undefined,
    input_price_above_threshold: undefined,
    output_price_above_threshold: undefined,
    multiplier_enabled: false,
    multiplier_mode: undefined,
    shared_multiplier: undefined,
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

  it('supports special, multiplier, and null-clear patches', () => {
    const base = {
      ...createBaseForm(),
      special_enabled: true,
      special: {
        grounding_search: 0.01,
      },
      multiplier_enabled: true,
      multiplier_mode: 'item',
      item_multipliers: {
        input_price: 0.8,
        output_price: 0.9,
      },
    }

    const next = applyBillingPricingLayerPatch(base as any, {
      special: {
        grounding_search: null,
        grounding_maps: 0.02,
      },
      shared_multiplier: 0.7,
      multiplier_mode: 'shared',
      item_multipliers: {
        input_price: null,
      },
    })

    expect(next.special.grounding_search).toBeUndefined()
    expect(next.special.grounding_maps).toBe(0.02)
    expect(next.multiplier_mode).toBe('shared')
    expect(next.shared_multiplier).toBe(0.7)
    expect(next.item_multipliers?.output_price).toBe(0.9)
    expect(next.item_multipliers?.input_price).toBeUndefined()
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

  it('builds executable templates with full patch structure', () => {
    const file = buildBillingPricingPatchFileV1([{
      model: 'gpt-5.4',
      display_name: 'GPT-5.4',
      provider: 'openai',
      mode: 'chat',
      currency: 'USD',
      pricing_status: 'missing',
      pricing_warnings: [],
      input_supported: true,
      output_charge_slot: 'text_output',
      supports_prompt_caching: true,
      supports_service_tier: false,
      long_context_input_token_threshold: 0,
      long_context_input_cost_multiplier: 0,
      long_context_output_cost_multiplier: 0,
      capabilities: {
        supports_tiered_pricing: true,
        supports_batch_pricing: true,
        supports_service_tier: false,
        supports_prompt_caching: true,
        supports_provider_special: true,
      },
      official_form: createBaseForm() as any,
      sale_form: createBaseForm() as any,
    }], { executableTemplate: true })

    expect(file.export_mode).toBe('executable_template')
    expect(file.models[0].patch.official).toMatchObject({
      input_price: 1,
      output_price: 2,
      cache_price: 0.1,
      special_enabled: false,
      special: expect.any(Object),
      tiered_enabled: false,
      multiplier_enabled: false,
      item_multipliers: {},
    })
  })

  it('validates the patch file envelope and nested patch fields', () => {
    expect(() => parseBillingPricingPatchFileV1(null)).toThrow()
    expect(() => parseBillingPricingPatchFileV1({ version: 2 })).toThrow()
    expect(() =>
      parseBillingPricingPatchFileV1({
        version: 1,
        kind: 'billing_pricing_patch',
        models: [{
          model: 'gpt-5.4',
          patch: {
            official: {
              special: {
                grounding_search: 0.03,
              },
              item_multipliers: {
                input_price: null,
              },
            },
          },
        }],
      }),
    ).not.toThrow()
  })

  it('materializes issue worklist entries from known current prices and preserves currency', () => {
    const result = materializeBillingPricingPatchFileV1({
      version: 1,
      kind: 'billing_pricing_patch',
      generated_at: '2026-05-06T12:37:48Z',
      export_mode: 'issue_worklist',
      models: [{
        model: 'command-r',
        currency: 'USD',
        current: {
          official: {
            ...createBaseForm(),
            input_price: 1.5e-7,
            output_price: 6e-7,
            cache_price: undefined,
          } as any,
          sale: {
            ...createBaseForm(),
            input_price: undefined,
            output_price: undefined,
            cache_price: undefined,
          } as any,
        },
        patch: {},
        notes: '',
      }],
    })

    expect(result.updated).toBe(1)
    expect(result.skipped).toBe(0)
    expect(result.file.models[0]?.currency).toBe('USD')
    expect(result.file.models[0]?.patch.official).toEqual({
      input_price: 1.5e-7,
      output_price: 6e-7,
    })
  })

  it('skips models when both official and sale have no known price fields', () => {
    const result = materializeBillingPricingPatchFileV1({
      version: 1,
      kind: 'billing_pricing_patch',
      generated_at: '2026-05-06T12:37:48Z',
      export_mode: 'issue_worklist',
      models: [{
        model: 'missing-model',
        currency: 'CNY',
        current: {
          official: {
            ...createBaseForm(),
            input_price: undefined,
            output_price: undefined,
            cache_price: undefined,
          } as any,
          sale: {
            ...createBaseForm(),
            input_price: undefined,
            output_price: undefined,
            cache_price: undefined,
          } as any,
        },
        patch: {},
        notes: '',
      }],
    })

    expect(result.updated).toBe(0)
    expect(result.skipped).toBe(1)
    expect(result.file.models).toHaveLength(0)
  })
})
