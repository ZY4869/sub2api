import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ApiKeyGroupSummary from '../ApiKeyGroupSummary.vue'
import ApiKeyUsageSummary from '../ApiKeyUsageSummary.vue'
import type { ApiKey, ApiKeyGroup, Group } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const baseApiKey = (overrides: Partial<ApiKey> = {}): ApiKey => ({
  id: 7,
  user_id: 1,
  key: 'sk-test',
  name: 'image-key',
  group_id: null,
  status: 'active',
  ip_whitelist: [],
  ip_blacklist: [],
  last_used_at: null,
  quota: 10,
  quota_used: 2,
  image_only_enabled: true,
  image_count_billing_enabled: true,
  image_max_count: 100,
  image_count_used: 25,
  image_count_weights: { '1K': 1, '2K': 1, '4K': 2 },
  expires_at: null,
  created_at: '2026-06-01T00:00:00Z',
  updated_at: '2026-06-01T00:00:00Z',
  rate_limit_5h: 0,
  rate_limit_1d: 0,
  rate_limit_7d: 0,
  usage_5h: 0,
  usage_1d: 0,
  usage_7d: 0,
  window_5h_start: null,
  window_1d_start: null,
  window_7d_start: null,
  reset_5h_at: null,
  reset_1d_at: null,
  reset_7d_at: null,
  ...overrides
})

const group = (id: number, name: string, priority = id): Group => ({
  id,
  name,
  description: null,
  platform: 'openai',
  priority,
  rate_multiplier: 1.25,
  is_exclusive: false,
  status: 'active',
  subscription_type: 'standard',
  daily_limit_usd: null,
  weekly_limit_usd: null,
  monthly_limit_usd: null,
  image_price_1k: null,
  image_price_2k: null,
  image_price_4k: null,
  claude_code_only: false,
  image_protocol_mode: 'native',
  fallback_group_id: null,
  fallback_group_id_on_invalid_request: null,
  created_at: '2026-06-01T00:00:00Z',
  updated_at: '2026-06-01T00:00:00Z'
})

describe('ApiKey summaries', () => {
  it('hides image quota for normal users', () => {
    const wrapper = mount(ApiKeyUsageSummary, {
      props: {
        apiKey: baseApiKey(),
        stats: {
          api_key_id: 7,
          today_actual_cost: 0.125,
          total_actual_cost: 3.5
        }
      }
    })

    expect(wrapper.text()).not.toContain('keys.imageCountUsage:')
    expect(wrapper.text()).not.toContain('25 / 100')
  })

  it('renders image quota inside the admin usage summary', () => {
    const wrapper = mount(ApiKeyUsageSummary, {
      props: {
        apiKey: baseApiKey(),
        isAdminMode: true,
        stats: {
          api_key_id: 7,
          today_actual_cost: 0.125,
          total_actual_cost: 3.5
        }
      }
    })

    expect(wrapper.text()).toContain('keys.imageCountUsage:')
    expect(wrapper.text()).toContain('25 / 100')
    expect(wrapper.text()).toContain('keys.imageCountRemaining: 75')
    expect(wrapper.html()).toContain('width: 25%')
  })

  it('marks exhausted image quota clearly', () => {
    const wrapper = mount(ApiKeyUsageSummary, {
      props: {
        apiKey: baseApiKey({
          image_count_used: 100,
          image_max_count: 100
        }),
        isAdminMode: true
      }
    })

    expect(wrapper.find('.text-red-500').exists()).toBe(true)
    expect(wrapper.html()).toContain('width: 100%')
  })

  it('summarizes many groups without losing accessible detail', () => {
    const bindings: ApiKeyGroup[] = [
      { group_id: 1, group_name: 'OpenAI Primary', platform: 'openai', priority: 1, quota: 0, quota_used: 0, model_patterns: [] },
      { group_id: 2, group_name: 'Gemini Backup', platform: 'gemini', priority: 2, quota: 0, quota_used: 0, model_patterns: [] },
      { group_id: 3, group_name: 'DeepSeek Overflow', platform: 'deepseek', priority: 3, quota: 0, quota_used: 0, model_patterns: [] }
    ]
    const groups = new Map<number, Group>([
      [1, group(1, 'OpenAI Primary', 1)],
      [2, group(2, 'Gemini Backup', 2)],
      [3, group(3, 'DeepSeek Overflow', 3)]
    ])

    const wrapper = mount(ApiKeyGroupSummary, {
      props: {
        apiKey: baseApiKey(),
        bindings,
        userGroupRates: { 1: 0.8 },
        isAdminMode: false,
        resolveGroup: (id?: number | null) => (id == null ? undefined : groups.get(id))
      },
      global: {
        stubs: {
          ApiKeyGroupPill: {
            props: ['name'],
            template: '<span class="group-pill">{{ name }}</span>'
          }
        }
      }
    })

    expect(wrapper.findAll('.group-pill')).toHaveLength(2)
    expect(wrapper.text()).toContain('+1')
    expect(wrapper.text()).not.toContain('P1')
    expect(wrapper.text()).not.toContain('P2')
    expect(wrapper.attributes('aria-label')).toBeUndefined()
    const focusable = wrapper.find('[tabindex="0"]')
    expect(focusable.attributes('aria-label')).toContain('OpenAI Primary')
    expect(focusable.attributes('aria-label')).toContain('DeepSeek Overflow')
    expect(focusable.attributes('aria-label')).toContain('keys.groupRate: 0.8x')
  })
})
