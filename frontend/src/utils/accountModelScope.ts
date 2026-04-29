import { getModelRegistrySnapshot } from '@/stores/modelRegistry'

export interface AccountModelScopeMappingRow {
  from: string
  to: string
}

export interface BuildAccountModelScopeOptions {
  platform: string
  enabled: boolean
  mode: 'whitelist' | 'mapping'
  allowedModels: string[]
  modelMappings: AccountModelScopeMappingRow[]
}

export interface LoadedAccountModelScopeDraft {
  enabled: boolean
  mode: 'whitelist' | 'mapping'
  allowedModels: string[]
  modelMappings: AccountModelScopeMappingRow[]
}

export function buildAccountModelScopeExtra(
  baseExtra: Record<string, unknown> | undefined,
  options: BuildAccountModelScopeOptions
): Record<string, unknown> | undefined {
  const nextExtra = { ...(baseExtra || {}) }
  if (!options.enabled) {
    delete nextExtra.model_scope_v2
    return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
  }

  const snapshot = getModelRegistrySnapshot()
  const normalizedAllowedModels = uniqueStrings(options.allowedModels)
  const normalizedMappings = uniqueMappingRows(options.modelMappings)
  const entries =
    options.mode === 'mapping'
      ? buildMappingEntries(snapshot.models, options.platform, normalizedAllowedModels, normalizedMappings)
      : normalizedAllowedModels.map((modelID) =>
          buildScopeEntry(snapshot.models, options.platform, modelID, modelID)
        )

  if (entries.length === 0) {
    delete nextExtra.model_scope_v2
    return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
  }

  nextExtra.model_scope_v2 = {
    policy_mode: options.mode === 'mapping' ? 'mapping' : 'whitelist',
    entries
  }
  return nextExtra
}

export function loadAccountModelScopeDraft(
  extra: Record<string, unknown> | undefined | null
): LoadedAccountModelScopeDraft | null {
  const raw = extra?.model_scope_v2
  if (!raw || typeof raw !== 'object') {
    return null
  }

  const scope = raw as Record<string, unknown>
  const policyMode = String(scope.policy_mode || '').trim().toLowerCase() === 'mapping'
    ? 'mapping'
    : 'whitelist'
  const rawEntries = Array.isArray(scope.entries) ? scope.entries : []
  const entries = rawEntries
    .map((item) => {
      const entry = item as Record<string, unknown>
      const displayModelID = String(entry.display_model_id || '').trim()
      const targetModelID = String(entry.target_model_id || displayModelID).trim()
      return {
        displayModelID,
        targetModelID
      }
    })
    .filter((entry) => entry.displayModelID.length > 0 && entry.targetModelID.length > 0)

  if (entries.length > 0) {
    const mappingRows = entries
      .filter((entry) => entry.displayModelID !== entry.targetModelID)
      .map((entry) => ({ from: entry.displayModelID, to: entry.targetModelID }))
    if (mappingRows.length > 0) {
      return {
        enabled: true,
        mode: policyMode,
        allowedModels: uniqueStrings(entries.map((entry) => entry.targetModelID)),
        modelMappings: mappingRows
      }
    }
    return {
      enabled: true,
      mode: policyMode,
      allowedModels: uniqueStrings(entries.map((entry) => entry.displayModelID)),
      modelMappings: []
    }
  }

  const selectedModelIDs = uniqueStrings(scope.selected_model_ids)
  if (selectedModelIDs.length > 0) {
    return {
      enabled: true,
      mode: 'whitelist',
      allowedModels: selectedModelIDs,
      modelMappings: []
    }
  }

  const manualMappingRows = Array.isArray(scope.manual_mapping_rows)
    ? uniqueMappingRows(
        scope.manual_mapping_rows.map((item) => {
          const row = item as Record<string, unknown>
          return {
            from: String(row.from || '').trim(),
            to: String(row.to || '').trim()
          }
        })
      )
    : []
  if (manualMappingRows.length > 0) {
    return {
      enabled: true,
      mode: 'mapping',
      allowedModels: uniqueStrings(manualMappingRows.map((item) => item.to)),
      modelMappings: manualMappingRows
    }
  }

  if (scope.manual_mappings && typeof scope.manual_mappings === 'object') {
    const manualMappings = uniqueMappingRows(
      Object.entries(scope.manual_mappings as Record<string, unknown>).map(([from, to]) => ({
        from: String(from || '').trim(),
        to: String(to || '').trim()
      }))
    )
    if (manualMappings.length > 0) {
      return {
        enabled: true,
        mode: 'mapping',
        allowedModels: uniqueStrings(manualMappings.map((item) => item.to)),
        modelMappings: manualMappings
      }
    }
  }

  if (scope.supported_models_by_provider && typeof scope.supported_models_by_provider === 'object') {
    const allowedModels = uniqueStrings(
      Object.values(scope.supported_models_by_provider as Record<string, unknown>).flatMap((value) =>
        Array.isArray(value) ? value : []
      )
    )
    if (allowedModels.length > 0) {
      return {
        enabled: true,
        mode: 'whitelist',
        allowedModels,
        modelMappings: []
      }
    }
  }

  return {
    enabled: true,
    mode: policyMode,
    allowedModels: [],
    modelMappings: []
  }
}

