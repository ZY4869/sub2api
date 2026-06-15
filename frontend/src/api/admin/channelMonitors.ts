/**
 * Channel Monitor API endpoints (admin)
 */

import { apiClient } from '../client'

export type ChannelMonitorBodyOverrideMode = 'off' | 'merge' | 'replace'
export type ChannelMonitorOpenAIAPIMode = 'chat_completions' | 'responses'
export type ChannelMonitorProbeMode = 'direct' | 'account_pool'
export type ChannelMonitorRequestProtocol = 'openai' | 'anthropic' | 'gemini'
export type ChannelMonitorModelProbeStrategy = 'primary_only' | 'all_selected'

export interface AdminChannelMonitor {
  id: number
  name: string
  provider: string
  probe_mode: ChannelMonitorProbeMode | string
  request_protocol: ChannelMonitorRequestProtocol | string
  endpoint: string
  interval_seconds: number
  enabled: boolean
  account_ids: number[]
  primary_model_id: string
  additional_model_ids: string[]
  model_source_protocols?: Record<string, ChannelMonitorRequestProtocol | string>
  model_probe_strategy: ChannelMonitorModelProbeStrategy | string
  test_prompt_template: string
  template_id?: number
  extra_headers: Record<string, string>
  body_override_mode: ChannelMonitorBodyOverrideMode | string
  body_override: Record<string, any>
  openai_api_mode: ChannelMonitorOpenAIAPIMode | string
  last_run_at?: string
  next_run_at?: string
  api_key_configured: boolean
  api_key_decrypt_failed: boolean
}

export interface AdminChannelMonitorHistory {
  id: number
  monitor_id: number
  account_id?: number
  account_name_snapshot?: string
  probe_mode?: ChannelMonitorProbeMode | string
  model_id: string
  status: string
  response_text: string
  error_message: string
  http_status?: number
  latency_ms: number
  started_at: string
  finished_at: string
  created_at: string
}

export interface AdminChannelMonitorTemplate {
  id: number
  name: string
  provider: string
  request_protocol: ChannelMonitorRequestProtocol | string
  description?: string
  extra_headers: Record<string, string>
  body_override_mode: ChannelMonitorBodyOverrideMode | string
  body_override: Record<string, any>
  openai_api_mode: ChannelMonitorOpenAIAPIMode | string
  test_prompt_template: string
  created_at: string
  updated_at: string
}

export interface CreateChannelMonitorRequest {
  name: string
  provider: string
  probe_mode?: ChannelMonitorProbeMode
  request_protocol?: ChannelMonitorRequestProtocol
  endpoint?: string
  api_key?: string
  interval_seconds?: number
  enabled?: boolean
  account_ids?: number[]
  primary_model_id: string
  additional_model_ids?: string[]
  model_source_protocols?: Record<string, ChannelMonitorRequestProtocol | string>
  model_probe_strategy?: ChannelMonitorModelProbeStrategy
  test_prompt_template?: string
  template_id?: number | null
  extra_headers?: Record<string, string>
  body_override_mode?: ChannelMonitorBodyOverrideMode
  body_override?: Record<string, any>
  openai_api_mode?: ChannelMonitorOpenAIAPIMode
  save_as_template?: boolean
  template_name?: string
}

export interface UpdateChannelMonitorRequest {
  name?: string
  provider?: string
  probe_mode?: ChannelMonitorProbeMode
  request_protocol?: ChannelMonitorRequestProtocol
  endpoint?: string
  api_key?: string | null
  interval_seconds?: number
  enabled?: boolean
  account_ids?: number[]
  primary_model_id?: string
  additional_model_ids?: string[]
  model_source_protocols?: Record<string, ChannelMonitorRequestProtocol | string>
  model_probe_strategy?: ChannelMonitorModelProbeStrategy
  test_prompt_template?: string
  template_id?: number | null
  extra_headers?: Record<string, string>
  body_override_mode?: ChannelMonitorBodyOverrideMode
  body_override?: Record<string, any>
  openai_api_mode?: ChannelMonitorOpenAIAPIMode
}

