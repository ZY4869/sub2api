import type { BillingPricingCurrency } from '@/api/admin/billing'
import {
  pricingFieldCurrencySymbol,
  pricingFieldUnitLabel,
  type PricingFieldUnit,
} from './pricingFieldPresentation'

const TOKENS_PER_MILLION = 1_000_000
const DECIMAL_INPUT_PATTERN = /^[+-]?(?:\d+\.?\d*|\.\d+)$/

export const DEFAULT_BILLING_PRICING_CURRENCY: BillingPricingCurrency = 'USD'

export function normalizeBillingPricingCurrency(
  currency?: string | null,
): BillingPricingCurrency {
  return currency === 'CNY' ? 'CNY' : 'USD'
}

export function getBillingPricingCurrencySymbol(
  currency: BillingPricingCurrency,
): string {
  return pricingFieldCurrencySymbol(currency)
}

export function getBillingPricingUnitMultiplier(
  unit: PricingFieldUnit,
): number {
  return unit === 'per_million_tokens' ? TOKENS_PER_MILLION : 1
}

export function resolveBillingPricingExchangeRate(
  currency: BillingPricingCurrency,
  usdToCnyRate?: number | null,
): number | null {
  if (currency === 'USD') {
    return 1
  }
  if (
    typeof usdToCnyRate === 'number'
    && Number.isFinite(usdToCnyRate)
    && usdToCnyRate > 0
  ) {
    return usdToCnyRate
  }
  return null
}

export function convertCanonicalUSDPriceToDisplayValue(options: {
  canonicalUSD?: number | null
  currency: BillingPricingCurrency
  unit: PricingFieldUnit
  usdToCnyRate?: number | null
}): number | undefined {
  const { canonicalUSD, currency, unit, usdToCnyRate } = options
  if (canonicalUSD == null || !Number.isFinite(canonicalUSD)) {
    return undefined
  }

  const exchangeRate = resolveBillingPricingExchangeRate(currency, usdToCnyRate)
  if (exchangeRate == null) {
    return undefined
  }

  return canonicalUSD * getBillingPricingUnitMultiplier(unit) * exchangeRate
}

export function convertDisplayValueToCanonicalUSD(options: {
  displayValue?: number | null
  currency: BillingPricingCurrency
  unit: PricingFieldUnit
  usdToCnyRate?: number | null
}): number | undefined {
  const { displayValue, currency, unit, usdToCnyRate } = options
  if (displayValue == null || !Number.isFinite(displayValue)) {
    return undefined
  }

  const exchangeRate = resolveBillingPricingExchangeRate(currency, usdToCnyRate)
  if (exchangeRate == null) {
    return undefined
  }

  return displayValue / getBillingPricingUnitMultiplier(unit) / exchangeRate
}

export function formatBillingPricingEditableNumber(
  value?: number | null,
): string {
  if (value == null || !Number.isFinite(value)) {
    return ''
  }

  const abs = Math.abs(value)
  let fractionDigits = 4
  if (abs === 0) {
    fractionDigits = 0
  } else if (abs < 0.000001) {
    fractionDigits = 12
  } else if (abs < 0.001) {
    fractionDigits = 9
  } else if (abs < 1) {
    fractionDigits = 6
  } else if (abs < 100) {
    fractionDigits = 4
  } else {
    fractionDigits = 2
  }

  return trimTrailingZeros(value.toFixed(fractionDigits))
}

export function formatBillingPricingValueWithUnit(options: {
  value?: number | null
  currency: BillingPricingCurrency
  unit: PricingFieldUnit
}): string {
  const { value, currency, unit } = options
  if (value == null || !Number.isFinite(value)) {
    return ''
  }

  const symbol = getBillingPricingCurrencySymbol(currency)
  const unitSuffix = pricingFieldUnitLabel(unit, currency).slice(symbol.length)
  return `${symbol}${formatBillingPricingEditableNumber(value)}${unitSuffix}`
}

export function buildBillingPricingAlternateText(options: {
  canonicalUSD?: number | null
  currency: BillingPricingCurrency
  unit: PricingFieldUnit
  usdToCnyRate?: number | null
}): string {
  const { canonicalUSD, currency, unit, usdToCnyRate } = options
  if (canonicalUSD == null || !Number.isFinite(canonicalUSD)) {
    return ''
  }

  const alternateCurrency: BillingPricingCurrency = currency === 'USD' ? 'CNY' : 'USD'
  const displayValue = convertCanonicalUSDPriceToDisplayValue({
    canonicalUSD,
    currency: alternateCurrency,
    unit,
    usdToCnyRate,
  })
  if (displayValue == null) {
    return ''
  }

  return `≈ ${formatBillingPricingValueWithUnit({
    value: displayValue,
    currency: alternateCurrency,
    unit,
  })}`
}

export function parseBillingPricingDecimalInput(raw: string): number | undefined {
  const normalized = raw.trim()
  if (!normalized) {
    return undefined
  }
  if (!DECIMAL_INPUT_PATTERN.test(normalized)) {
    return undefined
  }

  const parsed = Number(normalized)
  return Number.isFinite(parsed) ? parsed : undefined
}

function trimTrailingZeros(value: string): string {
  if (!value.includes('.')) {
    return value
  }
  return value.replace(/\.?0+$/, '')
}
