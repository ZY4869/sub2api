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
