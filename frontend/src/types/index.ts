/**
 * Core Type Definitions for Sub2API Frontend
 */

// ==================== Common Types ====================

export interface SelectOption {
  value: string | number | boolean | null;
  label: string;
  [key: string]: any; // Support extra properties for custom templates
}

export interface BasePaginationResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  pages: number;
}

export interface FetchOptions {
  signal?: AbortSignal;
}

// ==================== User & Auth Types ====================

export interface User {
  id: number;
  username: string;
  email: string;
  role: "admin" | "user"; // User role for authorization
  admin_free_billing?: boolean;
  balance: number; // User balance for API usage
  concurrency: number; // Allowed concurrent requests
  status: "active" | "disabled"; // Account status
  allowed_groups: number[] | null; // Allowed group IDs (null = all non-exclusive groups)
  subscriptions?: UserSubscription[]; // User's active subscriptions
  created_at: string;
  updated_at: string;
}

export interface AdminUser extends User {
  // 缂備胶濯寸槐鏇㈠箖婵犲洤宸濇俊顖欒濡插灚绻涙径妯煎帨缂佽鲸鐟╁鏌ヮ敋閳ь剟鍩€椤掍焦鐨戦柡浣靛€濋獮瀣煥鐎ｎ亜顦查梺鍛婄懕缁茶偐绮径瀣氦闁哄倹瀵х粈鈧梺?
  notes: string;
  admin_free_billing: boolean;
  // 闂佹椿娼块崝宥夊春濞戞瑧鈻旈柟鎯х－濞硷綁鏌涢幒鎴烆棤缂侇喖绉瑰畷鎰吋閸パ嗗У闂備焦婢樼粔鍫曟偪?(group_id -> rate_multiplier)
  group_rates?: Record<number, number>;
  // 閻熸粎澧楅幐鍛婃櫠閻樿崵宓侀悹鍝勬惈缁叉椽鏌℃担绋跨盎缂佽鲸鐟︾粋鎺楀川椤栵絽鎮侀梺鑽ゅ仜濡骞夐幎钘夌婵°倕瀚ㄩ埀顒€鍟撮獮鎺楀Ψ閵夈儳绋夐柡澶嗘櫆閺屻劌煤閺嶎厽鏅?
  current_concurrency?: number;
  // Sora 闁诲孩绋掗敋闁稿绉归弻濠傤吋婢舵ɑ婢撻梺鎸庣☉閻楀棝鎮鸿閹崇偤宕掗敂鍓ь槴
  sora_storage_quota_bytes: number;
  sora_storage_used_bytes: number;
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
}

export interface SendVerifyCodeRequest {
  email: string;
  turnstile_token?: string;
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
  api_base_url: string;
  contact_info: string;
  doc_url: string;
  home_content: string;
  hide_ccs_import_button: boolean;
  purchase_subscription_enabled: boolean;
  purchase_subscription_url: string;
  custom_menu_items: CustomMenuItem[];
  linuxdo_oauth_enabled: boolean;
  sora_client_enabled: boolean;
  backend_mode_enabled: boolean;
  version: string;
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

// ==================== Subscription Types ====================

export interface Subscription {
  id: number;
  user_id: number;
  name: string;
  url: string;
  type: "clash" | "v2ray" | "surge" | "quantumult" | "shadowrocket";
  update_interval: number; // in hours
  last_updated: string | null;
  node_count: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateSubscriptionRequest {
  name: string;
  url: string;
  type: Subscription["type"];
  update_interval?: number;
}

export interface UpdateSubscriptionRequest {
  name?: string;
  url?: string;
  type?: Subscription["type"];
  update_interval?: number;
  is_active?: boolean;
}

// ==================== Announcement Types ====================

export type AnnouncementStatus = "draft" | "active" | "archived";
export type AnnouncementNotifyMode = "silent" | "popup";

export type AnnouncementConditionType = "subscription" | "balance";

export type AnnouncementOperator = "in" | "gt" | "gte" | "lt" | "lte" | "eq";

export interface AnnouncementCondition {
  type: AnnouncementConditionType;
  operator: AnnouncementOperator;
  group_ids?: number[];
  value?: number;
}

export interface AnnouncementConditionGroup {
  all_of?: AnnouncementCondition[];
}

export interface AnnouncementTargeting {
  any_of?: AnnouncementConditionGroup[];
}

export interface Announcement {
  id: number;
  title: string;
  content: string;
  status: AnnouncementStatus;
  notify_mode: AnnouncementNotifyMode;
  targeting: AnnouncementTargeting;
  starts_at?: string;
  ends_at?: string;
  created_by?: number;
  updated_by?: number;
  created_at: string;
  updated_at: string;
}

export interface UserAnnouncement {
  id: number;
  title: string;
  content: string;
  notify_mode: AnnouncementNotifyMode;
  starts_at?: string;
  ends_at?: string;
  read_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAnnouncementRequest {
  title: string;
  content: string;
  status?: AnnouncementStatus;
  notify_mode?: AnnouncementNotifyMode;
  targeting: AnnouncementTargeting;
  starts_at?: number;
  ends_at?: number;
}

export interface UpdateAnnouncementRequest {
  title?: string;
  content?: string;
  status?: AnnouncementStatus;
  notify_mode?: AnnouncementNotifyMode;
  targeting?: AnnouncementTargeting;
  starts_at?: number;
  ends_at?: number;
}

export interface AnnouncementUserReadStatus {
  user_id: number;
  email: string;
  username: string;
  balance: number;
  eligible: boolean;
  read_at?: string;
}

// ==================== Proxy Node Types ====================

export interface ProxyNode {
  id: number;
  subscription_id: number;
  name: string;
  type: "ss" | "ssr" | "vmess" | "vless" | "trojan" | "hysteria" | "hysteria2";
  server: string;
  port: number;
  config: Record<string, unknown>; // JSON configuration specific to proxy type
  latency: number | null; // in milliseconds
  last_checked: string | null;
  is_available: boolean;
  created_at: string;
  updated_at: string;
}

// ==================== Conversion Types ====================

export interface ConversionRequest {
  subscription_ids: number[];
  target_type: "clash" | "v2ray" | "surge" | "quantumult" | "shadowrocket";
  filter?: {
    name_pattern?: string;
    types?: ProxyNode["type"][];
    min_latency?: number;
    max_latency?: number;
    available_only?: boolean;
  };
  sort?: {
    by: "name" | "latency" | "type";
    order: "asc" | "desc";
  };
}

export interface ConversionResult {
  url: string; // URL to download the converted subscription
  expires_at: string;
  node_count: number;
}

// ==================== Statistics Types ====================

export interface SubscriptionStats {
  subscription_id: number;
  total_nodes: number;
  available_nodes: number;
  avg_latency: number | null;
  by_type: Record<ProxyNode["type"], number>;
  last_update: string;
}

export interface UserStats {
  total_subscriptions: number;
  total_nodes: number;
  active_subscriptions: number;
  total_conversions: number;
  last_conversion: string | null;
}

// ==================== API Response Types ====================

export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

export interface ApiError {
  detail: string;
  code?: string;
  field?: string;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  pages: number;
}

// ==================== UI State Types ====================

export type ToastType = "success" | "error" | "info" | "warning";

export interface ToastOptions {
  title?: string;
  details?: string[];
  copyText?: string;
  persistent?: boolean;
  duration?: number;
}

export interface Toast extends ToastOptions {
  id: string;
  type: ToastType;
  message: string;
  startTime?: number; // timestamp when toast was created, for progress bar
}

export interface AppState {
  sidebarCollapsed: boolean;
  loading: boolean;
  toasts: Toast[];
}

// ==================== Validation Types ====================

export interface ValidationError {
  field: string;
  message: string;
}

// ==================== Table/List Types ====================

export interface SortConfig {
  key: string;
  order: "asc" | "desc";
}

export interface FilterConfig {
  [key: string]: string | number | boolean | null | undefined;
}

export interface PaginationConfig {
  page: number;
  page_size: number;
}

// ==================== API Key & Group Types ====================

export type GroupPlatform =
  | "anthropic"
  | "kiro"
  | "openai"
  | "copilot"
  | "grok"
  | "gemini"
  | "antigravity"
  | "sora";

export type SubscriptionType = "standard" | "subscription";

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
  // 闂佹悶鍎辨晶鑺ユ櫠閺嶎厽鍋ㄩ柣鏃傤焾閻忓洭鎮规导顔哄€曢悗顓㈡⒑閺夎法肖闁汇倕妫濋弫宥夊醇濠婂懐鐓?antigravity 濡ょ姷鍋涢崯鑳亹鐎涙ɑ濯撮悹鎭掑妽閺嗗繘鏌?
  image_price_1k: number | null;
  image_price_2k: number | null;
  image_price_4k: number | null;
  // Sora 闂佸湱顭堥ˇ浼搭敃閼测晜濯奸梽鍥垂閸岀偞鐓€鐎广儱娲ㄩ弸?
  sora_image_price_360: number | null;
  sora_image_price_540: number | null;
  sora_video_price_per_request: number | null;
  sora_video_price_per_request_hd: number | null;
  // Sora 闁诲孩绋掗敋闁稿绉归弻濠傤吋婢舵ɑ婢撻梺鎸庣☉閻楀棝鎮鸿閹崇偤宕掗敂鍓ь槴
  sora_storage_quota_bytes: number;
  // Claude Code 闁诲骸绠嶉崹娲春濞戞氨鍗氭い鏍仦椤庢瑩鏌?
  claude_code_only: boolean;
  fallback_group_id: number | null;
  fallback_group_id_on_invalid_request: number | null;
  // OpenAI Messages 闁荤姴顑呴崯顐も偓鐟板暱椤曪綁鍩€椤掑嫬绀傜紒娑樻贡缁€鍕煟椤剙濡介柛鈺傜⊕缁楃喕顦规繛鎾冲閹茬増鎷呯拠鈥冲Π闁诲孩绋掗〃鍡涱敊瀹€鍕闁靛牆妫欓悞浠嬫煛閸曢潧鐏犻柟顖欒兌娴狅箓寮撮悩顔荤驳 Claude Code 闂佽桨鐒﹂悷褔鍩㈡總鍛婃櫖?
  allow_messages_dispatch?: boolean;
  created_at: string;
  updated_at: string;
}

export interface AdminGroup extends Group {
  // 濠碘槅鍨埀顒€纾埀顒勵棑閹瑰嫰顢涘鍕闂備焦婢樼粔鍫曟偪閸℃稒鏅柛顐ｇ矌閻瞼绱掗悪鍛？闁诡喖锕畷銊ノ熼崫鍕唹闁荤喐鐟ょ欢銈囨濠靛绀冮柛娑欐綑閸斻儱菐閸ワ絽澧插ù鐓庢嚇閺?
  model_routing: Record<string, number[]> | null;
  model_routing_enabled: boolean;

