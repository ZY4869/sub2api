import type {
  PublicModelCatalogMultiplierSummary,
  PublicModelCatalogPriceEntry,
} from "@/api/meta";

export const PUBLIC_MODEL_PROTOCOL_ORDER = [
  "openai",
  "anthropic",
  "gemini",
  "grok",
  "antigravity",
  "vertex-batch",
] as const;

type Translate = (key: string, params?: Record<string, unknown>) => string;

export function priceEntryLabel(t: Translate, fieldID: string): string {
  switch (fieldID) {
    case "input_price":
      return t("ui.modelCatalog.priceFields.input");
    case "output_price":
      return t("ui.modelCatalog.priceFields.output");
    case "cache_price":
      return t("ui.modelCatalog.priceFields.cache");
    case "input_price_above_threshold":
      return t("ui.modelCatalog.priceFields.inputTier");
    case "output_price_above_threshold":
      return t("ui.modelCatalog.priceFields.outputTier");
    case "batch_input_price":
      return t("ui.modelCatalog.priceFields.batchInput");
    case "batch_output_price":
      return t("ui.modelCatalog.priceFields.batchOutput");
    case "batch_cache_price":
      return t("ui.modelCatalog.priceFields.batchCache");
    case "grounding_search":
      return t("ui.modelCatalog.priceFields.groundingSearch");
    case "grounding_maps":
      return t("ui.modelCatalog.priceFields.groundingMaps");
    case "file_search_embedding":
      return t("ui.modelCatalog.priceFields.embedding");
    case "file_search_retrieval":
      return t("ui.modelCatalog.priceFields.retrieval");
    default:
      return fieldID;
  }
}

export function multiplierSummaryLabel(
  t: Translate,
  summary: PublicModelCatalogMultiplierSummary,
): string {
  if (summary.kind === "disabled") {
    return t("ui.modelCatalog.multiplier.disabled");
  }
  if (summary.kind === "mixed") {
    return t("ui.modelCatalog.multiplier.mixed");
  }
  return `${formatNumber(summary.value || 1)}x`;
}

export function formatCatalogPrice(
  t: Translate,
  entry: PublicModelCatalogPriceEntry,
  currency: string,
  usdToCnyRate: number | null,
): string {
  const nextCurrency = currency === "CNY" ? "CNY" : "USD";
  const symbol = nextCurrency === "CNY" ? "¥" : "$";
  const unit = resolveDisplayUnit(entry.unit);
  const rawValue = convertCurrency(entry.value, nextCurrency, usdToCnyRate);
  const displayValue =
    unit === "per_million_tokens" ? rawValue * 1_000_000 : rawValue;
  const suffix =
    unit === "per_million_tokens"
      ? t("ui.modelCatalog.units.perMillionTokens")
      : unit === "per_image"
        ? t("ui.modelCatalog.units.perImage")
        : t("ui.modelCatalog.units.perRequest");
  return `${symbol}${formatNumber(displayValue)} ${suffix}`;
}

function resolveDisplayUnit(
  unit?: string,
): "per_million_tokens" | "per_request" | "per_image" {
  switch (unit) {
    case "image":
      return "per_image";
    case "video_request":
    case "grounding_search_request":
    case "grounding_maps_request":
      return "per_request";
    default:
      if (String(unit || "").includes("token")) {
        return "per_million_tokens";
      }
      return "per_request";
  }
}

function convertCurrency(
  value: number,
  currency: string,
  usdToCnyRate: number | null,
): number {
  if (currency !== "CNY") {
    return value;
  }
  if (typeof usdToCnyRate === "number" && usdToCnyRate > 0) {
    return value * usdToCnyRate;
  }
  return value;
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: value >= 1 ? 4 : 8,
  }).format(value);
}
