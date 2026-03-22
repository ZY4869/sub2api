import type { GroupPlatform } from "@/types";

export type CcsClientType = "claude" | "gemini";
type CcsProviderApp = "claude" | "codex" | "gemini";

interface CcsImportTarget {
  app: CcsProviderApp;
  endpoint: string;
}

interface CcsProviderImportOptions {
  apiKey: string;
  baseUrl: string;
  clientType: CcsClientType;
  platform: GroupPlatform;
  providerName: string;
}

const CCS_USAGE_SCRIPT = `({
  request: {
    url: "{{baseUrl}}/v1/usage",
    method: "GET",
    headers: { "Authorization": "Bearer {{apiKey}}" }
  },
  extractor: function(response) {
    const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
    const unit = response?.unit ?? response?.quota?.unit ?? "USD";
    return {
      isValid: response?.is_active ?? response?.isValid ?? true,
      remaining,
      unit
    };
  }
})`;

const CLAUDE_DEFAULT_REASONING_MODEL = {
  anthropic: "claude-opus-4-6",
  kiro: "claude-opus-4-6",
  antigravity: "claude-opus-4-6-thinking",
} as const;

function encodeBase64Utf8(value: string): string {
  const bytes = new TextEncoder().encode(value);

  if (typeof Buffer !== "undefined") {
    return Buffer.from(bytes).toString("base64");
  }

  let binary = "";
  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary);
}

function resolveCcsImportTarget(
  platform: GroupPlatform,
  clientType: CcsClientType,
  baseUrl: string,
): CcsImportTarget {
  if (platform === "antigravity") {
    return {
      app: clientType === "gemini" ? "gemini" : "claude",
      endpoint: `${baseUrl}/antigravity`,
    };
  }

  switch (platform) {
    case "copilot":
      return { app: "codex", endpoint: baseUrl };
    case "kiro":
      return { app: "claude", endpoint: baseUrl };
    case "openai":
      return { app: "codex", endpoint: baseUrl };
    case "gemini":
      return { app: "gemini", endpoint: baseUrl };
    default:
      return { app: "claude", endpoint: baseUrl };
  }
}

function buildClaudeImportConfig(
  endpoint: string,
  apiKey: string,
  platform: GroupPlatform,
): string {
  const reasoningModel =
    platform === "antigravity"
      ? CLAUDE_DEFAULT_REASONING_MODEL.antigravity
      : platform === "kiro"
        ? CLAUDE_DEFAULT_REASONING_MODEL.kiro
      : CLAUDE_DEFAULT_REASONING_MODEL.anthropic;

  return JSON.stringify({
    env: {
      ANTHROPIC_AUTH_TOKEN: apiKey,
      ANTHROPIC_BASE_URL: endpoint,
      ANTHROPIC_REASONING_MODEL: reasoningModel,
    },
  });
}

export function buildCcsProviderImportLink({
  apiKey,
  baseUrl,
  clientType,
  platform,
  providerName,
}: CcsProviderImportOptions): string {
  const { app, endpoint } = resolveCcsImportTarget(platform, clientType, baseUrl);
  const params = new URLSearchParams({
    resource: "provider",
    app,
    name: providerName,
    homepage: baseUrl,
    endpoint,
    apiKey,
    configFormat: "json",
    usageEnabled: "true",
    usageScript: encodeBase64Utf8(CCS_USAGE_SCRIPT),
    usageAutoInterval: "30",
  });

  if (app === "claude") {
    params.set("config", encodeBase64Utf8(buildClaudeImportConfig(endpoint, apiKey, platform)));
  }

  return `ccswitch://v1/import?${params.toString()}`;
}
