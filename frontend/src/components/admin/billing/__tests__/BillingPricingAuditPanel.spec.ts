import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { BillingPricingAudit } from '@/api/admin/billing'
import BillingPricingAuditPanel from '../BillingPricingAuditPanel.vue'

function createAudit(overrides: Partial<BillingPricingAudit> = {}): BillingPricingAudit {
  return {
    total_models: 12,
    pricing_status_counts: {
      ok: 8,
      fallback: 1,
      conflict: 1,
      missing: 2,
    },
    duplicate_model_ids: [],
    aux_identifier_collisions: [],
    collision_counts_by_source: {
      aliases: 1,
      protocol_ids: 0,
      pricing_lookup_ids: 2,
    },
    provider_issue_counts: [
      { provider: 'openai', total: 2, fallback: 1, conflict: 1, missing: 0 },
      { provider: 'gemini', total: 1, fallback: 0, conflict: 0, missing: 1 },
    ],
    pricing_issue_examples: [
      {
        model: 'gpt-5.4-mini',
        display_name: 'GPT-5.4 Mini',
        provider: 'openai',
        pricing_status: 'conflict',
        first_warning: 'aliases identifier "gpt-5" collides with 2 models',
      },
      {
        model: 'gpt-5.4',
        display_name: 'GPT-5.4',
        provider: 'openai',
        pricing_status: 'fallback',
        first_warning: 'Using billing fallback pricing source.',
      },
    ],
    missing_in_snapshot_count: 1,
    missing_in_snapshot_models: ['gpt-5.4'],
    snapshot_only_count: 0,
    snapshot_only_models: [],
    refresh_required: true,
    snapshot_updated_at: '2026-04-16T00:00:00Z',
    ...overrides,
  }
}

function mountPanel(audit: BillingPricingAudit | null) {
  return mount(BillingPricingAuditPanel, {
    props: {
      audit,
      loading: false,
      snapshotUpdatedAtLabel: '2026-04-16 08:00',
    },
  })
}

describe('BillingPricingAuditPanel', () => {
  it('renders overview cards without the issue ranking sections', () => {
    const wrapper = mountPanel(createAudit())

    expect(wrapper.text()).toContain('计费审计')
    expect(wrapper.text()).toContain('状态分布')
    expect(wrapper.text()).toContain('冲突来源')
    expect(wrapper.text()).toContain('快照健康度')
    expect(wrapper.text()).not.toContain('供应商问题榜')
    expect(wrapper.text()).not.toContain('重点问题模型')
  })

  it('renders snapshot and collision counts', () => {
    const wrapper = mountPanel(createAudit({
      duplicate_model_ids: ['gpt-5.4'],
      snapshot_only_count: 3,
    }))

    expect(wrapper.text()).toContain('主 ID 重复')
    expect(wrapper.text()).toContain('快照缺口')
    expect(wrapper.text()).toContain('仅快照模型')
    expect(wrapper.text()).toContain('3')
  })
})
