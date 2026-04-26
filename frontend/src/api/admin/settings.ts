/**
 * Admin Settings API endpoints
 * Handles system settings management for administrators
 */

import { apiClient } from '../client'
import type { CustomMenuItem } from '@/types'

export interface DefaultSubscriptionSetting {
  group_id: number
  validity_days: number
}

/**
 * System settings interface
 */
export interface SystemSettings {
  // Registration settings
  registration_enabled: boolean
  email_verify_enabled: boolean
  registration_email_suffix_whitelist: string[]
  promo_code_enabled: boolean
  password_reset_enabled: boolean
  frontend_url: string
  invitation_code_enabled: boolean
  totp_enabled: boolean // TOTP 双因素认证
  totp_encryption_key_configured: boolean // TOTP 加密密钥是否已配置
  // Default settings
  default_balance: number
  default_concurrency: number
  default_subscriptions: DefaultSubscriptionSetting[]
  // OEM settings
  site_name: string
  site_logo: string
  site_subtitle: string
  api_base_url: string
  contact_info: string
  doc_url: string
  home_content: string
  hide_ccs_import_button: boolean
  available_channels_enabled: boolean
  channel_monitor_enabled: boolean
  channel_monitor_default_interval_seconds: number
  public_model_catalog_enabled: boolean
  // Affiliate / Invite rebate (运营版)
  affiliate_enabled: boolean
  affiliate_transfer_enabled: boolean
  affiliate_rebate_on_usage_enabled: boolean
  affiliate_rebate_on_topup_enabled: boolean
  affiliate_rebate_rate: number
  affiliate_rebate_freeze_hours: number
  affiliate_rebate_duration_days: number
  affiliate_rebate_per_invitee_cap: number
  affiliate_aff_code_length: number
  purchase_subscription_enabled: boolean
  purchase_subscription_url: string
  backend_mode_enabled: boolean
  maintenance_mode_enabled: boolean
  custom_menu_items: CustomMenuItem[]
  // SMTP settings
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_password_configured: boolean
  smtp_from_email: string
  smtp_from_name: string
  smtp_use_tls: boolean
  telegram_chat_id: string
  telegram_bot_token_configured: boolean
  telegram_bot_token_masked: string
  // Cloudflare Turnstile settings
  turnstile_enabled: boolean
  turnstile_site_key: string
  turnstile_secret_key_configured: boolean

  // LinuxDo Connect OAuth settings
  linuxdo_connect_enabled: boolean
  linuxdo_connect_client_id: string
  linuxdo_connect_client_secret_configured: boolean
  linuxdo_connect_redirect_url: string

  // Model fallback configuration
  enable_model_fallback: boolean
  fallback_model_anthropic: string
  fallback_model_openai: string
  fallback_model_gemini: string
  fallback_model_antigravity: string

  // Identity patch configuration (Claude -> Gemini)
  enable_identity_patch: boolean
  identity_patch_prompt: string

  // Ops Monitoring (vNext)
  ops_monitoring_enabled: boolean
  ops_realtime_monitoring_enabled: boolean
  ops_query_mode_default: 'auto' | 'raw' | 'preagg' | string
  ops_metrics_interval_seconds: number

  // Claude Code version check
  min_claude_code_version: string
  max_claude_code_version: string

  // 分组隔离
  allow_ungrouped_key_scheduling: boolean
}

