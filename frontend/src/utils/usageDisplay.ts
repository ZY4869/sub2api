import { i18n } from "@/i18n";
import type { UsageLog } from "@/types";

type UsageEndpointLineKey = "inbound" | "upstream";
type UsageMillionContextLineKey = "requested" | "effective" | "source" | "betaToken";

export interface UsageEndpointDisplayLine {
  key: UsageEndpointLineKey;
  labelKey: `usage.${UsageEndpointLineKey}`;
  raw: string;
  display: string;
}

export interface UsageMillionContextDisplayLine {
  key: UsageMillionContextLineKey;
  labelKey:
    | "usage.millionContextRequested"
    | "usage.millionContextEffective"
    | "usage.millionContextSource"
    | "usage.millionContextBetaToken";
  raw: string;
  display: string;
}

export interface UsageMillionContextExportFields {
  requested: string;
  effective: string;
  source: string;
  betaToken: string;
}

function translate(key: string, fallback: string): string {
  if (typeof i18n.global.te === "function" && !i18n.global.te(key)) {
    return fallback;
  }
  const message = i18n.global.t(key);
  return typeof message === "string" && message !== key ? message : fallback;
}

function formatTruthLikeDisplay(
  value: boolean | number | string | null | undefined,
): string {
  if (value === true) {
    return translate("common.yes", "Yes");
  }
  if (value === false) {
    return translate("common.no", "No");
  }
  if (typeof value === "number") {
    if (!Number.isFinite(value)) {
      return "-";
    }
    if (value === 1) {
      return translate("common.yes", "Yes");
    }
    if (value === 0) {
      return translate("common.no", "No");
    }
    return String(value);
  }
  const raw = value?.toString().trim();
  if (!raw) return "-";
  const normalized = raw.toLowerCase();
  if (normalized === "true" || normalized === "1") {
    return translate("common.yes", "Yes");
  }
  if (normalized === "false" || normalized === "0") {
    return translate("common.no", "No");
  }
  return raw;
}

function toUsageRawDisplay(value: unknown): string {
  if (value === null || typeof value === "undefined") {
    return "-";
  }
  if (typeof value === "string") {
    const raw = value.trim();
    return raw || "-";
  }
  if (typeof value === "number") {
    return Number.isFinite(value) ? String(value) : "-";
  }
  if (typeof value === "boolean") {
    return String(value);
  }
  return String(value);
}

function hasUsageValue(value: unknown): boolean {
  if (value === null || typeof value === "undefined") {
    return false;
  }
  if (typeof value === "string") {
    return value.trim().length > 0;
  }
  if (typeof value === "number") {
    return Number.isFinite(value);
  }
  return true;
}

export function formatUsageEndpointPath(
  path: string | null | undefined,
): string {
  const raw = path?.trim();
  if (!raw) return "-";

  if (raw.startsWith("/v1/messages")) {
    return translate("usage.endpointNames.messages", "Messages API");
  }
  if (raw.startsWith("/v1/chat/completions")) {
    return translate("usage.endpointNames.chatCompletions", "Chat Completions");
  }
  if (raw.startsWith("/v1/responses")) {
    return translate("usage.endpointNames.responses", "Responses API");
  }
  if (raw.startsWith("/v1beta/models")) {
    return translate("usage.endpointNames.geminiModels", "Gemini Models API");
  }

  return raw;
}

export function formatUsageEndpointDisplay(
  log: Pick<UsageLog, "inbound_endpoint" | "upstream_endpoint">,
): UsageEndpointDisplayLine[] {
  const inbound = log.inbound_endpoint?.trim();
  const upstream = log.upstream_endpoint?.trim();
  const lines: UsageEndpointDisplayLine[] = [];

  if (inbound) {
    lines.push({
      key: "inbound",
      labelKey: "usage.inbound",
      raw: inbound,
      display: formatUsageEndpointPath(inbound),
    });
  }

  if (upstream) {
    lines.push({
      key: "upstream",
      labelKey: "usage.upstream",
      raw: upstream,
      display: formatUsageEndpointPath(upstream),
    });
  }

  return lines;
}

