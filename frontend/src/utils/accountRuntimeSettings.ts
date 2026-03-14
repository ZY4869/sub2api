export function normalizeAccountConcurrency(value: number | null | undefined): number {
  const normalized = Number(value)
  if (!Number.isFinite(normalized) || normalized < 1) {
    return 1
  }
  return normalized
}

export function normalizeAccountLoadFactor(value: number | null | undefined): number | null {
  const normalized = Number(value)
  if (!Number.isFinite(normalized) || normalized < 1) {
    return null
  }
  return normalized
}
