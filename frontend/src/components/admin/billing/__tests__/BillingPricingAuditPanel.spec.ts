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
    global: {
      stubs: {
        ModelIcon: {
          template: '<span data-testid="model-icon-stub" />',
        },
        ModelPlatformIcon: {
          template: '<span data-testid="provider-icon-stub" />',
        },
      },
    },
  })
}

describe('BillingPricingAuditPanel', () => {
  it('renders red and amber issue states with provider rankings', () => {
    const wrapper = mountPanel(createAudit())

    const issueCards = wrapper.findAll('[data-testid="billing-audit-issue-card"]')
    expect(issueCards).toHaveLength(2)
    expect(issueCards[0].classes()).toContain('border-rose-200')
    expect(issueCards[1].classes()).toContain('border-amber-200')

    const statusBadges = wrapper.findAll('[data-testid="billing-audit-issue-status"]')
    expect(statusBadges[0].classes()).toContain('bg-rose-100')
    expect(statusBadges[1].classes()).toContain('bg-amber-100')

    const providerCards = wrapper.findAll('[data-testid="billing-audit-provider-card"]')
    expect(providerCards).toHaveLength(2)
    expect(wrapper.text()).toContain('供应商问题榜')
    expect(wrapper.text()).toContain('OpenAI')
    expect(wrapper.text()).toContain('GPT-5.4 Mini')
    expect(wrapper.findAll('[data-testid="model-icon-stub"]')).toHaveLength(2)
    expect(wrapper.findAll('[data-testid="provider-icon-stub"]')).toHaveLength(2)
  })

  it('renders empty states when no issue examples or provider issues exist', () => {
    const wrapper = mountPanel(createAudit({
      provider_issue_counts: [],
      pricing_issue_examples: [],
    }))

    expect(wrapper.text()).toContain('当前没有需要优先处理的模型问题。')
    expect(wrapper.text()).toContain('当前没有供应商级计费异常。')
  })
})
