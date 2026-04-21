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
  entries?: AccountModelProbeSnapshotEntryDraft[]
  updated_at?: string
  source?: string
  probe_source?: string
}

export interface AccountModelProbeSnapshotEntryDraft {
  display_model_id: string
  target_model_id: string
  availability_state?: 'verified' | 'unavailable' | 'unknown'
  stale_state?: 'fresh' | 'stale' | 'unverified'
  updated_at?: string
  source?: string
}

interface LocalAccountModelMappingLike {
  to?: string | null
  from?: string | null
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
    entries: readAccountModelProbeSnapshotEntries(snapshot.entries),
    updated_at: readString(snapshot.updated_at),
    source: readString(snapshot.source),
    probe_source: readString(snapshot.probe_source)
  })
  return normalized
}

export function createAccountModelProbeSnapshotDraft(input?: {
  models?: string[] | null
  entries?: AccountModelProbeSnapshotEntryDraft[] | null
  updated_at?: string
  source?: string
  probe_source?: string
} | null): AccountModelProbeSnapshotDraft | null {
  if (!input) {
    return null
  }

  const models = normalizeModelIds(input.models)
  const entries = normalizeAccountModelProbeSnapshotEntries(input.entries)
  const derivedModels = models.length > 0 ? models : deriveModelsFromSnapshotEntries(entries)
  if (derivedModels.length === 0 && entries.length === 0) {
    return null
  }

  const snapshot: AccountModelProbeSnapshotDraft = {
    models: derivedModels,
    updated_at: readString(input.updated_at),
    source: readString(input.source),
    probe_source: readString(input.probe_source)
  }
  if (entries.length > 0) {
    snapshot.entries = entries
  }
  return snapshot
}