  // MCP XML 闂佸憡顨呯换妤咁敊閸涱厸鏋栭柕濞垮劚瀵娊鏌ㄥ☉妯煎缂?antigravity 濡ょ姷鍋涢崯鑳亹鐎涙ɑ濯撮悹鎭掑妽閺嗗繘鏌?
  // MCP XML injection toggle for antigravity groups.
  mcp_xml_inject: boolean;

  // 闂佽 鍋撴い鏍ㄧ☉閻︻噣鏌ｉ妸銉ヮ仾閼垛晠鏌涢妸銉剳闂侇喗鎸冲畷姘旂€ｎ剛顦╂繛?antigravity 濡ょ姷鍋涢崯鑳亹鐎涙ɑ濯撮悹鎭掑妽閺嗗繘鏌?
  supported_model_scopes?: string[];

  // 闂佸憡甯掑Λ娑氬垝瀹ュ棛鈻旈悗锝庡幖椤︹晠鏌涘▎鎾存暠闁哄棛鍠栭弻宀冪疀閵壯咁槱婵炲濮撮幊鎰邦敇閹间焦鍋犻柛鈩冾殕閸犲懘鏌涘▎妯虹仴妞ゎ偄妫濋弫?
  account_count?: number;
  active_account_count?: number;
  rate_limited_account_count?: number;

  // OpenAI Messages 闁荤姴顑呴崯顐も偓瑙勫▕閺屽﹤顓奸崶鈺傜€梺鎸庣☉閻楀懐鍒?openai 濡ょ姷鍋涢崯鑳亹鐎涙ɑ濯撮悹鎭掑妽閺嗗繘鏌?
  default_mapped_model?: string;

