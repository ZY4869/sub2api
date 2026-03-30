import type { AccountManualModel } from '@/api/admin/accounts'

export interface AccountResolvedUpstreamDraft {
  upstream_url?: string
  upstream_host?: string
  upstream_service?: string
  upstream_probe_source?: string
  upstream_probed_at?: string
  upstream_region?: string
}

export function normalizeAccountManualModels(
  models: AccountManualModel[] | null | undefined,
  allowSourceProtocol: boolean
): AccountManualModel[] {
  if (!Array.isArray(models) || models.length === 0) {
    return []
  }

  const seen = new Set<string>()
  const normalized: AccountManualModel[] = []
  for (const item of models) {
    const modelID = String(item?.model_id || '').trim()
    if (!modelID) {
      continue
    }
    const requestAlias = String(item?.request_alias || '').trim()
    const sourceProtocol = allowSourceProtocol ? normalizeSourceProtocol(item?.source_protocol) : undefined
    const dedupeKey = `${modelID}::${sourceProtocol || ''}`.toLowerCase()
    if (seen.has(dedupeKey)) {
      continue
    }
    seen.add(dedupeKey)
    normalized.push({
      model_id: modelID,
      request_alias: requestAlias || undefined,
      source_protocol: sourceProtocol
    })
  }
  return normalized
}

export function readAccountManualModelsFromExtra(
  extra: Record<string, unknown> | null | undefined,
  allowSourceProtocol: boolean
): AccountManualModel[] {
  const rawItems = extra?.manual_models
  if (!Array.isArray(rawItems)) {
    return []
  }
  return normalizeAccountManualModels(
    rawItems.map((item) => ({
      model_id: String((item as Record<string, unknown>)?.model_id || ''),
      request_alias: String((item as Record<string, unknown>)?.request_alias || ''),
      source_protocol: (item as Record<string, unknown>)?.source_protocol as AccountManualModel['source_protocol']
    })),
    allowSourceProtocol
  )
}

export function mergeAccountManualModelsIntoExtra(
  extra: Record<string, unknown> | null | undefined,
  models: AccountManualModel[] | null | undefined,
  allowSourceProtocol: boolean
): Record<string, unknown> | undefined {
  const nextExtra: Record<string, unknown> = { ...(extra || {}) }
  const normalized = normalizeAccountManualModels(models, allowSourceProtocol)
  if (normalized.length > 0) {
    nextExtra.manual_models = normalized.map((item) => {
      const entry: Record<string, unknown> = {
        model_id: item.model_id
      }
      if (item.request_alias) {
        entry.request_alias = item.request_alias
      }
      if (allowSourceProtocol && item.source_protocol) {
        entry.source_protocol = item.source_protocol
      }
      return entry
    })
  } else {
    delete nextExtra.manual_models
  }
  return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
}

export function readAccountResolvedUpstreamDraft(
  extra: Record<string, unknown> | null | undefined
): AccountResolvedUpstreamDraft | null {
  if (!extra || typeof extra !== 'object') {
    return null
  }
  const draft: AccountResolvedUpstreamDraft = {
    upstream_url: readString(extra.upstream_url),
    upstream_host: readString(extra.upstream_host),
    upstream_service: readString(extra.upstream_service),
    upstream_probe_source: readString(extra.upstream_probe_source),
    upstream_probed_at: readString(extra.upstream_probed_at),
    upstream_region: readString(extra.upstream_region)
  }
  return hasResolvedUpstreamDraft(draft) ? draft : null
}

export function mergeResolvedUpstreamDraftIntoExtra(
  extra: Record<string, unknown> | null | undefined,
  draft: AccountResolvedUpstreamDraft | null | undefined
): Record<string, unknown> | undefined {
  const nextExtra: Record<string, unknown> = { ...(extra || {}) }
  if (draft && hasResolvedUpstreamDraft(draft)) {
    assignIfPresent(nextExtra, 'upstream_url', draft.upstream_url)
    assignIfPresent(nextExtra, 'upstream_host', draft.upstream_host)
    assignIfPresent(nextExtra, 'upstream_service', draft.upstream_service)
    assignIfPresent(nextExtra, 'upstream_probe_source', draft.upstream_probe_source)
    assignIfPresent(nextExtra, 'upstream_probed_at', draft.upstream_probed_at)
    assignIfPresent(nextExtra, 'upstream_region', draft.upstream_region)
  }
  return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
}

export function createResolvedUpstreamDraft(input?: {
  upstream_url?: string
  upstream_host?: string
  upstream_service?: string
  upstream_probe_source?: string
  upstream_probed_at?: string
  upstream_region?: string
} | null): AccountResolvedUpstreamDraft | null {
  if (!input) {
    return null
  }
  const draft: AccountResolvedUpstreamDraft = {
    upstream_url: readString(input.upstream_url),
    upstream_host: readString(input.upstream_host),
    upstream_service: readString(input.upstream_service),
    upstream_probe_source: readString(input.upstream_probe_source),
    upstream_probed_at: readString(input.upstream_probed_at),
    upstream_region: readString(input.upstream_region)
  }
  return hasResolvedUpstreamDraft(draft) ? draft : null
}

function normalizeSourceProtocol(
  value: unknown
): AccountManualModel['source_protocol'] | undefined {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'openai' || normalized === 'anthropic' || normalized === 'gemini') {
    return normalized
  }
  return undefined
}

function readString(value: unknown): string | undefined {
  const normalized = String(value || '').trim()
  return normalized || undefined
}

function assignIfPresent(target: Record<string, unknown>, key: string, value: string | undefined) {
  if (value) {
    target[key] = value
  }
}

function hasResolvedUpstreamDraft(draft: AccountResolvedUpstreamDraft | null | undefined): boolean {
  return Boolean(
    draft?.upstream_url ||
      draft?.upstream_host ||
      draft?.upstream_service ||
      draft?.upstream_probe_source ||
      draft?.upstream_probed_at ||
      draft?.upstream_region
  )
}
