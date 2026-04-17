import type { DocsPageId } from '@/utils/markdownDocs'

export interface DocsTheme {
  badgeClass: string
  navActiveClass: string
  tocActiveClass: string
  tabActiveClass: string
  glowClass: string
}

const DOCS_THEMES: Record<DocsPageId, DocsTheme> = {
  common: {
    badgeClass: 'bg-sky-100 text-sky-700 dark:bg-sky-500/15 dark:text-sky-200',
    navActiveClass: 'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200',
    tocActiveClass: 'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200',
    tabActiveClass: 'bg-sky-600 text-white shadow-sm dark:bg-sky-500',
    glowClass: 'from-sky-500/20 via-cyan-500/10 to-transparent'
  },
  openai: {
    badgeClass: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200',
    navActiveClass: 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200',
    tocActiveClass: 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200',
    tabActiveClass: 'bg-emerald-600 text-white shadow-sm dark:bg-emerald-500',
    glowClass: 'from-emerald-500/20 via-lime-500/10 to-transparent'
  },
  anthropic: {
    badgeClass: 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-200',
    navActiveClass: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200',
    tocActiveClass: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200',
    tabActiveClass: 'bg-amber-600 text-white shadow-sm dark:bg-amber-500',
    glowClass: 'from-amber-500/20 via-orange-500/10 to-transparent'
  },
  gemini: {
    badgeClass: 'bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-200',
    navActiveClass: 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-500/30 dark:bg-blue-500/10 dark:text-blue-200',
    tocActiveClass: 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-500/30 dark:bg-blue-500/10 dark:text-blue-200',
    tabActiveClass: 'bg-blue-600 text-white shadow-sm dark:bg-blue-500',
    glowClass: 'from-blue-500/20 via-cyan-500/10 to-transparent'
  },
  grok: {
    badgeClass: 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-200',
    navActiveClass: 'border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-200',
    tocActiveClass: 'border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-200',
    tabActiveClass: 'bg-rose-600 text-white shadow-sm dark:bg-rose-500',
    glowClass: 'from-rose-500/20 via-pink-500/10 to-transparent'
  },
  antigravity: {
    badgeClass: 'bg-teal-100 text-teal-700 dark:bg-teal-500/15 dark:text-teal-200',
    navActiveClass: 'border-teal-200 bg-teal-50 text-teal-700 dark:border-teal-500/30 dark:bg-teal-500/10 dark:text-teal-200',
    tocActiveClass: 'border-teal-200 bg-teal-50 text-teal-700 dark:border-teal-500/30 dark:bg-teal-500/10 dark:text-teal-200',
    tabActiveClass: 'bg-teal-600 text-white shadow-sm dark:bg-teal-500',
    glowClass: 'from-teal-500/20 via-cyan-500/10 to-transparent'
  },
  'vertex-batch': {
    badgeClass: 'bg-slate-200 text-slate-700 dark:bg-slate-500/15 dark:text-slate-200',
    navActiveClass: 'border-slate-300 bg-slate-100 text-slate-700 dark:border-slate-500/30 dark:bg-slate-500/10 dark:text-slate-200',
    tocActiveClass: 'border-slate-300 bg-slate-100 text-slate-700 dark:border-slate-500/30 dark:bg-slate-500/10 dark:text-slate-200',
    tabActiveClass: 'bg-slate-700 text-white shadow-sm dark:bg-slate-500',
    glowClass: 'from-slate-500/20 via-slate-400/10 to-transparent'
  }
}

export function getDocsTheme(pageId: DocsPageId): DocsTheme {
  return DOCS_THEMES[pageId]
}
