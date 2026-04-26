/**
 * Affiliate (Invite Rebate) API endpoints
 * User self-service affiliate info + transfer
 */

import { apiClient } from './client'

export interface AffiliateUserInfo {
  enabled: boolean
  transfer_enabled: boolean
  aff_code: string
  inviter_user_id?: number
  invitee_count: number
  rebate_balance: number
  rebate_frozen_balance: number
  lifetime_rebate: number
  effective_rate_percent: number

  rebate_on_usage_enabled: boolean
  rebate_on_topup_enabled: boolean
  rebate_freeze_hours: number
  rebate_duration_days: number
  rebate_per_invitee_cap: number
}

export interface AffiliateTransferResult {
  transferred_amount: number
  new_balance: number
}

export async function getMyAffiliateInfo(): Promise<AffiliateUserInfo> {
  const { data } = await apiClient.get<AffiliateUserInfo>('/user/aff')
  return data
}

export async function transferToBalance(): Promise<AffiliateTransferResult> {
  const { data } = await apiClient.post<AffiliateTransferResult>('/user/aff/transfer')
  return data
}

export const affiliateAPI = {
  getMyAffiliateInfo,
  transferToBalance
}

export default affiliateAPI

