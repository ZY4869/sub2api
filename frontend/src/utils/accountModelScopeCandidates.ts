import type { ModelRegistryEntry } from '@/generated/modelRegistry'

export const COMMON_MAX_PER_PROVIDER = 24
export const MAX_RESULTS_PER_PROVIDER = 120

export type ModelScopeWhitelistViewMode = 'default' | 'search' | 'all'

export interface ModelScopeProviderGroup {
  provider: string
  label: string
  entries: ModelRegistryEntry[]
  selectedCount: number
  totalCount: number
  truncated: boolean
}

export interface GetModelScopeWhitelistGroupsOptions {
  platform: string
  selectedModelIds: Set<string>
  query: string
  showAllModels: boolean
  commonMaxPerProvider?: number
  maxResultsPerProvider?: number
}

export interface GetModelScopeWhitelistGroupsResult {
  mode: ModelScopeWhitelistViewMode
  providerGroups: ModelScopeProviderGroup[]
  providerAllModelIds: Map<string, string[]>
}

function normalizePlatform(platform: string): string {
  const value = platform.trim().toLowerCase()
  return value === 'claude' ? 'anthropic' : value
}

function sortEntries(entries: ModelRegistryEntry[]): ModelRegistryEntry[] {
  return [...entries].sort((left, right) => (left.ui_priority - right.ui_priority) || left.id.localeCompare(right.id))
}

export function formatProviderLabel(provider: string): string {
  switch (provider) {
    case 'openai':
      return 'OpenAI'
    case 'anthropic':
      return 'Anthropic'
    case 'gemini':
      return 'Gemini'
    case 'antigravity':
      return 'Antigravity'
    case 'sora':
      return 'Sora'
    default:
      return provider ? provider.charAt(0).toUpperCase() + provider.slice(1) : 'Unknown'
  }
}

function computeViewMode(query: string, showAllModels: boolean): ModelScopeWhitelistViewMode {
  if (query.trim()) return 'search'
  if (showAllModels) return 'all'
  return 'default'
}

function isCommonModel(entry: ModelRegistryEntry): boolean {
  return entry.exposed_in.includes('runtime') || entry.exposed_in.includes('test')
}

function buildUnknownEntry(modelId: string, platform: string): ModelRegistryEntry {
  return {
    id: modelId,
    display_name: modelId,
    provider: 'unknown',
    platforms: [platform],
    protocol_ids: [],
    aliases: [],
    pricing_lookup_ids: [],
    modalities: [],
    capabilities: [],
    ui_priority: Number.MAX_SAFE_INTEGER,
    exposed_in: ['whitelist']
  }
}

function truncateWithSelectedPinned(entries: ModelRegistryEntry[], selectedModelIds: Set<string>, limit: number): ModelRegistryEntry[] {
  if (entries.length <= limit) return entries

  const selected = entries.filter((entry) => selectedModelIds.has(entry.id))
  const unselected = entries.filter((entry) => !selectedModelIds.has(entry.id))
  const availableSlots = Math.max(limit - selected.length, 0)
  return [...selected.slice(0, limit), ...unselected.slice(0, availableSlots)]
}

export function getModelScopeWhitelistGroups(
  registryModels: ModelRegistryEntry[],
  options: GetModelScopeWhitelistGroupsOptions
): GetModelScopeWhitelistGroupsResult {
  const normalizedPlatform = normalizePlatform(options.platform)
  const query = options.query.trim().toLowerCase()
  const mode = computeViewMode(options.query, options.showAllModels)
  const commonMaxPerProvider = options.commonMaxPerProvider ?? COMMON_MAX_PER_PROVIDER
  const maxResultsPerProvider = options.maxResultsPerProvider ?? MAX_RESULTS_PER_PROVIDER

  const providerAllModelIds = new Map<string, string[]>()
  const providerEntryMap = new Map<string, ModelRegistryEntry[]>()
  const knownModelIds = new Set<string>()

  for (const entry of registryModels) {
    if (!entry.platforms.includes(normalizedPlatform)) continue
    const provider = (entry.provider || normalizedPlatform || 'unknown').trim().toLowerCase()
    knownModelIds.add(entry.id)
    const bucket = providerEntryMap.get(provider) || []
    bucket.push(entry)
    providerEntryMap.set(provider, bucket)
  }

  for (const [provider, entries] of providerEntryMap) {
    const sorted = sortEntries(entries)
    providerEntryMap.set(provider, sorted)
    providerAllModelIds.set(provider, sorted.map((entry) => entry.id))
  }

  const providerGroups: ModelScopeProviderGroup[] = []

  for (const [provider, sortedEntries] of providerEntryMap) {
    let matchedEntries = sortedEntries
    let totalCount = sortedEntries.length
    let truncated = false

    if (mode === 'search') {
      matchedEntries = sortedEntries.filter((entry) => {
        const haystack = `${entry.id} ${entry.display_name || ''} ${entry.provider || ''}`.toLowerCase()
        return haystack.includes(query)
      })
      totalCount = matchedEntries.length
      matchedEntries = truncateWithSelectedPinned(matchedEntries, options.selectedModelIds, maxResultsPerProvider)
      truncated = totalCount > matchedEntries.length
    } else if (mode === 'all') {
      totalCount = sortedEntries.length
      matchedEntries = sortedEntries.slice(0, maxResultsPerProvider)
      truncated = totalCount > matchedEntries.length
    } else {
      const selected = sortedEntries.filter((entry) => options.selectedModelIds.has(entry.id))
      const common = sortedEntries
        .filter((entry) => isCommonModel(entry) && !options.selectedModelIds.has(entry.id))
        .slice(0, commonMaxPerProvider)
      const seen = new Set<string>()
      matchedEntries = [...selected, ...common].filter((entry) => {
        if (seen.has(entry.id)) return false
        seen.add(entry.id)
        return true
      })
      totalCount = matchedEntries.length
      truncated = false
    }

    if (matchedEntries.length === 0) continue

    const selectedCount = matchedEntries.filter((entry) => options.selectedModelIds.has(entry.id)).length
    providerGroups.push({
      provider,
      label: formatProviderLabel(provider),
      entries: matchedEntries,
      selectedCount,
      totalCount,
      truncated
    })
  }

  if (mode === 'default') {
    const missingSelectedIds = [...options.selectedModelIds].filter((id) => !knownModelIds.has(id))
    if (missingSelectedIds.length > 0) {
      const entries = missingSelectedIds.map((id) => buildUnknownEntry(id, normalizedPlatform))
      providerGroups.push({
        provider: 'unknown',
        label: 'Unknown',
        entries,
        selectedCount: entries.length,
        totalCount: entries.length,
        truncated: false
      })
    }
  }

  providerGroups.sort((left, right) => left.label.localeCompare(right.label))

  return { mode, providerGroups, providerAllModelIds }
}