  // 闂佸憡甯掑Λ娑氬垝瀹ュ绠抽柟鐑樺灩绾?
  sort_order: number;
}

export interface ApiKey {
  id: number;
  user_id: number;
  key: string;
  name: string;
  group_id: number | null;
  group_ids?: number[];
  api_key_groups?: ApiKeyGroup[];
  status: "active" | "inactive" | "quota_exhausted" | "expired";
  ip_whitelist: string[];
  ip_blacklist: string[];
  last_used_at: string | null;
  quota: number; // Quota limit in USD (0 = unlimited)
  quota_used: number; // Used quota amount in USD
  expires_at: string | null; // Expiration time (null = never expires)
  created_at: string;
  updated_at: string;
  group?: Group;
  rate_limit_5h: number;
  rate_limit_1d: number;
  rate_limit_7d: number;
  usage_5h: number;
  usage_1d: number;
  usage_7d: number;
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
  model_patterns: string[];
}

export interface ApiKeyGroupBindingInput {
  group_id: number;
  quota?: number;
  model_patterns?: string[];
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
  rate_limit_5h?: number;
  rate_limit_1d?: number;
  rate_limit_7d?: number;
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
  reset_quota?: boolean; // Reset quota_used to 0
  rate_limit_5h?: number;
  rate_limit_1d?: number;
  rate_limit_7d?: number;
  reset_rate_limit_usage?: boolean;
}

export interface CreateGroupRequest {
  name: string;
  description?: string | null;
  platform?: GroupPlatform;
  priority?: number;
  rate_multiplier?: number;
  is_exclusive?: boolean;
  subscription_type?: SubscriptionType;
  daily_limit_usd?: number | null;
  weekly_limit_usd?: number | null;
  monthly_limit_usd?: number | null;
  image_price_1k?: number | null;
  image_price_2k?: number | null;
  image_price_4k?: number | null;
  sora_image_price_360?: number | null;
  sora_image_price_540?: number | null;
  sora_video_price_per_request?: number | null;
  sora_video_price_per_request_hd?: number | null;
  sora_storage_quota_bytes?: number;
  claude_code_only?: boolean;
  fallback_group_id?: number | null;
  fallback_group_id_on_invalid_request?: number | null;
  mcp_xml_inject?: boolean;
  supported_model_scopes?: string[];
  // 婵炲濮寸€涒晝鈧灚姘ㄩ埀顒冾潐閼归箖宕规惔锝囩＜闁告洦鍋掑Σ濠氭煕閹烘挸鍔跺璺哄瀹?
  copy_accounts_from_group_ids?: number[];
}

export interface UpdateGroupRequest {
  name?: string;
  description?: string | null;
  platform?: GroupPlatform;
  priority?: number;
  rate_multiplier?: number;
  is_exclusive?: boolean;
  status?: "active" | "inactive";
  subscription_type?: SubscriptionType;
  daily_limit_usd?: number | null;
  weekly_limit_usd?: number | null;
  monthly_limit_usd?: number | null;
  image_price_1k?: number | null;
  image_price_2k?: number | null;
  image_price_4k?: number | null;
  sora_image_price_360?: number | null;
  sora_image_price_540?: number | null;
  sora_video_price_per_request?: number | null;
  sora_video_price_per_request_hd?: number | null;
  sora_storage_quota_bytes?: number;
  claude_code_only?: boolean;
  fallback_group_id?: number | null;
  fallback_group_id_on_invalid_request?: number | null;
  mcp_xml_inject?: boolean;
  supported_model_scopes?: string[];
  copy_accounts_from_group_ids?: number[];
}

// ==================== Account & Proxy Types ====================

export type AccountPlatform =
  | "anthropic"
  | "kiro"
  | "openai"
  | "copilot"
  | "grok"
  | "gemini"
  | "antigravity"
  | "sora"
  | "protocol_gateway";
export type GatewayProtocol = "openai" | "anthropic" | "gemini" | "mixed";
export type GatewayAcceptedProtocol = "openai" | "anthropic" | "gemini";
export type GatewayClientProfile = "codex" | "gemini_cli";
export interface GatewayClientRoute {
  protocol: GatewayAcceptedProtocol;
  match_type: "exact" | "prefix";
  match_value: string;
  client_profile: GatewayClientProfile;
}
export type AccountType =
  | "oauth"
  | "setup-token"
  | "apikey"
  | "sso"
  | "bedrock"
  | "upstream";
export type AccountLifecycleState = "normal" | "archived" | "blacklisted";
export type AccountLimitedView = "all" | "normal_only" | "limited_only";
export type AccountRuntimeView = "all" | "in_use_only";
export type AccountRateLimitReason = "rate_429" | "usage_5h" | "usage_7d";
export type AccountViewMode = "table" | "card";
export type OAuthAddMethod = "oauth" | "setup-token";
export type ProxyProtocol = "http" | "https" | "socks5" | "socks5h";

// Claude Model type (returned by /v1/models and account models API)
export interface ClaudeModel {
  id: string;
  type: string;
  display_name: string;
  created_at: string;
  canonical_id?: string;
  source_protocol?: "openai" | "anthropic" | "gemini";
  status?: "stable" | "beta" | "deprecated";
  deprecated_at?: string;
  replaced_by?: string;
}

export interface Proxy {
  id: number;
  name: string;
  protocol: ProxyProtocol;
  host: string;
  port: number;
  username: string | null;
  password?: string | null;
  status: "active" | "inactive";
  account_count?: number; // Number of accounts using this proxy
  latency_ms?: number;
  latency_status?: "success" | "failed";
  latency_message?: string;
  ip_address?: string;
  country?: string;
  country_code?: string;
  region?: string;
  city?: string;
  quality_status?: "healthy" | "warn" | "challenge" | "failed";
  quality_score?: number;
  quality_grade?: string;
  quality_summary?: string;
  quality_checked?: number;
  created_at: string;
  updated_at: string;
}

export interface ProxyAccountSummary {
  id: number;
  name: string;
  platform: AccountPlatform;
  gateway_protocol?: GatewayProtocol;
  type: AccountType;
  notes?: string | null;
}

export interface ProxyQualityCheckItem {
  target: string;
  status: "pass" | "warn" | "fail" | "challenge";
  http_status?: number;
  latency_ms?: number;
  message?: string;
  cf_ray?: string;
}

export interface ProxyQualityCheckResult {
  proxy_id: number;
  score: number;
  grade: string;
  summary: string;
  exit_ip?: string;
  country?: string;
  country_code?: string;
  base_latency_ms?: number;
  passed_count: number;
  warn_count: number;
  failed_count: number;
  challenge_count: number;
  checked_at: number;
  items: ProxyQualityCheckItem[];
}

// Gemini credentials structure for OAuth and API Key authentication
export interface GeminiCredentials {
  // API Key authentication
  api_key?: string;

  // OAuth authentication
  access_token?: string;
  refresh_token?: string;
  oauth_type?: "code_assist" | "google_one" | "ai_studio" | string;
  tier_id?:
    | "google_one_free"
    | "google_ai_pro"
    | "google_ai_ultra"
    | "gcp_standard"
    | "gcp_enterprise"
    | "aistudio_free"
    | "aistudio_paid"
    | "LEGACY"
    | "PRO"
    | "ULTRA"
    | string;
  project_id?: string;
  token_type?: string;
  scope?: string;
  expires_at?: string;
  model_mapping?: Record<string, string>;
}

export interface TempUnschedulableRule {
  error_code: number;
  keywords: string[];
  duration_minutes: number;
  description: string;
}

export interface TempUnschedulableState {
  until_unix: number;
  triggered_at_unix: number;
  status_code: number;
  matched_keyword: string;
  rule_index: number;
  error_message: string;
}

export interface TempUnschedulableStatus {
  active: boolean;
  state?: TempUnschedulableState;
}

export interface Account {
  id: number;
  name: string;
  notes?: string | null;
  platform: AccountPlatform;
  gateway_protocol?: GatewayProtocol;
  type: AccountType;
  lifecycle_state?: AccountLifecycleState;
  lifecycle_reason_code?: string | null;
  lifecycle_reason_message?: string | null;
  blacklisted_at?: string | null;
  blacklist_purge_at?: string | null;
  credentials?: Record<string, unknown>;
  // Extra fields including Codex usage and model-level rate limits (Antigravity smart retry)
  extra?: CodexUsageSnapshot & {
    model_rate_limits?: Record<
      string,
      { rate_limited_at: string; rate_limit_reset_at: string }
    >;
  } & Record<string, unknown>;
  proxy_id: number | null;
  concurrency: number;
  load_factor?: number | null;
  current_concurrency?: number; // Real-time concurrency count from Redis
  priority: number;
  rate_multiplier?: number; // Account billing multiplier (>=0, 0 means free)
  status: "active" | "inactive" | "error";
  error_message: string | null;
  last_used_at: string | null;
  expires_at: number | null;
  auto_pause_on_expired: boolean;
  created_at: string;
  updated_at: string;
  proxy?: Proxy;
  group_ids?: number[]; // Groups this account belongs to
  groups?: Group[]; // Preloaded group objects

  // Rate limit & scheduling fields
  schedulable: boolean;
  rate_limited_at: string | null;
  rate_limit_reset_at: string | null;
  rate_limit_reason?: AccountRateLimitReason | null;
  overload_until: string | null;
  temp_unschedulable_until: string | null;
  temp_unschedulable_reason: string | null;

  // Session window fields (5-hour window)
  session_window_start: string | null;
  session_window_end: string | null;
  session_window_status: "allowed" | "allowed_warning" | "rejected" | null;

