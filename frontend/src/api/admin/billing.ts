import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'
import type { PublicModelCatalogItem, PublicModelCatalogPriceDisplay } from '@/api/meta'

export type BillingPricingCurrency = 'USD' | 'CNY'
export type BillingPricingSortBy = 'display_name' | 'provider'
export type BillingPricingSortOrder = 'asc' | 'desc'
export type BillingPricingMultiplierMode = 'shared' | 'item'
export type BillingPricingStatus = 'ok' | 'fallback' | 'conflict' | 'missing'

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
  formula_source?: string
  formula_multiplier?: number
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
  base_price?: number
  price: number
  cost: number
  actual_cost: number
  formula_source?: string
  formula_multiplier?: number
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
  formula_source?: string
  formula_multiplier?: number
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
  multiplier_enabled: boolean
  multiplier_mode?: BillingPricingMultiplierMode
  shared_multiplier?: number
  item_multipliers?: Record<string, number>
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
  currency?: BillingPricingCurrency
  price_item_count: number
  official_count: number
  sale_count: number
  pricing_status: BillingPricingStatus
  pricing_warnings?: string[]
  capabilities: BillingPricingCapabilities
  preview_group_id?: number | null
  preview_rate_multiplier?: number | null
  preview_price_display?: PublicModelCatalogPriceDisplay | null
}

export interface BillingPricingSheetDetail {
  model: string
  display_name?: string
  provider?: string
  mode?: string
  currency: BillingPricingCurrency
  pricing_status: BillingPricingStatus
  pricing_warnings?: string[]
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
  preview_group_id?: number | null
  preview_rate_multiplier?: number | null
  preview_sale_form?: BillingPricingLayerForm | null
}

export interface BillingPricingListParams {
  search?: string
  provider?: string
  mode?: string
  pricing_status?: BillingPricingStatus
  group_id?: number
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

export interface BillingPricingIdentifierCollision {
  source: string
  identifier: string
  models: string[]
  count: number
}

export interface BillingPricingStatusCounts {
  ok: number
  fallback: number
  conflict: number
  missing: number
}

export interface BillingPricingCollisionCountsBySource {
  aliases: number
  protocol_ids: number
  pricing_lookup_ids: number
}

export interface BillingPricingProviderIssueCount {
  provider: string
  total: number
  fallback: number
  conflict: number
  missing: number
}

export interface BillingPricingIssueExample {
  model: string
  display_name?: string
  provider?: string
  pricing_status: BillingPricingStatus
  first_warning?: string
}

export interface BillingPricingAudit {
  total_models: number
  pricing_status_counts: BillingPricingStatusCounts
  duplicate_model_ids: string[]
  aux_identifier_collisions: BillingPricingIdentifierCollision[]
  collision_counts_by_source: BillingPricingCollisionCountsBySource
  provider_issue_counts: BillingPricingProviderIssueCount[]
  pricing_issue_examples: BillingPricingIssueExample[]
  missing_in_snapshot_count: number
  missing_in_snapshot_models: string[]
  snapshot_only_count: number
  snapshot_only_models: string[]
  refresh_required: boolean
  snapshot_updated_at?: string | null
}

export interface BillingSavePricingLayerPayload {
  form: BillingPricingLayerForm
  currency?: BillingPricingCurrency
  group_id?: number | null
}

export interface BillingCopyOfficialToSalePayload {
  models: string[]
}

export interface BillingBulkDiscountPayload {
  models: string[]
  item_ids?: string[]
  discount_ratio: number
}

export interface BillingPricingDetailsPayload {
  models: string[]
  group_id?: number | null
}

export interface BillingPublicCatalogDraft {
  selected_models: string[]
  page_size: number
  updated_at?: string
}

export interface BillingPublicCatalogPublishedSummary {
  etag: string
  updated_at: string
  page_size: number
  model_count: number
}

export interface BillingPublicCatalogDraftPayload {
  draft: BillingPublicCatalogDraft
  available_items: PublicModelCatalogItem[]
  available_updated_at?: string
  available_source?: string
  published?: BillingPublicCatalogPublishedSummary | null
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
  const { data } = await apiClient.post<BillingPricingSheetDetail[]>(
    '/admin/billing/pricing/details',
    { models }
  )
  return data
}

export async function getBillingPricingDetailsWithPreview(
  payload: BillingPricingDetailsPayload
): Promise<BillingPricingSheetDetail[]> {
  const { data } = await apiClient.post<BillingPricingSheetDetail[]>(
    '/admin/billing/pricing/details',
    payload
  )
  return data
}

export async function refreshBillingPricingCatalog(): Promise<BillingPricingRefreshResult> {
  const { data } = await apiClient.post<BillingPricingRefreshResult>('/admin/billing/pricing/refresh')
  return data
}

export async function getBillingPricingAudit(): Promise<BillingPricingAudit> {
  const { data } = await apiClient.get<BillingPricingAudit>('/admin/billing/pricing/audit')
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

export async function getBillingPublicCatalogDraft(options: { force?: boolean } = {}): Promise<BillingPublicCatalogDraftPayload> {
  const params = options.force ? { force: 1 } : undefined
  const { data } = await apiClient.get<BillingPublicCatalogDraftPayload>('/admin/billing/public-model-catalog/draft', {
    params,
  })
  return data
}

export async function saveBillingPublicCatalogDraft(
  draft: BillingPublicCatalogDraft
): Promise<BillingPublicCatalogDraft> {
  const { data } = await apiClient.put<BillingPublicCatalogDraft>('/admin/billing/public-model-catalog/draft', draft)
  return data
}

export async function publishBillingPublicCatalog(
  draft?: BillingPublicCatalogDraft,
): Promise<BillingPublicCatalogPublishedSummary> {
  const { data } = await apiClient.post<BillingPublicCatalogPublishedSummary>(
    '/admin/billing/public-model-catalog/publish',
    draft,
  )
  return data
}

export async function getBillingPublicCatalogPublishedSummary(): Promise<BillingPublicCatalogPublishedSummary | null> {
  const { data } = await apiClient.get<BillingPublicCatalogPublishedSummary | null>('/admin/billing/public-model-catalog/published')
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
