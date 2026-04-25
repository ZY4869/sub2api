import { describe, expect, it, vi } from "vitest";

vi.mock("@/api/admin/accounts", () => ({
  getAntigravityDefaultModelMapping: vi.fn(),
}));

vi.mock("@/api/meta", () => ({
  metaAPI: {
    getModelRegistry: vi.fn().mockResolvedValue({
      notModified: true,
      etag: null,
      data: null,
    }),
  },
}));

import {
  buildModelMappingObject,
  getModelsByPlatform,
  getPresetMappingsByPlatform,
} from "../useModelWhitelist";

describe("useModelWhitelist", () => {
  it("keeps deprecated anthropic runtime ids selectable when requesting test exposure", () => {
    const models = getModelsByPlatform("anthropic", "whitelist", "test");

    expect(models).toContain("claude-opus-4.1");
    expect(models).toContain("claude-opus-4-5-20251101");
    expect(models).toContain("claude-sonnet-4-5-20250929");
    expect(models).toContain("claude-haiku-4-5-20251001");
  });

  it("keeps antigravity thinking ids selectable without legacy blocklists", () => {
    const models = getModelsByPlatform("antigravity", "test");

    expect(models).toContain("claude-opus-4-5-thinking");
    expect(models).toContain("claude-sonnet-4-5-thinking");
    expect(models).toContain("claude-opus-4-6-thinking");
  });

  it("openai models include GPT-5.4 mini/nano and GPT-5.4 official snapshots", () => {
    const models = getModelsByPlatform("openai");

    expect(models).toContain("gpt-5.4");
    expect(models).toContain("gpt-5.4-2026-03-05");
    expect(models).toContain("gpt-5.4-mini");
    expect(models).toContain("gpt-5.4-nano");
    expect(models).toContain("gpt-5.4-pro");
    expect(models).toContain("gpt-5.4-pro-2026-03-05");
  });

  it("gemini models include prioritized native image models", () => {
    const models = getModelsByPlatform("gemini");

    expect(models).toContain("gemini-2.5-flash-image");
    expect(models).toContain("gemini-3.1-flash-image");
    expect(models.indexOf("gemini-3.1-flash-image")).toBeLessThan(
      models.indexOf("gemini-2.0-flash"),
    );
    expect(models.indexOf("gemini-2.5-flash-image")).toBeLessThan(
      models.indexOf("gemini-2.5-flash"),
    );
  });

  it("antigravity models include prioritized image compatibility entries", () => {
    const models = getModelsByPlatform("antigravity");

    expect(models).toContain("gemini-2.5-flash-image");
    expect(models).toContain("gemini-3.1-flash-image");
    expect(models).toContain("gemini-3-pro-image");
    expect(models.indexOf("gemini-3.1-flash-image")).toBeLessThan(
      models.indexOf("gemini-2.5-flash"),
    );
    expect(models.indexOf("gemini-2.5-flash-image")).toBeLessThan(
      models.indexOf("gemini-2.5-flash-lite"),
    );
  });

  it("use_key exposure stays curated per platform", () => {
    const openAIModels = getModelsByPlatform("openai", "use_key");
    const geminiModels = getModelsByPlatform("gemini", "use_key");

    expect(openAIModels).toContain("gpt-image-2");
    expect(openAIModels).toContain("gpt-5.4-mini");
    expect(openAIModels).toContain("gpt-5.4-nano");
    expect(openAIModels).not.toContain("gpt-5.4");
    expect(geminiModels).toContain("gemini-2.0-flash");
    expect(geminiModels).toContain("gemini-2.5-flash");
    expect(geminiModels).not.toContain("gemini-3.1-flash-image");
  });

  it("test exposure includes runtime test models without leaking use_key-only curation", () => {
    const openAIModels = getModelsByPlatform("openai", "test");

    expect(openAIModels).toContain("gpt-5.4");
    expect(openAIModels).toContain("gpt-5.4-pro");
    expect(openAIModels).not.toContain("gpt-5-codex");
  });

  it("keeps antigravity presets available without hard-filtering 4.6 ids", () => {
    const presets = getPresetMappingsByPlatform("antigravity");

    expect(presets.some((preset) => preset.to === "claude-sonnet-4.5")).toBe(
      true,
    );
    expect(presets.some((preset) => preset.to === "claude-opus-4.1")).toBe(
      true,
    );
    expect(
      presets.some(
        (preset) =>
          preset.from === "gemini-2.5-flash-image" &&
          preset.to === "gemini-2.5-flash-image",
      ),
    ).toBe(true);
    expect(
      presets.some(
        (preset) =>
          preset.from === "gemini-3.1-flash-image" &&
          preset.to === "gemini-3.1-flash-image",
      ),
    ).toBe(true);
    expect(
      presets.some(
        (preset) =>
          preset.from === "gemini-3-pro-image" &&
          preset.to === "gemini-3.1-flash-image",
      ),
    ).toBe(true);
  });

  it("ignores wildcard entries in whitelist mode", () => {
    const mapping = buildModelMappingObject(
      "whitelist",
      ["claude-*", "gemini-3.1-flash-image"],
      [],
    );

    expect(mapping).toEqual({
      "gemini-3.1-flash-image": "gemini-3.1-flash-image",
    });
  });

  it("keeps GPT-5.4 official snapshot as exact whitelist mapping", () => {
    const mapping = buildModelMappingObject(
      "whitelist",
      ["gpt-5.4-2026-03-05"],
      [],
    );

    expect(mapping).toEqual({
      "gpt-5.4-2026-03-05": "gpt-5.4-2026-03-05",
    });
  });

  it("keeps GPT-5.4-pro official snapshot as exact whitelist mapping", () => {
    const mapping = buildModelMappingObject(
      "whitelist",
      ["gpt-5.4-pro-2026-03-05"],
      [],
    );

    expect(mapping).toEqual({
      "gpt-5.4-pro-2026-03-05": "gpt-5.4-pro-2026-03-05",
    });
  });
});
