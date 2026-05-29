export type PaymentProductType = "balance_topup" | "subscription";

export type PaymentOrderStatus =
  | "created"
  | "pending"
  | "paid"
  | "failed"
  | "cancelled"
  | "expired"
  | "partial_refunded"
  | "refunded";

export interface PaymentSubscriptionPlan {
  plan_id: string;
  name: string;
  group_id: number;
  validity_days: number;
  prices_by_currency: Record<string, number>;
  enabled: boolean;
}

export interface PaymentOrder {
  order_no: string;
  user_id?: number;
  product_type: PaymentProductType;
  status: PaymentOrderStatus;
  provider: string;
  provider_env: "demo" | "prod" | string;
  amount_minor: number;
  amount: number;
  refunded_amount_minor?: number;
  refunded_amount?: number;
  refundable_amount_minor?: number;
  refundable_amount?: number;
  currency: string;
  country_code?: string;
  provider_intent_id?: string;
  snapshot?: Record<string, unknown>;
  paid_at?: string | null;
  refunded_at?: string | null;
  expires_at?: string | null;
  created_at?: string;
  updated_at?: string;
}

export interface PaymentCreateOrderRequest {
  product_type: PaymentProductType;
  amount?: number;
  currency: string;
  country_code?: string;
  plan_id?: string;
  return_url?: string;
}

export interface PaymentCreateOrderResponse {
  order: PaymentOrder;
  client_secret: string;
  client_id: string;
  intent_id: string;
  resume_token: string;
  provider_env: "demo" | "prod" | string;
  payment_mode?: "default" | "qrcode" | string;
}

export interface PaymentResumeOrderResponse {
  order: PaymentOrder;
  client_secret: string;
  client_id: string;
  intent_id: string;
  provider_env: "demo" | "prod" | string;
  payment_mode?: "default" | "qrcode" | string;
}

export interface PaymentRefund {
  refund_no: string;
  order_no: string;
  provider_refund_id: string;
  amount_minor: number;
  currency: string;
  status: "received" | "accepted" | "settled" | "failed" | string;
  created_at?: string;
}