  // 5h缂備焦鍔栭〃鍛般亹濞戞碍瀚婚柛锔诲幗閺嗗繘鏌熺挩澶婂暙閻撴垿鏌ㄥ☉妯煎缂?Anthropic OAuth/SetupToken 闁荤姵鍔х粻鎴ｃ亹閸ф瀚夊璺侯儐濞呭繘鏌?
  window_cost_limit?: number | null;
  window_cost_sticky_reserve?: number | null;

  // 婵炴潙鍚嬫穱娲儊娴犲鏋佸ù鍏兼綑濞呫倝鏌熺挩澶婂暙閻撴垿鏌ㄥ☉妯煎缂?Anthropic OAuth/SetupToken 闁荤姵鍔х粻鎴ｃ亹閸ф瀚夊璺侯儐濞呭繘鏌?
  max_sessions?: number | null;
  session_idle_timeout_minutes?: number | null;

  // RPM 闂傚倸瀚崝鏇㈠春濡ゅ懏鏅柛顐ｇ矌閻?Anthropic OAuth/SetupToken 闁荤姵鍔х粻鎴ｃ亹閸ф瀚夊璺侯儐濞呭繘鏌?
  base_rpm?: number | null;
  rpm_strategy?: string | null;
  rpm_sticky_buffer?: number | null;
  user_msg_queue_mode?: string | null; // "serialize" | "throttle" | null

  // TLS闂佸湱顭堝ú銊バуΔ浣割嚤妞ゅ繐娴傚Λ鍛存煥濞戞澧旂紒?Anthropic OAuth/SetupToken 闁荤姵鍔х粻鎴ｃ亹閸ф瀚夊璺侯儐濞呭繘鏌?
  enable_tls_fingerprint?: boolean | null;
  tls_fingerprint_profile_id?: number | null;
  claude_code_mimic_enabled?: boolean | null;

  // 婵炴潙鍚嬫穱娲儊缁测偓D婵炲鈷堟禍锝壦夋繝鍥ㄦ櫖闁割偅绮庨惌?Anthropic OAuth/SetupToken 闁荤姵鍔х粻鎴ｃ亹閸ф瀚夊璺侯儐濞呭繘鏌?
  // 闂佸憡鍑归崹鎶藉极閵堝瑙﹂幖杈剧磿濞堟椽鏌?5闂佸憡甯掑Λ婵嬪箰閹捐绀冮柛娑卞幗缁佸ジ鎮?metadata.user_id 婵炴垶鎼╅崢鎯р枔?session ID
  session_id_masking_enabled?: boolean | null;

  // 缂傚倸鍊归幐鎼佹偤?TTL 閻庢鍠栭幖顐﹀春濡ゅ懎鍗抽柟绋块鎼村﹪鏌ㄥ☉妯煎缂?Anthropic OAuth/SetupToken 闁荤姵鍔х粻鎴ｃ亹閸ф瀚夊璺侯儐濞呭繘鏌?
  cache_ttl_override_enabled?: boolean | null;
  cache_ttl_override_target?: string | null;

  // API Key 闁荤姵鍔х粻鎴ｃ亹閸ф鐓€鐎广儱顦介弶娲⒒閸曨剙濮囬柛?
  quota_limit?: number | null;
  quota_used?: number | null;
  quota_daily_limit?: number | null;
  quota_daily_used?: number | null;
  quota_weekly_limit?: number | null;
  quota_weekly_used?: number | null;

  // 闁哄鏅滈崝姗€銆侀幋锕€绫嶉柛鎾茬绗戦梺璇″厸缁躲倗妲愬▎鎰浄闁告侗鍘剧粔濂告煕濮樼厧鐏ｉ柡浣靛€楅埀顒傛暩閹虫挾鑺遍弻銉︹挃闁归偊鍓欓悡鎴︽煛閸愨晛鍔剁紒缁樺灴瀹曞爼鎮滈崶鈺冾槴
  current_window_cost?: number | null; // 閻熸粎澧楅幐鍛婃櫠閻樼數鐜绘俊銈傚亾鐟滅増绋撻幏褰掑捶椤撶喐娈?
  active_sessions?: number | null; // 閻熸粎澧楅幐鍛婃櫠閻樿娲及韫囨洍鏀繛鏉戝悑娣囨椽鎯佹禒瀣瀬?
  current_rpm?: number | null; // 閻熸粎澧楅幐鍛婃櫠閻樿绀嗛柛鈩冪⊕鐎?RPM 闁荤姳璁查崜婵嬪汲?
}

export interface AccountStatusSummary {
  total: number;
  by_status: {
    active: number;
    inactive: number;
    error: number;
  };
  rate_limited: number;
  temp_unschedulable: number;
  overloaded: number;
  paused: number;
  in_use: number;
  remaining_available: number;
  by_platform: Partial<Record<AccountPlatform, number>>;
  limited_breakdown: {
    total: number;
    rate_429: number;
    usage_5h: number;
    usage_7d: number;
  };
}

export interface AccountRuntimeSummary {
  in_use: number;
}

// Account Usage types
export interface WindowStats {
  requests: number;
  tokens: number;
  cost: number; // Account cost (account multiplier)
  standard_cost?: number;
  user_cost?: number;
}

export interface UsageProgress {
  utilization: number; // Percentage (0-100+, 100 = 100%)
  resets_at: string | null;
  remaining_seconds: number;
  window_stats?: WindowStats | null; // 缂備焦鍔栭〃鍛般亹濞戙垹瀚夐柣鏃囨閸╃娀鎮规担鍙夘潐缂佽鲸鐟︾粋鎺撴償閿濆洤鐐婇梺鍛婄懕缁茬晫妲愰幋鐐村弿閻庯綆浜滈悡鍌滄喐閻楀牊灏褏濞€閹啴宕熼鍕ㄦ瀼闂佹椿娼块崝鎴﹀闯濞差亝鏅?
  used_requests?: number;
  limit_requests?: number;
}

// Antigravity 闂佸憡顨嗗ú鎴︽煂濠婂吘鐔煎灳瀹曞洠鍋撻悜鑺ュ剭闁告洦鍨扮敮鍐参涢悧鍫㈢畱濞ｅ洤锕獮?
export type AccountUsageRowColor = "indigo" | "emerald" | "purple" | "amber";

export interface AccountUsageResetRow {
  key: string;
  label: string;
  resetsAt: string | null;
  remainingSeconds?: number | null;
}

export interface AccountUsagePresentationRow extends AccountUsageResetRow {
  utilization: number;
  windowStats?: WindowStats | null;
  color: AccountUsageRowColor;
  inlineRemaining?: boolean;
}

export interface AccountUsagePresentationMeta {
  loadingRows: number;
  snapshotUpdatedAtText?: string;
  snapshotUpdatedAtTooltip?: string;
  sampledBadgeLabel?: string;
  sampledBadgeTooltip?: string;
  noteText?: string;
  antigravityTierLabel?: string | null;
  antigravityTierClass?: string;
  hasIneligibleTiers?: boolean;
  geminiAuthTypeLabel?: string | null;
  geminiTierClass?: string;
  geminiQuotaPolicyChannel?: string;
  geminiQuotaPolicyLimits?: string;
  geminiQuotaPolicyDocsUrl?: string;
}

export interface AccountUsagePresentation {
  loading: boolean;
  error: string | null;
  state: "bars" | "loading" | "error" | "empty" | "unlimited";
  windowRows: AccountUsagePresentationRow[];
  resetRows: AccountUsageResetRow[];
  meta: AccountUsagePresentationMeta;
}

export interface AntigravityModelQuota {
  utilization: number; // 婵炶揪缍€濞夋洟寮妶澶嬪仢?0-100
  reset_time: string; // 闂備焦褰冪粔鍫曟偪閸℃稑绫嶉柛顐ｆ礃閿?ISO8601
}

export interface AccountUsageInfo {
  source?: "passive" | "active";
  updated_at: string | null;
  five_hour: UsageProgress | null;
  seven_day: UsageProgress | null;
  seven_day_sonnet: UsageProgress | null;
  gemini_shared_daily?: UsageProgress | null;
  gemini_pro_daily?: UsageProgress | null;
  gemini_flash_daily?: UsageProgress | null;
  gemini_shared_minute?: UsageProgress | null;
  gemini_pro_minute?: UsageProgress | null;
  gemini_flash_minute?: UsageProgress | null;
  antigravity_quota?: Record<string, AntigravityModelQuota> | null;
  ai_credits?: Array<{
    credit_type?: string;
    amount?: number;
    minimum_balance?: number;
  }> | null;
  is_forbidden?: boolean;
  forbidden_reason?: string;
  forbidden_type?: string;
  validation_url?: string;
  needs_verify?: boolean;
  is_banned?: boolean;
  needs_reauth?: boolean;
  error_code?: string;
  error?: string;
}

// OpenAI Codex usage snapshot (from response headers)
export interface CodexUsageSnapshot {
  // Legacy fields (kept for backwards compatibility)
  // NOTE: The naming is ambiguous - actual window type is determined by window_minutes value
  codex_primary_used_percent?: number; // Usage percentage (check window_minutes for actual window type)
  codex_primary_reset_after_seconds?: number; // Seconds until reset
  codex_primary_window_minutes?: number; // Window in minutes
  codex_secondary_used_percent?: number; // Usage percentage (check window_minutes for actual window type)
  codex_secondary_reset_after_seconds?: number; // Seconds until reset
  codex_secondary_window_minutes?: number; // Window in minutes
  codex_primary_over_secondary_percent?: number; // Overflow ratio

