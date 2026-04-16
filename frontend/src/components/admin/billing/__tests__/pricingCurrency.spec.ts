import { describe, expect, it } from 'vitest'
import {
  buildBillingPricingAlternateText,
  convertCanonicalUSDPriceToDisplayValue,
  convertDisplayValueToCanonicalUSD,
  formatBillingPricingEditableNumber,
} from '../pricingCurrency'

describe('pricingCurrency', () => {
  it('converts token canonical usd price into editable usd per million tokens', () => {
    const displayValue = convertCanonicalUSDPriceToDisplayValue({
      canonicalUSD: 2.8e-7,
      currency: 'USD',
      unit: 'per_million_tokens',
    })

    expect(displayValue).toBe(0.28)
    expect(formatBillingPricingEditableNumber(displayValue)).toBe('0.28')
  })

  it('converts usd canonical values to cny display values and back across unit types', () => {
    expect(convertCanonicalUSDPriceToDisplayValue({
      canonicalUSD: 2.8e-7,
      currency: 'CNY',
      unit: 'per_million_tokens',
      usdToCnyRate: 7.2,
    })).toBeCloseTo(2.016)

    expect(convertCanonicalUSDPriceToDisplayValue({
      canonicalUSD: 0.12,
      currency: 'CNY',
      unit: 'per_request',
      usdToCnyRate: 7.2,
    })).toBeCloseTo(0.864)

    expect(convertCanonicalUSDPriceToDisplayValue({
      canonicalUSD: 0.08,
      currency: 'CNY',
      unit: 'per_image',
      usdToCnyRate: 7.2,
    })).toBeCloseTo(0.576)

    expect(convertDisplayValueToCanonicalUSD({
      displayValue: 2.016,
      currency: 'CNY',
      unit: 'per_million_tokens',
      usdToCnyRate: 7.2,
    })).toBeCloseTo(2.8e-7)
  })

  it('formats editable numbers without scientific notation and builds alternate currency text', () => {
    expect(formatBillingPricingEditableNumber(2.8e-7)).toBe('0.00000028')
    expect(formatBillingPricingEditableNumber(2.8e-7)).not.toContain('e')

    expect(buildBillingPricingAlternateText({
      canonicalUSD: 2.8e-7,
      currency: 'USD',
      unit: 'per_million_tokens',
      usdToCnyRate: 7.2,
    })).toBe('≈ ￥2.016 / M Tokens')
  })
})
