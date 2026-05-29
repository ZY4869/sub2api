import { computed, type ComputedRef } from "vue";
import type { ComposerTranslation } from "vue-i18n";
import type { Account, AccountUsageInfo, GeminiCredentials } from "@/types";
import {
  resolveAccountGatewayProtocol,
  resolveGatewayProtocolLabel,
} from "@/utils/accountProtocolGateway";
import {
  resolveGeminiChannel,
  resolveGeminiChannelDisplayName,
} from "@/utils/geminiAccount";
import {
  resolveGeminiAuthTypeLabel,
  resolveGeminiTierClass,
  resolveGeminiUserLevel,
} from "./geminiHelpers";
import {
  resolveGeminiQuotaPolicyChannel,
  resolveGeminiQuotaPolicyDocsUrl,
  resolveGeminiQuotaPolicyLimits,
} from "./geminiPolicy";
import { buildProgressRow, buildRows } from "./rows";
import { getRuntimePlatform } from "./support";

export function useGeminiUsagePresentationState(
  account: ComputedRef<Account>,
  usageInfo: ComputedRef<AccountUsageInfo | null>,
  t: ComposerTranslation,
) {
  const geminiTier = computed(() => {
    if (getRuntimePlatform(account.value) !== "gemini") return null;
    const credentials = account.value.credentials as
      | GeminiCredentials
      | undefined;
    return credentials?.tier_id || null;
  });

  const geminiOAuthType = computed(() => {
    if (getRuntimePlatform(account.value) !== "gemini") return null;
    const credentials = account.value.credentials as
      | GeminiCredentials
      | undefined;
    return (credentials?.oauth_type || "").trim() || null;
  });

  const geminiChannel = computed(() => {
    if (getRuntimePlatform(account.value) !== "gemini") return null;
    return resolveGeminiChannel({
      type: account.value.type,
      credentials: account.value.credentials as GeminiCredentials | undefined,
    });
  });

  const isGeminiCodeAssist = computed(() => {
    return geminiChannel.value === "gcp";
  });

  const geminiChannelShort = computed((): string | null => {
    if (getRuntimePlatform(account.value) !== "gemini") return null;
    return resolveGeminiChannelDisplayName(geminiChannel.value);
  });

  const geminiUserLevel = computed((): string | null => {
    if (getRuntimePlatform(account.value) !== "gemini") return null;
    return resolveGeminiUserLevel(geminiTier.value, geminiChannel.value);
  });

  const geminiAuthTypeLabel = computed(() => {
    if (
      getRuntimePlatform(account.value) !== "gemini" ||
      !geminiChannelShort.value
    )
      return null;
    return resolveGeminiAuthTypeLabel(
      geminiChannelShort.value,
      geminiChannel.value,
      geminiUserLevel.value,
    );
  });

  const geminiTierClass = computed(() => {
    return resolveGeminiTierClass(
      geminiChannelShort.value,
      geminiUserLevel.value,
    );
  });

  const geminiQuotaPolicyChannel = computed(() => {
    return resolveGeminiQuotaPolicyChannel(geminiChannel.value, t);
  });

  const geminiQuotaPolicyLimits = computed(() => {
    return resolveGeminiQuotaPolicyLimits(
      geminiTier.value,
      geminiChannel.value,
      geminiUserLevel.value,
      t,
    );
  });

  const geminiQuotaPolicyDocsUrl = computed(() => {
    return resolveGeminiQuotaPolicyDocsUrl(geminiChannel.value);
  });

  const geminiUsesSharedDaily = computed(() => {
    if (getRuntimePlatform(account.value) !== "gemini") return false;
    return (
      !!usageInfo.value?.gemini_shared_daily ||
      !!usageInfo.value?.gemini_shared_minute ||
      geminiOAuthType.value === "google_one" ||
      isGeminiCodeAssist.value
    );
  });

  const isProtocolGatewayGeminiAccount = computed(() => {
    return (
      account.value.platform === "protocol_gateway" &&
      getRuntimePlatform(account.value) === "gemini"
    );
  });

  const protocolGatewayBadgeLabel = computed(() => {
    if (!isProtocolGatewayGeminiAccount.value) return null;
    const protocolLabel =
      resolveGatewayProtocolLabel(
        resolveAccountGatewayProtocol(account.value),
      ) || "混合";
    return t("admin.accounts.protocolGateway.usageWindow.badge", {
      protocol: protocolLabel,
    });
  });

  const protocolGatewayBadgeClass = computed(() => {
    if (!isProtocolGatewayGeminiAccount.value) return "";
    return "bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-200";
  });

  const geminiRows = computed(() => {
    if (getRuntimePlatform(account.value) !== "gemini" || !usageInfo.value)
      return [];

    if (geminiUsesSharedDaily.value) {
      return buildRows(
        buildProgressRow(
          "gemini-shared-daily",
          "1d",
          usageInfo.value.gemini_shared_daily,
          "indigo",
        ),
      );
    }

    return buildRows(
      buildProgressRow(
        "gemini-pro-daily",
        "pro",
        usageInfo.value.gemini_pro_daily,
        "indigo",
      ),
      buildProgressRow(
        "gemini-flash-daily",
        "flash",
        usageInfo.value.gemini_flash_daily,
        "emerald",
      ),
    );
  });

  return {
    geminiAuthTypeLabel,
    geminiTierClass,
    geminiQuotaPolicyChannel,
    geminiQuotaPolicyLimits,
    geminiQuotaPolicyDocsUrl,
    isProtocolGatewayGeminiAccount,
    protocolGatewayBadgeLabel,
    protocolGatewayBadgeClass,
    geminiRows,
  };
}
