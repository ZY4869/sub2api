import { apiClient } from '../client'
import type { ContentModerationAudit, PaginatedResponse } from '@/types'

export interface ContentModerationAuditQuery {
  page?: number
  page_size?: number
  request_id?: string
  client_request_id?: string
  provider?: string
  model?: string
  source_endpoint?: string
  content_hash?: string
  hit?: boolean
  user_id?: number
}

export async function listAudits(
  params: ContentModerationAuditQuery,
  options?: { signal?: AbortSignal }
): Promise<PaginatedResponse<ContentModerationAudit>> {
  const { data } = await apiClient.get<PaginatedResponse<ContentModerationAudit>>(
    '/admin/moderation/audits',
    {
      params,
      signal: options?.signal
    }
  )
  return data
}

export async function getAuditDetail(id: number): Promise<ContentModerationAudit> {
  const { data } = await apiClient.get<ContentModerationAudit>(`/admin/moderation/audits/${id}`)
  return data
}

const moderationAPI = {
  listAudits,
  getAuditDetail
}

export default moderationAPI
