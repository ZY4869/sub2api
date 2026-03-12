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
    validateStatus: (status) => (status >= 200 && status < 300) || status === 304
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

export const metaAPI = {
  getUSDCNYExchangeRate,
  getModelRegistry
}

export default metaAPI
