/**
 * Admin Affiliate API endpoints
 * Manage per-user custom affiliate code + rebate rate.
 */

import { apiClient } from '../client'
import type { BasePaginationResponse } from '@/types'

export interface AffiliateAdminUser {
  user_id: number
  email: string
  aff_code: string
  custom_aff_code: boolean
  custom_rebate_rate_percent?: number
  inviter_user_id?: number
  invitee_count: number
  rebate_balance: number
  rebate_frozen_balance: number
  lifetime_rebate: number
  updated_at: string
}

export interface ListAffiliateUsersParams {
  page?: number
  page_size?: number
  has_custom_code?: boolean
  has_custom_rate?: boolean
  has_inviter?: boolean
}

export async function listAffiliateUsers(
  params: ListAffiliateUsersParams = {}
): Promise<BasePaginationResponse<AffiliateAdminUser>> {
  const { data } = await apiClient.get<BasePaginationResponse<AffiliateAdminUser>>(
    '/admin/affiliates/users',
    { params }
  )
  return data
}

export async function lookupAffiliateUsers(q: string, limit = 20): Promise<AffiliateAdminUser[]> {
  const { data } = await apiClient.get<AffiliateAdminUser[]>('/admin/affiliates/users/lookup', {
    params: { q, limit }
  })
  return Array.isArray(data) ? data : []
}

export interface UpdateAffiliateUserCustomRequest {
  aff_code?: string
  custom_rebate_rate_percent?: number | null
}

export async function updateAffiliateUserCustom(
  userId: number,
  req: UpdateAffiliateUserCustomRequest
): Promise<any> {
  const { data } = await apiClient.put<any>(`/admin/affiliates/users/${userId}`, req)
  return data
}

export async function resetAffiliateUserCustom(userId: number): Promise<any> {
  const { data } = await apiClient.delete<any>(`/admin/affiliates/users/${userId}`)
  return data
}

export async function batchUpdateAffiliateUserRates(
  userIds: number[],
  customRebateRatePercent: number
): Promise<{ updated: number }> {
  const { data } = await apiClient.post<{ updated: number }>(
    '/admin/affiliates/users/batch-rate',
    {
      user_ids: userIds,
      custom_rebate_rate_percent: customRebateRatePercent
    }
  )
  return data
}

export const affiliatesAPI = {
  listAffiliateUsers,
  lookupAffiliateUsers,
  updateAffiliateUserCustom,
  resetAffiliateUserCustom,
  batchUpdateAffiliateUserRates
}

export default affiliatesAPI

