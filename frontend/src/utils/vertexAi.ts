import { VERTEX_LOCATION_CODES, formatVertexLocationLabel } from '@/utils/displayLabels'

export interface VertexLocationOption {
  value: string
  label: string
  description: string
}

export interface VertexServiceAccountSummary {
  type: string
  client_email: string
  private_key_id: string
  project_id: string
  token_uri: string
}

export type VertexAuthMode = 'service_account' | 'express_api_key'

export const GEMINI_API_KEY_VARIANT_AI_STUDIO = 'ai_studio'
export const GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS = 'vertex_express'
export const VERTEX_DEFAULT_LOCATION = 'global'
export const VERTEX_DEFAULT_ALIAS_PREFIX = 'Vertex-'
export const VERTEX_EXPRESS_DEFAULT_BASE_URL = 'https://aiplatform.googleapis.com'

const VERTEX_LOCATION_DESCRIPTIONS: Partial<Record<string, string>> = {
  global: 'Global',
  'us-west1': 'Oregon',
  'us-west4': 'Las Vegas',
  'us-central1': 'Iowa',
  'us-east1': 'South Carolina',
  'us-east4': 'Northern Virginia',
  'us-east5': 'Columbus',
  'us-south1': 'Dallas',
  'northamerica-northeast1': 'Montreal',
  'southamerica-east1': 'Sao Paulo',
  'europe-west1': 'Belgium',
  'europe-west2': 'London',
  'europe-west3': 'Frankfurt',
  'europe-west4': 'Netherlands',
  'europe-west6': 'Zurich',
  'europe-west8': 'Milan',
  'europe-west9': 'Paris',
  'europe-north1': 'Finland',
  'europe-central2': 'Warsaw',
  'europe-southwest1': 'Madrid',
  'asia-east1': 'Taiwan',
  'asia-east2': 'Hong Kong',
  'asia-northeast1': 'Tokyo',
  'asia-northeast3': 'Seoul',
  'asia-south1': 'Mumbai',
  'asia-southeast1': 'Singapore',
  'australia-southeast1': 'Sydney',
}

export const VERTEX_LOCATION_OPTIONS: VertexLocationOption[] = VERTEX_LOCATION_CODES.map((value) => ({
  value,
  label: formatVertexLocationLabel(value, 'zh'),
  description: VERTEX_LOCATION_DESCRIPTIONS[value] || value
}))

export function normalizeVertexLocation(value: unknown): string {
  const normalized = typeof value === 'string' ? value.trim().toLowerCase() : ''
  return normalized || VERTEX_DEFAULT_LOCATION
}

export function resolveVertexBaseUrl(location: unknown): string {
  const normalized = normalizeVertexLocation(location)
  if (normalized === VERTEX_DEFAULT_LOCATION) {
    return VERTEX_EXPRESS_DEFAULT_BASE_URL
  }
  return `https://${normalized}-aiplatform.googleapis.com`
}

export function resolveVertexAuthBaseUrl(authMode: VertexAuthMode, location: unknown): string {
  if (authMode === 'express_api_key') {
    return VERTEX_EXPRESS_DEFAULT_BASE_URL
  }
  return resolveVertexBaseUrl(location)
}

export function isGeminiVertexExpressVariant(value: unknown): boolean {
  return String(value || '').trim().toLowerCase() === GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS
}

export function isGeminiVertexSourceCredentials(value: unknown): boolean {
  if (!value || typeof value !== 'object') {
    return false
  }
  const credentials = value as Record<string, unknown>
  const oauthType = String(credentials.oauth_type || '').trim().toLowerCase()
  return oauthType === 'vertex_ai' || isGeminiVertexExpressVariant(credentials.gemini_api_variant)
}

export function buildDefaultVertexAlias(modelId: string): string {
  const trimmed = modelId.trim()
  if (!trimmed) {
    return VERTEX_DEFAULT_ALIAS_PREFIX
  }
  return `${VERTEX_DEFAULT_ALIAS_PREFIX}${trimmed}`
}

export function extractVertexServiceAccountSummary(raw: string): VertexServiceAccountSummary {
  const parsed = JSON.parse(raw) as Record<string, unknown>
  if (String(parsed.type || '').trim() !== 'service_account') {
    throw new Error('service_account')
  }

  const clientEmail = String(parsed.client_email || '').trim()
  const privateKey = String(parsed.private_key || '').trim()
  const tokenUri = String(parsed.token_uri || '').trim()

  if (!clientEmail || !privateKey || !tokenUri) {
    throw new Error('missing_required_fields')
  }

  return {
    type: 'service_account',
    client_email: clientEmail,
    private_key_id: String(parsed.private_key_id || '').trim(),
    project_id: String(parsed.project_id || '').trim(),
    token_uri: tokenUri
  }
}