  // Canonical fields (normalized by backend, use these preferentially)
  codex_5h_used_percent?: number; // 5-hour window usage percentage
  codex_5h_reset_after_seconds?: number; // Seconds until 5h window reset
  codex_5h_reset_at?: string; // 5-hour window absolute reset time (RFC3339)
  codex_5h_window_minutes?: number; // 5h window in minutes (should be ~300)
  codex_7d_used_percent?: number; // 7-day window usage percentage
  codex_7d_reset_after_seconds?: number; // Seconds until 7d window reset
  codex_7d_reset_at?: string; // 7-day window absolute reset time (RFC3339)
  codex_7d_window_minutes?: number; // 7d window in minutes (should be ~10080)

  codex_usage_updated_at?: string; // Last update timestamp
  openai_known_models?: string[];
  openai_known_models_updated_at?: string;
  openai_known_models_source?:
    | "import_models"
    | "test_probe"
    | "model_mapping"
    | string;
}

export interface CreateAccountRequest {
  name: string;
  notes?: string | null;
  platform: AccountPlatform;
  gateway_protocol?: GatewayProtocol;
  type: AccountType;
  lifecycle_state?: AccountLifecycleState;
  lifecycle_reason_code?: string | null;
  lifecycle_reason_message?: string | null;
  credentials: Record<string, unknown>;
  extra?: Record<string, unknown>;
  proxy_id?: number | null;
  concurrency?: number;
  load_factor?: number | null;
  priority?: number;
  rate_multiplier?: number; // Account billing multiplier (>=0, 0 means free)
  group_ids?: number[];
  expires_at?: number | null;
  auto_pause_on_expired?: boolean;
  confirm_mixed_channel_risk?: boolean;
}

export interface BatchArchiveAccountsRequest {
  account_ids: number[];
  group_name: string;
}

export interface BatchArchiveAccountsResult {
  archived_count: number;
  failed_count: number;
  archive_group_id: number;
  archive_group_name: string;
  success_ids?: number[];
  failed_ids?: number[];
}

export interface ArchiveGroupAccountsRequest {
  source_group_id: number;
  group_name: string;
}

export interface ArchiveGroupAccountsResult {
  source_group_id: number;
  source_group_name: string;
  archived_count: number;
  failed_count: number;
  archive_group_id: number;
  archive_group_name: string;
  archived_account_ids?: number[];
  failed_account_ids?: number[];
}

export interface ArchivedAccountGroupSummary {
  group_id: number;
  group_name: string;
  total_count: number;
  available_count: number;
  invalid_count: number;
  latest_updated_at: string;
}

export interface UnarchiveAccountResult {
  account_id: number;
  success: boolean;
  restored_group_ids?: number[];
  used_fallback_current_group: boolean;
  error_message?: string;
}

export interface UnarchiveAccountsResult {
  restored_count: number;
  failed_count: number;
  restored_to_original_group_count: number;
  restored_in_place_count: number;
  results: UnarchiveAccountResult[];
}

export interface UpdateAccountRequest {
  name?: string;
  notes?: string | null;
  gateway_protocol?: GatewayProtocol;
  type?: AccountType;
  lifecycle_state?: AccountLifecycleState;
  lifecycle_reason_code?: string | null;
  lifecycle_reason_message?: string | null;
  credentials?: Record<string, unknown>;
  extra?: Record<string, unknown>;
  proxy_id?: number | null;
  concurrency?: number;
  load_factor?: number | null;
  priority?: number;
  rate_multiplier?: number; // Account billing multiplier (>=0, 0 means free)
  schedulable?: boolean;
  status?: "active" | "inactive" | "error";
  group_ids?: number[];
  expires_at?: number | null;
  auto_pause_on_expired?: boolean;
  confirm_mixed_channel_risk?: boolean;
}

export interface CheckMixedChannelRequest {
  platform: AccountPlatform;
  gateway_protocol?: GatewayProtocol;
  group_ids: number[];
  account_id?: number;
}

export interface MixedChannelWarningDetails {
  group_id: number;
  group_name: string;
  current_platform: string;
  other_platform: string;
}

export interface CheckMixedChannelResponse {
  has_risk: boolean;
  error?: string;
  message?: string;
  details?: MixedChannelWarningDetails;
}

export interface CreateProxyRequest {
  name: string;
  protocol: ProxyProtocol;
  host: string;
  port: number;
  username?: string | null;
  password?: string | null;
}

export interface UpdateProxyRequest {
  name?: string;
  protocol?: ProxyProtocol;
  host?: string;
  port?: number;
  username?: string | null;
  password?: string | null;
  status?: "active" | "inactive";
}

export interface AdminDataPayload {
  type?: string;
  version?: number;
  exported_at: string;
  proxies: AdminDataProxy[];
  accounts: AdminDataAccount[];
}

export interface AdminDataProxy {
  proxy_key: string;
  name: string;
  protocol: ProxyProtocol;
  host: string;
  port: number;
  username?: string | null;
  password?: string | null;
  status: "active" | "inactive";
}

export interface AdminDataAccount {
  name: string;
  notes?: string | null;
  platform: AccountPlatform;
  type: AccountType;
  credentials: Record<string, unknown>;
  extra?: Record<string, unknown>;
  proxy_key?: string | null;
  concurrency: number;
  priority: number;
  rate_multiplier?: number | null;
  expires_at?: number | null;
  auto_pause_on_expired?: boolean;
}

export interface AdminDataImportError {
  kind: "proxy" | "account";
  name?: string;
  proxy_key?: string;
  message: string;
}

export interface AdminDataImportResult {
  proxy_created: number;
  proxy_reused: number;
  proxy_failed: number;
  account_created: number;
  account_failed: number;
  errors?: AdminDataImportError[];
}

// ==================== Usage & Redeem Types ====================

export type RedeemCodeType =
  | "balance"
  | "concurrency"
  | "subscription"
  | "invitation";
export type UsageRequestType = "unknown" | "sync" | "stream" | "ws_v2";
export type UsageLogStatus = "succeeded" | "failed";
export type UsageLogSimulatedClient = "codex" | "gemini_cli";
export type TokenDisplayMode = "full" | "compact";

export interface UsageLog {
  id: number;
  user_id: number;
  api_key_id: number;
  account_id: number | null;
  request_id: string;
  model: string;
  upstream_model?: string | null;
  service_tier?: string | null;
  reasoning_effort?: string | null;
  thinking_enabled?: boolean | null;
  inbound_endpoint?: string | null;
  upstream_endpoint?: string | null;

