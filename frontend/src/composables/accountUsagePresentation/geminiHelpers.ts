export type GeminiUsageChannel =
  | "google_one"
  | "gcp"
  | "vertex_ai"
  | "ai_studio_client"
  | "ai_studio"
  | string
  | null;

export function resolveGeminiUserLevel(
  tierID: string | null,
  channel: GeminiUsageChannel,
): string | null {
  const tier = (tierID || "").toString().trim();
  const tierLower = tier.toLowerCase();
  const tierUpper = tier.toUpperCase();

  if (channel === "google_one") {
    if (tierLower === "google_one_free") return "free";
    if (tierLower === "google_ai_pro") return "pro";
    if (tierLower === "google_ai_ultra") return "ultra";
    if (tierUpper === "AI_PREMIUM" || tierUpper === "GOOGLE_ONE_STANDARD")
      return "pro";
    if (tierUpper === "GOOGLE_ONE_UNLIMITED") return "ultra";
    if (
      tierUpper === "FREE" ||
      tierUpper === "GOOGLE_ONE_BASIC" ||
      tierUpper === "GOOGLE_ONE_UNKNOWN" ||
      tierUpper === ""
    ) {
      return "free";
    }
    return null;
  }

  if (channel === "gcp") {
    if (tierLower === "gcp_enterprise") return "enterprise";
    if (tierLower === "gcp_standard") return "standard";
    if (tierUpper.includes("ULTRA") || tierUpper.includes("ENTERPRISE"))
      return "enterprise";
    return "standard";
  }

  if (channel === "ai_studio" || channel === "ai_studio_client") {
    if (tierLower === "aistudio_tier_3") return "tier_3";
    if (tierLower === "aistudio_tier_2") return "tier_2";
    if (tierLower === "aistudio_tier_1" || tierLower === "aistudio_paid")
      return "tier_1";
    if (tierLower === "aistudio_free") return "free";
    if (tierUpper.includes("TIER_3")) return "tier_3";
    if (tierUpper.includes("TIER_2")) return "tier_2";
    if (
      tierUpper.includes("PAID") ||
      tierUpper.includes("PAYG") ||
      tierUpper.includes("PAY") ||
      tierUpper.includes("TIER_1")
    )
      return "tier_1";
    if (tierUpper.includes("FREE")) return "free";
    if (channel === "ai_studio") return "free";
    return null;
  }

  return null;
}

export function resolveGeminiAuthTypeLabel(
  channelShort: string | null,
  channel: GeminiUsageChannel,
  userLevel: string | null,
): string | null {
  if (!channelShort) return null;
  let levelLabel = userLevel;
  if (channel === "ai_studio" || channel === "ai_studio_client") {
    switch (userLevel) {
      case "tier_3":
        levelLabel = "Tier 3";
        break;
      case "tier_2":
        levelLabel = "Tier 2";
        break;
      case "tier_1":
        levelLabel = "Tier 1";
        break;
      case "free":
        levelLabel = "Free";
        break;
    }
  }
  return levelLabel ? `${channelShort} ${levelLabel}` : channelShort;
}

export function resolveGeminiTierClass(
  channelShort: string | null,
  userLevel: string | null,
): string {
  if (channelShort === "AI Studio Client" || channelShort === "AI Studio") {
    if (userLevel === "tier_3")
      return "bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300";
    if (userLevel === "tier_2")
      return "bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300";
    if (userLevel === "tier_1")
      return "bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300";
    return "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300";
  }
  if (channelShort === "Vertex AI") {
    return "bg-sky-100 text-sky-700 dark:bg-sky-900/40 dark:text-sky-300";
  }
  if (channelShort === "Google One") {
    if (userLevel === "ultra")
      return "bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300";
    if (userLevel === "pro")
      return "bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300";
    return "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300";
  }
  if (channelShort === "GCP") {
    if (userLevel === "enterprise")
      return "bg-purple-100 text-purple-600 dark:bg-purple-900/40 dark:text-purple-300";
    return "bg-blue-100 text-blue-600 dark:bg-blue-900/40 dark:text-blue-300";
  }
  return "";
}
