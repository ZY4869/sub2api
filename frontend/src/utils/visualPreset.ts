import type { VisualPreset, VisualPresetPreference } from "@/types";

export const normalizeVisualPreset = (value?: string | null): VisualPreset => {
  return value === "airy" ? "airy" : "classic";
};

export const normalizeVisualPresetPreference = (
  value?: string | null,
): VisualPresetPreference => {
  if (value === "classic" || value === "airy") {
    return value;
  }
  return "inherit";
};

export const resolveVisualPreset = (
  siteDefault?: string | null,
  userPreference?: string | null,
  accountOverride?: string | null,
): VisualPreset => {
  let resolved = normalizeVisualPreset(siteDefault);
  const normalizedPreference = normalizeVisualPresetPreference(userPreference);
  if (normalizedPreference !== "inherit") {
    resolved = normalizedPreference;
  }
  const normalizedOverride = normalizeVisualPresetPreference(accountOverride);
  if (normalizedOverride !== "inherit") {
    resolved = normalizedOverride;
  }
  return resolved;
};

export const resolveGlobalVisualPreset = (
  siteDefault?: string | null,
  userPreference?: string | null,
): VisualPreset => resolveVisualPreset(siteDefault, userPreference, "inherit");

export const applyRootVisualPreset = (preset?: string | null): VisualPreset => {
  const normalized = normalizeVisualPreset(preset);
  document.documentElement.dataset.visualPreset = normalized;
  return normalized;
};
