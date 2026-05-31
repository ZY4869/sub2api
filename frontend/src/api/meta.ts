import { apiClient } from './client'
import type { ModelRegistrySnapshot } from '@/generated/modelRegistry'
import type { TimeAccessPolicy } from '@/types/api-key-groups'

export interface ExchangeRateInfo {
  base: string
  quote: string
  rate: number
  date: string
  updated_at: string
  cached: boolean
}

export interface ModelRegistryFetchResult {
  notModified: boolean
  etag: string | null
  data: ModelRegistrySnapshot | null
}

export interface PublicModelCatalogPriceEntry {
  id: string
  unit?: string
  unit_kind?: 'token' | 'image' | 'request' | 'video' | string
  display_unit?: 'per_million_tokens' | 'per_image' | 'per_request' | 'per_video' | string
  value: number
  configured?: boolean
  supported_unpriced?: boolean
}

export interface PublicModelCatalogPriceDisplay {
  primary: PublicModelCatalogPriceEntry[]
  secondary?: PublicModelCatalogPriceEntry[]
}

export interface PublicModelCatalogDiscountWindow {
  id?: string
  type: 'once' | 'daily' | string
  start_at?: string
  end_at?: string
  start_time?: string
  end_time?: string
  days?: number[]
}

export interface PublicModelCatalogDiscountPolicy {
  enabled: boolean
  reduction_percent?: number
  timezone?: string
  windows?: PublicModelCatalogDiscountWindow[]
}

export interface PublicModelCatalogDiscountStatus {
  active: boolean
  reduction_percent?: number
  window_id?: string
  window_type?: string
  timezone?: string
  completed_at?: string
}

export type PublicModelImageResolution = '1K' | '2K' | '4K'

export interface PublicModelImageFixedPricing {
  enabled: boolean
  always_fixed: boolean
  prices: Record<PublicModelImageResolution, number | null>
}

export interface PublicModelCatalogMultiplierSummary {
  enabled: boolean
  kind: 'disabled' | 'uniform' | 'mixed'
  mode?: 'shared' | 'item'
  value?: number
}

export type PublicModelPublicationStatus = 'published'
export type PublicModelLifecycleStatus = 'stable' | 'beta' | 'deprecated'
export type PublicModelHealthStatus = 'healthy' | 'warning' | 'error' | 'pending'
export type PublicModelScheduleStatus = 'active' | 'scheduled' | 'expired' | 'out_of_window' | 'invalid' | string
export type PublicModelVerificationSource = 'published_snapshot' | 'live_fallback'
export type PublicModelKeyAvailability = 'available' | 'unavailable'
export type PublicModelUnavailableReason =
  | 'not_selected_by_key'
  | 'group_unavailable'
  | 'image_only_key_restricted'
  | 'published_source_unavailable'
export type PublicModelHealthSource = 'traffic' | 'probe' | 'none'
export type PublicModelHealthReason =
  | 'traffic_recent'
  | 'probe_recent'
  | 'monitor_disabled'
  | 'no_history'
  | 'stale_history'
  | 'checking'

export type PublicModelSupport = 'supported' | 'partial' | 'unsupported' | 'unknown'

export interface PublicModelContextWindow {
  tokens?: number
  source?: string
  verified: boolean
  last_checked_at?: string
  limit_kind?: string
  notes?: string[]
}

export interface PublicModelCapabilityMatrixEntry {
  capability: string
  protocol?: string
  endpoint?: string
  support: PublicModelSupport
  mode?: string
  source?: string
  verified: boolean
  last_checked_at?: string
  limitations?: string[]
}

export interface PublicModelProtocolEndpoint {
  key: string
  protocol: string
  endpoint: string
  method?: string
  support: PublicModelSupport
  source?: string
  verified: boolean
  last_checked_at?: string
  limitations?: string[]
}

export interface PublicModelLifecycle {
  status?: PublicModelLifecycleStatus
  source?: string
  confidence?: 'verified' | 'declared' | 'inferred' | string
}

export interface PublicModelCatalogItem {
  entry_id?: string
  public_model_id?: string
  model: string
  base_model?: string
  source_model_id?: string
  source_protocol?: string
  source_alias?: string
  display_name?: string
  provider?: string
  provider_icon_key?: string
  publication_status?: PublicModelPublicationStatus
  health_status?: PublicModelHealthStatus
  verification_source?: PublicModelVerificationSource
  key_availability?: PublicModelKeyAvailability
  unavailable_reason?: PublicModelUnavailableReason
  lifecycle_status?: PublicModelLifecycleStatus
  lifecycle?: PublicModelLifecycle
  context_window_tokens?: number
  context_window?: PublicModelContextWindow
  modalities?: string[]
  capabilities?: string[]
  capability_matrix?: PublicModelCapabilityMatrixEntry[]
  request_protocols?: string[]
  protocol_endpoints?: PublicModelProtocolEndpoint[]
  is_demo?: boolean
  catalog_entry_source?: 'real_account' | 'live_projection' | 'demo' | 'legacy_snapshot' | string
  available_from?: string
  available_until?: string
  access_time_policy?: TimeAccessPolicy | null
  schedule_status?: PublicModelScheduleStatus
  mode?: string
  currency: string
  price_display: PublicModelCatalogPriceDisplay
  official_price_display?: PublicModelCatalogPriceDisplay
  sale_price_display?: PublicModelCatalogPriceDisplay
  original_price_display?: PublicModelCatalogPriceDisplay
  original_sale_price_display?: PublicModelCatalogPriceDisplay
  discount_policy?: PublicModelCatalogDiscountPolicy | null
  discount_status?: PublicModelCatalogDiscountStatus | null
  image_fixed_pricing?: PublicModelImageFixedPricing
  multiplier_summary: PublicModelCatalogMultiplierSummary
}

