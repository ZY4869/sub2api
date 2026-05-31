// ==================== API Key & Group Types ====================

export type GroupPlatform =
  | "anthropic"
  | "kiro"
  | "openai"
  | "openrouter"
  | "grok"
  | "deepseek"
  | "gemini"
  | "antigravity"
  | "protocol_gateway"
  | "baidu_document_ai";

export type APIKeyModelBindingMode = "model_required" | "group_allowed";

export interface TimeAccessWindow {
  days: number[];
  start: string;
  end: string;
}

export interface TimeAccessPolicy {
  enabled: boolean;
  timezone?: string;
  not_before?: string | null;
  not_after?: string | null;
  weekly_windows?: TimeAccessWindow[];
  daily_allowed_minutes?: number | null;
}

export type SubscriptionType = "standard" | "subscription";
export type OpenAIImageProtocolMode = "native" | "compat";
export type OpenAIGroupImageProtocolMode =
  | "inherit"
  | OpenAIImageProtocolMode;

export interface Group {
  id: number;
  name: string;
  description: string | null;
  platform: GroupPlatform;
  priority: number;
  rate_multiplier: number;
  is_exclusive: boolean;
  status: "active" | "inactive";
  subscription_type: SubscriptionType;
  daily_limit_usd: number | null;
  weekly_limit_usd: number | null;
  monthly_limit_usd: number | null;
  // Image pricing is only used by antigravity groups.
  image_price_1k: number | null;
  image_price_2k: number | null;
  image_price_4k: number | null;
  // Restrict the group to Claude Code clients only.
  claude_code_only: boolean;
  image_protocol_mode: OpenAIGroupImageProtocolMode;
  fallback_group_id: number | null;
  fallback_group_id_on_invalid_request: number | null;
  // Toggle OpenAI Messages dispatch support for this group.
  allow_messages_dispatch?: boolean;
  gemini_mixed_protocol_enabled?: boolean;
  visible_model_patterns?: string[];
  created_at: string;
  updated_at: string;
}

export interface AdminGroup extends Group {
  // Optional routing map from requested model ids to account ids.
  model_routing: Record<string, number[]> | null;
  model_routing_enabled: boolean;

  // MCP XML injection toggle for antigravity groups.
  mcp_xml_inject: boolean;
  // Optional model scope allowlist for antigravity groups.
  supported_model_scopes?: string[];
  // Aggregated account counters for admin list views.
  account_count?: number;
  active_account_count?: number;
  rate_limited_account_count?: number;
  available_account_count?: number;
  // Default mapped model for OpenAI Messages-compatible groups.
  default_mapped_model?: string;
  // UI sort weight for admin lists.
  sort_order: number;
}

export interface ApiKey {
  id: number;
  user_id: number;
  key: string;
  name: string;
  deleted?: boolean;
  model_display_mode?: "alias_only" | "source_only" | "alias_and_source";
  group_id: number | null;
  group_ids?: number[];
  api_key_groups?: ApiKeyGroup[];
  status: "active" | "inactive" | "quota_exhausted" | "expired";
  ip_whitelist: string[];
  ip_blacklist: string[];
  last_used_at: string | null;
  quota: number; // Quota limit in USD (0 = unlimited)
  quota_used: number; // Used quota amount in USD
  quota_used_by_currency?: Record<string, number>;
  // Image-only key settings
  image_only_enabled: boolean;
  image_count_billing_enabled: boolean;
  image_max_count: number; // 0 = not configured (falls back to token billing)
  image_count_used: number;
  image_count_weights: Record<string, number>;
  expires_at: string | null; // Expiration time (null = never expires)
  starts_at?: string | null;
  access_time_policy?: TimeAccessPolicy | null;
  created_at: string;
  updated_at: string;
  group?: Group;
  rate_limit_5h: number;
  rate_limit_1d: number;
  rate_limit_7d: number;
  usage_5h: number;
  usage_1d: number;
  usage_7d: number;
  usage_5h_by_currency?: Record<string, number>;
  usage_1d_by_currency?: Record<string, number>;
  usage_7d_by_currency?: Record<string, number>;
  window_5h_start: string | null;
  window_1d_start: string | null;
  window_7d_start: string | null;
  reset_5h_at: string | null;
  reset_1d_at: string | null;
  reset_7d_at: string | null;
}

