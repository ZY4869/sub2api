import { describe, expect, it } from 'vitest'
import {
  buildBillingPricingAlternateText,
  convertDisplayValueToSourcePrice,
  convertSourcePriceToDisplayValue,
  formatBillingPricingEditableNumber,
} from '../pricingCurrency'

describe('pricingCurrency', () => {
  it('converts token source price into editable usd per million tokens', () => {
    const displayValue = convertSourcePriceToDisplayValue({
      sourcePrice: 2.8e-7,
      currency: 'USD',
      unit: 'per_million_tokens',
    })

    expect(displayValue).toBe(0.28)
    expect(formatBillingPricingEditableNumber(displayValue)).toBe('0.28')
  })

  it('keeps cny source values in cny instead of saving converted usd values', () => {
    expect(convertSourcePriceToDisplayValue({
      sourcePrice: 2.016e-6,
      currency: 'CNY',
      unit: 'per_million_tokens',
      usdToCnyRate: 7.2,
    })).toBeCloseTo(2.016)

    expect(convertSourcePriceToDisplayValue({
      sourcePrice: 0.864,
      currency: 'CNY',
      unit: 'per_request',
      usdToCnyRate: 7.2,
    })).toBeCloseTo(0.864)

    expect(convertSourcePriceToDisplayValue({
      sourcePrice: 0.576,
      currency: 'CNY',
      unit: 'per_image',
      usdToCnyRate: 7.2,
    })).toBeCloseTo(0.576)

    expect(convertDisplayValueToSourcePrice({
      displayValue: 2.016,
      currency: 'CNY',
      unit: 'per_million_tokens',
      usdToCnyRate: 7.2,
    })).toBeCloseTo(2.016e-6)
  })

  it('formats editable numbers without scientific notation and builds alternate currency text', () => {
    expect(formatBillingPricingEditableNumber(2.8e-7)).toBe('0.00000028')
    expect(formatBillingPricingEditableNumber(2.8e-7)).not.toContain('e')

    expect(buildBillingPricingAlternateText({
      sourcePrice: 2.8e-7,
      currency: 'USD',
      unit: 'per_million_tokens',
      usdToCnyRate: 7.2,
    })).toBe('≈ ￥2.016 / M Tokens')

    expect(buildBillingPricingAlternateText({
      sourcePrice: 2.016e-6,
      currency: 'CNY',
      unit: 'per_million_tokens',
      usdToCnyRate: 7.2,
    })).toBe('≈ $0.28 / M Tokens')
  })
})
