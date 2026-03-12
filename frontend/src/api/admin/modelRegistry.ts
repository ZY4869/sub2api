import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'
import type { ModelRegistryEntry } from '@/generated/modelRegistry'

export interface ModelRegistryDetail extends ModelRegistryEntry {
  source: string
  hidden: boolean
  tombstoned: boolean
}

export interface ListModelRegistryParams {
  search?: string
  provider?: string
  platform?: string
  include_hidden?: boolean
  include_tombstoned?: boolean
  page?: number
  page_size?: number
}

export interface UpsertModelRegistryEntryPayload extends ModelRegistryEntry {}

export interface UpdateModelRegistryVisibilityPayload {
  model: string
  hidden: boolean
}

export async function listModelRegistry(
  params: ListModelRegistryParams = {}
): Promise<PaginatedResponse<ModelRegistryDetail>> {
  const { data } = await apiClient.get<PaginatedResponse<ModelRegistryDetail>>('/admin/models/registry', {
    params
  })
  return data
}

export async function getModelRegistryDetail(model: string): Promise<ModelRegistryDetail> {
  const { data } = await apiClient.get<ModelRegistryDetail>('/admin/models/registry/detail', {
    params: { model }
  })
  return data
}

export async function upsertModelRegistryEntry(
  payload: UpsertModelRegistryEntryPayload
): Promise<ModelRegistryDetail> {
  const { data } = await apiClient.put<ModelRegistryDetail>('/admin/models/registry/entry', payload)
  return data
}

export async function updateModelRegistryVisibility(
  payload: UpdateModelRegistryVisibilityPayload
): Promise<ModelRegistryDetail> {
  const { data } = await apiClient.post<ModelRegistryDetail>('/admin/models/registry/visibility', payload)
  return data
}

export async function deleteModelRegistryEntry(model: string): Promise<{ model: string }> {
  const { data } = await apiClient.delete<{ model: string }>('/admin/models/registry/entry', {
    params: { model }
  })
  return data
}

export const modelRegistryAPI = {
  listModelRegistry,
  getModelRegistryDetail,
  upsertModelRegistryEntry,
  updateModelRegistryVisibility,
  deleteModelRegistryEntry
}

export default modelRegistryAPI
