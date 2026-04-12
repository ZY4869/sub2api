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
  output_cost_per_video_request?: number
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

export interface BillingRuleMatchers {
  models?: string[]
  model_families?: string[]
  input_modality?: string
  output_modality?: string
  cache_phase?: string
  grounding_kind?: string
  context_window?: string
}

export interface BillingRule {
  id: string
  provider: string
  layer: string
  surface: string
  operation_type: string
  service_tier: string
  batch_mode: string
  matchers: BillingRuleMatchers
  unit: string
  price: number
  priority: number
  enabled: boolean
}

export interface ModelBillingSheet {
  id: string
  provider: string
  model: string
  model_family?: string
  display_name?: string
  official_pricing?: ModelCatalogPricing
  sale_pricing?: ModelCatalogPricing
  official_matrix?: GeminiBillingMatrix
  sale_matrix?: GeminiBillingMatrix
  supports_service_tier: boolean
  long_context_input_token_threshold?: number
  long_context_input_cost_multiplier?: number
  long_context_output_cost_multiplier?: number
}

export interface GeminiBillingMatrixCell {
  price?: number
  rule_id?: string
  derived_via?: string
}

export interface GeminiBillingMatrixRow {
  surface: string
  service_tier: string
  slots: Record<string, GeminiBillingMatrixCell>
}

export interface GeminiBillingMatrix {
  surfaces: string[]
  service_tiers: string[]
  charge_slots: string[]
  rows: GeminiBillingMatrixRow[]
}

export interface BillingCenterPayload {
  sheets: ModelBillingSheet[]
  rules: BillingRule[]
}

export interface GeminiRequestClassification {
  surface: string
  operation_type: string
  service_tier?: string
  batch_mode?: string
  input_modality?: string
  output_modality?: string
  cache_phase?: string
  grounding_kind?: string
  charge_source?: string
  media_type?: string
  media_units?: number
}

export interface BillingSimulationInput {
  provider: string
  layer: string
  model: string
  surface: string
  operation_type: string
  service_tier: string
  batch_mode: string
  input_modality: string
  output_modality: string
  cache_phase: string
  grounding_kind: string
  charges?: BillingSimulationCharges
  input_tokens?: number
  output_tokens?: number
  cache_creation_tokens?: number
  cache_read_tokens?: number
  image_count?: number
  video_requests?: number
  media_units?: number
}

export interface BillingSimulationCharges {
  text_input_tokens?: number
  text_output_tokens?: number
  audio_input_tokens?: number
  audio_output_tokens?: number
  cache_create_tokens?: number
  cache_read_tokens?: number
  cache_storage_token_hours?: number
  image_outputs?: number
  video_requests?: number
  file_search_embedding_tokens?: number
  file_search_retrieval_tokens?: number
  grounding_search_queries?: number
  grounding_maps_queries?: number
}

export interface BillingSimulationLine {
  charge_slot: string
  unit: string
  units: number
  price: number
  cost: number
  actual_cost: number
  rule_id?: string
  rule_label?: string
}

export interface BillingSimulationMatchedRule {
  id: string
  provider: string
  layer: string
  surface: string
  operation_type: string
  service_tier: string
  batch_mode: string
  unit: string
  price: number
  priority: number
  matchers: BillingRuleMatchers
}

export interface BillingSimulationUnmatchedDemand {
  charge_slot: string
  unit: string
  units: number
  reason: string
  missing_dimensions?: string[]
}

export interface BillingSimulationFallback {
  policy?: string
  applied: boolean
  reason?: string
  derived_from?: string
  cost_lines?: BillingSimulationLine[]
}

export interface BillingSimulationResult {
  classification?: GeminiRequestClassification
  matched_rules?: BillingSimulationMatchedRule[]
  matched_rule_ids?: string[]
  lines: BillingSimulationLine[]
  unmatched_demands?: BillingSimulationUnmatchedDemand[]
  fallback?: BillingSimulationFallback
  total_cost: number
  actual_cost: number
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

export interface UpsertBillingSheetPayload {
  model: string
  layer: string
  pricing?: ModelCatalogPricing
  matrix?: GeminiBillingMatrix
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

export async function getBillingCenter(): Promise<BillingCenterPayload> {
  const { data } = await apiClient.get<BillingCenterPayload>('/admin/models/billing')
  return data
}

export async function updateBillingSheet(payload: UpsertBillingSheetPayload): Promise<ModelBillingSheet> {
  const { data } = await apiClient.put<ModelBillingSheet>('/admin/models/billing/sheets', payload)
  return data
}

export async function deleteBillingSheet(
  model: string,
  layer: string
): Promise<{ model: string; layer: string }> {
  const { data } = await apiClient.delete<{ model: string; layer: string }>('/admin/models/billing/sheets', {
    params: { model, layer }
  })
  return data
}

export async function updateBillingRule(payload: BillingRule): Promise<BillingRule> {
  const { data } = await apiClient.put<BillingRule>('/admin/models/billing/rules', payload)
  return data
}

export async function deleteBillingRule(id: string): Promise<{ id: string }> {
  const { data } = await apiClient.delete<{ id: string }>('/admin/models/billing/rules', {
    params: { id }
  })
  return data
}

export async function simulateBilling(payload: BillingSimulationInput): Promise<BillingSimulationResult> {
  const { data } = await apiClient.post<BillingSimulationResult>('/admin/models/billing/simulate', payload)
  return data
}

export async function copyBillingSheetOfficialToSale(model: string): Promise<ModelBillingSheet> {
  const { data } = await apiClient.post<ModelBillingSheet>('/admin/models/billing/sheets/copy-official-to-sale', {
    model
  })
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
  getBillingCenter,
  updateBillingSheet,
  deleteBillingSheet,
  updateBillingRule,
  deleteBillingRule,
  simulateBilling,
  copyBillingSheetOfficialToSale,
  updateOfficialPricingOverride,
  deleteOfficialPricingOverride,
  updatePricingOverride,
  deletePricingOverride,
  copyOfficialPricingToSale
}

export default modelsAPI
