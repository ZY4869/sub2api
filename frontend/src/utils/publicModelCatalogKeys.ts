import type { PublicModelCatalogDetailResponse } from "@/api/meta";
import type { ApiKey, ApiKeyGroup, UserGroupModelOptionGroup } from "@/types";

export interface PublicModelSupportedKey {
  id: number;
  key: string;
  name: string;
}

export function findSupportedKeysForModel(
  keys: ApiKey[],
  groups: UserGroupModelOptionGroup[],
  detail: PublicModelCatalogDetailResponse | null,
): PublicModelSupportedKey[] {
  if (!detail) {
    return [];
  }

  const groupMap = new Map(groups.map((group) => [group.group_id, group] as const));
  const candidates = buildModelCandidates(detail);
  const protocol = String(detail.example_protocol || "").trim();

  return keys
    .filter((key) => keySupportsModel(key, groupMap, candidates, protocol))
    .map((key) => ({
      id: key.id,
      key: key.key,
      name: key.name,
    }))
    .sort((left, right) => {
      const nameCompare = left.name.localeCompare(right.name);
      if (nameCompare !== 0) {
        return nameCompare;
      }
      return left.id - right.id;
    });
}

function buildModelCandidates(detail: PublicModelCatalogDetailResponse): string[] {
  const values = new Set<string>();
  appendCandidate(values, detail.item.model);
  for (const item of detail.item.source_ids || []) {
    appendCandidate(values, item);
  }
  return Array.from(values);
}

function appendCandidate(target: Set<string>, value: string | undefined) {
  const trimmed = String(value || "").trim();
  if (!trimmed) {
    return;
  }
  target.add(trimmed);
}

function resolveBindings(key: ApiKey): ApiKeyGroup[] {
  if (key.api_key_groups?.length) {
    return key.api_key_groups;
  }
  if (!key.group_id) {
    return [];
  }
  return [
    {
      group_id: key.group_id,
      group_name: key.group?.name || `#${key.group_id}`,
      model_patterns: [],
      platform: key.group?.platform || "anthropic",
      priority: key.group?.priority ?? 1,
      quota: 0,
      quota_used: 0,
    },
  ];
}

function keySupportsModel(
  key: ApiKey,
  groupMap: Map<number, UserGroupModelOptionGroup>,
  candidates: string[],
  protocol: string,
): boolean {
  return resolveBindings(key).some((binding) => {
    const group = groupMap.get(binding.group_id);
    if (!group) {
      return false;
    }

    return group.models.some((model) => {
      if (!sameCandidate(model.public_id, candidates)) {
        return false;
      }
      if (
        protocol &&
        model.request_protocols?.length &&
        !model.request_protocols.includes(protocol)
      ) {
        return false;
      }
      if (!binding.model_patterns?.length) {
        return true;
      }

      const modelVariants = [model.public_id, ...(model.source_ids || [])];
      return binding.model_patterns.some((pattern) =>
        modelVariants.some((variant) => matchModelPattern(pattern, variant)),
      );
    });
  });
}

function sameCandidate(publicID: string, candidates: string[]): boolean {
  const trimmed = String(publicID || "").trim();
  if (!trimmed) {
    return false;
  }
  return candidates.includes(trimmed);
}

function matchModelPattern(pattern: string, modelID: string): boolean {
  const normalizedPattern = String(pattern || "").trim();
  const normalizedModel = String(modelID || "").trim();
  if (!normalizedPattern || !normalizedModel) {
    return false;
  }
  if (normalizedPattern.endsWith("*")) {
    return normalizedModel.startsWith(normalizedPattern.slice(0, -1));
  }
  return normalizedPattern === normalizedModel;
}
