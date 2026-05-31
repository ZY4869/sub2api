import type {
  ModelRegistryDetail,
  UpsertModelRegistryEntryPayload,
} from '@/api/admin/modelRegistry'
import type { TimeAccessPolicy } from '@/types/api-key-groups'

export type ModelRegistrySchedulePatch = {
  available_from?: string
  available_until?: string
  access_time_policy?: TimeAccessPolicy | null
}

export function buildModelRegistryScheduleUpsertPayload(
  model: ModelRegistryDetail,
  patch: ModelRegistrySchedulePatch
): UpsertModelRegistryEntryPayload {
  return {
    id: model.id,
    display_name: model.display_name || model.id,
    provider: model.provider || '',
    platforms: [...(model.platforms || [])],
    protocol_ids: [...(model.protocol_ids || [])],
    aliases: [...(model.aliases || [])],
    pricing_lookup_ids: [...(model.pricing_lookup_ids || [])],
    context_window_tokens: model.context_window_tokens,
    preferred_protocol_ids: model.preferred_protocol_ids ? { ...model.preferred_protocol_ids } : undefined,
    modalities: [...(model.modalities || [])],
    capabilities: [...(model.capabilities || [])],
    ui_priority: model.ui_priority || 0,
    exposed_in: [...(model.exposed_in || [])],
    status: model.status,
    available_from: patch.available_from || '',
    available_until: patch.available_until || '',
    access_time_policy: patch.access_time_policy ?? null,
    deprecated_at: model.deprecated_at,
    replaced_by: model.replaced_by,
    deprecation_notice: model.deprecation_notice,
  }
}
