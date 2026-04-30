import type {
  AdminModelDebugEndpointKind,
  AdminModelDebugProtocol,
} from "@/api/admin/modelDebug";
import type { PublicModelCatalogItem } from "@/api/meta";
import type { ApiKey, ApiKeyGroup, UserGroupModelOptionGroup } from "@/types";
import { buildPublicModelCatalogDisplayItem } from "@/utils/publicModelCatalog";

export const MODEL_DEBUG_PROTOCOLS: AdminModelDebugProtocol[] = [
  "openai",
  "anthropic",
  "gemini",
];

export const MODEL_DEBUG_ENDPOINTS: Record<
  AdminModelDebugProtocol,
  AdminModelDebugEndpointKind[]
> = {
  openai: ["responses", "chat_completions"],
  anthropic: ["messages"],
  gemini: ["generate_content"],
};

export interface ModelDebugFormState {
  protocol: AdminModelDebugProtocol;
  endpointKind: AdminModelDebugEndpointKind;
  stream: boolean;
  systemPrompt: string;
  userPrompt: string;
  temperature: string;
  maxOutputTokens: string;
  reasoningEffort: string;
}

export interface ModelDebugEditorState extends ModelDebugFormState {
  keyMode: "saved" | "manual";
  apiKeyID: number | null;
  manualAPIKey: string;
  model: string;
  advancedJSON: string;
}

export interface ModelDebugOutputEvent {
  id: string;
  type: string;
  title: string;
  body: string;
  tone: "info" | "success" | "warning" | "error";
}

export interface ModelDebugPromptDefaults {
  systemPrompt?: string;
  userPrompt?: string;
}

export function defaultModelDebugEndpoint(
  protocol: AdminModelDebugProtocol,
): AdminModelDebugEndpointKind {
  return MODEL_DEBUG_ENDPOINTS[protocol][0];
}

export function buildBaseModelDebugRequestBody(
  form: ModelDebugFormState,
  defaults: ModelDebugPromptDefaults = {},
): Record<string, any> {
  switch (form.protocol) {
    case "anthropic":
      return buildAnthropicRequestBody(form, defaults);
    case "gemini":
      return buildGeminiRequestBody(form, defaults);
    default:
      return form.endpointKind === "chat_completions"
        ? buildOpenAIChatRequestBody(form, defaults)
        : buildOpenAIResponsesRequestBody(form, defaults);
  }
}

export function mergeModelDebugRequestBody(
  baseBody: Record<string, any>,
  advancedJSONText: string,
): { body: Record<string, any>; error: string } {
  const trimmed = advancedJSONText.trim();
  if (!trimmed) {
    return { body: baseBody, error: "" };
  }
  try {
    const parsed = JSON.parse(trimmed);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return { body: baseBody, error: "advanced_json_must_be_object" };
    }
    return { body: deepMerge(baseBody, parsed as Record<string, any>), error: "" };
  } catch {
    return { body: baseBody, error: "advanced_json_invalid" };
  }
}

export function filterModelDebugCatalogItems(
  items: PublicModelCatalogItem[],
  keyMode: "saved" | "manual",
  apiKey: ApiKey | null,
  groups: UserGroupModelOptionGroup[],
): PublicModelCatalogItem[] {
  if (keyMode !== "saved" || !apiKey) {
    return sortCatalogItems(items);
  }
  const allowed = collectAllowedModelIDs(apiKey, groups);
  if (allowed.size === 0) {
    return [];
  }
  return sortCatalogItems(items.filter((item) => allowed.has(String(item.model || "").trim())));
}

function sortCatalogItems(items: PublicModelCatalogItem[]): PublicModelCatalogItem[] {
  return [...items].sort((left, right) => {
    const leftDisplay = buildPublicModelCatalogDisplayItem(left);
    const rightDisplay = buildPublicModelCatalogDisplayItem(right);
    return leftDisplay.title.localeCompare(rightDisplay.title);
  });
}

function collectAllowedModelIDs(
  apiKey: ApiKey,
  groups: UserGroupModelOptionGroup[],
): Set<string> {
  const groupMap = new Map(groups.map((group) => [group.group_id, group] as const));
  const models = new Set<string>();
  for (const binding of resolveBindings(apiKey)) {
    const group = groupMap.get(binding.group_id);
    if (!group) {
      continue;
    }
    for (const model of group.models || []) {
      const publicID = String(model.public_id || "").trim();
      if (publicID) {
        models.add(publicID);
      }
    }
  }
  return models;
}

