import type { User } from './auth'
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
