/**
 * System API endpoints for admin operations
 */

import { apiClient } from '../client'

export interface ReleaseInfo {
  name: string
  body: string
  published_at: string
  html_url: string
  assets?: Array<{ name: string; download_url: string; size: number }>
}

export interface RollbackVersionInfo {
  version: string
  tag_name: string
  published_at: string
  html_url: string
  source: string
  platform: string
  arch: string
  sha256_available: boolean
  signature_available: boolean
}

export interface VersionInfo {
  current_version: string
  latest_version: string
  has_update: boolean
  release_info?: ReleaseInfo
  rollback_versions?: RollbackVersionInfo[]
  cached: boolean
  warning?: string
  build_type: string // "source" for manual builds, "release" for CI builds
}

/**
 * Get current version
 */
export async function getVersion(): Promise<{ version: string }> {
  const { data } = await apiClient.get<{ version: string }>('/admin/system/version')
  return data
}

/**
 * Check for updates
 * @param force - Force refresh from GitHub API
 */
export async function checkUpdates(force = false): Promise<VersionInfo> {
  const { data } = await apiClient.get<VersionInfo>('/admin/system/check-updates', {
    params: force ? { force: 'true' } : undefined
  })
  return data
}

export interface UpdateResult {
  message: string
  need_restart: boolean
}

/**
 * Perform system update
 * Downloads and applies the latest version
 */
export async function performUpdate(): Promise<UpdateResult> {
  const { data } = await apiClient.post<UpdateResult>('/admin/system/update')
  return data
}

/**
 * Rollback to previous backup or a trusted target version.
 */
export async function rollback(targetVersion?: string): Promise<UpdateResult> {
  const payload = targetVersion?.trim() ? { target_version: targetVersion.trim() } : undefined
  const { data } = await apiClient.post<UpdateResult>('/admin/system/rollback', payload)
  return data
}

/**
 * Restart the service
 */
export async function restartService(): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>('/admin/system/restart')
  return data
}

export const systemAPI = {
  getVersion,
  checkUpdates,
  performUpdate,
  rollback,
  restartService
}

export default systemAPI
