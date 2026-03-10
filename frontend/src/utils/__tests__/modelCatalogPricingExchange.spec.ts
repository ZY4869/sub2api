import { describe, expect, it } from 'vitest'
import {
  formatModelCatalogCNYReference,
  formatModelCatalogPricePair
} from '@/utils/modelCatalogPricing'

describe('modelCatalogPricing exchange helpers', () => {
  it('formats CNY references when exchange rate exists', () => {
    expect(formatModelCatalogCNYReference(0.0000025, 'token', { rate: 7.2 })).toBe('≈ ¥18.00')
    expect(formatModelCatalogPricePair(0.04, 'image', { rate: 7.2 })).toEqual({
      usd: '$0.040000',
      cny: '≈ ¥0.2880'
    })
  })

  it('skips CNY reference when rate is unavailable', () => {
    expect(formatModelCatalogCNYReference(0.0000025, 'token')).toBeNull()
  })
})
