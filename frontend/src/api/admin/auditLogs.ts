import { apiClient } from '../client'
import { stepUpHeaders, type AdminStepUpOptions } from './stepUp'
import type { PaginatedResponse } from '@/types'

export interface AuditLog {
  id: number
  actor_user_id?: number | null
  actor_role: string
  action: string
  target_type: string
  target_id: string
  status: 'success' | 'denied' | 'failure' | string
  request_id: string
  client_ip: string
  user_agent: string
  metadata?: Record<string, unknown>
  created_at: string
}

export interface AuditLogQuery {
  page?: number
  page_size?: number
  actor_user_id?: number
  action?: string
  target_type?: string
  target_id?: string
  status?: string
  request_id?: string
}

export interface AuditLogCleanupResult {
  deleted: number
  cutoff: string
}

export async function list(
  params: AuditLogQuery,
  options?: { signal?: AbortSignal }
): Promise<PaginatedResponse<AuditLog>> {
  const { data } = await apiClient.get<PaginatedResponse<AuditLog>>('/admin/audit-logs', {
    params,
    signal: options?.signal
  })
  return data
}

export async function cleanupExpired(options?: AdminStepUpOptions): Promise<AuditLogCleanupResult> {
  const headers = stepUpHeaders(options)
  const { data } = await apiClient.post<AuditLogCleanupResult>(
    '/admin/audit-logs/cleanup-expired',
    undefined,
    headers ? { headers } : undefined
  )
  return data
}

const auditLogsAPI = {
  list,
  cleanupExpired
}

export default auditLogsAPI
