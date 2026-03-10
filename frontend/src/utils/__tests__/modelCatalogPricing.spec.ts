import { describe, expect, it } from 'vitest'
import {
  formatModelCatalogPrice,
  hasTieredPricing,
  millionPriceToToken,
  parsePricingInput,
  pricingInputValue,
  tokenPriceToMillion
} from '@/utils/modelCatalogPricing'

describe('modelCatalogPricing utils', () => {
  it('converts token price to and from per-million units', () => {
    expect(tokenPriceToMillion(0.0000025)).toBe(2.5)
    expect(millionPriceToToken(2.5)).toBe(0.0000025)
  })

  it('formats token and image prices for display', () => {
    expect(formatModelCatalogPrice(0.0000025, 'token')).toBe('$2.5000')
    expect(formatModelCatalogPrice(0.04, 'image')).toBe('$0.040000')
    expect(formatModelCatalogPrice(128000, 'threshold')).toBe('128000 tokens')
  })

  it('prepares editable values in UI units', () => {
    expect(pricingInputValue(0.00000125, 'token')).toBe('1.25')
    expect(pricingInputValue(0.08, 'image')).toBe('0.08')
  })

  it('parses only valid non-negative inputs', () => {
    expect(parsePricingInput('2.5', 'token')).toBe(0.0000025)
    expect(parsePricingInput('0.08', 'image')).toBe(0.08)
    expect(parsePricingInput('128000', 'threshold')).toBe(128000)
    expect(parsePricingInput('-1', 'token')).toBeUndefined()
    expect(parsePricingInput('abc', 'image')).toBeUndefined()
    expect(parsePricingInput('', 'image')).toBeUndefined()
    expect(parsePricingInput('12.5', 'threshold')).toBeUndefined()
  })

  it('detects tiered pricing only when threshold and above-threshold price coexist', () => {
    expect(hasTieredPricing()).toBe(false)
    expect(
      hasTieredPricing({
        input_token_threshold: 128000,
        input_cost_per_token_above_threshold: 0.000003
      })
    ).toBe(true)
    expect(
      hasTieredPricing({
        output_token_threshold: 64000
      })
    ).toBe(false)
  })
})
