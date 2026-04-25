/**
 * Channel Monitor API endpoints (admin)
 */

import { apiClient } from '../client'

export type ChannelMonitorBodyOverrideMode = 'off' | 'merge' | 'replace'

export interface AdminChannelMonitor {
  id: number
  name: string
  provider: string
  endpoint: string
  interval_seconds: number
  enabled: boolean
  primary_model_id: string
  additional_model_ids: string[]
  template_id?: number
  extra_headers: Record<string, string>
  body_override_mode: ChannelMonitorBodyOverrideMode | string
  body_override: Record<string, any>
  last_run_at?: string
  next_run_at?: string
  api_key_configured: boolean
  api_key_decrypt_failed: boolean
}

export interface AdminChannelMonitorHistory {
  id: number
  monitor_id: number
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
  description?: string
  extra_headers: Record<string, string>
  body_override_mode: ChannelMonitorBodyOverrideMode | string
  body_override: Record<string, any>
  created_at: string
  updated_at: string
}

export interface CreateChannelMonitorRequest {
  name: string
  provider: string
  endpoint: string
  api_key?: string
  interval_seconds?: number
  enabled?: boolean
  primary_model_id: string
  additional_model_ids?: string[]
  template_id?: number | null
  extra_headers?: Record<string, string>
  body_override_mode?: ChannelMonitorBodyOverrideMode
  body_override?: Record<string, any>
}

export interface UpdateChannelMonitorRequest {
  name?: string
  provider?: string
  endpoint?: string
  api_key?: string | null
  interval_seconds?: number
  enabled?: boolean
  primary_model_id?: string
  additional_model_ids?: string[]
  template_id?: number | null
  extra_headers?: Record<string, string>
  body_override_mode?: ChannelMonitorBodyOverrideMode
  body_override?: Record<string, any>
}

export interface CreateChannelMonitorTemplateRequest {
  name: string
  provider: string
  description?: string | null
  extra_headers?: Record<string, string>
  body_override_mode?: ChannelMonitorBodyOverrideMode
  body_override?: Record<string, any>
}

export interface UpdateChannelMonitorTemplateRequest {
  name?: string
  provider?: string
  description?: string | null
  extra_headers?: Record<string, string>
  body_override_mode?: ChannelMonitorBodyOverrideMode
  body_override?: Record<string, any>
}

export async function listMonitors(): Promise<AdminChannelMonitor[]> {
  const { data } = await apiClient.get<AdminChannelMonitor[]>('/admin/channel-monitors')
  return data
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
  const { data } = await apiClient.post<AdminChannelMonitorHistory[]>(`/admin/channel-monitors/${id}/run`)
  return data
}

export async function listMonitorHistories(id: number, limit = 50): Promise<AdminChannelMonitorHistory[]> {
  const { data } = await apiClient.get<AdminChannelMonitorHistory[]>(`/admin/channel-monitors/${id}/histories`, {
    params: { limit }
  })
  return data
}

export async function listTemplates(): Promise<AdminChannelMonitorTemplate[]> {
  const { data } = await apiClient.get<AdminChannelMonitorTemplate[]>('/admin/channel-monitor-templates')
  return data
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
  const { data } = await apiClient.get<Array<{ id: number; name: string }>>(
    `/admin/channel-monitor-templates/${id}/associated-monitors`
  )
  return data
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

