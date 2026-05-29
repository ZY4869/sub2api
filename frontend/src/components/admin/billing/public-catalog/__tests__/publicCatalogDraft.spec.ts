import { describe, expect, it } from 'vitest'
import type { BillingPublicCatalogAdminEntry } from '@/api/admin/billing'
import {
  normalizeDraftEntries,
  resolveAvailableEntry,
} from '../publicCatalogDraft'

function catalogEntry(overrides: Partial<BillingPublicCatalogAdminEntry>): BillingPublicCatalogAdminEntry {
  return {
    entry_id: 'entry-current',
    public_model_id: 'gpt-5.4@team-a',
    model: 'gpt-5.4@team-a',
    base_model: 'gpt-5.4',
    source_model_id: 'gpt-5.4',
    source_protocol: 'openai',
    source_alias: 'Team A',
    source_account_id: 42,
    display_name: 'GPT-5.4',
    provider: 'openai',
    currency: 'USD',
    price_display: { primary: [{ id: 'output_price', unit: 'output_token', value: 2e-6 }] },
    sale_price_display: { primary: [{ id: 'output_price', unit: 'output_token', value: 2.4e-6 }] },
    official_price_display: { primary: [{ id: 'output_price', unit: 'output_token', value: 2e-6 }] },
    multiplier_summary: { enabled: false, kind: 'disabled' },
    ...overrides,
  }
}

describe('publicCatalogDraft', () => {
  it('remaps stale entry ids to the current available source entry', () => {
    const current = catalogEntry({})
    const draftEntries = normalizeDraftEntries({
      selected_entries: [{
        entry_id: 'entry-stale',
        public_model_id: 'gpt-5.4-public',
        source_account_id: 42,
        source_model_id: 'gpt-5.4',
        base_model: 'gpt-5.4',
        source_protocol: 'openai',
        source_alias: 'Premium',
        sale_price_display: { primary: [{ id: 'output_price', unit: 'output_token', value: 3e-6 }] },
      }],
      page_size: 10,
    }, [current])

    expect(draftEntries).toEqual([
      expect.objectContaining({
        entry_id: 'entry-current',
        public_model_id: 'gpt-5.4-public',
        source_account_id: 42,
        source_model_id: 'gpt-5.4',
        source_alias: 'Premium',
        sale_price_display: expect.objectContaining({
          primary: [expect.objectContaining({ value: 3e-6 })],
        }),
      }),
    ])
  })

  it('does not source-match ambiguous accountless entries', () => {
    const selected = {
      entry_id: 'entry-stale',
      public_model_id: 'gpt-5.4-public',
      source_model_id: 'gpt-5.4',
      base_model: 'gpt-5.4',
      source_protocol: 'openai',
    }

    expect(resolveAvailableEntry(selected, [
      catalogEntry({ entry_id: 'entry-a', source_account_id: 42 }),
      catalogEntry({ entry_id: 'entry-b', source_account_id: 43 }),
    ])).toBeUndefined()
  })
})
