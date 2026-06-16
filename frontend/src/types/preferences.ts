export type UsageModelDisplayMode =
  | "model_only"
  | "display_only"
  | "display_and_model";

export type VisualPreset = "classic" | "airy";
export type VisualPresetPreference = "inherit" | VisualPreset;
export type AccountVisualStyle = VisualPreset;

export type UsageContextBadgeDisplayMode =
  | "request_only"
  | "native_only"
  | "both";

export type AccountTodayStatsWindow = "today" | "weekly" | "monthly" | "total";
export type AccountTodayStatsCycleMode = "calendar" | "fixed";
export type AccountGroupDisplayMode = "full" | "icon";
export type AccountStatusDisplayMode = "simple" | "detailed";
export type ExternalModelCatalogViewMode =
  | "follow_key_binding"
  | "group_first"
  | "model_only";
export type EffectiveExternalModelCatalogViewMode =
  | "group_first"
  | "model_only";