export interface UpdateSettingsRequest {
  registration_enabled?: boolean
  email_verify_enabled?: boolean
  registration_email_suffix_whitelist?: string[]
  promo_code_enabled?: boolean
  password_reset_enabled?: boolean
  frontend_url?: string
  invitation_code_enabled?: boolean
  totp_enabled?: boolean // TOTP 双因素认证
  default_balance?: number
  default_concurrency?: number
  default_subscriptions?: DefaultSubscriptionSetting[]
  site_name?: string
  site_logo?: string
  site_subtitle?: string
  api_base_url?: string
  contact_info?: string
  doc_url?: string
  home_content?: string
  hide_ccs_import_button?: boolean
  available_channels_enabled?: boolean
  channel_monitor_enabled?: boolean
  channel_monitor_default_interval_seconds?: number
  public_model_catalog_enabled?: boolean
  affiliate_enabled?: boolean
  affiliate_transfer_enabled?: boolean
  affiliate_rebate_on_usage_enabled?: boolean
  affiliate_rebate_on_topup_enabled?: boolean
  affiliate_rebate_rate?: number
  affiliate_rebate_freeze_hours?: number
  affiliate_rebate_duration_days?: number
  affiliate_rebate_per_invitee_cap?: number
  affiliate_aff_code_length?: number
  purchase_subscription_enabled?: boolean
  purchase_subscription_url?: string
  backend_mode_enabled?: boolean
  maintenance_mode_enabled?: boolean
  custom_menu_items?: CustomMenuItem[]
  smtp_host?: string
  smtp_port?: number
  smtp_username?: string
  smtp_password?: string
  smtp_from_email?: string
  smtp_from_name?: string
  smtp_use_tls?: boolean
  telegram_chat_id?: string
  telegram_bot_token?: string
  turnstile_enabled?: boolean
  turnstile_site_key?: string
  turnstile_secret_key?: string
  linuxdo_connect_enabled?: boolean
  linuxdo_connect_client_id?: string
  linuxdo_connect_client_secret?: string
  linuxdo_connect_redirect_url?: string
  enable_model_fallback?: boolean
  fallback_model_anthropic?: string
  fallback_model_openai?: string
  fallback_model_gemini?: string
  fallback_model_antigravity?: string
  enable_identity_patch?: boolean
  identity_patch_prompt?: string
  ops_monitoring_enabled?: boolean
  ops_realtime_monitoring_enabled?: boolean
  ops_query_mode_default?: 'auto' | 'raw' | 'preagg' | string
  ops_metrics_interval_seconds?: number
  min_claude_code_version?: string
  max_claude_code_version?: string
  allow_ungrouped_key_scheduling?: boolean
}

/**
 * Get all system settings
 * @returns System settings
 */
export async function getSettings(): Promise<SystemSettings> {
  const { data } = await apiClient.get<SystemSettings>('/admin/settings')
  return data
}

/**
 * Update system settings
 * @param settings - Partial settings to update
 * @returns Updated settings
 */
export async function updateSettings(settings: UpdateSettingsRequest): Promise<SystemSettings> {
  const { data } = await apiClient.put<SystemSettings>('/admin/settings', settings)
  return data
}

/**
 * Test SMTP connection request
 */
export interface TestSmtpRequest {
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_password: string
  smtp_use_tls: boolean
}

/**
 * Test SMTP connection with provided config
 * @param config - SMTP configuration to test
 * @returns Test result message
 */
export async function testSmtpConnection(config: TestSmtpRequest): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>('/admin/settings/test-smtp', config)
  return data
}

/**
 * Send test email request
 */
export interface SendTestEmailRequest {
  email: string
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_password: string
  smtp_from_email: string
  smtp_from_name: string
  smtp_use_tls: boolean
}

/**
 * Send test email with provided SMTP config
 * @param request - Email address and SMTP config
 * @returns Test result message
 */
export async function sendTestEmail(request: SendTestEmailRequest): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>(
    '/admin/settings/send-test-email',
    request
  )
  return data
}

export interface TestTelegramRequest {
  bot_token?: string
  chat_id?: string
}

export async function testTelegramConnection(
  request: TestTelegramRequest
): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>(
    '/admin/settings/telegram/test',
    request
  )
  return data
}

/**
 * Admin API Key status response
 */
export interface AdminApiKeyStatus {
  exists: boolean
  masked_key: string
}

/**
 * Get admin API key status
 * @returns Status indicating if key exists and masked version
 */
export async function getAdminApiKey(): Promise<AdminApiKeyStatus> {
  const { data } = await apiClient.get<AdminApiKeyStatus>('/admin/settings/admin-api-key')
  return data
}

/**
 * Regenerate admin API key
 * @returns The new full API key (only shown once)
 */
export async function regenerateAdminApiKey(): Promise<{ key: string }> {
  const { data } = await apiClient.post<{ key: string }>('/admin/settings/admin-api-key/regenerate')
  return data
}

/**
 * Delete admin API key
 * @returns Success message
 */
export async function deleteAdminApiKey(): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>('/admin/settings/admin-api-key')
  return data
}

// ==================== Overload Cooldown Settings ====================

/**
 * Overload cooldown settings interface (529 handling)
 */
export interface OverloadCooldownSettings {
  enabled: boolean
  cooldown_minutes: number
}

export async function getOverloadCooldownSettings(): Promise<OverloadCooldownSettings> {
  const { data } = await apiClient.get<OverloadCooldownSettings>('/admin/settings/overload-cooldown')
  return data
}

