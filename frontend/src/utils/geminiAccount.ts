export type GeminiOAuthType = 'code_assist' | 'google_one' | 'ai_studio' | 'vertex_ai'

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
