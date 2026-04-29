import { apiClient } from './client'
import type { ModelRegistrySnapshot } from '@/generated/modelRegistry'

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
  value: number
}

export interface PublicModelCatalogPriceDisplay {
  primary: PublicModelCatalogPriceEntry[]
  secondary?: PublicModelCatalogPriceEntry[]
}

export interface PublicModelCatalogMultiplierSummary {
  enabled: boolean
  kind: 'disabled' | 'uniform' | 'mixed'
  mode?: 'shared' | 'item'
  value?: number
}

export type PublicModelCatalogStatus = 'ok' | 'error' | 'maintenance' | 'warning' | 'info'
export type PublicModelAvailabilityState = 'verified' | 'unavailable' | 'unknown'
export type PublicModelStaleState = 'fresh' | 'stale' | 'unverified'
export type PublicModelLifecycleStatus = 'stable' | 'beta' | 'deprecated'

export interface PublicModelCatalogItem {
  model: string
  display_name?: string
  provider?: string
  provider_icon_key?: string
  status?: PublicModelCatalogStatus
  availability_state?: PublicModelAvailabilityState
  stale_state?: PublicModelStaleState
  lifecycle_status?: PublicModelLifecycleStatus
  request_protocols?: string[]
  source_ids?: string[]
  mode?: string
  currency: string
  price_display: PublicModelCatalogPriceDisplay
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
}

export interface PublicModelCatalogSnapshot {
  etag: string
  updated_at: string
  page_size?: number
  catalog_source?: PublicModelCatalogSource
  items: PublicModelCatalogItem[]
}

export interface ModelCatalogFetchResult {
  notModified: boolean
  etag: string | null
  data: PublicModelCatalogSnapshot | null
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

export async function getModelCatalog(etag?: string | null): Promise<ModelCatalogFetchResult> {
  const headers: Record<string, string> = {}
  if (etag) {
    headers['If-None-Match'] = etag
  }

  const response = await apiClient.get<PublicModelCatalogSnapshot>('/meta/model-catalog', {
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

export async function getModelCatalogDetail(model: string): Promise<PublicModelCatalogDetailResponse> {
  const { data } = await apiClient.get<PublicModelCatalogDetailResponse>(`/meta/model-catalog/${encodeURIComponent(model)}`)
  return data
}

export const metaAPI = {
  getUSDCNYExchangeRate,
  getModelRegistry,
  getModelCatalog,
  getModelCatalogDetail
}

export default metaAPI
