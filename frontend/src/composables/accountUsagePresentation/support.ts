import { i18n } from "@/i18n";
import type { Account, AccountUsageInfo } from "@/types";
import { resolveEffectiveAccountPlatformFromAccount } from "@/utils/accountProtocolGateway";
import type { LoadUsageOptions } from "./types";

export function getUsageLoadErrorMessage(): string {
  const translated = i18n.global.t("common.error");
  return typeof translated === "string" && translated.trim() !== ""
    ? translated
    : "Error";
}

export function getRuntimePlatform(account: Account): string {
  return resolveEffectiveAccountPlatformFromAccount(account);
}

export function prefersPassiveUsageSnapshot(account: Account): boolean {
  const runtimePlatform = getRuntimePlatform(account);
  return (
    (runtimePlatform === "anthropic" || runtimePlatform === "kiro") &&
    (account.type === "oauth" || account.type === "setup-token")
  );
}

export function supportsActiveAnthropicUsage(account: Account): boolean {
  return (
    getRuntimePlatform(account) === "anthropic" &&
    account.type === "oauth" &&
    account.active_usage_available === true
  );
}

export function supportsActiveOpenAIUsage(account: Account): boolean {
  return (
    getRuntimePlatform(account) === "openai" &&
    account.type === "oauth" &&
    account.active_usage_available === true
  );
}

export function shouldFallbackToActiveAnthropicUsage(
  account: Account,
  source: LoadUsageOptions["source"] | undefined,
  usageInfo: AccountUsageInfo,
): boolean {
  return (
    supportsActiveAnthropicUsage(account) &&
    source === "passive" &&
    !usageInfo.seven_day
  );
}

export function canAccountFetchUsage(account: Account): boolean {
  if (account.platform === "protocol_gateway") {
    return false;
  }
  const runtimePlatform = getRuntimePlatform(account);
  if (runtimePlatform === "anthropic" || runtimePlatform === "kiro") {
    return account.type === "oauth" || account.type === "setup-token";
  }
  if (runtimePlatform === "gemini") {
    return true;
  }
  if (runtimePlatform === "antigravity") {
    return account.type === "oauth";
  }
  if (runtimePlatform === "openai") {
    return account.type === "oauth";
  }
  return false;
}
