/**
 * Admin API Keys API endpoints
 * Handles API key management for administrators
 */

import { apiClient } from '../client'
import type { ApiKey, ApiKeyGroup, ApiKeyGroupBindingInput } from '@/types'

export interface UpdateApiKeyGroupResult {
  api_key: ApiKey
  auto_granted_group_access: boolean
  granted_group_id?: number
  granted_group_name?: string
}

export interface UpdateApiKeyGroupsPayload {
  group_id?: number | null
  groups?: ApiKeyGroupBindingInput[]
}

/**
 * Update an API key's group binding
 * @param id - API Key ID
 * @param groupId - Group ID (0 to unbind, positive to bind, null/undefined to skip)
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKeyGroup(
  id: number,
  groupIdOrPayload: number | null | UpdateApiKeyGroupsPayload
): Promise<UpdateApiKeyGroupResult> {
  const payload =
    typeof groupIdOrPayload === 'object' && groupIdOrPayload !== null
      ? groupIdOrPayload
      : { group_id: groupIdOrPayload === null ? 0 : groupIdOrPayload }
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, payload)
  return data
}

export async function getApiKeyGroups(id: number): Promise<ApiKeyGroup[]> {
  const { data } = await apiClient.get<ApiKeyGroup[]>(`/admin/api-keys/${id}/groups`)
  return data
}

export const apiKeysAPI = {
  updateApiKeyGroup,
  getApiKeyGroups
}

export default apiKeysAPI
