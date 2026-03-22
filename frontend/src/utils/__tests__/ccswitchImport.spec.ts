import { describe, expect, it } from "vitest";

import { buildCcsProviderImportLink } from "../ccswitchImport";

function decodeBase64(value: string): string {
  return Buffer.from(value, "base64").toString("utf-8");
}

function parseImportLink(url: string): URLSearchParams {
  const query = url.split("?")[1] ?? "";
  return new URLSearchParams(query);
}

describe("ccswitchImport", () => {
  it("injects a default Claude reasoning model for anthropic imports", () => {
    const params = parseImportLink(
      buildCcsProviderImportLink({
        apiKey: "sk-test",
        baseUrl: "https://relay.example.com",
        clientType: "claude",
        platform: "anthropic",
        providerName: "Sub2API",
      }),
    );

    expect(params.get("app")).toBe("claude");
    expect(params.get("endpoint")).toBe("https://relay.example.com");

    const config = JSON.parse(decodeBase64(params.get("config") || ""));
    expect(config.env.ANTHROPIC_BASE_URL).toBe("https://relay.example.com");
    expect(config.env.ANTHROPIC_AUTH_TOKEN).toBe("sk-test");
    expect(config.env.ANTHROPIC_REASONING_MODEL).toBe("claude-opus-4-6");
  });

  it("uses the explicit thinking model for antigravity Claude imports", () => {
    const params = parseImportLink(
      buildCcsProviderImportLink({
        apiKey: "sk-test",
        baseUrl: "https://relay.example.com",
        clientType: "claude",
        platform: "antigravity",
        providerName: "Sub2API",
      }),
    );

    expect(params.get("app")).toBe("claude");
    expect(params.get("endpoint")).toBe("https://relay.example.com/antigravity");

    const config = JSON.parse(decodeBase64(params.get("config") || ""));
    expect(config.env.ANTHROPIC_BASE_URL).toBe(
      "https://relay.example.com/antigravity",
    );
    expect(config.env.ANTHROPIC_REASONING_MODEL).toBe(
      "claude-opus-4-6-thinking",
    );
  });

  it("keeps antigravity gemini imports on the gemini app without Claude config", () => {
    const params = parseImportLink(
      buildCcsProviderImportLink({
        apiKey: "sk-test",
        baseUrl: "https://relay.example.com",
        clientType: "gemini",
        platform: "antigravity",
        providerName: "Sub2API",
      }),
    );

    expect(params.get("app")).toBe("gemini");
    expect(params.get("endpoint")).toBe("https://relay.example.com/antigravity");
    expect(params.has("config")).toBe(false);
  });

  it("preserves codex imports for openai groups", () => {
    const params = parseImportLink(
      buildCcsProviderImportLink({
        apiKey: "sk-test",
        baseUrl: "https://relay.example.com",
        clientType: "claude",
        platform: "openai",
        providerName: "Sub2API",
      }),
    );

    expect(params.get("app")).toBe("codex");
    expect(params.get("endpoint")).toBe("https://relay.example.com");
    expect(params.has("config")).toBe(false);
  });

  it("maps kiro imports onto the claude app and keeps Claude default thinking", () => {
    const params = parseImportLink(
      buildCcsProviderImportLink({
        apiKey: "sk-test",
        baseUrl: "https://relay.example.com",
        clientType: "claude",
        platform: "kiro",
        providerName: "Sub2API",
      }),
    );

    expect(params.get("app")).toBe("claude");
    expect(params.get("endpoint")).toBe("https://relay.example.com");

    const config = JSON.parse(decodeBase64(params.get("config") || ""));
    expect(config.env.ANTHROPIC_REASONING_MODEL).toBe("claude-opus-4-6");
  });

  it("maps copilot imports onto the codex app", () => {
    const params = parseImportLink(
      buildCcsProviderImportLink({
        apiKey: "sk-test",
        baseUrl: "https://relay.example.com",
        clientType: "claude",
        platform: "copilot",
        providerName: "Sub2API",
      }),
    );

    expect(params.get("app")).toBe("codex");
    expect(params.get("endpoint")).toBe("https://relay.example.com");
    expect(params.has("config")).toBe(false);
  });
});
