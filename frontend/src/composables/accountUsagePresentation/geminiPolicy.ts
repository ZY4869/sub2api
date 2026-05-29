import type { ComposerTranslation } from "vue-i18n";
import type { GeminiUsageChannel } from "./geminiHelpers";

export function resolveGeminiQuotaPolicyChannel(
  channel: GeminiUsageChannel,
  t: ComposerTranslation,
): string {
  if (channel === "google_one") {
    return t("admin.accounts.gemini.quotaPolicy.rows.googleOne.channel");
  }
  if (channel === "gcp") {
    return t("admin.accounts.gemini.quotaPolicy.rows.gcp.channel");
  }
  if (channel === "vertex_ai") {
    return t("admin.accounts.gemini.quotaPolicy.rows.vertex.channel");
  }
  if (channel === "ai_studio_client") {
    return t("admin.accounts.gemini.quotaPolicy.rows.customOAuth.channel");
  }
  return t("admin.accounts.gemini.quotaPolicy.rows.aiStudio.channel");
}

export function resolveGeminiQuotaPolicyLimits(
  tierID: string | null,
  channel: GeminiUsageChannel,
  userLevel: string | null,
  t: ComposerTranslation,
): string {
  const tierLower = (tierID || "").toString().trim().toLowerCase();

  if (channel === "google_one") {
    if (tierLower === "google_ai_ultra" || userLevel === "ultra") {
      return t("admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsUltra");
    }
    if (tierLower === "google_ai_pro" || userLevel === "pro") {
      return t("admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsPro");
    }
    return t("admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsFree");
  }

  if (channel === "gcp") {
    if (tierLower === "gcp_enterprise" || userLevel === "enterprise") {
      return t("admin.accounts.gemini.quotaPolicy.rows.gcp.limitsEnterprise");
    }
    return t("admin.accounts.gemini.quotaPolicy.rows.gcp.limitsStandard");
  }

  if (channel === "vertex_ai") {
    return t("admin.accounts.gemini.quotaPolicy.rows.vertex.limits");
  }

  if (channel === "ai_studio_client") {
    if (tierLower === "aistudio_tier_3" || userLevel === "tier_3") {
      return t("admin.accounts.gemini.quotaPolicy.rows.customOAuth.limitsTier3");
    }
    if (tierLower === "aistudio_tier_2" || userLevel === "tier_2") {
      return t("admin.accounts.gemini.quotaPolicy.rows.customOAuth.limitsTier2");
    }
    if (
      tierLower === "aistudio_tier_1" ||
      tierLower === "aistudio_paid" ||
      userLevel === "tier_1"
    ) {
      return t("admin.accounts.gemini.quotaPolicy.rows.customOAuth.limitsPaid");
    }
    return t("admin.accounts.gemini.quotaPolicy.rows.customOAuth.limitsFree");
  }

  if (tierLower === "aistudio_tier_3" || userLevel === "tier_3") {
    return t("admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsTier3");
  }
  if (tierLower === "aistudio_tier_2" || userLevel === "tier_2") {
    return t("admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsTier2");
  }
  if (
    tierLower === "aistudio_tier_1" ||
    tierLower === "aistudio_paid" ||
    userLevel === "tier_1"
  ) {
    return t("admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsTier1");
  }
  return t("admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsFree");
}

export function resolveGeminiQuotaPolicyDocsUrl(
  channel: GeminiUsageChannel,
): string {
  if (channel === "google_one" || channel === "gcp") {
    return "https://developers.google.com/gemini-code-assist/resources/quotas";
  }
  if (channel === "vertex_ai") {
    return "https://cloud.google.com/vertex-ai/generative-ai/docs/quotas";
  }
  return "https://ai.google.dev/pricing";
}
