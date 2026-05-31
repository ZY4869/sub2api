import type { APIKeyModelBindingMode, TimeAccessPolicy } from './api-key-groups'
import type { PaymentSubscriptionPlan } from './payments'
import type {
  UsageContextBadgeDisplayMode,
  UsageModelDisplayMode,
  VisualPreset,
  VisualPresetPreference
} from './preferences'
import type { UserSubscription } from './user-subscriptions'
// ==================== User & Auth Types ====================

export interface User {
  id: number;
  username: string;
  email: string;
  role: "admin" | "user"; // User role for authorization
  api_key_model_binding_mode?: APIKeyModelBindingMode;
  api_key_access_time_policy?: TimeAccessPolicy | null;
  request_details_review?: boolean;
  admin_free_billing?: boolean;
  usage_model_display_mode?: UsageModelDisplayMode;
  usage_context_badge_display_mode?: UsageContextBadgeDisplayMode;
  global_realtime_countdown_enabled?: boolean;
  account_realtime_countdown_enabled?: boolean;
  visual_preset_preference?: VisualPresetPreference;
  account_visual_preset_override?: VisualPresetPreference;
  balance: number; // User balance for API usage
  balances?: Record<string, number>; // Wallet balances by billing currency
  concurrency: number; // Allowed concurrent requests
  status: "active" | "disabled"; // Account status
  allowed_groups: number[] | null; // Allowed group IDs (null = all non-exclusive groups)
  subscriptions?: UserSubscription[]; // User's active subscriptions
  created_at: string;
  updated_at: string;
}

export interface AdminUser extends User {
  notes: string;
  admin_free_billing: boolean;
  // Per-group rate multipliers keyed by group_id.
  group_rates?: Record<number, number>;
  // Current concurrency snapshot for admin views.
  current_concurrency?: number;
}

export interface LoginRequest {
  email: string;
  password: string;
  turnstile_token?: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  verify_code?: string;
  turnstile_token?: string;
  promo_code?: string;
  invitation_code?: string;
  aff_code?: string;
}

export interface SendVerifyCodeRequest {
  email: string;
  turnstile_token?: string;
  locale?: string;
}

export interface SendVerifyCodeResponse {
  message: string;
  countdown: number;
}

export interface CustomMenuItem {
  id: string;
  label: string;
  icon_svg: string;
  url: string;
  visibility: "user" | "admin";
  sort_order: number;
  page_mode?: "iframe" | "markdown";
  page_slug?: string;
  page_content?: string;
  page_public?: boolean;
  page_published?: boolean;
}

export interface CustomPageContent {
  id: string;
  slug: string;
  label: string;
  visibility: "user" | "admin";
  page_mode: "markdown";
  content: string;
  updated_at?: string;
}

export interface LoginAgreementDocument {
  id: string;
  title: string;
  page_slug: string;
}

export interface PublicSettings {
  registration_enabled: boolean;
  email_verify_enabled: boolean;
  registration_email_suffix_whitelist: string[];
  promo_code_enabled: boolean;
  password_reset_enabled: boolean;
  invitation_code_enabled: boolean;
  turnstile_enabled: boolean;
  turnstile_site_key: string;
  site_name: string;
  site_logo: string;
  site_subtitle: string;
  visual_preset_default: VisualPreset;
  account_airy_white_surface_enabled?: boolean;
  api_base_url: string;
  contact_info: string;
  doc_url: string;
  home_content: string;
  hide_ccs_import_button: boolean;
  available_channels_enabled: boolean;
  channel_monitor_enabled: boolean;
  public_model_catalog_enabled: boolean;
  affiliate_enabled: boolean;
  purchase_subscription_enabled: boolean;
  purchase_subscription_url: string;
  payment_provider_airwallex_enabled: boolean;
  payment_mobile_force_qrcode_enabled: boolean;
  payment_allowed_currencies: string[];
  payment_default_currency: string;
  payment_min_topup_amount: number;
  payment_max_topup_amount: number;
  payment_subscription_plans: PaymentSubscriptionPlan[];
  custom_menu_items: CustomMenuItem[];
  login_agreement_enabled: boolean;
  login_agreement_mode: "checkbox" | string;
  login_agreement_updated_at: string;
  login_agreement_documents: LoginAgreementDocument[];
  linuxdo_oauth_enabled: boolean;
  github_oauth_enabled: boolean;
  google_oauth_enabled: boolean;
  dingtalk_oauth_enabled: boolean;
  backend_mode_enabled: boolean;
  maintenance_mode_enabled: boolean;
  version: string;
}

export interface AuthIdentity {
  id: number;
  provider: "github" | "google" | string;
  provider_user_id: string;
  email: string;
  email_verified: boolean;
  display_name: string;
  avatar_url: string;
  created_at?: string;
  updated_at?: string;
}

export interface ContentModerationAudit {
  id: number;
  request_id: string;
  client_request_id: string;
  user_id: number | null;
  api_key_id: number | null;
  provider: string;
  model: string;
  source_endpoint: string;
  content_hash: string;
  content_summary: string;
  categories: string[];
  hit: boolean;
  dedupe_hit: boolean;
  error_reason: string;
  latency_ms: number;
  created_at: string;
}

export type SocialOAuthProvider = "github" | "google" | "dingtalk";

export interface SocialOAuthCompleteResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  token_type: string;
}

export interface AuthResponse {
  access_token: string;
  refresh_token?: string; // New: Refresh Token for token renewal
  expires_in?: number; // New: Access Token expiry time in seconds
  token_type: string;
  user: User & { run_mode?: "standard" | "simple" };
}

export interface CurrentUserResponse extends User {
  run_mode?: "standard" | "simple";
}
