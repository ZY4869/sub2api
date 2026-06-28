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

export function resolveUsageResetWindowLabelClass(label: string): string {
  switch (resolveAccountUsageWindowKind(label)) {
    case '5h':
      return 'border-indigo-300 bg-indigo-100 text-indigo-800 ring-indigo-200 dark:border-indigo-300/40 dark:bg-indigo-400/20 dark:text-indigo-50 dark:ring-indigo-300/20'
    case '7d':
      return 'border-orange-300 bg-orange-100 text-orange-800 ring-orange-200 dark:border-orange-300/40 dark:bg-orange-400/20 dark:text-orange-50 dark:ring-orange-300/20'
    case 'month':
      return 'border-green-300 bg-green-100 text-green-800 ring-green-200 dark:border-green-300/40 dark:bg-green-400/20 dark:text-green-50 dark:ring-green-300/20'
    case 'day':
      return 'border-sky-300 bg-sky-100 text-sky-800 ring-sky-200 dark:border-sky-300/40 dark:bg-sky-400/20 dark:text-sky-50 dark:ring-sky-300/20'
    default:
      return 'border-slate-300 bg-slate-100 text-slate-700 ring-slate-200 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 dark:ring-slate-500/20'
  }
}

export function resolveUsageResetCountdownClass(label: string): string {
  switch (resolveAccountUsageWindowKind(label)) {
    case '5h':
      return 'bg-indigo-100 text-indigo-800 ring-indigo-200 dark:bg-indigo-400/20 dark:text-indigo-50 dark:ring-indigo-300/20'
    case '7d':
      return 'bg-orange-100 text-orange-800 ring-orange-200 dark:bg-orange-400/20 dark:text-orange-50 dark:ring-orange-300/20'
    case 'month':
      return 'bg-green-100 text-green-800 ring-green-200 dark:bg-green-400/20 dark:text-green-50 dark:ring-green-300/20'
    case 'day':
      return 'bg-sky-100 text-sky-800 ring-sky-200 dark:bg-sky-400/20 dark:text-sky-50 dark:ring-sky-300/20'
    default:
      return 'bg-slate-100 text-slate-700 ring-slate-200 dark:bg-slate-700 dark:text-slate-100 dark:ring-slate-500/20'
  }
}

export function resolveUsageResetIconClass(label: string): string {
  switch (resolveAccountUsageWindowKind(label)) {
    case '5h':
      return 'text-indigo-500 dark:text-indigo-200'
    case '7d':
      return 'text-orange-500 dark:text-orange-200'
    case 'month':
      return 'text-green-500 dark:text-green-200'
    case 'day':
      return 'text-sky-500 dark:text-sky-200'
    default:
      return 'text-slate-400 dark:text-slate-300'
  }
}
