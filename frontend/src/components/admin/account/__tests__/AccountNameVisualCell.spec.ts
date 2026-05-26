import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import AccountNameVisualCell from '../AccountNameVisualCell.vue'
import type { Account } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string) => key,
    }),
  }
})

const makeAccount = (platform: Account['platform']): Account => ({
  id: 1,
  name: 'Primary',
  platform,
  type: 'apikey',
  proxy_id: null,
  concurrency: 1,
  current_concurrency: 0,
  priority: 0,
  status: 'active',
  error_message: null,
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: false,
  created_at: '2026-05-22T00:00:00Z',
  updated_at: '2026-05-22T00:00:00Z',
  schedulable: true,
  rate_limited_at: null,
  rate_limit_reset_at: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  lifecycle_state: 'normal',
  extra: {},
} as Account)

describe('AccountNameVisualCell', () => {
  it('keeps airy platform names in brand casing', () => {
    const openai = mount(AccountNameVisualCell, {
      props: { account: makeAccount('openai') },
      global: {
        stubs: {
          PlatformIcon: true,
        },
      },
    })
    const deepseek = mount(AccountNameVisualCell, {
      props: { account: makeAccount('deepseek') },
      global: {
        stubs: {
          PlatformIcon: true,
        },
      },
    })

    expect(openai.text()).toContain('OpenAI')
    expect(openai.text()).not.toContain('OPENAI')
    expect(deepseek.text()).toContain('DeepSeek')
    expect(deepseek.text()).not.toContain('DEEPSEEK')
  })
})
