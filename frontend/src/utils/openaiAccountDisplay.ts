import type { Account } from '@/types'
import { readAccountModelProbeSnapshot } from '@/utils/accountProbeDraft'

interface OpenAIModelLabels {
  notProbed: string
  customMapping: string
}

const PLAN_LABELS: Record<string, string> = {
  plus: 'Plus',
  team: 'Team',
  pro: 'Pro',
  chatgptpro: 'Pro',
  free: 'Free'
}

function asString(value: unknown): string | null {
  if (typeof value !== 'string') return null
  const trimmed = value.trim()
  return trimmed === '' ? null : trimmed
}

function summarizeItems(items: string[]): string {
  if (items.length <= 3) return items.join(', ')
  return `${items.slice(0, 3).join(', ')} +${items.length - 3}`
}

export function normalizePlanTypeLabel(raw: unknown): string | null {
  const value = asString(raw)
  if (!value) return null
  const normalized = value.toLowerCase()
  return PLAN_LABELS[normalized] ?? value
}

export function shortenOpaqueId(raw: unknown): string | null {
  const value = asString(raw)
  if (!value) return null
  if (value.length <= 12) return value
  return `${value.slice(0, 6)}...${value.slice(-4)}`
}

export function getOpenAIIdentitySummary(account: Pick<Account, 'credentials'>): string {
  const credentials = account.credentials ?? {}
  const parts = [
    normalizePlanTypeLabel(credentials.plan_type),
    shortenOpaqueId(credentials.chatgpt_account_id)
  ].filter((value): value is string => Boolean(value))

  return parts.length > 0 ? parts.join(' · ') : 'OAuth'
}

export function getOpenAIModelSummary(
  account: Pick<Account, 'credentials' | 'extra'>,
  labels: OpenAIModelLabels
): string {
  const probeSnapshot = readAccountModelProbeSnapshot(account.extra)
  if (probeSnapshot?.models.length) {
    return summarizeItems(probeSnapshot.models)
  }

  const knownModels = Array.isArray(account.extra?.openai_known_models)
    ? account.extra?.openai_known_models.filter((model): model is string => typeof model === 'string' && model.trim() !== '')
    : []
  if (knownModels.length > 0) {
    return summarizeItems(knownModels)
  }

  const rawMapping = account.credentials?.model_mapping
  if (rawMapping && typeof rawMapping === 'object' && !Array.isArray(rawMapping)) {
    const mappingKeys = Object.keys(rawMapping as Record<string, unknown>).filter((key) => key.trim() !== '')
    const explicitKeys = mappingKeys.filter((key) => key !== '*')
    if (explicitKeys.length > 0) {
      return summarizeItems(explicitKeys)
    }
    if (mappingKeys.includes('*')) {
      return labels.customMapping
    }
  }

  return labels.notProbed
}
