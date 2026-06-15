import type {
  AccountUsageDisplayMode,
  AccountUsagePresentationRow,
} from '@/types'
import {
  resolveAccountUsageWindowKind,
  resolveUsageWindowCapsuleClass,
} from '@/utils/accountUsageWindowDisplay'

export type VisualUsageRow = AccountUsagePresentationRow & {
  shortLabel: string
  displayPercent: number
  usedPercent: number
}

const normalizePercent = (value: number) => Math.max(0, Math.min(value, 100))

const shortLabelForRow = (row: AccountUsagePresentationRow) => {
  const key = row.key.toLowerCase()
  const label = row.label.toLowerCase()
  const explicitLabel = row.label.match(/\b\d+\s*[dhm]\b/i)?.[0]?.replace(/\s+/g, '')
  if (explicitLabel) return explicitLabel
  if (label === '1d') return '1d'
  if (key.includes('5h')) return '5h'
  if (key.includes('7d')) return row.label
  if (key.includes('monthly')) return '30D'
  if (key.includes('daily')) return '1d'
  if (key.includes('weekly')) return '7d'
  return row.label
}

const rankRow = (row: AccountUsagePresentationRow) => {
  const label = shortLabelForRow(row)
  const kind = resolveAccountUsageWindowKind(label)
  if (kind === '5h') return 0
  if (kind === '7d') return 1
  if (kind === 'month' || kind === 'day') return 2
  return 3
}

export const createVisualUsageRows = (
  rows: AccountUsagePresentationRow[],
  displayMode: AccountUsageDisplayMode,
): VisualUsageRow[] => [...rows]
  .sort((first, second) => rankRow(first) - rankRow(second))
  .slice(0, 2)
  .map((row) => {
    const usedPercent = normalizePercent(row.utilization)
    const displayPercent = displayMode === 'remaining'
      ? 100 - usedPercent
      : usedPercent
    return {
      ...row,
      shortLabel: shortLabelForRow(row),
      usedPercent,
      displayPercent: normalizePercent(displayPercent),
    }
  })

export const rowTagClass = (row: VisualUsageRow) => {
  return resolveUsageWindowCapsuleClass(row.shortLabel)
}

export const rowFillClass = (usedPercent: number) => {
  if (usedPercent >= 100) return 'from-rose-500 to-rose-600'
  if (usedPercent > 75) return 'from-orange-400 to-orange-500'
  if (usedPercent > 50) return 'from-yellow-400 to-amber-400'
  if (usedPercent > 25) return 'from-teal-400 to-emerald-500'
  return 'from-emerald-300 to-teal-400'
}

export const rowTextClass = (usedPercent: number) => {
  if (usedPercent >= 100) return 'text-rose-700 dark:text-rose-200'
  if (usedPercent > 75) return 'text-orange-700 dark:text-orange-200'
  if (usedPercent > 50) return 'text-amber-700 dark:text-amber-100'
  if (usedPercent > 25) return 'text-teal-700 dark:text-teal-200'
  return 'text-emerald-700 dark:text-emerald-200'
}
