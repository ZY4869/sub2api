import type { ModelMapping } from '@/utils/accountFormShared'
import { buildAccountModelScopeExtra } from '@/utils/accountModelScope'
import { normalizeOpenAIPlanType } from '@/utils/openaiAccountDefaults'

export const OPENAI_OAUTH_DEFAULT_MODELS = [
  'gpt-image-2',
  'gpt-5.2',
  'gpt-5.4',
  'gpt-5.4-mini',
  'gpt-5.5',
] as const

export const OPENAI_OAUTH_PRO_SPARK_MODEL = 'gpt-5.3-codex-spark'

export interface ApplyOpenAIOAuthDefaultsOptions {
  planType?: string | null
  proMultiplier?: number | null
  currentAllowedModels: string[]
  currentModelMappings: ModelMapping[]
  modelRestrictionMode: 'whitelist' | 'mapping'
  userCustomized: boolean
}

export function shouldEnableOpenAIProSpark(options: {
  planType?: string | null
  proMultiplier?: number | null
}): boolean {
  if (normalizeOpenAIPlanType(options.planType) !== 'pro') {
    return false
  }
  return typeof options.proMultiplier === 'number'
    ? options.proMultiplier > 0
    : true
}

export function resolveOpenAIOAuthDefaultAllowedModels(options: {
  planType?: string | null
  proMultiplier?: number | null
}): string[] {
  const next: string[] = [...OPENAI_OAUTH_DEFAULT_MODELS]
  if (shouldEnableOpenAIProSpark(options)) {
    next.push(OPENAI_OAUTH_PRO_SPARK_MODEL)
  }
  return next
}

export function applyOpenAIOAuthDefaultModelState(
  options: ApplyOpenAIOAuthDefaultsOptions
): {
  allowedModels: string[]
  modelMappings: ModelMapping[]
} {
  if (options.userCustomized) {
    return {
      allowedModels: [...options.currentAllowedModels],
      modelMappings: options.currentModelMappings.map((item) => ({ ...item })),
    }
  }

  return {
    allowedModels: resolveOpenAIOAuthDefaultAllowedModels({
      planType: options.planType,
      proMultiplier: options.proMultiplier,
    }),
    modelMappings:
      options.modelRestrictionMode === 'mapping'
        ? options.currentModelMappings.map((item) => ({ ...item }))
        : [],
  }
}

export function buildOpenAIOAuthCreateExtra(baseExtra: Record<string, unknown> | undefined, options: {
  modelRestrictionEnabled: boolean
  modelRestrictionMode: 'whitelist' | 'mapping'
  allowedModels: string[]
  modelMappings: ModelMapping[]
}): Record<string, unknown> | undefined {
  return buildAccountModelScopeExtra(baseExtra, {
    platform: 'openai',
    enabled: options.modelRestrictionEnabled,
    mode: options.modelRestrictionMode,
    allowedModels: options.allowedModels,
    modelMappings: options.modelMappings,
  })
}
