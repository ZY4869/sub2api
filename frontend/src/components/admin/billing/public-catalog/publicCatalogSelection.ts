import type { BillingPublicCatalogAdminEntry, BillingPublicCatalogEntryDraft } from '@/api/admin/billing'
import {
  createDraftEntry,
  entryKey,
  normalizeDraftEntryForPayload,
  normalizePageSize,
} from './publicCatalogDraft'
import { scalePriceDisplay } from './publicCatalogPricing'

export function addCatalogEntry(
  selected: BillingPublicCatalogEntryDraft[],
  item: BillingPublicCatalogAdminEntry,
): BillingPublicCatalogEntryDraft[] {
  if (!isPublicCatalogEntryPublishable(item)) return selected
  const key = entryKey(item)
  return selected.some((entry) => entry.entry_id === key)
    ? selected
    : [...selected, createDraftEntry(item)]
}

export function addFilteredCatalogEntries(
  selected: BillingPublicCatalogEntryDraft[],
  filtered: BillingPublicCatalogAdminEntry[],
): BillingPublicCatalogEntryDraft[] {
  const existing = new Set(selected.map((entry) => entry.entry_id))
  return [
    ...selected,
    ...filtered.filter((item) => isPublicCatalogEntryPublishable(item) && !existing.has(entryKey(item))).map(createDraftEntry),
  ]
}

export function isPublicCatalogEntryPublishable(item: BillingPublicCatalogAdminEntry): boolean {
  if (item.unavailable_reason) return false
  if (item.route_confirmed === false) return false
  if (item.availability_state && item.availability_state !== 'verified') return false
  if (item.stale_state && item.stale_state !== 'fresh') return false
  return true
}

export function moveCatalogEntry(
  selected: BillingPublicCatalogEntryDraft[],
  index: number,
  delta: number,
): BillingPublicCatalogEntryDraft[] {
  const nextIndex = index + delta
  if (nextIndex < 0 || nextIndex >= selected.length) return selected
  const next = [...selected]
  const [target] = next.splice(index, 1)
  next.splice(nextIndex, 0, target)
  return next
}

export function reorderCatalogEntries(
  selected: BillingPublicCatalogEntryDraft[],
  orderedEntryIDs: string[],
): BillingPublicCatalogEntryDraft[] {
  const byID = new Map(selected.map((entry) => [entry.entry_id, entry] as const))
  const seen = new Set<string>()
  const reordered = orderedEntryIDs
    .map((entryID) => {
      seen.add(entryID)
      return byID.get(entryID)
    })
    .filter((entry): entry is BillingPublicCatalogEntryDraft => Boolean(entry))
  return [
    ...reordered,
    ...selected.filter((entry) => !seen.has(entry.entry_id)),
  ]
}

export function patchCatalogEntry(
  selected: BillingPublicCatalogEntryDraft[],
  entryID: string,
  patch: Partial<BillingPublicCatalogEntryDraft>,
): BillingPublicCatalogEntryDraft[] {
  return selected.map((entry) => (entry.entry_id === entryID ? { ...entry, ...patch } : entry))
}

export function applyCatalogBatchRatio(
  selected: BillingPublicCatalogEntryDraft[],
  targetEntryIDs: Set<string>,
  itemMap: Map<string, BillingPublicCatalogAdminEntry>,
  ratio: number,
): BillingPublicCatalogEntryDraft[] {
  return selected.map((entry) => {
    if (!targetEntryIDs.has(entry.entry_id)) return entry
    const item = itemMap.get(entry.entry_id)
    const official = item?.official_price_display || item?.price_display
    return official ? { ...entry, sale_price_display: scalePriceDisplay(official, ratio) } : entry
  })
}

export function buildPublicCatalogDraftPayload(
  selected: BillingPublicCatalogEntryDraft[],
  pageSize: number,
  updatedAt: string,
  itemMap: Map<string, BillingPublicCatalogAdminEntry>,
) {
  const entries = selected.map((entry) => normalizeDraftEntryForPayload(entry, itemMap.get(entry.entry_id)))
  return {
    selected_entries: entries,
    selected_models: entries.map((entry) => entry.public_model_id),
    page_size: normalizePageSize(pageSize),
    updated_at: updatedAt,
  }
}
