/**
 * User Groups API endpoints (non-admin)
 * Handles group-related operations for regular users
 */

import { apiClient } from './client'
import type { PublicModelCatalogSnapshot } from '@/api/meta'
import type {
  EffectiveExternalModelCatalogViewMode,
  ExternalModelCatalogViewMode,
  Group,
  GroupPlatform,
  UserGroupModelOptionGroup
} from '@/types'
import type { PublicModelCatalogItem, PublicModelCatalogSource } from '@/api/meta'

export interface ExternalModelCatalogGroupSummary {
  id: number
  name: string
  description?: string | null
  platform: GroupPlatform | string
  priority: number
  model_count: number
}

export interface ExternalModelCatalogView {
  external_model_catalog_view_mode: ExternalModelCatalogViewMode
  effective_external_model_catalog_view_mode: EffectiveExternalModelCatalogViewMode
  etag?: string
  updated_at?: string
  published_at?: string
  last_revalidated_at?: string
  stale_reason?: string
  page_size?: number
  catalog_source?: PublicModelCatalogSource
  groups: ExternalModelCatalogGroupSummary[]
  items: PublicModelCatalogItem[]
  group_catalogs?: Record<string, PublicModelCatalogItem[]>
}

/**
 * Get available groups that the current user can bind to API keys
 * This returns groups based on user's permissions:
 * - Standard groups: public (non-exclusive) or explicitly allowed
 * - Subscription groups: user has active subscription
 * @returns List of available groups
 */
export async function getAvailable(): Promise<Group[]> {
  const { data } = await apiClient.get<Group[]>('/groups/available')
  return data
}

/**
 * Get current user's custom group rate multipliers
 * @returns Map of group_id to custom rate_multiplier
 */
export async function getUserGroupRates(): Promise<Record<number, number>> {
  const { data } = await apiClient.get<Record<number, number> | null>('/groups/rates')
  return data || {}
}

export async function getModelOptions(): Promise<UserGroupModelOptionGroup[]> {
  const { data } = await apiClient.get<UserGroupModelOptionGroup[]>('/groups/model-options')
  return data
}

export async function getModelCatalog(groupId: number): Promise<PublicModelCatalogSnapshot> {
  const { data } = await apiClient.get<PublicModelCatalogSnapshot>('/groups/model-catalog', {
    params: { group_id: groupId },
  })
  return data
}

export async function getExternalModelCatalog(): Promise<ExternalModelCatalogView> {
  const { data } = await apiClient.get<ExternalModelCatalogView>('/user/external-model-catalog')
  return data
}

export const userGroupsAPI = {
  getAvailable,
  getExternalModelCatalog,
  getModelCatalog,
  getModelOptions,
  getUserGroupRates
}

export default userGroupsAPI
