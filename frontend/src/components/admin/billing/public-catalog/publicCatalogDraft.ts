import type {
  BillingPublicCatalogAdminEntry,
  BillingPublicCatalogDraft,
  BillingPublicCatalogEntryDraft,
} from '@/api/admin/billing'
import { clonePriceDisplay } from './publicCatalogPricing'

export type SelectedCatalogItem = BillingPublicCatalogAdminEntry & { missing?: boolean }

export function entryKey(
  item: Pick<BillingPublicCatalogAdminEntry, 'entry_id' | 'model' | 'source_model_id' | 'source_protocol' | 'source_account_id'>,
): string {
  return item.entry_id || [
    item.source_account_id || 0,
    item.source_protocol || '',
    item.source_model_id || item.model,
  ].join(':')
}

export function normalizeModelID(value?: string): string {
  return String(value || '').trim()
}

export function normalizePageSize(value: number): number {
  if (!Number.isFinite(value) || value <= 0) {
    return 10
  }
  return Math.min(100, Math.max(1, Math.round(value)))
}

export function normalizeAvailableEntries(items: BillingPublicCatalogAdminEntry[]): BillingPublicCatalogAdminEntry[] {
  return (items || []).map((item) => ({
    ...item,
    entry_id: entryKey(item),
    public_model_id: item.public_model_id || item.model,
    sale_price_display: clonePriceDisplay(item.sale_price_display || item.price_display),
    official_price_display: clonePriceDisplay(item.official_price_display || item.price_display),
  }))
}

export function normalizeDraftEntries(
  draft: BillingPublicCatalogDraft,
  items: BillingPublicCatalogAdminEntry[],
): BillingPublicCatalogEntryDraft[] {
  const itemMap = new Map(items.map((item) => [entryKey(item), item] as const))
  if (draft.selected_entries?.length) {
    return draft.selected_entries.map((entry) => {
      const source = itemMap.get(entry.entry_id)
      return createDraftEntry(source ? mergeDraftEntryWithItem(entry, source) : draftEntryToMissingItem(entry))
    })
  }
  const byModel = new Map(items.map((item) => [normalizeModelID(item.model), item] as const))
  return (draft.selected_models || [])
    .map((model) => byModel.get(normalizeModelID(model)))
    .filter((item): item is BillingPublicCatalogAdminEntry => Boolean(item))
    .map(createDraftEntry)
}

export function createDraftEntry(item: BillingPublicCatalogAdminEntry): BillingPublicCatalogEntryDraft {
  return {
    entry_id: entryKey(item),
    public_model_id: normalizeModelID(item.public_model_id || item.model),
    source_account_id: item.source_account_id,
    source_alias: item.source_alias || '',
    source_model_id: normalizeModelID(item.source_model_id || item.base_model || item.model),
    base_model: normalizeModelID(item.base_model || item.source_model_id || item.model),
    source_protocol: item.source_protocol || item.request_protocols?.[0] || '',
    sale_price_display: clonePriceDisplay(item.sale_price_display || item.price_display),
  }
}

export function mergeDraftEntryWithItem(
  entry: BillingPublicCatalogEntryDraft,
  item: BillingPublicCatalogAdminEntry,
): BillingPublicCatalogAdminEntry {
  return {
    ...item,
    public_model_id: entry.public_model_id || item.public_model_id || item.model,
    model: entry.public_model_id || item.public_model_id || item.model,
    source_alias: entry.source_alias || item.source_alias,
    source_model_id: entry.source_model_id || item.source_model_id,
    base_model: entry.base_model || item.base_model,
    source_protocol: entry.source_protocol || item.source_protocol,
    sale_price_display: clonePriceDisplay(entry.sale_price_display || item.sale_price_display || item.price_display),
  }
}

export function draftEntryToMissingItem(entry: BillingPublicCatalogEntryDraft): SelectedCatalogItem {
  const model = entry.public_model_id || entry.source_model_id || entry.base_model || entry.entry_id
  return {
    entry_id: entry.entry_id,
    public_model_id: entry.public_model_id || model,
    model,
    base_model: entry.base_model || entry.source_model_id || model,
    source_model_id: entry.source_model_id || entry.base_model || model,
    source_protocol: entry.source_protocol,
    source_alias: entry.source_alias,
    display_name: model,
    provider: entry.source_protocol,
    currency: 'USD',
    price_display: clonePriceDisplay(entry.sale_price_display),
    sale_price_display: clonePriceDisplay(entry.sale_price_display),
    official_price_display: clonePriceDisplay(entry.sale_price_display),
    multiplier_summary: { enabled: false, kind: 'disabled' },
    missing: true,
  }
}

export function normalizeDraftEntryForPayload(
  entry: BillingPublicCatalogEntryDraft,
  source?: BillingPublicCatalogAdminEntry,
): BillingPublicCatalogEntryDraft {
  return {
    entry_id: entry.entry_id,
    public_model_id: normalizeModelID(entry.public_model_id || source?.public_model_id || source?.model),
    source_account_id: entry.source_account_id || source?.source_account_id,
    source_alias: String(entry.source_alias || source?.source_alias || '').trim(),
    source_model_id: normalizeModelID(entry.source_model_id || source?.source_model_id || source?.base_model),
    base_model: normalizeModelID(entry.base_model || source?.base_model || source?.source_model_id),
    source_protocol: String(entry.source_protocol || source?.source_protocol || '').trim(),
    sale_price_display: clonePriceDisplay(entry.sale_price_display || source?.sale_price_display || source?.price_display),
  }
}

export function uniqueSorted(values: string[]): string[] {
  return Array.from(new Set(values.map((value) => value.trim()).filter(Boolean)))
    .sort((left, right) => left.localeCompare(right))
}
