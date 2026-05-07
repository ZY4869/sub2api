import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BillingPricingIssuesView from '../BillingPricingIssuesView.vue'

const apiMocks = vi.hoisted(() => ({
  getBillingPricingAudit: vi.fn(),
}))

const storeMocks = vi.hoisted(() => ({
  showError: vi.fn(),
}))

vi.mock('@/api/admin/billing', () => ({
  getBillingPricingAudit: apiMocks.getBillingPricingAudit,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: storeMocks.showError,
  }),
}))

describe('BillingPricingIssuesView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMocks.getBillingPricingAudit.mockResolvedValue({
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
      ],
      pricing_issue_examples: [
        {
          model: 'gpt-5.4-mini',
          display_name: 'GPT-5.4 Mini',
          provider: 'openai',
          pricing_status: 'conflict',
          first_warning: 'collision',
        },
      ],
      missing_in_snapshot_count: 1,
      missing_in_snapshot_models: ['gpt-5.4'],
      snapshot_only_count: 0,
      snapshot_only_models: [],
      refresh_required: true,
      snapshot_updated_at: '2026-04-16T00:00:00Z',
    })
  })

  it('renders the standalone billing issues page and issue ranking cards', async () => {
    const wrapper = mount(BillingPricingIssuesView, {
      global: {
        stubs: {
          ModelIcon: true,
          ModelPlatformIcon: true,
        },
      },
    })

    await flushPromises()

    expect(apiMocks.getBillingPricingAudit).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('计费问题榜')
    expect(wrapper.text()).toContain('问题榜快照')
    expect(wrapper.text()).toContain('重点问题模型')
    expect(wrapper.text()).toContain('供应商问题榜')
    expect(wrapper.findAll('[data-testid="billing-audit-issue-card"]')).toHaveLength(1)
  })
})