export interface CreateChannelMonitorTemplateRequest {
  name: string
  provider: string
  request_protocol?: ChannelMonitorRequestProtocol
  description?: string | null
  extra_headers?: Record<string, string>
  body_override_mode?: ChannelMonitorBodyOverrideMode
  body_override?: Record<string, any>
  openai_api_mode?: ChannelMonitorOpenAIAPIMode
  test_prompt_template?: string
}

export interface UpdateChannelMonitorTemplateRequest {
  name?: string
  provider?: string
  request_protocol?: ChannelMonitorRequestProtocol
  description?: string | null
  extra_headers?: Record<string, string>
  body_override_mode?: ChannelMonitorBodyOverrideMode
  body_override?: Record<string, any>
  openai_api_mode?: ChannelMonitorOpenAIAPIMode
  test_prompt_template?: string
}

export async function listMonitors(): Promise<AdminChannelMonitor[]> {
  const { data } = await apiClient.get<AdminChannelMonitor[] | null>('/admin/channel-monitors')
  return Array.isArray(data) ? data : []
}

export async function createMonitor(payload: CreateChannelMonitorRequest): Promise<AdminChannelMonitor> {
  const { data } = await apiClient.post<AdminChannelMonitor>('/admin/channel-monitors', payload)
  return data
}

export async function updateMonitor(id: number, payload: UpdateChannelMonitorRequest): Promise<AdminChannelMonitor> {
  const { data } = await apiClient.put<AdminChannelMonitor>(`/admin/channel-monitors/${id}`, payload)
  return data
}

export async function deleteMonitor(id: number): Promise<void> {
  await apiClient.delete(`/admin/channel-monitors/${id}`)
}

export async function runMonitor(id: number): Promise<AdminChannelMonitorHistory[]> {
  const { data } = await apiClient.post<AdminChannelMonitorHistory[] | null>(`/admin/channel-monitors/${id}/run`)
  return Array.isArray(data) ? data : []
}

export async function listMonitorHistories(id: number, limit = 50): Promise<AdminChannelMonitorHistory[]> {
  const { data } = await apiClient.get<AdminChannelMonitorHistory[] | null>(`/admin/channel-monitors/${id}/histories`, {
    params: { limit }
  })
  return Array.isArray(data) ? data : []
}

export async function listTemplates(): Promise<AdminChannelMonitorTemplate[]> {
  const { data } = await apiClient.get<AdminChannelMonitorTemplate[] | null>('/admin/channel-monitor-templates')
  return Array.isArray(data) ? data : []
}

export async function createTemplate(payload: CreateChannelMonitorTemplateRequest): Promise<AdminChannelMonitorTemplate> {
  const { data } = await apiClient.post<AdminChannelMonitorTemplate>('/admin/channel-monitor-templates', payload)
  return data
}

export async function updateTemplate(id: number, payload: UpdateChannelMonitorTemplateRequest): Promise<AdminChannelMonitorTemplate> {
  const { data } = await apiClient.put<AdminChannelMonitorTemplate>(`/admin/channel-monitor-templates/${id}`, payload)
  return data
}

export async function deleteTemplate(id: number): Promise<void> {
  await apiClient.delete(`/admin/channel-monitor-templates/${id}`)
}

export async function applyTemplate(id: number, monitorId: number): Promise<AdminChannelMonitor> {
  const { data } = await apiClient.post<AdminChannelMonitor>(`/admin/channel-monitor-templates/${id}/apply`, {
    monitor_id: monitorId
  })
  return data
}

export async function listAssociatedMonitors(id: number): Promise<Array<{ id: number; name: string }>> {
  const { data } = await apiClient.get<Array<{ id: number; name: string }> | null>(
    `/admin/channel-monitor-templates/${id}/associated-monitors`
  )
  return Array.isArray(data) ? data : []
}

export const channelMonitorsAdminAPI = {
  listMonitors,
  createMonitor,
  updateMonitor,
  deleteMonitor,
  runMonitor,
  listMonitorHistories,
  listTemplates,
  createTemplate,
  updateTemplate,
  deleteTemplate,
  applyTemplate,
  listAssociatedMonitors
}

export default channelMonitorsAdminAPI
