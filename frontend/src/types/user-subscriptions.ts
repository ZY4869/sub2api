import type { APIKeyModelBindingMode, Group, TimeAccessPolicy } from './api-key-groups'
import type { User } from './auth'
// ==================== Admin User Management ====================

export interface UpdateUserRequest {
  email?: string;
  password?: string;
  username?: string;
  notes?: string;
  role?: "admin" | "user";
  api_key_model_binding_mode?: APIKeyModelBindingMode;
  api_key_access_time_policy?: TimeAccessPolicy | null;
  clear_api_key_access_time_policy?: boolean;
  request_details_review?: boolean;
  admin_free_billing?: boolean;
  balance?: number;
  concurrency?: number;
  status?: "active" | "disabled";
  allowed_groups?: number[] | null;
  // Per-group billing overrides keyed by group_id. Use null to clear an override.
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
  daily_usage_by_currency?: Record<string, number>;
  weekly_usage_by_currency?: Record<string, number>;
  monthly_usage_by_currency?: Record<string, number>;
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
