import { getAntigravityDefaultModelMapping } from "@/api/admin/accounts";
import {
  ensureModelRegistryFresh,
  getModelRegistrySnapshot,
} from "@/stores/modelRegistry";
import type {
  ModelRegistryEntry,
  ModelRegistryPreset,
} from "@/generated/modelRegistry";

export interface ModelOption {
  value: string;
  label: string;
}

export interface ModelPresetMapping {
  label: string;
  from: string;
  to: string;
  color: string;
  order?: number;
}

export interface ModelCapabilityDefinition {
  name: string;
  limit: {
    context: number;
    output: number;
  };
  modalities?: {
    input: string[];
    output: string[];
  };
  options?: Record<string, any>;
  variants?: Record<string, Record<string, never>>;
}

const CAPABILITY_OVERRIDES: Record<string, ModelCapabilityDefinition> = {
  "gpt-5.4": {
    name: "GPT-5.4",
    limit: { context: 1050000, output: 128000 },
    options: { store: false },
    variants: { low: {}, medium: {}, high: {}, xhigh: {} },
  },
  "gpt-5.4-mini": {
    name: "GPT-5.4 Mini",
    limit: { context: 400000, output: 128000 },
    options: { store: false },
    variants: { low: {}, medium: {}, high: {} },
  },
  "gpt-5.4-nano": {
    name: "GPT-5.4 Nano",
    limit: { context: 400000, output: 128000 },
    options: { store: false },
    variants: { low: {}, medium: {}, high: {} },
  },
  "gpt-5.4-pro": {
    name: "GPT-5.4 Pro",
    limit: { context: 1050000, output: 128000 },
    options: { store: false },
    variants: { medium: {}, high: {}, xhigh: {} },
  },
  "gpt-5.4-pro-2026-03-05": {
    name: "GPT-5.4 Pro",
    limit: { context: 1050000, output: 128000 },
    options: { store: false },
    variants: { medium: {}, high: {}, xhigh: {} },
  },
  "gpt-5-codex": {
    name: "GPT-5 Codex",
    limit: { context: 400000, output: 128000 },
    options: { store: false },
    variants: { low: {}, medium: {}, high: {} },
  },
  "gpt-5.1-codex": {
    name: "GPT-5.1 Codex",
    limit: { context: 400000, output: 128000 },
    options: { store: false },
    variants: { low: {}, medium: {}, high: {} },
  },
  "gpt-5.1-codex-max": {
    name: "GPT-5.1 Codex Max",
    limit: { context: 400000, output: 128000 },
    options: { store: false },
    variants: { low: {}, medium: {}, high: {} },
  },
  "gpt-5.1-codex-mini": {
    name: "GPT-5.1 Codex Mini",
    limit: { context: 400000, output: 128000 },
    options: { store: false },
    variants: { low: {}, medium: {}, high: {} },
  },
  "gpt-5.2": {
    name: "GPT-5.2",
    limit: { context: 400000, output: 128000 },
    options: { store: false },
    variants: { low: {}, medium: {}, high: {}, xhigh: {} },
  },
  "gpt-5.3-codex": {
    name: "GPT-5.3 Codex",
    limit: { context: 400000, output: 128000 },
    options: { store: false },
    variants: { low: {}, medium: {}, high: {}, xhigh: {} },
  },
  "gpt-5.3-codex-spark": {
    name: "GPT-5.3 Codex Spark",
    limit: { context: 128000, output: 32000 },
    options: { store: false },
    variants: { low: {}, medium: {}, high: {}, xhigh: {} },
  },
  "gpt-5.2-codex": {
    name: "GPT-5.2 Codex",
    limit: { context: 400000, output: 128000 },
    options: { store: false },
    variants: { low: {}, medium: {}, high: {}, xhigh: {} },
  },
  "gemini-2.0-flash": {
    name: "Gemini 2.0 Flash",
    limit: { context: 1048576, output: 65536 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
  },
  "gemini-2.5-flash": {
    name: "Gemini 2.5 Flash",
    limit: { context: 1048576, output: 65536 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 24576, type: "disable" } },
  },
  "gemini-2.5-flash-lite": {
    name: "Gemini 2.5 Flash Lite",
    limit: { context: 1048576, output: 65536 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "gemini-2.5-flash-thinking": {
    name: "Gemini 2.5 Flash (Thinking)",
    limit: { context: 1048576, output: 65536 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "gemini-2.5-pro": {
    name: "Gemini 2.5 Pro",
    limit: { context: 2097152, output: 65536 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 32768, type: "enabled" } },
  },
  "gemini-3-flash-preview": {
    name: "Gemini 3 Flash Preview",
    limit: { context: 1048576, output: 65536 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "gemini-3-pro-preview": {
    name: "Gemini 3 Pro Preview",
    limit: { context: 1048576, output: 65536 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "gemini-3.1-pro-low": {
    name: "Gemini 3.1 Pro Low",
    limit: { context: 1048576, output: 65536 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "gemini-3.1-pro-high": {
    name: "Gemini 3.1 Pro High",
    limit: { context: 1048576, output: 65536 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "gemini-2.5-flash-image": {
    name: "Gemini 2.5 Flash Image",
    limit: { context: 1048576, output: 65536 },
    modalities: { input: ["text", "image"], output: ["image"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "gemini-3.1-flash-image": {
    name: "Gemini 3.1 Flash Image",
    limit: { context: 1048576, output: 65536 },
    modalities: { input: ["text", "image"], output: ["image"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "claude-opus-4.1": {
    name: "Claude Opus 4.1",
    limit: { context: 200000, output: 128000 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "claude-opus-4-6": {
    name: "Claude Opus 4.6",
    limit: { context: 1000000, output: 128000 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "claude-opus-4-7": {
    name: "Claude Opus 4.7",
    limit: { context: 1000000, output: 128000 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "claude-sonnet-4.5": {
    name: "Claude Sonnet 4.5",
    limit: { context: 200000, output: 64000 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "claude-sonnet-4-6": {
    name: "Claude Sonnet 4.6",
    limit: { context: 200000, output: 64000 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  },
  "claude-haiku-4.5": {
    name: "Claude Haiku 4.5",
    limit: { context: 200000, output: 64000 },
    modalities: { input: ["text", "image", "pdf"], output: ["text"] },
  },
};

export const commonErrorCodes = [
  { value: 401, label: "Unauthorized" },
  { value: 403, label: "Forbidden" },
  { value: 429, label: "Rate Limit" },
  { value: 500, label: "Server Error" },
  { value: 502, label: "Bad Gateway" },
  { value: 503, label: "Unavailable" },
  { value: 529, label: "Overloaded" },
];

let antigravityDefaultMappingsCache: { from: string; to: string }[] | null =
  null;

function normalizePlatform(platform: string): string {
  const value = platform.trim().toLowerCase();
  return value === "claude" ? "anthropic" : value;
}

function normalizeExposureTargets(exposures: string[]): string[] {
  const targets = Array.from(
    new Set(
      exposures
        .map((exposure) => exposure.trim().toLowerCase())
        .filter(Boolean),
    ),
  );
  return targets.length > 0 ? targets : ["whitelist"];
}

function normalizeEntryIds(
  platform: string,
  entries: ModelRegistryEntry[],
): string[] {
  void platform;
  return entries.map((entry) => entry.id);
}

function matchesExposure(
  entry: ModelRegistryEntry,
  ...exposures: string[]
): boolean {
  const targets = normalizeExposureTargets(exposures);
  return targets.some((target) => entry.exposed_in.includes(target));
}

function sortEntries(entries: ModelRegistryEntry[]): ModelRegistryEntry[] {
  return [...entries].sort((left, right) => {
    if (left.ui_priority !== right.ui_priority) {
      return left.ui_priority - right.ui_priority;
    }
    return left.id.localeCompare(right.id);
  });
}

function getRegistryEntriesByPlatform(
  platform: string,
  ...exposures: string[]
): ModelRegistryEntry[] {
  void ensureModelRegistryFresh();
  const normalizedPlatform = normalizePlatform(platform);
  const targets = normalizeExposureTargets(exposures);
  const snapshot = getModelRegistrySnapshot();
  return sortEntries(
    snapshot.models.filter((entry) => {
      if (!entry.platforms.includes(normalizedPlatform)) {
        return false;
      }
      return matchesExposure(entry, ...targets);
    }),
  );
}

function getRegistryEntry(modelId: string): ModelRegistryEntry | undefined {
  const normalizedId = modelId.trim();
  if (!normalizedId) {
    return undefined;
  }
  const snapshot = getModelRegistrySnapshot();
  return snapshot.models.find(
    (entry) =>
      entry.id === normalizedId ||
      entry.aliases.includes(normalizedId) ||
      entry.protocol_ids.includes(normalizedId),
  );
}

function deriveModalities(
  entry: ModelRegistryEntry | undefined,
): ModelCapabilityDefinition["modalities"] {
  if (!entry) {
    return { input: ["text"], output: ["text"] };
  }
  const hasImageOutput =
    entry.capabilities.includes("image") || entry.modalities.includes("image");
  const input =
    entry.modalities.length > 0 ? [...entry.modalities] : ["text", "image"];
  if (!input.includes("pdf")) {
    input.push("pdf");
  }
  return {
    input,
    output: hasImageOutput ? ["image"] : ["text"],
  };
}

function deriveFallbackCapability(
  platform: string,
  modelId: string,
  entry?: ModelRegistryEntry,
): ModelCapabilityDefinition {
  const normalizedPlatform = normalizePlatform(platform);
  const displayName = entry?.display_name || modelId;
  if (normalizedPlatform === "openai" || normalizedPlatform === "copilot") {
    return {
      name: displayName,
      limit: { context: 400000, output: 128000 },
      options: { store: false },
      variants: { low: {}, medium: {}, high: {} },
    };
  }
  if (normalizedPlatform === "gemini" || normalizedPlatform === "antigravity") {
    return {
      name: displayName,
      limit: { context: 1048576, output: 65536 },
      modalities: deriveModalities(entry),
      options: { thinking: { budgetTokens: 24576, type: "enabled" } },
    };
  }
  return {
    name: displayName,
    limit: { context: 200000, output: 64000 },
    modalities: deriveModalities(entry),
    options: { thinking: { budgetTokens: 24576, type: "enabled" } },
  };
}

export function getAllModelOptions(...exposures: string[]): ModelOption[] {
  const snapshot = getModelRegistrySnapshot();
  void ensureModelRegistryFresh();
  const targets = normalizeExposureTargets(exposures);
  const ids = sortEntries(snapshot.models)
    .filter((entry) => matchesExposure(entry, ...targets))
    .map((entry) => entry.id);

  return Array.from(new Set(ids)).map((value) => ({ value, label: value }));
}

export async function fetchAntigravityDefaultMappings(): Promise<
  { from: string; to: string }[]
> {
  if (antigravityDefaultMappingsCache !== null) {
    return antigravityDefaultMappingsCache;
  }
  try {
    const mapping = await getAntigravityDefaultModelMapping();
    antigravityDefaultMappingsCache = Object.entries(mapping).map(
      ([from, to]) => ({ from, to }),
    );
  } catch (error) {
    console.warn(
      "[fetchAntigravityDefaultMappings] API failed, using empty fallback",
      error,
    );
    antigravityDefaultMappingsCache = [];
  }
  return antigravityDefaultMappingsCache;
}

export function getModelsByPlatform(
  platform: string,
  ...exposures: string[]
): string[] {
  return normalizeEntryIds(
    platform,
    getRegistryEntriesByPlatform(platform, ...exposures),
  );
}

export function getPresetMappingsByPlatform(
  platform: string,
): ModelRegistryPreset[] {
  void ensureModelRegistryFresh();
  const normalizedPlatform = normalizePlatform(platform);
  const snapshot = getModelRegistrySnapshot();
  return snapshot.presets
    .filter(
      (preset) => normalizePlatform(preset.platform) === normalizedPlatform,
    )
    .sort((left, right) => (left.order || 0) - (right.order || 0));
}

export function sortModelsForTest<T extends { id: string }>(models: T[]): T[] {
  void ensureModelRegistryFresh();
  const snapshot = getModelRegistrySnapshot();
  const priorityMap = new Map(
    snapshot.models.map((entry) => [entry.id, entry.ui_priority]),
  );
  return [...models].sort((left, right) => {
    const leftPriority = priorityMap.get(left.id) ?? Number.MAX_SAFE_INTEGER;
    const rightPriority = priorityMap.get(right.id) ?? Number.MAX_SAFE_INTEGER;
    if (leftPriority !== rightPriority) {
      return leftPriority - rightPriority;
    }
    return left.id.localeCompare(right.id);
  });
}

export function getModelCapabilities(
  platform: string,
  modelId: string,
  _channel?: string,
): ModelCapabilityDefinition {
  void ensureModelRegistryFresh();
  const normalizedId = modelId.trim();
  const entry = getRegistryEntry(normalizedId);
  const override = CAPABILITY_OVERRIDES[normalizedId];
  if (override) {
    return {
      ...override,
      name: override.name || entry?.display_name || normalizedId,
      modalities: override.modalities || deriveModalities(entry),
    };
  }
  return deriveFallbackCapability(platform, normalizedId, entry);
}

export function isValidWildcardPattern(pattern: string): boolean {
  const starIndex = pattern.indexOf("*");
  if (starIndex === -1) return true;
  return (
    starIndex === pattern.length - 1 && pattern.lastIndexOf("*") === starIndex
  );
}

export function buildModelMappingObject(
  mode: "whitelist" | "mapping",
  allowedModels: string[],
  modelMappings: { from: string; to: string }[],
): Record<string, string> | null {
  const mapping: Record<string, string> = {};

  if (mode === "whitelist") {
    for (const model of allowedModels) {
      if (!model.includes("*")) {
        mapping[model] = model;
      }
    }
  } else {
    for (const item of modelMappings) {
      const from = item.from.trim();
      const to = item.to.trim();
      if (!from || !to) continue;
      if (!isValidWildcardPattern(from)) {
        console.warn(`[buildModelMappingObject] 跳过无效模型映射来源: ${from}`);
        continue;
      }
      if (to.includes("*")) {
        console.warn(
          `[buildModelMappingObject] 跳过无效模型映射目标: ${from} -> ${to}`,
        );
        continue;
      }
      mapping[from] = to;
    }
  }

  return Object.keys(mapping).length > 0 ? mapping : null;
}