export async function updateOverloadCooldownSettings(
  settings: OverloadCooldownSettings
): Promise<OverloadCooldownSettings> {
  const { data } = await apiClient.put<OverloadCooldownSettings>(
    '/admin/settings/overload-cooldown',
    settings
  )
  return data
}

// ==================== Stream Timeout Settings ====================

/**
 * Stream timeout settings interface
 */
export interface StreamTimeoutSettings {
  enabled: boolean
  action: 'temp_unsched' | 'error' | 'none'
  temp_unsched_minutes: number
  threshold_count: number
  threshold_window_minutes: number
}

/**
 * Get stream timeout settings
 * @returns Stream timeout settings
 */
export async function getStreamTimeoutSettings(): Promise<StreamTimeoutSettings> {
  const { data } = await apiClient.get<StreamTimeoutSettings>('/admin/settings/stream-timeout')
  return data
}

/**
 * Update stream timeout settings
 * @param settings - Stream timeout settings to update
 * @returns Updated settings
 */
export async function updateStreamTimeoutSettings(
  settings: StreamTimeoutSettings
): Promise<StreamTimeoutSettings> {
  const { data } = await apiClient.put<StreamTimeoutSettings>(
    '/admin/settings/stream-timeout',
    settings
  )
  return data
}

// ==================== Rectifier Settings ====================

/**
 * Rectifier settings interface
 */
export interface RectifierSettings {
  enabled: boolean
  thinking_signature_enabled: boolean
  thinking_budget_enabled: boolean
}

/**
 * Get rectifier settings
 * @returns Rectifier settings
 */
export async function getRectifierSettings(): Promise<RectifierSettings> {
  const { data } = await apiClient.get<RectifierSettings>('/admin/settings/rectifier')
  return data
}

/**
 * Update rectifier settings
 * @param settings - Rectifier settings to update
 * @returns Updated settings
 */
export async function updateRectifierSettings(
  settings: RectifierSettings
): Promise<RectifierSettings> {
  const { data } = await apiClient.put<RectifierSettings>(
    '/admin/settings/rectifier',
    settings
  )
  return data
}

// ==================== Beta Policy Settings ====================

/**
 * Beta policy rule interface
 */
export interface BetaPolicyRule {
  beta_token: string
  action: 'pass' | 'filter' | 'block'
  scope: 'all' | 'oauth' | 'apikey' | 'bedrock'
  error_message?: string
}

/**
 * Beta policy settings interface
 */
export interface BetaPolicySettings {
  rules: BetaPolicyRule[]
}

/**
 * Get beta policy settings
 * @returns Beta policy settings
 */
export async function getBetaPolicySettings(): Promise<BetaPolicySettings> {
  const { data } = await apiClient.get<BetaPolicySettings>('/admin/settings/beta-policy')
  return data
}

/**
 * Update beta policy settings
 * @param settings - Beta policy settings to update
 * @returns Updated settings
 */
export async function updateBetaPolicySettings(
  settings: BetaPolicySettings
): Promise<BetaPolicySettings> {
  const { data } = await apiClient.put<BetaPolicySettings>(
    '/admin/settings/beta-policy',
    settings
  )
  return data
}

export interface GoogleBatchGCSProfile {
  profile_id: string
  name: string
  is_active: boolean
  enabled: boolean
  bucket: string
  prefix: string
  project_id: string
  service_account_json_configured: boolean
  updated_at: string
}

export interface ListGoogleBatchGCSProfilesResponse {
  active_profile_id: string
  items: GoogleBatchGCSProfile[]
}

export interface CreateGoogleBatchGCSProfileRequest {
  profile_id: string
  name: string
  set_active?: boolean
  enabled: boolean
  bucket: string
  prefix: string
  project_id: string
  service_account_json?: string
}

export interface UpdateGoogleBatchGCSProfileRequest {
  name: string
  enabled: boolean
  bucket: string
  prefix: string
  project_id: string
  service_account_json?: string
}

export interface TestGoogleBatchGCSConnectionRequest {
  profile_id?: string
  enabled: boolean
  bucket: string
  prefix: string
  project_id: string
  service_account_json?: string
}

export interface GoogleBatchArchiveSettings {
  enabled: boolean
  poll_min_interval_seconds: number
  poll_max_interval_seconds: number
  poll_backoff_factor: number
  poll_jitter_seconds: number
  poll_max_concurrency: number
  prefetch_after_hours: number
  download_timeout_seconds: number
  cleanup_interval_minutes: number
  local_storage_root: string
}