export function formatUsageMillionContextDisplay(
  log: Pick<
    UsageLog,
    | "million_context_requested"
    | "million_context_effective"
    | "million_context_source"
    | "million_context_beta_token"
  >,
): UsageMillionContextDisplayLine[] {
  const lines: UsageMillionContextDisplayLine[] = [];

  if (hasUsageValue(log.million_context_requested)) {
    lines.push({
      key: "requested",
      labelKey: "usage.millionContextRequested",
      raw: toUsageRawDisplay(log.million_context_requested),
      display: formatTruthLikeDisplay(log.million_context_requested),
    });
  }

  if (hasUsageValue(log.million_context_effective)) {
    lines.push({
      key: "effective",
      labelKey: "usage.millionContextEffective",
      raw: toUsageRawDisplay(log.million_context_effective),
      display: formatTruthLikeDisplay(log.million_context_effective),
    });
  }

  if (hasUsageValue(log.million_context_source)) {
    lines.push({
      key: "source",
      labelKey: "usage.millionContextSource",
      raw: toUsageRawDisplay(log.million_context_source),
      display: toUsageRawDisplay(log.million_context_source),
    });
  }

  if (hasUsageValue(log.million_context_beta_token)) {
    lines.push({
      key: "betaToken",
      labelKey: "usage.millionContextBetaToken",
      raw: toUsageRawDisplay(log.million_context_beta_token),
      display: toUsageRawDisplay(log.million_context_beta_token),
    });
  }

  return lines;
}

export function formatUsageMillionContextExportFields(
  log: Pick<
    UsageLog,
    | "million_context_requested"
    | "million_context_effective"
    | "million_context_source"
    | "million_context_beta_token"
  >,
): UsageMillionContextExportFields {
  return {
    requested: toUsageRawDisplay(log.million_context_requested),
    effective: toUsageRawDisplay(log.million_context_effective),
    source: toUsageRawDisplay(log.million_context_source),
    betaToken: toUsageRawDisplay(log.million_context_beta_token),
  };
}

export function formatUsageUserAgentDisplay(
  userAgent: string | null | undefined,
): string {
  const raw = userAgent?.trim();
  if (!raw) return "-";

  const normalized = raw.toLowerCase();

  if (normalized.includes("claude code")) {
    return translate("usage.userAgentNames.claudeCode", "Claude Code");
  }
  if (normalized.includes("codex_cli") || normalized.includes("codex cli")) {
    return translate("usage.userAgentNames.codexCli", "Codex CLI");
  }
  if (
    normalized.includes("gemini-cli") ||
    normalized.includes("google genai") ||
    normalized.includes("genai-sdk")
  ) {
    return translate("usage.userAgentNames.geminiCli", "Gemini CLI");
  }
  if (normalized.includes("openai/") || normalized.includes("openai sdk")) {
    return translate("usage.userAgentNames.openaiSdk", "OpenAI SDK");
  }
  if (
    normalized.includes("anthropic-sdk") ||
    normalized.includes("@anthropic-ai") ||
    normalized.includes("anthropic/")
  ) {
    return translate("usage.userAgentNames.anthropicSdk", "Anthropic SDK");
  }
  if (normalized.startsWith("curl/")) {
    return translate("usage.userAgentNames.curl", "curl");
  }
  if (normalized.includes("postman")) {
    return translate("usage.userAgentNames.postman", "Postman");
  }
  if (
    normalized.includes("mozilla/") ||
    normalized.includes("chrome/") ||
    normalized.includes("safari/") ||
    normalized.includes("firefox/") ||
    normalized.includes("edg/")
  ) {
    return translate("usage.userAgentNames.browser", "Browser");
  }

  return raw;
}
