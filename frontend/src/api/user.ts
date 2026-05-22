/**
 * User API endpoints
 * Handles user profile management and password changes
 */

import { apiClient } from './client'
import type {
  User,
  ChangePasswordRequest,
  UsageContextBadgeDisplayMode,
  UsageModelDisplayMode,
  VisualPresetPreference,
  AuthIdentity,
  SocialOAuthProvider,
} from '@/types'

/**
 * Get current user profile
 * @returns User profile data
 */
export async function getProfile(): Promise<User> {
  const { data } = await apiClient.get<User>('/user/profile')
  return data
}

/**
 * Update current user profile
 * @param profile - Profile data to update
 * @returns Updated user profile data
 */
export async function updateProfile(profile: {
  username?: string
  usage_model_display_mode?: UsageModelDisplayMode
  usage_context_badge_display_mode?: UsageContextBadgeDisplayMode
  global_realtime_countdown_enabled?: boolean
  account_realtime_countdown_enabled?: boolean
  visual_preset_preference?: VisualPresetPreference
  account_visual_preset_override?: VisualPresetPreference
}): Promise<User> {
  const { data } = await apiClient.put<User>('/user', profile)
  return data
}

/**
 * Change current user password
 * @param passwords - Old and new password
 * @returns Success message
 */
export async function changePassword(
  oldPassword: string,
  newPassword: string
): Promise<{ message: string }> {
  const payload: ChangePasswordRequest = {
    old_password: oldPassword,
    new_password: newPassword
  }

  const { data } = await apiClient.put<{ message: string }>('/user/password', payload)
  return data
}

export async function getAuthIdentities(): Promise<AuthIdentity[]> {
  const { data } = await apiClient.get<AuthIdentity[]>('/user/auth-identities')
  return data
}

export async function deleteAuthIdentity(provider: SocialOAuthProvider): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/user/auth-identities/${provider}`)
  return data
}

export const userAPI = {
  getProfile,
  updateProfile,
  changePassword,
  getAuthIdentities,
  deleteAuthIdentity
}

export default userAPI