export interface GeminiRateCatalogModelRow {
  model_family: string
  display_name: string
  rpm: number
  tpm: number
  rpd: number
  notes?: string
}

export interface GeminiRateCatalogTier {
  tier_id: string
  display_name: string
  qualification: string
  billing_tier_cap: string
  model_families: GeminiRateCatalogModelRow[]
}

export interface GeminiRateCatalogBatchRow {
  model_family: string
  display_name: string
  enqueued_tokens: number
}

export interface GeminiRateCatalogBatchTier {
  tier_id: string
  entries: GeminiRateCatalogBatchRow[]
}

export interface GeminiRateCatalogBatchLimits {
  concurrent_batch_requests: number
  input_file_size_limit_bytes: number
  file_storage_limit_bytes: number
  by_tier: GeminiRateCatalogBatchTier[]
}

export interface GeminiRateCatalogLink {
  label: string
  url: string
}

export interface GeminiRateCatalog {
  effective_date: string
  remaining_quota_api_supported: boolean
  ai_studio_tiers: GeminiRateCatalogTier[]
  batch_limits: GeminiRateCatalogBatchLimits
  links: GeminiRateCatalogLink[]
  notes: string[]
}

export async function getGeminiRateCatalog(): Promise<GeminiRateCatalog> {
  const { data } = await apiClient.get<GeminiRateCatalog>('/admin/settings/gemini-rate-catalog')
  return data
}

export async function getGoogleBatchArchiveSettings(): Promise<GoogleBatchArchiveSettings> {
  const { data } = await apiClient.get<GoogleBatchArchiveSettings>('/admin/settings/google-batch-archive')
  return data
}

export async function updateGoogleBatchArchiveSettings(
  request: GoogleBatchArchiveSettings,
): Promise<GoogleBatchArchiveSettings> {
  const { data } = await apiClient.put<GoogleBatchArchiveSettings>(
    '/admin/settings/google-batch-archive',
    request,
  )
  return data
}

export async function listGoogleBatchGCSProfiles(): Promise<ListGoogleBatchGCSProfilesResponse> {
  const { data } = await apiClient.get<ListGoogleBatchGCSProfilesResponse>('/admin/settings/google-batch-gcs/profiles')
  return data
}

export async function createGoogleBatchGCSProfile(request: CreateGoogleBatchGCSProfileRequest): Promise<GoogleBatchGCSProfile> {
  const { data } = await apiClient.post<GoogleBatchGCSProfile>('/admin/settings/google-batch-gcs/profiles', request)
  return data
}

export async function updateGoogleBatchGCSProfile(profileID: string, request: UpdateGoogleBatchGCSProfileRequest): Promise<GoogleBatchGCSProfile> {
  const { data } = await apiClient.put<GoogleBatchGCSProfile>(`/admin/settings/google-batch-gcs/profiles/${profileID}`, request)
  return data
}

export async function deleteGoogleBatchGCSProfile(profileID: string): Promise<void> {
  await apiClient.delete(`/admin/settings/google-batch-gcs/profiles/${profileID}`)
}

export async function setActiveGoogleBatchGCSProfile(profileID: string): Promise<GoogleBatchGCSProfile> {
  const { data } = await apiClient.post<GoogleBatchGCSProfile>(`/admin/settings/google-batch-gcs/profiles/${profileID}/activate`)
  return data
}

export async function testGoogleBatchGCSConnection(request: TestGoogleBatchGCSConnectionRequest): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>('/admin/settings/google-batch-gcs/test', request)
  return data
}

export const settingsAPI = {
  getSettings,
  updateSettings,
  testSmtpConnection,
  testTelegramConnection,
  sendTestEmail,
  getAdminApiKey,
  regenerateAdminApiKey,
  deleteAdminApiKey,
  getOverloadCooldownSettings,
  updateOverloadCooldownSettings,
  getStreamTimeoutSettings,
  updateStreamTimeoutSettings,
  getRectifierSettings,
  updateRectifierSettings,
  getBetaPolicySettings,
  updateBetaPolicySettings,
  getGeminiRateCatalog,
  getGoogleBatchArchiveSettings,
  updateGoogleBatchArchiveSettings,
  listGoogleBatchGCSProfiles,
  createGoogleBatchGCSProfile,
  updateGoogleBatchGCSProfile,
  deleteGoogleBatchGCSProfile,
  setActiveGoogleBatchGCSProfile,
  testGoogleBatchGCSConnection
}

export default settingsAPI
