import type { BillingPriceItem } from '@/api/admin/billing'

export const billingChargeSlotOptions = [
  { value: 'text_input', label: 'Input' },
  { value: 'text_output', label: 'Output' },
  { value: 'cache_create', label: 'Cache Create' },
  { value: 'cache_read', label: 'Cache Read' },
  { value: 'cache_storage_token_hour', label: 'Cache Storage' },
  { value: 'image_output', label: 'Image' },
  { value: 'video_request', label: 'Video Request' },
  { value: 'grounding_search_request', label: 'Grounding Search' },
  { value: 'grounding_maps_request', label: 'Grounding Maps' },
  { value: 'file_search_embedding_token', label: 'File Search Embed' },
  { value: 'file_search_retrieval_token', label: 'File Search Retrieval' },
] as const

export const billingModeOptions = [
  { value: 'base', label: 'Base' },
  { value: 'tiered', label: 'Tiered' },
  { value: 'batch', label: 'Batch' },
  { value: 'service_tier', label: 'Service Tier' },
  { value: 'provider_special', label: 'Provider Special' },
] as const

export const billingServiceTierOptions = [
  { value: '', label: 'Default' },
  { value: 'standard', label: 'Standard' },
  { value: 'flex', label: 'Flex' },
  { value: 'priority', label: 'Priority' },
] as const

export const billingBatchModeOptions = [
  { value: '', label: 'Any' },
  { value: 'realtime', label: 'Realtime' },
  { value: 'batch', label: 'Batch' },
] as const

export function defaultUnitForChargeSlot(chargeSlot: string): string {
  switch (chargeSlot) {
    case 'text_input':
    case 'audio_input':
      return 'input_token'
    case 'text_output':
    case 'audio_output':
      return 'output_token'
    case 'cache_create':
      return 'cache_create_token'
    case 'cache_read':
      return 'cache_read_token'
    case 'cache_storage_token_hour':
      return 'cache_storage_token_hour'
    case 'image_output':
      return 'image'
    case 'video_request':
      return 'video_request'
    case 'grounding_search_request':
      return 'grounding_search_request'
    case 'grounding_maps_request':
      return 'grounding_maps_request'
    case 'file_search_embedding_token':
      return 'file_search_embedding_token'
    case 'file_search_retrieval_token':
      return 'file_search_retrieval_token'
    default:
      return 'input_token'
  }
}

export function newBillingPriceItem(layer: 'official' | 'sale', seed?: Partial<BillingPriceItem>): BillingPriceItem {
  const chargeSlot = seed?.charge_slot || 'text_input'
  return {
    id: seed?.id || `item_${Math.random().toString(36).slice(2, 10)}`,
    charge_slot: chargeSlot,
    unit: seed?.unit || defaultUnitForChargeSlot(chargeSlot),
    layer,
    mode: seed?.mode || 'base',
    service_tier: seed?.service_tier || '',
    batch_mode: seed?.batch_mode || '',
    surface: seed?.surface || '',
    operation_type: seed?.operation_type || '',
    input_modality: seed?.input_modality || '',
    output_modality: seed?.output_modality || '',
    cache_phase: seed?.cache_phase || '',
    grounding_kind: seed?.grounding_kind || '',
    context_window: seed?.context_window || '',
    threshold_tokens: seed?.threshold_tokens,
    price: seed?.price ?? 0,
    price_above_threshold: seed?.price_above_threshold,
    formula_source: seed?.formula_source || '',
    formula_multiplier: seed?.formula_multiplier,
    rule_id: seed?.rule_id || '',
    derived_via: seed?.derived_via || '',
    enabled: seed?.enabled ?? true,
  }
}