function buildMappingEntries(
  registryModels: Array<{
    id: string
    provider: string
    aliases: string[]
    protocol_ids: string[]
  }>,
  platform: string,
  allowedModels: string[],
  modelMappings: AccountModelScopeMappingRow[]
) {
  const normalizedTargets = uniqueStrings([
    ...allowedModels,
    ...modelMappings.map((item) => item.to)
  ])
  const aliasesByTarget = new Map<string, AccountModelScopeMappingRow[]>()
  for (const row of modelMappings) {
    const current = aliasesByTarget.get(row.to) || []
    current.push(row)
    aliasesByTarget.set(row.to, current)
  }

  const entries: Array<Record<string, unknown>> = []
  for (const targetModelID of normalizedTargets) {
    const aliases = (aliasesByTarget.get(targetModelID) || []).filter(
      (item) => item.from !== item.to
    )
    if (aliases.length > 0) {
      for (const alias of aliases) {
        entries.push(buildScopeEntry(registryModels, platform, alias.from, targetModelID))
      }
      continue
    }
    entries.push(buildScopeEntry(registryModels, platform, targetModelID, targetModelID))
  }
  return entries
}

function buildScopeEntry(
  registryModels: Array<{
    id: string
    provider: string
    aliases: string[]
    protocol_ids: string[]
  }>,
  platform: string,
  displayModelID: string,
  targetModelID: string
) {
  const normalizedDisplayModelID = String(displayModelID || '').trim()
  const normalizedTargetModelID = String(targetModelID || normalizedDisplayModelID).trim()
  const registryEntry = registryModels.find(
    (item) =>
      item.id === normalizedTargetModelID ||
      item.aliases.includes(normalizedTargetModelID) ||
      item.protocol_ids.includes(normalizedTargetModelID)
  )
  const entry: Record<string, unknown> = {
    display_model_id: normalizedDisplayModelID,
    target_model_id: normalizedTargetModelID,
    visibility_mode:
      normalizedDisplayModelID === normalizedTargetModelID ? 'direct' : 'alias'
  }
  const provider = (registryEntry?.provider || platform || '').trim().toLowerCase()
  if (provider) {
    entry.provider = provider
  }
  const sourceProtocol = normalizeSourceProtocol(platform || registryEntry?.provider)
  if (sourceProtocol) {
    entry.source_protocol = sourceProtocol
  }
  return entry
}

function uniqueStrings(values: unknown): string[] {
  if (!Array.isArray(values)) {
    return []
  }
  return [...new Set(values.map((item) => String(item || '').trim()).filter(Boolean))]
}

function uniqueMappingRows(rows: AccountModelScopeMappingRow[]): AccountModelScopeMappingRow[] {
  const normalized: AccountModelScopeMappingRow[] = []
  const seen = new Set<string>()
  for (const row of rows) {
    const from = String(row?.from || '').trim()
    const to = String(row?.to || '').trim()
    if (!from || !to) {
      continue
    }
    const dedupeKey = `${from}::${to}`.toLowerCase()
    if (seen.has(dedupeKey)) {
      continue
    }
    seen.add(dedupeKey)
    normalized.push({ from, to })
  }
  return normalized
}

function normalizeSourceProtocol(value: unknown): string {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'openai' || normalized === 'anthropic' || normalized === 'gemini') {
    return normalized
  }
  return ''
}
