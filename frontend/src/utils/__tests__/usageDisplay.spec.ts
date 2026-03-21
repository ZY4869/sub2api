import { describe, expect, it, vi } from "vitest";

const messages: Record<string, string> = {
  "usage.endpointNames.messages": "消息接口",
  "usage.endpointNames.chatCompletions": "聊天补全",
  "usage.endpointNames.responses": "响应接口",
  "usage.endpointNames.geminiModels": "Gemini 模型接口",
  "usage.userAgentNames.claudeCode": "Claude Code",
  "usage.userAgentNames.codexCli": "Codex CLI",
  "usage.userAgentNames.geminiCli": "Gemini CLI",
  "usage.userAgentNames.openaiSdk": "OpenAI SDK",
  "usage.userAgentNames.anthropicSdk": "Anthropic SDK",
  "usage.userAgentNames.browser": "浏览器",
  "usage.userAgentNames.curl": "curl",
  "usage.userAgentNames.postman": "Postman",
};

vi.mock("@/i18n", () => ({
  i18n: {
    global: {
      t: (key: string) => messages[key] ?? key,
    },
  },
}));

import {
  formatUsageEndpointDisplay,
  formatUsageEndpointPath,
  formatUsageUserAgentDisplay,
} from "../usageDisplay";

type EndpointLog = {
  inbound_endpoint?: string | null;
  upstream_endpoint?: string | null;
};

describe("usageDisplay", () => {
  it("formats endpoint lines for inbound and upstream combinations", () => {
    expect(
      formatUsageEndpointDisplay({
        inbound_endpoint: "/v1/messages",
        upstream_endpoint: "/v1/responses",
      } as EndpointLog),
    ).toEqual([
      {
        key: "inbound",
        labelKey: "usage.inbound",
        raw: "/v1/messages",
        display: "消息接口",
      },
      {
        key: "upstream",
        labelKey: "usage.upstream",
        raw: "/v1/responses",
        display: "响应接口",
      },
    ]);

    expect(
      formatUsageEndpointDisplay({
        inbound_endpoint: "/custom/inbound",
        upstream_endpoint: "",
      } as EndpointLog),
    ).toEqual([
      {
        key: "inbound",
        labelKey: "usage.inbound",
        raw: "/custom/inbound",
        display: "/custom/inbound",
      },
    ]);

    expect(
      formatUsageEndpointDisplay({
        inbound_endpoint: " ",
        upstream_endpoint: null,
      } as EndpointLog),
    ).toEqual([]);
  });

  it("maps common endpoint paths to readable labels", () => {
    expect(formatUsageEndpointPath("/v1/chat/completions")).toBe("聊天补全");
    expect(formatUsageEndpointPath("/v1beta/models/gemini-pro:generateContent")).toBe(
      "Gemini 模型接口",
    );
    expect(formatUsageEndpointPath("/internal/raw")).toBe("/internal/raw");
  });

  it("maps common user agents and preserves unknown values", () => {
    expect(formatUsageUserAgentDisplay("Claude Code/1.0")).toBe("Claude Code");
    expect(formatUsageUserAgentDisplay("codex_cli/0.1.0")).toBe("Codex CLI");
    expect(formatUsageUserAgentDisplay("google genai sdk")).toBe("Gemini CLI");
    expect(formatUsageUserAgentDisplay("curl/8.7.1")).toBe("curl");
    expect(formatUsageUserAgentDisplay("Mozilla/5.0")).toBe("浏览器");
    expect(formatUsageUserAgentDisplay("custom-app/2.0")).toBe("custom-app/2.0");
    expect(formatUsageUserAgentDisplay("")).toBe("-");
  });
});
