import { describe, expect, it } from 'vitest'
import {
  defaultUnitForChargeSlot,
  newBillingPriceItem,
} from '../pricingOptions'

describe('pricingOptions', () => {
  it('maps cache storage and media charge slots to the expected default units', () => {
    expect(defaultUnitForChargeSlot('cache_storage_token_hour')).toBe('cache_storage_token_hour')
    expect(defaultUnitForChargeSlot('image_output')).toBe('image')
    expect(defaultUnitForChargeSlot('video_request')).toBe('video_request')
  })

  it.each([
    {
      charge_slot: 'text_input',
      mode: 'tiered',
      threshold_tokens: 200000,
      price: 1.2,
      price_above_threshold: 2.4,
      enabled: false,
    },
    {
      charge_slot: 'text_output',
      mode: 'batch',
      batch_mode: 'batch',
      price: 0.8,
    },
    {
      charge_slot: 'image_output',
      mode: 'service_tier',
      service_tier: 'priority',
      price: 3.5,
    },
    {
      charge_slot: 'cache_storage_token_hour',
      mode: 'provider_special',
      service_tier: 'flex',
      batch_mode: 'batch',
      surface: 'live',
      operation_type: 'grounding',
      input_modality: 'text',
      output_modality: 'text',
      cache_phase: 'storage',
      grounding_kind: 'search',
      context_window: 'long',
      formula_source: 'gemini',
      formula_multiplier: 1.5,
      rule_id: 'rule-gemini-live',
      derived_via: 'provider_special',
      price: 4.2,
    },
  ] as const)('preserves seeded billing fields for $mode items', (seed) => {
    const item = newBillingPriceItem('sale', seed)

    expect(item).toMatchObject({
      ...seed,
      layer: 'sale',
      unit: defaultUnitForChargeSlot(seed.charge_slot),
    })
  })
})
