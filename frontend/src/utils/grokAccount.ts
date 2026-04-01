export type GrokTier = 'basic' | 'super' | 'heavy'

export const GROK_SHARED_MODEL_IDS = [
  'grok-auto',
  'grok-3-fast',
  'grok-4-expert',
  'grok-imagine-1.0-fast',
  'grok-imagine-1.0',
  'grok-imagine-1.0-edit',
  'grok-imagine-1.0-video'
] as const

export const GROK_HEAVY_ONLY_MODEL_IDS = [
  'grok-4-heavy'
] as const

export function normalizeGrokTier(value: unknown): GrokTier {
  switch (String(value || '').trim().toLowerCase()) {
    case 'heavy':
      return 'heavy'
    case 'super':
      return 'super'
    default:
      return 'basic'
  }
}

export function grokDefaultModelIdsForTier(value: unknown): string[] {
  const tier = normalizeGrokTier(value)
  return tier === 'heavy'
    ? [...GROK_SHARED_MODEL_IDS, ...GROK_HEAVY_ONLY_MODEL_IDS]
    : [...GROK_SHARED_MODEL_IDS]
}

export function grokDefaultModelMappingForTier(value: unknown): Record<string, string> {
  return grokDefaultModelIdsForTier(value).reduce<Record<string, string>>((mapping, modelId) => {
    mapping[modelId] = modelId
    return mapping
  }, {})
}

export function parseModelMappingRecord(value: unknown): Record<string, string> {
  if (!value || typeof value !== 'object') {
    return {}
  }

  return Object.entries(value as Record<string, unknown>).reduce<Record<string, string>>((mapping, [from, to]) => {
    const source = String(from || '').trim()
    const target = String(to || '').trim()
    if (!source || !target) {
      return mapping
    }
    mapping[source] = target
    return mapping
  }, {})
}

export function mappingRecordToRows(value: unknown): Array<{ from: string; to: string }> {
  return Object.entries(parseModelMappingRecord(value)).map(([from, to]) => ({ from, to }))
}
