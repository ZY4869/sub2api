import { apiClient } from '../client'
import { stepUpHeaders, type AdminStepUpOptions } from './stepUp'
import type {
  PromptAuditConfig,
  PromptAuditProbeResult,
  PromptAuditRuntime,
  PromptAuditUpdateConfig,
  PromptAuditUpdateEndpoint
} from './promptAuditTypes'
import type {
  PromptAuditDeletePreview,
  PromptAuditDeleteResult,
  PromptAuditEvent,
  PromptAuditEventFilter,
  PromptAuditEventList
} from './promptAuditEventTypes'

export type * from './promptAuditTypes'
export type * from './promptAuditEventTypes'

export async function getConfig(): Promise<PromptAuditConfig> {
  const { data } = await apiClient.get<PromptAuditConfig>('/admin/prompt-audit/config')
  return data
}

export async function updateConfig(
  payload: PromptAuditUpdateConfig,
  options?: AdminStepUpOptions
): Promise<PromptAuditConfig> {
  const { data } = await apiClient.put<PromptAuditConfig>('/admin/prompt-audit/config', payload, {
    headers: stepUpHeaders(options)
  })
  return data
}

export async function getRuntime(): Promise<PromptAuditRuntime> {
  const { data } = await apiClient.get<PromptAuditRuntime>('/admin/prompt-audit/runtime')
  return data
}

export async function probeEndpoint(endpoint: PromptAuditUpdateEndpoint): Promise<PromptAuditProbeResult> {
  const { data } = await apiClient.post<PromptAuditProbeResult>('/admin/prompt-audit/endpoints/probe', { endpoint })
  return data
}

export async function listEvents(params: PromptAuditEventFilter): Promise<PromptAuditEventList> {
  const { data } = await apiClient.get<PromptAuditEventList>('/admin/prompt-audit/events', { params })
  return data
}

export async function getEvent(id: number): Promise<PromptAuditEvent> {
  const { data } = await apiClient.get<PromptAuditEvent>(`/admin/prompt-audit/events/${id}`)
  return data
}

export async function deleteEvent(id: number, options?: AdminStepUpOptions): Promise<PromptAuditDeleteResult> {
  const { data } = await apiClient.delete<PromptAuditDeleteResult>(`/admin/prompt-audit/events/${id}`, {
    headers: stepUpHeaders(options)
  })
  return data
}

export async function previewDelete(filter: PromptAuditEventFilter): Promise<PromptAuditDeletePreview> {
  const { data } = await apiClient.post<PromptAuditDeletePreview>('/admin/prompt-audit/events/delete-preview', filter)
  return data
}

export async function deleteByFilter(
  filter: PromptAuditEventFilter,
  preview: PromptAuditDeletePreview,
  options?: AdminStepUpOptions
): Promise<PromptAuditDeleteResult> {
  const { data } = await apiClient.post<PromptAuditDeleteResult>(
    '/admin/prompt-audit/events/delete-by-filter',
    {
      filter,
      snapshot_max_id: preview.snapshot_max_id,
      filter_hash: preview.filter_hash,
      confirmation_token: preview.confirmation_token,
      confirm: true
    },
    { headers: stepUpHeaders(options) }
  )
  return data
}

export default {
  getConfig,
  updateConfig,
  getRuntime,
  probeEndpoint,
  listEvents,
  getEvent,
  deleteEvent,
  previewDelete,
  deleteByFilter
}
