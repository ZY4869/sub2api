import { ensureModelRegistryFresh, getModelRegistrySnapshot } from '@/stores/modelRegistry'

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

export function buildAccountModelScopeExtra(
  baseExtra: Record<string, unknown> | undefined,
  options: BuildAccountModelScopeOptions
): Record<string, unknown> | undefined {
  const nextExtra = { ...(baseExtra || {}) }
  if (!options.enabled) {
    delete nextExtra.model_scope_v2
    return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
  }

  void ensureModelRegistryFresh()
  const snapshot = getModelRegistrySnapshot()
  const normalizedAllowedModels = [...new Set(
    options.allowedModels
      .map((item) => item.trim())
      .filter(Boolean)
  )]
  const manualMappings =
    options.mode === 'mapping'
      ? Object.fromEntries(
          options.modelMappings
            .map((item) => [item.from.trim(), item.to.trim()] as const)
            .filter(([from, to]) => Boolean(from) && Boolean(to) && from !== to)
        )
      : {}
  const manualMappingRows =
    options.mode === 'mapping'
      ? options.modelMappings
          .map((item) => ({ from: item.from.trim(), to: item.to.trim() }))
          .filter((item) => Boolean(item.from) && Boolean(item.to) && item.from !== item.to)
      : []

  const selectedModels =
    options.mode === 'whitelist'
      ? normalizedAllowedModels
      : (normalizedAllowedModels.length > 0
          ? normalizedAllowedModels
          : options.modelMappings
              .map((item) => item.to.trim())
              .filter((value) => value && !value.includes('*')))

  const supportedModelsByProvider: Record<string, string[]> = {}

  for (const modelId of selectedModels) {
    const entry = snapshot.models.find(
      (item) =>
        item.id === modelId ||
        item.aliases.includes(modelId) ||
        item.protocol_ids.includes(modelId)
    )
    if (!entry) {
      const fallbackProvider = (options.platform || '').trim().toLowerCase()
      if (!fallbackProvider) {
        continue
      }
      const current = supportedModelsByProvider[fallbackProvider] || []
      if (!current.includes(modelId)) {
        supportedModelsByProvider[fallbackProvider] = [...current, modelId]
      }
      continue
    }
    const provider = (entry.provider || options.platform || '').trim().toLowerCase()
    if (!provider) {
      continue
    }
    const current = supportedModelsByProvider[provider] || []
    if (!current.includes(entry.id)) {
      supportedModelsByProvider[provider] = [...current, entry.id]
    }
  }

  const supportedProviders = Object.keys(supportedModelsByProvider).sort()
  for (const provider of supportedProviders) {
    supportedModelsByProvider[provider] = [...supportedModelsByProvider[provider]].sort()
  }

  if (supportedProviders.length === 0 && Object.keys(manualMappings).length === 0 && manualMappingRows.length === 0) {
    delete nextExtra.model_scope_v2
    return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
  }

  nextExtra.model_scope_v2 = {
    supported_providers: supportedProviders,
    supported_models_by_provider: supportedModelsByProvider,
    advanced_provider_override: false,
    manual_mapping_rows: manualMappingRows,
    manual_mappings: manualMappings
  }
  return nextExtra
}
