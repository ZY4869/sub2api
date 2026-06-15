const trimTrailingZeros = (value: string): string =>
  value.replace(/(\.\d*?[1-9])0+$/, '$1').replace(/\.0+$/, '')

export function formatUsdAmount(value: number | null | undefined): string {
  const amount = typeof value === 'number' && Number.isFinite(value) ? value : 0
  const sign = amount < 0 ? '-' : ''
  const abs = Math.abs(amount)
  if (abs === 0) return '$0.00'
  if (abs < 0.01) return `${sign}$${trimTrailingZeros(abs.toFixed(6))}`
  return `${sign}$${abs.toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })}`
}

export function formatDiscountPercent(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0%'
  if (value >= 99.95) return '100%'
  return `${value.toFixed(value >= 10 ? 0 : 1)}%`
}
