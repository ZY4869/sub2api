import type { GeminiCredentials } from '@/types'
import {
  GEMINI_API_KEY_VARIANT_AI_STUDIO,
  GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS
} from '@/utils/vertexAi'

export type GeminiOAuthType = 'code_assist' | 'google_one' | 'ai_studio' | 'vertex_ai'
export type GeminiAPIKeyVariant = 'ai_studio' | 'vertex_express'
export type GeminiChannel = 'ai_studio' | 'ai_studio_client' | 'vertex_ai' | 'google_one' | 'gcp'
export type GeminiAIStudioTier =
  | 'aistudio_free'
  | 'aistudio_tier_1'
  | 'aistudio_tier_2'
  | 'aistudio_tier_3'

export type GeminiBrowserOAuthType = Exclude<GeminiOAuthType, 'vertex_ai'>

export const GEMINI_VERTEX_DEFAULT_BASE_URL = 'https://aiplatform.googleapis.com'

export function normalizeGeminiOAuthType(value: unknown): GeminiOAuthType {
  const normalized = typeof value === 'string' ? value.trim().toLowerCase() : ''
  switch (normalized) {
    case 'google_one':
    case 'ai_studio':
    case 'vertex_ai':
      return normalized
    default:
      return 'code_assist'
  }
}

export function isGeminiVertexAI(value: unknown): value is 'vertex_ai' {
  return normalizeGeminiOAuthType(value) === 'vertex_ai'
}

export function normalizeGeminiAPIKeyVariant(value: unknown): GeminiAPIKeyVariant {
  const normalized = typeof value === 'string' ? value.trim().toLowerCase() : ''
  switch (normalized) {
    case GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS:
      return 'vertex_express'
    case GEMINI_API_KEY_VARIANT_AI_STUDIO:
    default:
      return 'ai_studio'
  }
}

export function normalizeGeminiAIStudioTier(value: unknown): GeminiAIStudioTier {
  const normalized = typeof value === 'string' ? value.trim().toLowerCase() : ''
  switch (normalized) {
    case 'aistudio_tier_2':
      return 'aistudio_tier_2'
    case 'aistudio_tier_3':
      return 'aistudio_tier_3'
    case 'aistudio_paid':
    case 'aistudio_tier_1':
      return 'aistudio_tier_1'
    case 'aistudio_free':
    default:
      return 'aistudio_free'
  }
}

export function resolveGeminiChannel(input?: {
  type?: unknown
  credentials?: GeminiCredentials | Record<string, unknown> | null
} | null): GeminiChannel | null {
  const type = typeof input?.type === 'string' ? input.type.trim().toLowerCase() : ''
  const credentials = input?.credentials as GeminiCredentials | undefined

  if (type === 'apikey') {
    return normalizeGeminiAPIKeyVariant(credentials?.gemini_api_variant) === 'vertex_express'
      ? 'vertex_ai'
      : 'ai_studio'
  }

  switch (normalizeGeminiOAuthType(credentials?.oauth_type)) {
    case 'google_one':
      return 'google_one'
    case 'ai_studio':
      return 'ai_studio_client'
    case 'vertex_ai':
      return 'vertex_ai'
    default:
      return 'gcp'
  }
}

export function resolveGeminiChannelDisplayName(channel: GeminiChannel | null): string | null {
  switch (channel) {
    case 'ai_studio':
      return 'AI Studio'
    case 'ai_studio_client':
      return 'AI Studio Client'
    case 'vertex_ai':
      return 'Vertex AI'
    case 'google_one':
      return 'Google One'
    case 'gcp':
      return 'GCP'
    default:
      return null
  }
}