  group_id: number | null;
  subscription_id: number | null;

  input_tokens: number;
  output_tokens: number;
  cache_creation_tokens: number;
  cache_read_tokens: number;
  cache_creation_5m_tokens: number;
  cache_creation_1h_tokens: number;

  input_cost: number;
  output_cost: number;
  cache_creation_cost: number;
  cache_read_cost: number;
  total_cost: number;
  actual_cost: number;
  billing_exempt_reason?: "admin_free" | null;
  rate_multiplier: number;
  billing_type: number;

  request_type?: UsageRequestType;
  status: UsageLogStatus;
  stream: boolean;
  openai_ws_mode?: boolean;
  duration_ms: number;
  first_token_ms: number | null;
  http_status?: number | null;
  error_code?: string | null;
  error_message?: string | null;
  simulated_client?: UsageLogSimulatedClient | null;

  // 闂佹悶鍎辨晶鑺ユ櫠閺嶎厽鍋ㄩ柣鏃傤焾閻忓洭鎮楀☉娆樻畷妞?
  image_count: number;
  image_size: string | null;

  // User-Agent
  user_agent: string | null;

  // Cache TTL Override
  cache_ttl_overridden: boolean;

  created_at: string;

  user?: User;
  api_key?: ApiKey;
  group?: Group;
  subscription?: UserSubscription;
}

export interface UsageLogAccountSummary {
  id: number;
  name: string;
}

export interface AdminUsageLog extends UsageLog {
  // 闁荤姵鍔х粻鎴ｃ亹鐠恒劍濯奸梽鍥垂閸岀偛纾圭€广儱娲ら懞鎶芥煥濞戞澧旂紒顔兼捣缁鏁嶉崟顒€鈧偤鏌涘☉娅亣銇愰懠顒佸枂濞撴艾锕︾粈?
  account_rate_multiplier?: number | null;

  // 闂佹椿娼块崝宥夊春濞戞碍瀚氶梺鍨儑濠€?IP闂佹寧绋戦悧鍛垝鎼达絿涓嶉柨娑樺閸婄偤鏌涘☉娅亣銇愰懠顒佸枂濞撴艾锕︾粈?
  ip_address?: string | null;

  // 闂佸搫鐗冮崑鎾绘倶韫囨挾绠哄璺哄瀹曪綁鎽庨崒娆戠畾闂佽鍙庨崹鐣屾濞嗘劗顩烽柛娑卞灱閸氣偓闂佽崵鍋涘Λ妤呭箟閹惰棄绠抽柕澶堝劚缂嶆捇寮堕埡鍌涚叆婵炲弶鐗犻弫?
  account?: UsageLogAccountSummary;
}

export interface UsageCleanupFilters {
  start_time: string;
  end_time: string;
  user_id?: number;
  api_key_id?: number;
  account_id?: number;
  group_id?: number;
  model?: string | null;
  request_type?: UsageRequestType | null;
  stream?: boolean | null;
  billing_type?: number | null;
}

export interface UsageCleanupTask {
  id: number;
  status: string;
  filters: UsageCleanupFilters;
  created_by: number;
  deleted_rows: number;
  error_message?: string | null;
  canceled_by?: number | null;
  canceled_at?: string | null;
  started_at?: string | null;
  finished_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface RedeemCode {
  id: number;
  code: string;
  type: RedeemCodeType;
  value: number;
  status: "active" | "used" | "expired" | "unused";
  used_by: number | null;
  used_at: string | null;
  created_at: string;
  updated_at?: string;
  group_id?: number | null; // 闁荤姳闄嶉崹钘壩ｉ崟顓犲暗閻犲洩灏欓埀顒勬敱缁嬪骞橀懜鍨
  validity_days?: number; // 闁荤姳闄嶉崹钘壩ｉ崟顓犲暗閻犲洩灏欓埀顒勬敱缁嬪骞橀懜鍨
  user?: User;
  group?: Group; // 闂佺绻愰悿鍥ㄧ閸儲鍎嶉柛鏇ㄥ亜閻庤崵绱?
}

export interface GenerateRedeemCodesRequest {
  count: number;
  type: RedeemCodeType;
  value: number;
  group_id?: number | null; // 闁荤姳闄嶉崹钘壩ｉ崟顓犲暗閻犲洩灏欓埀顒勬敱缁嬪骞橀懜鍨
  validity_days?: number; // 闁荤姳闄嶉崹钘壩ｉ崟顓犲暗閻犲洩灏欓埀顒勬敱缁嬪骞橀懜鍨
}

export interface RedeemCodeRequest {
  code: string;
}

// ==================== Dashboard & Statistics ====================

export interface DashboardStats {
  // 闂佹椿娼块崝宥夊春濞戞氨纾奸柣鏃€妞块崥鈧?
  total_users: number;
  today_new_users: number; // 婵炲濮撮敃銉ノ涢埡鍛闁哄顑欓弶濠氭煟椤剙濡介柛鈺傜洴瀵?
  active_users: number; // 婵炲濮撮敃銉ノ涢埡鍛珘濠㈣泛瀵掗崵鐐存叏閻熸澘鈧鈻撻幋锔藉仺闁靛绠戦悡鏇㈡煛?
  hourly_active_users: number; // 閻熸粎澧楅幐鍛婃櫠閻樺吀鐒婇煫鍥ㄦ⒐椤ρ勭箾閸欏顫楃紒宀婂墴閹粙濡搁敃鈧悡鏇㈡煛娴ｇ绨荤紒杈ㄢ攼TC闂?
  stats_updated_at: string; // 缂傚倷鑳堕崰鏇㈩敇閹间礁鍗抽悗娑櫳戦悡鈧梺鍝勫暙閻栫厧螞閸ф鏅柛锔炬緭C RFC3339闂?
  stats_stale: boolean; // 缂傚倷鑳堕崰鏇㈩敇閹间礁鍙婃い鏍ㄧ閸庡﹪寮堕埡浣瑰婵犫偓?