export interface ApiKeyGroup {
  group_id: number;
  group_name: string;
  platform: GroupPlatform;
  priority: number;
  quota: number;
  quota_used: number;
  quota_used_by_currency?: Record<string, number>;
  model_patterns: string[];
}

export interface ApiKeyGroupBindingInput {
  group_id: number;
  quota?: number;
  model_patterns?: string[];
}

export interface UserGroupModelOption {
  public_id: string;
  display_name: string;
  request_protocols?: string[];
}

export interface UserGroupModelOptionGroup {
  group_id: number;
  name: string;
  platform: GroupPlatform;
  priority: number;
  models: UserGroupModelOption[];
  model_count: number;
}

export interface CreateApiKeyRequest {
  name: string;
  group_id?: number | null;
  groups?: ApiKeyGroupBindingInput[];
  custom_key?: string; // Optional custom API Key
  ip_whitelist?: string[];
  ip_blacklist?: string[];
  quota?: number; // Quota limit in USD (0 = unlimited)
  expires_in_days?: number; // Days until expiry (null = never expires)
  starts_at?: string;
  access_time_policy?: TimeAccessPolicy;
  rate_limit_5h?: number;
  rate_limit_1d?: number;
  rate_limit_7d?: number;
  image_only_enabled?: boolean;
  image_count_billing_enabled?: boolean;
  image_max_count?: number;
  image_count_weights?: Record<string, number>;
}

export interface UpdateApiKeyRequest {
  name?: string;
  group_id?: number | null;
  groups?: ApiKeyGroupBindingInput[];
  status?: "active" | "inactive";
  ip_whitelist?: string[];
  ip_blacklist?: string[];
  quota?: number; // Quota limit in USD (null = no change, 0 = unlimited)
  expires_at?: string | null; // Expiration time (null = no change)
  starts_at?: string | null;
  access_time_policy?: TimeAccessPolicy | null;
  clear_access_time_policy?: boolean;
  reset_quota?: boolean; // Reset quota_used to 0
  rate_limit_5h?: number;
  rate_limit_1d?: number;
  rate_limit_7d?: number;
  reset_rate_limit_usage?: boolean;
  image_only_enabled?: boolean;
  image_count_billing_enabled?: boolean;
  image_max_count?: number;
  image_count_weights?: Record<string, number>;
}

export interface CreateGroupRequest {
  name: string;
  description?: string | null;
  platform?: GroupPlatform;
  priority?: number;
  rate_multiplier?: number;
  is_exclusive?: boolean;
  gemini_mixed_protocol_enabled?: boolean;
  subscription_type?: SubscriptionType;
  daily_limit_usd?: number | null;
  weekly_limit_usd?: number | null;
  monthly_limit_usd?: number | null;
  image_price_1k?: number | null;
  image_price_2k?: number | null;
  image_price_4k?: number | null;
  image_protocol_mode?: OpenAIGroupImageProtocolMode;
  claude_code_only?: boolean;
  fallback_group_id?: number | null;
  fallback_group_id_on_invalid_request?: number | null;
  mcp_xml_inject?: boolean;
  supported_model_scopes?: string[];
  visible_model_patterns?: string[];
  // Optional source groups to clone accounts from during group creation.
  copy_accounts_from_group_ids?: number[];
}

export interface UpdateGroupRequest {
  name?: string;
  description?: string | null;
  platform?: GroupPlatform;
  priority?: number;
  rate_multiplier?: number;
  is_exclusive?: boolean;
  gemini_mixed_protocol_enabled?: boolean;
  status?: "active" | "inactive";
  subscription_type?: SubscriptionType;
  daily_limit_usd?: number | null;
  weekly_limit_usd?: number | null;
  monthly_limit_usd?: number | null;
  image_price_1k?: number | null;
  image_price_2k?: number | null;
  image_price_4k?: number | null;
  image_protocol_mode?: OpenAIGroupImageProtocolMode;
  claude_code_only?: boolean;
  fallback_group_id?: number | null;
  fallback_group_id_on_invalid_request?: number | null;
  mcp_xml_inject?: boolean;
  supported_model_scopes?: string[];
  visible_model_patterns?: string[];
  copy_accounts_from_group_ids?: number[];
}
