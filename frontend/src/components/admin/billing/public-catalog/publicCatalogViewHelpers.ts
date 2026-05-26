import type { BillingPublicCatalogAdminEntry, BillingPublicCatalogEntryDraft } from '@/api/admin/billing'
import { normalizeModelID } from './publicCatalogDraft'

export function matchesPublicCatalogFilters(
  item: BillingPublicCatalogAdminEntry,
  keyword: string,
  providerFilter: string,
  accountFilter: string,
): boolean {
  if (providerFilter && (item.provider || item.source_protocol) !== providerFilter) return false
  if (accountFilter && item.source_alias !== accountFilter) return false
  if (!keyword) return true
  return [
    item.display_name,
    item.model,
    item.public_model_id,
    item.base_model,
    item.source_model_id,
    item.source_alias,
    item.source_account_name,
    item.provider,
  ].some((value) => String(value || '').toLowerCase().includes(keyword))
}

export function findDuplicatePublicIDs(entries: BillingPublicCatalogEntryDraft[]): string[] {
  const counts = new Map<string, number>()
  entries.forEach((entry) => {
    const publicID = normalizeModelID(entry.public_model_id)
    if (publicID) counts.set(publicID, (counts.get(publicID) || 0) + 1)
  })
  return Array.from(counts.entries()).filter(([, count]) => count > 1).map(([id]) => id)
}

export function sourceLabelKey(source: string): string {
  switch (source) {
    case 'refreshed_snapshot':
      return 'admin.billing.publicCatalog.source.refreshedSnapshot'
    case 'bootstrap_snapshot':
      return 'admin.billing.publicCatalog.source.bootstrapSnapshot'
    case 'cache_snapshot':
      return 'admin.billing.publicCatalog.source.cacheSnapshot'
    case 'persisted_snapshot':
      return 'admin.billing.publicCatalog.source.persistedSnapshot'
    default:
      return 'admin.billing.publicCatalog.source.fallback'
  }
}

export function downloadPublicCatalogDraft(payload: unknown) {
  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json;charset=utf-8' })
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `public-model-catalog-${new Date().toISOString().slice(0, 10)}.json`
  document.body.appendChild(link)
  link.click()
  link.remove()
  window.URL.revokeObjectURL(url)
}

export function resolveErrorMessage(error: unknown, fallback: string): string {
  if (
    typeof error === 'object'
    && error
    && 'message' in error
    && typeof (error as { message?: unknown }).message === 'string'
  ) {
    return String((error as { message: string }).message)
  }
  return fallback
}