  // API Key 缂傚倷鑳堕崰鏇㈩敇?
  total_api_keys: number;
  active_api_keys: number; // 闂佺粯顭堥崺鏍焵椤戞寧顦烽悹?active 闂?API Key 闂?

  // 闁荤姵鍔ч梽鍕春濞戞氨纾奸柣鏃€妞块崥鈧?
  total_accounts: number;
  normal_accounts: number; // 濠殿喗绻愮徊浠嬫偉閸撲焦瀚婚柨鏃囨閻撴洟鏌?
  error_accounts: number; // 閻庢鍠栭崐鎼佹偉閸撲焦瀚婚柨鏃囨閻撴洟鏌?
  ratelimit_accounts: number; // 闂傚倸瀚崝鏍矈閿旂偓瀚婚柨鏃囨閻撴洟鏌?
  overload_accounts: number; // 闁哄鏅涘ú鈺伱归崶鈺傚闁挎棁妫勯悡鏇㈡煛?

  // 缂備線纭搁崹鐢割敇?Token 婵炶揪缍€濞夋洟寮妶鍥╃＜闁绘梹妞块崥鈧?
  total_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_cache_creation_tokens: number;
  total_cache_read_tokens: number;
  total_tokens: number;
  total_cost: number; // 缂備線纭搁崹鐢割敇閹间礁鍐€闁搞儜鍐╃彲闁荤姳绫嶉妶鍛偓?
  total_actual_cost: number; // 缂備線纭搁崹鐢割敇閸濄儮鍋撻崷顓炰槐婵＄虎鍨堕獮宥夋晲婢跺褰?

  // 婵炲濮撮敃銉ノ?Token 婵炶揪缍€濞夋洟寮妶鍥╃＜闁绘梹妞块崥鈧?
  today_requests: number;
  today_input_tokens: number;
  today_output_tokens: number;
  today_cache_creation_tokens: number;
  today_cache_read_tokens: number;
  today_tokens: number;
  today_cost: number; // 婵炲濮撮敃銉ノ涢埡鍛唨闁搞儜鍐╃彲闁荤姳绫嶉妶鍛偓?
  today_actual_cost: number; // 婵炲濮撮敃銉ノ涢埡鍐ｅ亾閸︻厼浠辨俊缁㈠灦楠炲秹鏁愭径瀣綉

  // 缂備緡鍨靛畷鐢靛垝閻戞ɑ浜ら柟閭﹀灱閺€鐣岀磽娴ｅ搫鏋欐い?
  average_duration_ms: number; // 濡ょ姷鍋涢崯鍨焽鎼淬劌浼犵€广儱鎳愮€瑰鏌￠崘銊у煟婵?
  uptime: number; // 缂備緡鍨靛畷鐢靛垝閻戞ɑ浜ら柟閭﹀灱閺€浠嬫煛閸愩劎鍩ｆ俊?缂?

