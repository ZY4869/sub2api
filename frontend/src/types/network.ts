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
