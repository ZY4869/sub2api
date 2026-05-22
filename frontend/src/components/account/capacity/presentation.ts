export type CapacityTone = 'idle' | 'active' | 'warning' | 'high' | 'full'

export const MAX_AIRY_CAPACITY_CELLS = 10

export const resolveCapacityPadWidth = (total: number): number =>
  Math.max(2, String(Math.max(0, total)).length)

export const formatCapacityValue = (value: number, width: number): string =>
  String(Math.max(0, value)).padStart(width, '0')

export const resolveCapacityTone = (used: number, total: number): CapacityTone => {
  if (total > 0 && used >= total) {
    return 'full'
  }

  const percent = total > 0 ? (used / total) * 100 : 0
  if (percent >= 75) {
    return 'high'
  }
  if (percent >= 40) {
    return 'warning'
  }
  if (used > 0) {
    return 'active'
  }
  return 'idle'
}

export const resolveDisplayedCapacityCells = (used: number, total: number) => {
  const safeUsed = Math.max(0, used)
  const safeTotal = Math.max(0, total)
  const displayedTotal = safeTotal > 0 ? Math.min(safeTotal, MAX_AIRY_CAPACITY_CELLS) : 0

  if (displayedTotal === 0) {
    return {
      displayedTotal,
      displayedUsed: 0
    }
  }

  if (safeTotal <= MAX_AIRY_CAPACITY_CELLS) {
    return {
      displayedTotal,
      displayedUsed: Math.min(safeUsed, displayedTotal)
    }
  }

  if (safeUsed === 0) {
    return {
      displayedTotal,
      displayedUsed: 0
    }
  }

  if (safeUsed >= safeTotal) {
    return {
      displayedTotal,
      displayedUsed: displayedTotal
    }
  }

  return {
    displayedTotal,
    displayedUsed: Math.max(
      1,
      Math.min(Math.round((safeUsed / safeTotal) * displayedTotal), displayedTotal - 1)
    )
  }
}

export const formatQuotaCurrency = (value: number | null | undefined): string => {
  if (value === null || value === undefined) {
    return '0.00'
  }
  return value.toFixed(2)
}
