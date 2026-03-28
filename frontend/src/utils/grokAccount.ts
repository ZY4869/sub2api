export type GrokTier = 'basic' | 'super' | 'heavy'

export const GROK_SHARED_MODEL_IDS = [
  'grok-3-beta',
  'grok-3-mini-beta',
  'grok-3-fast-beta',
  'grok-2',
  'grok-2-vision',
  'grok-imagine-image',
  'grok-imagine-video',
  'grok-2-image',
  'grok-beta',
  'grok-vision-beta'
] as const

export const GROK_HEAVY_ONLY_MODEL_IDS = [
  'grok-4',
  'grok-4-0709'
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