function resolveBindings(apiKey: ApiKey): ApiKeyGroup[] {
  if (apiKey.api_key_groups?.length) {
    return apiKey.api_key_groups;
  }
  if (!apiKey.group_id) {
    return [];
  }
  return [
    {
      group_id: apiKey.group_id,
      group_name: apiKey.group?.name || `#${apiKey.group_id}`,
      platform: apiKey.group?.platform || "openai",
      priority: apiKey.group?.priority ?? 1,
      quota: 0,
      quota_used: 0,
      model_patterns: [],
    },
  ];
}

function buildOpenAIResponsesRequestBody(
  form: ModelDebugFormState,
  defaults: ModelDebugPromptDefaults,
) {
  const input = buildMessageList(form.systemPrompt, form.userPrompt, defaults);
  return omitEmpty({
    stream: form.stream,
    input,
    temperature: parseOptionalNumber(form.temperature),
    max_output_tokens: parseOptionalInteger(form.maxOutputTokens),
    reasoning: form.reasoningEffort.trim()
      ? { effort: form.reasoningEffort.trim() }
      : undefined,
  });
}

function buildOpenAIChatRequestBody(
  form: ModelDebugFormState,
  defaults: ModelDebugPromptDefaults,
) {
  return omitEmpty({
    stream: form.stream,
    messages: buildMessageList(form.systemPrompt, form.userPrompt, defaults),
    temperature: parseOptionalNumber(form.temperature),
    max_completion_tokens: parseOptionalInteger(form.maxOutputTokens),
  });
}

function buildAnthropicRequestBody(
  form: ModelDebugFormState,
  defaults: ModelDebugPromptDefaults,
) {
  return omitEmpty({
    stream: form.stream,
    system: trimOrUndefined(form.systemPrompt),
    max_tokens: parseOptionalInteger(form.maxOutputTokens) ?? 512,
    temperature: parseOptionalNumber(form.temperature),
    messages: [
      {
        role: "user",
        content: resolveUserPrompt(form.userPrompt, defaults),
      },
    ],
  });
}

function buildGeminiRequestBody(
  form: ModelDebugFormState,
  defaults: ModelDebugPromptDefaults,
) {
  const generationConfig = omitEmpty({
    temperature: parseOptionalNumber(form.temperature),
    maxOutputTokens: parseOptionalInteger(form.maxOutputTokens),
  });
  return omitEmpty({
    contents: [
      {
        role: "user",
        parts: [{ text: resolveUserPrompt(form.userPrompt, defaults) }],
      },
    ],
    system_instruction: trimOrUndefined(form.systemPrompt)
      ? { parts: [{ text: trimOrUndefined(form.systemPrompt) }] }
      : undefined,
    generationConfig: Object.keys(generationConfig).length ? generationConfig : undefined,
  });
}

function buildMessageList(
  systemPrompt: string,
  userPrompt: string,
  defaults: ModelDebugPromptDefaults,
) {
  const messages: Array<{ role: string; content: string }> = [];
  if (trimOrUndefined(systemPrompt)) {
    messages.push({ role: "system", content: trimOrUndefined(systemPrompt) as string });
  }
  messages.push({
    role: "user",
    content: resolveUserPrompt(userPrompt, defaults),
  });
  return messages;
}

function resolveUserPrompt(userPrompt: string, defaults: ModelDebugPromptDefaults) {
  return trimOrUndefined(userPrompt) || trimOrUndefined(defaults.userPrompt) || "Hello from Sub2API";
}

function deepMerge(base: Record<string, any>, override: Record<string, any>): Record<string, any> {
  const merged: Record<string, any> = { ...base };
  for (const [key, value] of Object.entries(override)) {
    if (isPlainObject(merged[key]) && isPlainObject(value)) {
      merged[key] = deepMerge(merged[key], value as Record<string, any>);
      continue;
    }
    merged[key] = value;
  }
  return merged;
}

function omitEmpty(source: Record<string, any>) {
  return Object.fromEntries(
    Object.entries(source).filter(([, value]) => value !== undefined && value !== ""),
  );
}

function isPlainObject(value: unknown): value is Record<string, any> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function trimOrUndefined(value?: string) {
  const trimmed = String(value || "").trim();
  return trimmed || undefined;
}

function parseOptionalNumber(value: string) {
  const trimmed = trimOrUndefined(value);
  if (!trimmed) {
    return undefined;
  }
  const parsed = Number(trimmed);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function parseOptionalInteger(value: string) {
  const parsed = parseOptionalNumber(value);
  if (!Number.isFinite(parsed)) {
    return undefined;
  }
  return Math.max(1, Math.floor(parsed as number));
}
