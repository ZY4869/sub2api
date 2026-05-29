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
