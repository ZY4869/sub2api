import { apiClient } from '../../client'
import type { OpsSystemLogCleanupRequest, OpsSystemLogListResponse, OpsSystemLogQuery, OpsSystemLogSinkHealth } from './types'

export async function listSystemLogs(params: OpsSystemLogQuery): Promise<OpsSystemLogListResponse> {
  const { data } = await apiClient.get<OpsSystemLogListResponse>('/admin/ops/system-logs', { params })
  return data
}

export async function cleanupSystemLogs(payload: OpsSystemLogCleanupRequest): Promise<{ deleted: number }> {
  const { data } = await apiClient.post<{ deleted: number }>('/admin/ops/system-logs/cleanup', payload)
  return data
}

export async function getSystemLogSinkHealth(): Promise<OpsSystemLogSinkHealth> {
  const { data } = await apiClient.get<OpsSystemLogSinkHealth>('/admin/ops/system-logs/health')
  return data
}
