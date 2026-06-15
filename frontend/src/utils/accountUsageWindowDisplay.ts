import type { AccountUsageRowColor } from '@/types'

export type AccountUsageWindowKind = '5h' | '7d' | 'month' | 'day' | 'other'

const normalizeUsageWindowLabel = (label: string): string =>
  String(label || '').trim().toUpperCase().replace(/\s+/g, '')

export function resolveAccountUsageWindowKind(label: string): AccountUsageWindowKind {
  const normalized = normalizeUsageWindowLabel(label)
  if (/\b5H\b/.test(normalized) || normalized.includes('5H')) return '5h'
  if (/\b(?:30|31)D\b/.test(normalized) || normalized.includes('30D') || normalized.includes('31D')) return 'month'
  if (/\b7D\b/.test(normalized) || normalized.includes('7D')) return '7d'
  if (/^\d+D$/.test(normalized) || normalized.includes('1D')) return 'day'
  return 'other'
}

export function resolveUsageWindowColorByLabel(label: string): AccountUsageRowColor {
  switch (resolveAccountUsageWindowKind(label)) {
    case '5h':
      return 'indigo'
    case '7d':
      return 'orange'
    case 'month':
      return 'green'
    default:
      return 'emerald'
  }
}

export function resolveUsageWindowCapsuleClass(label: string): string {
  switch (resolveAccountUsageWindowKind(label)) {
    case '5h':
      return 'border-indigo-200 bg-indigo-50 text-indigo-700 dark:border-indigo-400/25 dark:bg-indigo-400/10 dark:text-indigo-100'
    case '7d':
      return 'border-orange-200 bg-orange-50 text-orange-700 dark:border-orange-400/25 dark:bg-orange-400/10 dark:text-orange-100'
    case 'month':
      return 'border-green-200 bg-green-50 text-green-700 dark:border-green-400/25 dark:bg-green-400/10 dark:text-green-100'
    case 'day':
      return 'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-400/25 dark:bg-sky-400/10 dark:text-sky-100'
    default:
      return 'border-slate-200 bg-slate-50 text-slate-600 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200'
  }
}
