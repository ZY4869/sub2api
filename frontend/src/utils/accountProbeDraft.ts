import type { AccountManualModel } from '@/api/admin/accounts'
import { normalizeProviderSlug } from '@/utils/providerLabels'

export interface AccountResolvedUpstreamDraft {
  upstream_url?: string
  upstream_host?: string
  upstream_service?: string
  upstream_probe_source?: string
  upstream_probed_at?: string
  upstream_region?: string
}

export interface AccountModelProbeSnapshotDraft {
  models: string[]
  updated_at?: string
  source?: string
  probe_source?: string
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
    const provider = normalizeProvider(item?.provider)
    const sourceProtocol = allowSourceProtocol ? normalizeSourceProtocol(item?.source_protocol) : undefined
    const dedupeKey = `${modelID}::${sourceProtocol || ''}`.toLowerCase()
    if (seen.has(dedupeKey)) {
      continue
    }
    seen.add(dedupeKey)
    normalized.push({
      model_id: modelID,
      request_alias: requestAlias || undefined,
      provider,
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
      provider: (item as Record<string, unknown>)?.provider as AccountManualModel['provider'],
      source_protocol: (item as Record<string, unknown>)?.source_protocol as AccountManualModel['source_protocol']
    })),
    allowSourceProtocol
  )
}

export function readAccountModelProbeSnapshot(
  extra: Record<string, unknown> | null | undefined
): AccountModelProbeSnapshotDraft | null {
  const rawSnapshot = extra?.model_probe_snapshot
  if (!rawSnapshot || typeof rawSnapshot !== 'object') {
    return null
  }

  const snapshot = rawSnapshot as Record<string, unknown>
  const normalized = createAccountModelProbeSnapshotDraft({
    models: readStringArray(snapshot.models),
    updated_at: readString(snapshot.updated_at),
    source: readString(snapshot.source),
    probe_source: readString(snapshot.probe_source)
  })
  return normalized
}

export function createAccountModelProbeSnapshotDraft(input?: {
  models?: string[] | null
  updated_at?: string
  source?: string
  probe_source?: string
} | null): AccountModelProbeSnapshotDraft | null {
  if (!input) {
    return null
  }

  const models = normalizeModelIds(input.models)
  if (models.length === 0) {
    return null
  }

  return {
    models,
    updated_at: readString(input.updated_at),
    source: readString(input.source),
    probe_source: readString(input.probe_source)
  }
}

export function mergeAccountModelProbeSnapshotIntoExtra(
  extra: Record<string, unknown> | null | undefined,
  snapshot: AccountModelProbeSnapshotDraft | null | undefined
): Record<string, unknown> | undefined {
  const nextExtra: Record<string, unknown> = { ...(extra || {}) }
  const normalizedSnapshot = createAccountModelProbeSnapshotDraft(snapshot)
  if (normalizedSnapshot) {
    const nextSnapshot: Record<string, unknown> = {
      models: [...normalizedSnapshot.models]
    }
    assignIfPresent(nextSnapshot, 'updated_at', normalizedSnapshot.updated_at)
    assignIfPresent(nextSnapshot, 'source', normalizedSnapshot.source)
    assignIfPresent(nextSnapshot, 'probe_source', normalizedSnapshot.probe_source)
    nextExtra.model_probe_snapshot = nextSnapshot
  }
  return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
}

export function deriveConfiguredAccountModelIds(
  extra: Record<string, unknown> | null | undefined,
  credentials: Record<string, unknown> | null | undefined
): string[] {
  const ordered: string[] = []
  const seen = new Set<string>()
  const appendModel = (value: unknown) => {
    const modelID = String(value || '').trim()
    if (!modelID) {
      return
    }
    const dedupeKey = modelID.toLowerCase()
    if (seen.has(dedupeKey)) {
      return
    }
    seen.add(dedupeKey)
    ordered.push(modelID)
  }

  for (const model of readAccountManualModelsFromExtra(extra, true)) {
    appendModel(model.model_id)
  }

  const scope = extra?.model_scope_v2
  if (scope && typeof scope === 'object') {
    const scopeMap = scope as Record<string, unknown>
    const supportedModelsByProvider = scopeMap.supported_models_by_provider
    if (supportedModelsByProvider && typeof supportedModelsByProvider === 'object') {
      const providerNames = Object.keys(supportedModelsByProvider as Record<string, unknown>).sort()
      for (const provider of providerNames) {
        const models = (supportedModelsByProvider as Record<string, unknown>)[provider]
        for (const modelID of readStringArray(models)) {
          appendModel(modelID)
        }
      }
    }

    const manualMappingRows = Array.isArray(scopeMap.manual_mapping_rows)
      ? scopeMap.manual_mapping_rows
      : []
    for (const row of manualMappingRows) {
      appendModel((row as Record<string, unknown>)?.to)
    }

    const manualMappings = scopeMap.manual_mappings
    if (manualMappings && typeof manualMappings === 'object') {
      const fromKeys = Object.keys(manualMappings as Record<string, unknown>).sort()
      for (const from of fromKeys) {
        appendModel((manualMappings as Record<string, unknown>)[from])
      }
    }
  }

  const modelMapping = credentials?.model_mapping
  if (modelMapping && typeof modelMapping === 'object' && !Array.isArray(modelMapping)) {
    const aliases = Object.keys(modelMapping as Record<string, unknown>).sort()
    for (const alias of aliases) {
      appendModel((modelMapping as Record<string, unknown>)[alias])
    }
  }

  for (const modelID of readStringArray(extra?.openai_known_models)) {
    appendModel(modelID)
  }

  return ordered
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
      if (item.provider) {
        entry.provider = item.provider
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

function normalizeProvider(value: unknown): string | undefined {
  const normalized = normalizeProviderSlug(String(value || ''))
  return normalized || undefined
}

function readString(value: unknown): string | undefined {
  const normalized = String(value || '').trim()
  return normalized || undefined
}

function readStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return []
  }
  return normalizeModelIds(value)
}

function normalizeModelIds(values: unknown[] | string[] | null | undefined): string[] {
  if (!Array.isArray(values) || values.length === 0) {
    return []
  }

  const normalized: string[] = []
  const seen = new Set<string>()
  for (const value of values) {
    const modelID = String(value || '').trim()
    if (!modelID) {
      continue
    }
    const dedupeKey = modelID.toLowerCase()
    if (seen.has(dedupeKey)) {
      continue
    }
    seen.add(dedupeKey)
    normalized.push(modelID)
  }
  return normalized
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

function arraysEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false
  }
  return true
}

export function isProbeSnapshotEqual(
  a: AccountModelProbeSnapshotDraft | null | undefined,
  b: AccountModelProbeSnapshotDraft | null | undefined
): boolean {
  if (a === b) return true
  if (!a || !b) return false
  return (
    arraysEqual(a.models, b.models) &&
    a.updated_at === b.updated_at &&
    a.source === b.source &&
    a.probe_source === b.probe_source
  )
}

export function isUpstreamDraftEqual(
  a: AccountResolvedUpstreamDraft | null | undefined,
  b: AccountResolvedUpstreamDraft | null | undefined
): boolean {
  if (a === b) return true
  if (!a || !b) return false
  return (
    a.upstream_url === b.upstream_url &&
    a.upstream_host === b.upstream_host &&
    a.upstream_service === b.upstream_service &&
    a.upstream_probe_source === b.upstream_probe_source &&
    a.upstream_probed_at === b.upstream_probed_at &&
    a.upstream_region === b.upstream_region
  )
}
