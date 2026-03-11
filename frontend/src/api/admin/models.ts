import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'
import type { ExchangeRateInfo } from '@/api/meta'

export type ModelCatalogPricingSource = 'none' | 'dynamic' | 'fallback' | 'override'

export interface ModelCatalogPricing {
  input_cost_per_token?: number
  input_cost_per_token_priority?: number
  input_token_threshold?: number
  input_cost_per_token_above_threshold?: number
  input_cost_per_token_priority_above_threshold?: number
  output_cost_per_token?: number
  output_cost_per_token_priority?: number
  output_token_threshold?: number
  output_cost_per_token_above_threshold?: number
  output_cost_per_token_priority_above_threshold?: number
  cache_creation_input_token_cost?: number
  cache_creation_input_token_cost_above_1hr?: number
  cache_read_input_token_cost?: number
  cache_read_input_token_cost_priority?: number
  output_cost_per_image?: number
}

export interface ModelPricingOverride extends ModelCatalogPricing {
  updated_at: string
  updated_by_user_id: number
  updated_by_email?: string
}

export type ModelCatalogExchangeRate = ExchangeRateInfo

export interface ModelCatalogRouteReference {
  group_id: number
  group_name: string
  platform: string
  reference_types: string[]
  matched_routing_patterns?: string[]
}

export interface ModelCatalogItem {
  model: string
  display_name?: string
  icon_key?: string
  provider?: string
  mode?: string
  default_available: boolean
  default_platforms?: string[]
  access_sources?: string[]
  pricing_source: ModelCatalogPricingSource
  base_pricing_source: Exclude<ModelCatalogPricingSource, 'override'>
  has_override: boolean
  official_pricing?: ModelCatalogPricing
  sale_pricing?: ModelCatalogPricing
  effective_pricing?: ModelCatalogPricing
  supports_prompt_caching: boolean
  supports_service_tier: boolean
  long_context_input_token_threshold?: number
  long_context_input_cost_multiplier?: number
  long_context_output_cost_multiplier?: number
}

export interface ModelCatalogDetail extends ModelCatalogItem {
  upstream_pricing?: ModelCatalogPricing
  official_override_pricing?: ModelPricingOverride
  sale_override_pricing?: ModelPricingOverride
  base_pricing?: ModelCatalogPricing
  override_pricing?: ModelPricingOverride
  route_references: ModelCatalogRouteReference[]
  route_reference_count: number
}

export interface ListModelsParams {
  search?: string
  provider?: string
  mode?: string
  availability?: 'available' | 'unavailable'
  pricing_source?: ModelCatalogPricingSource
  page?: number
  page_size?: number
}

export interface UpdatePricingOverridePayload extends ModelCatalogPricing {
  model: string
}

export interface UpsertModelCatalogEntryPayload {
  model: string
}

export async function listModels(
  params: ListModelsParams = {}
): Promise<PaginatedResponse<ModelCatalogItem>> {
  const { data } = await apiClient.get<PaginatedResponse<ModelCatalogItem>>('/admin/models', {
    params
  })
  return data
}

export async function getModelDetail(model: string): Promise<ModelCatalogDetail> {
  const { data } = await apiClient.get<ModelCatalogDetail>('/admin/models/detail', {
    params: { model }
  })
  return data
}

export async function getExchangeRate(): Promise<ModelCatalogExchangeRate> {
  const { data } = await apiClient.get<ModelCatalogExchangeRate>('/admin/models/exchange-rate')
  return data
}

export async function updateOfficialPricingOverride(
  payload: UpdatePricingOverridePayload
): Promise<ModelCatalogDetail> {
  const { data } = await apiClient.put<ModelCatalogDetail>('/admin/models/official-pricing-override', payload)
  return data
}

export async function deleteOfficialPricingOverride(model: string): Promise<{ model: string }> {
  const { data } = await apiClient.delete<{ model: string }>('/admin/models/official-pricing-override', {
    params: { model }
  })
  return data
}

export async function updatePricingOverride(
  payload: UpdatePricingOverridePayload
): Promise<ModelCatalogDetail> {
  const { data } = await apiClient.put<ModelCatalogDetail>('/admin/models/pricing-override', payload)
  return data
}

export async function deletePricingOverride(model: string): Promise<{ model: string }> {
  const { data } = await apiClient.delete<{ model: string }>('/admin/models/pricing-override', {
    params: { model }
  })
  return data
}

export async function upsertCatalogEntry(payload: UpsertModelCatalogEntryPayload): Promise<ModelCatalogDetail> {
  const { data } = await apiClient.put<ModelCatalogDetail>('/admin/models/catalog-entry', payload)
  return data
}

export async function deleteCatalogEntry(model: string): Promise<{ model: string }> {
  const { data } = await apiClient.delete<{ model: string }>('/admin/models/catalog-entry', {
    params: { model }
  })
  return data
}

export async function copyOfficialPricingToSale(model: string): Promise<ModelCatalogDetail> {
  const { data } = await apiClient.post<ModelCatalogDetail>('/admin/models/pricing-override/copy-from-official', {
    model
  })
  return data
}

export const modelsAPI = {
  listModels,
  getModelDetail,
  getExchangeRate,
  updateOfficialPricingOverride,
  deleteOfficialPricingOverride,
  updatePricingOverride,
  deletePricingOverride,
  upsertCatalogEntry,
  deleteCatalogEntry,
  copyOfficialPricingToSale
}

export default modelsAPI
