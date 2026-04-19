import type {
  ApiKey,
  ApiKeyGroup,
  ApiKeyGroupBindingInput,
  Group,
} from "@/types";

export interface EditableApiKeyGroupBinding {
  group_id: number;
  quota: number | "" | null;
  model_patterns_text: string;
  selected_models: string[];
  model_selection_dirty: boolean;
}

export interface BindableGroup extends Pick<
  Group,
  | "id"
  | "name"
  | "description"
  | "platform"
  | "priority"
  | "rate_multiplier"
  | "subscription_type"
> {}

export const createEmptyEditableBinding = (): EditableApiKeyGroupBinding => ({
  group_id: 0,
  quota: 0,
  model_patterns_text: "",
  selected_models: [],
  model_selection_dirty: false,
});

export const sortApiKeyGroups = (bindings: ApiKeyGroup[]): ApiKeyGroup[] => {
  return [...bindings].sort((a, b) => {
    const priorityDiff = (a.priority ?? 1) - (b.priority ?? 1);
    if (priorityDiff !== 0) return priorityDiff;
    return a.group_id - b.group_id;
  });
};

export const buildFallbackApiKeyGroups = (
  key: ApiKey,
  resolveGroup: (
    groupId: number | null | undefined,
  ) => BindableGroup | undefined,
): ApiKeyGroup[] => {
  if (!key.group_id) return [];
  const group = key.group || resolveGroup(key.group_id);
  return [
    {
      group_id: key.group_id,
      group_name: group?.name || `#${key.group_id}`,
      platform: group?.platform || "anthropic",
      priority: group?.priority ?? 1,
      quota: 0,
      quota_used: 0,
      model_patterns: [],
    },
  ];
};

export const getDisplayApiKeyGroups = (
  key: ApiKey,
  resolveGroup: (
    groupId: number | null | undefined,
  ) => BindableGroup | undefined,
): ApiKeyGroup[] => {
  const bindings = key.api_key_groups?.length
    ? key.api_key_groups
    : buildFallbackApiKeyGroups(key, resolveGroup);
  return sortApiKeyGroups(bindings);
};

export const bindingToEditableDraft = (
  binding: ApiKeyGroup,
): EditableApiKeyGroupBinding => ({
  group_id: binding.group_id,
  quota: binding.quota ?? 0,
  model_patterns_text: (binding.model_patterns || []).join("\n"),
  selected_models: [...(binding.model_patterns || [])],
  model_selection_dirty: false,
});

export const parseModelPatterns = (value: string): string[] => {
  return value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean);
};

export const normalizeQuota = (
  value: number | "" | null | undefined,
): number => {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
};

export const buildApiKeyGroupBindingPayload = (
  bindings: EditableApiKeyGroupBinding[],
  adminMode: boolean,
): ApiKeyGroupBindingInput[] => {
  const seen = new Set<number>();
  const payload: ApiKeyGroupBindingInput[] = [];

  for (const binding of bindings) {
    const groupId = Number(binding.group_id);
    if (!Number.isFinite(groupId) || groupId <= 0 || seen.has(groupId)) {
      continue;
    }
    seen.add(groupId);

    const selectedModels = binding.model_selection_dirty
      ? Array.from(new Set(binding.selected_models.map((item) => item.trim()).filter(Boolean)))
      : parseModelPatterns(binding.model_patterns_text);
    const baseBinding: ApiKeyGroupBindingInput = {
      group_id: groupId,
    };

    if (adminMode) {
      baseBinding.quota = normalizeQuota(binding.quota);
    }
    if (selectedModels.length > 0) {
      baseBinding.model_patterns = selectedModels;
    }
    payload.push(baseBinding);
  }

  return payload;
};
