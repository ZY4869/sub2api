import type { AiryStatusKind } from './accountAiryStatusTypes'

export type AccountAiryIssueSummary = {
  helper?: string
  helperKey?: string
}

type IssueSummaryOptions = {
  defaultWhenEmpty?: boolean
}

const summaryKeyByKind: Partial<Record<AiryStatusKind, string>> = {
  banned: 'admin.accounts.status.issueSummaries.banned',
  locked: 'admin.accounts.status.issueSummaries.locked',
  maintenance: 'admin.accounts.status.issueSummaries.maintenance',
  offline: 'admin.accounts.status.issueSummaries.offline',
  overdue: 'admin.accounts.status.issueSummaries.overdue',
  degraded: 'admin.accounts.status.issueSummaries.degraded',
  captcha: 'admin.accounts.status.issueSummaries.captcha',
  syncing: 'admin.accounts.status.issueSummaries.syncing',
  error: 'admin.accounts.status.issueSummaries.error',
  tempUnschedulable: 'admin.accounts.status.issueSummaries.tempUnschedulable',
}

const patternSummaries = [
  {
    patterns: ['credential', 'oauth', 'unauthorized', '401', 'reauth'],
    key: 'admin.accounts.status.issueSummaries.credentials',
  },
  {
    patterns: ['blacklist', 'banned', 'ban', 'policy review', 'policy'],
    key: 'admin.accounts.status.issueSummaries.banned',
  },
  {
    patterns: ['locked', 'security', 'suspicious', 'login'],
    key: 'admin.accounts.status.issueSummaries.locked',
  },
  {
    patterns: ['maintenance', 'maintain'],
    key: 'admin.accounts.status.issueSummaries.maintenance',
  },
  {
    patterns: ['timeout', 'timed out', 'network', 'proxy', 'connect', 'connection refused', 'dial tcp'],
    key: 'admin.accounts.status.issueSummaries.offline',
  },
  {
    patterns: ['quota', 'credit', 'billing', 'payment', 'insufficient', 'exhausted', 'overdue', '402'],
    key: 'admin.accounts.status.issueSummaries.overdue',
  },
  {
    patterns: ['captcha', 'challenge', 'cloudflare', 'turnstile', 'verify', 'verification'],
    key: 'admin.accounts.status.issueSummaries.captcha',
  },
  {
    patterns: ['retry', 'sync', 'config refresh'],
    key: 'admin.accounts.status.issueSummaries.syncing',
  },
  {
    patterns: ['limited model', 'degraded', 'partial'],
    key: 'admin.accounts.status.issueSummaries.degraded',
  },
] as const

const hasChineseText = (value: string) => /[\u3400-\u9fff]/.test(value)

const firstText = (values: Array<unknown>) => {
  for (const value of values) {
    const text = String(value || '').trim()
    if (text) return text
  }
  return ''
}

export const resolveAccountAiryIssueSummary = (
  kind: AiryStatusKind,
  values: Array<unknown>,
  options: IssueSummaryOptions = {},
): AccountAiryIssueSummary => {
  const rawText = firstText(values)
  if (rawText && hasChineseText(rawText)) return { helper: rawText }

  const normalized = rawText.toLowerCase()
  const matched = patternSummaries.find((item) =>
    item.patterns.some((pattern) => normalized.includes(pattern)),
  )
  if (matched) return { helperKey: matched.key }

  if (rawText || options.defaultWhenEmpty) {
    return { helperKey: summaryKeyByKind[kind] || 'admin.accounts.status.issueSummaries.error' }
  }

  return {}
}
