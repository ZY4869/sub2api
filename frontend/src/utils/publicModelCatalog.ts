import type {
  PublicModelCatalogItem,
  PublicModelCatalogMultiplierSummary,
  PublicModelCatalogPriceDisplay,
  PublicModelCatalogPriceEntry,
  PublicModelCatalogStatus,
} from "@/api/meta";
import { formatModelDisplayName } from "@/utils/modelDisplayName";

export const PUBLIC_MODEL_PROTOCOL_ORDER = [
  "openai",
  "anthropic",
  "gemini",
  "grok",
  "deepseek",
  "antigravity",
  "vertex-batch",
] as const;

const PUBLIC_MODEL_CACHE_PRIMARY_IDS = new Set(["cache_price", "batch_cache_price"]);

type Translate = (key: string, params?: Record<string, unknown>) => string;

export interface PublicModelCatalogDisplayItem {
  raw: PublicModelCatalogItem;
  title: string;
  subtitle: string;
  primaryPrices: PublicModelCatalogPriceEntry[];
  secondaryPrices: PublicModelCatalogPriceEntry[];
  status: PublicModelCatalogStatus;
  searchText: string;
}

export function buildPublicModelCatalogDisplayItem(
  item: PublicModelCatalogItem,
): PublicModelCatalogDisplayItem {
  const normalizedTitle = resolvePublicModelTitle(item);
  const normalizedSubtitle = resolvePublicModelSubtitle(item, normalizedTitle);
  const normalizedPrices = normalizePublicModelPriceDisplay(item.price_display);
  return {
    raw: item,
    title: normalizedTitle,
    subtitle: normalizedSubtitle,
    primaryPrices: normalizedPrices.primary,
    secondaryPrices: normalizedPrices.secondary || [],
    status: item.status || "info",
    searchText: [
      normalizedTitle,
      normalizedSubtitle,
      item.display_name,
      item.model,
      ...(item.source_ids || []),
    ]
      .filter(Boolean)
      .join("\n")
      .toLowerCase(),
  };
}

export function resolvePublicModelTitle(item: PublicModelCatalogItem): string {
  const modelTitle = formatModelDisplayName(item.model) || item.model;
  const displayName = String(item.display_name || "").trim();
  if (!displayName) {
    return modelTitle;
  }
  if (sameModelSemantic(displayName, item.model) || sameModelSemantic(displayName, modelTitle)) {
    return modelTitle;
  }
  return displayName;
}

export function resolvePublicModelSubtitle(
  item: PublicModelCatalogItem,
  title = resolvePublicModelTitle(item),
): string {
  const modelID = String(item.model || "").trim();
  if (!modelID || sameModelSemantic(title, modelID)) {
    return "";
  }
  return modelID;
}

export function normalizePublicModelPriceDisplay(
  display: PublicModelCatalogPriceDisplay,
): PublicModelCatalogPriceDisplay {
  const primary = [...(display.primary || [])];
  const secondary = [...(display.secondary || [])];
  const promoted: PublicModelCatalogPriceEntry[] = [];
  const retained: PublicModelCatalogPriceEntry[] = [];
  for (const entry of secondary) {
    if (PUBLIC_MODEL_CACHE_PRIMARY_IDS.has(entry.id)) {
      promoted.push(entry);
      continue;
    }
    retained.push(entry);
  }
  return {
    primary: dedupePriceEntries([...primary, ...promoted]),
    secondary: dedupePriceEntries(retained),
  };
}

export function publicModelStatusLabel(
  t: Translate,
  status?: PublicModelCatalogStatus,
): string {
  switch (status) {
    case "ok":
      return t("ui.modelCatalog.status.ok");
    case "warning":
      return t("ui.modelCatalog.status.warning");
    case "maintenance":
      return t("ui.modelCatalog.status.maintenance");
    case "error":
      return t("ui.modelCatalog.status.error");
    default:
      return t("ui.modelCatalog.status.info");
  }
}

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
  _usdToCnyRate: number | null,
): string {
  const nextCurrency = currency === "CNY" ? "CNY" : "USD";
  const symbol = nextCurrency === "CNY" ? "¥" : "$";
  const unit = resolveDisplayUnit(entry.unit);
  const rawValue = entry.value;
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

function sameModelSemantic(left?: string | null, right?: string | null): boolean {
  return canonicalizeModelLabel(left) === canonicalizeModelLabel(right);
}

function canonicalizeModelLabel(value?: string | null): string {
  return String(value || "")
    .trim()
    .toLowerCase()
    .replace(/(?:^|[-_\s])(?:preview|beta|experimental)(?:$|[-_\s])/g, (token) =>
      token.replace(/[-_\s]/g, ""),
    )
    .replace(/[^a-z0-9]+/g, "");
}

function dedupePriceEntries(entries: PublicModelCatalogPriceEntry[]): PublicModelCatalogPriceEntry[] {
  const seen = new Set<string>();
  const result: PublicModelCatalogPriceEntry[] = [];
  for (const entry of entries) {
    if (!entry?.id || seen.has(entry.id)) {
      continue;
    }
    seen.add(entry.id);
    result.push(entry);
  }
  return result;
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

function formatNumber(value: number): string {
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: value >= 1 ? 4 : 8,
  }).format(value);
}
