import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export type BillingPricingCurrency = 'USD' | 'CNY'
export type BillingPricingSortBy = 'display_name' | 'provider'
export type BillingPricingSortOrder = 'asc' | 'desc'

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

export interface BillingPricingSimpleSpecial {
  batch_input_price?: number
  batch_output_price?: number
  batch_cache_price?: number
  grounding_search?: number
  grounding_maps?: number
  file_search_embedding?: number
  file_search_retrieval?: number
}

export interface BillingPricingLayerForm {
  input_price?: number
  output_price?: number
  cache_price?: number
  special_enabled: boolean
  special: BillingPricingSimpleSpecial
  tiered_enabled: boolean
  tier_threshold_tokens?: number
  input_price_above_threshold?: number
  output_price_above_threshold?: number
}

export interface BillingPricingCapabilities {
  supports_tiered_pricing: boolean
  supports_batch_pricing: boolean
  supports_service_tier: boolean
  supports_prompt_caching: boolean
  supports_provider_special: boolean
}

export interface BillingPricingProviderGroup {
  provider: string
  label: string
  total_count: number
  official_count: number
  sale_count: number
}

export interface BillingPricingListItem {
  model: string
  display_name?: string
  provider?: string
  mode?: string
  price_item_count: number
  official_count: number
  sale_count: number
  capabilities: BillingPricingCapabilities
}

export interface BillingPricingSheetDetail {
  model: string
  display_name?: string
  provider?: string
  mode?: string
  currency: BillingPricingCurrency
  input_supported: boolean
  output_charge_slot?: string
  supports_prompt_caching: boolean
  supports_service_tier: boolean
  long_context_input_token_threshold?: number
  long_context_input_cost_multiplier?: number
  long_context_output_cost_multiplier?: number
  capabilities: BillingPricingCapabilities
  official_form: BillingPricingLayerForm
  sale_form: BillingPricingLayerForm
}

export interface BillingPricingListParams {
  search?: string
  provider?: string
  mode?: string
  sort_by?: BillingPricingSortBy
  sort_order?: BillingPricingSortOrder
  page?: number
  page_size?: number
}

export interface BillingPricingRefreshResult {
  updated_at: string
  total_models: number
  provider_count: number
}

export interface BillingSavePricingLayerPayload {
  form: BillingPricingLayerForm
  currency?: BillingPricingCurrency
}

export interface BillingCopyOfficialToSalePayload {
  models: string[]
}

export interface BillingBulkDiscountPayload {
  models: string[]
  item_ids?: string[]
  discount_ratio: number
}

export async function listBillingPricingProviders(): Promise<BillingPricingProviderGroup[]> {
  const { data } = await apiClient.get<BillingPricingProviderGroup[]>('/admin/billing/pricing/providers')
  return data
}

export async function listBillingPricingModels(
  params: BillingPricingListParams = {},
): Promise<PaginatedResponse<BillingPricingListItem>> {
  const { data } = await apiClient.get<PaginatedResponse<BillingPricingListItem>>('/admin/billing/pricing/models', {
    params,
  })
  return data
}

export async function getBillingPricingDetails(models: string[]): Promise<BillingPricingSheetDetail[]> {
  const { data } = await apiClient.post<BillingPricingSheetDetail[]>('/admin/billing/pricing/details', {
    models,
  })
  return data
}

export async function refreshBillingPricingCatalog(): Promise<BillingPricingRefreshResult> {
  const { data } = await apiClient.post<BillingPricingRefreshResult>('/admin/billing/pricing/refresh')
  return data
}

export async function updateBillingPricingLayer(
  model: string,
  layer: 'official' | 'sale',
  payload: BillingSavePricingLayerPayload,
): Promise<BillingPricingSheetDetail> {
  const { data } = await apiClient.put<BillingPricingSheetDetail>(`/admin/billing/pricing/models/${encodeURIComponent(model)}/layers/${layer}`, payload)
  return data
}

export async function copyBillingPricingOfficialToSale(
  payload: BillingCopyOfficialToSalePayload,
): Promise<BillingPricingSheetDetail[]> {
  const { data } = await apiClient.post<BillingPricingSheetDetail[]>('/admin/billing/pricing/sale/copy-from-official', payload)
  return data
}

export async function applyBillingPricingDiscount(
  payload: BillingBulkDiscountPayload,
): Promise<BillingPricingSheetDetail[]> {
  const { data } = await apiClient.post<BillingPricingSheetDetail[]>('/admin/billing/pricing/sale/apply-discount', payload)
  return data
}

export async function listBillingRules(): Promise<BillingRule[]> {
  const { data } = await apiClient.get<BillingRule[]>('/admin/billing/rules')
  return data
}

export async function updateBillingRule(payload: BillingRule): Promise<BillingRule> {
  const { data } = await apiClient.put<BillingRule>('/admin/billing/rules', payload)
  return data
}

export async function deleteBillingRule(id: string): Promise<{ id: string }> {
  const { data } = await apiClient.delete<{ id: string }>('/admin/billing/rules', {
    params: { id },
  })
  return data
}

export async function simulateBilling(payload: BillingSimulationInput): Promise<BillingSimulationResult> {
  const { data } = await apiClient.post<BillingSimulationResult>('/admin/billing/rules/simulate', payload)
  return data
}