  // 闂佽鍎搁崱妤€骞嬮梺鍦焾濞诧箓鎮?
  rpm: number; // 闁?闂佸憡甯掑Λ婵嬪箰閹捐崵宓侀柛鎰级缂嶅棙鎱ㄩ敐鍛闁搞劌閰ｉ弻锕傛偄濞茶娈插┑顔炬嚀閸婂綊寮?
  tpm: number; // 闁?闂佸憡甯掑Λ婵嬪箰閹捐崵宓侀柛鎰级缂嶅棙鎱ㄩ敐鍛闁搞劌閰ｉ弻锕傛倻缁扁偓ken闂?
}

export interface UsageStatsResponse {
  period?: string;
  total_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_cache_tokens: number;
  total_tokens: number;
  total_cost: number;
  total_actual_cost: number;
  admin_free_requests: number;
  admin_free_standard_cost: number;
  average_duration_ms: number;
  models?: Record<string, number>;
}

// ==================== Trend & Chart Types ====================

export interface TrendDataPoint {
  date: string;
  requests: number;
  input_tokens: number;
  output_tokens: number;
  cache_creation_tokens: number;
  cache_read_tokens: number;
  total_tokens: number;
  cost: number; // 闂佸搫绉村ú銈夊闯椤栨粍濯奸梽鍥垂?
  actual_cost: number; // 闁诲骸婀遍崑銈咁瀶椤栫偛绠ラ柨婵嗩槹閻?
}

export interface ModelStat {
  model: string;
  requests: number;
  input_tokens: number;
  output_tokens: number;
  cache_creation_tokens: number;
  cache_read_tokens: number;
  total_tokens: number;
  cost: number; // 闂佸搫绉村ú銈夊闯椤栨粍濯奸梽鍥垂?
  actual_cost: number; // 闁诲骸婀遍崑銈咁瀶椤栫偛绠ラ柨婵嗩槹閻?
}

export interface EndpointStat {
  endpoint: string;
  requests: number;
  total_tokens: number;
  cost: number;
  actual_cost: number;
}

export interface GroupStat {
  group_id: number;
  group_name: string;
  requests: number;
  total_tokens: number;
  cost: number; // 闂佸搫绉村ú銈夊闯椤栨粍濯奸梽鍥垂?
  actual_cost: number; // 闁诲骸婀遍崑銈咁瀶椤栫偛绠ラ柨婵嗩槹閻?
}

export interface UserBreakdownItem {
  user_id: number
  email: string
  requests: number
  total_tokens: number
  cost: number
  actual_cost: number
}

export interface UserUsageTrendPoint {
  date: string;
  user_id: number;
  email: string;
  username?: string;
  requests: number;
  tokens: number;
  cost: number; // 闂佸搫绉村ú銈夊闯椤栨粍濯奸梽鍥垂?
  actual_cost: number; // 闁诲骸婀遍崑銈咁瀶椤栫偛绠ラ柨婵嗩槹閻?
}

export interface UserSpendingRankingItem {
  user_id: number;
  email: string;
  username?: string;
  actual_cost: number;
  requests: number;
  tokens: number;
}

export interface UserSpendingRankingResponse {
  ranking: UserSpendingRankingItem[];
  total_actual_cost: number;
  total_requests: number;
  total_tokens: number;
  start_date: string;
  end_date: string;
}

export interface ApiKeyUsageTrendPoint {
  date: string;
  api_key_id: number;
  key_name: string;
  requests: number;
  tokens: number;
}

// ==================== Admin User Management ====================

export interface UpdateUserRequest {
  email?: string;
  password?: string;
  username?: string;
  notes?: string;
  role?: "admin" | "user";
  admin_free_billing?: boolean;
  balance?: number;
  concurrency?: number;
  status?: "active" | "disabled";
  allowed_groups?: number[] | null;
  // 闂佹椿娼块崝宥夊春濞戞瑧鈻旈柟鎯х－濞硷綁鏌涢幒鎴烆棤缂侇喖绉瑰畷鎰吋閸パ嗗У闂備焦婢樼粔鍫曟偪?(group_id -> rate_multiplier | null)
  // null 闁荤偞绋忛崝搴ㄥΦ濮樿泛绀嗛柣妯肩帛閻濈喖鎮归崶銉ュ姎闁搞劌娴风槐鎺楀礋椤撶喓鏆犳繛鎴炴尰閹告悂鎯堝鈧畷鎰吋閸パ嗗У
  group_rates?: Record<number, number | null>;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

// ==================== User Subscription Types ====================

export interface UserSubscription {
  id: number;
  user_id: number;
  group_id: number;
  status: "active" | "expired" | "revoked";
  daily_usage_usd: number;
  weekly_usage_usd: number;
  monthly_usage_usd: number;
  daily_window_start: string | null;
  weekly_window_start: string | null;
  monthly_window_start: string | null;
  created_at: string;
  updated_at: string;
  expires_at: string | null;
  user?: User;
  group?: Group;
}

export interface SubscriptionProgress {
  subscription_id: number;
  daily: {
    used: number;
    limit: number | null;
    percentage: number;
    reset_in_seconds: number | null;
  } | null;
  weekly: {
    used: number;
    limit: number | null;
    percentage: number;
    reset_in_seconds: number | null;
  } | null;
  monthly: {
    used: number;
    limit: number | null;
    percentage: number;
    reset_in_seconds: number | null;
  } | null;
  expires_at: string | null;
  days_remaining: number | null;
}

export interface AssignSubscriptionRequest {
  user_id: number;
  group_id: number;
  validity_days?: number;
}

export interface BulkAssignSubscriptionRequest {
  user_ids: number[];
  group_id: number;
  validity_days?: number;
}

export interface ExtendSubscriptionRequest {
  days: number;
}

// ==================== Query Parameters ====================

export interface UsageQueryParams {
  page?: number;
  page_size?: number;
  api_key_id?: number;
  user_id?: number;
  account_id?: number;
  group_id?: number;
  model?: string;
  request_type?: UsageRequestType;
  stream?: boolean;
  billing_type?: number | null;
  start_date?: string;
  end_date?: string;
}

// ==================== Account Usage Statistics ====================

export interface AccountUsageHistory {
  date: string;
  label: string;
  requests: number;
  tokens: number;
  cost: number;
  actual_cost: number; // Account cost (account multiplier)
  user_cost: number; // User/API key billed cost (group multiplier)
}

export interface AccountUsageSummary {
  days: number;
  actual_days_used: number;
  total_cost: number; // Account cost (account multiplier)
  total_user_cost: number;
  total_standard_cost: number;
  total_requests: number;
  total_tokens: number;
  avg_daily_cost: number; // Account cost
  avg_daily_user_cost: number;
  avg_daily_requests: number;
  avg_daily_tokens: number;
  avg_duration_ms: number;
  today: {
    date: string;
    cost: number;
    user_cost: number;
    requests: number;
    tokens: number;
  } | null;
  highest_cost_day: {
    date: string;
    label: string;
    cost: number;
    user_cost: number;
    requests: number;
  } | null;
  highest_request_day: {
    date: string;
    label: string;
    requests: number;
    cost: number;
    user_cost: number;
  } | null;
}

export interface AccountUsageStatsResponse {
  history: AccountUsageHistory[];
  summary: AccountUsageSummary;
  models: ModelStat[];
  endpoints: EndpointStat[];
  upstream_endpoints: EndpointStat[];
}

// ==================== User Attribute Types ====================

export type UserAttributeType =
  | "text"
  | "textarea"
  | "number"
  | "email"
  | "url"
  | "date"
  | "select"
  | "multi_select";

export interface UserAttributeOption {
  value: string;
  label: string;
  [key: string]: unknown;
}

export interface UserAttributeValidation {
  min_length?: number;
  max_length?: number;
  min?: number;
  max?: number;
  pattern?: string;
  message?: string;
}

export interface UserAttributeDefinition {
  id: number;
  key: string;
  name: string;
  description: string;
  type: UserAttributeType;
  options: UserAttributeOption[];
  required: boolean;
  validation: UserAttributeValidation;
  placeholder: string;
  display_order: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface UserAttributeValue {
  id: number;
  user_id: number;
  attribute_id: number;
  value: string;
  created_at: string;
  updated_at: string;
}

export interface CreateUserAttributeRequest {
  key: string;
  name: string;
  description?: string;
  type: UserAttributeType;
  options?: UserAttributeOption[];
  required?: boolean;
  validation?: UserAttributeValidation;
  placeholder?: string;
  display_order?: number;
  enabled?: boolean;
}

export interface UpdateUserAttributeRequest {
  key?: string;
  name?: string;
  description?: string;
  type?: UserAttributeType;
  options?: UserAttributeOption[];
  required?: boolean;
  validation?: UserAttributeValidation;
  placeholder?: string;
  display_order?: number;
  enabled?: boolean;
}

export interface UserAttributeValuesMap {
  [attributeId: number]: string;
}

// ==================== Promo Code Types ====================

export interface PromoCode {
  id: number;
  code: string;
  bonus_amount: number;
  max_uses: number;
  used_count: number;
  status: "active" | "disabled";
  expires_at: string | null;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

export interface PromoCodeUsage {
  id: number;
  promo_code_id: number;
  user_id: number;
  bonus_amount: number;
  used_at: string;
  user?: User;
}

export interface CreatePromoCodeRequest {
  code?: string;
  bonus_amount: number;
  max_uses?: number;
  expires_at?: number | null;
  notes?: string;
}

export interface UpdatePromoCodeRequest {
  code?: string;
  bonus_amount?: number;
  max_uses?: number;
  status?: "active" | "disabled";
  expires_at?: number | null;
  notes?: string;
}

// ==================== TOTP (2FA) Types ====================

export interface TotpStatus {
  enabled: boolean;
  enabled_at: number | null; // Unix timestamp in seconds
  feature_enabled: boolean;
}

export interface TotpSetupRequest {
  email_code?: string;
  password?: string;
}

export interface TotpSetupResponse {
  secret: string;
  qr_code_url: string;
  setup_token: string;
  countdown: number;
}

export interface TotpEnableRequest {
  totp_code: string;
  setup_token: string;
}

export interface TotpEnableResponse {
  success: boolean;
}

export interface TotpDisableRequest {
  email_code?: string;
  password?: string;
}

export interface TotpVerificationMethod {
  method: "email" | "password";
}

export interface TotpLoginResponse {
  requires_2fa: boolean;
  temp_token?: string;
  user_email_masked?: string;
}

export interface TotpLogin2FARequest {
  temp_token: string;
  totp_code: string;
}

// ==================== Scheduled Test Types ====================

export interface ScheduledTestPlan {
  id: number;
  account_id: number;
  model_id: string;
  cron_expression: string;
  enabled: boolean;
  max_results: number;
  auto_recover: boolean;
  notify_policy: "none" | "always" | "failure_only";
  notify_failure_threshold: number;
  retry_interval_minutes: number;
  max_retries: number;
  consecutive_failures: number;
  current_retry_count: number;
  last_run_at: string | null;
  next_run_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ScheduledTestResult {
  id: number;
  plan_id: number;
  status: string;
  response_text: string;
  error_message: string;
  latency_ms: number;
  started_at: string;
  finished_at: string;
  created_at: string;
}

export interface CreateScheduledTestPlanRequest {
  account_id: number;
  model_id: string;
  cron_expression: string;
  enabled?: boolean;
  max_results?: number;
  auto_recover?: boolean;
  notify_policy?: "none" | "always" | "failure_only";
  notify_failure_threshold?: number;
  retry_interval_minutes?: number;
  max_retries?: number;
}

export interface UpdateScheduledTestPlanRequest {
  model_id?: string;
  cron_expression?: string;
  enabled?: boolean;
  max_results?: number;
  auto_recover?: boolean;
  notify_policy?: "none" | "always" | "failure_only";
  notify_failure_threshold?: number;
  retry_interval_minutes?: number;
  max_retries?: number;
}