export function buildLocalAccountModelProbeSnapshot(input: {
  current?: AccountModelProbeSnapshotDraft | null
  enabled?: boolean
  modelRestrictionMode?: 'whitelist' | 'mapping'
  allowedModels?: string[] | null
  modelMappings?: LocalAccountModelMappingLike[] | null
  source?: string
}): AccountModelProbeSnapshotDraft | null {
  const currentSnapshot = createAccountModelProbeSnapshotDraft(input.current)
  const entries = buildPolicyPreviewSnapshotEntries({
    enabled: input.enabled !== false,
    mode: input.modelRestrictionMode,
    allowedModels: input.allowedModels,
    modelMappings: input.modelMappings
  })
  if (entries.length === 0) {
    if (
      currentSnapshot &&
      (currentSnapshot.source || '').trim() &&
      (currentSnapshot.source || '').trim() !== 'model_scope_preview'
    ) {
      return currentSnapshot
    }
    return null
  }
  const orderedModels = deriveModelsFromSnapshotEntries(entries)
  if (
    currentSnapshot &&
    areModelIdListsEqual(currentSnapshot.models, orderedModels) &&
    areSnapshotEntriesEqual(currentSnapshot.entries || [], entries)
  ) {
    return currentSnapshot
  }
  const source = String(input.source || 'model_scope_preview').trim() || 'model_scope_preview'
  return createAccountModelProbeSnapshotDraft({
    models: orderedModels,
    entries,
    updated_at: new Date().toISOString(),
    source,
    probe_source: source
  })
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
    if (Array.isArray(normalizedSnapshot.entries) && normalizedSnapshot.entries.length > 0) {
      nextSnapshot.entries = normalizedSnapshot.entries.map((entry) => ({ ...entry }))
    }
    assignIfPresent(nextSnapshot, 'updated_at', normalizedSnapshot.updated_at)
    assignIfPresent(nextSnapshot, 'source', normalizedSnapshot.source)
    assignIfPresent(nextSnapshot, 'probe_source', normalizedSnapshot.probe_source)
    nextExtra.model_probe_snapshot = nextSnapshot
  } else {
    delete nextExtra.model_probe_snapshot
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
    const entries = readAccountModelProbeSnapshotEntries((scopeMap as Record<string, unknown>).entries)
    if (entries.length > 0) {
      for (const entry of entries) {
        appendModel(entry.target_model_id || entry.display_model_id)
      }
    }

    const selectedModelIDs = readStringArray(scopeMap.selected_model_ids)
    if (selectedModelIDs.length > 0) {
      for (const modelID of selectedModelIDs) {
        appendModel(modelID)
      }
    } else {
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

function areModelIdListsEqual(left: string[], right: string[]): boolean {
  if (left.length !== right.length) {
    return false
  }
  return left.every((value, index) => value === right[index])
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
    areSnapshotEntriesEqual(a.entries || [], b.entries || []) &&
    a.updated_at === b.updated_at &&
    a.source === b.source &&
    a.probe_source === b.probe_source
  )
}

function readAccountModelProbeSnapshotEntries(value: unknown): AccountModelProbeSnapshotEntryDraft[] {
  if (!Array.isArray(value)) {
    return []
  }
  return normalizeAccountModelProbeSnapshotEntries(
    value.map((item) => {
      const entry = item as Record<string, unknown>
      return {
        display_model_id: String(entry.display_model_id || '').trim(),
        target_model_id: String(entry.target_model_id || '').trim(),
        availability_state: normalizeAvailabilityState(entry.availability_state),
        stale_state: normalizeStaleState(entry.stale_state),
        updated_at: readString(entry.updated_at),
        source: readString(entry.source)
      }
    })
  )
}

function normalizeAccountModelProbeSnapshotEntries(
  entries: AccountModelProbeSnapshotEntryDraft[] | null | undefined
): AccountModelProbeSnapshotEntryDraft[] {
  if (!Array.isArray(entries) || entries.length === 0) {
    return []
  }
  const normalized: AccountModelProbeSnapshotEntryDraft[] = []
  const seen = new Set<string>()
  for (const entry of entries) {
    const displayModelID = String(entry?.display_model_id || '').trim()
    const targetModelID = String(entry?.target_model_id || displayModelID).trim()
    if (!displayModelID && !targetModelID) {
      continue
    }
    const dedupeKey = `${displayModelID}::${targetModelID}`.toLowerCase()
    if (seen.has(dedupeKey)) {
      continue
    }
    seen.add(dedupeKey)
    normalized.push({
      display_model_id: displayModelID || targetModelID,
      target_model_id: targetModelID || displayModelID,
      availability_state: normalizeAvailabilityState(entry?.availability_state),
      stale_state: normalizeStaleState(entry?.stale_state),
      updated_at: readString(entry?.updated_at),
      source: readString(entry?.source)
    })
  }
  return normalized
}

function deriveModelsFromSnapshotEntries(entries: AccountModelProbeSnapshotEntryDraft[]): string[] {
  return normalizeModelIds(entries.map((entry) => entry.target_model_id || entry.display_model_id))
}

function buildPolicyPreviewSnapshotEntries(input: {
  enabled: boolean
  mode?: 'whitelist' | 'mapping'
  allowedModels?: string[] | null
  modelMappings?: LocalAccountModelMappingLike[] | null
}): AccountModelProbeSnapshotEntryDraft[] {
  if (!input.enabled) {
    return []
  }

  const updatedAt = new Date().toISOString()
  if (input.mode === 'mapping') {
    const normalizedTargets = normalizeModelIds([
      ...(input.allowedModels || []),
      ...((input.modelMappings || []).map((item) => String(item?.to || '').trim()))
    ])
    const explicitMappings = (input.modelMappings || [])
      .map((item) => ({
        from: String(item?.from || '').trim(),
        to: String(item?.to || '').trim()
      }))
      .filter((item) => item.from && item.to)

    return normalizeAccountModelProbeSnapshotEntries(
      normalizedTargets.flatMap((targetModelID) => {
        const aliases = explicitMappings.filter(
          (item) => item.to === targetModelID && item.from !== targetModelID
        )
        if (aliases.length === 0) {
          return [
            createPreviewSnapshotEntry(targetModelID, targetModelID, updatedAt)
          ]
        }
        return aliases.map((alias) =>
          createPreviewSnapshotEntry(alias.from, targetModelID, updatedAt)
        )
      })
    )
  }

  return normalizeAccountModelProbeSnapshotEntries(
    normalizeModelIds(input.allowedModels || []).map((modelID) =>
      createPreviewSnapshotEntry(modelID, modelID, updatedAt)
    )
  )
}

function createPreviewSnapshotEntry(
  displayModelID: string,
  targetModelID: string,
  updatedAt: string
): AccountModelProbeSnapshotEntryDraft {
  return {
    display_model_id: displayModelID,
    target_model_id: targetModelID,
    availability_state: 'unknown',
    stale_state: 'unverified',
    updated_at: updatedAt,
    source: 'model_scope_preview'
  }
}

function normalizeAvailabilityState(
  value: unknown
): AccountModelProbeSnapshotEntryDraft['availability_state'] | undefined {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'verified' || normalized === 'unavailable' || normalized === 'unknown') {
    return normalized
  }
  return undefined
}

function normalizeStaleState(
  value: unknown
): AccountModelProbeSnapshotEntryDraft['stale_state'] | undefined {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'fresh' || normalized === 'stale' || normalized === 'unverified') {
    return normalized
  }
  return undefined
}

function areSnapshotEntriesEqual(
  left: AccountModelProbeSnapshotEntryDraft[],
  right: AccountModelProbeSnapshotEntryDraft[]
): boolean {
  if (left.length !== right.length) {
    return false
  }
  return left.every((entry, index) => {
    const other = right[index]
    return (
      entry.display_model_id === other.display_model_id &&
      entry.target_model_id === other.target_model_id &&
      entry.availability_state === other.availability_state &&
      entry.stale_state === other.stale_state &&
      entry.updated_at === other.updated_at &&
      entry.source === other.source
    )
  })
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
