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
  const manualMappings =
    options.mode === 'mapping'
      ? Object.fromEntries(
          options.modelMappings
            .map((item) => [item.from.trim(), item.to.trim()] as const)
            .filter(([from, to]) => Boolean(from) && Boolean(to))
        )
      : {}

  const selectedModels =
    options.mode === 'whitelist'
      ? options.allowedModels
      : options.modelMappings
          .map((item) => item.from.trim())
          .filter((value) => value && !value.includes('*'))

  const supportedModelsByProvider: Record<string, string[]> = {}

  for (const modelId of selectedModels) {
    const entry = snapshot.models.find(
      (item) =>
        item.id === modelId ||
        item.aliases.includes(modelId) ||
        item.protocol_ids.includes(modelId)
    )
    if (!entry) {
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

  if (supportedProviders.length === 0 && Object.keys(manualMappings).length === 0) {
    delete nextExtra.model_scope_v2
    return Object.keys(nextExtra).length > 0 ? nextExtra : undefined
  }

  nextExtra.model_scope_v2 = {
    supported_providers: supportedProviders,
    supported_models_by_provider: supportedModelsByProvider,
    advanced_provider_override: false,
    manual_mappings: manualMappings
  }
  return nextExtra
}
