function toFiniteNumber(value: unknown, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

export function normalizeUsageAmount(value: unknown): number {
  return toFiniteNumber(value, 0)
}

export function normalizeUsageMultiplier(value: unknown, fallback = 1): number {
  return toFiniteNumber(value, fallback)
}

export function hasPositiveUsageAmount(value: unknown): boolean {
  return normalizeUsageAmount(value) > 0
}

export function hasUsageAmount(value: unknown): boolean {
  return typeof value === 'number' && Number.isFinite(value)
}

export function formatUsageAmount(value: unknown, fractionDigits = 6): string {
  return normalizeUsageAmount(value).toFixed(fractionDigits)
}

export function formatUsageMultiplier(value: unknown, fallback = 1, fractionDigits = 2): string {
  return normalizeUsageMultiplier(value, fallback).toFixed(fractionDigits)
}

export function calculateUsageAmount(
  value: unknown,
  multiplier: unknown,
  multiplierFallback = 1
): number {
  return normalizeUsageAmount(value) * normalizeUsageMultiplier(multiplier, multiplierFallback)
}