export type PublicModelCatalogSource = 'published' | 'live_fallback'

export interface PublicModelCatalogDetailResponse {
  item: PublicModelCatalogItem
  catalog_source?: PublicModelCatalogSource
  example_source?: 'docs_section' | 'override_template'
  example_protocol?: string
  example_page_id?: string
  example_markdown?: string
  example_override_id?: string
  example_validation?: 'dry_run_contract' | string
}

export interface PublicModelCatalogSnapshot {
  etag: string
  updated_at: string
  published_at?: string
  last_revalidated_at?: string
  stale_reason?: string
  page_size?: number
  catalog_source?: PublicModelCatalogSource
  items: PublicModelCatalogItem[]
}

export interface ModelCatalogFetchResult {
  notModified: boolean
  etag: string | null
  data: PublicModelCatalogSnapshot | null
}

export interface PublicModelCatalogRequestOptions {
  catalogMode?: 'demo' | 'real'
}

export interface PublicModelCatalogDailyStatus {
  date: string
  status: PublicModelHealthStatus
  success_rate?: number
  latency_ms?: number
}

export interface PublicModelCatalogTrendPoint {
  timestamp: string
  success_rate?: number
  latency_ms?: number
}

export interface PublicModelCatalogStatusItem {
  public_model_id: string
  model: string
  aliases: string[]
  health_status: PublicModelHealthStatus
  health_source: PublicModelHealthSource
  status_reason: PublicModelHealthReason
  success_rate_today?: number
  success_rate_7d?: number
  latency_ms?: number
  last_checked_at?: string
  daily: PublicModelCatalogDailyStatus[]
  trend: PublicModelCatalogTrendPoint[]
}

export interface PublicModelCatalogStatusSnapshot {
  updated_at: string
  items: PublicModelCatalogStatusItem[]
}

export async function getUSDCNYExchangeRate(force = false): Promise<ExchangeRateInfo> {
  const requestConfig = force ? { params: { force: true } } : undefined
  const { data } = await apiClient.get<ExchangeRateInfo>('/meta/exchange-rate/usd-cny', requestConfig)
  return data
}

export async function getModelRegistry(etag?: string | null): Promise<ModelRegistryFetchResult> {
  const headers: Record<string, string> = {}
  if (etag) {
    headers['If-None-Match'] = etag
  }

  const response = await apiClient.get<ModelRegistrySnapshot>('/meta/model-registry', {
    headers,
    validateStatus: (status: number) => (status >= 200 && status < 300) || status === 304
  })

  const nextEtag = typeof response.headers?.etag === 'string' ? response.headers.etag : null
  if (response.status === 304) {
    return {
      notModified: true,
      etag: nextEtag || etag || null,
      data: null
    }
  }

  return {
    notModified: false,
    etag: nextEtag || response.data?.etag || null,
    data: response.data
  }
}

export async function getModelCatalog(
  etag?: string | null,
  options: PublicModelCatalogRequestOptions = {}
): Promise<ModelCatalogFetchResult> {
  const headers: Record<string, string> = {}
  if (etag) {
    headers['If-None-Match'] = etag
  }
  const params = options.catalogMode ? { catalog_mode: options.catalogMode } : undefined

  const response = await apiClient.get<PublicModelCatalogSnapshot>('/meta/model-catalog', {
    headers,
    params,
    validateStatus: (status: number) => (status >= 200 && status < 300) || status === 304
  })

  const nextEtag = typeof response.headers?.etag === 'string' ? response.headers.etag : null
  if (response.status === 304) {
    return {
      notModified: true,
      etag: nextEtag || etag || null,
      data: null
    }
  }

  return {
    notModified: false,
    etag: nextEtag || response.data?.etag || null,
    data: response.data
  }
}

export async function getModelCatalogDetail(
  model: string,
  options: PublicModelCatalogRequestOptions = {}
): Promise<PublicModelCatalogDetailResponse> {
  const params = options.catalogMode ? { catalog_mode: options.catalogMode } : undefined
  const { data } = await apiClient.get<PublicModelCatalogDetailResponse>(
    `/meta/model-catalog/${encodeURIComponent(model)}`,
    { params }
  )
  return data
}

export async function getModelCatalogStatus(): Promise<PublicModelCatalogStatusSnapshot> {
  const { data } = await apiClient.get<PublicModelCatalogStatusSnapshot>('/meta/model-catalog/status')
  return data
}

export const metaAPI = {
  getUSDCNYExchangeRate,
  getModelRegistry,
  getModelCatalog,
  getModelCatalogDetail,
  getModelCatalogStatus
}

export default metaAPI
