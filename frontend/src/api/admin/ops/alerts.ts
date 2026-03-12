import { apiClient } from '../../client'
import type { AlertEvent, AlertEventsQuery, AlertRule } from './types'

export async function listAlertRules(): Promise<AlertRule[]> {
  const { data } = await apiClient.get<AlertRule[]>('/admin/ops/alert-rules')
  return data
}

export async function createAlertRule(rule: AlertRule): Promise<AlertRule> {
  const { data } = await apiClient.post<AlertRule>('/admin/ops/alert-rules', rule)
  return data
}

export async function updateAlertRule(id: number, rule: Partial<AlertRule>): Promise<AlertRule> {
  const { data } = await apiClient.put<AlertRule>(`/admin/ops/alert-rules/${id}`, rule)
  return data
}

export async function deleteAlertRule(id: number): Promise<void> {
  await apiClient.delete(`/admin/ops/alert-rules/${id}`)
}

export async function listAlertEvents(params: AlertEventsQuery = {}): Promise<AlertEvent[]> {
  const { data } = await apiClient.get<AlertEvent[]>('/admin/ops/alert-events', { params })
  return data
}

export async function getAlertEvent(id: number): Promise<AlertEvent> {
  const { data } = await apiClient.get<AlertEvent>(`/admin/ops/alert-events/${id}`)
  return data
}

export async function updateAlertEventStatus(id: number, status: 'resolved' | 'manual_resolved'): Promise<void> {
  await apiClient.put(`/admin/ops/alert-events/${id}/status`, { status })
}

export async function createAlertSilence(payload: {
  rule_id: number
  platform: string
  group_id?: number | null
  region?: string | null
  until: string
  reason?: string
}): Promise<void> {
  await apiClient.post('/admin/ops/alert-silences', payload)
}
