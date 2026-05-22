export type AccountGlassTone =
  | 'emerald'
  | 'indigo'
  | 'sky'
  | 'amber'
  | 'orange'
  | 'red'
  | 'slate'

type AccountGlassToneStyles = {
  surfaceClass: string
  iconWrapClass: string
  iconClass: string
  titleClass: string
  statusBadgeClass: string
  helperTextClass: string
  timerBlockClass: string
  timerAccentClass: string
}

const TONE_STYLES: Record<AccountGlassTone, AccountGlassToneStyles> = {
  emerald: {
    surfaceClass:
      'border-emerald-200/75 bg-[linear-gradient(135deg,rgba(236,253,245,0.94),rgba(220,252,231,0.9))] dark:border-emerald-400/15 dark:bg-[linear-gradient(135deg,rgba(6,95,70,0.32),rgba(6,78,59,0.24))]',
    iconWrapClass:
      'border border-emerald-200/80 bg-white text-emerald-600 dark:border-emerald-400/20 dark:bg-emerald-500/10 dark:text-emerald-200',
    iconClass: 'text-emerald-600 dark:text-emerald-200',
    titleClass: 'text-emerald-900 dark:text-emerald-100',
    statusBadgeClass:
      'border-emerald-200/75 bg-emerald-50/78 text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-400/10 dark:text-emerald-100',
    helperTextClass: 'text-emerald-700/80 dark:text-emerald-100/80',
    timerBlockClass:
      'border-emerald-200/80 bg-white text-emerald-800 dark:border-emerald-400/20 dark:bg-emerald-500/10 dark:text-emerald-100',
    timerAccentClass: 'text-emerald-700 dark:text-emerald-200'
  },
  indigo: {
    surfaceClass:
      'border-indigo-200/75 bg-[linear-gradient(135deg,rgba(238,242,255,0.94),rgba(224,231,255,0.88))] dark:border-indigo-400/15 dark:bg-[linear-gradient(135deg,rgba(49,46,129,0.34),rgba(55,48,163,0.24))]',
    iconWrapClass:
      'border border-indigo-200/80 bg-white text-indigo-600 dark:border-indigo-400/20 dark:bg-indigo-400/10 dark:text-indigo-200',
    iconClass: 'text-indigo-600 dark:text-indigo-200',
    titleClass: 'text-indigo-900 dark:text-indigo-100',
    statusBadgeClass:
      'border-indigo-200/75 bg-indigo-50/80 text-indigo-700 dark:border-indigo-400/20 dark:bg-indigo-400/10 dark:text-indigo-100',
    helperTextClass: 'text-indigo-700/80 dark:text-indigo-100/80',
    timerBlockClass:
      'border-indigo-200/80 bg-white text-indigo-800 dark:border-indigo-400/20 dark:bg-indigo-400/10 dark:text-indigo-100',
    timerAccentClass: 'text-indigo-700 dark:text-indigo-200'
  },
  sky: {
    surfaceClass:
      'border-sky-200/75 bg-[linear-gradient(135deg,rgba(240,249,255,0.94),rgba(224,242,254,0.88))] dark:border-sky-400/15 dark:bg-[linear-gradient(135deg,rgba(3,105,161,0.3),rgba(14,116,144,0.22))]',
    iconWrapClass:
      'border border-sky-200/80 bg-white text-sky-600 dark:border-sky-400/20 dark:bg-sky-400/10 dark:text-sky-200',
    iconClass: 'text-sky-600 dark:text-sky-200',
    titleClass: 'text-sky-900 dark:text-sky-100',
    statusBadgeClass:
      'border-sky-200/75 bg-sky-50/80 text-sky-700 dark:border-sky-400/20 dark:bg-sky-400/10 dark:text-sky-100',
    helperTextClass: 'text-sky-700/80 dark:text-sky-100/80',
    timerBlockClass:
      'border-sky-200/80 bg-white text-sky-800 dark:border-sky-400/20 dark:bg-sky-400/10 dark:text-sky-100',
    timerAccentClass: 'text-sky-700 dark:text-sky-200'
  },
  amber: {
    surfaceClass:
      'border-amber-200/75 bg-[linear-gradient(135deg,rgba(255,251,235,0.96),rgba(254,243,199,0.9))] dark:border-amber-400/15 dark:bg-[linear-gradient(135deg,rgba(146,64,14,0.3),rgba(120,53,15,0.22))]',
    iconWrapClass:
      'border border-amber-200/80 bg-white text-amber-600 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-200',
    iconClass: 'text-amber-600 dark:text-amber-200',
    titleClass: 'text-amber-900 dark:text-amber-100',
    statusBadgeClass:
      'border-amber-200/75 bg-amber-50/80 text-amber-700 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-100',
    helperTextClass: 'text-amber-700/85 dark:text-amber-100/80',
    timerBlockClass:
      'border-amber-200/80 bg-white text-amber-800 dark:border-amber-400/20 dark:bg-amber-400/10 dark:text-amber-100',
    timerAccentClass: 'text-amber-700 dark:text-amber-200'
  },
  orange: {
    surfaceClass:
      'border-orange-200/75 bg-[linear-gradient(135deg,rgba(255,247,237,0.96),rgba(254,215,170,0.88))] dark:border-orange-400/15 dark:bg-[linear-gradient(135deg,rgba(154,52,18,0.32),rgba(124,45,18,0.22))]',
    iconWrapClass:
      'border border-orange-200/80 bg-white text-orange-600 dark:border-orange-400/20 dark:bg-orange-400/10 dark:text-orange-200',
    iconClass: 'text-orange-600 dark:text-orange-200',
    titleClass: 'text-orange-900 dark:text-orange-100',
    statusBadgeClass:
      'border-orange-200/75 bg-orange-50/80 text-orange-700 dark:border-orange-400/20 dark:bg-orange-400/10 dark:text-orange-100',
    helperTextClass: 'text-orange-700/85 dark:text-orange-100/80',
    timerBlockClass:
      'border-orange-200/80 bg-white text-orange-800 dark:border-orange-400/20 dark:bg-orange-400/10 dark:text-orange-100',
    timerAccentClass: 'text-orange-700 dark:text-orange-200'
  },
  red: {
    surfaceClass:
      'border-rose-200/75 bg-[linear-gradient(135deg,rgba(255,241,242,0.96),rgba(254,205,211,0.88))] dark:border-rose-400/15 dark:bg-[linear-gradient(135deg,rgba(136,19,55,0.34),rgba(127,29,29,0.24))]',
    iconWrapClass:
      'border border-rose-200/80 bg-white text-rose-600 dark:border-rose-400/20 dark:bg-rose-400/10 dark:text-rose-200',
    iconClass: 'text-rose-600 dark:text-rose-200',
    titleClass: 'text-rose-900 dark:text-rose-100',
    statusBadgeClass:
      'border-rose-200/75 bg-rose-50/80 text-rose-700 dark:border-rose-400/20 dark:bg-rose-400/10 dark:text-rose-100',
    helperTextClass: 'text-rose-700/85 dark:text-rose-100/80',
    timerBlockClass:
      'border-rose-200/80 bg-white text-rose-800 dark:border-rose-400/20 dark:bg-rose-400/10 dark:text-rose-100',
    timerAccentClass: 'text-rose-700 dark:text-rose-200'
  },
  slate: {
    surfaceClass:
      'border-slate-200/75 bg-[linear-gradient(135deg,rgba(248,250,252,0.96),rgba(226,232,240,0.88))] dark:border-slate-400/15 dark:bg-[linear-gradient(135deg,rgba(51,65,85,0.34),rgba(30,41,59,0.24))]',
    iconWrapClass:
      'border border-slate-200/80 bg-white text-slate-600 dark:border-slate-400/20 dark:bg-slate-400/10 dark:text-slate-200',
    iconClass: 'text-slate-600 dark:text-slate-200',
    titleClass: 'text-slate-900 dark:text-slate-100',
    statusBadgeClass:
      'border-slate-200/75 bg-slate-50/80 text-slate-700 dark:border-slate-400/20 dark:bg-slate-400/10 dark:text-slate-100',
    helperTextClass: 'text-slate-600/85 dark:text-slate-100/78',
    timerBlockClass:
      'border-slate-200/80 bg-white text-slate-800 dark:border-slate-400/20 dark:bg-slate-400/10 dark:text-slate-100',
    timerAccentClass: 'text-slate-700 dark:text-slate-200'
  }
}

export const resolveAccountGlassToneStyles = (
  tone: AccountGlassTone,
): AccountGlassToneStyles => TONE_STYLES[tone]
