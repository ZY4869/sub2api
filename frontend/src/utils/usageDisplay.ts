import { i18n } from "@/i18n";
import type { UsageLog } from "@/types";

type UsageEndpointLineKey = "inbound" | "upstream";

export interface UsageEndpointDisplayLine {
  key: UsageEndpointLineKey;
  labelKey: `usage.${UsageEndpointLineKey}`;
  raw: string;
  display: string;
}

function translate(key: string, fallback: string): string {
  if (typeof i18n.global.te === "function" && !i18n.global.te(key)) {
    return fallback;
  }
  const message = i18n.global.t(key);
  return typeof message === "string" && message !== key ? message : fallback;
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
