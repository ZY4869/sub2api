import { describe, expect, it } from 'vitest'
import {
  formatModelCatalogPrice,
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
  })

  it('prepares editable values in UI units', () => {
    expect(pricingInputValue(0.00000125, 'token')).toBe('1.25')
    expect(pricingInputValue(0.08, 'image')).toBe('0.08')
  })

  it('parses only valid non-negative inputs', () => {
    expect(parsePricingInput('2.5', 'token')).toBe(0.0000025)
    expect(parsePricingInput('0.08', 'image')).toBe(0.08)
    expect(parsePricingInput('-1', 'token')).toBeUndefined()
    expect(parsePricingInput('abc', 'image')).toBeUndefined()
    expect(parsePricingInput('', 'image')).toBeUndefined()
  })
})
