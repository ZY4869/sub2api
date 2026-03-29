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

const VERTEX_LOCATION_DEFINITIONS = [
  { value: 'global', zh: '全球', en: 'Global' },
  { value: 'us-west1', zh: '美国西部 1', en: 'Oregon' },
  { value: 'us-west4', zh: '美国西部 4', en: 'Las Vegas' },
  { value: 'us-central1', zh: '美国中部 1', en: 'Iowa' },
  { value: 'us-east1', zh: '美国东部 1', en: 'South Carolina' },
  { value: 'us-east4', zh: '美国东部 4', en: 'Northern Virginia' },
  { value: 'us-east5', zh: '美国东部 5', en: 'Columbus' },
  { value: 'us-south1', zh: '美国南部 1', en: 'Dallas' },
  { value: 'northamerica-northeast1', zh: '北美东北 1', en: 'Montreal' },
  { value: 'southamerica-east1', zh: '南美东部 1', en: 'Sao Paulo' },
  { value: 'europe-west1', zh: '欧洲西部 1', en: 'Belgium' },
  { value: 'europe-west2', zh: '欧洲西部 2', en: 'London' },
  { value: 'europe-west3', zh: '欧洲西部 3', en: 'Frankfurt' },
  { value: 'europe-west4', zh: '欧洲西部 4', en: 'Netherlands' },
  { value: 'europe-west6', zh: '欧洲西部 6', en: 'Zurich' },
  { value: 'europe-west8', zh: '欧洲西部 8', en: 'Milan' },
  { value: 'europe-west9', zh: '欧洲西部 9', en: 'Paris' },
  { value: 'europe-north1', zh: '欧洲北部 1', en: 'Finland' },
  { value: 'europe-central2', zh: '欧洲中部 2', en: 'Warsaw' },
  { value: 'europe-southwest1', zh: '欧洲西南 1', en: 'Madrid' },
  { value: 'asia-east1', zh: '亚洲东部 1', en: 'Taiwan' },
  { value: 'asia-east2', zh: '亚洲东部 2', en: 'Hong Kong' },
  { value: 'asia-northeast1', zh: '亚洲东北 1', en: 'Tokyo' },
  { value: 'asia-northeast3', zh: '亚洲东北 3', en: 'Seoul' },
  { value: 'asia-south1', zh: '亚洲南部 1', en: 'Mumbai' },
  { value: 'asia-southeast1', zh: '亚洲东南 1', en: 'Singapore' },
  { value: 'australia-southeast1', zh: '澳大利亚东南 1', en: 'Sydney' }
] as const

export const VERTEX_LOCATION_OPTIONS: VertexLocationOption[] =
  VERTEX_LOCATION_DEFINITIONS.map((item) => ({
    value: item.value,
    label: `${item.zh} (${item.value})`,
    description: item.en
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
