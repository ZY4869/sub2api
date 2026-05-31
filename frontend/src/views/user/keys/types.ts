import type { EditableApiKeyGroupBinding } from "@/components/keys/apiKeyGroupBindings";
import type { TimeAccessPolicy } from "@/types/api-key-groups";

export const imageCountWeightTiers = ["1K", "2K", "4K"] as const;

export type ImageCountWeightTier = (typeof imageCountWeightTiers)[number];

export interface ApiKeyFormData {
  name: string;
  group_bindings: EditableApiKeyGroupBinding[];
  status: "active" | "inactive";
  use_custom_key: boolean;
  custom_key: string;
  enable_ip_restriction: boolean;
  ip_whitelist: string;
  ip_blacklist: string;
  enable_quota: boolean;
  quota: number | null;
  image_only_enabled: boolean;
  image_count_billing_enabled: boolean;
  image_max_count: number | null;
  image_count_weights: Record<ImageCountWeightTier, number>;
  enable_rate_limit: boolean;
  rate_limit_5h: number | null;
  rate_limit_1d: number | null;
  rate_limit_7d: number | null;
  enable_expiration: boolean;
  expiration_preset: "7" | "30" | "90" | "custom";
  expiration_date: string;
  enable_starts_at: boolean;
  starts_at: string;
  enable_time_access: boolean;
  time_access_preset: "daytime" | "deep_night" | "eight_hours" | "twelve_hours" | "business_days_daytime" | "custom";
  access_time_policy: TimeAccessPolicy;
}
